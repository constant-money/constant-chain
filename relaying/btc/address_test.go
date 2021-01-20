package btcrelaying

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

func setGenesisBlockToChainParamsByNetwork(
	networkName string,
	genesisBlkHeight int,
	chainParams *chaincfg.Params,
) (*chaincfg.Params, error) {
	blk, err := buildBTCBlockFromCypher(networkName, genesisBlkHeight)
	if err != nil {
		return nil, err
	}

	// chainParams := chaincfg.MainNetParams
	// chainParams := chaincfg.TestNet3Params
	chainParams.GenesisBlock = blk.MsgBlock()
	chainParams.GenesisHash = blk.Hash()
	return chainParams, nil
}

func initBTCHeaderTestNetChain(t *testing.T) *BlockChain {
	networkName := "test3"
	genesisBlockHeight := int(1746520)

	chainParams, err := setGenesisBlockToChainParamsByNetwork(networkName, genesisBlockHeight, &chaincfg.TestNet3Params)
	if err != nil {
		t.Errorf("Could not set genesis block to chain params with err: %v", err)
		return nil
	}
	dbName := "btc-blocks-testnet"
	btcChain, err := GetChainV2(dbName, chainParams, int32(genesisBlockHeight))
	if err != nil {
		t.Errorf("Could not get chain instance with err: %v", err)
		return nil
	}
	return btcChain
}

func initBTCHeaderMainNetChain(t *testing.T) *BlockChain {
	networkName := "main"
	genesisBlockHeight := int(632061)

	chainParams, err := setGenesisBlockToChainParamsByNetwork(networkName, genesisBlockHeight, &chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Could not set genesis block to chain params with err: %v", err)
		return nil
	}
	dbName := "btc-blocks-mainnet"
	btcChain, err := GetChainV2(dbName, chainParams, int32(genesisBlockHeight))
	if err != nil {
		t.Errorf("Could not get chain instance with err: %v", err)
		return nil
	}
	return btcChain
}

func TestDecodeInvalidBTCTestNetAddress(t *testing.T) {
	btcChain := initBTCHeaderTestNetChain(t)
	if btcChain == nil {
		t.Error("BTC chain instance should not be null")
		return
	}
	// an address on mainnet
	testAddrStr := "1A7tWftaGHohhGcJMVkkm4zAYnF53KjRnU"
	params := btcChain.GetChainParams()
	_, err := btcutil.DecodeAddress(testAddrStr, params)
	if err == nil {
		t.Error("Expected returned error is not null, but got null")
	}
}

func TestDecodeValidBTCTestNetAddress(t *testing.T) {
	btcChain := initBTCHeaderTestNetChain(t)
	if btcChain == nil {
		t.Errorf("BTC chain instance should not be null")
		return
	}
	// an address on testnet
	testAddrStr := "mgLFmRTFRakf5zs23YHB4Pcd8JF7TWCy6E"
	params := btcChain.GetChainParams()
	_, err := btcutil.DecodeAddress(testAddrStr, params)
	if err != nil {
		t.Errorf("Expected returned error is null, but got %v\n", err)
	}
}

func TestDecodeInvalidBTCMainNetAddress(t *testing.T) {
	btcChain := initBTCHeaderMainNetChain(t)
	if btcChain == nil {
		t.Error("BTC chain instance should not be null")
		return
	}
	// an address on testnet
	testAddrStr := "mgLFmRTFRakf5zs23YHB4Pcd8JF7TWCy6E"
	params := btcChain.GetChainParams()
	_, err := btcutil.DecodeAddress(testAddrStr, params)
	if err == nil {
		t.Error("Expected returned error is not null, but got null")
	}
}

func TestDecodeValidBTCMainNetAddress(t *testing.T) {
	btcChain := initBTCHeaderMainNetChain(t)
	if btcChain == nil {
		t.Error("BTC chain instance should not be null")
		return
	}
	// an address on mainnet
	testAddrStr := "bc1qq7ndvtvyzcea44ps6d4nt3plk02ghpsha0t55y"
	params := btcChain.GetChainParams()
	_, err := btcutil.DecodeAddress(testAddrStr, params)
	if err != nil {
		t.Errorf("Expected returned error is null, but got %v\n", err)
	}
}

