package rpchelper

import (
	"context"
	"sync/atomic"

	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/gointerfaces/remote"

	"github.com/tenderly/erigon/erigon-lib/kv"

	"github.com/tenderly/erigon/core/types"
	"github.com/tenderly/erigon/p2p"
)

// ApiBackend - interface which must be used by API layer
// implementation can work with local Ethereum object or with Remote (grpc-based) one
// this is reason why all methods are accepting context and returning error
type ApiBackend interface {
	Etherbase(ctx context.Context) (libcommon.Address, error)
	NetVersion(ctx context.Context) (uint64, error)
	NetPeerCount(ctx context.Context) (uint64, error)
	ProtocolVersion(ctx context.Context) (uint64, error)
	ClientVersion(ctx context.Context) (string, error)
	Subscribe(ctx context.Context, cb func(*remote.SubscribeReply)) error
	SubscribeLogs(ctx context.Context, cb func(*remote.SubscribeLogsReply), requestor *atomic.Value) error
	BlockWithSenders(ctx context.Context, tx kv.Getter, hash libcommon.Hash, blockHeight uint64) (block *types.Block, senders []libcommon.Address, err error)
	NodeInfo(ctx context.Context, limit uint32) ([]p2p.NodeInfo, error)
	Peers(ctx context.Context) ([]*p2p.PeerInfo, error)
	AddPeer(ctx context.Context, url *remote.AddPeerRequest) (*remote.AddPeerReply, error)
	PendingBlock(ctx context.Context) (*types.Block, error)
}
