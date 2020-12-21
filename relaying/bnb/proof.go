package bnb

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/binance-chain/go-sdk/client/rpc"
	bnbtx "github.com/binance-chain/go-sdk/types/tx"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
)

func getProofByTxHash(txHashStr string, url string) (*types.TxProof, *BNBRelayingError) {
	txHash, err := hex.DecodeString(txHashStr)
	if err != nil {
		return nil, NewBNBRelayingError(UnexpectedErr, err)
	}

	client := client.NewHTTP(url, "/websocket")
	err = client.Start()
	if err != nil {
		// handle error
	}
	defer client.Stop()
	tx, err := client.Tx(txHash, true)
	//fmt.Printf("tx: %+v\n", tx)

	return &tx.Proof, nil
}

func getTxsInBlockHeight(blockHeight int64, url string) (*types.Txs, *BNBRelayingError) {
	block, err := GetBlock(blockHeight, url)
	if err != nil {
		return nil, err
	}

	return &block.Txs, nil
}

// GetBlock call API to url to get bnb block by blockHeight
func GetBlock(blockHeight int64, url string) (*types.Block, *BNBRelayingError) {
	client := client.NewHTTP(url, "/websocket")
	err := client.Start()
	if err != nil {
		// handle error
	}
	defer client.Stop()
	block, err := client.Block(&blockHeight)

	return block.Block, nil
}

type BNBProof struct {
	Proof       *types.TxProof
	BlockHeight int64
}

// buildProof creates a proof for tx at indexTx in block height
// Call API get all txs in a block height and build proof from those txs by Tendermint's code
func (p *BNBProof) Build(indexTx int, blockHeight int64, url string) *BNBRelayingError {
	txs, err := getTxsInBlockHeight(blockHeight, url)
	if err != nil {
		return err
	}

	proof := txs.Proof(indexTx)

	p.BlockHeight = blockHeight
	p.Proof = &proof

	return nil
}

func (p *BNBProof) Verify(dataHash []byte) (bool, *BNBRelayingError) {
	err := p.Proof.Validate(dataHash)
	if err != nil {
		return false, NewBNBRelayingError(InvalidTxProofErr, err)
	}
	return true, nil
}

func ParseBNBProofFromB64EncodeStr(b64EncodedStr string) (*BNBProof, *BNBRelayingError) {
	jsonBytes, err := base64.StdEncoding.DecodeString(b64EncodedStr)
	if err != nil {
		return nil, NewBNBRelayingError(UnexpectedErr, err)
	}

	proof := BNBProof{}
	err = json.Unmarshal(jsonBytes, &proof)
	if err != nil {
		return nil, NewBNBRelayingError(UnexpectedErr, err)
	}

	return &proof, nil
}

func ParseTxFromData(data []byte) (*bnbtx.StdTx, *BNBRelayingError) {
	tx, err := rpc.ParseTx(bnbtx.Cdc, data)
	if err != nil {
		return nil, NewBNBRelayingError(UnexpectedErr, err)
	}
	stdTx := tx.(bnbtx.StdTx)
	return &stdTx, nil
}

func BuildProof(txIndex int, blockHeight int64, url string) (string, error) {
	bnbProof := new(BNBProof)
	err := bnbProof.Build(txIndex, blockHeight, url)
	if err != nil {
		return "", err
	}

	bnbProofBytes, err2 := json.Marshal(bnbProof)
	if err2 != nil {
		return "", err2
	}

	bnbProofStr := base64.StdEncoding.EncodeToString(bnbProofBytes)

	return bnbProofStr, nil
}