func TestBTCMainnetAddress(t *testing.T) {
	type BTCMainnetAddressTestCases struct {
		address string
		isValid bool
	}
	testcases := []BTCMainnetAddressTestCases{
		{"bc1qq7ndvtvyzcea44ps6d4nt3plk02ghpsha0t55y", true},                      // AddressWitnessPubKeyHash
		{"1KN7N34ZUd1HyXgqcJpeGrooQcLf2L4xFC", true},                              // AddressPubKeyHash
		{"3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQX", true},                              // AddressScriptHash
		{"bc1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3qccfmv3", true},  // AddressWitnessScriptHash
		{"37ExSZhkPhSwmzdjbeznK529vvrgS3qsJW", true},                              // legacy - AddressScriptHash
		{"3EtBoGNHBd1zCH2A5WTExJrizB7TiBw4ci", true},                              // p2sh-segwit -- AddressScriptHash
		{"bc1qpx5p30dcfxqpz5sxemv30ky34lf20jwe7nl95exqcphjvxxehalql9mmrd", true},  // bech32 -- AddressWitnessScriptHash
		{"tb1qwtlr3cmn0kg3h6passf7wktmy7596p7swpmxdz6nsp6pmvhzg3eq93qvfz", false}, // bech32 -- AddressWitnessScriptHash - testnet
	}

	btcChain := initBTCHeaderMainNetChain(t)
	if btcChain == nil {
		t.Error("BTC chain instance should not be null")
		return
	}
	params := btcChain.GetChainParams()

	var pkScript []byte
	var addrs []btcutil.Address
	var isRightNet bool
	for _, tc := range testcases {
		actualResult := true
		// decode address from string to bytes array
		btcAddress, err := btcutil.DecodeAddress(tc.address, params)
		if err != nil {
			actualResult = false
			t.Logf("Can not decode btc address %v - Error %v", tc.address, err)
			goto checkResult
		}
		// check right network
		isRightNet = btcAddress.IsForNet(params)
		if !isRightNet {
			actualResult = false
			t.Logf("Invalid network btc address %v", tc.address)
			goto checkResult
		}
		// convert btcAddress to pkScript
		pkScript, err = txscript.PayToAddrScript(btcAddress)
		if err != nil {
			actualResult = false
			t.Logf("Can not convert btc address %v to pkScript - Error %v", tc.address, err)
			goto checkResult
		}

		// extract pkscript to address
		_, addrs, _, err = txscript.ExtractPkScriptAddrs(pkScript, params)
		if err != nil || len(addrs) == 0 {
			actualResult = false
			t.Logf("Can not extract btc address %v - Error %v", tc.address, err)
			goto checkResult
		} else {
			if tc.address != addrs[0].EncodeAddress() {
				actualResult = false
				t.Logf("Different btc address before %v - after %v", tc.address, addrs[0].EncodeAddress())
				goto checkResult
			}
		}

	checkResult:
		assert.Equal(t, tc.isValid, actualResult)
	}
}

