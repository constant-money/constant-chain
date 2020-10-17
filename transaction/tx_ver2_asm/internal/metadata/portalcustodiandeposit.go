package metadata

import (
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"sort"
	"strconv"
)

// PortalCustodianDeposit - portal custodian deposit collateral (PRV)
// metadata - custodian deposit - create normal tx with this metadata
type PortalCustodianDeposit struct {
	MetadataBase
	IncogAddressStr string
	RemoteAddresses map[string]string // tokenID: remote address
	DepositedAmount uint64
}

func (object *PortalCustodianDeposit) UnmarshalJSON(data []byte) error {
	type Alias PortalCustodianDeposit
	temp := &struct {
		RemoteAddresses interface{}
		*Alias
	}{
		Alias: (*Alias)(object),
	}

	err := json.Unmarshal(data, &temp)
	if err != nil {
		return errors.New("can not parse data for PortalCustodianDeposit")
	}

	remoteAddreses := make(map[string]string)
	tempJson, _ := json.MarshalIndent(temp.RemoteAddresses, "  ", "  ")
	err2 := json.Unmarshal(tempJson, &remoteAddreses)
	if err2 != nil {
		// int testnet, exception:
		type RemoteAddress struct {
			PTokenID string
			Address  string
		}

		tmpRemoteAddress := make([]RemoteAddress, 0)
		tempJson, _ := json.MarshalIndent(temp.RemoteAddresses, "  ", "  ")
		err1 := json.Unmarshal(tempJson, &tmpRemoteAddress)
		if err1 != nil {
			return errors.New("can not parse data for PortalCustodianDeposit RemoteAddress")
		} else {
			remoteAddreses = make(map[string]string)
			for _, v := range tmpRemoteAddress {
				remoteAddreses[v.PTokenID] = v.Address
			}
		}
	}
	object.RemoteAddresses = remoteAddreses
	return nil

}

// PortalCustodianDepositAction - shard validator creates instruction that contain this action content
// it will be append to ShardToBeaconBlock
type PortalCustodianDepositAction struct {
	Meta    PortalCustodianDeposit
	TxReqID common.Hash
	ShardID byte
}

// PortalCustodianDepositContent - Beacon builds a new instruction with this content after receiving a instruction from shard
// It will be appended to beaconBlock
// both accepted and refund status
type PortalCustodianDepositContent struct {
	IncogAddressStr string
	RemoteAddresses map[string]string // tokenID: remote address
	DepositedAmount uint64
	TxReqID         common.Hash
	ShardID         byte
}

// PortalCustodianDepositStatus - Beacon tracks status of custodian deposit tx into db
type PortalCustodianDepositStatus struct {
	Status          byte
	IncogAddressStr string
	RemoteAddresses map[string]string // tokenID: remote address
	DepositedAmount uint64
}

func NewPortalCustodianDeposit(metaType int, incognitoAddrStr string, remoteAddrs map[string]string, amount uint64) (*PortalCustodianDeposit, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	custodianDepositMeta := &PortalCustodianDeposit{
		IncogAddressStr: incognitoAddrStr,
		RemoteAddresses: remoteAddrs,
		DepositedAmount: amount,
	}
	custodianDepositMeta.MetadataBase = metadataBase
	return custodianDepositMeta, nil
}

func (custodianDeposit PortalCustodianDeposit) Hash() *common.Hash {
	record := custodianDeposit.MetadataBase.Hash().String()
	record += custodianDeposit.IncogAddressStr
	tokenIDKeys := make([]string, 0)
	for tokenID := range custodianDeposit.RemoteAddresses {
		tokenIDKeys = append(tokenIDKeys, tokenID)
	}
	sort.Strings(tokenIDKeys)
	for _, tokenID := range tokenIDKeys {
		record += custodianDeposit.RemoteAddresses[tokenID]
	}
	record += strconv.FormatUint(custodianDeposit.DepositedAmount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}
