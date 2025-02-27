package heimdallgrpc

import (
	"context"

	"github.com/tenderly/erigon/consensus/bor/heimdall/span"
	"github.com/tenderly/erigon/consensus/bor/valset"

	proto "github.com/maticnetwork/polyproto/heimdall"
	protoutils "github.com/maticnetwork/polyproto/utils"
)

func (h *HeimdallGRPCClient) Span(ctx context.Context, spanID uint64) (*span.HeimdallSpan, error) {
	req := &proto.SpanRequest{
		ID: spanID,
	}

	h.logger.Info("Fetching span", "spanID", spanID)

	res, err := h.client.Span(ctx, req)
	if err != nil {
		return nil, err
	}

	h.logger.Info("Fetched span", "spanID", spanID)

	return parseSpan(res.Result), nil
}

func parseSpan(protoSpan *proto.Span) *span.HeimdallSpan {
	resp := &span.HeimdallSpan{
		Span: span.Span{
			ID:         protoSpan.ID,
			StartBlock: protoSpan.StartBlock,
			EndBlock:   protoSpan.EndBlock,
		},
		ValidatorSet:      valset.ValidatorSet{},
		SelectedProducers: []valset.Validator{},
		ChainID:           protoSpan.ChainID,
	}

	for _, validator := range protoSpan.ValidatorSet.Validators {
		resp.ValidatorSet.Validators = append(resp.ValidatorSet.Validators, parseValidator(validator))
	}

	resp.ValidatorSet.Proposer = parseValidator(protoSpan.ValidatorSet.Proposer)

	for _, validator := range protoSpan.SelectedProducers {
		resp.SelectedProducers = append(resp.SelectedProducers, *parseValidator(validator))
	}

	return resp
}

func parseValidator(validator *proto.Validator) *valset.Validator {
	return &valset.Validator{
		ID:               validator.ID,
		Address:          protoutils.ConvertH160toAddress(validator.Address),
		VotingPower:      validator.VotingPower,
		ProposerPriority: validator.ProposerPriority,
	}
}