func TestBTCTestnetAddress(t *testing.T) {
	type BTCTestnetAddressTestCases struct {
		address string
		isValid bool
	}
	testcases := []BTCTestnetAddressTestCases{
		{"tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx", true},                      // AddressWitnessPubKeyHash
		{"mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn", true},                              // AddressPubKeyHash
		{"2MzQwSSnBHWHqSAqtTVQ6v47XtaisrJa1Vc", true},                             // AddressScriptHash
		{"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sl5k7", true},  // AddressWitnessScriptHash
		{"2Mx7sVozbZZXPiqsTRWLnZ7bC7vGUEEwX6g", true},                             // legacy - AddressScriptHash
		{"2MuiiTCHGtQ3MMFhAQ3kFGsJ6N9K89itPcw", true},                             // p2sh-segwit -- AddressScriptHash
		{"tb1qwtlr3cmn0kg3h6passf7wktmy7596p7swpmxdz6nsp6pmvhzg3eq93qvfz", true},  // bech32 -- AddressWitnessScriptHash
		{"bc1qpx5p30dcfxqpz5sxemv30ky34lf20jwe7nl95exqcphjvxxehalql9mmrd", false}, // bech32 -- AddressWitnessScriptHash - mainnet
	}

	btcChain := initBTCHeaderTestNetChain(t)
	if btcChain == nil {
		t.Error("BTC chain instance should not be null")
		return
	}
	params := btcChain.GetChainParams()

	var pkScript []byte
	var addrs []btcutil.Address
	var isRightNet bool
	for _, tc := range testcases {
		actualResult := true
		// decode address from string to bytes array
		btcAddress, err := btcutil.DecodeAddress(tc.address, params)
		if err != nil {
			actualResult = false
			t.Logf("Can not decode btc address %v - Error %v", tc.address, err)
			goto checkResult
		}
		// check right network
		isRightNet = btcAddress.IsForNet(params)
		if !isRightNet {
			actualResult = false
			t.Logf("Invalid network btc address %v", tc.address)
			goto checkResult
		}
		// convert btcAddress to pkScript
		pkScript, err = txscript.PayToAddrScript(btcAddress)
		if err != nil {
			actualResult = false
			t.Logf("Can not convert btc address %v to pkScript - Error %v", tc.address, err)
			goto checkResult
		}

		// extract pkscript to address
		_, addrs, _, err = txscript.ExtractPkScriptAddrs(pkScript, params)
		if err != nil || len(addrs) == 0 {
			actualResult = false
			t.Logf("Can not extract btc address %v - Error %v", tc.address, err)
			goto checkResult
		} else {
			if tc.address != addrs[0].EncodeAddress() {
				actualResult = false
				t.Logf("Different btc address before %v - after %v", tc.address, addrs[0].EncodeAddress())
				goto checkResult
			}
		}

	checkResult:
		assert.Equal(t, tc.isValid, actualResult)
	}
}

func TestBTCMultiSigRawTx(t *testing.T) {
	incKey1 := "112t8rnXRDT21fsx5UYR1kGd8yjiygUS3tXfhcRfXy2nmJS3U39vkf76wbQsXguwhHwN2EtBF4YZJ8o1i7MMF9BsKngcgxfkCZBa5P3Fq9xp"
	incKey2 := "112t8rnXLn4sD7rP98ALejLKoTJm4N9uwavXE2m98h9hcMN9fC7Lp1MRoZJ4G4aXd3ShpaaSzna8s3V8xkvDaKHBBD9mzx9ToDHk6gz5nTnf"
	incKey3 := "112t8rnXQoMqs6hcf36qSzyypndZFxRSPQEjoq8AqV7DBM14TY66CdVRTxHwaMSTQ4XCBPEXF5zfwE4wEPSqeD1MaacWa9DwYNcxr16bMFSR"
	incKey4 := "112t8rnXMK3U2VNDaHhxLx9FrS75wVq5YupVk99YenYTYGU2KXJg4iR1j7KDGesi7ju1btmELbPqxtMni1gNHUp6HmYTapBd6Bq4WcjvqdoG"
	incKey5 := "112t8rnXL7kKwhkeZurgPAyJqjwrsWLWHMXzmFJS5XimqFFMxtv94ZiQ9YhriGdNvNA7JQckVoMjfrkzhzSoeRdNZAhRJKNfXbreZ22yBzLB"
	incKey6 := "112t8rnXUAuUsefZK35qVWrtvQpZn9RqX9LSLT5XxGyvNF5VyKrjAZXy7ZZg1qrC1v18j81p5ckDukMFgpPVxSeLopKwCw7KoUWkYPWuPXJ5"
	incKey7 := "112t8rnXbYudqWAujTBJMFjf4DtEK7Hs8vVsSM7588svyrs1mYjFMYEwpc7roxdZLbZUmjwiru85q19die3dhUvFePuccEmUWkbfCNXeU2vU"

	redeemScript, bitcoinKeys, err := BuildMultiSigP2SHAddr([]string{incKey1, incKey2, incKey3, incKey4, incKey5, incKey6, incKey7}, 5)
	require.Equal(t, err, nil)
	multiAddress := btcutil.Hash160(redeemScript)

	// if using Bitcoin main net then pass &chaincfg.MainNetParams as second argument
	addr, err := btcutil.NewAddressScriptHashFromHash(multiAddress, &chaincfg.TestNet3Params)
	require.Equal(t, err, nil)
	fmt.Println(addr)
	utxos := []*Outputs{
		{
			txHash: "48175851f91decc5afaa86e014d0fb64905de6c4f872cf3de14415a7bac09be4",
			index:  1,
		},
	}
	recievers := []*Receiver{
		{
			to:     "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X",
			amount: 95000,
		},
	}

	hexSignedTx, err := SpendMultiSig(utxos, recievers, bitcoinKeys, redeemScript)
	require.Equal(t, err, nil)
	fmt.Println(hexSignedTx)
}

