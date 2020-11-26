package blsmultisig

import (
	"errors"
	"fmt"
	"sync"
	"time"

	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"github.com/incognitochain/incognito-chain/common/base58"
)

type memoryCache struct {
	db      map[string]bn256.G2
	expired map[string]time.Time
	lock    sync.RWMutex
}

// New returns a wrapped map with all the required database interface methods
// implemented.
func New() *memoryCache {
	return &memoryCache{
		db:      make(map[string]bn256.G2),
		expired: make(map[string]time.Time),
	}
}

// Close deallocates the internal map and ensures any consecutive data access op
// failes with an error.
func (db *memoryCache) close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db = nil
	return nil
}

// Get retrieves the given key if it's present in the key-value store.
func (db *memoryCache) get(key []byte) (bn256.G2, error) {
	db.lock.RLock()
	//defer db.lock.RUnlock()

	if db.db == nil {
		db.lock.RUnlock()
		return bn256.G2{}, errors.New("DB close")
	}
	keyStr := base58.Base58Check{}.Encode(key, 0x0)
	if entry, ok := db.db[keyStr]; ok {
		// check expired time
		if expired, ok1 := db.expired[keyStr]; ok1 {
			if expired.Before(time.Now()) {
				// is expired
				db.lock.RUnlock()
				db.delete(key)
				return bn256.G2{}, NewBLSSignatureError(MemCacheErr, fmt.Errorf("Key %s expired", keyStr))
			}
		}
		db.lock.RUnlock()
		return entry, nil
	}
	db.lock.RUnlock()
	return bn256.G2{}, NewBLSSignatureError(MemCacheErr, fmt.Errorf("Key %s not found", keyStr))
}

// Delete removes the key from the key-value store.
func (db *memoryCache) delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return NewBLSSignatureError(MemCacheErr, errors.New("DB close"))
	}
	keyStr := base58.Base58Check{}.Encode(key, 0x0)
	delete(db.db, keyStr)
	return nil
}

func (db *memoryCache) put(key []byte, value bn256.G2) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return NewBLSSignatureError(MemCacheErr, errors.New("DB close"))
	}
	keyStr := base58.Base58Check{}.Encode(key, 0x0)
	db.db[keyStr] = value
	return nil
}
