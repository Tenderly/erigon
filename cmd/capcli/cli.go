package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/tenderly/erigon/eth/ethconfig/estimate"
	"github.com/tenderly/erigon/turbo/debug"

	lg "github.com/anacrolix/log"
	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/direct"
	downloader3 "github.com/tenderly/erigon/erigon-lib/downloader"
	"github.com/tenderly/erigon/erigon-lib/metrics"
	state2 "github.com/tenderly/erigon/erigon-lib/state"

	"github.com/c2h5oh/datasize"
	"github.com/tenderly/erigon/erigon-lib/chain/snapcfg"
	"github.com/tenderly/erigon/erigon-lib/downloader"

	"github.com/tenderly/erigon/cl/abstract"
	"github.com/tenderly/erigon/cl/antiquary"
	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/cl/clparams/initial_state"
	"github.com/tenderly/erigon/cl/cltypes"
	persistence2 "github.com/tenderly/erigon/cl/persistence"
	"github.com/tenderly/erigon/cmd/caplin/caplin1"
	"github.com/tenderly/erigon/eth/ethconfig"
	"github.com/tenderly/erigon/params"
	"github.com/tenderly/erigon/turbo/snapshotsync"
	"github.com/tenderly/erigon/turbo/snapshotsync/freezeblocks"

	"github.com/tenderly/erigon/cl/persistence"
	"github.com/tenderly/erigon/cl/persistence/beacon_indicies"
	"github.com/tenderly/erigon/cl/persistence/db_config"
	"github.com/tenderly/erigon/cl/persistence/format/snapshot_format"
	"github.com/tenderly/erigon/cl/persistence/format/snapshot_format/getters"
	state_accessors "github.com/tenderly/erigon/cl/persistence/state"
	"github.com/tenderly/erigon/cl/persistence/state/historical_states_reader"
	"github.com/tenderly/erigon/cl/phase1/core"
	"github.com/tenderly/erigon/cl/phase1/core/state"
	"github.com/tenderly/erigon/cl/phase1/network"
	"github.com/tenderly/erigon/cl/phase1/stages"
	"github.com/tenderly/erigon/cl/rpc"
	"github.com/tenderly/erigon/cl/sentinel/peers"
	"github.com/tenderly/erigon/cl/transition/impl/eth2"
	"github.com/tenderly/erigon/cl/transition/machine"
	"github.com/tenderly/erigon/cl/utils"
	"github.com/tenderly/erigon/erigon-lib/common/datadir"
	"github.com/tenderly/erigon/erigon-lib/downloader/downloadercfg"
	"github.com/tenderly/erigon/erigon-lib/downloader/snaptype"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/erigon-lib/kv/mdbx"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/ledgerwatch/log/v3"
	"github.com/spf13/afero"
	"github.com/tenderly/erigon/erigon-lib/gointerfaces/sentinel"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var CLI struct {
	Migrate Migrate `cmd:"" help:"migrate from one state to another"`

	Blocks Blocks `cmd:"" help:"download blocks from reqresp network"`
	Epochs Epochs `cmd:"" help:"download epochs from reqresp network"`

	Chain                   Chain                   `cmd:"" help:"download the entire chain from reqresp network"`
	DumpSnapshots           DumpSnapshots           `cmd:"" help:"generate caplin snapshots"`
	CheckSnapshots          CheckSnapshots          `cmd:"" help:"check snapshot folder against content of chain data"`
	DownloadSnapshots       DownloadSnapshots       `cmd:"" help:"download snapshots from webseed"`
	LoopSnapshots           LoopSnapshots           `cmd:"" help:"loop over snapshots"`
	RetrieveHistoricalState RetrieveHistoricalState `cmd:"" help:"retrieve historical state from db"`
}

type chainCfg struct {
	Chain string `help:"chain" default:"mainnet"`
}

func (c *chainCfg) configs() (beaconConfig *clparams.BeaconChainConfig, genesisConfig *clparams.GenesisConfig, err error) {
	genesisConfig, _, beaconConfig, _, err = clparams.GetConfigsByNetworkName(c.Chain)
	return
}

type outputFolder struct {
	Datadir string `help:"datadir" default:"~/.local/share/erigon" type:"existingdir"`
}

type withSentinel struct {
	Sentinel string `help:"sentinel url" default:"localhost:7777"`
}

type withPPROF struct {
	Pprof bool `help:"enable pprof" default:"false"`
}

