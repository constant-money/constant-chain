package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy/coin"
	"github.com/incognitochain/incognito-chain/transaction"
	"sort"
)

type ShardBlock struct {
	ValidationData string `json:"ValidationData"`
	Body           ShardBody
	Header         ShardHeader
}

type ShardHeader struct {
	Producer              string                 `json:"Producer"`
	ProducerPubKeyStr     string                 `json:"ProducerPubKeyStr"`
	ShardID               byte                   `json:"ShardID"`               // shard ID which block belong to
	Version               int                    `json:"Version"`               // version of block structure
	PreviousBlockHash     common.Hash            `json:"PreviousBlockHash"`     // previous block hash or Parent block hash
	Height                uint64                 `json:"Height"`                // block height
	Round                 int                    `json:"Round"`                 // bpft consensus round
	Epoch                 uint64                 `json:"Epoch"`                 // epoch of block (according to current beacon height)
	CrossShardBitMap      []byte                 `json:"CrossShardBitMap"`      // crossShards bitmap for beacon
	BeaconHeight          uint64                 `json:"BeaconHeight"`          // beacon check point height
	BeaconHash            common.Hash            `json:"BeaconHash"`            // beacon check point hash
	TotalTxsFee           map[common.Hash]uint64 `json:"TotalTxsFee"`           // fee of all txs in block
	ConsensusType         string                 `json:"ConsensusType"`         // consensus type, by which this block is produced
	Timestamp             int64                  `json:"Timestamp"`             // timestamp of block
	TxRoot                common.Hash            `json:"TxRoot"`                // Transaction root created from transaction in shard
	ShardTxRoot           common.Hash            `json:"ShardTxRoot"`           // output root created for other shard
	CrossTransactionRoot  common.Hash            `json:"CrossTransactionRoot"`  // transaction root created from transaction of micro shard to shard block (from other shard)
	InstructionsRoot      common.Hash            `json:"InstructionsRoot"`      // actions root created from Instructions and Metadata of transaction
	CommitteeRoot         common.Hash            `json:"CommitteeRoot"`         // hash from public key list of all committees designated to create this block
	PendingValidatorRoot  common.Hash            `json:"PendingValidatorRoot"`  // hash from public key list of all pending validators designated to this ShardID
	StakingTxRoot         common.Hash            `json:"StakingTxRoot"`         // hash from staking transaction map in shard best state
	InstructionMerkleRoot common.Hash            `json:"InstructionMerkleRoot"` // Merkle root of all instructions (using Keccak256 hash func) to relay to Ethreum
	// This obsoletes InstructionMerkleRoot but for simplicity, we keep it for now

	//for version 2
	Proposer    string
	ProposeTime int64
}

type ShardBody struct {
	Instructions      [][]string
	CrossTransactions map[byte][]CrossTransaction //CrossOutputCoin from all other shard
	Transactions      []metadata.Transaction
}

type CrossShardBlock struct {
	ValidationData  string `json:"ValidationData"`
	Header          ShardHeader
	ToShardID       byte
	MerklePathShard []common.Hash
	// Cross Shard data for PRV
	CrossOutputCoin []coin.Coin
	// Cross Shard For Custom token privacy
	CrossTxTokenPrivacyData []ContentCrossShardTokenPrivacyData
}

func NewShardBlock() *ShardBlock {
	return &ShardBlock{
		Header: ShardHeader{},
		Body: ShardBody{
			Instructions:      [][]string{},
			CrossTransactions: make(map[byte][]CrossTransaction),
			Transactions:      make([]metadata.Transaction, 0),
		},
	}
}
func NewShardBlockWithHeader(header ShardHeader) *ShardBlock {
	return &ShardBlock{
		Header: header,
		Body: ShardBody{
			Instructions:      [][]string{},
			CrossTransactions: make(map[byte][]CrossTransaction),
			Transactions:      make([]metadata.Transaction, 0),
		},
	}
}
func NewShardBlockWithBody(body ShardBody) *ShardBlock {
	return &ShardBlock{
		Header: ShardHeader{},
		Body:   body,
	}
}
func NewShardBlockFull(header ShardHeader, body ShardBody) *ShardBlock {
	return &ShardBlock{
		Header: header,
		Body:   body,
	}
}

