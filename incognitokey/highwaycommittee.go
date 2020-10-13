package incognitokey

// import (
// 	"encoding/json"
// )

// type ChainCommittee struct {
// 	Epoch             uint64
// 	BeaconCommittee   []CommitteePublicKey
// 	AllShardCommittee map[byte][]CommitteePublicKey
// 	AllShardPending   map[byte][]CommitteePublicKey
// }

// func (cc *ChainCommittee) ToByte() ([]byte, error) {
// 	data, err := json.Marshal(cc)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return data, nil
// }

// func ChainCommitteeFromByte(data []byte) (*ChainCommittee, error) {
// 	cc := &ChainCommittee{}
// 	err := json.Unmarshal(data, cc)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return cc, nil
// }