func (w *withPPROF) withProfile() {
	if w.Pprof {
		debug.StartPProf("localhost:6060", metrics.Setup("localhost:6060", log.Root()))
	}
}

func (w *withSentinel) connectSentinel() (sentinel.SentinelClient, error) {
	// YOLO message size
	gconn, err := grpc.Dial(w.Sentinel, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt)))
	if err != nil {
		return nil, err
	}
	return sentinel.NewSentinelClient(gconn), nil
}

func openFs(fsName string, path string) (afero.Fs, error) {
	return afero.NewBasePathFs(afero.NewBasePathFs(afero.NewOsFs(), fsName), path), nil
}

type Blocks struct {
	chainCfg
	outputFolder
	withSentinel

	FromBlock int `arg:"" name:"from" default:"0"`
	ToBlock   int `arg:"" name:"to" default:"-1"`
}

func (b *Blocks) Run(ctx *Context) error {
	s, err := b.withSentinel.connectSentinel()
	if err != nil {
		return err
	}
	beaconConfig, genesisConfig, err := b.configs()
	if err != nil {
		return err
	}

	beacon := rpc.NewBeaconRpcP2P(ctx, s, beaconConfig, genesisConfig)
	err = beacon.SetStatus(
		genesisConfig.GenesisValidatorRoot,
		beaconConfig.GenesisEpoch,
		genesisConfig.GenesisValidatorRoot,
		beaconConfig.GenesisSlot)
	if err != nil {
		return err
	}

	if b.ToBlock < 0 {
		b.ToBlock = int(utils.GetCurrentSlot(genesisConfig.GenesisTime, beaconConfig.SecondsPerSlot))
	}

	resp, _, err := beacon.SendBeaconBlocksByRangeReq(ctx, uint64(b.FromBlock), uint64(b.ToBlock))
	if err != nil {
		return fmt.Errorf("error get beacon blocks: %w", err)
	}
	aferoFS, err := openFs(b.Datadir, "caplin/beacon")
	if err != nil {
		return err
	}

	db := mdbx.MustOpen("caplin/db")
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginRw(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	beaconDB := persistence2.NewBeaconChainDatabaseFilesystem(persistence2.NewAferoRawBlockSaver(aferoFS, beaconConfig), nil, beaconConfig)
	for _, vv := range resp {
		err := beaconDB.WriteBlock(ctx, tx, vv, true)
		if err != nil {
			return err
		}
	}
	return nil
}

type Epochs struct {
	chainCfg
	outputFolder
	withSentinel

	Concurrency int `help:"number of epochs to ask concurrently for" name:"concurrency" short:"c" default:"4"`

	FromEpoch int `arg:"" name:"from" default:"0"`
	ToEpoch   int `arg:"" name:"to" default:"-1"`
}

func (b *Epochs) Run(cctx *Context) error {
	ctx := cctx.Context
	s, err := b.withSentinel.connectSentinel()
	if err != nil {
		return err
	}
	beaconConfig, genesisConfig, err := b.configs()
	if err != nil {
		return err
	}

	aferoFS, err := openFs(b.Datadir, "caplin/beacon")
	if err != nil {
		return err
	}

	beaconDB := persistence.NewBeaconChainDatabaseFilesystem(persistence.NewAferoRawBlockSaver(aferoFS, beaconConfig), nil, beaconConfig)

	beacon := rpc.NewBeaconRpcP2P(ctx, s, beaconConfig, genesisConfig)
	rpcSource := persistence2.NewBeaconRpcSource(beacon)

	err = beacon.SetStatus(
		genesisConfig.GenesisValidatorRoot,
		beaconConfig.GenesisEpoch,
		genesisConfig.GenesisValidatorRoot,
		beaconConfig.GenesisSlot)
	if err != nil {
		return err
	}

	if b.ToEpoch < 0 {
		b.ToEpoch = int(utils.GetCurrentEpoch(genesisConfig.GenesisTime, beaconConfig.SecondsPerSlot, beaconConfig.SlotsPerEpoch))
	}

	ctx, cn := context.WithCancel(ctx)
	defer cn()
	egg, ctx := errgroup.WithContext(ctx)

	totalEpochs := (b.ToEpoch - b.FromEpoch + 1)
	pw := progress.NewWriter()
	pw.SetTrackerLength(50)
	pw.SetMessageWidth(24)
	pw.SetStyle(progress.StyleDefault)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.Style().Visibility.Percentage = true
	pw.Style().Visibility.Speed = true
	pw.Style().Visibility.Value = true
	pw.Style().Visibility.ETA = true
	pw.Style().Visibility.ETAOverall = false
	pw.Style().Visibility.Tracker = true
	pw.Style().Visibility.TrackerOverall = false
	pw.Style().Visibility.SpeedOverall = true
	pw.Style().Options.Separator = ""

	go pw.Render()

	total := int64(uint64(totalEpochs) * beaconConfig.SlotsPerEpoch)
	tk := &progress.Tracker{
		Message: fmt.Sprintf("downloading %d blocks", total),
		Total:   total,
		Units:   progress.UnitsDefault,
	}
	pw.AppendTracker(tk)
	tk.UpdateTotal(total)

	egg.SetLimit(b.Concurrency)

	db := mdbx.MustOpen("caplin/db")
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginRw(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	defer cn()
	for i := b.FromEpoch; i <= b.ToEpoch; i = i + 1 {
		ii := i
		egg.Go(func() error {
			var blocks *peers.PeeredObject[[]*cltypes.SignedBeaconBlock]
			for {
				blocks, err = rpcSource.GetRange(ctx, tx, uint64(ii)*beaconConfig.SlotsPerEpoch, beaconConfig.SlotsPerEpoch)
				if err != nil {
					log.Error("dl error", "err", err, "epoch", ii)
				} else {
					break
				}
			}
			for _, v := range blocks.Data {
				tk.Increment(1)
				_, _ = beaconDB, v
				err := beaconDB.WriteBlock(ctx, tx, v, true)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	err = egg.Wait()
	if err != nil {
		return err
	}
	tk.MarkAsDone()
	return tx.Commit()
}

type Migrate struct {
	outputFolder
	chainCfg

	State  string   `arg:"" help:"state to start from (file or later url to checkpoint)"`
	Blocks []string `arg:"" name:"blocks" help:"blocks to migrate, in order" type:"path"`
}

func resolveState(source string, chain chainCfg) (abstract.BeaconState, error) {
	beaconConfig, _, err := chain.configs()
	if err != nil {
		return nil, err
	}
	s := state.New(beaconConfig)
	switch {
	default:
		var stateByte []byte
		if _, stateByte, err = clparams.ParseGenesisSSZToGenesisConfig(
			source,
			beaconConfig.GetCurrentStateVersion(0)); err != nil {
			return nil, err
		}
		if s.DecodeSSZ(stateByte, int(beaconConfig.GetCurrentStateVersion(0))); err != nil {
			return nil, err
		}
		return s, nil
	case strings.HasPrefix(strings.ToLower(source), "http://"), strings.HasPrefix(strings.ToLower(source), "https://"):
	}
	return nil, fmt.Errorf("unknown state format: '%s'", source)
}

func (m *Migrate) getBlock(ctx *Context, block string) (*cltypes.SignedBeaconBlock, error) {
	afs := afero.NewOsFs()

	bts, err := afero.ReadFile(afs, block)
	if err != nil {
		return nil, err
	}
	b, _, err := m.chainCfg.configs()
	if err != nil {
		return nil, err
	}
	blk := cltypes.NewSignedBeaconBlock(b)
	err = blk.DecodeSSZ(bts, 0)
	if err != nil {
		return nil, err
	}

	return blk, nil
}

func (m *Migrate) Run(ctx *Context) error {
	state, err := resolveState(m.State, m.chainCfg)
	if err != nil {
		return err
	}
	// get the machine
	cl := &eth2.Impl{}

	// TODO: two queues for download and transition
	for _, v := range m.Blocks {
		blk, err := m.getBlock(ctx, v)
		if err != nil {
			return err
		}
		err = machine.TransitionState(cl, state, blk)
		if err != nil {
			return err
		}
	}
	return nil
}

type Chain struct {
	chainCfg
	withSentinel
	outputFolder
}

func (c *Chain) Run(ctx *Context) error {
	s, err := c.withSentinel.connectSentinel()
	if err != nil {
		return err
	}

	genesisConfig, _, beaconConfig, networkType, err := clparams.GetConfigsByNetworkName(c.Chain)
	if err != nil {
		return err
	}
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StderrHandler))
	log.Info("Started chain download", "chain", c.Chain)

	dirs := datadir.New(c.Datadir)
	csn := freezeblocks.NewCaplinSnapshots(ethconfig.BlocksFreezing{}, dirs.Snap, log.Root())

	rawDB, _ := persistence.AferoRawBeaconBlockChainFromOsPath(beaconConfig, dirs.CaplinHistory)
	beaconDB, db, err := caplin1.OpenCaplinDatabase(ctx, db_config.DatabaseConfiguration{PruneDepth: math.MaxUint64}, beaconConfig, rawDB, dirs.CaplinIndexing, nil, false)
	if err != nil {
		return err
	}
	defer db.Close()

	beacon := rpc.NewBeaconRpcP2P(ctx, s, beaconConfig, genesisConfig)

	bs, err := core.RetrieveBeaconState(ctx, beaconConfig, genesisConfig, clparams.GetCheckpointSyncEndpoint(networkType))
	if err != nil {
		return err
	}

	bRoot, err := bs.BlockRoot()
	if err != nil {
		return err
	}

	if err := db.Update(ctx, func(tx kv.RwTx) error {
		return beacon_indicies.WriteHighestFinalized(tx, bs.Slot())
	}); err != nil {
		return err
	}

	err = beacon.SetStatus(
		genesisConfig.GenesisValidatorRoot,
		beaconConfig.GenesisEpoch,
		genesisConfig.GenesisValidatorRoot,
		beaconConfig.GenesisSlot)
	if err != nil {
		return err
	}

	downloader := network.NewBackwardBeaconDownloader(ctx, beacon, db)
	cfg := stages.StageHistoryReconstruction(downloader, antiquary.NewAntiquary(ctx, nil, nil, nil, dirs, nil, nil, nil, nil, nil, nil, false, nil), csn, beaconDB, db, nil, genesisConfig, beaconConfig, true, true, bRoot, bs.Slot(), "/tmp", log.Root())
	return stages.SpawnStageHistoryDownload(cfg, ctx, log.Root())
}

type DumpSnapshots struct {
	chainCfg
	outputFolder
}

func (c *DumpSnapshots) Run(ctx *Context) error {
	_, _, beaconConfig, _, err := clparams.GetConfigsByNetworkName(c.Chain)
	if err != nil {
		return err
	}
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StderrHandler))
	log.Info("Started chain download", "chain", c.Chain)

	dirs := datadir.New(c.Datadir)
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StderrHandler))

	rawDB, _ := persistence.AferoRawBeaconBlockChainFromOsPath(beaconConfig, dirs.CaplinHistory)
	beaconDB, db, err := caplin1.OpenCaplinDatabase(ctx, db_config.DatabaseConfiguration{PruneDepth: math.MaxUint64}, beaconConfig, rawDB, dirs.CaplinIndexing, nil, false)
	if err != nil {
		return err
	}
	var to uint64
	db.View(ctx, func(tx kv.Tx) (err error) {
		to, err = beacon_indicies.ReadHighestFinalized(tx)
		return
	})

	return freezeblocks.DumpBeaconBlocks(ctx, db, beaconDB, 0, to, snaptype.Erigon2RecentMergeLimit, dirs.Tmp, dirs.Snap, estimate.CompressSnapshot.Workers(), log.LvlInfo, log.Root())
}