func (shardBlock *ShardBlock) GetProposer() string {
	return shardBlock.Header.Proposer
}

func (shardBlock *ShardBlock) GetProposeTime() int64 {
	return shardBlock.Header.ProposeTime
}

func (shardBlock *ShardBlock) GetProduceTime() int64 {
	return shardBlock.Header.Timestamp
}

func (shardBlock *ShardBlock) GetShardID() int {
	return int(shardBlock.Header.ShardID)
}

func (shardBlock *ShardBlock) GetPrevHash() common.Hash {
	return shardBlock.Header.PreviousBlockHash
}

func (shardBlock *ShardBlock) BuildShardBlockBody(instructions [][]string, crossTransaction map[byte][]CrossTransaction, transactions []metadata.Transaction) {
	shardBlock.Body.Instructions = append(shardBlock.Body.Instructions, instructions...)
	shardBlock.Body.CrossTransactions = crossTransaction
	shardBlock.Body.Transactions = append(shardBlock.Body.Transactions, transactions...)
}

func (crossShardBlock *CrossShardBlock) UnmarshalJSON(data []byte) error {
	type Alias CrossShardBlock
	temp := &struct {
		CrossOutputCoin []json.RawMessage
		*Alias
	}{
		Alias: (*Alias)(crossShardBlock),
	}

	if err := json.Unmarshal(data, temp); err != nil {
		Logger.log.Error("UnmarshalJSON crossShardBlock", string(data))
		return err
	}

	outputCoinList, err := coin.ParseCoinsFromBytes(temp.CrossOutputCoin)
	if err != nil {
		Logger.log.Error("UnmarshalJSON Cannot parse crossOutputCoins", err)
		return err
	}
	for i := 0; i < len(temp.CrossOutputCoin); i++ {
		fmt.Println("Detail of CrossOutputCoins", outputCoinList[i])
	}
	crossShardBlock.CrossOutputCoin = outputCoinList
	return nil
}

func (crossShardBlock CrossShardBlock) GetProposer() string {
	return crossShardBlock.Header.Proposer
}

func (crossShardBlock CrossShardBlock) GetProposeTime() int64 {
	return crossShardBlock.Header.ProposeTime
}

func (crossShardBlock CrossShardBlock) GetProduceTime() int64 {
	return crossShardBlock.Header.Timestamp
}

func (crossShardBlock CrossShardBlock) GetCurrentEpoch() uint64 {
	return crossShardBlock.Header.Epoch
}

func (crossShardBlock CrossShardBlock) GetPrevHash() common.Hash {
	return crossShardBlock.Header.PreviousBlockHash
}

func (crossShardBlock *CrossShardBlock) Hash() *common.Hash {
	hash := crossShardBlock.Header.Hash()
	return &hash
}

