package solid

import (
	"encoding/json"

	ssz2 "github.com/tenderly/erigon/cl/ssz"
	"github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/types/ssz"
)

type IterableSSZ[T any] interface {
	Clear()
	CopyTo(IterableSSZ[T])
	Range(fn func(index int, value T, length int) bool)
	Get(index int) T
	Set(index int, v T)
	Length() int
	Cap() int
	Bytes() []byte
	Pop() T
	Append(v T)

	ssz2.Sized
	ssz.EncodableSSZ
	ssz.HashableSSZ
}

type Uint64ListSSZ interface {
	IterableSSZ[uint64]
	json.Marshaler
	json.Unmarshaler
}

type Uint64VectorSSZ interface {
	IterableSSZ[uint64]
	json.Marshaler
	json.Unmarshaler
}

type HashListSSZ interface {
	IterableSSZ[common.Hash]
	json.Marshaler
	json.Unmarshaler
}

type HashVectorSSZ interface {
	IterableSSZ[common.Hash]
	json.Marshaler
	json.Unmarshaler
}
