package btcrelaying

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/blockcypher/gobcy"
)

func setGenesisBlockToChainParams(networkName string, genesisBlkHeight int) (*chaincfg.Params, error) {
	blk, err := buildBTCBlockFromCypher(networkName, genesisBlkHeight)
	if err != nil {
		return nil, err
	}

	// chainParams := chaincfg.MainNetParams
	chainParams := chaincfg.TestNet3Params
	chainParams.GenesisBlock = blk.MsgBlock()
	chainParams.GenesisHash = blk.Hash()
	return &chainParams, nil
}

func tearDownRelayBTCHeadersTest(dbName string) {
	fmt.Println("Tearing down RelayBTCHeadersTest...")
	dbPath := filepath.Join(testDbRoot, dbName)
	os.RemoveAll(dbPath)
	os.RemoveAll(testDbRoot)
}

func getAllTxsFromCypherBlock(blockHeight int) (string, []string, error) {
	bc := getBlockCypherAPI("main")
	cypherBlock1, err := bc.GetBlock(
		blockHeight,
		"",
		map[string]string{
			"txstart": "0",
			"limit":   "500",
		},
	)
	if err != nil {
		return "", []string{}, err
	}
	cypherBlock2, err := bc.GetBlock(
		blockHeight,
		"",
		map[string]string{
			"txstart": "500",
			"limit":   "1000",
		},
	)
	if err != nil {
		return "", []string{}, err
	}
	txIDs := append(cypherBlock1.TXids, cypherBlock2.TXids...)
	return cypherBlock2.Hash, txIDs, nil
}

