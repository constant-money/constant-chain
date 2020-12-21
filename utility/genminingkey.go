package main

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/wallet"
)

func main() {
	privateKeys := []string{
		"112t8rpnK9Hq2EqZbGJpoS2t7rB3z46sFeRwogUjvzMYHhfBvB3B2X1Mx9W1jCahUZ9aXnbcmrhLXQJsjKDzMvX8vbsA8wjKDArxXfknsVy9",
		"112t8rq19Uu7UGbTApZzZwCAvVszAgRNAzHzr3p8Cu75jPH3h5AUtRXMKiqF3hw8NbEfeLcjtbpeUvJfw4tGj7pbqwDYngc8wB13Gf77o33f",
		"112t8rrEW3NPNgU8xzbeqE7cr4WTT8JvyaQqSZyczA5hBJVvpQMTBVqNfcCdzhvquWCHH11jHihZtgyJqbdWPhWYbmmsw5aV29WSXBEsgbVX",
		"112t8roHikeAFyuBpdCU76kXurEqrC9VYWyRyfFb6PwX6nip9KGYbwpXL78H92mUoWK2GWkA2WysgXbHqwSxnC6XCkmtxBVb3zJeCXgfcYyL",
		"112t8rr4sE2L8WzsVNEN9WsiGcMTDCmEH9TC1ZK8517cxURRFNoWoStYQTgqXpiAMU4gzmkmnWahHdGvQqFaY1JTVsn3nHfD5Ppgz8hQDiVC",
		"112t8rtt9Kd5LUcfXNmd7aMnQehCnKabArVB3BUk2RHVjeh88x5MJnJY4okB8JdFm4JNm4A2WjSe58qWNVkJPEFjpLHNYfKHpWfRdqyfDD9f",
	}
	for _, privateKey := range privateKeys {
		wl, err := wallet.Base58CheckDeserialize(privateKey)
		if err != nil {
			panic(err)
		}
		privateSeedBytes := common.HashB(common.HashB(wl.KeySet.PrivateKey))
		privateSeed := base58.Base58Check{}.Encode(privateSeedBytes, common.Base58Version)
		fmt.Println("MiningKey:", privateSeed)
	}
}
