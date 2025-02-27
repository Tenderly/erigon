package heimdallgrpc

import (
	"context"
	"math/big"

	"github.com/tenderly/erigon/consensus/bor/heimdall/checkpoint"

	proto "github.com/maticnetwork/polyproto/heimdall"
	protoutils "github.com/maticnetwork/polyproto/utils"
)

func (h *HeimdallGRPCClient) FetchCheckpointCount(ctx context.Context) (int64, error) {
	h.logger.Info("Fetching checkpoint count")

	res, err := h.client.FetchCheckpointCount(ctx, nil)
	if err != nil {
		return 0, err
	}

	h.logger.Info("Fetched checkpoint count")

	return res.Result.Result, nil
}

func (h *HeimdallGRPCClient) FetchCheckpoint(ctx context.Context, number int64) (*checkpoint.Checkpoint, error) {
	req := &proto.FetchCheckpointRequest{
		ID: number,
	}

	h.logger.Info("Fetching checkpoint", "number", number)

	res, err := h.client.FetchCheckpoint(ctx, req)
	if err != nil {
		return nil, err
	}

	h.logger.Info("Fetched checkpoint", "number", number)

	checkpoint := &checkpoint.Checkpoint{
		StartBlock: new(big.Int).SetUint64(res.Result.StartBlock),
		EndBlock:   new(big.Int).SetUint64(res.Result.EndBlock),
		RootHash:   protoutils.ConvertH256ToHash(res.Result.RootHash),
		Proposer:   protoutils.ConvertH160toAddress(res.Result.Proposer),
		BorChainID: res.Result.BorChainID,
		Timestamp:  uint64(res.Result.Timestamp.GetSeconds()),
	}

	return checkpoint, nil
}
