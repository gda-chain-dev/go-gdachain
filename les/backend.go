// Copyright 2016 The go-ethereum Authors
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

// Package les implements the Light gdachain Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/gdachain/go-gdachain/accounts"
	"github.com/gdachain/go-gdachain/common"
	"github.com/gdachain/go-gdachain/common/hexutil"
	"github.com/gdachain/go-gdachain/consensus"
	"github.com/gdachain/go-gdachain/core"
	"github.com/gdachain/go-gdachain/core/bloombits"
	"github.com/gdachain/go-gdachain/core/types"
	"github.com/gdachain/go-gdachain/gda"
	"github.com/gdachain/go-gdachain/gda/downloader"
	"github.com/gdachain/go-gdachain/gda/filters"
	"github.com/gdachain/go-gdachain/gda/gasprice"
	"github.com/gdachain/go-gdachain/gdadb"
	"github.com/gdachain/go-gdachain/event"
	"github.com/gdachain/go-gdachain/internal/ethapi"
	"github.com/gdachain/go-gdachain/light"
	"github.com/gdachain/go-gdachain/log"
	"github.com/gdachain/go-gdachain/node"
	"github.com/gdachain/go-gdachain/p2p"
	"github.com/gdachain/go-gdachain/p2p/discv5"
	"github.com/gdachain/go-gdachain/params"
	rpc "github.com/gdachain/go-gdachain/rpc"
)

type Lightgdachain struct {
	config *gda.Config

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb gdadb.Database // Block chain database

	bloomRequests                              chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *gda.Config) (*Lightgdachain, error) {
	chainDb, err := gda.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lgda := &Lightgdachain{
		config:           config,
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           gda.CreateConsensusEngine(ctx, &config.gdaash, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     gda.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	lgda.relay = NewLesTxRelay(peers, lgda.reqDist)
	lgda.serverPool = newServerPool(chainDb, quitSync, &lgda.wg)
	lgda.retriever = newRetrieveManager(peers, lgda.reqDist, lgda.serverPool)
	lgda.odr = NewLesOdr(chainDb, lgda.chtIndexer, lgda.bloomTrieIndexer, lgda.bloomIndexer, lgda.retriever)
	if lgda.blockchain, err = light.NewLightChain(lgda.odr, lgda.chainConfig, lgda.engine); err != nil {
		return nil, err
	}
	lgda.bloomIndexer.Start(lgda.blockchain)
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lgda.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lgda.txPool = light.NewTxPool(lgda.chainConfig, lgda.blockchain, lgda.relay)
	if lgda.protocolManager, err = NewProtocolManager(lgda.chainConfig, true, ClientProtocolVersions, config.NetworkId, lgda.eventMux, lgda.engine, lgda.peers, lgda.blockchain, nil, chainDb, lgda.odr, lgda.relay, quitSync, &lgda.wg); err != nil {
		return nil, err
	}
	lgda.ApiBackend = &LesApiBackend{lgda, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	lgda.ApiBackend.gpo = gasprice.NewOracle(lgda.ApiBackend, gpoParams)
	return lgda, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// gdaerbase is the address that mining rewards will be send to
func (s *LightDummyAPI) gdaerbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for gdaerbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the gdaereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Lightgdachain) APIs() []rpc.API {
	return append(ethapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "gda",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "gda",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "gda",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Lightgdachain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Lightgdachain) BlockChain() *light.LightChain      { return s.blockchain }
func (s *Lightgdachain) TxPool() *light.TxPool              { return s.txPool }
func (s *Lightgdachain) Engine() consensus.Engine           { return s.engine }
func (s *Lightgdachain) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Lightgdachain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *Lightgdachain) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Lightgdachain) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// gdachain protocol implementation.
func (s *Lightgdachain) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// gdachain protocol.
func (s *Lightgdachain) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
