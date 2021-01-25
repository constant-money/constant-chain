package statedb

import (
	"encoding/json"
)

type UTXO struct {
	txHash       string
	outputIdx    int
	outputAmount uint64
}

func NewUTXO() *UTXO {
	return &UTXO{}
}

func NewUTXOWithValue(
	txHash string,
	outputIdx int,
	outputAmount uint64,
) *UTXO {
	return &UTXO{
		txHash:       txHash,
		outputAmount: outputAmount,
		outputIdx:    outputIdx,
	}
}

func (uo *UTXO) GetTxHash() string {
	return uo.txHash
}

func (uo *UTXO) SetTxHash(txHash string) {
	uo.txHash = txHash
}

func (uo *UTXO) GetOutputAmount() uint64 {
	return uo.outputAmount
}

func (uo *UTXO) SetOutputAmount(amount uint64) {
	uo.outputAmount = amount
}

func (uo *UTXO) GetOutputIndex() int {
	return uo.outputIdx
}

func (uo *UTXO) SetOutputIndex(index int) {
	uo.outputIdx = index
}

func (uo *UTXO) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		TxHash       string
		OutputIdx    int
		OutputAmount uint64
	}{
		TxHash:       uo.txHash,
		OutputIdx:    uo.outputIdx,
		OutputAmount: uo.outputAmount,
	})
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (uo *UTXO) UnmarshalJSON(data []byte) error {
	temp := struct {
		TxHash       string
		OutputIdx    int
		OutputAmount uint64
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	uo.txHash = temp.TxHash
	uo.outputIdx = temp.OutputIdx
	uo.outputAmount = temp.OutputAmount
	return nil
}
