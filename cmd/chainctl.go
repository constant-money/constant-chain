package main

import (
	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/consensus"
	"github.com/incognitochain/incognito-chain/dataaccessobject"
	"github.com/incognitochain/incognito-chain/peerv2"
	"github.com/incognitochain/incognito-chain/trie"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incdb"
	_ "github.com/incognitochain/incognito-chain/incdb/lvdb"
	"github.com/incognitochain/incognito-chain/mempool"
	"github.com/incognitochain/incognito-chain/pubsub"
)

func makeBlockChain(databaseDir string, testNet bool) (*blockchain.BlockChain, error) {
	blockchain.Logger.Init(common.NewBackend(nil).Logger("ChainCMD", true))
	blockchain.BLogger.Init(common.NewBackend(nil).Logger("ChainCMD", true))
	mempool.Logger.Init(common.NewBackend(nil).Logger("ChainCMD", true))
	dataaccessobject.Logger.Init(common.NewBackend(nil).Logger("ChainCMD", true))
	trie.Logger.Init(common.NewBackend(nil).Logger("ChainCMD", true))
	db, err := incdb.Open("leveldb", filepath.Join(databaseDir))
	if err != nil {
		return nil, err
	}
	log.Printf("Open leveldb at %+v successfully", filepath.Join(databaseDir))
	bc := blockchain.NewBlockChain(&blockchain.Config{}, false)
	var bcParams *blockchain.Params
	if testNet {
		bcParams = &blockchain.ChainTestParam
	} else {
		bcParams = &blockchain.ChainMainParam
	}
	pb := pubsub.NewPubSubManager()
	txPool := &mempool.TxPool{}
	txPool.Init(&mempool.Config{
		PubSubManager: pb,
		DataBase:      db,
		BlockChain:    bc,
		ChainParams:   bcParams,
	})
	err = bc.Init(&blockchain.Config{
		ChainParams:     bcParams,
		DataBase:        db,
		PubSubManager:   pb,
		TxPool:          txPool,
		ConsensusEngine: &consensus.Engine{},
		Highway:         &peerv2.ConnManager{},
	})
	if err != nil {
		return nil, err
	}
	return bc, nil
}

//default chainDataDir is data/testnet/block
func backupShardChain(bc *blockchain.BlockChain, shardID byte, outDatadir string, fileName string) error {
	if fileName == "" {
		fileName = "export-incognito-shard-" + strconv.Itoa(int(shardID))
	}
	if outDatadir == "" {
		outDatadir = "./"
	}
	file := filepath.Join(outDatadir, fileName)
	fileHandler, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fileHandler.Close()
	var writer io.Writer = fileHandler
	if err := bc.BackupShardChain(writer, shardID); err != nil {
		return err
	}
	log.Printf("Backup Shard %+v Chain, file %+v", shardID, file)
	return nil
}

func backupBeaconChain(bc *blockchain.BlockChain, outDatadir string, fileName string) error {
	if fileName == "" {
		fileName = "export-incognito-beacon"
	}
	if outDatadir == "" {
		outDatadir = "./"
	}
	file := filepath.Join(outDatadir, fileName)
	fileHandler, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fileHandler.Close()
	var writer io.Writer = fileHandler
	if err := bc.BackupBeaconChain(writer); err != nil {
		return err
	}
	log.Printf("Backup Beacon Chain, file %+v", file)
	return nil
}

func RestoreShardChain(bc *blockchain.BlockChain, filename string) error {
	var shardID byte
	// Watch for Ctrl-C while the import is running.
	// If a signal is received, the import will stop at the next batch.
	interrupt := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	defer close(interrupt)
	go func() {
		if _, ok := <-interrupt; ok {
			log.Println("Interrupted during import, stopping at next block")
		}
		close(stop)
	}()
	checkInterrupt := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}
	log.Println("Importing blockchain", "file", filename)
	// Open the file handle and potentially unwrap the gzip stream
	fileHanlder, err := os.Open(filename)
	if err != nil {
		return err
	}
	log.Println(fileHanlder.Name())
	defer fileHanlder.Close()
	var reader io.Reader = fileHanlder
	for {
		numberOfByteToRead := make([]byte, 8)
		_, err := reader.Read(numberOfByteToRead)
		if err == io.EOF {
			break
		} else {
			if err != nil {
				return err
			}
		}
		blockLength, err := blockchain.GetNumberOfByteToRead(numberOfByteToRead)
		if err != nil {
			return err
		}
		blockBytes := make([]byte, blockLength)
		_, err = reader.Read(blockBytes)
		if err == io.EOF {
			break
		} else {
			if err != nil {
				return err
			}
		}
		block := &types.ShardBlock{}
		err = block.UnmarshalJSON(blockBytes)
		if err != nil {
			return err
		}
		if bc.GetBestStateShard(block.Header.ShardID).ShardHeight >= block.Header.Height {
			continue
		}
		if block.Header.Height%100 == 0 {
			log.Printf("Restore Shard %+v Block %+v \n", block.Header.ShardID, block.Header.Height)
		}
		err = bc.InsertShardBlock(block, true)
		if bcErr, ok := err.(*blockchain.BlockChainError); ok {
			if bcErr.Code == blockchain.ErrCodeMessage[blockchain.DuplicateShardBlockError].Code {
				continue
			}
		}
		if err != nil {
			return err
		}
		// check interupt whenever finish insert 1 block
		shardID = block.Header.ShardID
		checkInterrupt()
	}
	log.Printf("Restore Shard %+v Chain Successfully", shardID)
	return nil
}

func restoreBeaconChain(bc *blockchain.BlockChain, filename string) error {
	// Watch for Ctrl-C while the import is running.
	// If a signal is received, the import will stop at the next batch.
	interrupt := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	defer close(interrupt)
	go func() {
		if _, ok := <-interrupt; ok {
			log.Println("Interrupted during import, stopping at next block")
		}
		close(stop)
	}()
	checkInterrupt := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}
	log.Println("Importing blockchain", "file", filename)
	// Open the file handle and potentially unwrap the gzip stream
	fh, err := os.Open(filename)
	if err != nil {
		return err
	}
	log.Println(fh.Name())
	defer fh.Close()
	var reader io.Reader = fh
	for {
		numberOfByteToRead := make([]byte, 8)
		_, err := reader.Read(numberOfByteToRead)
		if err == io.EOF {
			break
		} else {
			if err != nil {
				return err
			}
		}
		blockLength, err := blockchain.GetNumberOfByteToRead(numberOfByteToRead)
		if err != nil {
			return err
		}
		blockBytes := make([]byte, blockLength)
		_, err = reader.Read(blockBytes)
		if err == io.EOF {
			break
		} else {
			if err != nil {
				return err
			}
		}
		block := &types.BeaconBlock{}
		err = block.UnmarshalJSON(blockBytes)
		if err != nil {
			return err
		}
		if bc.GetBeaconBestState().BeaconHeight >= block.Header.Height {
			continue
		}
		if block.Header.Height%100 == 0 {
			log.Printf("Restore Block %+v \n", block.Header.Height)
		}
		if block.Header.Height == 1 {
			continue
		}
		err = bc.InsertBeaconBlock(block, true)
		if bcErr, ok := err.(*blockchain.BlockChainError); ok {
			if bcErr.Code == blockchain.ErrCodeMessage[blockchain.DuplicateShardBlockError].Code {
				continue
			}
		}
		if err != nil {
			return err
		}
		// check interupt whenever finish insert 1 block
		checkInterrupt()
	}
	log.Println("Restore Beacon Chain Successfully")
	return nil
}
