// Copyright 2014 The go-ethereum Authors
// This file is part of the go-gdaereum library.
//
// The go-gdaereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-gdaereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-gdaereum library. If not, see <http://www.gnu.org/licenses/>.

// Package gda implements the gdachain protocol.
package gda

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gdachain/go-gdachain/accounts"
	"github.com/gdachain/go-gdachain/common"
	"github.com/gdachain/go-gdachain/common/hexutil"
	"github.com/gdachain/go-gdachain/consensus"
	"github.com/gdachain/go-gdachain/consensus/clique"
	"github.com/gdachain/go-gdachain/consensus/ethash"
	"github.com/gdachain/go-gdachain/core"
	"github.com/gdachain/go-gdachain/core/bloombits"
	"github.com/gdachain/go-gdachain/core/types"
	"github.com/gdachain/go-gdachain/core/vm"
	"github.com/gdachain/go-gdachain/gda/downloader"
	"github.com/gdachain/go-gdachain/gda/filters"
	"github.com/gdachain/go-gdachain/gda/gasprice"
	"github.com/gdachain/go-gdachain/gdadb"
	"github.com/gdachain/go-gdachain/event"
	"github.com/gdachain/go-gdachain/internal/ethapi"
	"github.com/gdachain/go-gdachain/log"
	"github.com/gdachain/go-gdachain/miner"
	"github.com/gdachain/go-gdachain/node"
	"github.com/gdachain/go-gdachain/p2p"
	"github.com/gdachain/go-gdachain/params"
	"github.com/gdachain/go-gdachain/rlp"
	"github.com/gdachain/go-gdachain/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// gdachain implements the gdachain full node service.
type gdachain struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the gdaereum
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb gdadb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *gdaApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	gdaerbase common.Address

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and gdaerbase)
}

func (s *gdachain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new gdachain object (including the
// initialisation of the common gdachain object)
func New(ctx *node.ServiceContext, config *Config) (*gdachain, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run gda.gdachain in light sync mode, use les.Lightgdachain")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	gda := &gdachain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.gdaash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		gdaerbase:      config.gdaerbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising gdachain protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run ggda upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	gda.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, gda.chainConfig, gda.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		gda.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	gda.bloomIndexer.Start(gda.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	gda.txPool = core.NewTxPool(config.TxPool, gda.chainConfig, gda.blockchain)

	if gda.protocolManager, err = NewProtocolManager(gda.chainConfig, config.SyncMode, config.NetworkId, gda.eventMux, gda.txPool, gda.engine, gda.blockchain, chainDb); err != nil {
		return nil, err
	}
	gda.miner = miner.New(gda, gda.chainConfig, gda.EventMux(), gda.engine)
	gda.miner.SetExtra(makeExtraData(config.ExtraData))

	gda.ApiBackend = &gdaApiBackend{gda, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	gda.ApiBackend.gpo = gasprice.NewOracle(gda.ApiBackend, gpoParams)

	return gda, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"ggda",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (gdadb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*gdadb.LDBDatabase); ok {
		db.Meter("gda/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an gdachain service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ethash.Config, chainConfig *params.ChainConfig, db gdadb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowMode == ethash.ModeFake:
		log.Warn("gdaash used in fake mode")
		return ethash.NewFaker()
	case config.PowMode == ethash.ModeTest:
		log.Warn("gdaash used in test mode")
		return ethash.NewTester()
	case config.PowMode == ethash.ModeShared:
		log.Warn("gdaash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ethash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the gdaereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *gdachain) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "gda",
			Version:   "1.0",
			Service:   NewPublicgdachainAPI(s),
			Public:    true,
		}, {
			Namespace: "gda",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "gda",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "gda",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *gdachain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *gdachain) gdaerbase() (eb common.Address, err error) {
	s.lock.RLock()
	gdaerbase := s.gdaerbase
	s.lock.RUnlock()

	if gdaerbase != (common.Address{}) {
		return gdaerbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			gdaerbase := accounts[0].Address

			s.lock.Lock()
			s.gdaerbase = gdaerbase
			s.lock.Unlock()

			log.Info("gdaerbase automatically configured", "address", gdaerbase)
			return gdaerbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("gdaerbase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *gdachain) Setgdaerbase(gdaerbase common.Address) {
	self.lock.Lock()
	self.gdaerbase = gdaerbase
	self.lock.Unlock()

	self.miner.Setgdaerbase(gdaerbase)
}

func (s *gdachain) StartMining(local bool) error {
	eb, err := s.gdaerbase()
	if err != nil {
		log.Error("Cannot start mining without gdaerbase", "err", err)
		return fmt.Errorf("gdaerbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("gdaerbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *gdachain) StopMining()         { s.miner.Stop() }
func (s *gdachain) IsMining() bool      { return s.miner.Mining() }
func (s *gdachain) Miner() *miner.Miner { return s.miner }

func (s *gdachain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *gdachain) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *gdachain) TxPool() *core.TxPool               { return s.txPool }
func (s *gdachain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *gdachain) Engine() consensus.Engine           { return s.engine }
func (s *gdachain) ChainDb() gdadb.Database            { return s.chainDb }
func (s *gdachain) IsListening() bool                  { return true } // Always listening
func (s *gdachain) gdaVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *gdachain) NetVersion() uint64                 { return s.networkId }
func (s *gdachain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *gdachain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// gdachain protocol implementation.
func (s *gdachain) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// gdachain protocol.
func (s *gdachain) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
