package debugtool

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/rpcserver"
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

	if respond.Error != nil{
		return nil, errors.New(fmt.Sprintf("RPC returns an error: %v", respond.Error))
	}

	return &respond, nil
}