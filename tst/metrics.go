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
	"github.com/gdachain/go-gdachain/metrics"
	"github.com/gdachain/go-gdachain/p2p"
)

var (
	propTxnInPacketsMeter     = metrics.NewRegisteredMeter("gda/prop/txns/in/packets", nil)
	propTxnInTrafficMeter     = metrics.NewRegisteredMeter("gda/prop/txns/in/traffic", nil)
	propTxnOutPacketsMeter    = metrics.NewRegisteredMeter("gda/prop/txns/out/packets", nil)
	propTxnOutTrafficMeter    = metrics.NewRegisteredMeter("gda/prop/txns/out/traffic", nil)
	propHashInPacketsMeter    = metrics.NewRegisteredMeter("gda/prop/hashes/in/packets", nil)
	propHashInTrafficMeter    = metrics.NewRegisteredMeter("gda/prop/hashes/in/traffic", nil)
	propHashOutPacketsMeter   = metrics.NewRegisteredMeter("gda/prop/hashes/out/packets", nil)
	propHashOutTrafficMeter   = metrics.NewRegisteredMeter("gda/prop/hashes/out/traffic", nil)
	propBlockInPacketsMeter   = metrics.NewRegisteredMeter("gda/prop/blocks/in/packets", nil)
	propBlockInTrafficMeter   = metrics.NewRegisteredMeter("gda/prop/blocks/in/traffic", nil)
	propBlockOutPacketsMeter  = metrics.NewRegisteredMeter("gda/prop/blocks/out/packets", nil)
	propBlockOutTrafficMeter  = metrics.NewRegisteredMeter("gda/prop/blocks/out/traffic", nil)
	reqHeaderInPacketsMeter   = metrics.NewRegisteredMeter("gda/req/headers/in/packets", nil)
	reqHeaderInTrafficMeter   = metrics.NewRegisteredMeter("gda/req/headers/in/traffic", nil)
	reqHeaderOutPacketsMeter  = metrics.NewRegisteredMeter("gda/req/headers/out/packets", nil)
	reqHeaderOutTrafficMeter  = metrics.NewRegisteredMeter("gda/req/headers/out/traffic", nil)
	reqBodyInPacketsMeter     = metrics.NewRegisteredMeter("gda/req/bodies/in/packets", nil)
	reqBodyInTrafficMeter     = metrics.NewRegisteredMeter("gda/req/bodies/in/traffic", nil)
	reqBodyOutPacketsMeter    = metrics.NewRegisteredMeter("gda/req/bodies/out/packets", nil)
	reqBodyOutTrafficMeter    = metrics.NewRegisteredMeter("gda/req/bodies/out/traffic", nil)
	reqStateInPacketsMeter    = metrics.NewRegisteredMeter("gda/req/states/in/packets", nil)
	reqStateInTrafficMeter    = metrics.NewRegisteredMeter("gda/req/states/in/traffic", nil)
	reqStateOutPacketsMeter   = metrics.NewRegisteredMeter("gda/req/states/out/packets", nil)
	reqStateOutTrafficMeter   = metrics.NewRegisteredMeter("gda/req/states/out/traffic", nil)
	reqReceiptInPacketsMeter  = metrics.NewRegisteredMeter("gda/req/receipts/in/packets", nil)
	reqReceiptInTrafficMeter  = metrics.NewRegisteredMeter("gda/req/receipts/in/traffic", nil)
	reqReceiptOutPacketsMeter = metrics.NewRegisteredMeter("gda/req/receipts/out/packets", nil)
	reqReceiptOutTrafficMeter = metrics.NewRegisteredMeter("gda/req/receipts/out/traffic", nil)
	miscInPacketsMeter        = metrics.NewRegisteredMeter("gda/misc/in/packets", nil)
	miscInTrafficMeter        = metrics.NewRegisteredMeter("gda/misc/in/traffic", nil)
	miscOutPacketsMeter       = metrics.NewRegisteredMeter("gda/misc/out/packets", nil)
	miscOutTrafficMeter       = metrics.NewRegisteredMeter("gda/misc/out/traffic", nil)
)

// meteredMsgReadWriter is a wrapper around a p2p.MsgReadWriter, capable of
// accumulating the above defined metrics based on the data stream contents.
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

// Init sets the protocol version used by the stream to know which meters to
// increment in case of overlapping message ids between protocol versions.
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	// Read the message and short circuit in case of an error
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
	// Account for the data traffic
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	switch {
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqHeaderInPacketsMeter, reqHeaderInTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqBodyInPacketsMeter, reqBodyInTrafficMeter

	case rw.version >= gda63 && msg.Code == NodeDataMsg:
		packets, traffic = reqStateInPacketsMeter, reqStateInTrafficMeter
	case rw.version >= gda63 && msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptInPacketsMeter, reqReceiptInTrafficMeter

	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propHashInPacketsMeter, propHashInTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propBlockInPacketsMeter, propBlockInTrafficMeter
	case msg.Code == TxMsg:
		packets, traffic = propTxnInPacketsMeter, propTxnInTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	// Account for the data traffic
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	switch {
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqHeaderOutPacketsMeter, reqHeaderOutTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqBodyOutPacketsMeter, reqBodyOutTrafficMeter

	case rw.version >= gda63 && msg.Code == NodeDataMsg:
		packets, traffic = reqStateOutPacketsMeter, reqStateOutTrafficMeter
	case rw.version >= gda63 && msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptOutPacketsMeter, reqReceiptOutTrafficMeter

	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propHashOutPacketsMeter, propHashOutTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propBlockOutPacketsMeter, propBlockOutTrafficMeter
	case msg.Code == TxMsg:
		packets, traffic = propTxnOutPacketsMeter, propTxnOutTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	// Send the packet to the p2p layer
	return rw.MsgReadWriter.WriteMsg(msg)
}
