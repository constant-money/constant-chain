package jsonresult

import (
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
)

type CustomToken struct {
	ID                 string   `json:"ID"`
	Name               string   `json:"Name"`
	Symbol             string   `json:"Symbol"`
	Image              string   `json:"Image"`
	Amount             uint64   `json:"Amount"`
	IsPrivacy          bool     `json:"IsPrivacy"`
	IsBridgeToken      bool     `json:"IsBridgeToken"`
	ListTxs            []string `json:"ListTxs"`
	CountTxs           int      `json:"CountTxs"`
	InitiatorPublicKey string   `json:"InitiatorPublicKey"`
	TxInfo             string   `json:"TxInfo"`
}

func NewPrivacyToken(tokenState *statedb.TokenState) *CustomToken {
	customToken := &CustomToken{}
	customToken.ID = tokenState.TokenID().String()
	customToken.Symbol = tokenState.PropertySymbol()
	customToken.Name = tokenState.PropertyName()
	customToken.Amount = tokenState.Amount()
	customToken.IsPrivacy = true
	customToken.TxInfo = base58.Base58Check{}.Encode(tokenState.Info(), common.ZeroByte)
	customToken.CountTxs = len(tokenState.Txs())
	for _, tx := range tokenState.Txs() {
		customToken.ListTxs = append(customToken.ListTxs, tx.String())
	}
	return customToken
}

func NewPrivacyForCrossShard(obj types.CrossShardTokenPrivacyMetaData) *CustomToken {
	customToken := &CustomToken{}
	customToken.ID = obj.TokenID.String()
	customToken.Symbol = obj.PropertySymbol
	customToken.Name = obj.PropertyName
	customToken.Amount = obj.Amount
	//customToken.Image = common.Render(obj.TokenID[:])
	customToken.IsPrivacy = true
	customToken.TxInfo = base58.Base58Check{}.Encode([]byte{}, common.ZeroByte)
	return customToken
}

type ListCustomToken struct {
	ListCustomToken []CustomToken `json:"ListCustomToken"`
}

type GetCustomToken struct {
	IsExist     bool        `json:"IsExist"`
	CustomToken CustomToken `json:"CustomToken"`
}

func NewGetCustomToken(isExist bool, customToken CustomToken) *GetCustomToken {
	return &GetCustomToken{IsExist: isExist, CustomToken: customToken}
}
