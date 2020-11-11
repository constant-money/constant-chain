package wire

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/blockchain/types"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/libp2p/go-libp2p-peer"
)

// const (
// 	MaxBlockPayload = 1000000 // 1 Mb
// )

type MessageCrossShard struct {
	Block *types.CrossShardBlock
}

func (msg *MessageCrossShard) Hash() string {
	rawBytes, err := msg.JsonSerialize()
	if err != nil {
		return ""
	}
	return common.HashH(rawBytes).String()
}

func (msg *MessageCrossShard) MessageType() string {
	return CmdCrossShard
}

func (msg *MessageCrossShard) MaxPayloadLength(pver int) int {
	return MaxBlockPayload
}

func (msg *MessageCrossShard) JsonSerialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(msg)
	return jsonBytes, err
}

func (msg *MessageCrossShard) JsonDeserialize(jsonStr string) error {
	err := json.Unmarshal([]byte(jsonStr), msg)
	return err
}

func (msg *MessageCrossShard) SetSenderID(senderID peer.ID) error {
	return nil
}

func (msg *MessageCrossShard) SignMsg(_ *incognitokey.KeySet) error {
	return nil
}

func (msg *MessageCrossShard) VerifyMsgSanity() error {
	return nil
}
