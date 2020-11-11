package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wallet"
)

func parseTradeRefundContent(
	contentStr string,
) (*metadata.PDETradeRequestAction, error) {
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde trade refund instruction: %+v", err)
		return nil, err
	}
	var pdeTradeRequestAction metadata.PDETradeRequestAction
	err = json.Unmarshal(contentBytes, &pdeTradeRequestAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde trade refund content: %+v", err)
		return nil, err
	}
	return &pdeTradeRequestAction, nil
}

func parseCrossPoolTradeRefundContent(
	contentStr string,
) (*metadata.PDERefundCrossPoolTrade, error) {
	contentBytes := []byte(contentStr)
	var pdeRefundCrossPoolTrade metadata.PDERefundCrossPoolTrade
	err := json.Unmarshal(contentBytes, &pdeRefundCrossPoolTrade)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde cross pool trade refund content: %+v", err)
		return nil, err
	}
	return &pdeRefundCrossPoolTrade, nil
}

func parseTradeAcceptedContent(
	contentStr string,
) (*metadata.PDETradeAcceptedContent, error) {
	contentBytes := []byte(contentStr)
	var pdeTradeAcceptedContent metadata.PDETradeAcceptedContent
	err := json.Unmarshal(contentBytes, &pdeTradeAcceptedContent)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde trade accepted content: %+v", err)
		return nil, err
	}
	return &pdeTradeAcceptedContent, nil
}

func parseCrossPoolTradeAcceptedContent(
	contentStr string,
) ([]metadata.PDECrossPoolTradeAcceptedContent, error) {
	contentBytes := []byte(contentStr)
	var pdeCrossPoolTradeAcceptedContent []metadata.PDECrossPoolTradeAcceptedContent
	err := json.Unmarshal(contentBytes, &pdeCrossPoolTradeAcceptedContent)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde cross pool trade accepted content: %+v", err)
		return nil, err
	}
	return pdeCrossPoolTradeAcceptedContent, nil
}

func buildTradeResTx(
	receiverAddressStr string,
	receiveAmt uint64,
	tokenIDStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	transactionStateDB *statedb.StateDB,
	stateDB *statedb.StateDB,
	meta metadata.Metadata,
) (metadata.Transaction, error) {
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while converting tokenid to hash: %+v", err)
		return nil, err
	}
	keyWallet, err := wallet.Base58CheckDeserialize(receiverAddressStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while deserializing trader address string: %+v", err)
		return nil, err
	}
	receiverAddr := keyWallet.KeySet.PaymentAddress
	// the returned currency is PRV
	if tokenIDStr == common.PRVCoinID.String() {
		resTx := new(transaction.Tx)
		err = resTx.InitTxSalary(
			receiveAmt,
			&receiverAddr,
			producerPrivateKey,
			transactionStateDB,
			meta,
		)
		if err != nil {
			return nil, NewBlockChainError(InitPDETradeResponseTransactionError, err)
		}
		return resTx, nil
	}
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
	initErr := resTx.Init(
		transaction.NewTxPrivacyTokenInitParams(
			producerPrivateKey,
			[]*privacy.PaymentInfo{},
			nil,
			0,
			tokenParams,
			transactionStateDB,
			meta,
			false,
			false,
			shardID,
			nil,
			stateDB,
		),
	)
	if initErr != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing trade response tx: %+v", initErr)
		return nil, initErr
	}
	return resTx, nil
}

func (blockGenerator *BlockGenerator) buildPDECrossPoolTradeRefundTx(
	instStatus string,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	crossPoolTradeRefundContent, err := parseCrossPoolTradeRefundContent(contentStr)
	if err != nil {
		return nil, nil
	}
	if shardID != crossPoolTradeRefundContent.ShardID {
		return nil, nil
	}
	meta := metadata.NewPDECrossPoolTradeResponse(
		instStatus,
		crossPoolTradeRefundContent.TxReqID,
		metadata.PDECrossPoolTradeResponseMeta,
	)
	resTx, err := buildTradeResTx(
		crossPoolTradeRefundContent.TraderAddressStr,
		crossPoolTradeRefundContent.Amount,
		crossPoolTradeRefundContent.TokenIDStr,
		producerPrivateKey,
		shardID,
		shardView.GetCopiedTransactionStateDB(),
		featureStateDB,
		meta,
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing refunded trading response tx: %+v", err)
		return nil, nil
	}
	Logger.log.Info("[PDE Cross Pool Trade] Create refunded tx ok.")
	return resTx, nil
}

