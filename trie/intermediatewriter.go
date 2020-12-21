package trie

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incdb"
)

// secureKeyPrefix is the database key prefix used to store trie node preimages.
var secureKeyPrefix = []byte("secure-key-")

// secureKeyLength is the length of the above prefix + 32byte hash.
const secureKeyLength = 11 + 32

// IntermediateWriter is an intermediate write layer between the trie data structures and
// the disk database. The aim is to accumulate trie writes in-memory and only
// periodically flush a couple tries to disk, garbage collecting the remainder.
//
// Note, the trie Database is **not** thread safe in its mutations, but it **is**
// thread safe in providing individual, independent node access. The rationale
// behind this split design is to provide read access to RPC handlers and sync
// servers even while the trie is executing expensive garbage collection.
type IntermediateWriter struct {
	diskdb incdb.Database // Persistent storage for matured trie nodes

	cleans  *bigcache.BigCache          // GC friendly memory cache of clean node RLPs
	dirties map[common.Hash]*cachedNode // Data and references relationships of dirty nodes
	oldest  common.Hash                 // Oldest tracked node, flush-list head
	newest  common.Hash                 // Newest tracked node, flush-list tail

	preimages map[common.Hash][]byte // Preimages of nodes from the secure trie
	seckeybuf [secureKeyLength]byte  // Ephemeral buffer for calculating preimage keys

	gctime  time.Duration      // Time spent on garbage collection since last commit
	gcnodes uint64             // Nodes garbage collected since last commit
	gcsize  common.StorageSize // Data storage garbage collected since last commit

	flushtime  time.Duration      // Time spent on data flushing since last commit
	flushnodes uint64             // Nodes flushed since last commit
	flushsize  common.StorageSize // Data storage flushed since last commit

	dirtiesSize   common.StorageSize // Storage size of the dirty node cache (exc. metadata)
	childrenSize  common.StorageSize // Storage size of the external children tracking
	preimagesSize common.StorageSize // Storage size of the preimages cache

	lock sync.RWMutex
}

// rawNode is a simple binary blob used to differentiate between collapsed trie
// nodes and already encoded RLP binary blobs (while at the same time store them
// in the same cache fields).
type rawNode []byte

func (n rawNode) canUnload(uint16, uint16) bool { panic("this should never end up in a live trie") }
func (n rawNode) cache() (hashNode, bool)       { panic("this should never end up in a live trie") }
func (n rawNode) fstring(ind string) string     { panic("this should never end up in a live trie") }

// rawFullNode represents only the useful data content of a full node, with the
// caches and flags stripped out to minimize its data storage. This type honors
// the same RLP encoding as the original parent.
type rawFullNode [17]node

func (n rawFullNode) canUnload(uint16, uint16) bool { panic("this should never end up in a live trie") }
func (n rawFullNode) cache() (hashNode, bool)       { panic("this should never end up in a live trie") }
func (n rawFullNode) fstring(ind string) string     { panic("this should never end up in a live trie") }

func (n rawFullNode) EncodeRLP(w io.Writer) error {
	var nodes [17]node

	for i, child := range n {
		if child != nil {
			nodes[i] = child
		} else {
			nodes[i] = nilValueNode
		}
	}
	return rlp.Encode(w, nodes)
}

// rawShortNode represents only the useful data content of a short node, with the
// caches and flags stripped out to minimize its data storage. This type honors
// the same RLP encoding as the original parent.
type rawShortNode struct {
	Key []byte
	Val node
}

func (n rawShortNode) canUnload(uint16, uint16) bool { panic("this should never end up in a live trie") }
func (n rawShortNode) cache() (hashNode, bool)       { panic("this should never end up in a live trie") }
func (n rawShortNode) fstring(ind string) string     { panic("this should never end up in a live trie") }

// cachedNode is all the information we know about a single cached node in the
// memory database write layer.
type cachedNode struct {
	node node   // Cached collapsed trie node, or raw rlp data
	size uint16 // Byte size of the useful cached data

	parents  uint32                 // Number of live nodes referencing this one
	children map[common.Hash]uint16 // External children referenced by this node

	flushPrev common.Hash // Previous node in the flush-list
	flushNext common.Hash // Next node in the flush-list
}

