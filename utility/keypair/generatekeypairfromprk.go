package main

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wallet"
)

func main() {
	wl, _ := wallet.Base58CheckDeserialize("112t8rrEW3NPNgU8xzbeqE7cr4WTT8JvyaQqSZyczA5hBJVvpQMTBVqNfcCdzhvquWCHH11jHihZtgyJqbdWPhWYbmmsw5aV29WSXBEsgbVX")
	//privKeyB58 := wl.Base58CheckSerialize(wallet.PriKeyType)
	wl.KeySet.InitFromPrivateKey(&wl.KeySet.PrivateKey)
	//readOnlyKeyB58 := wl.Base58CheckSerialize(wallet.ReadonlyKeyType)
	paymentAddressB58 := wl.Base58CheckSerialize(wallet.PaymentAddressType)
	//shardID := common.GetShardIDFromLastByte(wl.KeySet.PaymentAddress.Pk[len(wl.KeySet.PaymentAddress.Pk)-1])
	miningSeed := base58.Base58Check{}.Encode(common.HashB(common.HashB(wl.KeySet.PrivateKey)), common.ZeroByte)
	publicKey := base58.Base58Check{}.Encode(wl.KeySet.PaymentAddress.Pk, common.ZeroByte)
	committeeKey, _ := incognitokey.NewCommitteeKeyFromSeed(common.HashB(common.HashB(wl.KeySet.PrivateKey)), wl.KeySet.PaymentAddress.Pk)
	res, _ := incognitokey.CommitteeKeyListToString([]incognitokey.CommitteePublicKey{committeeKey})
	fmt.Println("Payment Address", paymentAddressB58)
	fmt.Println("Public Key", publicKey)
	fmt.Println("Mining Seed", miningSeed)
	fmt.Println("Committee Public Key", res)
	fmt.Println("ShardID", wl.KeySet.PaymentAddress.Pk[len(wl.KeySet.PaymentAddress.Pk)-1])
	fmt.Println("-----------------------")
	str := base58.Base58Check{}.Encode([]byte{}, common.Base58Version)
	fmt.Println(str)
	fmt.Println(base58.Base58Check{}.Decode(str))
}
