package ethconsensusconfig

import (
	"context"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/ledgerwatch/log/v3"

	"github.com/tenderly/erigon/erigon-lib/chain"

	"github.com/tenderly/erigon/consensus"
	"github.com/tenderly/erigon/consensus/aura"
	"github.com/tenderly/erigon/consensus/bor"
	"github.com/tenderly/erigon/consensus/bor/contract"
	"github.com/tenderly/erigon/consensus/bor/heimdall"
	"github.com/tenderly/erigon/consensus/bor/heimdall/span"
	"github.com/tenderly/erigon/consensus/clique"
	"github.com/tenderly/erigon/consensus/ethash"
	"github.com/tenderly/erigon/consensus/ethash/ethashcfg"
	"github.com/tenderly/erigon/consensus/merge"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/node"
	"github.com/tenderly/erigon/node/nodecfg"
	"github.com/tenderly/erigon/params"
	"github.com/tenderly/erigon/turbo/services"
)

func CreateConsensusEngine(ctx context.Context, nodeConfig *nodecfg.Config, chainConfig *chain.Config, config interface{}, notify []string, noVerify bool,
	heimdallClient heimdall.IHeimdallClient, withoutHeimdall bool, blockReader services.FullBlockReader, readonly bool,
	logger log.Logger,
) consensus.Engine {
	var eng consensus.Engine

	switch consensusCfg := config.(type) {
	case *ethashcfg.Config:
		switch consensusCfg.PowMode {
		case ethashcfg.ModeFake:
			logger.Warn("Ethash used in fake mode")
			eng = ethash.NewFaker()
		case ethashcfg.ModeTest:
			logger.Warn("Ethash used in test mode")
			eng = ethash.NewTester(nil, noVerify)
		case ethashcfg.ModeShared:
			logger.Warn("Ethash used in shared mode")
			eng = ethash.NewShared()
		default:
			eng = ethash.New(ethashcfg.Config{
				CachesInMem:      consensusCfg.CachesInMem,
				CachesLockMmap:   consensusCfg.CachesLockMmap,
				DatasetDir:       consensusCfg.DatasetDir,
				DatasetsInMem:    consensusCfg.DatasetsInMem,
				DatasetsOnDisk:   consensusCfg.DatasetsOnDisk,
				DatasetsLockMmap: consensusCfg.DatasetsLockMmap,
			}, notify, noVerify)
		}
	case *params.ConsensusSnapshotConfig:
		if chainConfig.Clique != nil {
			if consensusCfg.InMemory {
				nodeConfig.Dirs.DataDir = ""
			} else {
				if consensusCfg.DBPath != "" {
					if filepath.Base(consensusCfg.DBPath) == "clique" {
						nodeConfig.Dirs.DataDir = filepath.Dir(consensusCfg.DBPath)
					} else {
						nodeConfig.Dirs.DataDir = consensusCfg.DBPath
					}
				}
			}

			var err error
			var db kv.RwDB

			db, err = node.OpenDatabase(ctx, nodeConfig, kv.ConsensusDB, "clique", readonly, logger)

			if err != nil {
				panic(err)
			}

			eng = clique.New(chainConfig, consensusCfg, db, logger)
		}
	case *chain.AuRaConfig:
		if chainConfig.Aura != nil {
			var err error
			var db kv.RwDB

			db, err = node.OpenDatabase(ctx, nodeConfig, kv.ConsensusDB, "aura", readonly, logger)

			if err != nil {
				panic(err)
			}

			eng, err = aura.NewAuRa(chainConfig.Aura, db)
			if err != nil {
				panic(err)
			}
		}
	case *chain.BorConfig:
		// If Matic bor consensus is requested, set it up
		// In order to pass the ethereum transaction tests, we need to set the burn contract which is in the bor config
		// Then, bor != nil will also be enabled for ethash and clique. Only enable Bor for real if there is a validator contract present.
		if chainConfig.Bor != nil && chainConfig.Bor.ValidatorContract != "" {
			genesisContractsClient := contract.NewGenesisContractsClient(chainConfig, chainConfig.Bor.ValidatorContract, chainConfig.Bor.StateReceiverContract, logger)

			spanner := span.NewChainSpanner(contract.ValidatorSet(), chainConfig, withoutHeimdall, logger)

			var err error
			var db kv.RwDB

			db, err = node.OpenDatabase(ctx, nodeConfig, kv.ConsensusDB, "bor", readonly, logger)

			if err != nil {
				panic(err)
			}

			eng = bor.New(chainConfig, db, blockReader, spanner, heimdallClient, genesisContractsClient, logger)
		}
	}

	if eng == nil {
		panic("unknown config" + spew.Sdump(config))
	}

	if chainConfig.TerminalTotalDifficulty == nil {
		return eng
	} else {
		return merge.New(eng) // the Merge
	}
}

func CreateConsensusEngineBareBones(ctx context.Context, chainConfig *chain.Config, logger log.Logger) consensus.Engine {
	var consensusConfig interface{}

	if chainConfig.Clique != nil {
		consensusConfig = params.CliqueSnapshot
	} else if chainConfig.Aura != nil {
		consensusConfig = chainConfig.Aura
	} else if chainConfig.Bor != nil {
		consensusConfig = chainConfig.Bor
	} else {
		var ethashCfg ethashcfg.Config
		ethashCfg.PowMode = ethashcfg.ModeFake
		consensusConfig = &ethashCfg
	}

	return CreateConsensusEngine(ctx, &nodecfg.Config{}, chainConfig, consensusConfig, nil /* notify */, true, /* noVerify */
		nil /* heimdallClient */, true /* withoutHeimdall */, nil /* blockReader */, false /* readonly */, logger)
}