type CheckSnapshots struct {
	chainCfg
	outputFolder
	withPPROF
}

func (c *CheckSnapshots) Run(ctx *Context) error {
	_, _, beaconConfig, _, err := clparams.GetConfigsByNetworkName(c.Chain)
	if err != nil {
		return err
	}
	c.withProfile()
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StderrHandler))
	log.Info("Started the checking process", "chain", c.Chain)
	dirs := datadir.New(c.Datadir)
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StderrHandler))

	rawDB, _ := persistence.AferoRawBeaconBlockChainFromOsPath(beaconConfig, dirs.CaplinHistory)
	_, db, err := caplin1.OpenCaplinDatabase(ctx, db_config.DatabaseConfiguration{PruneDepth: math.MaxUint64}, beaconConfig, rawDB, dirs.CaplinIndexing, nil, false)
	if err != nil {
		return err
	}
	var to uint64
	tx, err := db.BeginRo(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	to, err = beacon_indicies.ReadHighestFinalized(tx)
	if err != nil {
		return err
	}

	to = (to / snaptype.Erigon2RecentMergeLimit) * snaptype.Erigon2RecentMergeLimit

	csn := freezeblocks.NewCaplinSnapshots(ethconfig.BlocksFreezing{}, dirs.Snap, log.Root())
	if err := csn.ReopenFolder(); err != nil {
		return err
	}

	genesisHeader, _, _, err := csn.ReadHeader(0)
	if err != nil {
		return err
	}
	previousBlockRoot, err := genesisHeader.Header.HashSSZ()
	if err != nil {
		return err
	}
	previousBlockSlot := genesisHeader.Header.Slot
	for i := uint64(1); i < to; i++ {
		if utils.Min64(0, i-320) > previousBlockSlot {
			return fmt.Errorf("snapshot %d has invalid slot", i)
		}
		// Checking of snapshots is a chain contiguity problem
		currentHeader, _, _, err := csn.ReadHeader(i)
		if err != nil {
			return err
		}
		if currentHeader == nil {
			continue
		}
		if currentHeader.Header.ParentRoot != previousBlockRoot {
			return fmt.Errorf("snapshot %d has invalid parent root", i)
		}
		previousBlockRoot, err = currentHeader.Header.HashSSZ()
		if err != nil {
			return err
		}
		previousBlockSlot = currentHeader.Header.Slot
		if i%2000 == 0 {
			log.Info("Successfully checked", "slot", i)
		}
	}
	return nil
}

