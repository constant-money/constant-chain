package metadata

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/wallet"
)

// PDEFeeWithdrawalRequest - privacy dex withdrawal request
type PDEFeeWithdrawalRequest struct {
	WithdrawerAddressStr  string
	WithdrawalToken1IDStr string
	WithdrawalToken2IDStr string
	WithdrawalFeeAmt      uint64
	MetadataBase
}

type PDEFeeWithdrawalRequestAction struct {
	Meta    PDEFeeWithdrawalRequest
	TxReqID common.Hash
	ShardID byte
}

func NewPDEFeeWithdrawalRequest(
	withdrawerAddressStr string,
	withdrawalToken1IDStr string,
	withdrawalToken2IDStr string,
	withdrawalFeeAmt uint64,
	metaType int,
) (*PDEFeeWithdrawalRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType, Sig: []byte{},
	}
	pdeFeeWithdrawalRequest := &PDEFeeWithdrawalRequest{
		WithdrawerAddressStr:  withdrawerAddressStr,
		WithdrawalToken1IDStr: withdrawalToken1IDStr,
		WithdrawalToken2IDStr: withdrawalToken2IDStr,
		WithdrawalFeeAmt:      withdrawalFeeAmt,
	}
	pdeFeeWithdrawalRequest.MetadataBase = metadataBase
	return pdeFeeWithdrawalRequest, nil
}

func (*PDEFeeWithdrawalRequest) ShouldSignMetaData() bool { return true }

func (pc PDEFeeWithdrawalRequest) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	// NOTE: verify supported tokens pair as needed
	return true, nil
}

func (pc PDEFeeWithdrawalRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	keyWallet, err := wallet.Base58CheckDeserialize(pc.WithdrawerAddressStr)
	if err != nil {
		return false, false, NewMetadataTxError(PDEFeeWithdrawalRequestFromMapError, errors.New("WithdrawerAddressStr incorrect"))
	}
	withdrawerAddr := keyWallet.KeySet.PaymentAddress
	if len(withdrawerAddr.Pk) == 0 {
		return false, false, errors.New("Wrong request info's withdrawer address")
	}

	if ok, err := tx.CheckAuthorizedSender(withdrawerAddr.Pk); err != nil || !ok {
		fmt.Println("Check authorized sender fail:", ok, err)
		return false, false, errors.New("WithdrawerAddr unauthorized")
	}

	_, err = common.Hash{}.NewHashFromStr(pc.WithdrawalToken1IDStr)
	if err != nil {
		return false, false, NewMetadataTxError(PDEFeeWithdrawalRequestFromMapError, errors.New("WithdrawalTokenID1Str incorrect"))
	}
	_, err = common.Hash{}.NewHashFromStr(pc.WithdrawalToken2IDStr)
	if err != nil {
		return false, false, NewMetadataTxError(PDEFeeWithdrawalRequestFromMapError, errors.New("WithdrawalTokenID2Str incorrect"))
	}
	if pc.WithdrawalFeeAmt == 0 {
		return false, false, NewMetadataTxError(PDEFeeWithdrawalRequestFromMapError, errors.New("WithdrawalFeeAmt should be large than 0"))
	}
	return true, true, nil
}

func (pc PDEFeeWithdrawalRequest) ValidateMetadataByItself() bool {
	return pc.Type == PDEFeeWithdrawalRequestMeta
}

func (pc PDEFeeWithdrawalRequest) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.WithdrawerAddressStr
	record += pc.WithdrawalToken1IDStr
	record += pc.WithdrawalToken2IDStr
	record += strconv.FormatUint(pc.WithdrawalFeeAmt, 10)
	if pc.Sig != nil && len(pc.Sig) != 0 {
		record += string(pc.Sig)
	}
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc PDEFeeWithdrawalRequest) HashWithoutSig() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.WithdrawerAddressStr
	record += pc.WithdrawalToken1IDStr
	record += pc.WithdrawalToken2IDStr
	record += strconv.FormatUint(pc.WithdrawalFeeAmt, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc *PDEFeeWithdrawalRequest) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	actionContent := PDEFeeWithdrawalRequestAction{
		Meta:    *pc,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(pc.Type), actionContentBase64Str}
	return [][]string{action}, nil
}

func (pc *PDEFeeWithdrawalRequest) CalculateSize() uint64 {
	return calculateSize(pc)
}
