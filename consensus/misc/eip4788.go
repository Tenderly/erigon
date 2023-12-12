package misc

import (
	"github.com/ledgerwatch/log/v3"

	"github.com/idrecun/erigon/consensus"
	libcommon "github.com/idrecun/erigon/erigon-lib/common"
	"github.com/idrecun/erigon/params"
)

func ApplyBeaconRootEip4788(parentBeaconBlockRoot *libcommon.Hash, syscall consensus.SystemCall) {
	_, err := syscall(params.BeaconRootsAddress, parentBeaconBlockRoot.Bytes())
	if err != nil {
		log.Warn("Failed to call beacon roots contract", "err", err)
	}
}
