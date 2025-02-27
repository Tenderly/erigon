package persistence

import (
	"context"
	"io"

	"github.com/tenderly/erigon/cl/cltypes"
	"github.com/tenderly/erigon/cl/sentinel/peers"
	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/kv"
)

type BlockSource interface {
	GetRange(ctx context.Context, tx kv.Tx, from uint64, count uint64) (*peers.PeeredObject[[]*cltypes.SignedBeaconBlock], error)
	PurgeRange(ctx context.Context, tx kv.Tx, from uint64, count uint64) error
	GetBlock(ctx context.Context, tx kv.Tx, slot uint64) (*peers.PeeredObject[*cltypes.SignedBeaconBlock], error)
}

type BeaconChainWriter interface {
	WriteBlock(ctx context.Context, tx kv.RwTx, block *cltypes.SignedBeaconBlock, canonical bool) error
}

type RawBeaconBlockChain interface {
	BlockWriter(ctx context.Context, slot uint64, blockRoot libcommon.Hash) (io.WriteCloser, error)
	BlockReader(ctx context.Context, slot uint64, blockRoot libcommon.Hash) (io.ReadCloser, error)
	DeleteBlock(ctx context.Context, slot uint64, blockRoot libcommon.Hash) error
}

type BeaconChainDatabase interface {
	BlockSource
	BeaconChainWriter
}