func (blockGenerator *BlockGenerator) buildPDETradeRefundTx(
	instStatus string,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	pdeTradeRequestAction, err := parseTradeRefundContent(contentStr)
	if err != nil {
		return nil, nil
	}
	if shardID != pdeTradeRequestAction.ShardID {
		return nil, nil
	}
	meta := metadata.NewPDETradeResponse(
		instStatus,
		pdeTradeRequestAction.TxReqID,
		metadata.PDETradeResponseMeta,
	)
	resTx, err := buildTradeResTx(
		pdeTradeRequestAction.Meta.TraderAddressStr,
		pdeTradeRequestAction.Meta.SellAmount+pdeTradeRequestAction.Meta.TradingFee,
		pdeTradeRequestAction.Meta.TokenIDToSellStr,
		producerPrivateKey,
		shardID,
		shardView.GetCopiedTransactionStateDB(),
		featureStateDB,
		meta,
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing refunded trading response tx: %+v", err)
		return nil, nil
	}
	Logger.log.Info("[PDE Trade] Create refunded tx ok.")
	return resTx, nil
}

func (blockGenerator *BlockGenerator) buildPDECrossPoolTradeAcceptedTx(
	instStatus string,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	crossPoolTradeAcceptedContents, err := parseCrossPoolTradeAcceptedContent(contentStr)
	if err != nil {
		return nil, nil
	}
	len := len(crossPoolTradeAcceptedContents)
	if len == 0 {
		Logger.log.Warn("WARNING: cross pool trade contents is empty.")
		return nil, nil
	}
	finalCrossPoolTradeAcceptedContent := crossPoolTradeAcceptedContents[len-1]
	if shardID != finalCrossPoolTradeAcceptedContent.ShardID {
		return nil, nil
	}
	meta := metadata.NewPDECrossPoolTradeResponse(
		instStatus,
		finalCrossPoolTradeAcceptedContent.RequestedTxID,
		metadata.PDECrossPoolTradeResponseMeta,
	)
	resTx, err := buildTradeResTx(
		finalCrossPoolTradeAcceptedContent.TraderAddressStr,
		finalCrossPoolTradeAcceptedContent.ReceiveAmount,
		finalCrossPoolTradeAcceptedContent.TokenIDToBuyStr,
		producerPrivateKey,
		shardID,
		shardView.GetCopiedTransactionStateDB(),
		featureStateDB,
		meta,
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing accepted cross pool trading response tx: %+v", err)
		return nil, nil
	}
	Logger.log.Info("[PDE Cross Pool Trade] Create accepted tx ok.")
	return resTx, nil
}

func (blockGenerator *BlockGenerator) buildPDETradeAcceptedTx(
	instStatus string,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	pdeTradeAcceptedContent, err := parseTradeAcceptedContent(contentStr)
	if err != nil {
		return nil, nil
	}
	if shardID != pdeTradeAcceptedContent.ShardID {
		return nil, nil
	}
	meta := metadata.NewPDETradeResponse(
		instStatus,
		pdeTradeAcceptedContent.RequestedTxID,
		metadata.PDETradeResponseMeta,
	)
	resTx, err := buildTradeResTx(
		pdeTradeAcceptedContent.TraderAddressStr,
		pdeTradeAcceptedContent.ReceiveAmount,
		pdeTradeAcceptedContent.TokenIDToBuyStr,
		producerPrivateKey,
		shardID,
		shardView.GetCopiedTransactionStateDB(),
		featureStateDB,
		meta,
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing accepted trading response tx: %+v", err)
		return nil, nil
	}
	Logger.log.Info("[PDE Trade] Create accepted tx ok.")
	return resTx, nil
}

