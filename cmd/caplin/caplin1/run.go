package caplin1

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/tenderly/erigon/cl/antiquary"
	"github.com/tenderly/erigon/cl/beacon"
	"github.com/tenderly/erigon/cl/beacon/beacon_router_configuration"
	"github.com/tenderly/erigon/cl/beacon/handler"
	"github.com/tenderly/erigon/cl/beacon/synced_data"
	"github.com/tenderly/erigon/cl/beacon/validatorapi"
	"github.com/tenderly/erigon/cl/clparams/initial_state"
	"github.com/tenderly/erigon/cl/cltypes/solid"
	"github.com/tenderly/erigon/cl/freezer"
	freezer2 "github.com/tenderly/erigon/cl/freezer"
	proto_downloader "github.com/tenderly/erigon/erigon-lib/gointerfaces/downloader"
	"github.com/tenderly/erigon/eth/ethconfig"
	"github.com/tenderly/erigon/turbo/snapshotsync/freezeblocks"

	"github.com/spf13/afero"
	"github.com/tenderly/erigon/cl/persistence"
	persistence2 "github.com/tenderly/erigon/cl/persistence"
	"github.com/tenderly/erigon/cl/persistence/beacon_indicies"
	"github.com/tenderly/erigon/cl/persistence/db_config"
	"github.com/tenderly/erigon/cl/persistence/format/snapshot_format"
	state_accessors "github.com/tenderly/erigon/cl/persistence/state"
	"github.com/tenderly/erigon/cl/phase1/core/state"
	"github.com/tenderly/erigon/cl/phase1/execution_client"
	"github.com/tenderly/erigon/cl/phase1/forkchoice"
	"github.com/tenderly/erigon/cl/phase1/forkchoice/fork_graph"
	"github.com/tenderly/erigon/cl/phase1/network"
	"github.com/tenderly/erigon/cl/phase1/stages"
	"github.com/tenderly/erigon/cl/pool"
	"github.com/tenderly/erigon/cl/rpc"

	"github.com/Giulio2002/bls"
	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/erigon-lib/common/datadir"
	"github.com/tenderly/erigon/erigon-lib/gointerfaces/sentinel"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/erigon-lib/kv/mdbx"
)