// cachedNodeSize is the raw size of a cachedNode data structure without any
// node data included. It's an approximate size, but should be a lot better
// than not counting them.
var cachedNodeSize = int(reflect.TypeOf(cachedNode{}).Size())

// cachedNodeChildrenSize is the raw size of an initialized but empty external
// reference map.
const cachedNodeChildrenSize = 48

// rlp returns the raw rlp encoded blob of the cached node, either directly from
// the cache, or by regenerating it from the collapsed node.
func (n *cachedNode) rlp() []byte {
	if node, ok := n.node.(rawNode); ok {
		return node
	}
	blob, err := rlp.EncodeToBytes(n.node)
	if err != nil {
		panic(err)
	}
	return blob
}

// obj returns the decoded and expanded trie node, either directly from the cache,
// or by regenerating it from the rlp encoded blob.
func (n *cachedNode) obj(hash common.Hash) node {
	if node, ok := n.node.(rawNode); ok {
		return mustDecodeNode(hash[:], node)
	}
	return expandNode(hash[:], n.node)
}

// childs returns all the tracked children of this node, both the implicit ones
// from inside the node as well as the explicit ones from outside the node.
func (n *cachedNode) childs() []common.Hash {
	children := make([]common.Hash, 0, 16)
	for child := range n.children {
		children = append(children, child)
	}
	if _, ok := n.node.(rawNode); !ok {
		gatherChildren(n.node, &children)
	}
	return children
}