func TestRelayBTCHeaders(t *testing.T) {
	networkName := "test3"
	genesisBlockHeight := int(1719640)

	chainParams, err := setGenesisBlockToChainParams(networkName, genesisBlockHeight)
	if err != nil {
		t.Errorf("Could not set genesis block to chain params with err: %v", err)
		return
	}
	dbName := "btc-blocks-test"
	btcChain1, err := GetChainV2(dbName, chainParams, int32(genesisBlockHeight))
	defer tearDownRelayBTCHeadersTest(dbName)
	if err != nil {
		t.Errorf("Could not get chain instance with err: %v", err)
		return
	}

	for i := genesisBlockHeight + 1; i <= genesisBlockHeight+10; i++ {
		blk, err := buildBTCBlockFromCypher(networkName, i)
		if err != nil {
			t.Errorf("buildBTCBlockFromCypher fail on block %v: %v\n", i, err)
			return
		}
		isMainChain, isOrphan, err := btcChain1.ProcessBlockV2(blk, BFNone)
		if err != nil {
			t.Errorf("ProcessBlock fail on block %v: %v\n", i, err)
			return
		}
		if isOrphan {
			t.Errorf("ProcessBlock incorrectly returned block %v "+
				"is an orphan\n", i)
			return
		}
		fmt.Printf("Block %s (%d) is on main chain: %t\n", blk.Hash(), blk.Height(), isMainChain)
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("Session 1: best block hash %s and block height %d\n", btcChain1.BestSnapshot().Hash.String(), btcChain1.BestSnapshot().Height)
	btcChain1.db.Close()

	hh, _ := json.Marshal(btcChain1.BestSnapshot())
	fmt.Println("best state: ", string(hh))
	return

	// simulate new session
	btcChain2, err := GetChainV2(dbName, chainParams, int32(genesisBlockHeight))
	if err != nil {
		t.Errorf("Could not get chain instance (for session 2) with err: %v", err)
		return
	}
	fmt.Printf("Session 2: best block hash %s and block height %d\n", btcChain2.BestSnapshot().Hash.String(), btcChain2.BestSnapshot().Height)

	if btcChain2.BestSnapshot().Hash != btcChain1.BestSnapshot().Hash ||
		btcChain2.BestSnapshot().Height != btcChain1.BestSnapshot().Height {
		t.Errorf("Best states of session 1 & 2 are different")
		return
	}

	txID := "8bae12b5f4c088d940733dcd1455efc6a3a69cf9340e17a981286d3778615684"
	msgTx := buildMsgTxFromCypher(txID, "main")

	blockHash, txIDs, err := getAllTxsFromCypherBlock(308570)
	if err != nil {
		t.Errorf("Could not get cypher block by height with err: %v", err)
		return
	}
	txHashes := make([]*chainhash.Hash, len(txIDs))
	for i := 0; i < len(txIDs); i++ {
		txHashes[i], _ = chainhash.NewHashFromStr(txIDs[i])
	}

	txHash := msgTx.TxHash()
	blkHash, _ := chainhash.NewHashFromStr(blockHash)
	merkleProofs := buildMerkleProof(txHashes, &txHash)
	btcProof := BTCProof{
		MerkleProofs: merkleProofs,
		BTCTx:        msgTx,
		BlockHash:    blkHash,
	}
	btcProofBytes, _ := json.Marshal(btcProof)
	btcProofStr := base64.StdEncoding.EncodeToString(btcProofBytes)
	decodedProof, err := ParseBTCProofFromB64EncodeStr(btcProofStr)
	if err != nil {
		t.Errorf("Could not parse btc proof from base64 string with err: %v", err)
		return
	}

	isValid, err := btcChain2.VerifyTxWithMerkleProofs(decodedProof)
	if err != nil {
		t.Errorf("Could not verify tx with merkle proofs with err: %v", err)
		return
	}
	if !isValid {
		t.Error("Failed to verify tx with merkle proofs")
		return
	}
	msg, err := ExtractAttachedMsgFromTx(decodedProof.BTCTx)
	if err != nil {
		t.Errorf("Could not extract attached message from tx with err: %v", err)
		return
	}
	if msg != "charley loves heidi" {
		t.Errorf("Expect attached message is %s but got %s", "charley loves heidi", msg)
		return
	}

	addrStr, err := btcChain2.ExtractPaymentAddrStrFromPkScript(decodedProof.BTCTx.TxOut[1].PkScript)
	if err != nil {
		t.Errorf("Could not extract payment address from tx with err: %v", err)
		return
	}
	if addrStr != "1HnhWpkMHMjgt167kvgcPyurMmsCQ2WPgg" {
		t.Errorf("Expect payment address is %s but got %s", "1HnhWpkMHMjgt167kvgcPyurMmsCQ2WPgg", addrStr)
		return
	}
}

func TestBuildBTCBlockFromTestNetCypher(t *testing.T) {
	bc := gobcy.API{"029727206f7e4c8fb19301e4629c5793", "bcy", "test"}
	cypherBlock, err := bc.GetBlock(2820650, "", nil)
	if err != nil {
		return
	}
	prevBlkHash, _ := chainhash.NewHashFromStr(cypherBlock.PrevBlock)
	merkleRoot, _ := chainhash.NewHashFromStr(cypherBlock.MerkleRoot)
	msgBlk := wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    int32(cypherBlock.Ver),
			PrevBlock:  *prevBlkHash,
			MerkleRoot: *merkleRoot,
			Timestamp:  cypherBlock.Time,
			Bits:       uint32(cypherBlock.Bits),
			Nonce:      uint32(cypherBlock.Nonce),
		},
		Transactions: []*wire.MsgTx{},
	}

	if err != nil {
		t.Errorf("Could not build btc block from cypher - with err: %v", err)
		return
	}
	unixTs := msgBlk.Header.Timestamp.Unix()
	fmt.Println("unixTs: ", unixTs)
}

func TestBuildBTCBlockFromCypher(t *testing.T) {
	// blk, err := buildBTCBlockFromCypher("main", 623600)
	blk, err := buildBTCBlockFromCypher("test3", 1720520)
	if err != nil {
		t.Errorf("Could not build btc block from cypher - with err: %v", err)
		return
	}
	unixTs := blk.MsgBlock().Header.Timestamp.Unix()
	fmt.Println("unixTs: ", unixTs)
	if unixTs != 1585564707 {
		t.Errorf("Wrong timestamp: expected %d, got %d", 1585564707, unixTs)
		return
	}
	ts := time.Unix(unixTs, 0)
	if ts.UnixNano() != blk.MsgBlock().Header.Timestamp.UnixNano() {
		t.Error("Convertion from unix timestamp to Time is not correct")
	}
}

