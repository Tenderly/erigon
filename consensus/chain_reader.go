package consensus

import (
	"context"
	"math/big"

	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/erigon-lib/chain"
	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/turbo/services"

	"github.com/tenderly/erigon/core/rawdb"
	"github.com/tenderly/erigon/core/types"
)

// Implements consensus.ChainReader
type ChainReaderImpl struct {
	Cfg         chain.Config
	Db          kv.Getter
	BlockReader services.FullBlockReader
}

// Config retrieves the blockchain's chain configuration.
func (cr ChainReaderImpl) Config() *chain.Config {
	return &cr.Cfg
}

// CurrentHeader retrieves the current header from the local chain.
func (cr ChainReaderImpl) CurrentHeader() *types.Header {
	hash := rawdb.ReadHeadHeaderHash(cr.Db)
	number := rawdb.ReadHeaderNumber(cr.Db, hash)
	h, _ := cr.BlockReader.Header(context.Background(), cr.Db, hash, *number)
	return h
}

// GetHeader retrieves a block header from the database by hash and number.
func (cr ChainReaderImpl) GetHeader(hash libcommon.Hash, number uint64) *types.Header {
	h, _ := cr.BlockReader.Header(context.Background(), cr.Db, hash, number)
	return h
}

// GetHeaderByNumber retrieves a block header from the database by number.
func (cr ChainReaderImpl) GetHeaderByNumber(number uint64) *types.Header {
	h, _ := cr.BlockReader.HeaderByNumber(context.Background(), cr.Db, number)
	return h
}

// GetHeaderByHash retrieves a block header from the database by its hash.
func (cr ChainReaderImpl) GetHeaderByHash(hash libcommon.Hash) *types.Header {
	number := rawdb.ReadHeaderNumber(cr.Db, hash)
	h, _ := cr.BlockReader.Header(context.Background(), cr.Db, hash, *number)
	return h
}

// GetBlock retrieves a block from the database by hash and number.
func (cr ChainReaderImpl) GetBlock(hash libcommon.Hash, number uint64) *types.Block {
	b, _, _ := cr.BlockReader.BlockWithSenders(context.Background(), cr.Db, hash, number)
	return b
}

// HasBlock retrieves a block from the database by hash and number.
func (cr ChainReaderImpl) HasBlock(hash libcommon.Hash, number uint64) bool {
	b, _ := cr.BlockReader.BodyRlp(context.Background(), cr.Db, hash, number)
	return b != nil
}

// GetTd retrieves the total difficulty from the database by hash and number.
func (cr ChainReaderImpl) GetTd(hash libcommon.Hash, number uint64) *big.Int {
	td, err := rawdb.ReadTd(cr.Db, hash, number)
	if err != nil {
		log.Error("ReadTd failed", "err", err)
		return nil
	}
	return td
}

func (cr ChainReaderImpl) FrozenBlocks() uint64 {
	return cr.BlockReader.FrozenBlocks()
}

func (cr ChainReaderImpl) BorSpan(spanId uint64) []byte {
	spanBytes, err := cr.BlockReader.Span(context.Background(), cr.Db, spanId)
	if err != nil {
		log.Error("BorSpan failed", "err", err)
	}
	return spanBytes
}
