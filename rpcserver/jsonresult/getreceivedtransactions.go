package jsonresult

type ReceivedTransaction struct {
	TxDetail        *TransactionDetail	   `json:"TransactionDetail"`
	ReceivedAmounts map[string]interface{} `json:"ReceivedAmounts"`
	FromShardID     byte                   `json:"FromShardID"`
}

type ReceivedCoin struct {
	PublicKey string `json:"PublicKey"`
	Info      string `json:"Info"`
	Value     uint64 `json:"Value"`
}

type ListReceivedTransaction struct {
	ReceivedTransactions []ReceivedTransaction `json:"ReceivedTransactions"`
}

type ReceivedTransactionV2 struct {
	TxDetail           *TransactionDetail
	ReceivedAmounts    map[string]interface{} `json:"ReceivedAmounts"`
	InputSerialNumbers map[string][]string    `json:"InputSerialNumbers"`
	FromShardID        byte                   `json:"FromShardID"`
}

type ListReceivedTransactionV2 struct {
	ReceivedTransactions []ReceivedTransactionV2 `json:"ReceivedTransactions"`
}

type Temp struct {

}