type LoopSnapshots struct {
	chainCfg
	outputFolder
	withPPROF

	Slot uint64 `name:"slot" help:"slot to check"`
}

func (c *LoopSnapshots) Run(ctx *Context) error {
	c.withProfile()

	_, _, beaconConfig, _, err := clparams.GetConfigsByNetworkName(c.Chain)
	if err != nil {
		return err
	}
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StderrHandler))
	log.Info("Started the checking process", "chain", c.Chain)

	dirs := datadir.New(c.Datadir)
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StderrHandler))

	rawDB, _ := persistence.AferoRawBeaconBlockChainFromOsPath(beaconConfig, dirs.CaplinHistory)
	beaconDB, db, err := caplin1.OpenCaplinDatabase(ctx, db_config.DatabaseConfiguration{PruneDepth: math.MaxUint64}, beaconConfig, rawDB, dirs.CaplinIndexing, nil, false)
	if err != nil {
		return err
	}
	var to uint64
	tx, err := db.BeginRo(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	to, err = beacon_indicies.ReadHighestFinalized(tx)
	if err != nil {
		return err
	}

	to = (to / snaptype.Erigon2RecentMergeLimit) * snaptype.Erigon2RecentMergeLimit

	csn := freezeblocks.NewCaplinSnapshots(ethconfig.BlocksFreezing{}, dirs.Snap, log.Root())
	if err := csn.ReopenFolder(); err != nil {
		return err
	}

	br := &snapshot_format.MockBlockReader{}
	snReader := freezeblocks.NewBeaconSnapshotReader(csn, br, beaconDB, beaconConfig)
	start := time.Now()
	for i := c.Slot; i < to; i++ {
		snReader.ReadBlockBySlot(ctx, tx, i)
	}
	log.Info("Successfully checked", "slot", c.Slot, "time", time.Since(start))
	return nil
}

