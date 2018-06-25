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

package les

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
	"github.com/gdachain/go-gdachain/light"
	"github.com/gdachain/go-gdachain/params"
	"github.com/gdachain/go-gdachain/rpc"
)

type LesApiBackend struct {
	gda *Lightgdachain
	gpo *gasprice.Oracle
}

func (b *LesApiBackend) ChainConfig() *params.ChainConfig {
	return b.gda.chainConfig
}

func (b *LesApiBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(b.gda.BlockChain().CurrentHeader())
}

func (b *LesApiBackend) SetHead(number uint64) {
	b.gda.protocolManager.downloader.Cancel()
	b.gda.blockchain.SetHead(number)
}

func (b *LesApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		return b.gda.blockchain.CurrentHeader(), nil
	}

	return b.gda.blockchain.GetHeaderByNumberOdr(ctx, uint64(blockNr))
}

func (b *LesApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, err
	}
	return b.GetBlock(ctx, header.Hash())
}

func (b *LesApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	return light.NewState(ctx, header, b.gda.odr), header, nil
}

func (b *LesApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.gda.blockchain.GetBlockByHash(ctx, blockHash)
}

func (b *LesApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return light.GetBlockReceipts(ctx, b.gda.odr, blockHash, core.GetBlockNumber(b.gda.chainDb, blockHash))
}

func (b *LesApiBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	return light.GetBlockLogs(ctx, b.gda.odr, blockHash, core.GetBlockNumber(b.gda.chainDb, blockHash))
}

func (b *LesApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.gda.blockchain.GetTdByHash(blockHash)
}

func (b *LesApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	context := core.NewEVMContext(msg, header, b.gda.blockchain, nil)
	return vm.NewEVM(context, state, b.gda.chainConfig, vmCfg), state.Error, nil
}

func (b *LesApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.gda.txPool.Add(ctx, signedTx)
}

func (b *LesApiBackend) RemoveTx(txHash common.Hash) {
	b.gda.txPool.RemoveTx(txHash)
}

func (b *LesApiBackend) GetPoolTransactions() (types.Transactions, error) {
	return b.gda.txPool.GetTransactions()
}

func (b *LesApiBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	return b.gda.txPool.GetTransaction(txHash)
}

func (b *LesApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.gda.txPool.GetNonce(ctx, addr)
}

func (b *LesApiBackend) Stats() (pending int, queued int) {
	return b.gda.txPool.Stats(), 0
}

func (b *LesApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.gda.txPool.Content()
}

func (b *LesApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.gda.txPool.SubscribeTxPreEvent(ch)
}

func (b *LesApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.gda.blockchain.SubscribeChainEvent(ch)
}

func (b *LesApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.gda.blockchain.SubscribeChainHeadEvent(ch)
}

func (b *LesApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.gda.blockchain.SubscribeChainSideEvent(ch)
}

func (b *LesApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.gda.blockchain.SubscribeLogsEvent(ch)
}

func (b *LesApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.gda.blockchain.SubscribeRemovedLogsEvent(ch)
}

func (b *LesApiBackend) Downloader() *downloader.Downloader {
	return b.gda.Downloader()
}

func (b *LesApiBackend) ProtocolVersion() int {
	return b.gda.LesVersion() + 10000
}

func (b *LesApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *LesApiBackend) ChainDb() gdadb.Database {
	return b.gda.chainDb
}

func (b *LesApiBackend) EventMux() *event.TypeMux {
	return b.gda.eventMux
}

func (b *LesApiBackend) AccountManager() *accounts.Manager {
	return b.gda.accountManager
}

func (b *LesApiBackend) BloomStatus() (uint64, uint64) {
	if b.gda.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := b.gda.bloomIndexer.Sections()
	return light.BloomTrieFrequency, sections
}

func (b *LesApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.gda.bloomRequests)
	}
}