func (blockGenerator *BlockGenerator) buildPDETradeIssuanceTx(
	instStatus string,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	Logger.log.Info("[PDE Trade] Starting...")
	if instStatus == common.PDETradeRefundChainStatus {
		return blockGenerator.buildPDETradeRefundTx(
			instStatus,
			contentStr,
			producerPrivateKey,
			shardID,
			shardView,
			featureStateDB,
		)
	}
	return blockGenerator.buildPDETradeAcceptedTx(
		instStatus,
		contentStr,
		producerPrivateKey,
		shardID,
		shardView,
		featureStateDB,
	)
}

func (blockGenerator *BlockGenerator) buildPDECrossPoolTradeIssuanceTx(
	instStatus string,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	Logger.log.Info("[PDE Cross Pool Trade] Starting...")
	if instStatus == common.PDECrossPoolTradeFeeRefundChainStatus || instStatus == common.PDECrossPoolTradeSellingTokenRefundChainStatus {
		return blockGenerator.buildPDECrossPoolTradeRefundTx(
			instStatus,
			contentStr,
			producerPrivateKey,
			shardID,
			shardView,
			featureStateDB,
		)
	}
	return blockGenerator.buildPDECrossPoolTradeAcceptedTx(
		instStatus,
		contentStr,
		producerPrivateKey,
		shardID,
		shardView,
		featureStateDB,
	)
}

func (blockGenerator *BlockGenerator) buildPDEWithdrawalTx(
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	Logger.log.Info("[PDE Withdrawal] Starting...")
	contentBytes := []byte(contentStr)
	var wdAcceptedContent metadata.PDEWithdrawalAcceptedContent
	err := json.Unmarshal(contentBytes, &wdAcceptedContent)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde withdrawal content: %+v", err)
		return nil, nil
	}
	if wdAcceptedContent.ShardID != shardID {
		return nil, nil
	}

	withdrawalTokenIDStr := wdAcceptedContent.WithdrawalTokenIDStr
	meta := metadata.NewPDEWithdrawalResponse(
		withdrawalTokenIDStr,
		wdAcceptedContent.TxReqID,
		metadata.PDEWithdrawalResponseMeta,
	)
	tokenID, err := common.Hash{}.NewHashFromStr(withdrawalTokenIDStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while converting tokenid to hash: %+v", err)
		return nil, nil
	}
	keyWallet, err := wallet.Base58CheckDeserialize(wdAcceptedContent.WithdrawerAddressStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while deserializing trader address string: %+v", err)
		return nil, nil
	}
	receiverAddr := keyWallet.KeySet.PaymentAddress
	// the returned currency is PRV
	if withdrawalTokenIDStr == common.PRVCoinID.String() {
		resTx := new(transaction.Tx)
		err = resTx.InitTxSalary(
			wdAcceptedContent.DeductingPoolValue,
			&receiverAddr,
			producerPrivateKey,
			shardView.GetCopiedTransactionStateDB(),
			meta,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while initializing withdrawal (normal) tx: %+v", err)
			return nil, nil
		}
		//modify the type of the salary transaction
		// resTx.Type = common.TxBlockProducerCreatedType
		return resTx, nil
	}
	// in case the returned currency is privacy custom token
	receiver := &privacy.PaymentInfo{
		Amount:         wdAcceptedContent.DeductingPoolValue,
		PaymentAddress: receiverAddr,
	}
	var propertyID [common.HashSize]byte
	copy(propertyID[:], tokenID[:])
	propID := common.Hash(propertyID)
	tokenParams := &transaction.CustomTokenPrivacyParamTx{
		PropertyID: propID.String(),
		// PropertyName:   tokeName,
		// PropertySymbol: tokenSymbol,
		Amount:      wdAcceptedContent.DeductingPoolValue,
		TokenTxType: transaction.CustomTokenInit,
		Receiver:    []*privacy.PaymentInfo{receiver},
		TokenInput:  []*privacy.InputCoin{},
		Mintable:    true,
	}
	resTx := &transaction.TxCustomTokenPrivacy{}
	initErr := resTx.Init(
		transaction.NewTxPrivacyTokenInitParams(
			producerPrivateKey,
			[]*privacy.PaymentInfo{},
			nil,
			0,
			tokenParams,
			shardView.GetCopiedTransactionStateDB(),
			meta,
			false,
			false,
			shardID,
			nil,
			featureStateDB,
		),
	)
	if initErr != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing withdrawal response (privacy custom token) tx: %+v", initErr)
		return nil, nil
	}
	return resTx, nil
}

