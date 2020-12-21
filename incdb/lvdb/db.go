package lvdb

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/incognitochain/incognito-chain/common"

	"github.com/pkg/errors"

	"github.com/incognitochain/incognito-chain/incdb"
	"github.com/syndtr/goleveldb/leveldb"
	lvdbErrors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type db struct {
	fn     string // filename for reporting
	dbPath string
	lvdb   *leveldb.DB
	lock   sync.RWMutex
}

func init() {
	driver := incdb.Driver{
		DbType: "leveldb",
		Open:   openDriver,
	}
	if err := incdb.RegisterDriver(driver); err != nil {
		panic("failed to register db driver")
	}
}

func openDriver(args ...interface{}) (incdb.Database, error) {
	if len(args) != 1 {
		return nil, errors.New("invalid arguments")
	}
	dbPath, ok := args[0].(string)
	if !ok {
		return nil, errors.New("expected db path")
	}
	return open(dbPath)
}

func open(dbPath string) (incdb.Database, error) {
	handles := 256
	cache := 8
	lvdb, err := leveldb.OpenFile(dbPath, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*lvdbErrors.ErrCorrupted); corrupted {
		lvdb, err = leveldb.RecoverFile(dbPath, nil)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "levelvdb.OpenFile %s", dbPath)
	}
	return &db{fn: dbPath, lvdb: lvdb, dbPath: dbPath}, nil
}
func (db *db) GetPath() string {
	return db.fn
}

func (db *db) Close() error {
	return errors.Wrap(db.lvdb.Close(), "db.lvdb.Close")
}

func (db *db) ReOpen() error {
	handles := 256
	cache := 8
	lvdb, err := leveldb.OpenFile(db.dbPath, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*lvdbErrors.ErrCorrupted); corrupted {
		lvdb, err = leveldb.RecoverFile(db.dbPath, nil)
	}
	if err != nil {
		return errors.Wrapf(err, "levelvdb.OpenFile %s", db.dbPath)
	}
	db.lvdb = lvdb
	return err
}

func (db *db) Has(key []byte) (bool, error) {
	ret, err := db.lvdb.Has(key, nil)
	if err != nil {
		return false, err
	}
	return ret, nil
}

func (db *db) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	value, err := db.lvdb.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (db *db) Put(key, value []byte) error {
	if err := db.lvdb.Put(key, value, nil); err != nil {
		return err
	}
	return nil
}

func (db *db) Delete(key []byte) error {
	err := db.lvdb.Delete(key, nil)
	if err != nil {
		return err
	}
	return nil
}

// NewBatch creates a write-only key-value store that buffers changes to its host
// database until a final write is called.
func (db *db) NewBatch() incdb.Batch {
	return &batch{
		db: db.lvdb,
		b:  new(leveldb.Batch),
	}
}

// NewIterator creates a binary-alphabetical iterator over the entire keyspace
// contained within the leveldb database.
func (db *db) NewIterator() incdb.Iterator {
	return db.lvdb.NewIterator(new(util.Range), nil)
}

// NewIteratorWithStart creates a binary-alphabetical iterator over a subset of
// database content starting at a particular initial key (or after, if it does
// not exist).
func (db *db) NewIteratorWithStart(start []byte) incdb.Iterator {
	return db.lvdb.NewIterator(&util.Range{Start: start}, nil)
}

// NewIteratorWithPrefix creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix.
func (db *db) NewIteratorWithPrefix(prefix []byte) incdb.Iterator {
	return db.lvdb.NewIterator(util.BytesPrefix(prefix), nil)
}

// Stat returns a particular internal stat of the database.
func (db *db) Stat(property string) (string, error) {
	return db.lvdb.GetProperty(property)
}

// Compact flattens the underlying data store for the given key range. In essence,
// deleted and overwritten versions are discarded, and the data is rearranged to
// reduce the cost of operations needed to access them.
//
// A nil start is treated as a key before all keys in the data store; a nil limit
// is treated as a key after all keys in the data store. If both is nil then it
// will compact entire data store.
func (db *db) Compact(start []byte, limit []byte) error {
	return db.lvdb.CompactRange(util.Range{Start: start, Limit: limit})
}

// Path returns the path to the database directory.
func (db *db) Path() string {
	return db.fn
}

// batch is a write-only leveldb batch that commits changes to its host database
// when Write is called. A batch cannot be used concurrently.
type batch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