// gatherChildren traverses the node hierarchy of a collapsed storage node and
// retrieves all the hashnode children.
func gatherChildren(n node, children *[]common.Hash) {
	switch n := n.(type) {
	case *rawShortNode:
		gatherChildren(n.Val, children)

	case rawFullNode:
		for i := 0; i < 16; i++ {
			gatherChildren(n[i], children)
		}
	case hashNode:
		*children = append(*children, common.BytesToHash(n))

	case valueNode, nil:

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// simplifyNode traverses the hierarchy of an expanded memory node and discards
// all the internal caches, returning a node that only contains the raw data.
func simplifyNode(n node) node {
	switch n := n.(type) {
	case *shortNode:
		// Short nodes discard the flags and cascade
		return &rawShortNode{Key: n.Key, Val: simplifyNode(n.Val)}

	case *fullNode:
		// Full nodes discard the flags and cascade
		node := rawFullNode(n.Children)
		for i := 0; i < len(node); i++ {
			if node[i] != nil {
				node[i] = simplifyNode(node[i])
			}
		}
		return node

	case valueNode, hashNode, rawNode:
		return n

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// expandNode traverses the node hierarchy of a collapsed storage node and converts
// all fields and keys into expanded memory form.
func expandNode(hash hashNode, n node) node {
	switch n := n.(type) {
	case *rawShortNode:
		// Short nodes need key and child expansion
		return &shortNode{
			Key: compactToHex(n.Key),
			Val: expandNode(nil, n.Val),
			flags: nodeFlag{
				hash: hash,
			},
		}

	case rawFullNode:
		// Full nodes need child expansion
		node := &fullNode{
			flags: nodeFlag{
				hash: hash,
			},
		}
		for i := 0; i < len(node.Children); i++ {
			if n[i] != nil {
				node.Children[i] = expandNode(nil, n[i])
			}
		}
		return node

	case valueNode, hashNode:
		return n

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// trienodeHasher is a struct to be used with BigCache, which uses a Hasher to
// determine which shard to place an entry into. It's not a cryptographic hash,
// just to provide a bit of anti-collision (default is FNV64a).
//
// Since trie keys are already hashes, we can just use the key directly to
// map shard id.
type trienodeHasher struct{}

// Sum64 implements the bigcache.Hasher interface.
func (t trienodeHasher) Sum64(key string) uint64 {
	return binary.BigEndian.Uint64([]byte(key))
}

// NewIntermediateWriter creates a new trie database to store ephemeral trie content before
// its written out to disk or garbage collected. No read cache is created, so all
// data retrievals will hit the underlying disk database.
func NewIntermediateWriter(diskdb incdb.Database) *IntermediateWriter {
	return NewDatabaseWithCache(diskdb, 0)
}

// NewDatabaseWithCache creates a new trie database to store ephemeral trie content
// before its written out to disk or garbage collected. It also acts as a read cache
// for nodes loaded from disk.
func NewDatabaseWithCache(diskdb incdb.Database, cache int) *IntermediateWriter {
	var cleans *bigcache.BigCache
	if cache > 0 {
		cleans, _ = bigcache.NewBigCache(bigcache.Config{
			Shards:             1024,
			LifeWindow:         time.Hour,
			MaxEntriesInWindow: cache * 1024,
			MaxEntrySize:       512,
			HardMaxCacheSize:   cache,
			Hasher:             trienodeHasher{},
		})
	}
	return &IntermediateWriter{
		diskdb: diskdb,
		cleans: cleans,
		dirties: map[common.Hash]*cachedNode{{}: {
			children: make(map[common.Hash]uint16),
		}},
		preimages: make(map[common.Hash][]byte),
	}
}

// DiskDB retrieves the persistent storage backing the trie database.
func (intermediateWriter *IntermediateWriter) DiskDB() incdb.KeyValueReader {
	return intermediateWriter.diskdb
}

// InsertBlob writes a new reference tracked blob to the memory database if it's
// yet unknown. This method should only be used for non-trie nodes that require
// reference counting, since trie nodes are garbage collected directly through
// their embedded children.
func (intermediateWriter *IntermediateWriter) InsertBlob(hash common.Hash, blob []byte) {
	intermediateWriter.lock.Lock()
	defer intermediateWriter.lock.Unlock()

	intermediateWriter.insert(hash, blob, rawNode(blob))
}

// insert inserts a collapsed trie node into the memory database. This method is
// a more generic version of InsertBlob, supporting both raw blob insertions as
// well ex trie node insertions. The blob must always be specified to allow proper
// size tracking.
func (intermediateWriter *IntermediateWriter) insert(hash common.Hash, blob []byte, node node) {
	// If the node's already cached, skip
	if _, ok := intermediateWriter.dirties[hash]; ok {
		return
	}
	// Create the cached entry for this node
	entry := &cachedNode{
		node:      simplifyNode(node),
		size:      uint16(len(blob)),
		flushPrev: intermediateWriter.newest,
	}
	for _, child := range entry.childs() {
		if c := intermediateWriter.dirties[child]; c != nil {
			c.parents++
		}
	}
	intermediateWriter.dirties[hash] = entry

	// Update the flush-list endpoints
	if intermediateWriter.oldest == (common.Hash{}) {
		intermediateWriter.oldest, intermediateWriter.newest = hash, hash
	} else {
		intermediateWriter.dirties[intermediateWriter.newest].flushNext, intermediateWriter.newest = hash, hash
	}
	intermediateWriter.dirtiesSize += common.StorageSize(common.HashSize + entry.size)
}

// insertPreimage writes a new trie node pre-image to the memory database if it's
// yet unknown. The method will make a copy of the slice.
//
// Note, this method assumes that the database's lock is held!
func (intermediateWriter *IntermediateWriter) insertPreimage(hash common.Hash, preimage []byte) {
	if _, ok := intermediateWriter.preimages[hash]; ok {
		return
	}
	intermediateWriter.preimages[hash] = common.CopyBytes(preimage)
	intermediateWriter.preimagesSize += common.StorageSize(common.HashSize + len(preimage))
}

// node retrieves a cached trie node from memory, or returns nil if none can be
// found in the memory cache.
func (intermediateWriter *IntermediateWriter) node(hash common.Hash) node {
	// Retrieve the node from the clean cache if available
	if intermediateWriter.cleans != nil {
		if enc, err := intermediateWriter.cleans.Get(string(hash[:])); err == nil && enc != nil {
			//memcacheCleanHitMeter.Mark(1)
			//memcacheCleanReadMeter.Mark(int64(len(enc)))
			return mustDecodeNode(hash[:], enc)
		}
	}
	// Retrieve the node from the dirty cache if available
	intermediateWriter.lock.RLock()
	dirty := intermediateWriter.dirties[hash]
	intermediateWriter.lock.RUnlock()

	if dirty != nil {
		return dirty.obj(hash)
	}
	// Content unavailable in memory, attempt to retrieve from disk
	enc, err := intermediateWriter.diskdb.Get(hash[:])
	if err != nil || enc == nil {
		return nil
	}
	if intermediateWriter.cleans != nil {
		intermediateWriter.cleans.Set(string(hash[:]), enc)
		//memcacheCleanMissMeter.Mark(1)
		//memcacheCleanWriteMeter.Mark(int64(len(enc)))
	}
	return mustDecodeNode(hash[:], enc)
}

// Node retrieves an encoded cached trie node from memory. If it cannot be found
// cached, the method queries the persistent database for the content.
func (intermediateWriter *IntermediateWriter) Node(hash common.Hash) ([]byte, error) {
	// It doens't make sense to retrieve the metaroot
	if hash == (common.Hash{}) {
		return nil, errors.New("not found")
	}
	// Retrieve the node from the clean cache if available
	if intermediateWriter.cleans != nil {
		if enc, err := intermediateWriter.cleans.Get(string(hash[:])); err == nil && enc != nil {
			//memcacheCleanHitMeter.Mark(1)
			//memcacheCleanReadMeter.Mark(int64(len(enc)))
			return enc, nil
		}
	}
	// Retrieve the node from the dirty cache if available
	intermediateWriter.lock.RLock()
	dirty := intermediateWriter.dirties[hash]
	intermediateWriter.lock.RUnlock()

	if dirty != nil {
		return dirty.rlp(), nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	enc, err := intermediateWriter.diskdb.Get(hash[:])
	if err == nil && enc != nil {
		if intermediateWriter.cleans != nil {
			//memcacheCleanMissMeter.Mark(1)
			//memcacheCleanWriteMeter.Mark(int64(len(enc)))
			intermediateWriter.cleans.Set(string(hash[:]), enc)
		}
	}
	return enc, err
}

// preimage retrieves a cached trie node pre-image from memory. If it cannot be
// found cached, the method queries the persistent database for the content.
func (intermediateWriter *IntermediateWriter) preimage(hash common.Hash) ([]byte, error) {
	// Retrieve the node from cache if available
	intermediateWriter.lock.RLock()
	preimage := intermediateWriter.preimages[hash]
	intermediateWriter.lock.RUnlock()

	if preimage != nil {
		return preimage, nil
	}
	// Content unavailable in memory, attempt to retrieve from disk
	return intermediateWriter.diskdb.Get(intermediateWriter.secureKey(hash[:]))
}

// secureKey returns the database key for the preimage of key, as an ephemeral
// buffer. The caller must not hold onto the return value because it will become
// invalid on the next call.
func (intermediateWriter *IntermediateWriter) secureKey(key []byte) []byte {
	buf := append(intermediateWriter.seckeybuf[:0], secureKeyPrefix...)
	buf = append(buf, key...)
	return buf
}

// Nodes retrieves the hashes of all the nodes cached within the memory database.
// This method is extremely expensive and should only be used to validate internal
// states in test code.
func (intermediateWriter *IntermediateWriter) Nodes() []common.Hash {
	intermediateWriter.lock.RLock()
	defer intermediateWriter.lock.RUnlock()

	var hashes = make([]common.Hash, 0, len(intermediateWriter.dirties))
	for hash := range intermediateWriter.dirties {
		if hash != (common.Hash{}) { // Special case for "root" references/nodes
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

// Reference adds a new reference from a parent node to a child node.
func (intermediateWriter *IntermediateWriter) Reference(child common.Hash, parent common.Hash) {
	intermediateWriter.lock.Lock()
	defer intermediateWriter.lock.Unlock()

	intermediateWriter.reference(child, parent)
}

// reference is the private locked version of Reference.
func (intermediateWriter *IntermediateWriter) reference(child common.Hash, parent common.Hash) {
	// If the node does not exist, it's a node pulled from disk, skip
	node, ok := intermediateWriter.dirties[child]
	if !ok {
		return
	}
	// If the reference already exists, only duplicate for roots
	if intermediateWriter.dirties[parent].children == nil {
		intermediateWriter.dirties[parent].children = make(map[common.Hash]uint16)
		intermediateWriter.childrenSize += cachedNodeChildrenSize
	} else if _, ok = intermediateWriter.dirties[parent].children[child]; ok && parent != (common.Hash{}) {
		return
	}
	node.parents++
	intermediateWriter.dirties[parent].children[child]++
	if intermediateWriter.dirties[parent].children[child] == 1 {
		intermediateWriter.childrenSize += common.HashSize + 2 // uint16 counter
	}
}

// Dereference removes an existing reference from a root node.
func (intermediateWriter *IntermediateWriter) Dereference(root common.Hash) {
	// Sanity check to ensure that the meta-root is not removed
	if root == (common.Hash{}) {
		Logger.log.Error("Attempted to dereference the trie cache meta root")
		return
	}
	intermediateWriter.lock.Lock()
	defer intermediateWriter.lock.Unlock()

	nodes, storage, start := len(intermediateWriter.dirties), intermediateWriter.dirtiesSize, time.Now()
	intermediateWriter.dereference(root, common.Hash{})

	intermediateWriter.gcnodes += uint64(nodes - len(intermediateWriter.dirties))
	intermediateWriter.gcsize += storage - intermediateWriter.dirtiesSize
	intermediateWriter.gctime += time.Since(start)

	//memcacheGCTimeTimer.Update(time.Since(start))
	//memcacheGCSizeMeter.Mark(int64(storage - iw.dirtiesSize))
	//memcacheGCNodesMeter.Mark(int64(nodes - len(iw.dirties)))

	Logger.log.Debug("Dereferenced trie from memory database", "nodes", nodes-len(intermediateWriter.dirties), "size", storage-intermediateWriter.dirtiesSize, "time", time.Since(start),
		"gcnodes", intermediateWriter.gcnodes, "gcsize", intermediateWriter.gcsize, "gctime", intermediateWriter.gctime, "livenodes", len(intermediateWriter.dirties), "livesize", intermediateWriter.dirtiesSize)
}

// dereference is the private locked version of Dereference.
func (intermediateWriter *IntermediateWriter) dereference(child common.Hash, parent common.Hash) {
	// Dereference the parent-child
	node := intermediateWriter.dirties[parent]

	if node.children != nil && node.children[child] > 0 {
		node.children[child]--
		if node.children[child] == 0 {
			delete(node.children, child)
			intermediateWriter.childrenSize -= (common.HashSize + 2) // uint16 counter
		}
	}
	// If the child does not exist, it's a previously committed node.
	node, ok := intermediateWriter.dirties[child]
	if !ok {
		return
	}
	// If there are no more references to the child, delete it and cascade
	if node.parents > 0 {
		// This is a special cornercase where a node loaded from disk (i.e. not in the
		// memcache any more) gets reinjected as a new node (short node split into full,
		// then reverted into short), causing a cached node to have no parents. That is
		// no problem in itself, but don't make maxint parents out of it.
		node.parents--
	}
	if node.parents == 0 {
		// Remove the node from the flush-list
		switch child {
		case intermediateWriter.oldest:
			intermediateWriter.oldest = node.flushNext
			intermediateWriter.dirties[node.flushNext].flushPrev = common.Hash{}
		case intermediateWriter.newest:
			intermediateWriter.newest = node.flushPrev
			intermediateWriter.dirties[node.flushPrev].flushNext = common.Hash{}
		default:
			intermediateWriter.dirties[node.flushPrev].flushNext = node.flushNext
			intermediateWriter.dirties[node.flushNext].flushPrev = node.flushPrev
		}
		// Dereference all children and delete the node
		for _, hash := range node.childs() {
			intermediateWriter.dereference(hash, child)
		}
		delete(intermediateWriter.dirties, child)
		intermediateWriter.dirtiesSize -= common.StorageSize(common.HashSize + int(node.size))
		if node.children != nil {
			intermediateWriter.childrenSize -= cachedNodeChildrenSize
		}
	}
}

// Cap iteratively flushes old but still referenced trie nodes until the total
// memory usage goes below the given threshold.
//
// Note, this method is a non-synchronized mutator. It is unsafe to call this
// concurrently with other mutators.
func (intermediateWriter *IntermediateWriter) Cap(limit common.StorageSize) error {
	// Create a database batch to flush persistent data out. It is important that
	// outside code doesn't see an inconsistent state (referenced data removed from
	// memory cache during commit but not yet in persistent storage). This is ensured
	// by only uncaching existing data when the database write finalizes.
	nodes, storage, start := len(intermediateWriter.dirties), intermediateWriter.dirtiesSize, time.Now()
	batch := intermediateWriter.diskdb.NewBatch()

	// iw.dirtiesSize only contains the useful data in the cache, but when reporting
	// the total memory consumption, the maintenance metadata is also needed to be
	// counted.
	size := intermediateWriter.dirtiesSize + common.StorageSize((len(intermediateWriter.dirties)-1)*cachedNodeSize)
	size += intermediateWriter.childrenSize - common.StorageSize(len(intermediateWriter.dirties[common.Hash{}].children)*(common.HashSize+2))

	// If the preimage cache got large enough, push to disk. If it's still small
	// leave for later to deduplicate writes.
	flushPreimages := intermediateWriter.preimagesSize > 4*1024*1024
	if flushPreimages {
		for hash, preimage := range intermediateWriter.preimages {
			if err := batch.Put(intermediateWriter.secureKey(hash[:]), preimage); err != nil {
				Logger.log.Error("Failed to commit preimage from trie database", "err", err)
				return err
			}
			if batch.ValueSize() > incdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					return err
				}
				batch.Reset()
			}
		}
	}
	// Keep committing nodes from the flush-list until we're below allowance
	oldest := intermediateWriter.oldest
	for size > limit && oldest != (common.Hash{}) {
		// Fetch the oldest referenced node and push into the batch
		node := intermediateWriter.dirties[oldest]
		if err := batch.Put(oldest[:], node.rlp()); err != nil {
			return err
		}
		// If we exceeded the ideal batch size, commit and reset
		if batch.ValueSize() >= incdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				Logger.log.Error("Failed to write flush list to disk", "err", err)
				return err
			}
			batch.Reset()
		}
		// Iterate to the next flush item, or abort if the size cap was achieved. Size
		// is the total size, including the useful cached data (hash -> blob), the
		// cache item metadata, as well as external children mappings.
		size -= common.StorageSize(common.HashSize + int(node.size) + cachedNodeSize)
		if node.children != nil {
			size -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashSize+2))
		}
		oldest = node.flushNext
	}
	// Flush out any remainder data from the last batch
	if err := batch.Write(); err != nil {
		Logger.log.Error("Failed to write flush list to disk", "err", err)
		return err
	}
	// Write successful, clear out the flushed data
	intermediateWriter.lock.Lock()
	defer intermediateWriter.lock.Unlock()

	if flushPreimages {
		intermediateWriter.preimages = make(map[common.Hash][]byte)
		intermediateWriter.preimagesSize = 0
	}
	for intermediateWriter.oldest != oldest {
		node := intermediateWriter.dirties[intermediateWriter.oldest]
		delete(intermediateWriter.dirties, intermediateWriter.oldest)
		intermediateWriter.oldest = node.flushNext

		intermediateWriter.dirtiesSize -= common.StorageSize(common.HashSize + int(node.size))
		if node.children != nil {
			intermediateWriter.childrenSize -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashSize+2))
		}
	}
	if intermediateWriter.oldest != (common.Hash{}) {
		intermediateWriter.dirties[intermediateWriter.oldest].flushPrev = common.Hash{}
	}
	intermediateWriter.flushnodes += uint64(nodes - len(intermediateWriter.dirties))
	intermediateWriter.flushsize += storage - intermediateWriter.dirtiesSize
	intermediateWriter.flushtime += time.Since(start)

	//memcacheFlushTimeTimer.Update(time.Since(start))
	//memcacheFlushSizeMeter.Mark(int64(storage - iw.dirtiesSize))
	//memcacheFlushNodesMeter.Mark(int64(nodes - len(iw.dirties)))

	Logger.log.Debug("Persisted nodes from memory database", "nodes", nodes-len(intermediateWriter.dirties), "size", storage-intermediateWriter.dirtiesSize, "time", time.Since(start),
		"flushnodes", intermediateWriter.flushnodes, "flushsize", intermediateWriter.flushsize, "flushtime", intermediateWriter.flushtime, "livenodes", len(intermediateWriter.dirties), "livesize", intermediateWriter.dirtiesSize)

	return nil
}

// Commit iterates over all the children of a particular node, writes them out
// to disk, forcefully tearing down all references in both directions. As a side
// effect, all pre-images accumulated up to this point are also written.
//
// Note, this method is a non-synchronized mutator. It is unsafe to call this
// concurrently with other mutators.
func (intermediateWriter *IntermediateWriter) Commit(node common.Hash, report bool) error {
	// Create a database batch to flush persistent data out. It is important that
	// outside code doesn't see an inconsistent state (referenced data removed from
	// memory cache during commit but not yet in persistent storage). This is ensured
	// by only uncaching existing data when the database write finalizes.
	//start := time.Now()
	batch := intermediateWriter.diskdb.NewBatch()

	// Move all of the accumulated preimages into a write batch
	for hash, preimage := range intermediateWriter.preimages {
		if err := batch.Put(intermediateWriter.secureKey(hash[:]), preimage); err != nil {
			Logger.log.Error("Failed to commit preimage from trie database", "err", err)
			return err
		}
		// If the batch is too large, flush to disk
		if batch.ValueSize() > incdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return err
			}
			batch.Reset()
		}
	}
	// Since we're going to replay trie node writes into the clean cache, flush out
	// any batched pre-images before continuing.
	if err := batch.Write(); err != nil {
		return err
	}
	batch.Reset()

	// Move the trie itself into the batch, flushing if enough data is accumulated
	//nodes, storage := len(intermediateWriter.dirties), intermediateWriter.dirtiesSize

	uncacher := &cleaner{intermediateWriter}
	if err := intermediateWriter.commit(node, batch, uncacher); err != nil {
		Logger.log.Error("Failed to commit trie from trie database", "err", err)
		return err
	}
	// Trie mostly committed to disk, flush any batch leftovers
	if err := batch.Write(); err != nil {
		Logger.log.Error("Failed to write trie to disk", "err", err)
		return err
	}
	// Uncache any leftovers in the last batch
	intermediateWriter.lock.Lock()
	defer intermediateWriter.lock.Unlock()

	batch.Replay(uncacher)
	batch.Reset()

	// Reset the storage counters and bumpd metrics
	intermediateWriter.preimages = make(map[common.Hash][]byte)
	intermediateWriter.preimagesSize = 0

	//memcacheCommitTimeTimer.Update(time.Since(start))
	//memcacheCommitSizeMeter.Mark(int64(storage - iw.dirtiesSize))
	//memcacheCommitNodesMeter.Mark(int64(nodes - len(iw.dirties)))

	//Logger.log.Info("Persisted trie from memory database", "nodes", nodes-len(intermediateWriter.dirties)+int(intermediateWriter.flushnodes), "size", storage-intermediateWriter.dirtiesSize+intermediateWriter.flushsize, "time", time.Since(start)+intermediateWriter.flushtime,
	//	"gcnodes", intermediateWriter.gcnodes, "gcsize", intermediateWriter.gcsize, "gctime", intermediateWriter.gctime, "livenodes", len(intermediateWriter.dirties), "livesize", intermediateWriter.dirtiesSize)

	// Reset the garbage collection statistics
	intermediateWriter.gcnodes, intermediateWriter.gcsize, intermediateWriter.gctime = 0, 0, 0
	intermediateWriter.flushnodes, intermediateWriter.flushsize, intermediateWriter.flushtime = 0, 0, 0

	return nil
}