func (blockGenerator *BlockGenerator) buildPDERefundContributionTx(
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	Logger.log.Info("[PDE Refund contribution] Starting...")
	contentBytes := []byte(contentStr)
	var refundContribution metadata.PDERefundContribution
	err := json.Unmarshal(contentBytes, &refundContribution)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde refund contribution content: %+v", err)
		return nil, nil
	}
	if refundContribution.ShardID != shardID {
		return nil, nil
	}

	meta := metadata.NewPDEContributionResponse(
		"refund",
		refundContribution.TxReqID,
		refundContribution.TokenIDStr,
		metadata.PDEContributionResponseMeta,
	)
	refundTokenIDStr := refundContribution.TokenIDStr
	tokenID, err := common.Hash{}.NewHashFromStr(refundTokenIDStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while converting tokenid to hash: %+v", err)
		return nil, nil
	}
	keyWallet, err := wallet.Base58CheckDeserialize(refundContribution.ContributorAddressStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while deserializing contributor address string: %+v", err)
		return nil, nil
	}
	receiverAddr := keyWallet.KeySet.PaymentAddress
	// the returned currency is PRV
	if refundTokenIDStr == common.PRVCoinID.String() {
		resTx := new(transaction.Tx)
		err = resTx.InitTxSalary(
			refundContribution.ContributedAmount,
			&receiverAddr,
			producerPrivateKey,
			shardView.GetCopiedTransactionStateDB(),
			meta,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while initializing refund contribution (normal) tx: %+v", err)
			return nil, nil
		}
		//modify the type of the salary transaction
		// resTx.Type = common.TxBlockProducerCreatedType
		return resTx, nil
	}

	// in case the returned currency is privacy custom token
	receiver := &privacy.PaymentInfo{
		Amount:         refundContribution.ContributedAmount,
		PaymentAddress: receiverAddr,
	}
	var propertyID [common.HashSize]byte
	copy(propertyID[:], tokenID[:])
	propID := common.Hash(propertyID)
	tokenParams := &transaction.CustomTokenPrivacyParamTx{
		PropertyID: propID.String(),
		// PropertyName:   issuingAcceptedInst.IncTokenName,
		// PropertySymbol: issuingAcceptedInst.IncTokenName,
		Amount:      refundContribution.ContributedAmount,
		TokenTxType: transaction.CustomTokenInit,
		Receiver:    []*privacy.PaymentInfo{receiver},
		TokenInput:  []*privacy.InputCoin{},
		Mintable:    true,
	}
	resTx := &transaction.TxCustomTokenPrivacy{}
	initErr := resTx.Init(
		transaction.NewTxPrivacyTokenInitParams(
			producerPrivateKey,
			[]*privacy.PaymentInfo{},
			nil,
			0,
			tokenParams,
			shardView.GetCopiedTransactionStateDB(),
			meta,
			false,
			false,
			shardID,
			nil,
			featureStateDB,
		),
	)
	if initErr != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing refund contribution response (privacy custom token) tx: %+v", initErr)
		return nil, nil
	}
	return resTx, nil
}

