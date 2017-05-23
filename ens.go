package ens

import (
	"github.com/ethereum/go-ethereum/common"
	ens"github.com/ethereum/go-ethereum/contracts/ens/contract"

	"github.com/ethereum/go-ethereum/crypto"
	"strings"

	"github.com/ethereum/go-ethereum/mobile"
	"path/filepath"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/node"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/ethstats"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ENSLiteClient struct {
	node *node.Node
	nameService *ens.ENS
}

func NewENSLiteClient(dataDir string) (*ENSLiteClient, error){
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
	return &ENSLiteClient{rawStack, nil}, nil
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
func (self *ENSLiteClient) Resolve(name string) ([32]byte, error) {
	var h [32]byte
	rpc, err := self.node.Attach()
	if err != nil {
		return h, err
	}
	api := ethclient.NewClient(rpc)
	if self.nameService == nil {
		// Geth must be patched here to return the RawClient()
		ns, err := ens.NewENS(common.HexToAddress("0x314159265dD8dbb310642f98f50C066173C1259b"), api)
		if err != nil {
			return h, err
		}
		self.nameService = ns
	}
	resolverAddress, err := self.nameService.Resolver(nil, ensNode(name))
	if err != nil {
		return h, err
	}
	resolver, err := ens.NewResolver(resolverAddress, api)
	if err != nil {
		return h, err
	}
	h, err = resolver.Content(nil, ensNode(name))
	if err != nil {
		return h, err
	}
	return h, nil
}

func ensParentNode(name string) (common.Hash, common.Hash) {
	parts := strings.SplitN(name, ".", 2)
	label := crypto.Keccak256Hash([]byte(parts[0]))
	if len(parts) == 1 {
		return [32]byte{}, label
	} else {
		parentNode, parentLabel := ensParentNode(parts[1])
		return crypto.Keccak256Hash(parentNode[:], parentLabel[:]), label
	}
}

func ensNode(name string) common.Hash {
	parentNode, parentLabel := ensParentNode(name)
	return crypto.Keccak256Hash(parentNode[:], parentLabel[:])
}
