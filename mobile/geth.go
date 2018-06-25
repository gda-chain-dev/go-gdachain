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

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package ggda

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/gdachain/go-gdachain/core"
	"github.com/gdachain/go-gdachain/gda"
	"github.com/gdachain/go-gdachain/gda/downloader"
	"github.com/gdachain/go-gdachain/gdaclient"
	"github.com/gdachain/go-gdachain/gdastats"
	"github.com/gdachain/go-gdachain/les"
	"github.com/gdachain/go-gdachain/node"
	"github.com/gdachain/go-gdachain/p2p"
	"github.com/gdachain/go-gdachain/p2p/nat"
	"github.com/gdachain/go-gdachain/params"
	whisper "github.com/gdachain/go-gdachain/whisper/whisperv5"
)

// NodeConfig represents the collection of configuration values to fine tune the Ggda
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by go-gdaereum to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Boogdarap nodes used to establish connectivity with the rest of the network.
	BoogdarapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// gdachainEnabled specifies whgdaer the node should run the gdachain protocol.
	gdachainEnabled bool

	// gdachainNetworkID is the network identifier used by the gdachain protocol to
	// decide if remote peers should be accepted or not.
	gdachainNetworkID int64 // uint64 in truth, but Java can't handle that...

	// gdachainGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	gdachainGenesis string

	// gdachainDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	gdachainDatabaseCache int

	// gdachainNegdaats is a negdaats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	gdachainNegdaats string

	// WhisperEnabled specifies whgdaer the node should run the Whisper protocol.
	WhisperEnabled bool
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BoogdarapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	gdachainEnabled:       true,
	gdachainNetworkID:     1,
	gdachainDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Ggda gdachain node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Ggda node.
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BoogdarapNodes == nil || config.BoogdarapNodes.Size() == 0 {
		config.BoogdarapNodes = defaultNodeConfig.BoogdarapNodes
	}
	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.Version,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			BoogdarapNodesV5: config.BoogdarapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	var genesis *core.Genesis
	if config.gdachainGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.gdachainGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.gdachainGenesis == TestnetGenesis() {
			genesis.Config = params.TestnetChainConfig
			if config.gdachainNetworkID == 1 {
				config.gdachainNetworkID = 3
			}
		}
	}
	// Register the gdachain protocol if requested
	if config.gdachainEnabled {
		gdaConf := gda.DefaultConfig
		gdaConf.Genesis = genesis
		gdaConf.SyncMode = downloader.LightSync
		gdaConf.NetworkId = uint64(config.gdachainNetworkID)
		gdaConf.DatabaseCache = config.gdachainDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &gdaConf)
		}); err != nil {
			return nil, fmt.Errorf("gdaereum init: %v", err)
		}
		// If negdaats reporting is requested, do it
		if config.gdachainNegdaats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.Lightgdachain
				ctx.Service(&lesServ)

				return gdastats.New(config.gdachainNegdaats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("negdaats init: %v", err)
			}
		}
	}
	// Register the Whisper protocol if requested
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) {
			return whisper.New(&whisper.DefaultConfig), nil
		}); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetgdachainClient retrieves a client to access the gdachain subsystem.
func (n *Node) GetgdachainClient() (client *gdachainClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &gdachainClient{gdaclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