type DownloadSnapshots struct {
	chainCfg
	outputFolder
}

func (d *DownloadSnapshots) Run(ctx *Context) error {
	webSeeds := snapcfg.KnownWebseeds[d.Chain]
	dirs := datadir.New(d.Datadir)

	_, _, beaconConfig, _, err := clparams.GetConfigsByNetworkName(d.Chain)
	if err != nil {
		return err
	}

	rawDB, _ := persistence.AferoRawBeaconBlockChainFromOsPath(beaconConfig, dirs.CaplinHistory)

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StderrHandler))

	_, db, err := caplin1.OpenCaplinDatabase(ctx, db_config.DatabaseConfiguration{PruneDepth: math.MaxUint64}, beaconConfig, rawDB, dirs.CaplinIndexing, nil, false)
	if err != nil {
		return err
	}
	tx, err := db.BeginRw(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	downloadRate, err := datasize.ParseString("16mb")
	if err != nil {
		return err
	}

	uploadRate, err := datasize.ParseString("0mb")
	if err != nil {
		return err
	}
	version := "erigon: " + params.VersionWithCommit(params.GitCommit)

	downloaderCfg, err := downloadercfg.New(dirs, version, lg.Info, downloadRate, uploadRate, 42069, 10, 3, nil, webSeeds, d.Chain)
	if err != nil {
		return err
	}
	downloaderCfg.DownloadTorrentFilesFromWebseed = true
	downlo, err := downloader.New(ctx, downloaderCfg, dirs, log.Root(), log.LvlInfo, true)
	if err != nil {
		return err
	}
	s, err := state2.NewAggregatorV3(ctx, dirs.Tmp, dirs.Tmp, 200000, db, log.Root())
	if err != nil {
		return err
	}
	downlo.MainLoopInBackground(false)
	bittorrentServer, err := downloader3.NewGrpcServer(downlo)
	if err != nil {
		return fmt.Errorf("new server: %w", err)
	}
	return snapshotsync.WaitForDownloader("CapCliDownloader", ctx, false, snapshotsync.OnlyCaplin, s, tx, freezeblocks.NewBlockReader(freezeblocks.NewRoSnapshots(ethconfig.NewSnapCfg(false, false, false), dirs.Snap, log.Root()), freezeblocks.NewBorRoSnapshots(ethconfig.NewSnapCfg(false, false, false), dirs.Snap, log.Root())), nil, params.ChainConfigByChainName(d.Chain), direct.NewDownloaderClient(bittorrentServer))
}

