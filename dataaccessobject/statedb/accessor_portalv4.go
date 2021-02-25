package statedb

import "github.com/incognitochain/incognito-chain/common"

// ================= Shielding Request =================
func StoreShieldingRequestStatus(stateDB *StateDB, txID string, statusContent []byte) error {
	statusType := PortalShieldingRequestStatusPrefix()
	statusSuffix := []byte(txID)
	err := StorePortalStatus(stateDB, statusType, statusSuffix, statusContent)
	if err != nil {
		return NewStatedbError(StorePortalShieldingRequestStatusError, err)
	}

	return nil
}

func GetShieldingRequestStatus(stateDB *StateDB, txID string) ([]byte, error) {
	statusType := PortalShieldingRequestStatusPrefix()
	statusSuffix := []byte(txID)
	data, err := GetPortalStatus(stateDB, statusType, statusSuffix)
	if err != nil {
		return []byte{}, NewStatedbError(GetPortalShieldingRequestStatusError, err)
	}

	return data, nil
}

func GetShieldingRequestsByTokenID(stateDB *StateDB, tokenID string) (map[string]*ShieldingRequest, error) {
	return stateDB.getShieldingRequestsByTokenID(tokenID), nil
}

func StoreShieldingRequests(stateDB *StateDB, shieldingRequests map[string]*ShieldingRequest) error {
	for keyStr, shieldingReq := range shieldingRequests {
		key, err := common.Hash{}.NewHashFromStr(keyStr)
		if err != nil {
			return NewStatedbError(StorePortalShieldingRequestsError, err)
		}
		err = stateDB.SetStateObject(PortalShieldingRequestObjectType, *key, shieldingReq)
		if err != nil {
			return NewStatedbError(StorePortalShieldingRequestsError, err)
		}
	}

	return nil
}

// ================= List UTXOs =================
func GetUTXOsByTokenID(stateDB *StateDB, tokenID string) (map[string]*UTXO, error) {
	return stateDB.getUTXOsByTokenID(tokenID), nil
}

func StoreUTXOs(stateDB *StateDB, utxos map[string]*UTXO) error {
	for keyStr, utxo := range utxos {
		key, err := common.Hash{}.NewHashFromStr(keyStr)
		if err != nil {
			return NewStatedbError(StorePortalUTXOsError, err)
		}
		err = stateDB.SetStateObject(PortalUTXOObjectType, *key, utxo)
		if err != nil {
			return NewStatedbError(StorePortalUTXOsError, err)
		}
	}

	return nil
}

// ================= List Waiting Unshielding Requests =================
func GetWaitingUnshieldRequestsByTokenID(stateDB *StateDB, tokenID string) (map[string]*WaitingUnshieldRequest, error) {
	return stateDB.getListWaitingUnshieldRequestsByTokenID(tokenID), nil
}

func StoreWaitingUnshieldRequests(
	stateDB *StateDB,
	waitingUnshieldReqs map[string]*WaitingUnshieldRequest) error {
	for keyStr, waitingReq := range waitingUnshieldReqs {
		key, err := common.Hash{}.NewHashFromStr(keyStr)
		if err != nil {
			return NewStatedbError(StorePortalListWaitingUnshieldRequestError, err)
		}
		err = stateDB.SetStateObject(PortalWaitingUnshieldObjectType, *key, waitingReq)
		if err != nil {
			return NewStatedbError(StorePortalListWaitingUnshieldRequestError, err)
		}
	}

	return nil
}

func DeleteWaitingUnshieldRequest(stateDB *StateDB, tokenID string, unshieldID string) {
	key := GenerateWaitingUnshieldRequestObjectKey(tokenID, unshieldID)
	stateDB.MarkDeleteStateObject(PortalWaitingUnshieldObjectType, key)
}

// ================= Unshielding Request Status =================
// Store and get the status of the Unshield Request by unshieldID
func StorePortalUnshieldRequestStatus(stateDB *StateDB, unshieldID string, statusContent []byte) error {
	statusType := PortalUnshieldRequestStatusPrefix()
	statusSuffix := []byte(unshieldID)
	err := StorePortalStatus(stateDB, statusType, statusSuffix, statusContent)
	if err != nil {
		return NewStatedbError(StorePortalUnshieldRequestStatusError, err)
	}

	return nil
}

