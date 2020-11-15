package mempool

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/incdb"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/incognitochain/incognito-chain/pubsub"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/databasemp"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/transaction"
)

// default value
const (
	defaultScanTime          = 10 * time.Minute
	defaultIsUnlockMempool   = true
	defaultIsBlockGenStarted = false
	defaultRoleInCommittees  = -1
	defaultIsTest            = false
	defaultReplaceFeeRatio   = 1.1
)

// config is a descriptor containing the memory pool configuration.
type Config struct {
	BlockChain        *blockchain.BlockChain       // Block chain of node
	DataBase          map[int]incdb.Database       // main database of blockchain
	DataBaseMempool   databasemp.DatabaseInterface // database is used for storage data in mempool into lvdb
	ChainParams       *blockchain.Params
	FeeEstimator      map[byte]*FeeEstimator // FeeEstimatator provides a feeEstimator. If it is not nil, the mempool records all new transactions it observes into the feeEstimator.
	TxLifeTime        uint                   // Transaction life time in pool
	MaxTx             uint64                 //Max transaction pool may have
	IsLoadFromMempool bool                   //Reset mempool database when run node
	PersistMempool    bool
	RelayShards       []byte
	// UserKeyset            *incognitokey.KeySet
	PubSubManager         *pubsub.PubSubManager
	RoleInCommitteesEvent pubsub.EventChannel
}

// TxDesc is transaction message in mempool
type TxDesc struct {
	Desc            metadata.TxDesc // transaction details
	StartTime       time.Time       //Unix Time that transaction enter mempool
	IsFowardMessage bool
}

type TxPool struct {
	// The following variables must only be used atomically.
	config                    Config
	lastUpdated               int64 // last time pool was updated
	pool                      map[common.Hash]*TxDesc
	poolSerialNumbersHashList map[common.Hash][]common.Hash // [txHash] -> list hash serialNumbers of input coin
	poolSerialNumberHash      map[common.Hash]common.Hash   // [hash from list of serialNumber] -> txHash
	mtx                       sync.RWMutex
	poolCandidate             map[common.Hash]string //Candidate List in mempool
	candidateMtx              sync.RWMutex
	poolRequestStopStaking    map[common.Hash]string //request stop staking list in mempool
	requestStopStakingMtx     sync.RWMutex
	CPendingTxs               chan<- metadata.Transaction // channel to deliver txs to block gen
	CRemoveTxs                chan<- metadata.Transaction // channel to deliver txs to block gen
	RoleInCommittees          int                         //Current Role of Node
	roleMtx                   sync.RWMutex
	ScanTime                  time.Duration
	IsBlockGenStarted         bool
	IsUnlockMempool           bool
	ReplaceFeeRatio           float64

	//for testing
	IsTest       bool
	duplicateTxs map[common.Hash]uint64 //For testing
}

/*
Init Txpool from config
*/
func (tp *TxPool) Init(cfg *Config) {
	tp.config = *cfg
	tp.pool = make(map[common.Hash]*TxDesc)
	tp.poolSerialNumbersHashList = make(map[common.Hash][]common.Hash)
	tp.poolSerialNumberHash = make(map[common.Hash]common.Hash)
	tp.poolCandidate = make(map[common.Hash]string)
	tp.poolRequestStopStaking = make(map[common.Hash]string)
	tp.duplicateTxs = make(map[common.Hash]uint64)
	_, subChanRole, _ := tp.config.PubSubManager.RegisterNewSubscriber(pubsub.ShardRoleTopic)
	tp.config.RoleInCommitteesEvent = subChanRole
	tp.ScanTime = defaultScanTime
	tp.IsUnlockMempool = defaultIsUnlockMempool
	tp.IsBlockGenStarted = defaultIsBlockGenStarted
	tp.RoleInCommittees = defaultRoleInCommittees
	tp.IsTest = defaultIsTest
	tp.ReplaceFeeRatio = defaultReplaceFeeRatio
}

// InitChannelMempool - init channel
func (tp *TxPool) InitChannelMempool(cPendingTxs chan metadata.Transaction, cRemoveTxs chan metadata.Transaction) {
	tp.CPendingTxs = cPendingTxs
	tp.CRemoveTxs = cRemoveTxs
}

func (tp *TxPool) AnnouncePersisDatabaseMempool() {
	if tp.config.PersistMempool {
		Logger.log.Critical("Turn on Mempool Persistence Database")
	} else {
		Logger.log.Critical("Turn off Mempool Persistence Database")
	}
}

// LoadOrResetDatabaseMempool - Load and reset database of mempool when start node
func (tp *TxPool) LoadOrResetDatabaseMempool() error {
	if !tp.config.IsLoadFromMempool {
		err := tp.resetDatabaseMempool()
		if err != nil {
			Logger.log.Errorf("Fail to reset mempool database, error: %+v \n", err)
			return NewMempoolTxError(DatabaseError, err)
		} else {
			Logger.log.Critical("Successfully Reset from database")
		}
	} else {
		txDescs, err := tp.loadDatabaseMP()
		if err != nil {
			Logger.log.Errorf("Fail to load mempool database, error: %+v \n", err)
			return NewMempoolTxError(DatabaseError, err)
		} else {
			Logger.log.Criticalf("Successfully load %+v from database \n", len(txDescs))
		}
	}
	return nil
}

// loop forever in mempool
// receive data from other package
func (tp *TxPool) Start(cQuit chan struct{}) {
	for {
		select {
		case <-cQuit:
			return
		case msg := <-tp.config.RoleInCommitteesEvent:
			{
				shardID, ok := msg.Value.(int)
				if !ok {
					continue
				}
				go func() {
					tp.roleMtx.Lock()
					defer tp.roleMtx.Unlock()
					tp.RoleInCommittees = shardID
				}()
			}
		}
	}
}