func (shardBlock ShardBlock) Hash() *common.Hash {
	hash := shardBlock.Header.Hash()
	return &hash
}
func (shardBlock *ShardBlock) validateSanityData() (bool, error) {
	//Check Header
	if shardBlock.Header.Height == 1 && len(shardBlock.ValidationData) != 0 {
		return false, NewBlockChainError(ShardBlockSanityError, errors.New("Expect Shard Block with Height 1 to not have validationData"))
	}
	// producer address must have 66 bytes: 33-byte public key, 33-byte transmission key
	if shardBlock.Header.Height > 1 && len(shardBlock.ValidationData) == 0 {
		return false, NewBlockChainError(ShardBlockSanityError, errors.New("Expect Shard Block to have validationData"))
	}
	if int(shardBlock.Header.ShardID) < 0 || int(shardBlock.Header.ShardID) > 256 {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block ShardID in range 0 - 255 but get %+v ", shardBlock.Header.ShardID))
	}
	if shardBlock.Header.Version < SHARD_BLOCK_VERSION {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Version greater or equal than %+v but get %+v ", SHARD_BLOCK_VERSION, shardBlock.Header.Version))
	}
	if len(shardBlock.Header.PreviousBlockHash[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Previous Hash in the right format"))
	}
	if shardBlock.Header.Height < 1 {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Height to be greater than 0"))
	}
	if shardBlock.Header.Height == 1 && !shardBlock.Header.PreviousBlockHash.IsEqual(&common.Hash{}) {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block with Height 1 (first block) have Zero Hash Value"))
	}
	if shardBlock.Header.Height > 1 && shardBlock.Header.PreviousBlockHash.IsEqual(&common.Hash{}) {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block with Height greater than 1 have Non-Zero Hash Value"))
	}
	if shardBlock.Header.Round < 1 {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Round greater or equal than 1"))
	}
	if shardBlock.Header.Epoch < 1 {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Epoch greater or equal than 1"))
	}
	if shardBlock.Header.Timestamp <= 0 {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Time greater than 0"))
	}
	if len(shardBlock.Header.TxRoot[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Tx Root in the right format"))
	}
	if len(shardBlock.Header.ShardTxRoot[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Shard Tx Root in the right format"))
	}
	if len(shardBlock.Header.CrossTransactionRoot[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Cross Transaction Root in the right format"))
	}
	if len(shardBlock.Header.InstructionsRoot[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Instructions Root in the right format"))
	}
	if len(shardBlock.Header.CommitteeRoot[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Committee Root in the right format"))
	}
	if shardBlock.Header.Height == 1 && !shardBlock.Header.CommitteeRoot.IsEqual(&common.Hash{}) {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block with Height 1 have Zero Hash Value"))
	}
	if shardBlock.Header.Height > 1 && shardBlock.Header.CommitteeRoot.IsEqual(&common.Hash{}) {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block with Height greater than 1 have Non-Zero Hash Value"))
	}
	if len(shardBlock.Header.PendingValidatorRoot[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Committee Root in the right format"))
	}
	if len(shardBlock.Header.StakingTxRoot[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Staking Tx Root in the right format"))
	}
	if len(shardBlock.Header.CrossShardBitMap) > 254 {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Cross Shard Length Less Than 255"))
	}
	if shardBlock.Header.BeaconHeight < 1 {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block has Beacon Height greater or equal than 1"))
	}
	//if shardBlock.Header.BeaconHeight == 1 && !shardBlock.Header.BeaconHash.IsPointEqual(&common.Hash{}) {
	//	return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block with Beacon Height 1 have Zero Hash Value"))
	//}
	if shardBlock.Header.BeaconHeight > 1 && shardBlock.Header.BeaconHash.IsEqual(&common.Hash{}) {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block with Beacon Height greater or equal than 1 have Non-Zero Hash Value"))
	}
	if shardBlock.Header.TotalTxsFee == nil {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Total Txs Fee have nil value"))
	}
	if len(shardBlock.Header.InstructionMerkleRoot[:]) != common.HashSize {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Instruction Merkle Root in the right format"))
	}
	// body
	if shardBlock.Body.Instructions == nil {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Instruction is not nil"))
	}
	if len(shardBlock.Body.Instructions) != 0 && shardBlock.Header.InstructionMerkleRoot.IsEqual(&common.Hash{}) {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Instruction Merkle Root have Non-Zero Hash Value because Instrucstion List is not empty"))
	}
	if shardBlock.Body.CrossTransactions == nil {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Cross Transactions Map is not nil"))
	}
	if len(shardBlock.Body.CrossTransactions) != 0 && shardBlock.Header.CrossTransactionRoot.IsEqual(&common.Hash{}) {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Cross Transaction Root have Non-Zero Hash Value because Cross Transaction List is not empty"))
	}
	if shardBlock.Body.Transactions == nil {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Transactions is not nil"))
	}
	if len(shardBlock.Body.Transactions) != 0 && shardBlock.Header.TxRoot.IsEqual(&common.Hash{}) {
		return false, NewBlockChainError(ShardBlockSanityError, fmt.Errorf("Expect Shard Block Tx Root have Non-Zero Hash Value because Transactions List is not empty"))
	}
	return true, nil
}

func (shardBlock *ShardBlock) UnmarshalJSON(data []byte) error {
	tempShardBlock := &struct {
		ValidationData string `json:"ValidationData"`
		Header         ShardHeader
		Body           *json.RawMessage
	}{}
	err := json.Unmarshal(data, &tempShardBlock)
	if err != nil {
		return NewBlockChainError(UnmashallJsonShardBlockError, err)
	}
	shardBlock.ValidationData = tempShardBlock.ValidationData

	blkBody := ShardBody{}
	err = blkBody.UnmarshalJSON(*tempShardBlock.Body)
	if err != nil {
		return NewBlockChainError(UnmashallJsonShardBlockError, err)
	}
	shardBlock.Header = tempShardBlock.Header
	if shardBlock.Body.Transactions == nil {
		shardBlock.Body.Transactions = []metadata.Transaction{}
	}
	if shardBlock.Body.Instructions == nil {
		shardBlock.Body.Instructions = [][]string{}
	}
	if shardBlock.Body.CrossTransactions == nil {
		shardBlock.Body.CrossTransactions = make(map[byte][]CrossTransaction)
	}
	if shardBlock.Header.TotalTxsFee == nil {
		shardBlock.Header.TotalTxsFee = make(map[common.Hash]uint64)
	}
	if ok, err := shardBlock.validateSanityData(); !ok || err != nil {
		// panic(string(data) + err.Error())
		return NewBlockChainError(UnmashallJsonShardBlockError, err)
	}
	shardBlock.Body = blkBody
	return nil
}

func (shardBlock *ShardBlock) AddTransaction(tx metadata.Transaction) error {
	if shardBlock.Body.Transactions == nil {
		return NewBlockChainError(UnExpectedError, errors.New("not init tx arrays"))
	}
	shardBlock.Body.Transactions = append(shardBlock.Body.Transactions, tx)
	return nil
}

func (shardHeader *ShardHeader) String() string {
	res := common.EmptyString
	// res += shardHeader.ProducerAddress.String()
	res += string(shardHeader.ShardID)
	res += fmt.Sprintf("%v", shardHeader.Version)
	res += shardHeader.PreviousBlockHash.String()
	res += fmt.Sprintf("%v", shardHeader.Height)
	res += fmt.Sprintf("%v", shardHeader.Round)
	res += fmt.Sprintf("%v", shardHeader.Epoch)
	res += fmt.Sprintf("%v", shardHeader.Timestamp)
	res += shardHeader.TxRoot.String()
	res += shardHeader.ShardTxRoot.String()
	res += shardHeader.CrossTransactionRoot.String()
	res += shardHeader.InstructionsRoot.String()
	res += shardHeader.CommitteeRoot.String()
	res += shardHeader.PendingValidatorRoot.String()
	res += shardHeader.BeaconHash.String()
	res += shardHeader.StakingTxRoot.String()
	res += fmt.Sprintf("%v", shardHeader.BeaconHeight)
	tokenIDs := make([]common.Hash, 0)
	for tokenID, _ := range shardHeader.TotalTxsFee {
		tokenIDs = append(tokenIDs, tokenID)
	}
	sort.Slice(tokenIDs, func(i int, j int) bool {
		res, _ := tokenIDs[i].Cmp(&tokenIDs[j])
		return res == -1
	})

	for _, tokenID := range tokenIDs {
		res += fmt.Sprintf("%v~%v", tokenID.String(), shardHeader.TotalTxsFee[tokenID])
	}
	for _, value := range shardHeader.CrossShardBitMap {
		res += string(value)
	}

	if shardHeader.Version == 2 {
		res += shardHeader.Proposer
		res += fmt.Sprintf("%v", shardHeader.ProposeTime)
	}
	return res
}

func (shardHeader *ShardHeader) MetaHash() common.Hash {
	return common.Keccak256([]byte(shardHeader.String()))
}

func (shardHeader *ShardHeader) Hash() common.Hash {
	// Block header of bridge uses Keccak256 as a hash func to check on Ethereum when relaying blocks
	blkMetaHash := shardHeader.MetaHash()
	blkInstHash := shardHeader.InstructionMerkleRoot
	combined := append(blkMetaHash[:], blkInstHash[:]...)
	return common.Keccak256(combined)
}

/*
Customize UnmarshalJSON to parse list TxNormal
because we have many types of block, so we can need to customize data from marshal from json string to build a block
*/
func (shardBody *ShardBody) UnmarshalJSON(data []byte) error {
	type Alias ShardBody
	temp := &struct {
		Transactions []json.RawMessage
		*Alias
	}{
		Alias: (*Alias)(shardBody),
	}

	err := json.Unmarshal(data, &temp)
	if err != nil {
		return NewBlockChainError(UnmashallJsonShardBlockError, err)
	}

	// process tx from tx interface of temp
	for _, txTemp := range temp.Transactions {
		// txTempJson, _ := json.MarshalIndent(txTemp, "", "\t")
		var tx metadata.Transaction
		var parseErr error
		var txChoice *transaction.TxChoice
		txChoice, parseErr = transaction.DeserializeTransactionJSON(txTemp)
		tx = txChoice.ToTx()
		if tx==nil{
			return NewBlockChainError(UnmashallJsonShardBlockError, errors.New("corrupted TX"))
		}
		// txType := ""

		// if txTemp["Type"] != nil {
		// 	if err = json.Unmarshal(*txTemp["Type"], &txType); err != nil {
		// 		panic(fmt.Sprintf("Cannot parse Tx Type from Shard Body: %v", txTemp))
		// 		return NewBlockChainError(UnmashallJsonShardBlockError, err)
		// 	}
		// } else {
		// 	panic(fmt.Sprintf("Cannot parse Shard Body from data : %v", txTemp))
		// 	return NewBlockChainError(UnmashallJsonShardBlockError, err)
		// }

		// switch txType {
		// case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType, common.TxConversionType:
		// 	{
		// 		tx, parseErr = transaction.NewTransactionFromJsonBytes(txTempJson)
		// 	}
		// case common.TxCustomTokenPrivacyType, common.TxTokenConversionType:
		// 	{
		// 		tx, parseErr = transaction.NewTransactionTokenFromJsonBytes(txTempJson)
		// 	}
		// default:
		// 	{
		// 		return NewBlockChainError(UnmashallJsonShardBlockError, errors.New("Cannot parse a wrong tx "))
		// 	}
		// }
		if parseErr != nil {
			return NewBlockChainError(UnmashallJsonShardBlockError, parseErr)
		}
		shardBody.Transactions = append(shardBody.Transactions, tx)
	}
	return nil
}

func (shardBody ShardBody) Hash() common.Hash {
	res := []byte{}

	for _, item := range shardBody.Instructions {
		for _, l := range item {
			res = append(res, []byte(l)...)
		}
	}
	keys := []int{}
	for k := range shardBody.CrossTransactions {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, shardID := range keys {
		for _, value := range shardBody.CrossTransactions[byte(shardID)] {
			res = append(res, []byte(fmt.Sprintf("%v", value.BlockHeight))...)
			res = append(res, value.BlockHash.GetBytes()...)
			for _, coins := range value.OutputCoin {
				res = append(res, coins.Bytes()...)
			}
			for _, coins := range value.TokenPrivacyData {
				res = append(res, coins.Bytes()...)
			}
		}
	}
	for _, tx := range shardBody.Transactions {
		res = append(res, tx.Hash().GetBytes()...)
	}
	return common.HashH(res)
}

/*
- Concatenate all transaction in one shard as a string
- Then each shard producer a string value include all transactions within this block
- For each string value: Convert string value to hash value
- So if we have 256 shard, we will have 256 leaf value for merkle tree
- Make merkle root from these value
*/

func (shardBody ShardBody) CalcMerkleRootTx() *common.Hash {
	merkleRoots := Merkle{}.BuildMerkleTreeStore(shardBody.Transactions)
	merkleRoot := merkleRoots[len(merkleRoots)-1]
	return merkleRoot
}

func (shardBody ShardBody) ExtractIncomingCrossShardMap() (map[byte][]common.Hash, error) {
	crossShardMap := make(map[byte][]common.Hash)
	for shardID, crossblocks := range shardBody.CrossTransactions {
		for _, crossblock := range crossblocks {
			crossShardMap[shardID] = append(crossShardMap[shardID], crossblock.BlockHash)
		}
	}
	return crossShardMap, nil
}

func (shardBody ShardBody) ExtractOutgoingCrossShardMap() (map[byte][]common.Hash, error) {
	crossShardMap := make(map[byte][]common.Hash)
	return crossShardMap, nil
}

func (shardBlock *ShardBlock) CreateAllCrossShardBlock(activeShards int) map[byte]*CrossShardBlock {
	allCrossShard := make(map[byte]*CrossShardBlock)
	if activeShards == 1 {
		return allCrossShard
	}
	for i := 0; i < activeShards; i++ {
		shardID := common.GetShardIDFromLastByte(byte(i))
		if shardID != shardBlock.Header.ShardID {
			crossShard, err := shardBlock.CreateCrossShardBlock(shardID)
			if crossShard != nil {
				Logger.log.Criticalf("Create CrossShardBlock from Shard %+v to Shard %+v: %+v \n", shardBlock.Header.ShardID, shardID, crossShard)
			}
			if crossShard != nil && err == nil {
				allCrossShard[byte(i)] = crossShard
			}
		}
	}
	return allCrossShard
}

func (shardBlock ShardBlock) CreateCrossShardBlock(shardID byte) (*CrossShardBlock, error) {
	crossShard := &CrossShardBlock{}
	crossOutputCoin, crossCustomTokenPrivacyData, err := getCrossShardData(shardBlock.Body.Transactions, shardID)
	if err != nil {
		return nil, err
	}
	// Return nothing if nothing to cross
	if len(crossOutputCoin) == 0 && len(crossCustomTokenPrivacyData) == 0 {
		return nil, NewBlockChainError(CreateCrossShardBlockError, errors.New("No cross Outputcoin, Cross Custom Token, Cross Custom Token Privacy"))
	}
	merklePathShard, merkleShardRoot := GetMerklePathCrossShard(shardBlock.Body.Transactions, shardID)
	if merkleShardRoot != shardBlock.Header.ShardTxRoot {
		return crossShard, NewBlockChainError(VerifyCrossShardBlockShardTxRootError, fmt.Errorf("Expect Shard Tx Root To be %+v but get %+v", shardBlock.Header.ShardTxRoot, merkleShardRoot))
	}
	crossShard.ValidationData = shardBlock.ValidationData
	crossShard.Header = shardBlock.Header
	crossShard.MerklePathShard = merklePathShard
	crossShard.CrossOutputCoin = crossOutputCoin
	crossShard.CrossTxTokenPrivacyData = crossCustomTokenPrivacyData
	crossShard.ToShardID = shardID
	return crossShard, nil
}

func (block *ShardBlock) AddValidationField(validationData string) error {
	block.ValidationData = validationData
	return nil
}

func (block ShardBlock) GetCurrentEpoch() uint64 {
	return block.Header.Epoch
}

func (block ShardBlock) GetProducer() string {
	return block.Header.Producer
}

func (block ShardBlock) GetProducerPubKeyStr() string {
	return block.Header.ProducerPubKeyStr
}

func (block ShardBlock) GetValidationField() string {
	return block.ValidationData
}

func (block ShardBlock) GetVersion() int {
	return block.Header.Version
}

func (block ShardBlock) GetHeight() uint64 {
	return block.Header.Height
}

func (block ShardBlock) GetRound() int {
	return block.Header.Round
}

func (block ShardBlock) GetRoundKey() string {
	return fmt.Sprint(block.Header.Height, "_", block.Header.Round)
}

func (block ShardBlock) GetInstructions() [][]string {
	return block.Body.Instructions
}

func (block CrossShardBlock) GetProducer() string {
	return block.Header.Producer
}

func (block CrossShardBlock) GetVersion() int {
	return block.Header.Version
}

func (block CrossShardBlock) GetHeight() uint64 {
	return block.Header.Height
}

func (block CrossShardBlock) GetShardID() int {
	return int(block.Header.ShardID)
}

func (block CrossShardBlock) GetValidationField() string {
	return block.ValidationData
}

func (block CrossShardBlock) GetRound() int {
	return block.Header.Round
}

func (block CrossShardBlock) GetRoundKey() string {
	return fmt.Sprint(block.Header.Height, "_", block.Header.Round)
}

func (block CrossShardBlock) GetInstructions() [][]string {
	return [][]string{}
}

func (block ShardBlock) GetConsensusType() string {
	return block.Header.ConsensusType
}

func (block CrossShardBlock) GetConsensusType() string {
	return block.Header.ConsensusType
}