func TestGenerateAndFaucetBTCAddresses(t *testing.T) {
	bc := getBlockCypherAPI("test3")

	// bc := gobcy.API{"029727206f7e4c8fb19301e4629c5793", "bcy", "test"}
	addrKeys1, err := bc.GenAddrKeychain()
	if err != nil {
		t.Errorf("Could not generate btc address by using cypher api - with err: %v", err)
		return
	}
	fmt.Printf("Account 1: %+v\n", addrKeys1)

	addrKeys2, err := bc.GenAddrKeychain()
	if err != nil {
		t.Errorf("Could not generate btc address by using cypher api - with err: %v", err)
		return
	}
	fmt.Printf("Account 2: %+v\n", addrKeys2)

	//Fund it with faucet
	txhash, err := bc.Faucet(addrKeys1, 100000)
	if err != nil {
		t.Errorf("Could not do faucet by using cypher api - with err: %v", err)
		return
	}
	fmt.Printf("Address: %v, Faucet TXHash: %v\n", addrKeys1.Address, txhash)
}

func TestGetBTCTxFromCypher(t *testing.T) {
	bc := gobcy.API{"029727206f7e4c8fb19301e4629c5793", "bcy", "test"}
	cypherTx, _ := bc.GetTX("84ac285019633b12020cb20e67a50a4f8f63c71448a13d24438f15e877989d12", nil)
	txB, _ := json.Marshal(cypherTx)
	fmt.Printf("Tx: %s\n", string(txB))
}

func TestCreateAndSendBTCTxToCypher(t *testing.T) {
	// bc := gobcy.API{"029727206f7e4c8fb19301e4629c5793", "bcy", "test"}
	bc := getBlockCypherAPI("test3")

	inAddr := "msTYtu7nsMiwFUtNgCSQBk26JeBf9q3GTM"
	outAddr := "2MvpFqydTR43TT4emMD84Mzhgd8F6dCow1X"
	amount := int(400)
	trans := gobcy.TX{}
	trans.Fees = int(5000)
	trans.Inputs = make([]gobcy.TXInput, 1)
	trans.Inputs[0].Addresses = make([]string, 1)
	trans.Inputs[0].Addresses[0] = inAddr

	trans.Outputs = make([]gobcy.TXOutput, 2)
	trans.Outputs[0].ScriptType = "null-data"
	uniquePortingID := "batch2"
	script := []byte{
		txscript.OP_RETURN,
		byte(len(uniquePortingID)),
	}

	//msg := HashAndEncodeBase58(uniquePortingID)
	fmt.Println("memo: ", uniquePortingID)
	script = append(script, []byte(uniquePortingID)...)
	trans.Outputs[0].Script = hex.EncodeToString(script)

	trans.Outputs[1].Addresses = make([]string, 1)
	trans.Outputs[1].Addresses[0] = outAddr
	trans.Outputs[1].Value = amount

	// bc := getBlockCypherAPI("test3")
	skelTx, err := bc.NewTX(trans, false)
	if err != nil {
		t.Errorf("Could not init btc tx by using cypher api - with err: %v", err)
		return
	}
	wifDecoded, err := btcutil.DecodeWIF("cSoBn9ayvDT1WsxmArCxq2Deot5sYswSqtxu4UTuAbSTHBsDdTCm")
	if err != nil {
		t.Errorf("Could not decode wif - with err: %v", err)
		return
	}
	privateKeys := []string{wifDecoded.PrivKey.D.Text(16)}
	err = skelTx.Sign(privateKeys)
	if err != nil {
		t.Errorf("Could not sign btc tx by using cypher api - with err: %v", err)
		return
	}

	tx, err := bc.SendTX(skelTx)
	if err != nil {
		t.Errorf("Could not send btc tx by using cypher api - with err: %v", err)
		return
	}
	bb, _ := json.Marshal(tx)
	fmt.Println("Newly created tx: ", string(bb))
}

