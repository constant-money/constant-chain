package wire

import (
	"encoding/json"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	peer "github.com/libp2p/go-libp2p-peer"
)

const (
	MaxBFTPayload = 1 << 23 // 8 MB
)

type MessageBFT struct {
	PeerID    string
	Type      string
	Content   []byte
	ChainKey  string
	Timestamp int64
	TimeSlot  int64
}

func (msg *MessageBFT) Hash() string {
	rawBytes, err := msg.JsonSerialize()
	if err != nil {
		return ""
	}
	return common.HashH(rawBytes).String()
}

func (msg *MessageBFT) MessageType() string {
	return CmdBFT
}

func (msg *MessageBFT) MaxPayloadLength(pver int) int {
	return MaxBFTPayload
}

func (msg *MessageBFT) JsonSerialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(msg)
	return jsonBytes, err
}

func (msg *MessageBFT) JsonDeserialize(jsonStr string) error {
	err := json.Unmarshal([]byte(jsonStr), msg)
	return err
}

func (msg *MessageBFT) SetSenderID(senderID peer.ID) error {
	return nil
}

func (msg *MessageBFT) SignMsg(keySet *incognitokey.KeySet) error {
	return nil
}

func (msg *MessageBFT) VerifyMsgSanity() error {
	return nil
}
