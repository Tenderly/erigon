package rpchelper_test

import (
	"context"
	"testing"

	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/gointerfaces"
	"github.com/tenderly/erigon/erigon-lib/gointerfaces/remote"
	types2 "github.com/tenderly/erigon/erigon-lib/gointerfaces/types"

	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/core/types"
	"github.com/tenderly/erigon/eth/filters"
	"github.com/tenderly/erigon/turbo/rpchelper"
)

func TestFiltersDeadlock_Test(t *testing.T) {
	t.Parallel()
	logger := log.New()
	f := rpchelper.New(context.TODO(), nil, nil, nil, func() {}, logger)
	crit := filters.FilterCriteria{
		Addresses: nil,
		Topics:    [][]libcommon.Hash{},
	}
	subCount := 20
	logCount := 100
	type sub struct {
		id rpchelper.LogsSubID
		ch <-chan *types.Log
	}
	ctx, cancel := context.WithCancel(context.TODO())
	for i := 0; i < subCount; i++ {
		n := &sub{}
		n.ch, n.id = f.SubscribeLogs(128, crit)
		// start a loop similar to an rpcdaemon subscription, that calls unsubscribe on return
		go func() {
			defer f.UnsubscribeLogs(n.id)
			for {
				select {
				case l := <-n.ch:
					_ = l
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	// cancel the subs at the same time
	cancel()
	// try to send logs
	for i := 0; i < logCount; i++ {
		log := createLog()
		// this will deadlock
		f.OnNewLogs(log)
	}
}

func createLog() *remote.SubscribeLogsReply {
	return &remote.SubscribeLogsReply{
		Address:          gointerfaces.ConvertAddressToH160([20]byte{}),
		BlockHash:        gointerfaces.ConvertHashToH256([32]byte{}),
		BlockNumber:      0,
		Data:             []byte{},
		LogIndex:         0,
		Topics:           []*types2.H256{gointerfaces.ConvertHashToH256([32]byte{99, 99})},
		TransactionHash:  gointerfaces.ConvertHashToH256([32]byte{}),
		TransactionIndex: 0,
		Removed:          false,
	}
}
