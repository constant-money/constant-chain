package metadata

// type WithDrawRewardRequest struct {
// 	privacy.PaymentAddress
// 	MetadataBase
// 	TokenID common.Hash
// 	Version int
// }

// func (withDrawRewardRequest WithDrawRewardRequest) Hash() *common.Hash {
// 	if withDrawRewardRequest.Version == 1 {
// 		bArr := append(withDrawRewardRequest.PaymentAddress.Bytes(), withDrawRewardRequest.TokenID.GetBytes()...)
// 		txReqHash := common.HashH(bArr)
// 		return &txReqHash
// 	} else {
// 		record := strconv.Itoa(withDrawRewardRequest.Type)
// 		data := []byte(record)
// 		hash := common.HashH(data)
// 		return &hash
// 	}
// }

// func NewWithDrawRewardRequestFromRPC(data map[string]interface{}) (Metadata, error) {
// 	metadataBase := MetadataBase{
// 		Type: WithDrawRewardRequestMeta,
// 	}
// 	requesterPaymentStr, ok := data["PaymentAddress"].(string)
// 	if !ok {
// 		return nil, errors.New("Invalid payment address receiver")
// 	}
// 	requestTokenID, ok := data["TokenID"].(string)
// 	if !ok {
// 		return nil, errors.New("Invalid token Id")
// 	}
// 	tokenID, err := common.Hash{}.NewHashFromStr(requestTokenID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requesterPublicKeySet, err := wallet.Base58CheckDeserialize(requesterPaymentStr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	result := &WithDrawRewardRequest{
// 		MetadataBase:   metadataBase,
// 		PaymentAddress: requesterPublicKeySet.KeySet.PaymentAddress,
// 		TokenID:        *tokenID,
// 	}

// 	versionFloat, ok := data["Version"].(float64)
// 	if ok {
// 		version := int(versionFloat)
// 		result.Version = version
// 	}
// 	if ok, err := common.SliceExists(AcceptedWithdrawRewardRequestVersion, result.Version); !ok || err != nil {
// 		return nil, errors.Errorf("Invalid version %d", result.Version)
// 	}
// 	return result, nil
// }

// type WithDrawRewardResponse struct {
// 	MetadataBase
// 	TxRequest *common.Hash
// 	TokenID   common.Hash
// 	Version   int
// }

// func NewWithDrawRewardResponse(txRequest *WithDrawRewardRequest, reqID *common.Hash) (Metadata, error) {
// 	metadataBase := MetadataBase{
// 		Type: WithDrawRewardResponseMeta,
// 	}
// 	result := &WithDrawRewardResponse{
// 		MetadataBase: metadataBase,
// 		TxRequest:    reqID,
// 		TokenID:      txRequest.TokenID,
// 	}
// 	result.Version = txRequest.Version

// 	if ok, err := common.SliceExists(AcceptedWithdrawRewardRequestVersion, result.Version); !ok || err != nil {
// 		return nil, errors.Errorf("Invalid version %d", result.Version)
// 	}

// 	return result, nil
// }

// func (withDrawRewardResponse WithDrawRewardResponse) Hash() *common.Hash {
// 	if withDrawRewardResponse.Version == 1 {
// 		if withDrawRewardResponse.TxRequest == nil {
// 			return &common.Hash{}
// 		}
// 		bArr := append(withDrawRewardResponse.TxRequest.GetBytes(), withDrawRewardResponse.TokenID.GetBytes()...)
// 		version := strconv.Itoa(withDrawRewardResponse.Version)
// 		bArr = append(bArr, []byte(version)...)
// 		txResHash := common.HashH(bArr)
// 		return &txResHash
// 	} else {
// 		return withDrawRewardResponse.TxRequest
// 	}
// }

// func (withDrawRewardRequest WithDrawRewardRequest) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, stateDB *statedb.StateDB) bool {
// 	return true
// }

// func (withDrawRewardRequest WithDrawRewardRequest) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
// 	if tx.IsPrivacy() {
// 		return false, errors.New("This transaction is not private")
// 	}

// 	isValid := false
// 	if withDrawRewardRequest.TokenID.IsEqual(&common.PRVCoinID) {
// 		// this is withdraw request for PRV
// 		isValid = true
// 	} else {
// 		// this is withdraw request for token
// 		allTokenID, err := chainRetriever.ListPrivacyTokenAndBridgeTokenAndPRVByShardID(common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte()))
// 		if err != nil {
// 			return false, err
// 		}

// 		for _, availableCoinID := range allTokenID {
// 			cmp := withDrawRewardRequest.TokenID.IsEqual(&availableCoinID)
// 			if cmp {
// 				isValid = true
// 				break
// 			}
// 		}
// 	}
// 	if !isValid {
// 		return false, errors.New("Invalid TokenID, maybe this coin not available at current shard")
// 	}
// 	isPositive := false
// 	tempPublicKey := base58.Base58Check{}.Encode(withDrawRewardRequest.PaymentAddress.Pk, common.Base58Version)
// 	value, err := statedb.GetCommitteeReward(shardViewRetriever.GetShardRewardStateDB(), tempPublicKey, withDrawRewardRequest.TokenID)
// 	if err != nil {
// 		return false, err
// 	}
// 	if value > 0 {
// 		isPositive = true
// 	}

// 	if !isPositive {
// 		return false, errors.New("Not enough reward")
// 	}
// 	receivers, _ := tx.GetReceivers()
// 	if len(receivers) > 0 {
// 		return false, errors.New("This metadata just for request withdraw reward")
// 	}
// 	return true, nil
// }

// func (withDrawRewardRequest WithDrawRewardRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
// 	return false, true, nil
// }

// func (withDrawRewardRequest WithDrawRewardRequest) ValidateMetadataByItself() bool {
// 	// The validation just need to check at tx level, so returning true here
// 	return true
// }

// func (withDrawRewardResponse *WithDrawRewardResponse) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, db *statedb.StateDB) bool {
// 	//this transaction can be a zero-fee transaction, but in fact, user can set nonzero-fee for this tx
// 	return true
// }

// func (withDrawRewardResponse *WithDrawRewardResponse) ValidateTxWithBlockChain(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte, transactionStateDB *statedb.StateDB) (bool, error) {
// 	if tx.IsPrivacy() {
// 		return false, errors.New("This transaction is not private")
// 	}
// 	unique, requesterRes, amountRes, coinID := tx.GetTransferData()
// 	if !unique {
// 		return false, errors.New("Just one receiver")
// 	}
// 	cmp, err := withDrawRewardResponse.TokenID.Cmp(coinID)
// 	if (cmp != 0) || (err != nil) {
// 		return false, errors.Errorf("WithdrawResponse metadata want tokenID %v, got %v, error %v", withDrawRewardResponse.TokenID.String(), coinID.String(), err)
// 	}
// 	tempPublicKey := base58.Base58Check{}.Encode(requesterRes, common.Base58Version)
// 	value, err := statedb.GetCommitteeReward(shardViewRetriever.GetShardRewardStateDB(), tempPublicKey, *coinID)
// 	if (err != nil) || (value == 0) {
// 		return false, errors.New("Not enough reward")
// 	}
// 	if value != amountRes {
// 		return false, errors.New("Wrong amounts")
// 	}
// 	return true, nil
// }

// func (withDrawRewardResponse WithDrawRewardResponse) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
// 	return false, true, nil
// }

// func (withDrawRewardResponse WithDrawRewardResponse) ValidateMetadataByItself() bool {
// 	// The validation just need to check at tx level, so returning true here
// 	return true
// }
