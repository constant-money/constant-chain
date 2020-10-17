package metadata

type PortalLiquidateTopPercentileExchangeRatesContent struct {
	CustodianAddress   string
	Status             string
	MetaType           int
	TP                 map[string]LiquidateTopPercentileExchangeRatesDetail
	RemainUnlockAmount map[string]uint64
}

type LiquidateTopPercentileExchangeRatesDetail struct {
	TPKey                    int
	TPValue                  uint64
	HoldAmountFreeCollateral uint64
	HoldAmountPubToken       uint64
}

type LiquidateTopPercentileExchangeRatesStatus struct {
	CustodianAddress string
	Status           byte
	Rates            map[string]LiquidateTopPercentileExchangeRatesDetail //ptoken | detail
}

func NewLiquidateTopPercentileExchangeRatesStatus(custodianAddress string, status byte, rates map[string]LiquidateTopPercentileExchangeRatesDetail) *LiquidateTopPercentileExchangeRatesStatus {
	return &LiquidateTopPercentileExchangeRatesStatus{CustodianAddress: custodianAddress, Status: status, Rates: rates}
}
