package debug

import (
	"os"
	"syscall"

	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/erigon-lib/common/dbg"
)

var sigc chan os.Signal

func GetSigC(sig *chan os.Signal) {
	sigc = *sig
}

// LogPanic - does log panic to logger and to <datadir>/crashreports then stops the process
func LogPanic() {
	panicResult := recover()
	if panicResult == nil {
		return
	}

	log.Error("catch panic", "err", panicResult, "stack", dbg.Stack())
	if sigc != nil {
		sigc <- syscall.SIGINT
	}
}
