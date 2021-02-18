package portaltokens

import (
	"encoding/base64"
	"encoding/json"
	bMeta "github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type PortalTokenProcessor interface {
	IsValidRemoteAddress(address string, bcr bMeta.ChainRetriever) (bool, error)
	GetChainID() string
	GetMinTokenAmount() uint64
	ConvertIncToExternalAmount(incAmt uint64) uint64

	GetExpectedMemoForShielding(incAddress string) string
	GetExpectedMemoForRedeem(redeemID string, custodianIncAddress string) string
	ParseAndVerifyProof(
		proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string) (bool, []*statedb.UTXO, error)
	ParseAndVerifyUnshieldProof(
		proof string, bc bMeta.ChainRetriever, expectedMemo string, expectedMultisigAddress string, expectPaymentInfo map[string]uint64) (bool, []*statedb.UTXO, error)
	GetExternalTxHashFromProof(proof string) (string, error)
	ChooseUnshieldIDsFromCandidates(utxos map[string]*statedb.UTXO, waitingUnshieldReqs map[string]*statedb.WaitingUnshieldRequest) []*BroadcastTx

	CreateRawExternalTx(inputs []*statedb.UTXO, outputs []*OutputTx, networkFee uint64, memo string, bc bMeta.ChainRetriever) (string, string, error)
	GeneratePrivateKeyFromSeed(seed []byte) ([]byte, error)
	GenerateMultiSigWalletFromSeeds(bc bMeta.ChainRetriever, seeds [][]byte, numSigsRequired int) ([]byte, []string, string, error)
}

// set MinTokenAmount to avoid attacking with amount is less than smallest unit of cryptocurrency
// such as satoshi in BTC
type PortalToken struct {
	ChainID        string
	MinTokenAmount uint64 // minimum amount for shielding/redeem
}

type BroadcastTx struct {
	UTXOs       []*statedb.UTXO
	UnshieldIDs []string
}

type OutputTx struct {
	ReceiverAddress string
	Amount          uint64
}

func (p PortalToken) GetExpectedMemoForShielding(incAddress string) string {
	type shieldingMemoStruct struct {
		IncAddress string `json:"ShieldingIncAddress"`
	}
	memoShielding := shieldingMemoStruct{IncAddress: incAddress}
	memoShieldingBytes, _ := json.Marshal(memoShielding)
	memoShieldingHashBytes := common.HashB(memoShieldingBytes)
	memoShieldingStr := base64.StdEncoding.EncodeToString(memoShieldingHashBytes)
	return memoShieldingStr
}

//todo:
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

func (p PortalToken) IsAcceptableTxSize(num_utxos int, num_unshield_id int) bool {
	// TODO: do experiments depend on external chain miner's habit
	A := 1
	B := 1
	C := 10
	return A*num_utxos+B*num_unshield_id <= C
}


