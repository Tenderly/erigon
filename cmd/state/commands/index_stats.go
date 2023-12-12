package commands

import (
	"github.com/spf13/cobra"
	"github.com/tenderly/erigon/cmd/state/stats"
)

func init() {
	withDataDir(indexStatsCmd)
	withStatsfile(indexStatsCmd)
	withIndexBucket(indexStatsCmd)
	rootCmd.AddCommand(indexStatsCmd)
}

var indexStatsCmd = &cobra.Command{
	Use:   "indexStats",
	Short: "Stats about index chunks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statsfile == "stateless.csv" {
			statsfile = ""
		}
		return stats.IndexStats(chaindata, indexBucket, statsfile)
	},
}
