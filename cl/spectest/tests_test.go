package spectest

import (
	"os"
	"testing"

	"github.com/idrecun/erigon/spectest"

	"github.com/idrecun/erigon/cl/transition"

	"github.com/idrecun/erigon/cl/spectest/consensus_tests"
)

func Test(t *testing.T) {
	spectest.RunCases(t, consensus_tests.TestFormats, transition.ValidatingMachine, os.DirFS("./tests"))
}