type RetrieveHistoricalState struct {
	chainCfg
	outputFolder
	CompareFile string `help:"compare file" default:""`
	CompareSlot uint64 `help:"compare slot" default:"0"`
}

func (r *RetrieveHistoricalState) Run(ctx *Context) error {
	vt := state_accessors.NewStaticValidatorTable()
	_, _, beaconConfig, t, err := clparams.GetConfigsByNetworkName(r.Chain)
	if err != nil {
		return err
	}
	dirs := datadir.New(r.Datadir)
	rawDB, fs := persistence.AferoRawBeaconBlockChainFromOsPath(beaconConfig, dirs.CaplinHistory)
	beaconDB, db, err := caplin1.OpenCaplinDatabase(ctx, db_config.DatabaseConfiguration{PruneDepth: math.MaxUint64}, beaconConfig, rawDB, dirs.CaplinIndexing, nil, false)
	if err != nil {
		return err
	}
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StderrHandler))

	tx, err := db.BeginRo(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	allSnapshots := freezeblocks.NewRoSnapshots(ethconfig.BlocksFreezing{}, dirs.Snap, log.Root())
	if err := allSnapshots.ReopenFolder(); err != nil {
		return err
	}
	if err := state_accessors.ReadValidatorsTable(tx, vt); err != nil {
		return err
	}
	fmt.Println(allSnapshots.BlocksAvailable(), allSnapshots.Dir())

	var bor *freezeblocks.BorRoSnapshots
	blockReader := freezeblocks.NewBlockReader(allSnapshots, bor)
	eth1Getter := getters.NewExecutionSnapshotReader(ctx, blockReader, db)
	csn := freezeblocks.NewCaplinSnapshots(ethconfig.BlocksFreezing{}, dirs.Snap, log.Root())
	if err := csn.ReopenFolder(); err != nil {
		return err
	}
	snr := freezeblocks.NewBeaconSnapshotReader(csn, eth1Getter, beaconDB, beaconConfig)
	gSpot, err := initial_state.GetGenesisState(t)
	if err != nil {
		return err
	}

	hr := historical_states_reader.NewHistoricalStatesReader(beaconConfig, snr, vt, fs, gSpot)
	start := time.Now()
	haveState, err := hr.ReadHistoricalState(ctx, tx, r.CompareSlot)
	if err != nil {
		return err
	}

	v := haveState.Version()
	// encode and decode the state
	enc, err := haveState.EncodeSSZ(nil)
	if err != nil {
		return err
	}
	haveState = state.New(beaconConfig)
	if err := haveState.DecodeSSZ(enc, int(v)); err != nil {
		return err
	}
	endTime := time.Since(start)
	hRoot, err := haveState.HashSSZ()
	if err != nil {
		return err
	}
	log.Info("Got state", "slot", haveState.Slot(), "root", libcommon.Hash(hRoot), "elapsed", endTime)
	if r.CompareFile == "" {
		return nil
	}
	// Read the content of CompareFile in a []byte  without afero
	rawBytes, err := os.ReadFile(r.CompareFile)
	if err != nil {
		return err
	}
	// Decode the []byte into a state
	wantState := state.New(beaconConfig)
	if err := wantState.DecodeSSZ(rawBytes, int(haveState.Version())); err != nil {
		return err
	}
	wRoot, err := wantState.HashSSZ()
	if err != nil {
		return err
	}
	if hRoot != wRoot {
		return fmt.Errorf("state mismatch: got %s, want %s", libcommon.Hash(hRoot), libcommon.Hash(wRoot))
	}
	return nil
}
