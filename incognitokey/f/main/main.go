package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/incognitochain/incognito-chain/common/base58"
)

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

func GetCommitteeHasRWReceiverAtShard(cIDs []byte, rwShardAddress byte, bcHeight uint64) map[string]byte {
	url := "http://51.83.237.20:9338/"
	fmt.Printf("Get all committee at shard %v which has reward receiver in shard %v, at block beacon %v\n", cIDs, rwShardAddress, bcHeight-1)
	payload := strings.NewReader(fmt.Sprintf("{\n    \"id\": 1,\n    \"jsonrpc\": \"1.0\",\n    \"method\": \"getcommitteebybeaconheight\",\n    \"params\": [\n    \t%v\n    \t\n    ]\n}", bcHeight-1))

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("cache-control", "no-cache")

	res, _ := http.DefaultClient.Do(req)
	mapKeycID := map[string]byte{}
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
				mapKeycID[pk] = cID
			}
		}
	}
	if len(mapKeycID) != len(listPKWanted) {
		panic("eii")
	}
	for _, pk := range listPKWanted {
		if _, ok := mapKeycID[pk]; !ok {
			panic("ei")
		}
	}
	return mapKeycID
}

//map[0:522981 1:526864 2:524188 3:526448 4:525584 5:524936 6:525902 7:526031]
func main() {
	// x := GetShardHeightConfirmBeaconHeight(525001)
	// fmt.Println(x)
	// for i := byte(0); i < 8; i++ {
	// 	y := GetCommitteeHasRWReceiverAtShard([]byte{0, 1, 2, 3, 4, 5, 6, 7}, i, 525001)
	// 	for pk, cID := range y {
	// 		rw := GetRewardOfCommitteeAtBlock(pk, x[i])
	// 		rwdesc1 := GetRewardOfCommitteeAtBlock(pk, x[i]-1)
	// 		fmt.Println(pk, " in shard ", cID, ": ", rw, rwdesc1)
	// 	}
	// }

	y := []string{"1QLHG2Abw4ZXJK5dQKC5A3KZSj2C8ZMXP7cse5Mxqzgf8VxKPh",
		"1599KnMqdCRv4WFXC61Y6AG5gcpqj9j4DkK76K9cwFpxoSUJic",
		"1HDAzMtk4qeymRq35QARagf1855Jyr88qYvSTMgx9AAGqHYgH7",
		"12L45PBwnfAXR4R1hgNgvzyH5JSg91nqx9GGXETrgx6VbKzrCsk",
		"1RepTxcZETeecUTjYPm5vU2nsBhe7UgQogv77HQvodYZnSqFHs",
		"12W9LGz8VTkQ5vEkP9pM13n8GUduq4MzayYWBwShBkXb1pE4kCY",
		"1aVe8Msw3gZ3ZfXnDhjdKQvMpFo6hBnReonBbqT7ArVEGrszkU",
		"1ErG83mjPc7n4vWUioxBXaWq8EtGfDn4b6UC5avdDaJeGGKRkk",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12V27nQr5ResgdvZqTZJQLByiwP69GoJc5ZfaAnfZ7fdhkYVdBP",
		"12o9emgakf259RN5bVLQ4QZRz52ySWpUMG3A4Krwif3jh6n6XTH",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"11QfnhXccXHVPzNq3zGdEBRVt9DXz1sfXRoneEvwEmBgMsDLkt",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"19eCxN5KSCYwbgESukgR7YuofHLHM3bUkPKTgYpSnz3daqBAJL",
		"1epU2cpWkyWbTTgcXTEc3RpFDERybiCYXxeSQdzVstuYQi9qYv",
		"16VyQDT1cH373okAWKiHqeLrcyVf2jNL4jK8FzgPqeyM3m3B6J",
		"12Xn5fr5JQpjBL3hqjwLSkmvNQqsfQWKtJBsisG9UJJjCVbzYP4",
		"1oNhTNBtr3sLm3i6krWDkYUkRmrXqEtZMCqmK5Pv8uJomuJz7F",
		"12VEESneua3XRTgimSHB4PKonQtqnjhcoKztibCpzpd3cKfr2cy",
		"1sQLgTTex4r6jvcNBsyzW7iRvKjB6JWwvYGAGJ2kWFxFWh6KHq",
		"1LUADbLBh9jq2Q9sDnTp1Ysk7WW9wfbpSgA3iUDqtzoxPDhKNA",
		"11KyzW7ehjpNeGXpfFtFzJf5J4rgyHiwDTZsXv9B6MDMjCm4e",
		"1ozbr9V5haBZvhdiukiyGfMxUdnwiHK4smQVG1My2K8aGhvtu9",
		"12ZbHAcSBugaCmk7iJr7Yn8KUb5FyBmAho3m8YbcfsLpY5PFbCZ",
		"1FD9Z8oXEpAdhnZBFkVoCVXE18YNL73L3215T5rAb8pmDLUTQz",
		"12SA88WKyWLqoQByQLSBaqMANft14NyoamgkM7RU5jr3HWCfe2r",
		"123pt85RBYx6nEKEfJ1XhRD3K3cT31Ygs47WUkUcrRRAxM2bEvw",
		"12CrEhGcZ4rpcDAbcW6hATwN67UAZqce8sDmPcWDYChyhyhKnYW",
		"1ULT4bgzQhESqhre7GjJtkpn51P2DXBrfFRiorHLahY72pL2tH",
		"1iYTpEBbDtcmfwh6M3wXPtD1B8o2fGdJcxy1r578FgntDsRpH9",
		"12WgcHbKbmFDXfRxeu2PEkyd5pQp2LmQnTR5obJkc9h6G2H7gio",
		"1s79dJJxe4BNiq23qDMio1VyvhxbBr6GVzuo5qRjCf8AYyef6P",
		"1uGYFhZAb1idsv637pDuPtUCygPzAzHChcyv6UY2N2g2tAea3r",
		"12cfrM2S1nj3qdDN7PHd6VkwQ1F8H9u13ALmnC7vFNJehwgUT6X",
		"12d7Wzi8tuWREqYcn7upYAA82a7ETTC5P65Pvo2JiorjMy5jnrS",
		"1FmUUjuJXHu9S3iUjBJLs8Xr7zU4uAf4ScniNMXW9kMXxJD2sK",
		"12XddT6eUmz9aySjvNB8o5bfXoAH6eiP6WFTG8E9R9CGmPyFpoM",
		"12n9G3qN4JBsfS24rHeSQSMW9xmJfvqkevMw1Y28HjtVFhRPgk8",
		"12ZNbGJAK4YKgxZpqnKBs5GafZRbAc5vc2c6LqBGSU4VxPDigm6",
		"1cDDQYWGBP18hrrefsLreXbNgDfygzHeEv3gxiwH1zudmnkUx3",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1qpm6dT4WMq8hqdxUyfHScbbEyACLNeVyYGCHKUZnQr48RKsgG",
		"1ELE9X6HbaSQxPmJGgWTVdPE22ThD46M339k54kwAV9kkjwuUx",
		"12DhoH96MyMigzbNiMejydpvWRNTd8vBML93n3WfzSy7oEhkHjj",
		"1cRF7ybJ2Nro7LeCs71GP7VcR3j2RaCiFE26h2TPWRkMizr5iT",
		"1Yw1dgJrWX5FR1DMcgtHk1YZAfC53obHQmb7JXXd4Enw5YScDD",
		"128LabtuKfDMyX7s6T4ErUJnFCMq6UN3dy3UufKNiYq34dac9fW",
		"1TLSd2J1vAfMTzzCyLM1LBDF6eMGPg2tpx3wXfxBjJSAFzc8ks",
		"12XzWgrFYg5qCXoUGxVvLjycbkzBb1mSx36bk1LPzNBBLrCKKH3",
		"19dFU9jDCrgmUPmeFU6ekBY2GFTnmCCpqx7abwooxL5KK4LDAe",
		"1f9J5dVGKbo6oXJaxaC6faZvn8FWUbzb3MjzHj6K4heRpQn2dW",
		"12NxyyJREyNuzxL6Q1u8JVNqPPHBhPrnQzLGey8N251JrcCNixq",
		"12oHS7kfD3f8gA7Js4EX4yntkQeahpxLvwBACKy81hqU5GYPZJ4",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12RtBtjFwEUiMcHnqgAHTPk5fhsXZ74N2M7iQHqkJRcmFWQZBB8",
		"1K3QrgVbtGPDgedm4Dd9VVdouiZdDT7d81W219o4PobS12ZiTn",
		"12uBPEFCDeU6Zy9jFsaGPJ93KHPviKiq1zzsRsJ5X1fjasjYw8a",
		"12UJqf8C5Er2nxxES7wotZrgXH6UTM6FTqM865YcbDRENLok8wb",
		"15e6vjgSTDhPADSCn1vReLJswt4Pgqa8S3cuhtjWWSwhKRyC3a",
		"12sJXV5kqVSvYv9tccPyKwZo8WUbZ9LRCfxXFZe56WLLh1Esd9x",
		"12H8o4d1p1yHpPRSfRD35dxGpr6NArnDjNDzzrqM4bApqUddZR3",
		"1Y8wkb1UusXEx5PATHG8qTy8GVLo4EbMuu5ikaBiBZ4FNSabVt",
		"125jMD3nUhP67dVCeY2i44nigjD2JRz4xJhnCgJ67crKDFMMaAQ",
		"12nVyqCHSaPiq4ZeSAsBAZ3po2xcRrEvYC37K2S2RzZqYg3ho9N",
		"12evutT1nzujoagEjiE1bbEkgQpf9i8Mr4znYGgR1eT2t57w5w9",
		"1JdP4MKv7ocnW8DypjEvofGc5nwG7tUGvEeWgHhPqiimYuTn6N",
		"13U5bYPuW9eoNbHUKT3r3xY9NPVU6XjoS5QXVBuTVn6kyWceiH",
		"12M73TtZZPCX2Vja7fNWP4oZhEmrSXWeEfiRMrKwSSh4gYZuBtM",
		"12UrQwBzThYWFHexZH1QZL1qzUzfDWPb1HJAL3P5fZkTLxjLGxK",
		"1oCiscMVUBSmJKSLqKEyzNpcxCNv23LSzAMHnVt9EdxwzrRJhp",
		"17krX9PGgTj6NVsFiuiGd3u8czgwVyNhVnVS7bVAFWs2uYTUkC",
		"12XVjJo8rhZEPPgvBGqdDhLLs8WW6NDbPdeXtquciVDrx9Hd3Zb",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12q7tH1VfDMMcDNMDoHAqJySJALwfRGiRjtYAj7dRxh9184TFHj",
		"12ppJuoow9LKXc3iiTNV3pWPJ1M6pE5LRjLKAgXFcMKvzSuVXkT",
		"1VLkjmtHTECmtj9QTXGBK1rZcWbAxhP5wr7L3hwcyUVcAAfhQQ",
		"1o3Xzscwg5211n47iGtzBmEppY7cinK1Aa2rUUESGEkxXba7db",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1Ts1ZPGTFBBNhdw8787txDFrgLfW1jGKqd1KFuWzsQSxo3VArK",
		"12c5Ea6GZ48umVzqycYNfoMvYfQNKPv338dSWpmhgedHoxGYNiQ",
		"1276bbG2R43uQVBcrjMwknc5pmAkbbcihtjynLmNgAzcP7BYQM4",
		"1pF1W7P1Jeyt3PNpMQ1ZS9nvdtyqKU63pJcwEruZZyGnowhfcC",
		"1dwiTwN5Eqx5jCk63nG4gkqJqDXAiET2GrwnfxZGGu6rTY7Jqm",
		"1MgQTn28QgN7DPwu26NN43NAroKL3WU2gzHsXLnYzZVb8ga9NG",
		"12zei2jAYgg4yRNmbcdkbotYGHtwNACz4AKRk16UE8BCcmtyqf",
		"1Kb113Ht356f5kD6KK8zeXGh7L9CuDdSt9fCAbdxQnJRVGiqUk",
		"12Fao7QSWCQdNgjX81q7GY8H6LtEZ9BaKJrvxJkCV47rNKDJ5rv",
		"12tu2sgfmgSkYaFq2LRjpodkg7ZRWwfA9LXzeML2o8Gu6uwG5Jb",
		"12mrQkEVdBxUVT7b2afKggJkmTRQ6UAgtnJVthxy478bkPuLgzn",
		"12BwnzNGhwgtMGsyrpBxpqYA4DCWZoyKCVaURGSNcYxV5WrMFPd",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1p4h2jK7zNs9RPoi6nMoPQWoewAJzCosDU3zrYaiZLvLtxAvDt",
		"12veDZ7XAjjyy9wTS6EToWoM2ZRPqXBcBX6mXjH62bh5QBAHpr",
		"12WpsPgTBPRvb121DwWdrJGzohZvAy1Ng7p2DyRn9vBH4EM5rQx",
		"12p1Ercdc41GLuiAk6nuP2y7JMMnrBo2g39XvnXwdDqoKDGwXQp",
		"1yGA52wd7RTgP8UbXWnqYPg33cLPdNN4Rp9EgnjPLRmArUZG1C",
		"1eZBvdXQB8BYF36DXQnsov1SoJ2NJuerC8asY6dhR68fxoLZxf",
		"12bXvnQB9KU4Pmp1a7ouyyC8ht5udG7hLibswaVCyTesccwcxte",
		"1uMbRh7eUncRt4CAAsQuKpoKmepe7UztTwi6S4zFV5CpnKbGUA",
		"1rKkEVzrEV2DXSE5Sf5jtM27N6AmruH5Lznu92en2Np99HSz6B",
		"15XYrdNQuzRSQqEkDV6iw1qG4b4ZZTep81dfC76H8azpdRmgTL",
		"12LeHTp4BvREQPq2d1FcnkT11znyZmfbRSFPYzggUvtApLfUwmu",
		"12wKBxNbTkb5dvYxX6odCHHDFVD51B8Bv6k5GhTu4LnasnpXWTU",
		"1uid8eef1NeQGKLMJa6aUEYqJsNcwVrQbYBvst2y7Scod2pdaB",
		"1qWpj1NRBn8mrSWrLHUqK5VdNgx49ygMXNFRCBQTdYfPiCzV4W",
		"12wU1kxZ34eNjxHrabZYQZUiw9hX9UkPQn2x1xJfFhYXsWS6BiP",
		"1qgtVd7BwgfaeouxopPEMiYKeneYGr67zd22teEq97uMxYVsVs",
		"1gKUR2u6EEgxrXBhQugmWaRpam63F6JXfJduS1K98J6wGc8R9a",
		"12traJnxJp25DGksQ2xAo5rxVo2LmXWmXBwQQCiQmpPLZizB5NX",
		"1MpA6WycX6YVQVsUcAbuWzYw3HRJbjxQ2mzmqRYzxPHZLh6rdZ",
		"128jWAmotWLNP1JDHXohimWGyrQiAWLse8wQYhvcbKQXuZZ6ik7",
		"12KWcgkupLGHbuVx5NV9Av2DC1dYSe45Yct1CjAER5ajmb59rfV",
		"12bPyCp9kSPFxTNL6tM6VzpjHqT3ZrmPvUDcWH9wGTfbDWuAkF3",
		"1RCbRFVKz3qr2FpQab7i7itgPqjFAcTSDdJdFCRrgHPqBioWhL",
		"12JRe6PSUhwh8dA5Ss7HRGsWr7Jxr1MgyUNTvngYdCJLDbcU7Wn",
		"1ae5GW1jiq3naxBWRc1wb7CCtZ7Cs8CtFFxtujhTSP4Y6hqA8f",
		"1j7J3htqupv57JhPMaNdAXgmPQkZLYjZkB4cfviLYFjH2kiJ31",
		"12Mp2mftBcV26XZnph2F2BAksiin2SHT1P3VNt9W8uT4rhN1aQA",
		"1HnSuZ5FP3oSSpyKca9QZsVjoxKzSGQYq95DV6P9Ctu53YDkQ7",
		"12uMvmgGQPzANw1ZS9emTbHwwxScWRprrFoUuh1DarSRWQvNMHJ",
		"12r8sHn8AhtKDbi4F7k3Nt2j1pMMdS7QcN2MfS76tYJDimfjKZU",
		"126xw17FFuoYgck8KBp7GPSmoXSsyXxyMnbaTxcrpuJex9fk6VR",
		"12An6dacpbMSgqr2UQNW73Qz5ZXxqEf626UX1NPdaVBGmJtCAKg",
		"1suYnT3hvE8yuyMe6LTCVsaSNmMAQzTz8AR5iRX4LC3mPs9fo5",
		"1QNA3kdFfzPmmBBDz9iGDyFEarmNN9yWTcHMEJ7V6tSc8i9h2z",
		"12qruCio9FF8YdKuz5dAoU1ySJxxr1HZFozydSXhHEWkhJQG3xu",
		"1L4PAvZZ77u5uS3NCaRtaEUDZpku9v7DGhecV7LTU2eN1idKji",
		"12etq43pAw7KokBq5Qd7WLiPqsXfL3FhD3LwVRYVq9WuPWiQEp6",
		"122NDmtXNf9G1fKg6exwHshSYqGVw37BqYqpvMVjjKvL7HSB6GS",
		"1jizkDHqiUeMpX1Dt7BdXxAyiCFNWGJyaXFfYzrSqg9kpNp6Pu",
		"1Xt51o46FNpW8ThiW1q1HECfx1Rtgwvo4gx3rbuCMFtycNQw6Y",
		"1iVu1D9y2uvQn88cfQ2qVmB9rLC6J1X3iGuwRF9b8dbgJ8DDEW",
		"1Ayi4wyaJpcN1yFSUFnA4zxWxS1qfUu7BZ22pvioNVKF4ZR4sa",
		"1Eik6aiZebdNXeBin8nz2yst46xPK7TsDd9Kx1VU5NtHZnz9MV",
		"12oGVRBgoFNxFvUuCCvBm751YUNf12A1dSioEWghS2AE5nWpUne",
		"123XPPs8riJmJ3Fgzn2mUXQsFi1nZ3sBJHJp5kYCZTMN4hA9Bio",
		"12ifez5Ju8gyHx1frCYzJoE2H3JvpniJux3QnbNJvbYLKzSD6oM",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12HnkyCcjeg8qjNSCt2pYZYAseExLm1gm5FUC3vryX8LbS3EGmQ",
		"127isWyceCUHYSYTEua3jeLfzwLR19z8aEJLaCiA3i5WAGtP6pK",
		"1WdNm63iKBi4W52PQfqHnmhZSsqxh7LLiMZuj2kshndJh439a6",
		"1YRBdMMwBCNDJvktGbR7uL8fv5RkUuBJ9daseYid81JzvQBbj5",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1JYweUeBiRyJgwDYqfdhKF1h9M6S5F8HVAvZ5Qgq77PZsxfPtz",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"124ULqAP665rgpK73hiUxWNvxPABrRkcFx3tupcwMuY1gVbjyUr",
		"1NDiyYYg4XGxwtmeKVXMUULNzrRH7o8Q4CZ1nR3Vs2P7GSVm6R",
		"126ak1ZeKcsNTBVE5MUAU1ArsvQCnn63uUepreJ7mUphM3qZ3vN",
		"1cmUzpNW7vrxzhft41yRfvzpaZedm7W9Zir4zwCkDBHx468HXD",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1UCeh6VHcQUmLfiKj7Fw9FLsoDeP3vyvUpP5JttfuUg2KAqPfR",
		"1xVYcbQZMHVxq1HqAGhczUhiKmox3oMBsZu2XHj2pUdZiGjVn",
		"141D9D1xqPoc5s3NUgPhFSdCnvZzmCamEy9k29wgFy8JA4SqF8",
		"12jfJ1fBrG55ETNYf9onKMPBZGiKRWbir8JStoc5PuXkAH2a4QB",
		"1xEb3RW4TJG3C25AgJHHAktmzZauFpCBFQkXUNEGZ7AUovc7q2",
		"1m9G7tqMP7Dq1XU8GL5Uy9CCWc6cKRDBtiLefddahszNjDQnEd",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12jnLdbdQfUmBqTcm1GQTg4J4iDDajMHoVX5MsK1o3Qwpd9WChC",
		"1rTsXUkuSVkenGwX6TdX4zvz2UcHiR648wgC5fNHct8bE5GuxG",
		"18FBL1UvNvYscGahb8mua4NGGWSttLZqwuhmkyYWyP6j9hZJwo",
		"1TLgR3wLLrNRiRo2dsZHudWBBkPmJuUPPK8v3ZZ8v1GcPEWQUw",
		"12SX5qRcXmS99q1gHnhjmpriupZ8gqGsxuX1iY2MoHWz86pJnSs",
		"12dvUeid5ojJtBC8h78fX5Zqsx1RMktWeA8cXrVke3ZcxLKJQj8",
		"1u2ny9PVA7JbAYkPgUwy4Xt55xD5SXnw922TTGngLfAFdDmT2e",
		"12dcPG5kFWg7MmUqQtg5CZt7sVtFzeUNf5CC1ebxUdbrEnVtPhX",
		"1tpz5tieQ9S1zim3fzwUTEPesgo1RvbnoamGcbUWJeA9z8HEjv",
		"1DSxhHWv317a6iDQd9wPbTmbcuvJ6HcsgVuPhpUGKKQdwC5qWJ",
		"12je1EQKkUnuGci48KZLf4FcwgAbycDGUB6msbMLatmDAh8P2d9",
		"121kYkUqFDBU5caBpQYbougpZE471rfXp1JG5GgzyhMVWwcc9MH",
		"12XfjYoNfxLQCJvaqaBwGGrr2oZzwKhMXM8ngDW7e3YFQxM2AEe",
		"1bAKJK3nYaGLqyjGtUjFrVde4CkfrMVNwS49JULbTxdbmgN8zA",
		"1LYz2CMYe1LAsWmHQJ7SDSuo4AFwgT7VadZXpnUsNFxKtgT6UC",
		"1xQwSVGXkfdDwnyEMsZv2b26RxGgPrN2AfNCqobQctMHAimKZ5",
		"1Ns9NCpgnkGHYXawvVbspegVf4oJwi23GpvsyfeAgLcTH6DBtK",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1UtGgcJSYRekkrZrT8dF3vgu7xYUaY275GTH6VNXbCgH6Loi8b",
		"1pruDCKuZcxSzhg3ZF2WgfgGsehAeGmcPrvFKmt92sU68RXzat",
		"12cEUVRAnko6PKwvcax7pvdpo4tGsme2QhyiP3pBVGrcd7zkFLX",
		"1RFZtKDxm852QsDA9vaHoaZTLT2S8eEUw8AQPqN4gEPakLMnBU",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12ePYYGMKyLAPQcEjXC2kQZmwmRRuL8iEQRDMm2zgX3vr2Wu8Z4",
		"1FKBxXjbQwKCc25ViMWeRT37jGKCXFZWpyCfdPf297TFfc4bDS",
		"12YWBJB4zMQUZVffQDwziqcCYRyV56RfB8UCTmrjfKm7BXymgXn",
		"12fa4uWmh22PVyvVUqMeGjcE21y39ZYvBZdW8e8vjaQ1Sf8UYJG",
		"12sbvqDiunKUxudBJnyr9Bge69v7mEU4A9X9LH4rDwpi5bBu4Xn",
		"12qRW9mB3STWkHntQ15L3a4tsbx1ogxYaJJUSKyNoub37iJ5f7n",
		"19gfefXn3bL9UMqirP8huX4GHxfQd3ZZ3Jeop17h3qk7F2AZcr",
		"1pa9DCg2Ng5ydC6RnVAhydSGrA2DdXW1o9dHeh5AReRbgLnPQt",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1ywFQqCZNpcPDahY9y8vsHWDK9PuocvtmBmFfuKhGh81MK8HU3",
		"1kqdGQHNq9WK5T952EQFn9stiqVdVYLt3xkrTCbCH8jPmQdxbK",
		"1A2HTTKiTk1bgArBZyorEAiNVe6bQaLrfW1RPX5cUZNfVKb5d5",
		"12XW4rCjiUWwiWPNNUTpSBBaUFVBijxudMqVL9vpFa4PLG4e3Qu",
		"15oqSpvVy5JPh7iM5HedXektYcyTdV1at4gmgfRKt3mKgyHvv4",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1xEjJTS79ewX7rVRY449QTE5T2B9buyc11bVr1N7GEzPBYyUd8",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"1dEwWTJsxKbbLeit5AAwhQJ6PWbDH5MpHt4SjJwZFVWe5wzLU5",
		"1vTBWA5JE8Bzg561oJmMw9x58F6rrV4ddZP73pqsHNufFNzWzS",
		"12ZL74X7FSXAkNQSaKZqx4o2ReNJRVqCP8isy9BdecDi38Mw9s7",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12ZTav4H4ts8QVu4ZBhNaNZ2kH48LfkGmGa7YNBV2FkoWT5GcTp",
		"1VVQRGZFhMyp9LRS1yjyXsW6ugMqtHaYyUYMLutRhy6ASF7AWf",
		"1tdUrk467sd6XoXKVmks8YrN41NUPHYyYDNtmHw77vSqycJRGJ",
		"12CP1EUirQzB1mKBVRRg3EEkVXMnjwoJXYu6CQaKCDNshgsTkwZ",
		"12iLjLUeVvuX2ZTMaYU9qXpgou7VCZtDQCSqvG9aSRDTGuAc71L",
		"12VUQiPzvrK4GxmRVE95yhY1uyvMW1efz4GhRbnDJYHYPquXNDk",
		"12DByVJjzXFHBee3Y7vfQkvJUGTejcN56iatdn7HQJpMEThyg8S",
		"12pH2dwoe8tu2QfEMqQznN5sS9mHStkHbwTBEH57Rc783vFkZDC",
		"1LLTdxGb3WPeSKNanni3hSTbEoatqcJ2DiShC4uupB25HngKPv",
		"12P15ufNRZLH76h3vZzPrG39EDtfVzjBZW7DDwSUWgKN7KriANh",
		"1Fzo4FNMR7rynrSc2nhjN61cNdkFybLaEjTdZ4TLocMLutEbVz",
		"12rgotB8bcyz4NBnJAEtQidVHRpSR4HWHj5Dz6UKaDbgxApM1hB",
		"13LqNJdqKuY5dP8rpizsyzSc1JeGiGNgUV1Y8rQS1hF7rsRndd",
		"121dxsK2J8YUKYiAfXbeyYe2dZcfpKG91VQzebNGz4wavfkBvh4",
		"1HEHiury7NMbZmxYknC7vqhrcfDcbyg8RN24Jsx2V6oa1uRgwX",
		"1fEUcV3prCWCDNN3n4XiXuV1CzKztWsTFcLjA71djYTRAtLzam",
		"1kA1eY3Sgh7DukKcq9QowBiN4GbqDicCzMdw4U6Z1yQ83UX7Kf",
		"12go14vBsHtsg3GYSMTodx6rKqD3UctwmHXP3gXFgXZkEa7LinL",
		"12xb69A9cEYux4XCfeP14Sk3o9rg7SzPjCUssDwECHfuC9Dzhd",
		"12RRMGAEXdtD796sbmyHwNv9oiPPfGYa1r6KssMqXyiUMfT6r86",
		"12FHD5peCB31bQBoafNdtuB9cwE2gPFPdWh4V3jzu2DbNuHM16H",
		"1dqnbSReTz2kXJF1EdMKkRp25F7EJfGeKZDDBtrHRQJKSKNFZD",
		"1trtyu4v5FWBRBaqV7kMuorgpkiBfzLUQkiemjZBNxXCeuKkx2",
		"1ywRMuTaVWYvquNYurGzE9BkPwxiuUyZQXqgPBTwezhABJxAzJ",
		"12fDumXbpbKVz8V7dav3Wbt4VETXkoQL4XNCCZdH8h8fDHeLNrQ",
		"1QtXuDDBNc99s93sZXvnn6rLkLJ6fa19U7L8nSLUiw8xgch2Pz",
		"1mST43Vh61PGBvFCf95KAijQrrJfi6mUyXC7ZhEa6dGzKGqcis",
		"12ZZ63UnVfbAQZjV9mDDTuskNeLnkEVMd53kppdzuSonQK28bEZ",
		"1Cw25jN2UYGtGnkcEtwKKP8wfBaVSUHRL87YiCu6otjV1CPJJh",
		"12jGF6Bm2aEPgiJYqs5J5edCJANXx8NFtzodVvFKQrwfCiqUBqf",
		"12bFpGRCT4ZggDSqiWLvk9XsB6v4Ji7XnUrStaPChZajoqHutgy",
		"1d2ySoHT4jACLzEFJCWWGnxDdmWCHBbgwdVzfmAvGyUneMuZAp",
		"12Phpcuu2SjFCSREM94RSNE6WF6JFKbc4HM96K8tK4fF1v75cuR",
		"12UkBTpH6Z4KyjkSrSycgsk59QiTgHRPa3jq66tj3KURMtxhLB6",
		"12fiNT7RBmdHXdP23vQQhYkwa7zH3y3Xp4tWEFM9qp4eVKjYt7S",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"14rV6pxTfCL2AFXM3s3LBJCMLePHuH8fvUyCiGFzQNFaFeJjBG",
		"1Z696kvzYDUGRgxpx2P57WASUWp211hHEcsSqGQJRC8TXcsAJd",
		"12FqxWxiKvJF98FBU99TErJnsoU15YrAcogRYDBpAGbJCriKFAt",
		"14cNecHmi2JEos4LdgurPpQ6fnSmg4Qx4ExVUQ6f5KkYCZxXBF",
		"1bddjTGbpUPRirk9UzrsjvsFSottXc7G7pM4icV6f2ULofv7mX",
		"18cRtNAUrEcrKQiUf1jmWMHcTEMmdzjrUuw4nnfWdVEKNi8kJ3",
		"12ue31WRXrEeSUi2AKQTadcKkyLeyBz4nZmn7a29Jq9HcNSycRb",
		"12U74gXJWMkgpGE88F5neua3aFJ8Jwh2Q9znxbXcZ9qf5mhWmga",
		"12V6nSFmjBzkJRuhuCE9E15Rai6EZjndnvJdNZkkw1o5XtsYhdB",
		"1kAjJ7mmiN43LoncuxP2dY5xDCFSVMNeEeNpXp61ZbXdp1QA3z",
		"1md1tdscbkEWNbNdDtUmC5DTYz6iCKuTkCnkVd1uuB4qVTxptY",
		"12UPXzxUiR6LTugSZsynLYJvNKQsqwS6QAgUJSCSR3ccj2mrqrF",
		"12pQYYu484HA18YgtKCT3SgbVtLMr5PFpR5uV17DYoM7hD9BaD9",
		"1GAFrTFSEDDDG6tkSMTa1fXb1C9aj5UAGvBVBofyGz9sN53xn9",
		"12gnm6v3vo2Xo4696kp2pTVXf8Sxu69TiCzHwRLoUgyK4JEFFs1",
		"1Vig3QQL3SkAJHsCCZ8231frgNgb6xnrfSgACx8USagsvJk9CX"}
	for _, pk := range y {
		pkByte, _, _ := base58.Base58Check{}.Decode(pk)
		if pkByte[31]%8 == 3 {
			rw := GetRewardOfCommitteeAtBlock(pk, 526448)
			rwdesc1 := GetRewardOfCommitteeAtBlock(pk, 526448-1)
			fmt.Println(pk, " in shard ", 3, ": ", rw, rwdesc1)
		}
	}
}
