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

package gdaclient

import "github.com/gdachain/go-gdachain"

// Verify that Client implements the gdaereum interfaces.
var (
	_ = gdaereum.ChainReader(&Client{})
	_ = gdaereum.TransactionReader(&Client{})
	_ = gdaereum.ChainStateReader(&Client{})
	_ = gdaereum.ChainSyncReader(&Client{})
	_ = gdaereum.ContractCaller(&Client{})
	_ = gdaereum.GasEstimator(&Client{})
	_ = gdaereum.GasPricer(&Client{})
	_ = gdaereum.LogFilterer(&Client{})
	_ = gdaereum.PendingStateReader(&Client{})
	// _ = gdaereum.PendingStateEventer(&Client{})
	_ = gdaereum.PendingContractCaller(&Client{})
)
