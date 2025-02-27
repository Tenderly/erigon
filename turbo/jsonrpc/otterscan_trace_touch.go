package jsonrpc

import (
	"bytes"

	"github.com/holiman/uint256"
	"github.com/tenderly/erigon/erigon-lib/common"

	"github.com/tenderly/erigon/core/vm"
)

type TouchTracer struct {
	DefaultTracer
	searchAddr common.Address
	Found      bool
}

func NewTouchTracer(searchAddr common.Address) *TouchTracer {
	return &TouchTracer{
		searchAddr: searchAddr,
	}
}

func (t *TouchTracer) captureStartOrEnter(from, to common.Address) {
	if !t.Found && (bytes.Equal(t.searchAddr.Bytes(), from.Bytes()) || bytes.Equal(t.searchAddr.Bytes(), to.Bytes())) {
		t.Found = true
	}
}

func (t *TouchTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, precompile bool, create bool, input []byte, gas uint64, value *uint256.Int, code []byte) {
	t.captureStartOrEnter(from, to)
}

func (t *TouchTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, precompile bool, create bool, input []byte, gas uint64, value *uint256.Int, code []byte) {
	t.captureStartOrEnter(from, to)
}
