package jsonresult

type EstimateFeeResult struct {
	EstimateFeeCoinPerKb uint64
	EstimateTxSizeInKb   uint64
}


func NewEstimateFeeResult(estimateFeeCoinPerKb uint64, estimateTxSizeInKb uint64) *EstimateFeeResult {
	result := &EstimateFeeResult{
		EstimateFeeCoinPerKb: estimateFeeCoinPerKb,
		EstimateTxSizeInKb:   estimateTxSizeInKb,
	}
	return result
}