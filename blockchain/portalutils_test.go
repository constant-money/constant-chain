package blockchain

// import (
// 	"github.com/stretchr/testify/assert"
// 	"testing"
// )

// func TestCalculatePortingFees(t *testing.T) {
// 	result := CalculatePortingFees(3106511852580)
// 	assert.Equal(t, result, uint64(310651185))
// }

// func TestCurrentPortalStateStruct(t *testing.T) {
// 	currentPortalState := &CurrentPortalState{}

// 	assert.NotEqual(t, currentPortalState, nil)
// 	assert.Equal(t, len(currentPortalState.CustodianPoolState), 0)
// 	assert.Equal(t, len(currentPortalState.ExchangeRatesRequests), 0)
// 	assert.Equal(t, len(currentPortalState.WaitingPortingRequests), 0)
// 	assert.Equal(t, len(currentPortalState.WaitingRedeemRequests), 0)
// 	assert.Equal(t, len(currentPortalState.FinalExchangeRatesState), 0)

// 	finalExchangeRates := currentPortalState.FinalExchangeRatesState["abc"]
// 	assert.Equal(t, finalExchangeRates.Rates, nil)

// 	_, ok := currentPortalState.CustodianPoolState["abc"]
// 	assert.Equal(t, ok, false)

// 	for _, v := range currentPortalState.CustodianPoolState {
// 		assert.Equal(t, 1, 0)
// 		assert.NotNil(t, v)
// 	}
// }
