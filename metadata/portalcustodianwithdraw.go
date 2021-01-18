package metadata

// type PortalCustodianWithdrawRequest struct {
// 	MetadataBase
// 	PaymentAddress string
// 	Amount         uint64
// }

// type PortalCustodianWithdrawRequestAction struct {
// 	Meta    PortalCustodianWithdrawRequest
// 	TxReqID common.Hash
// 	ShardID byte
// }

// type PortalCustodianWithdrawRequestContent struct {
// 	PaymentAddress       string
// 	Amount               uint64
// 	RemainFreeCollateral uint64
// 	TxReqID              common.Hash
// 	ShardID              byte
// }

// type CustodianWithdrawRequestStatus struct {
// 	PaymentAddress                string
// 	Amount                        uint64
// 	Status                        int
// 	RemainCustodianFreeCollateral uint64
// }

// func NewCustodianWithdrawRequestStatus(paymentAddress string, amount uint64, status int, remainCustodianFreeCollateral uint64) *CustodianWithdrawRequestStatus {
// 	return &CustodianWithdrawRequestStatus{PaymentAddress: paymentAddress, Amount: amount, Status: status, RemainCustodianFreeCollateral: remainCustodianFreeCollateral}
// }

// func NewPortalCustodianWithdrawRequest(metaType int, paymentAddress string, amount uint64) (*PortalCustodianWithdrawRequest, error) {
// 	metadataBase := MetadataBase{
// 		Type: metaType, Sig: []byte{},
// 	}

// 	portalCustodianWithdrawReq := &PortalCustodianWithdrawRequest{
// 		PaymentAddress: paymentAddress,
// 		Amount:         amount,
// 	}

// 	portalCustodianWithdrawReq.MetadataBase = metadataBase

// 	return portalCustodianWithdrawReq, nil
// }

// func (*PortalCustodianWithdrawRequest) ShouldSignMetaData() bool { return true }

// func (Withdraw PortalCustodianWithdrawRequest) ValidateTxWithBlockChain(
// 	txr Transaction,
// 	chainRetriever ChainRetriever,
// 	shardViewRetriever ShardViewRetriever,
// 	beaconViewRetriever BeaconViewRetriever,
// 	shardID byte,
// 	db *statedb.StateDB,
// ) (bool, error) {
// 	// NOTE: verify supported tokens pair as needed
// 	return true, nil
// }

// func (Withdraw PortalCustodianWithdrawRequest) ValidateSanityData(chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, beaconHeight uint64, tx Transaction) (bool, bool, error) {
// 	if tx.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(tx).String() == "*transaction.Tx" {
// 		return true, true, nil
// 	}

// 	if len(Withdraw.PaymentAddress) <= 0 {
// 		return false, false, errors.New("Payment address should be not empty")
// 	}

// 	// validate Payment address
// 	keyWallet, err := wallet.Base58CheckDeserialize(Withdraw.PaymentAddress)
// 	if err != nil {
// 		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("ContributorAddressStr incorrect"))
// 	}

// 	incogAddr := keyWallet.KeySet.PaymentAddress
// 	if len(incogAddr.Pk) == 0 {
// 		return false, false, errors.New("wrong custodian incognito address")
// 	}
// 	if ok, err := tx.CheckAuthorizedSender(incogAddr.Pk); err != nil || !ok {
// 		return false, false, errors.New("Withdraw request is unauthorized")
// 	}

// 	// check tx type
// 	if tx.GetType() != common.TxNormalType {
// 		return false, false, errors.New("tx custodian deposit must be TxNormalType")
// 	}

// 	if Withdraw.Amount <= 0 {
// 		return false, false, errors.New("Amount should be larger than 0")
// 	}

// 	return true, true, nil
// }

// func (Withdraw PortalCustodianWithdrawRequest) ValidateMetadataByItself() bool {
// 	return Withdraw.Type == PortalCustodianWithdrawRequestMeta
// }

// func (Withdraw PortalCustodianWithdrawRequest) Hash() *common.Hash {
// 	record := Withdraw.MetadataBase.Hash().String()
// 	record += Withdraw.PaymentAddress
// 	record += strconv.FormatUint(Withdraw.Amount, 10)
// 	if Withdraw.Sig != nil && len(Withdraw.Sig) != 0 {
// 		record += string(Withdraw.Sig)
// 	}
// 	// final hash
// 	hash := common.HashH([]byte(record))
// 	return &hash
// }

// func (Withdraw PortalCustodianWithdrawRequest) HashWithoutSig() *common.Hash {
// 	record := Withdraw.MetadataBase.Hash().String()
// 	record += Withdraw.PaymentAddress
// 	record += strconv.FormatUint(Withdraw.Amount, 10)

// 	// final hash
// 	hash := common.HashH([]byte(record))
// 	return &hash
// }

// func (Withdraw *PortalCustodianWithdrawRequest) BuildReqActions(tx Transaction, chainRetriever ChainRetriever, shardViewRetriever ShardViewRetriever, beaconViewRetriever BeaconViewRetriever, shardID byte) ([][]string, error) {
// 	actionContent := PortalCustodianWithdrawRequestAction{
// 		Meta:    *Withdraw,
// 		TxReqID: *tx.Hash(),
// 		ShardID: shardID,
// 	}
// 	actionContentBytes, err := json.Marshal(actionContent)
// 	if err != nil {
// 		return [][]string{}, err
// 	}
// 	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
// 	action := []string{strconv.Itoa(PortalCustodianWithdrawRequestMeta), actionContentBase64Str}
// 	return [][]string{action}, nil
// }

// func (Withdraw *PortalCustodianWithdrawRequest) CalculateSize() uint64 {
// 	return calculateSize(Withdraw)
// }
