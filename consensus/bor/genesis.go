package bor

import (
	"math/big"

	"github.com/idrecun/erigon/consensus"
	"github.com/idrecun/erigon/rlp"
)

//go:generate mockgen -destination=./genesis_contract_mock.go -package=bor . GenesisContract
type GenesisContract interface {
	CommitState(event rlp.RawValue, syscall consensus.SystemCall) error
	LastStateId(syscall consensus.SystemCall) (*big.Int, error)
}
