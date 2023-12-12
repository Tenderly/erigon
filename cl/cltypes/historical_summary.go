package cltypes

import (
	libcommon "github.com/tenderly/erigon/erigon-lib/common"

	"github.com/tenderly/erigon/cl/merkle_tree"
	ssz2 "github.com/tenderly/erigon/cl/ssz"
	"github.com/tenderly/erigon/erigon-lib/common/length"
)

type HistoricalSummary struct {
	BlockSummaryRoot libcommon.Hash `json:"block_summary_root"`
	StateSummaryRoot libcommon.Hash `json:"state_summary_root"`
}

func (h *HistoricalSummary) EncodeSSZ(buf []byte) ([]byte, error) {
	return ssz2.MarshalSSZ(buf, h.BlockSummaryRoot[:], h.StateSummaryRoot[:])
}

func (h *HistoricalSummary) DecodeSSZ(buf []byte, _ int) error {
	return ssz2.UnmarshalSSZ(buf, 0, h.BlockSummaryRoot[:], h.StateSummaryRoot[:])
}

func (h *HistoricalSummary) HashSSZ() ([32]byte, error) {
	return merkle_tree.HashTreeRoot(h.BlockSummaryRoot[:], h.StateSummaryRoot[:])
}

func (*HistoricalSummary) EncodingSizeSSZ() int {
	return length.Hash * 2
}