func (blockGenerator *BlockGenerator) buildPDEMatchedNReturnedContributionTx(
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	Logger.log.Info("[PDE Matched and Returned contribution] Starting...")
	contentBytes := []byte(contentStr)
	var matchedNReturnedContribution metadata.PDEMatchedNReturnedContribution
	err := json.Unmarshal(contentBytes, &matchedNReturnedContribution)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde matched and  returned contribution content: %+v", err)
		return nil, nil
	}
	if matchedNReturnedContribution.ShardID != shardID {
		return nil, nil
	}
	if matchedNReturnedContribution.ReturnedContributedAmount == 0 {
		return nil, nil
	}

	meta := metadata.NewPDEContributionResponse(
		common.PDEContributionMatchedNReturnedChainStatus,
		matchedNReturnedContribution.TxReqID,
		matchedNReturnedContribution.TokenIDStr,
		metadata.PDEContributionResponseMeta,
	)
	tokenIDStr := matchedNReturnedContribution.TokenIDStr
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while converting tokenid to hash: %+v", err)
		return nil, nil
	}
	keyWallet, err := wallet.Base58CheckDeserialize(matchedNReturnedContribution.ContributorAddressStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while deserializing contributor address string: %+v", err)
		return nil, nil
	}
	receiverAddr := keyWallet.KeySet.PaymentAddress
	// the returned currency is PRV
	if tokenIDStr == common.PRVCoinID.String() {
		resTx := new(transaction.Tx)
		err = resTx.InitTxSalary(
			matchedNReturnedContribution.ReturnedContributedAmount,
			&receiverAddr,
			producerPrivateKey,
			shardView.GetCopiedTransactionStateDB(),
			meta,
		)
		if err != nil {
			Logger.log.Errorf("ERROR: an error occured while initializing refund contribution (normal) tx: %+v", err)
			return nil, nil
		}
		return resTx, nil
	}

	// in case the returned currency is privacy custom token
	receiver := &privacy.PaymentInfo{
		Amount:         matchedNReturnedContribution.ReturnedContributedAmount,
		PaymentAddress: receiverAddr,
	}
	var propertyID [common.HashSize]byte
	copy(propertyID[:], tokenID[:])
	propID := common.Hash(propertyID)
	tokenParams := &transaction.CustomTokenPrivacyParamTx{
		PropertyID: propID.String(),
		// PropertyName:   issuingAcceptedInst.IncTokenName,
		// PropertySymbol: issuingAcceptedInst.IncTokenName,
		Amount:      matchedNReturnedContribution.ReturnedContributedAmount,
		TokenTxType: transaction.CustomTokenInit,
		Receiver:    []*privacy.PaymentInfo{receiver},
		TokenInput:  []*privacy.InputCoin{},
		Mintable:    true,
	}
	resTx := &transaction.TxCustomTokenPrivacy{}
	initErr := resTx.Init(
		transaction.NewTxPrivacyTokenInitParams(
			producerPrivateKey,
			[]*privacy.PaymentInfo{},
			nil,
			0,
			tokenParams,
			shardView.GetCopiedTransactionStateDB(),
			meta,
			false,
			false,
			shardID,
			nil,
			featureStateDB,
		),
	)
	if initErr != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing matched and returned contribution response (privacy custom token) tx: %+v", initErr)
		return nil, nil
	}
	return resTx, nil
}

func (blockGenerator *BlockGenerator) buildPDEFeeWithdrawalTx(
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	featureStateDB *statedb.StateDB,
) (metadata.Transaction, error) {
	Logger.log.Info("[PDE Fee Withdrawal] Starting...")
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde withdrawal action: %+v", err)
		return nil, nil
	}
	var pdeFeeWithdrawalRequestAction metadata.PDEFeeWithdrawalRequestAction
	err = json.Unmarshal(contentBytes, &pdeFeeWithdrawalRequestAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde fee withdrawal request action: %+v", err)
		return nil, nil
	}

	if pdeFeeWithdrawalRequestAction.ShardID != shardID {
		return nil, nil
	}

	meta := metadata.NewPDEFeeWithdrawalResponse(
		pdeFeeWithdrawalRequestAction.TxReqID,
		metadata.PDEFeeWithdrawalResponseMeta,
	)

	keyWallet, err := wallet.Base58CheckDeserialize(pdeFeeWithdrawalRequestAction.Meta.WithdrawerAddressStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while deserializing withdrawer address string: %+v", err)
		return nil, nil
	}
	receiverAddr := keyWallet.KeySet.PaymentAddress

	resTx := new(transaction.Tx)
	err = resTx.InitTxSalary(
		pdeFeeWithdrawalRequestAction.Meta.WithdrawalFeeAmt,
		&receiverAddr,
		producerPrivateKey,
		shardView.GetCopiedTransactionStateDB(),
		meta,
	)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while initializing trading fee withdrawal (normal) tx: %+v", err)
		return nil, nil
	}
	return resTx, nil
}