// Put inserts the given value into the batch for later committing.
func (b *batch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

// Delete inserts the a key removal into the batch for later committing.
func (b *batch) Delete(key []byte) error {
	b.b.Delete(key)
	b.size++
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *batch) ValueSize() int {
	return b.size
}

// Write flushes any accumulated data to disk.
func (b *batch) Write() error {
	return b.db.Write(b.b, nil)
}

// Reset resets the batch for reuse.
func (b *batch) Reset() {
	b.b.Reset()
	b.size = 0
}

// Replay replays the batch contents.
func (b *batch) Replay(w incdb.KeyValueWriter) error {
	return b.b.Replay(&replayer{writer: w})
}

// replayer is a small wrapper to implement the correct replay methods.
type replayer struct {
	writer  incdb.KeyValueWriter
	failure error
}

// Put inserts the given value into the key-value data store.
func (r *replayer) Put(key, value []byte) {
	// If the replay already failed, stop executing ops
	if r.failure != nil {
		return
	}
	r.failure = r.writer.Put(key, value)
}

// Delete removes the key from the key-value data store.
func (r *replayer) Delete(key []byte) {
	// If the replay already failed, stop executing ops
	if r.failure != nil {
		return
	}
	r.failure = r.writer.Delete(key)
}

func (db *db) Batch(data []incdb.BatchData) leveldb.Batch {
	batch := new(leveldb.Batch)
	for _, v := range data {
		batch.Put(v.Key, v.Value)
	}
	return *batch
}

func (db *db) PreloadBackup(backupFile string) error {
	err := uncompress(backupFile, db.dbPath+"_")
	if err != nil {
		return err
	}

	fmt.Println("remove ", db.dbPath)
	err = os.RemoveAll(db.dbPath)
	if err != nil {
		return err
	}
	fmt.Println("rename ", db.dbPath)
	err = os.Rename(db.dbPath+"_", db.dbPath)
	if err != nil {
		return err
	}
	return nil
}

func (db *db) LatestBackup(path string) (int, string) {
	backupFolder := filepath.Join(db.dbPath, path)
	//fmt.Println("backupFolder", backupFolder)
	files, err := ioutil.ReadDir(backupFolder)
	if err != nil {
		return 0, ""
	}
	//fmt.Println("files", files)
	if len(files) == 0 {
		return 0, ""
	}
	latestBackupEpoch := 0
	//Get max epoch
	for _, file := range files {
		//fmt.Println("file", file.Name())
		epoch, err := strconv.Atoi(file.Name())
		if err != nil {
			return 0, ""
		}
		if epoch > latestBackupEpoch {
			latestBackupEpoch = epoch
		}
	}

	return latestBackupEpoch, fmt.Sprintf("%v/%v", backupFolder, latestBackupEpoch)
}

func (db *db) RemoveBackup(backupFile string) {
	backupFile = filepath.Join(db.dbPath, backupFile)
	os.Remove(backupFile)
}

func (db *db) Backup(backupFile string) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	backupFile = filepath.Join(db.dbPath, backupFile)
	fmt.Println("backupFile", backupFile)

	if err := os.MkdirAll(filepath.Dir(backupFile), 0700); err != nil {
		panic(err)
	}

	if err := db.Close(); err != nil {
		return err
	}

	err := common.CompressDatabase(db.dbPath, backupFile)
	if err != nil {
		return err
	}

	if err := db.ReOpen(); err != nil {
		panic(err)
	}

	if err := removeUnusedBackupDatabase(backupFile); err != nil {
		panic(err)
	}

	return nil
}

func (db *db) Clear() error {
	files, err := filepath.Glob(filepath.Join(db.dbPath, "*"))
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

//removeUnusedBackupDatabase ...
// for remove unused databases in backup folder
func removeUnusedBackupDatabase(filePath string) error {
	strs := strings.Split(filePath, "/")

	//Get latest epoch
	latestEpoch, err := strconv.Atoi(strs[len(strs)-1])
	if err != nil {
		return err
	}

	//Get path directory of this file
	path := filePath
	for i := len(path) - 1; i > -1; i-- {
		if path[i] != '/' {
			path = path[:len(path)-1]
		} else {
			log.Println(1)
			break
		}
	}

	//Get needed epoch to download
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return err
	}

	//Get file name and compare with latest epoch
	for _, file := range files {

		epoch, err := strconv.Atoi(file.Name())
		if err != nil {
			return err
		}

		if epoch != latestEpoch && epoch != latestEpoch-1 {
			name := path + "/" + file.Name()
			err = os.Remove(name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//Uncompress file from zip file
func uncompress(srcPath, desPath string) error {

	//uncompress write
	//Remove all old data
	// if err := os.RemoveAll(srcPath); err != nil {
	// 	panic(err)
	// }
	fmt.Println("start decompress", srcPath)
	if err := os.RemoveAll(desPath); err != nil {
		panic(err)
	}
	//Create new data
	if err := os.MkdirAll(desPath, 0700); err != nil {
		panic(err)
	}

	err := common.DecompressDatabaseBackup(srcPath, desPath)
	if err != nil {
		return err
	}

	fmt.Println("done decompress", desPath)
	return nil
}
