package ens

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethstats"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/mobile"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"path/filepath"
	ens "github.com/Arachnid/ensdns/ens"

	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	ens2 "github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/miekg/dns"
)

var ErrorBlockchainSyncing error = errors.New("Cannot resolve names while the chain is syncing")
var ErrorNodeInitializing error = errors.New("Node is still initializing")
var ErrorNoRecords error = errors.New("No DNS records found")

type ENSLiteClient struct {
	node        *node.Node
	dnsService  *ens.Registry
	addrService *ens2.ENS
}

func NewENSLiteClient(dataDir string) (*ENSLiteClient, error) {
	config := geth.NewNodeConfig()
	config.MaxPeers = 25
	config.BootstrapNodes = geth.FoundationBootnodes()
	// Create the empty networking stack

	v5Nodes := make([]*discv5.Node, len(params.DiscoveryV5Bootnodes))
	for i, url := range params.DiscoveryV5Bootnodes {
		v5Nodes[i] = discv5.MustParseNode(url)
	}

	nodeConf := &node.Config{
		Name:        "ens-lite",
		Version:     params.Version,
		DataDir:     dataDir,
		KeyStoreDir: filepath.Join(dataDir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			DiscoveryV5Addr:  ":0",
			BootstrapNodesV5: v5Nodes,
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
	enc, err := json.Marshal(core.DefaultTestnetGenesisBlock())
	if err != nil {
		return nil, err
	}
	if config.EthereumGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.EthereumGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.EthereumGenesis == string(enc) {
			genesis.Config = params.TestnetChainConfig
			if config.EthereumNetworkID == 1 {
				config.EthereumNetworkID = 3
			}
		}
	}
	// Register the Ethereum protocol if requested
	if config.EthereumEnabled {
		ethConf := eth.DefaultConfig
		ethConf.Genesis = genesis
		ethConf.SyncMode = downloader.LightSync
		ethConf.NetworkId = uint64(config.EthereumNetworkID)
		ethConf.DatabaseCache = config.EthereumDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &ethConf)
		}); err != nil {
			return nil, fmt.Errorf("ethereum init: %v", err)
		}
		// If netstats reporting is requested, do it
		if config.EthereumNetStats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.LightEthereum
				ctx.Service(&lesServ)

				return ethstats.New(config.EthereumNetStats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("netstats init: %v", err)
			}
		}
	}
	return &ENSLiteClient{rawStack, nil, nil}, nil
}

// Start the SPV node
func (self *ENSLiteClient) Start() {
	self.node.Start()
}

// Stop the SPV node
func (self *ENSLiteClient) Stop() {
	self.node.Stop()
}

// Resolve a name. The merkle proofs will be validated automatically.
func (self *ENSLiteClient) ResolveDNS(name string) ([]dns.RR, error) {
	var rr []dns.RR
	rpc, err := self.node.Attach()
	if err != nil {
		return rr, err
	}
	api := ethclient.NewClient(rpc)
	sp, _ := api.SyncProgress(context.Background())
	if sp != nil {
		return rr, ErrorBlockchainSyncing
	}
	if self.dnsService == nil {
		reg, err := ens.New(api, common.HexToAddress("0x314159265dD8dbb310642f98f50C066173C1259b"), bind.TransactOpts{})
		if err != nil {
			return rr, err
		}
		self.dnsService = reg
	}
	resolver, err := self.dnsService.GetResolver(name)
	if err != nil {
		return rr, err
	}
	return resolver.GetRRs()
}

func (self *ENSLiteClient) ResolveAddress(name string) (addr common.Hash, err error) {
	rpc, err := self.node.Attach()
	if err != nil {
		return addr, err
	}
	api := ethclient.NewClient(rpc)
	sp, _ := api.SyncProgress(context.Background())
	if sp != nil {
		return addr, ErrorBlockchainSyncing
	}
	if self.addrService == nil {
		reg, err := ens2.NewENS(&bind.TransactOpts{}, common.HexToAddress("0x314159265dD8dbb310642f98f50C066173C1259b"), api)
		if err != nil {
			return addr, err
		}
		self.addrService = reg
	}
	return self.addrService.Resolve(name)
}

func (self *ENSLiteClient) SyncProgress() (*ethereum.SyncProgress, error) {
	if self.node == nil {
		return nil, ErrorNodeInitializing
	}
	rpc, err := self.node.Attach()
	if err != nil {
		return nil, err
	}
	api := ethclient.NewClient(rpc)
	return api.SyncProgress(context.Background())
}
