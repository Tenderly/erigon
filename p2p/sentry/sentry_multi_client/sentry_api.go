package sentry_multi_client

import (
	"context"
	"math/rand"

	"github.com/holiman/uint256"
	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/gointerfaces"
	proto_sentry "github.com/tenderly/erigon/erigon-lib/gointerfaces/sentry"
	"google.golang.org/grpc"

	"github.com/tenderly/erigon/eth/protocols/eth"
	"github.com/tenderly/erigon/p2p/sentry"
	"github.com/tenderly/erigon/rlp"
	"github.com/tenderly/erigon/turbo/stages/bodydownload"
	"github.com/tenderly/erigon/turbo/stages/headerdownload"
)

// Methods of sentry called by Core

func (cs *MultiClient) UpdateHead(ctx context.Context, height, time uint64, hash libcommon.Hash, td *uint256.Int) {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	cs.headHeight = height
	cs.headTime = time
	cs.headHash = hash
	cs.headTd = td
	statusMsg := cs.makeStatusData()
	for _, sentry := range cs.sentries {
		if !sentry.Ready() {
			continue
		}

		if _, err := sentry.SetStatus(ctx, statusMsg, &grpc.EmptyCallOption{}); err != nil {
			cs.logger.Error("Update status message for the sentry", "err", err)
		}
	}
}

func (cs *MultiClient) SendBodyRequest(ctx context.Context, req *bodydownload.BodyRequest) (peerID [64]byte, ok bool) {
	// if sentry not found peers to send such message, try next one. stop if found.
	for i, ok, next := cs.randSentryIndex(); ok; i, ok = next() {
		if !cs.sentries[i].Ready() {
			continue
		}

		//log.Info(fmt.Sprintf("Sending body request for %v", req.BlockNums))
		var bytes []byte
		var err error
		bytes, err = rlp.EncodeToBytes(&eth.GetBlockBodiesPacket66{
			RequestId:            rand.Uint64(), // nolint: gosec
			GetBlockBodiesPacket: req.Hashes,
		})
		if err != nil {
			cs.logger.Error("Could not encode block bodies request", "err", err)
			return [64]byte{}, false
		}
		outreq := proto_sentry.SendMessageByMinBlockRequest{
			MinBlock: req.BlockNums[len(req.BlockNums)-1],
			Data: &proto_sentry.OutboundMessageData{
				Id:   proto_sentry.MessageId_GET_BLOCK_BODIES_66,
				Data: bytes,
			},
			MaxPeers: 1,
		}

		sentPeers, err1 := cs.sentries[i].SendMessageByMinBlock(ctx, &outreq, &grpc.EmptyCallOption{})
		if err1 != nil {
			cs.logger.Error("Could not send block bodies request", "err", err1)
			return [64]byte{}, false
		}
		if sentPeers == nil || len(sentPeers.Peers) == 0 {
			continue
		}
		return sentry.ConvertH512ToPeerID(sentPeers.Peers[0]), true
	}
	return [64]byte{}, false
}

func (cs *MultiClient) SendHeaderRequest(ctx context.Context, req *headerdownload.HeaderRequest) (peerID [64]byte, ok bool) {
	// if sentry not found peers to send such message, try next one. stop if found.
	for i, ok, next := cs.randSentryIndex(); ok; i, ok = next() {
		if !cs.sentries[i].Ready() {
			continue
		}
		//log.Info(fmt.Sprintf("Sending header request {hash: %x, height: %d, length: %d}", req.Hash, req.Number, req.Length))
		reqData := &eth.GetBlockHeadersPacket66{
			RequestId: rand.Uint64(), // nolint: gosec
			GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
				Amount:  req.Length,
				Reverse: req.Reverse,
				Skip:    req.Skip,
				Origin:  eth.HashOrNumber{Hash: req.Hash},
			},
		}
		if req.Hash == (libcommon.Hash{}) {
			reqData.Origin.Number = req.Number
		}
		bytes, err := rlp.EncodeToBytes(reqData)
		if err != nil {
			cs.logger.Error("Could not encode header request", "err", err)
			return [64]byte{}, false
		}
		minBlock := req.Number

		outreq := proto_sentry.SendMessageByMinBlockRequest{
			MinBlock: minBlock,
			Data: &proto_sentry.OutboundMessageData{
				Id:   proto_sentry.MessageId_GET_BLOCK_HEADERS_66,
				Data: bytes,
			},
			MaxPeers: 5,
		}
		sentPeers, err1 := cs.sentries[i].SendMessageByMinBlock(ctx, &outreq, &grpc.EmptyCallOption{})
		if err1 != nil {
			cs.logger.Error("Could not send header request", "err", err1)
			return [64]byte{}, false
		}
		if sentPeers == nil || len(sentPeers.Peers) == 0 {
			continue
		}
		return sentry.ConvertH512ToPeerID(sentPeers.Peers[0]), true
	}
	return [64]byte{}, false
}

func (cs *MultiClient) randSentryIndex() (int, bool, func() (int, bool)) {
	var i int
	if len(cs.sentries) > 1 {
		i = rand.Intn(len(cs.sentries) - 1) // nolint: gosec
	}
	to := i
	return i, true, func() (int, bool) {
		i = (i + 1) % len(cs.sentries)
		return i, i != to
	}
}

// sending list of penalties to all sentries
func (cs *MultiClient) Penalize(ctx context.Context, penalties []headerdownload.PenaltyItem) {
	for i := range penalties {
		outreq := proto_sentry.PenalizePeerRequest{
			PeerId:  gointerfaces.ConvertHashToH512(penalties[i].PeerID),
			Penalty: proto_sentry.PenaltyKind_Kick, // TODO: Extend penalty kinds
		}
		for i, ok, next := cs.randSentryIndex(); ok; i, ok = next() {
			if !cs.sentries[i].Ready() {
				continue
			}

			if _, err1 := cs.sentries[i].PenalizePeer(ctx, &outreq, &grpc.EmptyCallOption{}); err1 != nil {
				cs.logger.Error("Could not send penalty", "err", err1)
			}
		}
	}
}
