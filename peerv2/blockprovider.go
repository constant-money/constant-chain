package peerv2

import (
	"context"

	p2pgrpc "github.com/incognitochain/go-libp2p-grpc"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/peerv2/proto"
	"github.com/incognitochain/incognito-chain/peerv2/wrapper"
	"github.com/incognitochain/incognito-chain/wire"
)

func NewBlockProvider(p *p2pgrpc.GRPCProtocol, ns NetSync) *BlockProvider {
	bp := &BlockProvider{NetSync: ns}
	proto.RegisterHighwayServiceServer(p.GetGRPCServer(), bp)
	go p.Serve() // NOTE: must serve after registering all services
	return bp
}

func (bp *BlockProvider) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	Logger.Infof("Receive new request from %v via gRPC", req.GetCommitteePublicKey())
	return nil, nil
}

func (bp *BlockProvider) GetBlockShardByHash(ctx context.Context, req *proto.GetBlockShardByHashRequest) (*proto.GetBlockShardByHashResponse, error) {
	uuid := req.GetUUID()
	hashes := []common.Hash{}
	for _, blkHashBytes := range req.Hashes {
		blkHash := common.Hash{}
		err := blkHash.SetBytes(blkHashBytes)
		if err != nil {
			continue
		}
		hashes = append(hashes, blkHash)
	}
	Logger.Infof("[blkbyhash] Receive GetBlockShardByHash shard %v request hash %v, uuid = %s", req.Shard, hashes, uuid)
	blkMsgs := bp.NetSync.GetBlockShardByHash(
		hashes,
	)
	Logger.Infof("[blkbyhash] Blockshard received from netsync: %d, uuid = %s", len(blkMsgs), uuid)
	resp := &proto.GetBlockShardByHashResponse{}
	for _, msg := range blkMsgs {
		encoded, err := encodeMessage(msg)
		if err != nil {
			Logger.Warnf("ERROR Failed encoding message %v", msg.MessageType())
			continue
		}
		resp.Data = append(resp.Data, []byte(encoded))
	}
	return resp, nil
}

func (bp *BlockProvider) GetBlockBeaconByHash(ctx context.Context, req *proto.GetBlockBeaconByHashRequest) (*proto.GetBlockBeaconByHashResponse, error) {
	uuid := req.GetUUID()
	hashes := []common.Hash{}
	for _, blkHashBytes := range req.Hashes {
		blkHash := common.Hash{}
		err := blkHash.SetBytes(blkHashBytes)
		if err != nil {
			continue
		}
		hashes = append(hashes, blkHash)
	}
	Logger.Infof("[blkbyhash] Receive GetBlockBeaconByHash request hash %v, uuid = %s", hashes, uuid)
	blkMsgs := bp.NetSync.GetBlockBeaconByHash(
		hashes,
	)
	Logger.Infof("[blkbyhash] Block beacon received from netsync: %d, uuid = %s", len(blkMsgs), uuid)
	resp := &proto.GetBlockBeaconByHashResponse{}
	for _, msg := range blkMsgs {
		encoded, err := encodeMessage(msg)
		if err != nil {
			Logger.Warnf("ERROR Failed encoding message %v", msg.MessageType())
			continue
		}
		resp.Data = append(resp.Data, []byte(encoded))
	}
	return resp, nil
}

func (bp *BlockProvider) GetBlockCrossShardByHash(ctx context.Context, req *proto.GetBlockCrossShardByHashRequest) (*proto.GetBlockCrossShardByHashResponse, error) {
	Logger.Info("Receive GetBlockCrossShardByHash request")
	return nil, nil
}

func (bp *BlockProvider) StreamBlockByHeight(
	req *proto.BlockByHeightRequest,
	stream proto.HighwayService_StreamBlockByHeightServer,
) error {
	uuid := req.GetUUID()
	cnt := 0
	Logger.Infof("[stream] Block provider received request stream block type %v, spec %v, height [%v..%v] len %v, from %v to %v, uuid = %s ", req.Type, req.Specific, req.Heights[0], req.Heights[len(req.Heights)-1], len(req.Heights), req.From, req.To, uuid)
	blkRecv := bp.NetSync.StreamBlockByHeight(false, req)
	for blk := range blkRecv {
		cnt++
		rdata, err := wrapper.EnCom(blk)
		blkData := append([]byte{byte(req.Type)}, rdata...)
		if err != nil {
			Logger.Infof("[stream] block channel return error when marshal %v, uuid = %s", err, uuid)
			Logger.Infof("[stream] Successfully sent %v blocks to client, uuid %v", cnt, uuid)
			return err
		}
		if err := stream.Send(&proto.BlockData{Data: blkData}); err != nil {
			Logger.Infof("[stream] Server send block to client return err %v, uuid = %s", err, uuid)
			Logger.Infof("[stream] Successfully sent %v blocks to client, uuid %v", cnt, uuid)
			return err
		}
	}
	Logger.Infof("[stream] Successfully sent %v blocks to client, uuid %v", cnt, uuid)
	return nil
}

func (bp *BlockProvider) StreamBlockByHash(
	req *proto.BlockByHashRequest,
	stream proto.HighwayService_StreamBlockByHashServer,
) error {
	uuid := req.GetUUID()
	Logger.Infof("[stream] Block provider received request stream block type %v, hashes [%v..%v] len %v, from %v to %v, uuid = %s ", req.Type, req.Hashes[0], req.Hashes[len(req.Hashes)-1], len(req.Hashes), req.From, req.To, uuid)
	cnt := 0
	blkRecv := bp.NetSync.StreamBlockByHash(false, req)
	for blk := range blkRecv {
		cnt++
		rdata, err := wrapper.EnCom(blk)
		blkData := append([]byte{byte(req.Type)}, rdata...)
		if err != nil {
			Logger.Infof("[stream] blkbyhash block channel return error when marshal %v, uuid = %s", err, uuid)
			Logger.Infof("[stream] Successfully sent %v blocks to client, uuid %v", cnt, uuid)
			return err
		}
		if err := stream.Send(&proto.BlockData{Data: blkData}); err != nil {
			Logger.Infof("[stream] blkbyhash Server send block to client return err %v, uuid = %s", err, uuid)
			Logger.Infof("[stream] Successfully sent %v blocks to client, uuid %v", cnt, uuid)
			return err
		}
	}
	Logger.Infof("[stream] Successfully sent %v blocks to client, uuid %v", cnt, uuid)
	return nil
}

type BlockProvider struct {
	proto.UnimplementedHighwayServiceServer
	NetSync NetSync
}

type NetSync interface {
	GetBlockShardByHash(blkHashes []common.Hash) []wire.Message
	GetBlockBeaconByHash(blkHashes []common.Hash) []wire.Message
	StreamBlockByHeight(fromPool bool, req *proto.BlockByHeightRequest) chan interface{}
	StreamBlockByHash(fromPool bool, req *proto.BlockByHashRequest) chan interface{}
}
