package rawdbhelpers

import (
	"encoding/binary"

	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/eth/ethconfig"
)

func IdxStepsCountV3(tx kv.Tx) float64 {
	fst, _ := kv.FirstKey(tx, kv.TblTracesToKeys)
	lst, _ := kv.LastKey(tx, kv.TblTracesToKeys)
	if len(fst) > 0 && len(lst) > 0 {
		fstTxNum := binary.BigEndian.Uint64(fst)
		lstTxNum := binary.BigEndian.Uint64(lst)

		return float64(lstTxNum-fstTxNum) / float64(ethconfig.HistoryV3AggregationStep)
	}
	return 0
}
