package portaltokens

import (
	"encoding/base64"
	"encoding/json"
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
)

type PortalTokenProcessor interface {
	IsValidRemoteAddress(address string, bcr bMeta.ChainRetriever) (bool, error)
	GetChainID() string
	GetMinTokenAmount() uint64

	GetExpectedMemoForPorting(portingID string) string
	GetExpectedMemoForRedeem(redeemID string, custodianIncAddress string) string
	ParseAndVerifyProof(
		proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedPaymentInfos map[string]uint64) (bool, error)

	CreateRawExternalTx() error
}

// set MinTokenAmount to avoid attacking with amount is less than smallest unit of cryptocurrency
// such as satoshi in BTC
type PortalToken struct {
	ChainID        string
	MinTokenAmount uint64 // minimum amount for porting/redeem
}

func (p PortalToken) GetExpectedMemoForPorting(portingID string) string {
	type portingMemoStruct struct {
		PortingID string `json:"PortingID"`
	}
	memoPorting := portingMemoStruct{PortingID: portingID}
	memoPortingBytes, _ := json.Marshal(memoPorting)
	memoPortingHashBytes := common.HashB(memoPortingBytes)
	memoPortingStr := base64.StdEncoding.EncodeToString(memoPortingHashBytes)
	return memoPortingStr
}

func (p PortalToken) GetExpectedMemoForRedeem(redeemID string, custodianAddress string) string {
	type redeemMemoStruct struct {
		RedeemID                  string `json:"RedeemID"`
		CustodianIncognitoAddress string `json:"CustodianIncognitoAddress"`
	}

	redeemMemo := redeemMemoStruct{
		RedeemID:                  redeemID,
		CustodianIncognitoAddress: custodianAddress,
	}
	redeemMemoBytes, _ := json.Marshal(redeemMemo)
	redeemMemoHashBytes := common.HashB(redeemMemoBytes)
	redeemMemoStr := base64.StdEncoding.EncodeToString(redeemMemoHashBytes)
	return redeemMemoStr
}
