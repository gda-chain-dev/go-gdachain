// Copyright 2015 The go-ethereum Authors
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

package gda

import (
	"context"
	"math/big"

	"github.com/gdachain/go-gdachain/accounts"
	"github.com/gdachain/go-gdachain/common"
	"github.com/gdachain/go-gdachain/common/math"
	"github.com/gdachain/go-gdachain/core"
	"github.com/gdachain/go-gdachain/core/bloombits"
	"github.com/gdachain/go-gdachain/core/state"
	"github.com/gdachain/go-gdachain/core/types"
	"github.com/gdachain/go-gdachain/core/vm"
	"github.com/gdachain/go-gdachain/gda/downloader"
	"github.com/gdachain/go-gdachain/gda/gasprice"
	"github.com/gdachain/go-gdachain/gdadb"
	"github.com/gdachain/go-gdachain/event"
	"github.com/gdachain/go-gdachain/params"
	"github.com/gdachain/go-gdachain/rpc"
)

// gdaApiBackend implements ethapi.Backend for full nodes
type gdaApiBackend struct {
	gda *gdachain
	gpo *gasprice.Oracle
}

func (b *gdaApiBackend) ChainConfig() *params.ChainConfig {
	return b.gda.chainConfig
}

func (b *gdaApiBackend) CurrentBlock() *types.Block {
	return b.gda.blockchain.CurrentBlock()
}

func (b *gdaApiBackend) SetHead(number uint64) {
	b.gda.protocolManager.downloader.Cancel()
	b.gda.blockchain.SetHead(number)
}

func (b *gdaApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.gda.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.gda.blockchain.CurrentBlock().Header(), nil
	}
	return b.gda.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *gdaApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.gda.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.gda.blockchain.CurrentBlock(), nil
	}
	return b.gda.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *gdaApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.gda.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.gda.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *gdaApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.gda.blockchain.GetBlockByHash(blockHash), nil
}

func (b *gdaApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.gda.chainDb, blockHash, core.GetBlockNumber(b.gda.chainDb, blockHash)), nil
}

func (b *gdaApiBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	receipts := core.GetBlockReceipts(b.gda.chainDb, blockHash, core.GetBlockNumber(b.gda.chainDb, blockHash))
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *gdaApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.gda.blockchain.GetTdByHash(blockHash)
}

func (b *gdaApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.gda.BlockChain(), nil)
	return vm.NewEVM(context, state, b.gda.chainConfig, vmCfg), vmError, nil
}

func (b *gdaApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.gda.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *gdaApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.gda.BlockChain().SubscribeChainEvent(ch)
}

func (b *gdaApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.gda.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *gdaApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.gda.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *gdaApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.gda.BlockChain().SubscribeLogsEvent(ch)
}

func (b *gdaApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.gda.txPool.AddLocal(signedTx)
}

func (b *gdaApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.gda.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *gdaApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.gda.txPool.Get(hash)
}

func (b *gdaApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.gda.txPool.State().GetNonce(addr), nil
}

func (b *gdaApiBackend) Stats() (pending int, queued int) {
	return b.gda.txPool.Stats()
}

func (b *gdaApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.gda.TxPool().Content()
}

func (b *gdaApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.gda.TxPool().SubscribeTxPreEvent(ch)
}

func (b *gdaApiBackend) Downloader() *downloader.Downloader {
	return b.gda.Downloader()
}

func (b *gdaApiBackend) ProtocolVersion() int {
	return b.gda.gdaVersion()
}

func (b *gdaApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *gdaApiBackend) ChainDb() gdadb.Database {
	return b.gda.ChainDb()
}

func (b *gdaApiBackend) EventMux() *event.TypeMux {
	return b.gda.EventMux()
}

func (b *gdaApiBackend) AccountManager() *accounts.Manager {
	return b.gda.AccountManager()
}

func (b *gdaApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.gda.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *gdaApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.gda.bloomRequests)
	}
}
