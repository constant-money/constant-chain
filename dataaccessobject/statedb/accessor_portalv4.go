package statedb

import "github.com/incognitochain/incognito-chain/common"

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