func TestCreateAndSendBTCTxToCypherForRedeem(t *testing.T) {
	// bc := gobcy.API{"029727206f7e4c8fb19301e4629c5793", "bcy", "test"}
	bc := getBlockCypherAPI("test3")

	inAddr := "mgLFmRTFRakf5zs23YHB4Pcd8JF7TWCy6E"
	outAddr := "n4UtqQiW3qYjtiEUMscwPoBtUuYAK1AqKJ"
	amount := int(100)
	trans := gobcy.TX{}
	trans.Fees = int(5000)
	trans.Inputs = make([]gobcy.TXInput, 1)
	trans.Inputs[0].Addresses = make([]string, 1)
	trans.Inputs[0].Addresses[0] = inAddr

	trans.Outputs = make([]gobcy.TXOutput, 2)
	trans.Outputs[0].ScriptType = "null-data"
	script := []byte{
		txscript.OP_RETURN,
		0x10,
	}

	rawMsg := fmt.Sprintf("%s%s", "btcredeem1", "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci")
	msg := HashAndEncodeBase58(rawMsg)
	fmt.Println("hashed msg: ", msg)
	script = append(script, []byte(msg)...)
	trans.Outputs[0].Script = hex.EncodeToString(script)

	trans.Outputs[1].Addresses = make([]string, 1)
	trans.Outputs[1].Addresses[0] = outAddr
	trans.Outputs[1].Value = amount

	// bc := getBlockCypherAPI("test3")
	skelTx, err := bc.NewTX(trans, false)
	if err != nil {
		t.Errorf("Could not init btc tx by using cypher api - with err: %v", err)
		return
	}

	privateKeys := []string{"d50ae4f47e5f7f3746c13e8eb843a0004ac734ba88b0d5ad0f3e510df8f07345"}
	err = skelTx.Sign(privateKeys)
	if err != nil {
		t.Errorf("Could not sign btc tx by using cypher api - with err: %v", err)
		return
	}

	tx, err := bc.SendTX(skelTx)
	if err != nil {
		t.Errorf("Could not send btc tx by using cypher api - with err: %v", err)
		return
	}
	bb, _ := json.Marshal(tx)
	fmt.Println("Newly created tx: ", string(bb))
}

func TestBuildBTCMerkleProof(t *testing.T) {
	bc := getBlockCypherAPI("test3")
	txID := "4478038c54fe1ea19668afc5f088861152cb35559f90ee39024b393a21a612cb"
	msgTx := buildMsgTxFromCypher(txID, "test3")

	cypherBlock, err := bc.GetBlock(
		1696574,
		"",
		map[string]string{
			"txstart": "0",
			"limit":   "500",
		},
	)

	if err != nil {
		t.Errorf("Could not get cypher block by height with err: %v", err)
		return
	}

	txIDs := cypherBlock.TXids
	txHashes := make([]*chainhash.Hash, len(txIDs))
	for i := 0; i < len(txIDs); i++ {
		txHashes[i], _ = chainhash.NewHashFromStr(txIDs[i])
	}

	txHash := msgTx.TxHash()
	blkHash, _ := chainhash.NewHashFromStr(cypherBlock.Hash)
	merkleProofs := buildMerkleProof(txHashes, &txHash)
	btcProof := BTCProof{
		MerkleProofs: merkleProofs,
		BTCTx:        msgTx,
		BlockHash:    blkHash,
	}
	btcProofBytes, _ := json.Marshal(btcProof)
	btcProofStr := base64.StdEncoding.EncodeToString(btcProofBytes)
	fmt.Println("btcProofStr: ", btcProofStr)

	// verify btc proof
	decodedProof, err := ParseBTCProofFromB64EncodeStr(btcProofStr)
	if err != nil {
		t.Errorf("Could not parse btc proof from base64 string with err: %v", err)
		return
	}
	merkleRoot, _ := chainhash.NewHashFromStr(cypherBlock.MerkleRoot)
	isValid := verify(merkleRoot, decodedProof.MerkleProofs, &txHash)
	if !isValid {
		t.Error("Failed to verify merkle proofs")
	}
}
