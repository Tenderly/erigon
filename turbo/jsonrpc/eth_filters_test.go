package jsonrpc

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/common/length"

	"github.com/tenderly/erigon/rpc/rpccfg"

	"github.com/stretchr/testify/assert"
	"github.com/tenderly/erigon/erigon-lib/gointerfaces/txpool"
	"github.com/tenderly/erigon/erigon-lib/kv/kvcache"

	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/cmd/rpcdaemon/rpcdaemontest"
	"github.com/tenderly/erigon/eth/filters"
	"github.com/tenderly/erigon/turbo/rpchelper"
	"github.com/tenderly/erigon/turbo/stages/mock"
)

func TestNewFilters(t *testing.T) {
	assert := assert.New(t)
	m, _, _ := rpcdaemontest.CreateTestSentry(t)
	agg := m.HistoryV3Components()
	stateCache := kvcache.New(kvcache.DefaultCoherentConfig)
	ctx, conn := rpcdaemontest.CreateTestGrpcConn(t, mock.Mock(t))
	mining := txpool.NewMiningClient(conn)
	ff := rpchelper.New(ctx, nil, nil, mining, func() {}, m.Log)
	api := NewEthAPI(NewBaseApi(ff, stateCache, m.BlockReader, agg, false, rpccfg.DefaultEvmCallTimeout, m.Engine, m.Dirs), m.DB, nil, nil, nil, 5000000, 100_000, false, 100_000, log.New())

	ptf, err := api.NewPendingTransactionFilter(ctx)
	assert.Nil(err)

	nf, err := api.NewFilter(ctx, filters.FilterCriteria{})
	assert.Nil(err)

	bf, err := api.NewBlockFilter(ctx)
	assert.Nil(err)

	ok, err := api.UninstallFilter(ctx, nf)
	assert.Nil(err)
	assert.Equal(ok, true)

	ok, err = api.UninstallFilter(ctx, bf)
	assert.Nil(err)
	assert.Equal(ok, true)

	ok, err = api.UninstallFilter(ctx, ptf)
	assert.Nil(err)
	assert.Equal(ok, true)
}

func TestLogsSubscribeAndUnsubscribe_WithoutConcurrentMapIssue(t *testing.T) {
	m := mock.Mock(t)
	ctx, conn := rpcdaemontest.CreateTestGrpcConn(t, m)
	mining := txpool.NewMiningClient(conn)
	ff := rpchelper.New(ctx, nil, nil, mining, func() {}, m.Log)

	// generate some random topics
	topics := make([][]libcommon.Hash, 0)
	for i := 0; i < 10; i++ {
		bytes := make([]byte, length.Hash)
		rand.Read(bytes)
		toAdd := []libcommon.Hash{libcommon.BytesToHash(bytes)}
		topics = append(topics, toAdd)
	}

	// generate some addresses
	addresses := make([]libcommon.Address, 0)
	for i := 0; i < 10; i++ {
		bytes := make([]byte, length.Addr)
		rand.Read(bytes)
		addresses = append(addresses, libcommon.BytesToAddress(bytes))
	}

	crit := filters.FilterCriteria{
		Topics:    topics,
		Addresses: addresses,
	}

	ids := make([]rpchelper.LogsSubID, 1000)

	// make a lot of subscriptions
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(idx int) {
			_, id := ff.SubscribeLogs(32, crit)
			defer func() {
				time.Sleep(100 * time.Nanosecond)
				ff.UnsubscribeLogs(id)
				wg.Done()
			}()
			ids[idx] = id
		}(i)
	}
	wg.Wait()
}