func GetPortalUnshieldRequestStatus(stateDB *StateDB, unshieldID string) ([]byte, error) {
	statusType := PortalUnshieldRequestStatusPrefix()
	statusSuffix := []byte(unshieldID)
	data, err := GetPortalStatus(stateDB, statusType, statusSuffix)
	if err != nil && err.(*StatedbError).GetErrorCode() != ErrCodeMessage[GetPortalStatusNotFoundError].Code {
		return []byte{}, NewStatedbError(GetPortalUnshieldRequestStatusError, err)
	}

	return data, nil
}

// ================= Batching Unshielding Request Status =================
// Store and get the status of the Unshield Request by unshieldID
func StorePortalBatchUnshieldRequestStatus(stateDB *StateDB, batchID string, statusContent []byte) error {
	statusType := PortalBatchUnshieldRequestStatusPrefix()
	statusSuffix := []byte(batchID)
	err := StorePortalStatus(stateDB, statusType, statusSuffix, statusContent)
	if err != nil {
		return NewStatedbError(StorePortalBatchUnshieldRequestStatusError, err)
	}

	return nil
}

func GetPortalBatchUnshieldRequestStatus(stateDB *StateDB, batchID string) ([]byte, error) {
	statusType := PortalBatchUnshieldRequestStatusPrefix()
	statusSuffix := []byte(batchID)
	data, err := GetPortalStatus(stateDB, statusType, statusSuffix)
	if err != nil && err.(*StatedbError).GetErrorCode() != ErrCodeMessage[GetPortalStatusNotFoundError].Code {
		return []byte{}, NewStatedbError(GetPortalBatchUnshieldRequestStatusError, err)
	}

	return data, nil
}

// ================= Unshielding Batch Replacement Status =================
// Store and get the status of the Unshield Batch Replacement Request by batchID
func StorePortalUnshieldBatchReplacementRequestStatus(stateDB *StateDB, txID string, statusContent []byte) error {
	statusType := PortalUnshielReplacementFeeBatchStatusPrefix()
	statusSuffix := []byte(txID)
	err := StorePortalStatus(stateDB, statusType, statusSuffix, statusContent)
	if err != nil {
		return NewStatedbError(StorePortalUnshieldBatchReplacementRequestStatusError, err)
	}

	return nil
}

func GetPortalUnshieldBatchReplacementRequestStatus(stateDB *StateDB, txID string) ([]byte, error) {
	statusType := PortalUnshielReplacementFeeBatchStatusPrefix()
	statusSuffix := []byte(txID)
	data, err := GetPortalStatus(stateDB, statusType, statusSuffix)
	if err != nil && err.(*StatedbError).GetErrorCode() != ErrCodeMessage[GetPortalStatusNotFoundError].Code {
		return []byte{}, NewStatedbError(GetPortalUnshieldBatchReplacementRequestStatusError, err)
	}

	return data, nil
}

// ================= Submit unshield batch confirmed Status =================
// Store and get the status of the Unshield Batch Replacement Request by batchID
func StorePortalSubmitConfirmedTxRequestStatus(stateDB *StateDB, txID string, statusContent []byte) error {
	statusType := PortalSubmitConfirmedTxStatusPrefix()
	statusSuffix := []byte(txID)
	err := StorePortalStatus(stateDB, statusType, statusSuffix, statusContent)
	if err != nil {
		return NewStatedbError(StorePortalSubmitConfirmedTxRequestStatusError, err)
	}

	return nil
}

func GetPortalSubmitConfirmedTxRequestStatus(stateDB *StateDB, txID string) ([]byte, error) {
	statusType := PortalSubmitConfirmedTxStatusPrefix()
	statusSuffix := []byte(txID)
	data, err := GetPortalStatus(stateDB, statusType, statusSuffix)
	if err != nil && err.(*StatedbError).GetErrorCode() != ErrCodeMessage[GetPortalStatusNotFoundError].Code {
		return []byte{}, NewStatedbError(GetPortalSubmitConfirmedTxRequestStatusError, err)
	}

	return data, nil
}
