package spectest

import (
	"os"
	"testing"

	"github.com/tenderly/erigon/spectest"

	"github.com/tenderly/erigon/cl/transition"

	"github.com/tenderly/erigon/cl/spectest/consensus_tests"
)

func Test(t *testing.T) {
	spectest.RunCases(t, consensus_tests.TestFormats, transition.ValidatingMachine, os.DirFS("./tests"))
}