func OpenCaplinDatabase(ctx context.Context,
	databaseConfig db_config.DatabaseConfiguration,
	beaconConfig *clparams.BeaconChainConfig,
	rawBeaconChain persistence2.RawBeaconBlockChain,
	dbPath string,
	engine execution_client.ExecutionEngine,
	wipeout bool,
) (persistence.BeaconChainDatabase, kv.RwDB, error) {
	dataDirIndexer := path.Join(dbPath, "beacon_indicies")
	if wipeout {
		os.RemoveAll(dataDirIndexer)
	}

	os.MkdirAll(dbPath, 0700)

	db := mdbx.MustOpen(dataDirIndexer)

	tx, err := db.BeginRw(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	if err := db_config.WriteConfigurationIfNotExist(ctx, tx, databaseConfig); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	{ // start ticking forkChoice
		go func() {
			<-ctx.Done()
			db.Close() // close sql database here
		}()
	}
	return persistence2.NewBeaconChainDatabaseFilesystem(rawBeaconChain, engine, beaconConfig), db, nil
}

func RunCaplinPhase1(ctx context.Context, sentinel sentinel.SentinelClient, engine execution_client.ExecutionEngine,
	beaconConfig *clparams.BeaconChainConfig, genesisConfig *clparams.GenesisConfig, state *state.CachingBeaconState,
	caplinFreezer freezer.Freezer, dirs datadir.Dirs, cfg beacon_router_configuration.RouterConfiguration, eth1Getter snapshot_format.ExecutionBlockReaderByNumber,
	snDownloader proto_downloader.DownloaderClient, backfilling bool, states bool) error {
	rawDB, af := persistence.AferoRawBeaconBlockChainFromOsPath(beaconConfig, dirs.CaplinHistory)
	beaconDB, db, err := OpenCaplinDatabase(ctx, db_config.DefaultDatabaseConfiguration, beaconConfig, rawDB, dirs.CaplinIndexing, engine, false)
	if err != nil {
		return err
	}
	ctx, cn := context.WithCancel(ctx)
	defer cn()

	beaconRpc := rpc.NewBeaconRpcP2P(ctx, sentinel, beaconConfig, genesisConfig)

	logger := log.New("app", "caplin")

	csn := freezeblocks.NewCaplinSnapshots(ethconfig.BlocksFreezing{}, dirs.Snap, logger)
	rcsn := freezeblocks.NewBeaconSnapshotReader(csn, eth1Getter, beaconDB, beaconConfig)

	if caplinFreezer != nil {
		if err := freezer2.PutObjectSSZIntoFreezer("beaconState", "caplin_core", 0, state, caplinFreezer); err != nil {
			return err
		}
	}

	pool := pool.NewOperationsPool(beaconConfig)

	caplinFcuPath := path.Join(dirs.Tmp, "caplin-forkchoice")
	os.RemoveAll(caplinFcuPath)
	err = os.MkdirAll(caplinFcuPath, 0o755)
	if err != nil {
		return err
	}
	fcuFs := afero.NewBasePathFs(afero.NewOsFs(), caplinFcuPath)

	forkChoice, err := forkchoice.NewForkChoiceStore(ctx, state, engine, caplinFreezer, pool, fork_graph.NewForkGraphDisk(state, fcuFs))
	if err != nil {
		logger.Error("Could not create forkchoice", "err", err)
		return err
	}
	bls.SetEnabledCaching(true)
	state.ForEachValidator(func(v solid.Validator, idx, total int) bool {
		pk := v.PublicKey()
		if err := bls.LoadPublicKeyIntoCache(pk[:], false); err != nil {
			panic(err)
		}
		return true
	})
	gossipManager := network.NewGossipReceiver(sentinel, forkChoice, beaconConfig, genesisConfig, caplinFreezer)
	{ // start ticking forkChoice
		go func() {
			tickInterval := time.NewTicker(50 * time.Millisecond)
			for {
				select {
				case <-tickInterval.C:
					forkChoice.OnTick(uint64(time.Now().Unix()))
				case <-ctx.Done():
					return
				}

			}
		}()
	}

	syncedDataManager := synced_data.NewSyncedDataManager(cfg.Active, beaconConfig)
	if cfg.Active {
		apiHandler := handler.NewApiHandler(genesisConfig, beaconConfig, rawDB, db, forkChoice, pool, rcsn, syncedDataManager)
		headApiHandler := &validatorapi.ValidatorApiHandler{
			FC:             forkChoice,
			BeaconChainCfg: beaconConfig,
			GenesisCfg:     genesisConfig,
		}
		go beacon.ListenAndServe(&beacon.LayeredBeaconHandler{
			ValidatorApi: headApiHandler,
			ArchiveApi:   apiHandler,
		}, cfg)
		log.Info("Beacon API started", "addr", cfg.Address)
	}

	{ // start the gossip manager
		go gossipManager.Start(ctx)
		logger.Info("Started Ethereum 2.0 Gossip Service")
	}

	{ // start logging peers
		go func() {
			logIntervalPeers := time.NewTicker(1 * time.Minute)
			for {
				select {
				case <-logIntervalPeers.C:
					if peerCount, err := beaconRpc.Peers(); err == nil {
						logger.Info("P2P", "peers", peerCount)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	tx, err := db.BeginRw(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	dbConfig, err := db_config.ReadConfiguration(ctx, tx)
	if err != nil {
		return err
	}

	if err := state_accessors.InitializeStaticTables(tx, state); err != nil {
		return err
	}
	if err := beacon_indicies.WriteHighestFinalized(tx, 0); err != nil {
		return err
	}

	vTables := state_accessors.NewStaticValidatorTable()
	// Read the the current table
	if states {
		if err := state_accessors.ReadValidatorsTable(tx, vTables); err != nil {
			return err
		}
	}
	// get the initial state
	genesisState, err := initial_state.GetGenesisState(clparams.NetworkType(beaconConfig.DepositNetworkID))
	if err != nil {
		return err
	}
	antiq := antiquary.NewAntiquary(ctx, genesisState, vTables, beaconConfig, dirs, snDownloader, db, csn, rcsn, beaconDB, logger, states, af)
	// Create the antiquary
	go func() {
		if err := antiq.Loop(); err != nil {
			logger.Error("Antiquary failed", "err", err)
		}
	}()

	if err := tx.Commit(); err != nil {
		return err
	}

	stageCfg := stages.ClStagesCfg(beaconRpc, antiq, genesisConfig, beaconConfig, state, engine, gossipManager, forkChoice, beaconDB, db, csn, dirs.Tmp, dbConfig, backfilling, syncedDataManager)
	sync := stages.ConsensusClStages(ctx, stageCfg)

	logger.Info("[Caplin] starting clstages loop")
	err = sync.StartWithStage(ctx, "WaitForPeers", logger, stageCfg)
	logger.Info("[Caplin] exiting clstages loop")
	if err != nil {
		return err
	}
	return err
}
