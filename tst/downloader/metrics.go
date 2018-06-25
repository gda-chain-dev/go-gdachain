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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/gdachain/go-gdachain/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("gda/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("gda/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("gda/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("gda/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("gda/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("gda/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("gda/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("gda/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("gda/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("gda/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("gda/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("gda/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("gda/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("gda/downloader/states/drop", nil)
)
