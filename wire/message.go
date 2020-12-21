package wire

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/metadata"
	"reflect"
	"time"

	"github.com/incognitochain/incognito-chain/common"

	peer "github.com/libp2p/go-libp2p-peer"
)

// list cmd type in message header
const (
	CmdGetBlockBeacon     = "getblkbeacon"
	CmdGetBlockShard      = "getblkshard"
	CmdGetCrossShard      = "getcrossshd"
	CmdBlockShard         = "blockshard"
	CmdBlockBeacon        = "blockbeacon"
	CmdCrossShard         = "crossshard"
	CmdTx                 = "tx"
	CmdPrivacyCustomToken = "txprivacytok"
	CmdVersion            = "version"
	CmdVerack             = "verack"
	CmdGetAddr            = "getaddr"
	CmdAddr               = "addr"
	CmdPing               = "ping"

	// POS Cmd
	CmdBFT       = "bft"
	CmdPeerState = "peerstate"

	// heavy message check cmd
	CmdMsgCheck     = "msgcheck"
	CmdMsgCheckResp = "msgcheckresp"
)

// Interface for message wire on P2P network
type Message interface {
	Hash() string
	MessageType() string
	MaxPayloadLength(version int) int // update version can change length of message
	JsonSerialize() ([]byte, error)
	JsonDeserialize(string) error
	SetSenderID(peer.ID) error

	// //SignMsg sig this msg with a keyset
	// SignMsg(*incognitokey.KeySet) error

	//VerifyMsgSanity verify msg before push it to final handler
	VerifyMsgSanity() error
}

func MakeEmptyMessage(messageType string) (Message, error) {
	var msg Message
	switch messageType {
	case CmdBlockBeacon:
		msg = &MessageBlockBeacon{}
		break
	case CmdBlockShard:
		msg = &MessageBlockShard{}
		break
	case CmdGetCrossShard:
		msg = &MessageGetCrossShard{}
		break
	case CmdCrossShard:
		msg = &MessageCrossShard{}
		break
	case CmdPrivacyCustomToken:
		var tx metadata.Transaction
		msg = &MessageTxPrivacyToken{
			Transaction: tx,
		}
		break
	case CmdGetBlockBeacon:
		msg = &MessageGetBlockBeacon{
			Timestamp: time.Now().Unix(),
		}
		break
	case CmdGetBlockShard:
		msg = &MessageGetBlockShard{}
		break
	case CmdTx:
		var tx metadata.Transaction
		msg = &MessageTx{
			Transaction: tx,
		}
		break
	case CmdVersion:
		msg = &MessageVersion{}
		break
	case CmdVerack:
		msg = &MessageVerAck{}
		break
	// case CmdBFTReq:
	// 	msg = &MessageBFTReq{
	// 		Timestamp: time.Now().Unix(),
	// 	}
	// 	break
	// case CmdBFTReady:
	// 	msg = &MessageBFTReady{
	// 		Timestamp: time.Now().Unix(),
	// 	}
	// 	break
	// case CmdBFTPropose:
	// 	msg = &MessageBFTProposeV2{
	// 		Timestamp: time.Now().Unix(),
	// 	}
	// 	break
	// case CmdBFTPrepare:
	// 	msg = &MessageBFTPrepareV2{
	// 		Timestamp: time.Now().Unix(),
	// 	}
	// 	break
	// case CmdBFTAgree:
	// 	msg = &MessageBFTAgree{
	// 		Timestamp: time.Now().Unix(),
	// 	}
	// 	break
	// case CmdBFTCommit:
	// 	msg = &MessageBFTCommit{
	// 		Timestamp: time.Now().Unix(),
	// 	}
	// 	break
	case CmdPeerState:
		msg = &MessagePeerState{
			Timestamp:      time.Now().Unix(),
			Shards:         make(map[byte]ChainState),
			CrossShardPool: make(map[byte]map[byte][]uint64),
		}
		break
	case CmdGetAddr:
		msg = &MessageGetAddr{
			Timestamp: time.Now(),
		}
		break
	case CmdAddr:
		msg = &MessageAddr{
			Timestamp: time.Now(),
		}
		break
	case CmdPing:
		msg = &MessagePing{}
		break
	case CmdMsgCheck:
		msg = &MessageMsgCheck{
			Timestamp: time.Now().Unix(),
		}
		break
	case CmdMsgCheckResp:
		msg = &MessageMsgCheckResp{
			Timestamp: time.Now().Unix(),
		}
		break
	case CmdBFT:
		msg = &MessageBFT{
			Timestamp: time.Now().Unix(),
		}
	default:
		return nil, fmt.Errorf("unhandled this message type [%s]", messageType)
	}
	return msg, nil
}

func GetCmdType(msgType reflect.Type) (string, error) {
	switch msgType {
	case reflect.TypeOf(&MessageBlockBeacon{}):
		return CmdBlockBeacon, nil
	case reflect.TypeOf(&MessageBlockShard{}):
		return CmdBlockShard, nil
	case reflect.TypeOf(&MessageGetCrossShard{}):
		return CmdGetCrossShard, nil
	case reflect.TypeOf(&MessageCrossShard{}):
		return CmdCrossShard, nil
	case reflect.TypeOf(&MessageGetBlockBeacon{}):
		return CmdGetBlockBeacon, nil
	case reflect.TypeOf(&MessageGetBlockShard{}):
		return CmdGetBlockShard, nil
	case reflect.TypeOf(&MessageTx{}):
		return CmdTx, nil
	case reflect.TypeOf(&MessageTxPrivacyToken{}):
		return CmdPrivacyCustomToken, nil
	case reflect.TypeOf(&MessageVersion{}):
		return CmdVersion, nil
	case reflect.TypeOf(&MessageVerAck{}):
		return CmdVerack, nil
	case reflect.TypeOf(&MessageGetAddr{}):
		return CmdGetAddr, nil
	case reflect.TypeOf(&MessageAddr{}):
		return CmdAddr, nil
	case reflect.TypeOf(&MessagePing{}):
		return CmdPing, nil
	// case reflect.TypeOf(&MessageBFTPropose{}):
	// 	return CmdBFTPropose, nil
	// case reflect.TypeOf(&MessageBFTProposeV2{}):
	// 	return CmdBFTPropose, nil
	// case reflect.TypeOf(&MessageBFTPrepareV2{}):
	// 	return CmdBFTPrepare, nil
	// case reflect.TypeOf(&MessageBFTAgree{}):
	// 	return CmdBFTAgree, nil
	// case reflect.TypeOf(&MessageBFTCommit{}):
	// 	return CmdBFTCommit, nil
	// case reflect.TypeOf(&MessageBFTReady{}):
	// 	return CmdBFTReady, nil
	// case reflect.TypeOf(&MessageBFTReq{}):
	// 	return CmdBFTReq, nil
	case reflect.TypeOf(&MessagePeerState{}):
		return CmdPeerState, nil
	case reflect.TypeOf(&MessageMsgCheck{}):
		return CmdMsgCheck, nil
	case reflect.TypeOf(&MessageMsgCheckResp{}):
		return CmdMsgCheckResp, nil
	case reflect.TypeOf(&MessageBFT{}):
		return CmdBFT, nil
	default:
		return common.EmptyString, fmt.Errorf("unhandled this message type [%s]", msgType)
	}
}
