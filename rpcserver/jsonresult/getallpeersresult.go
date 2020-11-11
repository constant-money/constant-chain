package jsonresult

import (
	"fmt"

	"github.com/incognitochain/incognito-chain/addrmanager"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/connmanager"
	"github.com/incognitochain/incognito-chain/incognitokey"
)

type GetAllPeersResult struct {
	Peers []string `json:"Peers"`
}

func NewGetAllPeersResult(addrMgr addrmanager.AddrManager) *GetAllPeersResult {
	result := &GetAllPeersResult{}
	peersMap := []string{}
	peers := addrMgr.AddressCache()
	for _, peer := range peers {
		for _, peerConn := range peer.GetPeerConns() {
			peersMap = append(peersMap, peerConn.GetRemotePeer().GetRawAddress())
		}
	}
	result.Peers = peersMap
	return result
}

type GetAllConnectedPeersResult struct {
	Peers []map[string]string `json:"Peers"`
}

func NewGetAllConnectedPeersResult(connMgr connmanager.ConnManager, bc *blockchain.BlockChain) *GetAllConnectedPeersResult {
	result := &GetAllConnectedPeersResult{}
	peersMap := []map[string]string{}
	listeningPeer := connMgr.GetListeningPeer()
	bestState := bc.GetBeaconBestState()
	beaconCommitteeList, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(bestState.GetBeaconCommittee(), bestState.ConsensusAlgorithm)
	shardCommitteeList := make(map[byte][]string)
	for shardID, committee := range bestState.GetShardCommittee() {
		shardCommitteeList[shardID], _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(committee, bestState.ShardConsensusAlgorithm[shardID])
	}
	for _, peerConn := range listeningPeer.GetPeerConns() {
		pk, pkT := peerConn.GetRemotePeer().GetPublicKey()
		peerItem := map[string]string{
			"PeerID":        peerConn.GetRemotePeer().GetPeerID().Pretty(),
			"RawAddress1":   peerConn.GetRemotePeer().GetRawAddress(),
			"PublicKey":     pk,
			"PublicKeyType": pkT,
			"NodeType":      "",
		}
		isInBeaconCommittee := common.IndexOfStr(pk, beaconCommitteeList) != -1
		if isInBeaconCommittee {
			peerItem["NodeType"] = "Beacon"
		}
		for shardID, committees := range shardCommitteeList {
			isInShardCommitee := common.IndexOfStr(pk, committees) != -1
			if isInShardCommitee {
				peerItem["NodeType"] = fmt.Sprintf("Shard-%d", shardID)
				break
			}
		}
		peersMap = append(peersMap, peerItem)
	}
	result.Peers = peersMap
	return result
}
