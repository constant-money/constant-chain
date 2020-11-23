package f

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/incognitochain/incognito-chain/common/base58"
)

func Test(t *testing.T) {
	// paymentAddressStr := "12RwjWYppMHcMBbnteDEptK2vqnPbfMDQPVzyTLrHKv982XuidxyEjSdBc1KUTARFpmxa5JRXXZfsEbPDpDYGkausn4VmsBsM766Vxr"

	// keyWallet, err := wallet.Base58CheckDeserialize(paymentAddressStr)
	// if err != nil {
	// 	panic("aaaaaaaaa")
	// }
	// if len(keyWallet.KeySet.PaymentAddress.Pk) == 0 {
	// 	panic("aaaaaaaaa")
	// }
	// x:=[]string{}\
	x := []string{"12SX5qRcXmS99q1gHnhjmpriupZ8gqGsxuX1iY2MoHWz86pJnSs",
		"12dcPG5kFWg7MmUqQtg5CZt7sVtFzeUNf5CC1ebxUdbrEnVtPhX",
		"1RFZtKDxm852QsDA9vaHoaZTLT2S8eEUw8AQPqN4gEPakLMnBU",
		"12ePYYGMKyLAPQcEjXC2kQZmwmRRuL8iEQRDMm2zgX3vr2Wu8Z4",
		"1xQwSVGXkfdDwnyEMsZv2b26RxGgPrN2AfNCqobQctMHAimKZ5",
		"1UtGgcJSYRekkrZrT8dF3vgu7xYUaY275GTH6VNXbCgH6Loi8b",
		"12XfjYoNfxLQCJvaqaBwGGrr2oZzwKhMXM8ngDW7e3YFQxM2AEe",
		"1ywFQqCZNpcPDahY9y8vsHWDK9PuocvtmBmFfuKhGh81MK8HU3",
		"1u2ny9PVA7JbAYkPgUwy4Xt55xD5SXnw922TTGngLfAFdDmT2e",
		"121kYkUqFDBU5caBpQYbougpZE471rfXp1JG5GgzyhMVWwcc9MH",
		"12fa4uWmh22PVyvVUqMeGjcE21y39ZYvBZdW8e8vjaQ1Sf8UYJG",
		"12sbvqDiunKUxudBJnyr9Bge69v7mEU4A9X9LH4rDwpi5bBu4Xn",
		"1tpz5tieQ9S1zim3fzwUTEPesgo1RvbnoamGcbUWJeA9z8HEjv",
		"12qRW9mB3STWkHntQ15L3a4tsbx1ogxYaJJUSKyNoub37iJ5f7n",
		"1bAKJK3nYaGLqyjGtUjFrVde4CkfrMVNwS49JULbTxdbmgN8zA",
		"1DSxhHWv317a6iDQd9wPbTmbcuvJ6HcsgVuPhpUGKKQdwC5qWJ",
		"1rTsXUkuSVkenGwX6TdX4zvz2UcHiR648wgC5fNHct8bE5GuxG",
		"12dvUeid5ojJtBC8h78fX5Zqsx1RMktWeA8cXrVke3ZcxLKJQj8",
		"1FKBxXjbQwKCc25ViMWeRT37jGKCXFZWpyCfdPf297TFfc4bDS",
		"1pa9DCg2Ng5ydC6RnVAhydSGrA2DdXW1o9dHeh5AReRbgLnPQt",
		"18FBL1UvNvYscGahb8mua4NGGWSttLZqwuhmkyYWyP6j9hZJwo",
		"12je1EQKkUnuGci48KZLf4FcwgAbycDGUB6msbMLatmDAh8P2d9",
		"12uFtXehjgZXWvcVf258SQ53abpZEEi8oqjpAQoFnCoQJgonATV",
		"12DsLU8i2bqbfpBkMGmkauMNXGTrqC4vdi6ygWSKoeRnZQV2kd4",
		"12Z3vEVimoHhaJEVgCfpqgcPGrFYTVSU3xmotyBxkX98YcPBDYV",
		"12ADdmKNyRYzpMd1LTEyQvfC4GGsATVFwGiHiFRuGAj1juoBZPu",
		"12uzdkrCtNhYGcSkgFj3f6QybhvNAHeztiQmWJA3KkD9sU3Np8h",
		"12M2hz8QLZwiA9JVEHJLj5hwtvHcWN4oCS9a6v4v71cJpJ3PRk1",
		"12kUFuTdJJgpSbhcWvcCnrvZsUnfa9gqn42Z8bWwtvvFMJZCfxb",
		"1FkfznWNFr36mxjZWxNx22VKv2kVeRSzcHhmRq2aP7179YUBbS",
		"1oPfC2CXZxcfU5N1gyrgyGsr77cqbWNmkbnURhty9g5DSVNL7W",
		"12HZnDDaSggjCgJGJwkk8ZFpReC65SjK5b6qo9e9fZNBexPkfvM"}
	for _, pkstr := range x {
		pkByte, _, _ := base58.Base58Check{}.Decode(pkstr)

		shardID := pkByte[31] % 8
		fmt.Println(pkstr, ":", shardID)
	}

	// seed, _, _ := base58.Base58Check{}.Decode("12chUJ2Av5iR8xGVzby8rCtEA8kpMkR5XsPR1XTK6caLpWZzSvu")
	// x, _ := incognitokey.NewCommitteeKeyFromSeed(seed, keyWallet.KeySet.PaymentAddress.Pk)
	// fmt.Println(x.ToBase58())

}

