package graph

import (
	"github.com/idrecun/erigon/erigon-lib/kv"
	"github.com/idrecun/erigon/turbo/jsonrpc"
	"github.com/idrecun/erigon/turbo/rpchelper"
	"github.com/idrecun/erigon/turbo/services"
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
