package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/coin"
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
	txRandomStr string,
	receiveAmt uint64,
	tokenIDStr string,
	producerPrivateKey *privacy.PrivateKey,
	transactionStateDB *statedb.StateDB,
	meta metadata.Metadata,
) (metadata.Transaction, error) {

	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while converting tokenid to hash: %+v", err)
		return nil, err
	}

	var txParam transaction.TxSalaryOutputParams
	if len(txRandomStr) > 0 {
		publicKey, txRandom, err := coin.ParseOTAInfoFromString(receiverAddressStr, txRandomStr)
		if err != nil {
			return nil, err
		}
		txParam = transaction.TxSalaryOutputParams{Amount: receiveAmt, ReceiverAddress: nil, PublicKey: publicKey, TxRandom: txRandom, TokenID: tokenID, Info: []byte{}}
	} else {
		return nil, errors.New("cannot create trade response without txRandom info")
	}
	return txParam.BuildTxSalary(nil, producerPrivateKey, transactionStateDB, meta)
}

func (blockGenerator *BlockGenerator) buildPDECrossPoolTradeRefundTx(
	instStatus string,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	beaconView *BeaconBestState,
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
		crossPoolTradeRefundContent.TxRandomStr,
		crossPoolTradeRefundContent.Amount,
		crossPoolTradeRefundContent.TokenIDStr,
		producerPrivateKey,
		shardView.GetCopiedTransactionStateDB(),
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
	beaconView *BeaconBestState,
) (metadata.Transaction, error) {
	pdeTradeRequestAction, err := parseTradeRefundContent(contentStr)
	if err != nil {
		return nil, nil
	}

	meta := metadata.NewPDETradeResponse(
		instStatus,
		pdeTradeRequestAction.TxReqID,
		metadata.PDETradeResponseMeta,
	)

	resTx, err := buildTradeResTx(
		pdeTradeRequestAction.Meta.TraderAddressStr,
		pdeTradeRequestAction.Meta.TxRandomStr,
		pdeTradeRequestAction.Meta.SellAmount+pdeTradeRequestAction.Meta.TradingFee,
		pdeTradeRequestAction.Meta.TokenIDToSellStr,
		producerPrivateKey,
		shardView.GetCopiedTransactionStateDB(),
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
	beaconView *BeaconBestState,
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

	finalCrossPoolTradeAcceptedContent := crossPoolTradeAcceptedContents[len - 1]

	meta := metadata.NewPDECrossPoolTradeResponse(
		instStatus,
		finalCrossPoolTradeAcceptedContent.RequestedTxID,
		metadata.PDECrossPoolTradeResponseMeta,
	)

	resTx, err := buildTradeResTx(
		finalCrossPoolTradeAcceptedContent.TraderAddressStr,
		finalCrossPoolTradeAcceptedContent.TxRandomStr,
		finalCrossPoolTradeAcceptedContent.ReceiveAmount,
		finalCrossPoolTradeAcceptedContent.TokenIDToBuyStr,
		producerPrivateKey,
		shardView.GetCopiedTransactionStateDB(),
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
	beaconView *BeaconBestState,
) (metadata.Transaction, error) {
	pdeTradeAcceptedContent, err := parseTradeAcceptedContent(contentStr)
	if err != nil {
		return nil, nil
	}

	meta := metadata.NewPDETradeResponse(
		instStatus,
		pdeTradeAcceptedContent.RequestedTxID,
		metadata.PDETradeResponseMeta,
	)

	resTx, err := buildTradeResTx(
		pdeTradeAcceptedContent.TraderAddressStr,
		pdeTradeAcceptedContent.TxRandomStr,
		pdeTradeAcceptedContent.ReceiveAmount,
		pdeTradeAcceptedContent.TokenIDToBuyStr,
		producerPrivateKey,
		shardView.GetCopiedTransactionStateDB(),
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
	beaconView *BeaconBestState,
) (metadata.Transaction, error) {
	Logger.log.Info("[PDE Trade] Starting...")
	if instStatus == common.PDETradeRefundChainStatus {
		return blockGenerator.buildPDETradeRefundTx(
			instStatus,
			contentStr,
			producerPrivateKey,
			shardID,
			shardView,
			beaconView,
		)
	}
	return blockGenerator.buildPDETradeAcceptedTx(
		instStatus,
		contentStr,
		producerPrivateKey,
		shardID,
		shardView,
		beaconView,
	)
}

func (blockGenerator *BlockGenerator) buildPDECrossPoolTradeIssuanceTx(
	instStatus string,
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	beaconView *BeaconBestState,
) (metadata.Transaction, error) {
	Logger.log.Info("[PDE Cross Pool Trade] Starting...")
	if instStatus == common.PDECrossPoolTradeFeeRefundChainStatus || instStatus == common.PDECrossPoolTradeSellingTokenRefundChainStatus {
		return blockGenerator.buildPDECrossPoolTradeRefundTx(
			instStatus,
			contentStr,
			producerPrivateKey,
			shardID,
			shardView,
			beaconView,
		)
	}
	return blockGenerator.buildPDECrossPoolTradeAcceptedTx(
		instStatus,
		contentStr,
		producerPrivateKey,
		shardID,
		shardView,
		beaconView,
	)
}

func (blockGenerator *BlockGenerator) buildPDEWithdrawalTx(
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	beaconView *BeaconBestState,
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

	txParam := transaction.TxSalaryOutputParams{Amount: wdAcceptedContent.DeductingPoolValue, ReceiverAddress: &receiverAddr, TokenID: tokenID}
	otaCoin, err := txParam.GenerateOutputCoin()
	if err != nil {
		Logger.log.Errorf("Cannot get new coin from amount and payment address")
		return nil, err
	}
	meta.SetSharedRandom(otaCoin.GetSharedRandom().ToBytesS())
	return txParam.BuildTxSalary(otaCoin, producerPrivateKey, shardView.GetCopiedTransactionStateDB(), meta)
}

func (blockGenerator *BlockGenerator) buildPDERefundContributionTx(
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	beaconView *BeaconBestState,
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
	// create ota coin
	txParam := transaction.TxSalaryOutputParams{Amount: refundContribution.ContributedAmount, ReceiverAddress: &receiverAddr, TokenID: tokenID}
	otaCoin, err := txParam.GenerateOutputCoin()
	if err != nil {
		Logger.log.Errorf("Cannot get new coin from amount and receiver")
		return nil, err
	}
	// set shareRandom for metadata
	meta.SetSharedRandom(otaCoin.GetSharedRandom().ToBytesS())
	return txParam.BuildTxSalary(otaCoin, producerPrivateKey, shardView.GetCopiedTransactionStateDB(), meta)
}

func (blockGenerator *BlockGenerator) buildPDEMatchedNReturnedContributionTx(
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	beaconView *BeaconBestState,
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
	// create ota coin
	txParam := transaction.TxSalaryOutputParams{Amount: matchedNReturnedContribution.ReturnedContributedAmount, ReceiverAddress: &receiverAddr, TokenID: tokenID}
	otaCoin, err := txParam.GenerateOutputCoin()
	if err != nil {
		Logger.log.Errorf("Cannot get new coin from amount and receiver")
		return nil, err
	}
	// set shareRandom for metadata
	meta.SetSharedRandom(otaCoin.GetSharedRandom().ToBytesS())
	return txParam.BuildTxSalary(otaCoin, producerPrivateKey, shardView.GetCopiedTransactionStateDB(), meta)
}

func (blockGenerator *BlockGenerator) buildPDEFeeWithdrawalTx(
	contentStr string,
	producerPrivateKey *privacy.PrivateKey,
	shardID byte,
	shardView *ShardBestState,
	beaconView *BeaconBestState,
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


	txParam := transaction.TxSalaryOutputParams{
		Amount: pdeFeeWithdrawalRequestAction.Meta.WithdrawalFeeAmt,
		ReceiverAddress: &receiverAddr,
		TokenID: &common.PRVCoinID}
	otaCoin, err := txParam.GenerateOutputCoin()
	if err != nil {
		Logger.log.Errorf("Cannot get new coin from amount and payment address")
		return nil, err
	}
	meta.SetSharedRandom(otaCoin.GetSharedRandom().ToBytesS())
	return txParam.BuildTxSalary(otaCoin, producerPrivateKey, shardView.GetCopiedTransactionStateDB(), meta)
}
