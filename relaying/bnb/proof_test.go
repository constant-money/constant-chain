package bnb

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/binance-chain/go-sdk/types/msg"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetProofByTxHash(t *testing.T) {
	txProof, err := getProofByTxHash("421B68266AC570DEC49A12B1DDA0518D59205F4A874A24DB0F9448D4E03720A3", MainnetURLRemote)
	assert.Nil(t, err)
	fmt.Printf("txProof %v\n", txProof.Data)

	tx, err := ParseTxFromData(txProof.Data)
	assert.Nil(t, err)
	fmt.Printf("tx %+v\n", tx)
}

func TestParseBNBTxFromData(t *testing.T) {
	data, _ := hex.DecodeString("A702F0625DEE0A482A2C87FA0A200A141C4693E2455A9DA63C5D8F1240BE3D8466CD0E4612080A03424E4210904E12200A141C4693E2455A9DA63C5D8F1240BE3D8466CD0E4612080A03424E4210904E12710A26EB5AE98721037985B53085AEF69B8B481B5BF35BDE7B20DBF98DB970909048F725849412E3AC12408120FDA9DD3326D440C8D058851AFB5830BC6E12EF896B513EE0095D47273DFF0A640382A848E9B628F6B24672515C33776ABE38EAA5FA188DC8467DA1A63C3618A25C20D0EC031A647B2270726F746F636F6C223A22616C6570682D6F6666636861696E222C2276657273696F6E223A312C22636F6E74656E74223A22516D63644C5A6853736D47364C72445A44397364616A647077616266345978355650483950376152393765324D52227D")
	tx, err := ParseTxFromData(data)

	senderAddress := "bnb1r3rf8cj9t2w6v0za3ufyp03as3nv6rjxteku6g"
	receiverAddress := "bnb1r3rf8cj9t2w6v0za3ufyp03as3nv6rjxteku6g"
	amount := int64(10000)
	assert.Nil(t, err)
	assert.Equal(t, senderAddress, tx.Msgs[0].(msg.SendMsg).Inputs[0].Address.String())
	assert.Equal(t, receiverAddress, tx.Msgs[0].(msg.SendMsg).Outputs[0].Address.String())

	outputCoins := tx.Msgs[0].(msg.SendMsg).Outputs[0].Coins
	actualAmount := int64(0)
	for _, coin := range outputCoins {
		actualAmount += coin.Amount
	}
	assert.Equal(t, amount, actualAmount)
}

func TestBNBProof(t *testing.T) {
	txIndex := 0
	blockHeight := int64(60479432)

	// build bnb proof
	bnbProof := new(BNBProof)
	err := bnbProof.Build(txIndex, blockHeight, MainnetURLRemote)
	assert.Nil(t, err)
	assert.Equal(t, blockHeight, bnbProof.BlockHeight)

	// verify bnb proof
	isValid, err := bnbProof.Verify(nil)
	assert.Nil(t, err)
	assert.Equal(t, true, isValid)

	// encode bnb proof to string
	jsonStr, _ := json.Marshal(bnbProof)
	b64EncodeProof := base64.StdEncoding.EncodeToString(jsonStr)
	fmt.Printf("b64EncodeProof: %+v\n", b64EncodeProof)

	// decode bnb proof from string
	bnbProof2, err := ParseBNBProofFromB64EncodeStr(b64EncodeProof)
	assert.Nil(t, err)

	isValid2, err := bnbProof2.Verify(nil)
	assert.Nil(t, err)
	assert.Equal(t, true, isValid2)
}

type PortingMemoBNB struct {
	PortingID string `json:"PortingID"`
}

type RedeemMemoBNB struct {
	RedeemID                  string `json:"RedeemID"`
	CustodianIncognitoAddress string `json:"CustodianIncognitoAddress"`
}

func TestB64EncodeMemo(t *testing.T) {
	portingID := "porting-10"
	memoPorting := PortingMemoBNB{PortingID: portingID}
	memoPortingBytes, err := json.Marshal(memoPorting)
	fmt.Printf("err: %v\n", err)
	memoPortingStr := base64.StdEncoding.EncodeToString(memoPortingBytes)
	fmt.Printf("memoPortingStr: %v\n", memoPortingStr)

	redeemID := "bnb13"
	custodianIncAddr := "12Rwz4HXkVABgRnSb5Gfu1FaJ7auo3fLNXVGFhxx1dSytxHpWhbkimT1Mv5Z2oCMsssSXTVsapY8QGBZd2J4mPiCTzJAtMyCzb4dDcy"
	memoRedeem := RedeemMemoBNB{RedeemID: redeemID, CustodianIncognitoAddress: custodianIncAddr}
	memoRedeemBytes, err := json.Marshal(memoRedeem)
	fmt.Printf("err: %v\n", err)
	memoRedeemHash := common.HashB(memoRedeemBytes)
	memoRedeemStr := base64.StdEncoding.EncodeToString(memoRedeemHash)
	fmt.Printf("memoRedeemStr: %v\n", memoRedeemStr)
}

func TestBuildAndPushBNBProof(t *testing.T) {
	txIndex := 0
	blockHeight := int64(80536994)
	url := TestnetURLRemote

	portingProof, err := BuildProof(txIndex, blockHeight, url)
	if err != nil {
		fmt.Printf("err BuildProof: %+v\n", err)
	}
	fmt.Printf("BNB portingProof: %+v\n", portingProof)

	//redeemProof, err := BuildProof(txIndex, blockHeight, url)
	//if err != nil {
	//	fmt.Printf("err BuildProof: %+v\n", err)
	//}
	//fmt.Printf("BNB redeemProof: %+v\n", redeemProof)
}