func (tp *TxPool) MonitorPool() {
	if tp.config.TxLifeTime == 0 {
		return
	}
	ticker := time.NewTicker(tp.ScanTime)
	defer ticker.Stop()
	for _ = range ticker.C {
		tp.mtx.Lock()
		ttl := time.Duration(tp.config.TxLifeTime) * time.Second
		txsToBeRemoved := []*TxDesc{}
		Logger.log.Info("MonitorPool: Start to collect timeout ttl tx")
		for _, txDesc := range tp.pool {
			if time.Since(txDesc.StartTime) > ttl {
				Logger.log.Infof("MonitorPool: Add to list removed tx with txHash=%+v", txDesc.Desc.Tx.Hash().String())
				txsToBeRemoved = append(txsToBeRemoved, txDesc)
			}
		}
		Logger.log.Infof("MonitorPool: End to collect timeout ttl tx - Count of txsToBeRemoved=%+v", len(txsToBeRemoved))
		for _, txDesc := range txsToBeRemoved {
			txHash := *txDesc.Desc.Tx.Hash()
			tp.removeTx(txDesc.Desc.Tx)
			tp.TriggerCRemoveTxs(txDesc.Desc.Tx)
			tp.removeCandidateByTxHash(txHash)
			//tp.removeRequestStopStakingByTxHash(txHash)
			err := tp.config.DataBaseMempool.RemoveTransaction(txDesc.Desc.Tx.Hash())
			if err != nil {
				Logger.log.Errorf("MonitorPool: RemoveTransaction tx hash=%+v with error %+v", txDesc.Desc.Tx.Hash().String(), err)
				Logger.log.Error(err)
			}
		}
		tp.mtx.Unlock()
	}
}

// MaybeAcceptTransaction is the main workhorse for handling insertion of new
// free-standing transactions into a memory pool.  It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, detecting orphan transactions, and insertion into the memory pool.
//
// If the transaction is an orphan (missing parent transactions), the
// transaction is NOT added to the orphan pool, but each unknown referenced
// parent is returned.  Use ProcessTransaction instead if new orphans should
// be added to the orphan pool.
//
// This function is safe for concurrent access.
// #1: tx
// #2: default nil, contain input coins hash, which are used for creating this tx
func (tp *TxPool) MaybeAcceptTransaction(tx metadata.Transaction, beaconHeight int64) (*common.Hash, *TxDesc, error) {
	//beaconView.BeaconHeight
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	if tp.IsTest {
		return &common.Hash{}, &TxDesc{}, nil
	}
	go func(txHash common.Hash) {
		tp.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.TransactionHashEnterNodeTopic, txHash))
	}(*tx.Hash())
	senderShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	if !tp.checkRelayShard(tx) && !tp.checkPublicKeyRole(tx) {
		err := NewMempoolTxError(UnexpectedTransactionError, errors.New("Unexpected Transaction From Shard "+fmt.Sprintf("%d", senderShardID)))
		Logger.log.Error(err)
		return &common.Hash{}, &TxDesc{}, err
	}
	beaconView := tp.config.BlockChain.BeaconChain.GetFinalView().(*blockchain.BeaconBestState)
	shardView := tp.config.BlockChain.ShardChain[senderShardID].GetBestView().(*blockchain.ShardBestState)
	//==========
	if uint64(len(tp.pool)) >= tp.config.MaxTx {
		return nil, nil, NewMempoolTxError(MaxPoolSizeError, errors.New("Pool reach max number of transaction"))
	}
	if tx.GetType() == common.TxReturnStakingType{
		return &common.Hash{}, &TxDesc{}, NewMempoolTxError(RejectInvalidTx, fmt.Errorf("%+v is a return staking tx", tx.Hash().String()))
	}
	if tx.GetType() == common.TxCustomTokenPrivacyType{
		tempTx, ok := tx.(*transaction.TxCustomTokenPrivacy)
		if !ok {
			return &common.Hash{}, &TxDesc{}, NewMempoolTxError(RejectInvalidTx, fmt.Errorf("cannot detect transaction type for tx %+v", tx.Hash().String()))
		}
		if tempTx.TxPrivacyTokenData.Mintable{
			return &common.Hash{}, &TxDesc{}, NewMempoolTxError(RejectInvalidTx, fmt.Errorf("%+v is a minteable tx", tx.Hash().String()))
		}
	}
	hash, txDesc, err := tp.maybeAcceptTransaction(shardView, beaconView, tx, tp.config.PersistMempool, true, beaconHeight)
	//==========
	if err != nil {
		Logger.log.Error(err)
	} else {
		if tp.IsBlockGenStarted {
			if tp.IsUnlockMempool {
				go func(tx metadata.Transaction) {
					tp.CPendingTxs <- tx
				}(tx)
			}
		}
		// Publish Message
		go tp.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.MempoolInfoTopic, tp.listTxs()))
	}
	return hash, txDesc, err
}

// This function is safe for concurrent access.
func (tp *TxPool) MaybeAcceptTransactionForBlockProducing(tx metadata.Transaction, beaconHeight int64, shardView *blockchain.ShardBestState) (*metadata.TxDesc, error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	bHeight := shardView.BestBlock.Header.BeaconHeight
	beaconBlockHash := shardView.BestBlock.Header.BeaconHash
	beaconView, err := tp.config.BlockChain.GetBeaconViewStateDataFromBlockHash(beaconBlockHash)
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}
	_, txDesc, err := tp.maybeAcceptTransaction(shardView, beaconView, tx, false, false, int64(bHeight))
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}
	tempTxDesc := &txDesc.Desc
	return tempTxDesc, err
}

