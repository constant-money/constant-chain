package metadata

import (
	"encoding/json"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/privacy"
)

type BlockRewardAcceptInstruction struct {
	BeaconSalary uint64
}

type BeaconRewardInfo struct {
	BeaconReward   map[common.Hash]uint64
	PayToPublicKey string
	// InfoHash       *common.Hash
}

func BuildInstForBeaconReward(reward map[common.Hash]uint64, payToPublicKey []byte) ([]string, error) {
	publicKeyString := base58.Base58Check{}.Encode(payToPublicKey, common.ZeroByte)
	beaconRewardInfo := BeaconRewardInfo{
		PayToPublicKey: publicKeyString,
		BeaconReward:   reward,
	}

	contentStr, err := json.Marshal(beaconRewardInfo)
	if err != nil {
		return nil, err
	}

	returnedInst := []string{
		strconv.Itoa(BeaconRewardRequestMeta),
		strconv.Itoa(int(common.GetShardIDFromLastByte(payToPublicKey[len(payToPublicKey)-1]))),
		"beaconRewardInst",
		string(contentStr),
	}

	return returnedInst, nil
}

func NewBeaconBlockRewardInfoFromStr(inst string) (*BeaconRewardInfo, error) {
	Ins := &BeaconRewardInfo{}
	err := json.Unmarshal([]byte(inst), Ins)
	if err != nil {
		return nil, err
	}
	return Ins, nil
}

type BeaconBlockSalaryRes struct {
	MetadataBase
	BeaconBlockHeight uint64
	ProducerAddress   *privacy.PaymentAddress
	InfoHash          *common.Hash
}

type BeaconBlockSalaryInfo struct {
	BeaconSalary      uint64
	PayToAddress      *privacy.PaymentAddress
	BeaconBlockHeight uint64
	InfoHash          *common.Hash
}

func (sbsRes BeaconBlockSalaryRes) Hash() *common.Hash {
	record := sbsRes.ProducerAddress.String()
	record += string(sbsRes.BeaconBlockHeight)
	record += sbsRes.InfoHash.String()

	// final hash
	record += sbsRes.MetadataBase.Hash().String()
	hash := common.HashH([]byte(record))
	return &hash
}
