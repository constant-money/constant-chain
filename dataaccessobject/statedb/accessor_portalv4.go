package statedb

// Unshield request status
func StorePortalUnshieldRequestStatus(stateDB *StateDB, unshieldID string, statusContent []byte) error {
	statusType := PortalUnshieldRequestStatusPrefix()
	statusSuffix := []byte(unshieldID)
	err := StorePortalStatus(stateDB, statusType, statusSuffix, statusContent)
	if err != nil {
		return NewStatedbError(StorePortalRedeemRequestStatusError, err)
	}

	return nil
}

func GetPortalUnshieldRequestStatus(stateDB *StateDB, unshieldID string) ([]byte, error) {
	statusType := PortalUnshieldRequestStatusPrefix()
	statusSuffix := []byte(unshieldID)
	data, err := GetPortalStatus(stateDB, statusType, statusSuffix)
	if err != nil && err.(*StatedbError).GetErrorCode() != ErrCodeMessage[GetPortalStatusNotFoundError].Code {
		return []byte{}, NewStatedbError(GetPortalRedeemRequestStatusError, err)
	}

	return data, nil
}