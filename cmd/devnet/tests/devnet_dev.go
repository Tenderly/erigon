package tests

import (
	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/cmd/devnet/accounts"
	"github.com/tenderly/erigon/cmd/devnet/args"
	"github.com/tenderly/erigon/cmd/devnet/devnet"
	account_services "github.com/tenderly/erigon/cmd/devnet/services/accounts"
	"github.com/tenderly/erigon/core/types"
	"github.com/tenderly/erigon/erigon-lib/chain/networkname"
)

func NewDevDevnet(
	dataDir string,
	baseRpcHost string,
	baseRpcPort int,
	logger log.Logger,
) devnet.Devnet {
	faucetSource := accounts.NewAccount("faucet-source")

	network := devnet.Network{
		DataDir:            dataDir,
		Chain:              networkname.DevChainName,
		Logger:             logger,
		BasePrivateApiAddr: "localhost:10090",
		BaseRPCHost:        baseRpcHost,
		BaseRPCPort:        baseRpcPort,
		Alloc: types.GenesisAlloc{
			faucetSource.Address: {Balance: accounts.EtherAmount(200_000)},
		},
		Services: []devnet.Service{
			account_services.NewFaucet(networkname.DevChainName, faucetSource),
		},
		MaxNumberOfEmptyBlockChecks: 30,
		Nodes: []devnet.Node{
			&args.BlockProducer{
				NodeArgs: args.NodeArgs{
					ConsoleVerbosity: "0",
					DirVerbosity:     "5",
				},
				AccountSlots: 200,
			},
			&args.NonBlockProducer{
				NodeArgs: args.NodeArgs{
					ConsoleVerbosity: "0",
					DirVerbosity:     "5",
				},
			},
		},
	}

	return devnet.Devnet{&network}
}
