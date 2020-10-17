package metadata

import (
	"encoding/json"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/wallet"
)

type IncDAORewardInfo struct {
	IncDAOReward map[common.Hash]uint64
}

func BuildInstForIncDAOReward(reward map[common.Hash]uint64, incDAOAddress string) ([]string, error) {

	devRewardInfo := IncDAORewardInfo{
		IncDAOReward: reward,
	}

	contentStr, err := json.Marshal(devRewardInfo)
	if err != nil {
		return nil, err
	}

	keyWalletDevAccount, err := wallet.Base58CheckDeserialize(incDAOAddress)
	if err != nil {
		return nil, err
	}
	returnedInst := []string{
		strconv.Itoa(IncDAORewardRequestMeta),
		strconv.Itoa(int(common.GetShardIDFromLastByte(keyWalletDevAccount.KeySet.PaymentAddress.Pk[len(keyWalletDevAccount.KeySet.PaymentAddress.Pk)-1]))),
		"devRewardInst",
		string(contentStr),
	}

	return returnedInst, nil
}

func NewIncDAORewardInfoFromStr(inst string) (*IncDAORewardInfo, error) {
	Ins := &IncDAORewardInfo{}
	err := json.Unmarshal([]byte(inst), Ins)
	if err != nil {
		return nil, err
	}
	return Ins, nil
}