// commit is the private locked version of Commit.
func (intermediateWriter *IntermediateWriter) commit(hash common.Hash, batch incdb.Batch, uncacher *cleaner) error {
	// If the node does not exist, it's a previously committed node
	node, ok := intermediateWriter.dirties[hash]
	if !ok {
		return nil
	}
	for _, child := range node.childs() {
		if err := intermediateWriter.commit(child, batch, uncacher); err != nil {
			return err
		}
	}
	if err := batch.Put(hash[:], node.rlp()); err != nil {
		return err
	}
	// If we've reached an optimal batch size, commit and start over
	if batch.ValueSize() >= incdb.IdealBatchSize {
		if err := batch.Write(); err != nil {
			return err
		}
		intermediateWriter.lock.Lock()
		batch.Replay(uncacher)
		batch.Reset()
		intermediateWriter.lock.Unlock()
	}
	return nil
}

// cleaner is a database batch replayer that takes a batch of write operations
// and cleans up the trie database from anything written to disk.
type cleaner struct {
	db *IntermediateWriter
}

// Put reacts to database writes and implements dirty data uncaching. This is the
// post-processing step of a commit operation where the already persisted trie is
// removed from the dirty cache and moved into the clean cache. The reason behind
// the two-phase commit is to ensure ensure data availability while moving from
// memory to disk.
func (c *cleaner) Put(key []byte, rlp []byte) error {
	hash := common.BytesToHash(key)

	// If the node does not exist, we're done on this path
	node, ok := c.db.dirties[hash]
	if !ok {
		return nil
	}
	// Node still exists, remove it from the flush-list
	switch hash {
	case c.db.oldest:
		c.db.oldest = node.flushNext
		c.db.dirties[node.flushNext].flushPrev = common.Hash{}
	case c.db.newest:
		c.db.newest = node.flushPrev
		c.db.dirties[node.flushPrev].flushNext = common.Hash{}
	default:
		c.db.dirties[node.flushPrev].flushNext = node.flushNext
		c.db.dirties[node.flushNext].flushPrev = node.flushPrev
	}
	// Remove the node from the dirty cache
	delete(c.db.dirties, hash)
	c.db.dirtiesSize -= common.StorageSize(common.HashSize + int(node.size))
	if node.children != nil {
		c.db.dirtiesSize -= common.StorageSize(cachedNodeChildrenSize + len(node.children)*(common.HashSize+2))
	}
	// Move the flushed node into the clean cache to prevent insta-reloads
	if c.db.cleans != nil {
		c.db.cleans.Set(string(hash[:]), rlp)
	}
	return nil
}

