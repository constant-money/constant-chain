package statedb

import (
	"github.com/incognitochain/incognito-chain/common"
)

type StateObject interface {
	GetVersion() int
	GetValue() interface{}
	GetValueBytes() []byte
	GetHash() common.Hash
	GetType() int
	SetValue(interface{}) error
	GetTrie(DatabaseAccessWarper) Trie
	SetError(error)
	MarkDelete()
	IsDeleted() bool
	IsEmpty() bool
	Reset() bool
}

func newStateObjectWithValue(db *StateDB, objectType int, hash common.Hash, value interface{}) (StateObject, error) {
	switch objectType {
	case TestObjectType:
		return newTestObjectWithValue(db, hash, value)
	case CommitteeObjectType:
		return newCommitteeObjectWithValue(db, hash, value)
	case CommitteeRewardObjectType:
		return newCommitteeRewardObjectWithValue(db, hash, value)
	case RewardRequestObjectType:
		return newRewardRequestObjectWithValue(db, hash, value)
	case BlackListProducerObjectType:
		return newBlackListProducerObjectWithValue(db, hash, value)
	case TokenObjectType:
		return newTokenObjectWithValue(db, hash, value)
	case SerialNumberObjectType:
		return newSerialNumberObjectWithValue(db, hash, value)
	case CommitmentObjectType:
		return newCommitmentObjectWithValue(db, hash, value)
	case CommitmentIndexObjectType:
		return newCommitmentIndexObjectWithValue(db, hash, value)
	case CommitmentLengthObjectType:
		return newCommitmentLengthObjectWithValue(db, hash, value)
	case OutputCoinObjectType:
		return newOutputCoinObjectWithValue(db, hash, value)
	case OTACoinObjectType:
		return newOTACoinObjectWithValue(db, hash, value)
	case OTACoinIndexObjectType:
		return newOTACoinIndexObjectWithValue(db, hash, value)
	case OTACoinLengthObjectType:
		return newOTACoinLengthObjectWithValue(db, hash, value)
	case OnetimeAddressObjectType:
		return newOnetimeAddressObjectWithValue(db, hash, value)
	case SNDerivatorObjectType:
		return newSNDerivatorObjectWithValue(db, hash, value)
	case WaitingPDEContributionObjectType:
		return newWaitingPDEContributionObjectWithValue(db, hash, value)
	case PDEPoolPairObjectType:
		return newPDEPoolPairObjectWithValue(db, hash, value)
	case PDEShareObjectType:
		return newPDEShareObjectWithValue(db, hash, value)
	case PDETradingFeeObjectType:
		return newPDETradingFeeObjectWithValue(db, hash, value)
	case PDEStatusObjectType:
		return newPDEStatusObjectWithValue(db, hash, value)
	case BridgeEthTxObjectType:
		return newBridgeEthTxObjectWithValue(db, hash, value)
	case BridgeTokenInfoObjectType:
		return newBridgeTokenInfoObjectWithValue(db, hash, value)
	case BridgeStatusObjectType:
		return newBridgeStatusObjectWithValue(db, hash, value)
	case BurningConfirmObjectType:
		return newBurningConfirmObjectWithValue(db, hash, value)
	case TokenTransactionObjectType:
		return newTokenTransactionObjectWithValue(db, hash, value)
	case PortalFinalExchangeRatesStateObjectType:
		return newFinalExchangeRatesStateObjectWithValue(db, hash, value)
	case PortalLiquidationPoolObjectType:
		return newLiquidationPoolObjectWithValue(db, hash, value)
	case PortalWaitingPortingRequestObjectType:
		return newWaitingPortingRequestObjectWithValue(db, hash, value)
	case PortalStatusObjectType:
		return newPortalStatusObjectWithValue(db, hash, value)
	case PortalRewardInfoObjectType:
		return newPortalRewardInfoObjectWithValue(db, hash, value)
	case WaitingRedeemRequestObjectType:
		return newRedeemRequestObjectWithValue(db, hash, value)
	case CustodianStateObjectType:
		return newCustodianStateObjectWithValue(db, hash, value)
	case LockedCollateralStateObjectType:
		return newLockedCollateralStateObjectWithValue(db, hash, value)
	case RewardFeatureStateObjectType:
		return newRewardFeatureStateObjectWithValue(db, hash, value)
	case StakerObjectType:
		return newStakerObjectWithValue(db, hash, value)
	default:
		panic("state object type not exist")
	}
}

func newStateObject(db *StateDB, objectType int, hash common.Hash) StateObject {
	switch objectType {
	case TestObjectType:
		return newTestObject(db, hash)
	case CommitteeObjectType:
		return newCommitteeObject(db, hash)
	case CommitteeRewardObjectType:
		return newCommitteeRewardObject(db, hash)
	case RewardRequestObjectType:
		return newRewardRequestObject(db, hash)
	case BlackListProducerObjectType:
		return newBlackListProducerObject(db, hash)
	case SerialNumberObjectType:
		return newSerialNumberObject(db, hash)
	case CommitmentObjectType:
		return newCommitteeObject(db, hash)
	case CommitmentIndexObjectType:
		return newCommitmentIndexObject(db, hash)
	case CommitmentLengthObjectType:
		return newCommitmentLengthObject(db, hash)
	case SNDerivatorObjectType:
		return newSNDerivatorObject(db, hash)
	case OTACoinObjectType:
		return newOTACoinObject(db, hash)
	case OTACoinIndexObjectType:
		return newOTACoinIndexObject(db, hash)
	case OTACoinLengthObjectType:
		return newOTACoinLengthObject(db, hash)
	case OnetimeAddressObjectType:
		return newOnetimeAddressObject(db, hash)
	case WaitingPDEContributionObjectType:
		return newWaitingPDEContributionObject(db, hash)
	case PDEPoolPairObjectType:
		return newPDEPoolPairObject(db, hash)
	case PDEShareObjectType:
		return newPDEShareObject(db, hash)
	case PDETradingFeeObjectType:
		return newPDETradingFeeObject(db, hash)
	case PDEStatusObjectType:
		return newPDEStatusObject(db, hash)
	case BridgeEthTxObjectType:
		return newBridgeEthTxObject(db, hash)
	case BridgeTokenInfoObjectType:
		return newBridgeTokenInfoObject(db, hash)
	case BridgeStatusObjectType:
		return newBridgeStatusObject(db, hash)
	case BurningConfirmObjectType:
		return newBurningConfirmObject(db, hash)
	case PortalFinalExchangeRatesStateObjectType:
		return newFinalExchangeRatesStateObject(db, hash)
	case PortalLiquidationPoolObjectType:
		return newLiquidationPoolObject(db, hash)
	case PortalWaitingPortingRequestObjectType:
		return newWaitingPortingRequestObject(db, hash)
	case PortalStatusObjectType:
		return newPortalStatusObject(db, hash)
	case PortalRewardInfoObjectType:
		return newPortalRewardInfoObject(db, hash)
	case WaitingRedeemRequestObjectType:
		return newRedeemRequestObject(db, hash)
	case CustodianStateObjectType:
		return newCustodianStateObject(db, hash)
	case LockedCollateralStateObjectType:
		return newLockedCollateralStateObject(db, hash)
	case RewardFeatureStateObjectType:
		return newRewardFeatureStateObject(db, hash)
	case StakerObjectType:
		return newStakerObject(db, hash)
	default:
		panic("state object type not exist")
	}
}