func BuildMultiSigP2SHAddr(incPrvStrs []string, required int) ([]byte, []*btcec.PrivateKey, error) {
	if len(incPrvStrs) < required || required < 0 {
		return nil, nil, errors.New("Invalid signature requirment")
	}
	bitcoinPrvs := make([]*btcec.PrivateKey, 0)
	// create redeem script for 2 of 3 multi-sig
	builder := txscript.NewScriptBuilder()
	// add the minimum number of needed signatures
	builder.AddOp(byte(txscript.OP_1 - 1 + required))
	for _, v := range incPrvStrs {
		keyWallet, err := wallet.Base58CheckDeserialize(v)
		if err != nil {
			return nil, nil, err
		}
		IncKeyBytes := keyWallet.KeySet.PrivateKey
		BTCKeyBytes := GenBTCPrivateKey(IncKeyBytes)
		prvBtc, pubKey := btcec.PrivKeyFromBytes(ethcrypto.S256(), BTCKeyBytes)
		pk := pubKey.SerializeCompressed()
		// add the 3 public key
		builder.AddData(pk)
		bitcoinPrvs = append(bitcoinPrvs, prvBtc)
	}
	// add the total number of public keys in the multi-sig screipt
	builder.AddOp(byte(txscript.OP_1 - 1 + len(incPrvStrs)))
	// add the check-multi-sig op-code
	builder.AddOp(txscript.OP_CHECKMULTISIG)
	// redeem script is the script program in the format of []byte
	redeemScript, err := builder.Script()
	if err != nil {
		return nil, nil, err
	}

	return redeemScript, bitcoinPrvs, nil
}

type Receiver struct {
	amount int64
	to     string
}

type Outputs struct {
	txHash string
	index  uint32
}

func SpendMultiSig(utxos []*Outputs, recievers []*Receiver, beaconKeys []*btcec.PrivateKey, redeemScript []byte) (string, error) {
	// thanks to: https://medium.com/coinmonks

	redeemTx := wire.NewMsgTx(wire.TxVersion)
	for _, v := range utxos {
		utxoHash, err := chainhash.NewHashFromStr(v.txHash)
		if err != nil {
			return "", nil
		}
		outPoint := wire.NewOutPoint(utxoHash, v.index)
		txIn := wire.NewTxIn(outPoint, nil, nil)
		redeemTx.AddTxIn(txIn)
	}

	for _, v := range recievers {
		// adding the output to tx
		decodedAddr, err := btcutil.DecodeAddress(v.to, &chaincfg.TestNet3Params)
		if err != nil {
			return "", err
		}
		destinationAddrByte, err := txscript.PayToAddrScript(decodedAddr)
		if err != nil {
			return "", err
		}

		// adding the destination address and the amount to the transaction
		redeemTxOut := wire.NewTxOut(v.amount, destinationAddrByte)
		redeemTx.AddTxOut(redeemTxOut)
	}

	for i := range redeemTx.TxIn {
		// signing the tx
		signature := txscript.NewScriptBuilder()
		signature.AddOp(txscript.OP_FALSE)
		for _, v := range beaconKeys {
			sig, err := txscript.RawTxInSignature(redeemTx, i, redeemScript, txscript.SigHashAll, v)
			if err != nil {
				return "", err
			}
			signature.AddData(sig)
		}
		signature.AddData(redeemScript)
		signatureScript, err := signature.Script()
		if err != nil {
			// Handle the error.
			return "", err
		}
		redeemTx.TxIn[i].SignatureScript = signatureScript
	}

	var signedTx bytes.Buffer
	err := redeemTx.Serialize(&signedTx)
	if err != nil {
		// Handle the error.
		return "", err
	}

	hexSignedTx := hex.EncodeToString(signedTx.Bytes())

	return hexSignedTx, nil
}

func GenBTCPrivateKey(IncKeyBytes []byte) []byte {
	BTCKeyBytes := ed25519.NewKeyFromSeed(IncKeyBytes)[32:]
	return BTCKeyBytes
}