func (c *cleaner) Delete(key []byte) error {
	panic("Not implemented")
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (intermediateWriter *IntermediateWriter) Size() (common.StorageSize, common.StorageSize) {
	intermediateWriter.lock.RLock()
	defer intermediateWriter.lock.RUnlock()

	// iw.dirtiesSize only contains the useful data in the cache, but when reporting
	// the total memory consumption, the maintenance metadata is also needed to be
	// counted.
	var metadataSize = common.StorageSize((len(intermediateWriter.dirties) - 1) * cachedNodeSize)
	var metarootRefs = common.StorageSize(len(intermediateWriter.dirties[common.Hash{}].children) * (common.HashSize + 2))
	return intermediateWriter.dirtiesSize + intermediateWriter.childrenSize + metadataSize - metarootRefs, intermediateWriter.preimagesSize
}

// verifyIntegrity is a debug method to iterate over the entire trie stored in
// memory and check whether every node is reachable from the meta root. The goal
// is to find any errors that might cause memory leaks and or trie nodes to go
// missing.
//
// This IntermediateWriter is extremely CPU and memory intensive, only use when must.
func (intermediateWriter *IntermediateWriter) verifyIntegrity() {
	// Iterate over all the cached nodes and accumulate them into a set
	reachable := map[common.Hash]struct{}{{}: {}}

	for child := range intermediateWriter.dirties[common.Hash{}].children {
		intermediateWriter.accumulate(child, reachable)
	}
	// Find any unreachable but cached nodes
	var unreachable []string
	for hash, node := range intermediateWriter.dirties {
		if _, ok := reachable[hash]; !ok {
			unreachable = append(unreachable, fmt.Sprintf("%x: {Node: %v, Parents: %d, Prev: %x, Next: %x}",
				hash, node.node, node.parents, node.flushPrev, node.flushNext))
		}
	}
	if len(unreachable) != 0 {
		panic(fmt.Sprintf("trie cache memory leak: %v", unreachable))
	}
}

// accumulate iterates over the trie defined by hash and accumulates all the
// cached children found in memory.
func (intermediateWriter *IntermediateWriter) accumulate(hash common.Hash, reachable map[common.Hash]struct{}) {
	// Mark the node reachable if present in the memory cache
	node, ok := intermediateWriter.dirties[hash]
	if !ok {
		return
	}
	reachable[hash] = struct{}{}

	// Iterate over all the children and accumulate them too
	for _, child := range node.childs() {
		intermediateWriter.accumulate(child, reachable)
	}
}