func (tp *TxPool) MaybeAcceptBatchTransactionForBlockProducing(shardID byte, txs []metadata.Transaction, beaconHeight int64, shardView *blockchain.ShardBestState) ([]*metadata.TxDesc, error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	bHeight := shardView.BestBlock.Header.BeaconHeight
	beaconBlockHash := shardView.BestBlock.Header.BeaconHash
	beaconView, err := tp.config.BlockChain.GetBeaconViewStateDataFromBlockHash(beaconBlockHash)
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}
	_, txDesc, err := tp.maybeAcceptBatchTransaction(shardView, beaconView, shardID, txs, int64(bHeight))
	return txDesc, err
}

func (tp *TxPool) maybeAcceptBatchTransaction(shardView *blockchain.ShardBestState, beaconView *blockchain.BeaconBestState, shardID byte, txs []metadata.Transaction, beaconHeight int64) ([]common.Hash, []*metadata.TxDesc, error) {
	txDescs := []*metadata.TxDesc{}
	txHashes := []common.Hash{}
	batch := transaction.NewBatchTransaction(txs)

	boolParams := make(map[string]bool)
	boolParams["isNewTransaction"] = false
	boolParams["isBatch"] = true
	boolParams["isNewZKP"] = tp.config.BlockChain.IsAfterNewZKPCheckPoint(uint64(beaconHeight))

	ok, err, _ := batch.Validate(shardView.GetCopiedTransactionStateDB(), beaconView.GetBeaconFeatureStateDB(), boolParams)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, fmt.Errorf("Verify Batch Transaction failed %+v", txs)
	}
	for _, tx := range txs {
		// validate tx
		err := tp.validateTransaction(shardView, beaconView, tx, beaconHeight, true, false)
		if err != nil {
			return nil, nil, err
		}
		shardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
		bestHeight := tp.config.BlockChain.GetBestStateShard(byte(shardID)).BestBlock.Header.Height
		txFee := tx.GetTxFee()
		txFeeToken := tx.GetTxFeeToken()
		txD := createTxDescMempool(tx, bestHeight, txFee, txFeeToken)
		err = tp.addTx(txD, false)
		if err != nil {
			return nil, nil, err
		}
		txDescs = append(txDescs, &txD.Desc)
		txHashes = append(txHashes, *tx.Hash())
	}
	return txHashes, txDescs, nil
}

/*
// maybeAcceptTransaction into pool
// #1: tx
// #2: store into db
// #3: default nil, contain input coins hash, which are used for creating this tx
*/
func (tp *TxPool) maybeAcceptTransaction(shardView *blockchain.ShardBestState, beaconView *blockchain.BeaconBestState, tx metadata.Transaction, isStore bool, isNewTransaction bool, beaconHeight int64) (*common.Hash, *TxDesc, error) {
	// validate tx
	err := tp.validateTransaction(shardView, beaconView, tx, beaconHeight, false, isNewTransaction)
	if err != nil {
		return nil, nil, err
	}
	shardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	bestHeight := shardView.BestBlock.Header.Height
	txFee := tx.GetTxFee()
	txFeeToken := tx.GetTxFeeToken()
	txD := createTxDescMempool(tx, bestHeight, txFee, txFeeToken)
	err = tp.addTx(txD, isStore)
	if err != nil {
		return nil, nil, err
	}
	if isNewTransaction {
		Logger.log.Infof("Add New Txs Into Pool %+v FROM SHARD %+v\n", *tx.Hash(), shardID)
	}
	return tx.Hash(), txD, nil
}

// createTxDescMempool - return an object TxDesc for mempool from original Tx
func createTxDescMempool(tx metadata.Transaction, height uint64, fee uint64, feeToken uint64) *TxDesc {
	txDesc := &TxDesc{
		Desc: metadata.TxDesc{
			Tx:       tx,
			Height:   height,
			Fee:      fee,
			FeeToken: feeToken,
		},
		StartTime:       time.Now(),
		IsFowardMessage: false,
	}
	return txDesc
}

