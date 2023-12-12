package graph

import (
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/turbo/jsonrpc"
	"github.com/tenderly/erigon/turbo/rpchelper"
	"github.com/tenderly/erigon/turbo/services"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	GraphQLAPI  jsonrpc.GraphQLAPI
	db          kv.RoDB
	filters     *rpchelper.Filters
	blockReader services.FullBlockReader
}
