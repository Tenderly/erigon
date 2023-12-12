package commands

import (
	"github.com/spf13/cobra"
	"github.com/tenderly/erigon/cmd/state/verify"
	"github.com/tenderly/erigon/turbo/debug"
)

func init() {
	withDataDir(verifyTxLookupCmd)
	rootCmd.AddCommand(verifyTxLookupCmd)
}

var verifyTxLookupCmd = &cobra.Command{
	Use:   "verifyTxLookup",
	Short: "Generate tx lookup index",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := debug.SetupCobra(cmd, "verify_txlookup")
		return verify.ValidateTxLookups(chaindata, logger)
	},
}