func GetShardID(pkstr string) byte {
	pkByte, _, _ := base58.Base58Check{}.Decode(pkstr)
	return pkByte[31] % 8
}

func InterfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

type SState struct {
	Height     uint64
	Hash       string
	CrossShard string
}

func GetShardHeightConfirmedByBeaconHeight(bcHeight uint64) map[byte]uint64 {
	url := "http://51.83.237.20:9338/"
	sHeightMap := map[byte]uint64{}
	payload := strings.NewReader(fmt.Sprintf("{\n    \"id\": 1,\n    \"jsonrpc\": \"1.0\",\n    \"method\": \"retrievebeaconblockbyheight\",\n    \"params\": [\n    \t%v\n    \t\n    ]\n}", bcHeight))

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("cache-control", "no-cache")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	x := map[string]interface{}{}
	json.Unmarshal(body, &x)
	result := InterfaceSlice(x["Result"])[0]
	resultMap := result.(map[string]interface{})
	sState := resultMap["ShardStates"]
	sStateMap := sState.(map[string]interface{})
	for i := 0; i < 8; i++ {
		temp := InterfaceSlice(sStateMap[fmt.Sprintf("%v", i)])[0]
		// fmt.Println(temp.(map[string]interface{})["Height"])
		sHeightMap[byte(i)] = uint64(temp.(map[string]interface{})["Height"].(float64))
	}
	return sHeightMap
}

func GetBeaconOfShardHeight(sHeight uint64, sID byte) uint64 {
	url := "http://51.83.237.20:9338/"

	payload := strings.NewReader(fmt.Sprintf("{\n    \"id\": 1,\n    \"jsonrpc\": \"1.0\",\n    \"method\": \"retrieveblockbyheight\",\n    \"params\": [\n    \t%v,\n    \t%v,\n    \t\"1\"\n    ]\n}", sHeight, sID))

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("cache-control", "no-cache")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	x := map[string]interface{}{}
	json.Unmarshal(body, &x)
	result := InterfaceSlice(x["Result"])[0]
	resultMap := result.(map[string]interface{})
	return uint64(resultMap["BeaconHeight"].(float64))
}

func GetShardHeightConfirmBeaconHeight(bcHeight uint64) map[byte]uint64 {
	fmt.Println("Finding shard block height which confirmed in beacon block ", bcHeight, " ...")
	from := GetShardHeightConfirmedByBeaconHeight(bcHeight)
	fmt.Println("Got!")
	fmt.Println("Finding shard block height which confirm beacon block ", bcHeight)
	wanted := map[byte]uint64{}
	for k, v := range from {
		for i := v; ; i++ {
			bcH := GetBeaconOfShardHeight(i, k)
			if bcH >= bcHeight {
				wanted[k] = i
				break
			}
		}
	}
	fmt.Println("Got!")
	return wanted
}

func GetRewardOfCommitteeAtBlock(pk string, blkHeight uint64) uint64 {
	url := "http://51.83.237.20:9338/"
	// pk := "12bGNd9ofTJSbZYB2BXtaAQpsRV4a3KJ3xP5kLQmoxYBMaqhtXw"
	// blkHeight := uint64(3)
	payload := strings.NewReader(fmt.Sprintf("{  \n   \"jsonrpc\":\"1.0\",\n   \"method\":\"getrewardofincpubkeybyblock\",\n   \"params\":[\n   \t\t\"%v\",\n   \t\t%v\n   \t],\n   \"id\":1\n}", pk, blkHeight))

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("cache-control", "no-cache")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	x := map[string]map[string]float64{}
	json.Unmarshal(body, &x)
	result := x["Result"]
	return uint64(result["PRV"])
}

func GetCommitteeHasRWReceiverAtShard(cIDs []byte, rwShardAddress byte, bcHeight uint64) []string {
	url := "http://51.83.237.20:9338/"
	fmt.Printf("Get all committee at shard %v which has reward receiver in shard %v, at block beacon %v", cIDs, rwShardAddress, bcHeight)
	payload := strings.NewReader(fmt.Sprintf("{\n    \"id\": 1,\n    \"jsonrpc\": \"1.0\",\n    \"method\": \"getcommitteebybeaconheight\",\n    \"params\": [\n    \t%v\n    \t\n    ]\n}", bcHeight))

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("cache-control", "no-cache")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	x := map[string]map[string]interface{}{}
	json.Unmarshal(body, &x)
	result := x["Result"]["ShardRewardReceiver"]
	resultMap := result.(map[string]interface{})
	listPKWanted := []string{}
	for _, cID := range cIDs {
		listPK := InterfaceSlice(resultMap[fmt.Sprintf("%v", cID)])
		for _, v := range listPK {
			pk := v.(string)
			if GetShardID(pk) == rwShardAddress {
				listPKWanted = append(listPKWanted, pk)
			}
		}
	}
	return listPKWanted
}

func Test2(t *testing.T) {
	x := GetShardHeightConfirmBeaconHeight(665000)
	y := GetCommitteeHasRWReceiverAtShard([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 3, 665000)
	fmt.Println(x)
	for _, pk := range y {
		rw := GetRewardOfCommitteeAtBlock(pk, x[3])
		fmt.Println(rw)
	}
}