func (tp *TxPool) checkFees(
	beaconView *blockchain.BeaconBestState,
	tx metadata.Transaction,
	shardID byte,
	beaconHeight int64,
) bool {
	Logger.log.Info("Beacon heigh for checkFees: ", beaconHeight, tx.Hash().String())
	txType := tx.GetType()
	if txType == common.TxCustomTokenPrivacyType {
		limitFee := tp.config.FeeEstimator[shardID].GetLimitFeeForNativeToken()
		beaconStateDB, err := tp.config.BlockChain.GetBestStateBeaconFeatureStateDBByHeight(uint64(beaconHeight), tp.config.DataBase[common.BeaconChainDataBaseID])
		if err != nil {
			Logger.log.Errorf("ERROR: %+v", NewMempoolTxError(RejectInvalidFee,
					fmt.Errorf("transaction %+v - cannot get beacon state db at height: %d",
						tx.Hash().String(), beaconHeight)))
				return false
		}

		// check transaction fee for meta data
		meta := tx.GetMetadata()
		// verify at metadata level
		if meta != nil {
			ok := meta.CheckTransactionFee(tx, limitFee, beaconHeight, beaconStateDB)
			if !ok {
				Logger.log.Errorf("Error: %+v", NewMempoolTxError(RejectInvalidFee,
					fmt.Errorf("transaction %+v: Invalid fee metadata",
						tx.Hash().String())))
			}
			return ok
		}
		// verify at transaction level
		tokenID := tx.GetTokenID()
		feeNativeToken := tx.GetTxFee()
		feePToken := tx.GetTxFeeToken()
		//convert fee in Ptoken to fee in native token (if feePToken > 0)
		if feePToken > 0 {
			feePTokenToNativeTokenTmp, err := metadata.ConvertPrivacyTokenToNativeToken(feePToken, tokenID, beaconHeight, beaconStateDB)
			if err != nil {
				Logger.log.Errorf("ERROR: %+v", NewMempoolTxError(RejectInvalidFee,
					fmt.Errorf("transaction %+v: %+v %v can not convert to native token %+v",
						tx.Hash().String(), feePToken, tokenID, err)))
				return false
			}

			feePTokenToNativeToken := uint64(math.Ceil(feePTokenToNativeTokenTmp))
			feeNativeToken += feePTokenToNativeToken
		}
		// get limit fee in native token
		actualTxSize := tx.GetTxActualSize()
		// check fee in native token
		minFee := actualTxSize * limitFee
		if feeNativeToken < minFee {
			Logger.log.Errorf("ERROR: %+v", NewMempoolTxError(RejectInvalidFee,
				fmt.Errorf("transaction %+v has %d fees PRV which is under the required amount of %d, tx size %d",
					tx.Hash().String(), feeNativeToken, minFee, actualTxSize)))
			return false
		}
	} else {
		// This is a normal tx -> only check like normal tx with PRV
		limitFee := tp.config.FeeEstimator[shardID].limitFee
		txFee := tx.GetTxFee()
		// txNormal := tx.(*transaction.Tx)
		if limitFee > 0 {
			meta := tx.GetMetadata()
			if meta != nil {
				ok := tx.GetMetadata().CheckTransactionFee(tx, limitFee, beaconHeight, beaconView.GetBeaconFeatureStateDB())
				if !ok {
					Logger.log.Errorf("ERROR: %+v", NewMempoolTxError(RejectInvalidFee,
						fmt.Errorf("transaction %+v has %d fees which is under the required amount of %d",
							tx.Hash().String(), txFee, limitFee*tx.GetTxActualSize())))
				}
				return ok
			}
			fullFee := limitFee * tx.GetTxActualSize()
			// ok := tx.CheckTransactionFee(limitFee)
			if txFee < fullFee {
				Logger.log.Errorf("ERROR: %+v", NewMempoolTxError(RejectInvalidFee,
					fmt.Errorf("transaction %+v has %d fees which is under the required amount of %d",
						tx.Hash().String(), txFee, limitFee*tx.GetTxActualSize())))
				return false
			}
		}
	}
	return true
}

