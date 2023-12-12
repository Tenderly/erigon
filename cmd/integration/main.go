package main

import (
	"fmt"
	"os"

	"github.com/tenderly/erigon/cmd/integration/commands"
	"github.com/tenderly/erigon/erigon-lib/common"
)

func main() {
	rootCmd := commands.RootCommand()
	ctx, _ := common.RootContext()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
