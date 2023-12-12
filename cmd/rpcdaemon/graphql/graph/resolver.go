package graph

import (
	"github.com/idrecun/erigon/turbo/jsonrpc"
	"github.com/idrecun/erigon/turbo/rpchelper"
	"github.com/idrecun/erigon/turbo/services"
	"github.com/ledgerwatch/erigon-lib/kv"
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
