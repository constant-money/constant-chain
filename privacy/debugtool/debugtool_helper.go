package debugtool

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/rpcserver"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
)

func EncodeBase58Check(data []byte) string {
	b := base58.Base58Check{}.Encode(data, 0)
	return b
}

func DecodeBase58Check(s string) ([]byte, error) {
	b, _, err := base58.Base58Check{}.Decode(s)
	return b, err
}

/*Common functions*/
// RandIntInterval returns a random int in range [L; R]
func RandIntInterval(L, R int) int {
	length := R - L + 1
	r := common.RandInt() % length
	return L + r
}

func ParseResponse(respondInBytes []byte) (*rpcserver.JsonResponse, error) {
	var respond rpcserver.JsonResponse
	err := json.Unmarshal(respondInBytes, &respond)
	if err != nil {
		return nil, err
	}

	return &respond, nil
}

func ParseCoinFromJsonResponse(b []byte) ([]jsonresult.ICoinInfo, error){
	respond, err := ParseResponse(b)
	if err != nil{
		panic(err)
	}

	var msg json.RawMessage
	err = json.Unmarshal(respond.Result, &msg)
	if err != nil {
		panic(err)
	}

	var tmp jsonresult.ListOutputCoins
	err = json.Unmarshal(msg, &tmp)
	if err != nil {
		panic(err)
	}

	resultOutCoins := make([]jsonresult.ICoinInfo, 0)

	listOutputCoins := tmp.Outputs
	for _, value := range listOutputCoins {
		for _, outCoin := range value {
			out, err := jsonresult.NewCoinFromJsonOutCoin(outCoin)
			if err != nil {
				return nil, err
			}

			resultOutCoins = append(resultOutCoins, out)
		}
	}

	return resultOutCoins, nil
}