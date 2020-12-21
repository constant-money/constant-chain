package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
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
		Logger.log.Error("UnmarshalJSON PortalCustodianDeposit", string(data))
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
			Logger.log.Error("Parse []RemoteAddress fail %+v err %+v", temp.RemoteAddresses, err)
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

func (custodianDeposit PortalCustodianDeposit) ValidateTxWithBlockChain(
	txr Transaction,
	chainRetriever ChainRetriever,
	shardViewRetriever ShardViewRetriever,
	beaconViewRetriever BeaconViewRetriever,
	shardID byte,
	db *statedb.StateDB,
) (bool, error) {
	return true, nil
}

func (custodianDeposit PortalCustodianDeposit) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, txr Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	//if txr.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(txr).String() == "*transaction.Tx" {
	//	return true, true, nil
	//}

	// validate IncogAddressStr
	keyWallet, err := wallet.Base58CheckDeserialize(custodianDeposit.IncogAddressStr)
	if err != nil {
		return false, false, errors.New("IncogAddressStr of custodian incorrect")
	}
	if len(keyWallet.KeySet.PaymentAddress.Pk) == 0 {
		return false, false, errors.New("wrong custodian incognito address")
	}

	// check burning tx
	isBurned, burnCoin, burnedTokenID, err := txr.GetTxBurnData()
	if err != nil || !isBurned {
		return false, false, errors.New("Error This is not Tx Burn")
	}

	// validate amount deposit
	if custodianDeposit.DepositedAmount == 0 || custodianDeposit.DepositedAmount != burnCoin.GetValue(){
		return false, false, errors.New("deposit amount should be larger than 0 and equal burn value")
	}

	// check tx type and token burn
	if txr.GetType() != common.TxNormalType || !bytes.Equal(burnedTokenID.Bytes(), common.PRVCoinID[:]) {
		return false, false, errors.New("tx custodian deposit must be TxNormalType")
	}

	// validate remote addresses
	if len(custodianDeposit.RemoteAddresses) == 0 {
		return false, false, errors.New("remote addresses should be at least one")
	}

	for tokenID, remoteAddr := range custodianDeposit.RemoteAddresses {
		if !common.IsPortalToken(tokenID) {
			return false, false, errors.New("TokenID in remote address is invalid")
		}
		if len(remoteAddr) == 0 {
			return false, false, errors.New("Remote address is invalid")
		}
		chainID := GetChainIDByTokenID(tokenID, chainRetriever)
		if !IsValidRemoteAddress(chainRetriever, remoteAddr, tokenID, chainID) {
			return false, false, fmt.Errorf("Remote address %v is not a valid address of tokenID %v", remoteAddr, tokenID)
		}
	}

	return true, true, nil
}

func (custodianDeposit PortalCustodianDeposit) ValidateMetadataByItself() bool {
	return custodianDeposit.Type == PortalCustodianDepositMeta
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

func (custodianDeposit *PortalCustodianDeposit) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	actionContent := PortalCustodianDepositAction{
		Meta:    *custodianDeposit,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(PortalCustodianDepositMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (custodianDeposit *PortalCustodianDeposit) CalculateSize() uint64 {
	return calculateSize(custodianDeposit)
}
