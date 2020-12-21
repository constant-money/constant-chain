package wire

import (
	"encoding/json"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	peer "github.com/libp2p/go-libp2p-peer"
)

type MessageGetCrossShard struct {
	FromPool         bool
	ByHash           bool
	BySpecificHeight bool
	BlkHashes        []common.Hash
	BlkHeights       []uint64
	FromShardID      byte
	ToShardID        byte
	SenderID         string
	Timestamp        int64
}

func (msg *MessageGetCrossShard) Hash() string {
	rawBytes, err := msg.JsonSerialize()
	if err != nil {
		return ""
	}
	return common.HashH(rawBytes).String()
}

func (msg *MessageGetCrossShard) MessageType() string {
	return CmdGetCrossShard
}

func (msg *MessageGetCrossShard) MaxPayloadLength(pver int) int {
	return MaxBlockPayload
}

func (msg *MessageGetCrossShard) JsonSerialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(msg)
	return jsonBytes, err
}

func (msg *MessageGetCrossShard) JsonDeserialize(jsonStr string) error {
	err := json.Unmarshal([]byte(jsonStr), msg)
	return err
}

func (msg *MessageGetCrossShard) SetSenderID(senderID peer.ID) error {
	msg.SenderID = senderID.Pretty()
	return nil
}

func (msg *MessageGetCrossShard) SignMsg(_ *incognitokey.KeySet) error {
	return nil
}

func (msg *MessageGetCrossShard) VerifyMsgSanity() error {
	return nil
}