/*
// maybeAcceptTransaction is the internal function which implements the public
// See the comment for MaybeAcceptTransaction for more details.
// This function MUST be called with the mempool lock held (for writes).
In Param#2: isStore: store transaction to persistence storage only work for transaction come from user (not for validation process)
1. Validate sanity data of tx
2. Validate duplicate tx
3. Do not accept a salary tx
4. Validate fee with tx size
5. Validate with other txs in mempool
5.1 Check for Replacement or Cancel transaction
6. Validate data in tx: privacy proof, metadata,...
7. Validate tx with blockchain: douple spend, ...
9. Staking Transaction: Check Duplicate stake public key in pool ONLY with staking transaction
10. RequestStopAutoStaking
*/
func (tp *TxPool) validateTransaction(shardView *blockchain.ShardBestState, beaconView *blockchain.BeaconBestState, tx metadata.Transaction, beaconHeight int64, isBatch bool, isNewTransaction bool) error {
	var err error
	txHash := tx.Hash()
	shardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	// Condition 1: sanity data
	validated := false
	if !isNewTransaction {
		// need to use beacon height from
		validated, err = tx.ValidateSanityData(tp.config.BlockChain, shardView, beaconView, uint64(beaconHeight))
	} else {
		validated, err = tx.ValidateSanityData(tp.config.BlockChain, shardView, beaconView, 0)
	}
	if !validated {
		// try parse to TransactionError
		sanityError, ok := err.(*transaction.TransactionError)
		if ok {
			switch sanityError.Code {
			case transaction.ErrCodeMessage[transaction.RejectInvalidLockTime].Code:
				{
					return NewMempoolTxError(RejectSanityTxLocktime, fmt.Errorf("transaction's sansity %v is error %v", txHash.String(), sanityError))
				}
			case transaction.ErrCodeMessage[transaction.RejectTxType].Code:
				{
					return NewMempoolTxError(RejectInvalidTxType, fmt.Errorf("transaction's sansity %v is error %v", txHash.String(), sanityError))
				}
			case transaction.ErrCodeMessage[transaction.RejectTxVersion].Code:
				{
					return NewMempoolTxError(RejectVersion, fmt.Errorf("transaction's sansity %v is error %v", txHash.String(), sanityError))
				}
			}
		}
		return NewMempoolTxError(RejectSanityTx, fmt.Errorf("transaction's sansity %v is error %v", txHash.String(), err))
	}

	// Condition 2: Don't accept the transaction if it already exists in the pool.
	isTxInPool := tp.isTxInPool(txHash)
	if isTxInPool {
		return NewMempoolTxError(RejectDuplicateTx, fmt.Errorf("already had transaction %+v in mempool", txHash.String()))
	}
	// Condition 3: A standalone transaction must not be a salary transaction.
	isSalaryTx := tx.IsSalaryTx()
	if isSalaryTx {
		return NewMempoolTxError(RejectSalaryTx, fmt.Errorf("%+v is salary tx", txHash.String()))
	}
	// Condition 4: check fee PRV of tx
	validFee := tp.checkFees(beaconView, tx, shardID, beaconHeight)
	if !validFee {
		return NewMempoolTxError(RejectInvalidFee,
			fmt.Errorf("Transaction %+v has invalid fees.",
				tx.Hash().String()))
	}
	// Condition 5: check tx with all txs in current mempool
	err = tx.ValidateTxWithCurrentMempool(tp)
	if err != nil {
		replaceErr, isReplacedTx := tp.validateTransactionReplacement(tx)
		// if replace tx success (no replace error found) then continue with next validate condition
		if isReplacedTx {
			if replaceErr != nil {
				return replaceErr
			}
		} else {
			// replace fail
			return NewMempoolTxError(RejectDoubleSpendWithMempoolTx, err)
		}
	}
	// Condition 6: ValidateTransaction tx by it self
	if !isBatch {
		isNewZKP := tp.config.BlockChain.IsAfterNewZKPCheckPoint(uint64(beaconHeight))

		boolParams := make(map[string]bool)
		boolParams["hasPrivacy"] = tx.IsPrivacy()
		boolParams["isNewTransaction"] = isNewTransaction
		boolParams["isNewZKP"] = isNewZKP

		validated, errValidateTxByItself := tx.ValidateTxByItself(boolParams, shardView.GetCopiedTransactionStateDB(), beaconView.GetBeaconFeatureStateDB(), tp.config.BlockChain, shardID, nil, nil)
		if !validated {
			return NewMempoolTxError(RejectInvalidTx, errValidateTxByItself)
		}
	}
	// Condition 7: validate tx with data of blockchain
	err = tx.ValidateTxWithBlockChain(tp.config.BlockChain, shardView, beaconView, shardID, shardView.GetCopiedTransactionStateDB())
	if err != nil {
		// parse error
		e1, ok := err.(*transaction.TransactionError)
		if ok {
			switch e1.Code {
			case transaction.RejectTxMedataWithBlockChain:
				{
					return NewMempoolTxError(RejectMetadataWithBlockchainTx, err)
				}
			}
		}
		return NewMempoolTxError(RejectDoubleSpendWithBlockchainTx, err)
	}
	// Condition 9: check duplicate stake public key ONLY with staking transaction
	pubkey := ""
	foundPubkey := -1
	if tx.GetMetadata() != nil {
		if tx.GetMetadata().GetType() == metadata.ShardStakingMeta || tx.GetMetadata().GetType() == metadata.BeaconStakingMeta {
			stakingMetadata, ok := tx.GetMetadata().(*metadata.StakingMetadata)
			if !ok {
				return NewMempoolTxError(GetStakingMetadataError, fmt.Errorf("Expect metadata type to be *metadata.StakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata())))
			}
			pubkey = stakingMetadata.CommitteePublicKey
			tp.candidateMtx.RLock()
			foundPubkey = common.IndexOfStrInHashMap(stakingMetadata.CommitteePublicKey, tp.poolCandidate)
			tp.candidateMtx.RUnlock()
		}
	}
	if foundPubkey > 0 {
		return NewMempoolTxError(RejectDuplicateStakePubkey, fmt.Errorf("This public key already stake and still in pool %+v", pubkey))
	}
	// Condition 10: check duplicate request stop auto staking
	requestedPublicKey := ""
	foundRequestStopAutoStaking := -1
	if tx.GetMetadata() != nil {
		if tx.GetMetadata().GetType() == metadata.StopAutoStakingMeta {
			stopAutoStakingMetadata, ok := tx.GetMetadata().(*metadata.StopAutoStakingMetadata)
			if !ok {
				return NewMempoolTxError(GetStakingMetadataError, fmt.Errorf("Expect metadata type to be *metadata.StopAutoStakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata())))
			}
			requestedPublicKey = stopAutoStakingMetadata.CommitteePublicKey
			tp.requestStopStakingMtx.RLock()
			foundRequestStopAutoStaking = common.IndexOfStrInHashMap(stopAutoStakingMetadata.CommitteePublicKey, tp.poolRequestStopStaking)
			tp.requestStopStakingMtx.RUnlock()
		}
	}
	if foundRequestStopAutoStaking > 0 {
		return NewMempoolTxError(RejectDuplicateRequestStopAutoStaking, fmt.Errorf("This public key already request to stop auto staking and still in pool %+v", requestedPublicKey))
	}
	return nil
}

// check transaction in pool
func (tp *TxPool) isTxInPool(hash *common.Hash) bool {
	if _, exists := tp.pool[*hash]; exists {
		return true
	}
	return false
}

func (tp *TxPool) validateTransactionReplacement(tx metadata.Transaction) (error, bool) {
	// calculate match serial number list in pool for replaced tx
	serialNumberHashList := tx.ListSerialNumbersHashH()
	hash := common.HashArrayOfHashArray(serialNumberHashList)
	// find replace tx in pool
	if txHashToBeReplaced, ok := tp.poolSerialNumberHash[hash]; ok {
		if txDescToBeReplaced, ok := tp.pool[txHashToBeReplaced]; ok {
			var baseReplaceFee float64
			var baseReplaceFeeToken float64
			var replaceFee float64
			var replaceFeeToken float64
			var isReplaced = false
			if txDescToBeReplaced.Desc.Fee > 0 && txDescToBeReplaced.Desc.FeeToken == 0 {
				// paid by prv fee only
				baseReplaceFee = float64(txDescToBeReplaced.Desc.Fee)
				replaceFee = float64(tx.GetTxFee())
				// not a higher enough fee than return error
				if baseReplaceFee*tp.ReplaceFeeRatio >= replaceFee {
					return NewMempoolTxError(RejectReplacementTxError, fmt.Errorf("Expect fee to be greater than %+v but get %+v ", baseReplaceFee, replaceFee)), true
				}
				isReplaced = true
			} else if txDescToBeReplaced.Desc.Fee == 0 && txDescToBeReplaced.Desc.FeeToken > 0 {
				//paid by token fee only
				baseReplaceFeeToken = float64(txDescToBeReplaced.Desc.FeeToken)
				replaceFeeToken = float64(tx.GetTxFeeToken())
				// not a higher enough fee than return error
				if baseReplaceFeeToken*tp.ReplaceFeeRatio >= replaceFeeToken {
					return NewMempoolTxError(RejectReplacementTxError, fmt.Errorf("Expect fee to be greater than %+v but get %+v ", baseReplaceFeeToken, replaceFeeToken)), true
				}
				isReplaced = true
			} else if txDescToBeReplaced.Desc.Fee > 0 && txDescToBeReplaced.Desc.FeeToken > 0 {
				// paid by both prv fee and token fee
				// then only one of fee is higher then it will be accepted
				baseReplaceFee = float64(txDescToBeReplaced.Desc.Fee)
				replaceFee = float64(tx.GetTxFee())
				baseReplaceFeeToken = float64(txDescToBeReplaced.Desc.FeeToken)
				replaceFeeToken = float64(tx.GetTxFeeToken())
				// not a higher enough fee than return error
				if baseReplaceFee*tp.ReplaceFeeRatio >= replaceFee || baseReplaceFeeToken*tp.ReplaceFeeRatio >= replaceFeeToken {
					return NewMempoolTxError(RejectReplacementTxError, fmt.Errorf("Expect fee to be greater than %+v but get %+v ", baseReplaceFee, replaceFee)), true
				}
				isReplaced = true
			}
			if isReplaced {
				txToBeReplaced := txDescToBeReplaced.Desc.Tx
				tp.removeTx(txToBeReplaced)
				tp.TriggerCRemoveTxs(txToBeReplaced)
				//tp.removeRequestStopStakingByTxHash(*txToBeReplaced.Hash())
				// send tx into channel of CRmoveTxs
				tp.TriggerCRemoveTxs(tx)
				return nil, true
			} else {
				return NewMempoolTxError(RejectReplacementTxError, fmt.Errorf("Unexpected error occur")), true
			}
		} else {
			//found no tx to be replaced
			return nil, false
		}
	} else {
		// no match serial number list to be replaced
		return nil, false
	}
}

// TriggerCRemoveTxs - send a tx channel into CRemoveTxs of tx mempool
func (tp *TxPool) TriggerCRemoveTxs(tx metadata.Transaction) {
	if tp.IsBlockGenStarted {
		go func(tx metadata.Transaction) {
			tp.CRemoveTxs <- tx
		}(tx)
	}
}

/*
// add transaction into pool
// #1: tx
// #2: store into db
// #3: default nil, contain input coins hash, which are used for creating this tx
*/
func (tp *TxPool) addTx(txD *TxDesc, isStore bool) error {
	tx := txD.Desc.Tx
	txHash := tx.Hash()
	if isStore {
		err := tp.addTransactionToDatabaseMempool(txHash, *txD)
		if err != nil {
			Logger.log.Errorf("Fail to add tx %+v to mempool database %+v \n", *txHash, err)
		} else {
			Logger.log.Criticalf("Add tx %+v to mempool database success \n", *txHash)
		}
	}
	tp.pool[*txHash] = txD
	var serialNumberList []common.Hash
	serialNumberList = append(serialNumberList, txD.Desc.Tx.ListSerialNumbersHashH()...)
	serialNumberListHash := common.HashArrayOfHashArray(serialNumberList)
	tp.poolSerialNumberHash[serialNumberListHash] = *txD.Desc.Tx.Hash()
	tp.poolSerialNumbersHashList[*txHash] = serialNumberList
	atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())
	// Record this tx for fee estimation if enabled, apply for normal tx and privacy token tx
	if tp.config.FeeEstimator != nil {
		var shardID byte
		flag := false
		switch tx.GetType() {
		case common.TxNormalType:
			{
				shardID = common.GetShardIDFromLastByte(tx.(*transaction.Tx).PubKeyLastByteSender)
				flag = true
			}
		case common.TxCustomTokenPrivacyType:
			{
				shardID = common.GetShardIDFromLastByte(tx.(*transaction.TxCustomTokenPrivacy).PubKeyLastByteSender)
				flag = true
			}
		}
		if flag {
			if temp, ok := tp.config.FeeEstimator[shardID]; ok {
				temp.ObserveTransaction(txD)
			}
		}
	}
	// add candidate into candidate list ONLY with staking transaction
	if tx.GetMetadata() != nil {
		metadataType := tx.GetMetadata().GetType()
		switch metadataType {
		case metadata.ShardStakingMeta, metadata.BeaconStakingMeta:
			{
				stakingMetadata, ok := tx.GetMetadata().(*metadata.StakingMetadata)
				if !ok {
					return NewMempoolTxError(GetStakingMetadataError, fmt.Errorf("Expect metadata type to be *metadata.StakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata())))
				}
				tp.addCandidateToList(*txHash, stakingMetadata.CommitteePublicKey)
			}
		case metadata.StopAutoStakingMeta:
			{
				stopAutoStakingMetadata, ok := tx.GetMetadata().(*metadata.StopAutoStakingMetadata)
				if !ok {
					return NewMempoolTxError(GetStakingMetadataError, fmt.Errorf("Expect metadata type to be *metadata.StopAutoStakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata())))
				}
				tp.addRequestStopStakingToList(*txHash, stopAutoStakingMetadata.CommitteePublicKey)
			}
		default:
			{
				Logger.log.Debug("Metadata Type:", metadataType)
			}
		}
	}
	Logger.log.Infof("Add Transaction %+v Successs \n", txHash.String())
	return nil
}

// Check relay shard and public key role before processing transaction
func (tp *TxPool) checkRelayShard(tx metadata.Transaction) bool {
	senderShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	if common.IndexOfByte(senderShardID, tp.config.RelayShards) > -1 {
		return true
	}
	return false
}

func (tp *TxPool) checkPublicKeyRole(tx metadata.Transaction) bool {
	senderShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	tp.roleMtx.RLock()
	if tp.RoleInCommittees > -1 && byte(tp.RoleInCommittees) == senderShardID {
		tp.roleMtx.RUnlock()
		return true
	} else {
		tp.roleMtx.RUnlock()
		return false
	}
}

// RemoveTx safe remove transaction for pool
func (tp *TxPool) RemoveTx(txs []metadata.Transaction, isInBlock bool) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	// remove transaction from database mempool
	for _, tx := range txs {
		if tp.config.PersistMempool {
			err := tp.removeTransactionFromDatabaseMP(tx.Hash())
			if err != nil {
				Logger.log.Error(err)
			}
		}
		tp.removeTx(tx)
		tp.TriggerCRemoveTxs(tx)
	}
	return
}

/*
	- Remove transaction out of pool
		+ Tx Description pool
		+ List Serial Number Pool
		+ Hash of List Serial Number Pool
	- Transaction want to be removed maybe replaced by another transaction:
		+ New tx (Replacement tx) still exist in pool
		+ Using the same list serial number to delete new transaction out of pool
*/
func (tp *TxPool) removeTx(tx metadata.Transaction) {
	//Logger.log.Infof((*tx).Hash().String())
	if _, exists := tp.pool[*tx.Hash()]; exists {
		delete(tp.pool, *tx.Hash())
		atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())
	}
	if _, exists := tp.poolSerialNumbersHashList[*tx.Hash()]; exists {
		delete(tp.poolSerialNumbersHashList, *tx.Hash())
	}
	serialNumberHashList := tx.ListSerialNumbersHashH()
	hash := common.HashArrayOfHashArray(serialNumberHashList)
	if _, exists := tp.poolSerialNumberHash[hash]; exists {
		delete(tp.poolSerialNumberHash, hash)
		// Using the same list serial number to delete new transaction out of pool
		// this new transaction maybe not exist
		if _, exists := tp.pool[hash]; exists {
			delete(tp.pool, hash)
			atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())
		}
		if _, exists := tp.poolSerialNumbersHashList[hash]; exists {
			delete(tp.poolSerialNumbersHashList, hash)
		}
	}
	tp.removeRequestStopStakingByTxHash(*tx.Hash())
}

func (tp *TxPool) addCandidateToList(txHash common.Hash, candidate string) {
	tp.candidateMtx.Lock()
	defer tp.candidateMtx.Unlock()
	tp.poolCandidate[txHash] = candidate
}

func (tp *TxPool) removeCandidateByTxHash(txHash common.Hash) {
	tp.candidateMtx.Lock()
	defer tp.candidateMtx.Unlock()
	if _, exist := tp.poolCandidate[txHash]; exist {
		delete(tp.poolCandidate, txHash)
	}
}

func (tp *TxPool) RemoveCandidateList(candidate []string) {
	tp.candidateMtx.Lock()
	defer tp.candidateMtx.Unlock()
	candidateToBeRemoved := []common.Hash{}
	for _, value := range candidate {
		for txHash, currentCandidate := range tp.poolCandidate {
			if strings.Compare(value, currentCandidate) == 0 {
				candidateToBeRemoved = append(candidateToBeRemoved, txHash)
				break
			}
		}
	}
	for _, txHash := range candidateToBeRemoved {
		delete(tp.poolCandidate, txHash)
	}
}
func (tp *TxPool) addRequestStopStakingToList(txHash common.Hash, requestStopStaking string) {
	tp.requestStopStakingMtx.Lock()
	defer tp.requestStopStakingMtx.Unlock()
	tp.poolRequestStopStaking[txHash] = requestStopStaking
}

func (tp *TxPool) removeRequestStopStakingByTxHash(txHash common.Hash) {
	tp.requestStopStakingMtx.Lock()
	defer tp.requestStopStakingMtx.Unlock()
	if _, exist := tp.poolRequestStopStaking[txHash]; exist {
		delete(tp.poolRequestStopStaking, txHash)
	}
}

func (tp *TxPool) RemoveRequestStopStakingList(requestStopStakings []string) {
	tp.requestStopStakingMtx.Lock()
	defer tp.requestStopStakingMtx.Unlock()
	requestStopStakingsToBeRemoved := []common.Hash{}
	for _, requestStopStaking := range requestStopStakings {
		for txHash, currentRequestStopStaking := range tp.poolRequestStopStaking {
			if strings.Compare(requestStopStaking, currentRequestStopStaking) == 0 {
				requestStopStakingsToBeRemoved = append(requestStopStakingsToBeRemoved, txHash)
				break
			}
		}
	}
	for _, txHash := range requestStopStakingsToBeRemoved {
		delete(tp.poolRequestStopStaking, txHash)
	}
}

//=======================Service for other package
// SendTransactionToBlockGen - push tx into channel and send to Block generate of consensus
func (tp *TxPool) SendTransactionToBlockGen() {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	for _, txdesc := range tp.pool {
		tp.CPendingTxs <- txdesc.Desc.Tx
	}
	tp.IsUnlockMempool = true
}

// MarkForwardedTransaction - mart a transaction is forward message
func (tp *TxPool) MarkForwardedTransaction(txHash common.Hash) {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	if tp.IsTest {
		return
	}
	if _, ok := tp.pool[txHash]; ok {
		tp.pool[txHash].IsFowardMessage = true
	}
}

// GetTx get transaction info by hash
func (tp *TxPool) GetTx(txHash *common.Hash) (metadata.Transaction, error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	Logger.log.Info(txHash.String())
	txDesc, exists := tp.pool[*txHash]
	if exists {
		return txDesc.Desc.Tx, nil
	}
	err := NewMempoolTxError(TransactionNotFoundError, fmt.Errorf("Transaction "+txHash.String()+" Not Found!"))
	Logger.log.Error(err)
	return nil, err
}

// // MiningDescs returns a slice of mining descriptors for all the transactions
// // in the pool.
func (tp *TxPool) MiningDescs() []*metadata.TxDesc {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	descs := []*metadata.TxDesc{}
	for _, desc := range tp.pool {
		descs = append(descs, &desc.Desc)
	}
	return descs
}

func (tp TxPool) GetPool() map[common.Hash]*TxDesc {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	pool := make(map[common.Hash]*TxDesc)
	for k, v := range tp.pool {
		pool[k] = v
	}
	return tp.pool
}

// Count return len of transaction pool
func (tp *TxPool) Count() int {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	count := len(tp.pool)
	return count
}

func (tp TxPool) GetClonedPoolCandidate() map[common.Hash]string {
	tp.candidateMtx.RLock()
	defer tp.candidateMtx.RUnlock()
	result := make(map[common.Hash]string)
	for k, v := range tp.poolCandidate {
		result[k] = v
	}
	return result
}

/*
Sum of all transactions sizes
*/
func (tp *TxPool) Size() uint64 {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	size := uint64(0)
	for _, tx := range tp.pool {
		size += tx.Desc.Tx.GetTxActualSize()
	}
	return size
}

// Get Max fee
func (tp *TxPool) MaxFee() uint64 {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	fee := uint64(0)
	for _, tx := range tp.pool {
		if tx.Desc.Fee > fee {
			fee = tx.Desc.Fee
		}
	}
	return fee
}

/*
// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
*/
func (tp *TxPool) LastUpdated() time.Time {
	return time.Unix(tp.lastUpdated, 0)
}

/*
// HaveTransaction returns whether or not the passed transaction hash
	// exists in the source pool.
*/
func (tp *TxPool) HaveTransaction(hash *common.Hash) bool {
	// Protect concurrent access.
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	haveTx := tp.isTxInPool(hash)
	return haveTx
}

/*
List all tx ids in mempool
*/
func (tp *TxPool) ListTxs() []string {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	return tp.listTxs()
}

/*
List all tx ids in mempool
*/
func (tp *TxPool) listTxs() []string {
	result := make([]string, 0)
	for _, tx := range tp.pool {
		result = append(result, tx.Desc.Tx.Hash().String())
	}
	return result
}

/*
List all tx ids in mempool
*/
func (tp *TxPool) ListTxsDetail() []metadata.Transaction {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	result := make([]metadata.Transaction, 0)
	for _, tx := range tp.pool {
		result = append(result, tx.Desc.Tx)
	}
	return result
}

// ValidateSerialNumberHashH - check serialNumberHashH which is
// used by a tx in mempool
func (tp *TxPool) ValidateSerialNumberHashH(serialNumber []byte) error {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	hash := common.HashH(serialNumber)
	for txHash, serialNumbersHashH := range tp.poolSerialNumbersHashList {
		for _, serialNumberHashH := range serialNumbersHashH {
			if serialNumberHashH.IsEqual(&hash) {
				return NewMempoolTxError(DuplicateSerialNumbersHashError, fmt.Errorf("Transaction %+v use duplicate current serial number in pool", txHash))
			}
		}
	}
	return nil
}

func (tp *TxPool) EmptyPool() bool {
	tp.mtx.Lock()
	tp.candidateMtx.Lock()
	tp.requestStopStakingMtx.Lock()
	tp.roleMtx.Lock()
	defer func() {
		tp.candidateMtx.Unlock()
		tp.mtx.Unlock()
		tp.requestStopStakingMtx.Unlock()
		tp.roleMtx.Unlock()
	}()

	if len(tp.pool) == 0 && len(tp.poolSerialNumbersHashList) == 0 && len(tp.poolCandidate) == 0 {
		return true
	}
	tp.pool = make(map[common.Hash]*TxDesc)
	tp.poolSerialNumbersHashList = make(map[common.Hash][]common.Hash)
	tp.poolSerialNumberHash = make(map[common.Hash]common.Hash)
	tp.poolCandidate = make(map[common.Hash]string)
	tp.poolRequestStopStaking = make(map[common.Hash]string)
	if len(tp.pool) == 0 && len(tp.poolSerialNumbersHashList) == 0 && len(tp.poolSerialNumberHash) == 0 && len(tp.poolCandidate) == 0 && len(tp.poolRequestStopStaking) == 0 {
		return true
	}
	return false
}

func (tp *TxPool) calPoolSize() uint64 {
	var totalSize uint64
	for _, txDesc := range tp.pool {
		size := txDesc.Desc.Tx.GetTxActualSize()
		totalSize += size
	}
	return totalSize
}

// ----------- transaction.MempoolRetriever's implementation -----------------
func (tp TxPool) GetSerialNumbersHashH() map[common.Hash][]common.Hash {
	//tp.mtx.RLock()
	//defer tp.mtx.RUnlock()
	m := make(map[common.Hash][]common.Hash)
	for k, hashList := range tp.poolSerialNumbersHashList {
		m[k] = []common.Hash{}
		for _, v := range hashList {
			m[k] = append(m[k], v)
		}
	}
	return m
}

func (tp TxPool) GetTxsInMem() map[common.Hash]metadata.TxDesc {
	//tp.mtx.RLock()
	//defer tp.mtx.RUnlock()
	txsInMem := make(map[common.Hash]metadata.TxDesc)
	for hash, txDesc := range tp.pool {
		txsInMem[hash] = txDesc.Desc
	}
	return txsInMem
}

// ----------- end of transaction.MempoolRetriever's implementation -----------------
