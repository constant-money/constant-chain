package blockchain

import (
	"encoding/json"

	"github.com/incognitochain/incognito-chain/basemeta"
	"github.com/incognitochain/incognito-chain/common"
	pMeta "github.com/incognitochain/incognito-chain/portalv4/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wallet"
)

// buildPortalAcceptedShieldingRequestTx builds response tx for the shielding request tx with status "accepted"
// mints pToken to return to user
func (curView *ShardBestState) buildPortalAcceptedShieldingRequestTx(
	beaconState *BeaconBestState,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
) (basemeta.Transaction, error) {
	Logger.log.Errorf("[Shard buildPortalAcceptedShieldingRequestTx] Starting...")
	contentBytes := []byte(contentStr)
	var acceptedShieldingReq pMeta.PortalShieldingRequestContent
	err := json.Unmarshal(contentBytes, &acceptedShieldingReq)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling portal custodian deposit content: %+v", err)
		return nil, nil
	}
	if acceptedShieldingReq.ShardID != shardID {
		Logger.log.Errorf("ERROR: ShardID unexpected expect %v, but got %+v", shardID, acceptedShieldingReq.ShardID)
		return nil, nil
	}

	shieldingAmount := uint64(0)
	for _, utxo := range acceptedShieldingReq.ShieldingUTXO {
		shieldingAmount += utxo.GetOutputAmount()
	}

	meta := pMeta.NewPortalShieldingResponse(
		"accepted",
		acceptedShieldingReq.TxReqID,
		acceptedShieldingReq.IncogAddressStr,
		shieldingAmount,
		acceptedShieldingReq.TokenID,
		basemeta.PortalShieldingRequestMeta,
	)

	keyWallet, err := wallet.Base58CheckDeserialize(acceptedShieldingReq.IncogAddressStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while deserializing custodian address string: %+v", err)
		return nil, nil
	}
	receiverAddr := keyWallet.KeySet.PaymentAddress
	receiveAmt := shieldingAmount
	tokenID, _ := new(common.Hash).NewHashFromStr(acceptedShieldingReq.TokenID)

	// in case the returned currency is privacy custom token
	receiver := &privacy.PaymentInfo{
		Amount:         receiveAmt,
		PaymentAddress: receiverAddr,
	}
	var propertyID [common.HashSize]byte
	copy(propertyID[:], tokenID[:])
	propID := common.Hash(propertyID)
	tokenParams := &transaction.CustomTokenPrivacyParamTx{
		PropertyID: propID.String(),
		// PropertyName:   issuingAcceptedInst.IncTokenName,
		// PropertySymbol: issuingAcceptedInst.IncTokenName,
		Amount:      receiveAmt,
		TokenTxType: transaction.CustomTokenInit,
		Receiver:    []*privacy.PaymentInfo{receiver},
		TokenInput:  []*privacy.InputCoin{},
		Mintable:    true,
	}
	resTx := &transaction.TxCustomTokenPrivacy{}
	txStateDB := curView.GetCopiedTransactionStateDB()
	featureStateDB := beaconState.GetBeaconFeatureStateDB()
	initErr := resTx.Init(
		transaction.NewTxPrivacyTokenInitParams(
			producerPrivateKey,
			[]*privacy.PaymentInfo{},
			nil,
			0,
			tokenParams,
			txStateDB,
			meta,
			false,
			false,
			shardID,
			nil,
			featureStateDB,
		),
	)
	if initErr != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing request ptoken response tx: %+v", initErr)
		return nil, nil
	}
	return resTx, nil
}
