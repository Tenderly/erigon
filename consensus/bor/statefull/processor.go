package statefull

import (
	"github.com/holiman/uint256"
	ethereum "github.com/tenderly/erigon"
	"github.com/tenderly/erigon/consensus"
	"github.com/tenderly/erigon/core"
	"github.com/tenderly/erigon/core/types"
	"github.com/tenderly/erigon/core/vm"
	libcommon "github.com/tenderly/erigon/erigon-lib/common"
)

type ChainContext struct {
	Chain consensus.ChainReader
	Bor   consensus.Engine
}

func (c ChainContext) Engine() consensus.Engine {
	return c.Bor
}

func (c ChainContext) GetHeader(hash libcommon.Hash, number uint64) *types.Header {
	return c.Chain.GetHeader(hash, number)
}

// callmsg implements core.Message to allow passing it as a transaction simulator.
type Callmsg struct {
	ethereum.CallMsg
}

func (m Callmsg) From() libcommon.Address { return m.CallMsg.From }
func (m Callmsg) Nonce() uint64           { return 0 }
func (m Callmsg) CheckNonce() bool        { return false }
func (m Callmsg) To() *libcommon.Address  { return m.CallMsg.To }
func (m Callmsg) GasPrice() *uint256.Int  { return m.CallMsg.GasPrice }
func (m Callmsg) Gas() uint64             { return m.CallMsg.Gas }
func (m Callmsg) Value() *uint256.Int     { return m.CallMsg.Value }
func (m Callmsg) Data() []byte            { return m.CallMsg.Data }

func ApplyBorMessage(vmenv vm.EVM, msg Callmsg) (*core.ExecutionResult, error) {
	initialGas := msg.Gas()

	// Apply the transaction to the current state (included in the env)
	ret, gasLeft, err := vmenv.Call(
		vm.AccountRef(msg.From()),
		*msg.To(),
		msg.Data(),
		msg.Gas(),
		msg.Value(),
		false,
	)

	gasUsed := initialGas - gasLeft

	return &core.ExecutionResult{
		UsedGas:    gasUsed,
		Err:        err,
		ReturnData: ret,
	}, nil
}
