// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/tenderly/erigon/consensus/bor (interfaces: GenesisContract)

// Package bor is a generated GoMock package.
package bor

import (
	big "math/big"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	consensus "github.com/tenderly/erigon/consensus"
	clerk "github.com/tenderly/erigon/consensus/bor/clerk"
)

// MockGenesisContract is a mock of GenesisContract interface.
type MockGenesisContract struct {
	ctrl     *gomock.Controller
	recorder *MockGenesisContractMockRecorder
}

// MockGenesisContractMockRecorder is the mock recorder for MockGenesisContract.
type MockGenesisContractMockRecorder struct {
	mock *MockGenesisContract
}

// NewMockGenesisContract creates a new mock instance.
func NewMockGenesisContract(ctrl *gomock.Controller) *MockGenesisContract {
	mock := &MockGenesisContract{ctrl: ctrl}
	mock.recorder = &MockGenesisContractMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGenesisContract) EXPECT() *MockGenesisContractMockRecorder {
	return m.recorder
}

// CommitState mocks base method.
func (m *MockGenesisContract) CommitState(arg0 *clerk.EventRecordWithTime, arg1 consensus.SystemCall) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CommitState", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CommitState indicates an expected call of CommitState.
func (mr *MockGenesisContractMockRecorder) CommitState(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitState", reflect.TypeOf((*MockGenesisContract)(nil).CommitState), arg0, arg1)
}

// LastStateId mocks base method.
func (m *MockGenesisContract) LastStateId(arg0 consensus.SystemCall) (*big.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LastStateId", arg0)
	ret0, _ := ret[0].(*big.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LastStateId indicates an expected call of LastStateId.
func (mr *MockGenesisContractMockRecorder) LastStateId(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastStateId", reflect.TypeOf((*MockGenesisContract)(nil).LastStateId), arg0)
}
