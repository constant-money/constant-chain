package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"reflect"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/wallet"
)

// PDEContribution - privacy dex contribution
type PDEContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64 // must be equal to vout value
	TokenIDStr            string
	MetadataBase
}

type PDEContributionAction struct {
	Meta    PDEContribution
	TxReqID common.Hash
	ShardID byte
}

type PDEWaitingContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64
	TokenIDStr            string
	TxReqID               common.Hash
}

type PDERefundContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64
	TokenIDStr            string
	TxReqID               common.Hash
	ShardID               byte
}

type PDEMatchedContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64
	TokenIDStr            string
	TxReqID               common.Hash
}

type PDEMatchedNReturnedContribution struct {
	PDEContributionPairID      string
	ContributorAddressStr      string
	ActualContributedAmount    uint64
	ReturnedContributedAmount  uint64
	TokenIDStr                 string
	ShardID                    byte
	TxReqID                    common.Hash
	ActualWaitingContribAmount uint64
}

type PDEContributionStatus struct {
	Status             byte
	TokenID1Str        string
	Contributed1Amount uint64
	Returned1Amount    uint64
	TokenID2Str        string
	Contributed2Amount uint64
	Returned2Amount    uint64
}

func NewPDEContribution(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
	metaType int,
) (*PDEContribution, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	pdeContribution := &PDEContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
	}
	pdeContribution.MetadataBase = metadataBase
	return pdeContribution, nil
}

func (pc PDEContribution) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
	// NOTE: verify supported tokens pair as needed
	return true, nil
}

func (pc PDEContribution) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	if tx.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(tx).String() == "*transaction.Tx" {
		return true, true, nil
	}
	if pc.PDEContributionPairID == "" {
		return false, false, errors.New("PDE contribution pair id should not be empty.")
	}

	keyWallet, err := wallet.Base58CheckDeserialize(pc.ContributorAddressStr)
	if err != nil {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("ContributorAddressStr incorrect"))
	}
	contributorAddr := keyWallet.KeySet.PaymentAddress

	if len(contributorAddr.Pk) == 0 {
		return false, false, errors.New("Wrong request info's contributed address")
	}

	isBurned, burnCoin, burnedTokenID, err := tx.GetTxBurnData()
	if err != nil || !isBurned {
		return false, false, errors.New("Error This is not Tx Burn")
	}

	if pc.ContributedAmount == 0 || pc.ContributedAmount != burnCoin.GetValue() {
		return false, false, errors.New("Contributed Amount is not valid ")
	}

	// No need to check this condition ...
	//if !bytes.Equal(tx.GetSigPubKey()[:], contributorAddr.Pk[:]) {
	//	return false, false, errors.New("ContributorAddress incorrect")
	//}

	tokenID, err := common.Hash{}.NewHashFromStr(pc.TokenIDStr)
	if err != nil {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenIDStr incorrect"))
	}
	if !bytes.Equal(burnedTokenID[:], tokenID[:]) {
		return false, false, errors.New("Wrong request info's token id, it should be equal to tx's token id.")
	}

	if tx.GetType() == common.TxNormalType && pc.TokenIDStr != common.PRVCoinID.String() {
		panic("OK")
		return false, false, errors.New("With tx normal privacy, the tokenIDStr should be PRV, not custom token.")
	}

	if tx.GetType() == common.TxCustomTokenPrivacyType && pc.TokenIDStr == common.PRVCoinID.String() {
		panic("OK")
		return false, false, errors.New("With tx custome token privacy, the tokenIDStr should not be PRV, but custom token.")
	}

	return true, true, nil
}

func (pc PDEContribution) ValidateMetadataByItself() bool {
	return pc.Type == PDEContributionMeta || pc.Type == PDEPRVRequiredContributionRequestMeta
}

func (pc PDEContribution) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.PDEContributionPairID
	record += pc.ContributorAddressStr
	record += pc.TokenIDStr
	record += strconv.FormatUint(pc.ContributedAmount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc *PDEContribution) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
	actionContent := PDEContributionAction{
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

func (pc *PDEContribution) CalculateSize() uint64 {
	return calculateSize(pc)
}
