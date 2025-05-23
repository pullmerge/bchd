// Copyright (c) 2015-2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/gcash/bchd/txscript"

	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/database"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
)

const (
	// The utxo writes big amounts of data to the database.  In order to limit
	// the size of individual database transactions, it works in batches.

	// utxoBatchSizeEntries is the maximum number of utxo entries to be written
	// in a single transaction.
	utxoBatchSizeEntries = 200000

	// utxoBatchSizeBlocks is the maximum number of blocks to be processed in a
	// single transaction.
	utxoBatchSizeBlocks = 50

	// utxoFlushPeriodicThreshold is the threshold percentage at which a flush is
	// performed when the flush mode FlushPeriodic is used.
	utxoFlushPeriodicThreshold = 90
)

const (
	// This value is calculated by running the following on a 64-bit system:
	//   unsafe.Sizeof(UtxoEntry{}) before adding tokenData
	baseUtxoEntrySizeWithoutTokenData = 40

	// This value is calculated by running the following on a 64-bit system:
	//   unsafe.Sizeof(wire.TokenData{}) assuming commitment of length 40
	baseUtxoEntryTokenDataSize = 88
)

// txoFlags is a bitmask defining additional information and state for a
// transaction output in a utxo view.
type txoFlags uint8

const (
	// tfCoinBase indicates that a txout was contained in a coinbase tx.
	tfCoinBase txoFlags = 1 << iota

	// tfSpent indicates that a txout is spent.
	tfSpent

	// tfModified indicates that a txout has been modified since it was
	// loaded.
	tfModified

	// tfFresh indicates that the entry is fresh.  This means that the parent
	// view never saw this entry.  Note that tfFresh is a performance
	// optimization with which we can erase entries that are fully spent if we
	// know we do not need to commit them.  It is always safe to not mark
	// tfFresh if that condition is not guaranteed.
	tfFresh
)

// UtxoEntry houses details about an individual transaction output in a utxo
// view such as whether or not it was contained in a coinbase tx, the height of
// the block that contains the tx, whether or not it is spent, its public key
// script, and how much it pays.
type UtxoEntry struct {
	// NOTE: Additions, deletions, or modifications to the order of the
	// definitions in this struct should not be changed without considering
	// how it affects alignment on 64-bit platforms.  The current order is
	// specifically crafted to result in minimal padding.  There will be a
	// lot of these in memory, so a few extra bytes of padding adds up.
	// Any changes here should also be reflected in the memoryUsage() function.

	amount      int64
	pkScript    []byte // The public key script for the output.
	blockHeight int32  // Height of block containing tx.

	tokenData wire.TokenData // TODO this can use better memorry optimization. Check the paddings.

	// packedFlags contains additional info about output such as whether it
	// is a coinbase, whether it is spent, and whether it has been modified
	// since it was loaded.  This approach is used in order to reduce memory
	// usage since there will be a lot of these in memory.
	packedFlags txoFlags
}

// NewUtxoEntry returns a new UtxoEntry built from the arguments.
func NewUtxoEntry(txOut *wire.TxOut, blockHeight int32, isCoinbase bool) *UtxoEntry {
	var cbFlag txoFlags
	if isCoinbase {
		cbFlag |= tfCoinBase
	}
	txOut.PkScript, _ = txOut.TokenData.SeparateTokenDataFromPKScriptIfExists(txOut.PkScript, 0)
	return &UtxoEntry{
		amount:      txOut.Value,
		pkScript:    txOut.PkScript,
		tokenData:   txOut.TokenData,
		blockHeight: blockHeight,
		packedFlags: cbFlag,
	}
}

// IsCoinBase returns whether or not the output was contained in a coinbase
// transaction.
func (entry *UtxoEntry) IsCoinBase() bool {
	return entry.packedFlags&tfCoinBase == tfCoinBase
}

// IsSpent returns whether or not the output has been spent based upon the
// current state of the unspent transaction output view it was obtained from.
func (entry *UtxoEntry) IsSpent() bool {
	return entry.packedFlags&tfSpent == tfSpent
}

// isModified returns whether or not the output has been modified since it was
// loaded.
func (entry *UtxoEntry) isModified() bool {
	return entry.packedFlags&tfModified == tfModified
}

// isFresh returns whether or not it's certain the output has never previously
// been stored in the database.
func (entry *UtxoEntry) isFresh() bool {
	return entry.packedFlags&tfFresh == tfFresh
}

// BlockHeight returns the height of the block containing the output.
func (entry *UtxoEntry) BlockHeight() int32 {
	return entry.blockHeight
}

// Amount returns the amount of the output.
func (entry *UtxoEntry) Amount() int64 {
	return entry.amount
}

// PkScript returns the public key script for the output.
func (entry *UtxoEntry) PkScript() []byte {
	return entry.pkScript
}

// TokenData returns the token data for the output.
func (entry *UtxoEntry) TokenData() wire.TokenData {
	return entry.tokenData
}

// memoryUsage returns the memory usage in bytes of the UTXO entry.
// It returns 0 for the nil element.
func (entry *UtxoEntry) memoryUsage() uint64 {
	if entry == nil {
		return 0
	}

	baseEntrySize := uint64(baseUtxoEntrySizeWithoutTokenData + baseUtxoEntryTokenDataSize)

	return baseEntrySize + uint64(len(entry.pkScript))
}

// Spend marks the output as spent.  Spending an output that is already spent
// has no effect.
func (entry *UtxoEntry) Spend() {
	// Nothing to do if the output is already spent.
	if entry.IsSpent() {
		return
	}

	// Mark the output as spent and modified.
	entry.packedFlags |= tfSpent | tfModified
}

// Clone returns a shallow copy of the utxo entry.
func (entry *UtxoEntry) Clone() *UtxoEntry {
	if entry == nil {
		return nil
	}

	return &UtxoEntry{
		amount:      entry.amount,
		pkScript:    entry.pkScript,
		tokenData:   entry.tokenData,
		blockHeight: entry.blockHeight,
		packedFlags: entry.packedFlags,
	}
}

// utxoView is a common interface for structures that implement a UTXO view.
type utxoView interface {
	// getEntry tries to get an entry from the view.  If the entry is not in the
	// view, both the returned entry and the error are nil.
	getEntry(outpoint wire.OutPoint) (*UtxoEntry, error)

	// addEntry adds a new entry to the view.  Set overwrite to true if this
	// entry should overwrite any existing entry for the same outpoint.
	addEntry(outpoint wire.OutPoint, entry *UtxoEntry, overwrite bool) error

	// spendEntry marks an entry as spent.
	spendEntry(outpoint wire.OutPoint, entry *UtxoEntry) error
}

type utxoCacheInterface interface {

	// AddEntry adds a utxo entry for the given input index.
	AddEntry(i int, output wire.TxOut)

	// GetEntry adds a utxo entry for the given input index.
	GetEntry(i int) (wire.TxOut, error)
}

// utxoCache is a cached utxo view in the chainstate of a BlockChain.
//
// It implements the utxoView interface, but should only be used as such with the
// state mutex held.  It also implements the utxoByHashSource interface.
type utxoCache struct {
	db database.DB

	// maxTotalMemoryUsage is the maximum memory usage in bytes that the state
	// should contain in normal circumstances.
	maxTotalMemoryUsage uint64

	// This mutex protects the internal state.
	// A simple mutex instead of a read-write mutex is chosen because the main
	// read method also possibly does a write on a cache miss.
	mtx sync.Mutex

	// cachedEntries keeps the internal cache of the utxo state.  The tfModified
	// flag indicates that the state of the entry (potentially) deviates from the
	// state in the database.  Explicit nil values in the map are used to
	// indicate that the database does not contain the entry.
	cachedEntries    map[wire.OutPoint]*UtxoEntry
	totalEntryMemory uint64 // Total memory usage in bytes.
	lastFlushHash    chainhash.Hash

	// flushInProgress reports whether the cache is currently being flushed
	flushInProgress bool
}

// newUtxoCache initiates a new utxo cache instance with its memory usage limited
// to the given maximum.
func newUtxoCache(db database.DB, maxTotalMemoryUsage uint64) *utxoCache {
	return &utxoCache{
		db:                  db,
		maxTotalMemoryUsage: maxTotalMemoryUsage,

		cachedEntries: make(map[wire.OutPoint]*UtxoEntry),
	}
}

// totalMemoryUsage returns the total memory usage in bytes of the UTXO cache.
//
// This method should be called with the state lock held.
func (s *utxoCache) totalMemoryUsage() uint64 {
	// This value is calculated by running the following on a 64-bit system:
	// unsafe.Sizeof(wire.OutPoint{})
	outpointSize := uint64(36)

	// Total memory is all the keys plus the total memory of all the entries.
	nbEntries := uint64(len(s.cachedEntries))

	// Total size is total size of the keys + total size of the pointers in the
	// map + total size of the elements held in the pointers.
	return nbEntries*outpointSize + nbEntries*8 + s.totalEntryMemory
}

// TotalMemoryUsage returns the total memory usage in bytes of the UTXO cache.
//
// This method is safe for concurrent access.
func (s *utxoCache) TotalMemoryUsage() uint64 {
	s.mtx.Lock()
	tmu := s.totalMemoryUsage()
	s.mtx.Unlock()
	return tmu
}

// fetchAndCacheEntry tries to fetch an entry from the database.  In none is
// found, nil is returned.  If an entry is found, it is cached.
//
// This method should be called with the state lock held.
func (s *utxoCache) fetchAndCacheEntry(outpoint wire.OutPoint) (*UtxoEntry, error) {
	var entry *UtxoEntry
	err := s.db.View(func(dbTx database.Tx) error {
		var err error
		entry, err = dbFetchUtxoEntry(dbTx, outpoint)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Add the entry to the memory cache.
	// NOTE: When the fetched entry is nil, it is still added to the cache as a
	// miss; this prevents future lookups to perform the same database fetch.
	s.cachedEntries[outpoint] = entry
	s.totalEntryMemory += entry.memoryUsage()

	return entry, nil
}

// getEntry returns the UTXO entry for the given outpoint.  It returns nil if
// there is no entry for the outpoint in the UTXO state.
//
// This method is part of the utxoView interface.
// This method should be called with the state lock held.
// The returned entry is NOT safe for concurrent access.
func (s *utxoCache) getEntry(outpoint wire.OutPoint) (*UtxoEntry, error) {
	if entry, found := s.cachedEntries[outpoint]; found {
		return entry, nil
	}

	return s.fetchAndCacheEntry(outpoint)
}

// FetchEntry returns the UTXO entry for the given outpoint.  It returns nil if
// there is no entry for the outpoint in the UTXO state.
//
// This method is safe for concurrent access.
func (s *utxoCache) FetchEntry(outpoint wire.OutPoint) (*UtxoEntry, error) {
	s.mtx.Lock()
	entry, err := s.getEntry(outpoint)
	s.mtx.Unlock()
	return entry.Clone(), err
}

// FetchUtxoEntry returns the requested unspent transaction output from the point
// of view of the end of the main chain.
//
// NOTE: Requesting an output for which there is no data will NOT return an
// error.  Instead both the entry and the error will be nil.  This is done to
// allow pruning of spent transaction outputs.  In practice this means the
// caller must check if the returned entry is nil before invoking methods on it.
//
// This function is safe for concurrent access.
func (b *BlockChain) FetchUtxoEntry(outpoint wire.OutPoint) (*UtxoEntry, error) {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()
	return b.utxoCache.FetchEntry(outpoint)
}

// spendEntry marks the output as spent.  Spending an output that is already
// spent has no effect.  Entries that need not be stored anymore after being
// spent will be removed from the cache.
//
// This method is part of the utxoView interface.
// This method should be called with the state lock held.
func (s *utxoCache) spendEntry(outpoint wire.OutPoint, addIfNil *UtxoEntry) error {
	entry := s.cachedEntries[outpoint]

	// If we don't have an entry in cache and an entry was provided, we add it.
	if entry == nil && addIfNil != nil {
		if err := s.addEntry(outpoint, addIfNil, false); err != nil {
			return err
		}
		entry = addIfNil
	}

	// If it's nil or already spent, nothing to do.
	if entry == nil || entry.IsSpent() {
		return nil
	}

	// If an entry is fresh, meaning that there hasn't been a flush since it was
	// introduced, it can simply be removed.
	if entry.isFresh() {
		// We don't delete it from the map, but set the value to nil, so that
		// later lookups for the entry know that the entry does not exist in the
		// database.
		s.cachedEntries[outpoint] = nil
		s.totalEntryMemory -= entry.memoryUsage()
		return nil
	}

	// Mark the output as spent and modified.
	entry.packedFlags |= tfSpent | tfModified

	//TODO(stevenroose) check if it's ok to drop the pkScript
	// Since we don't need it anymore, drop the pkScript value of the entry.
	s.totalEntryMemory -= entry.memoryUsage()
	entry.pkScript = nil
	s.totalEntryMemory += entry.memoryUsage()

	return nil
}

// AddEntry adds a new unspent entry if it is not probably unspendable.  Set
// overwrite to true to skip validity and freshness checks and simply add the
// item, possibly overwriting another entry that is not-fully-spent.
//
// This function is safe for concurrent access
func (s *utxoCache) AddEntry(outpoint wire.OutPoint, entry *UtxoEntry, overwrite bool) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.addEntry(outpoint, entry, overwrite)
}

// addEntry adds a new unspent entry if it is not probably unspendable.  Set
// overwrite to true to skip validity and freshness checks and simply add the
// item, possibly overwriting another entry that is not-fully-spent.
//
// This method is part of the utxoView interface.
// This method should be called with the state lock held.
func (s *utxoCache) addEntry(outpoint wire.OutPoint, entry *UtxoEntry, overwrite bool) error {
	// Don't add provably unspendable outputs.
	if txscript.IsUnspendable(entry.pkScript) {
		return nil
	}

	cachedEntry := s.cachedEntries[outpoint]

	// In overwrite mode, simply add the entry without doing these checks.
	if !overwrite {
		// Prevent overwriting not-fully-spent entries.  Note that this is not
		// a consensus check.
		if cachedEntry != nil && !cachedEntry.IsSpent() {
			log.Warnf("utxo entry %s attempted to overwrite existing unspent "+
				"entry (pre-bip30?) ", outpoint)
			return nil
		}

		// If we didn't have an entry for the outpoint and the existing entry is
		// not marked modified, we can mark it fresh as the database does not
		// know about this entry.  This will allow us to erase it when it gets
		// spent before the next flush.
		if cachedEntry == nil && !entry.isModified() {
			entry.packedFlags |= tfFresh
		}
	}

	entry.packedFlags |= tfModified
	s.cachedEntries[outpoint] = entry
	s.totalEntryMemory -= cachedEntry.memoryUsage() // 0 for nil
	s.totalEntryMemory += entry.memoryUsage()
	return nil
}

// FetchTxView returns a local view on the utxo state for the given transaction.
//
// This method is safe for concurrent access.
func (s *utxoCache) FetchTxView(tx *bchutil.Tx) (*UtxoViewpoint, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	view := NewUtxoViewpoint()
	viewEntries := view.Entries()
	if !IsCoinBase(tx) {
		for _, txIn := range tx.MsgTx().TxIn {
			entry, err := s.getEntry(txIn.PreviousOutPoint)
			if err != nil {
				return nil, err
			}
			viewEntries[txIn.PreviousOutPoint] = entry.Clone()
		}
	}
	prevOut := wire.OutPoint{Hash: *tx.Hash()}
	for txOutIdx := range tx.MsgTx().TxOut {
		prevOut.Index = uint32(txOutIdx)

		entry, err := s.getEntry(prevOut)
		if err != nil {
			return nil, err
		}
		viewEntries[prevOut] = entry.Clone()
	}

	return view, nil
}

// FetchUtxoView loads unspent transaction outputs for the inputs referenced by
// the passed transaction from the point of view of the end of the main chain.
// It also attempts to get the utxos for the outputs of the transaction itself
// so the returned view can be examined for duplicate transactions.
//
// This function is safe for concurrent access however the returned view is NOT.
func (b *BlockChain) FetchUtxoView(tx *bchutil.Tx) (*UtxoViewpoint, error) {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()
	return b.utxoCache.FetchTxView(tx)
}

// Commit commits all the entries in the view to the cache.
//
// This method should be called with the state lock held.
func (s *utxoCache) Commit(view *UtxoViewpoint) error {
	for outpoint, entry := range view.Entries() {
		// No need to update the database if the entry was not modified or fresh.
		if entry == nil || (!entry.isModified() && !entry.isFresh()) {
			continue
		}

		// We can't use the view entry directly because it can be modified
		// later on.
		ourEntry := s.cachedEntries[outpoint]
		if ourEntry == nil {
			ourEntry = entry.Clone()
		}

		// Remove the utxo entry if it is spent.
		if entry.IsSpent() {
			if err := s.spendEntry(outpoint, ourEntry); err != nil {
				return err
			}
			continue
		}

		// It's possible if we disconnected this UTXO at some point, removing it from
		// the UTXO set, only to have a future block add it back. In that case it could
		// be going from being marked spent to needing to be marked unspent so we handle
		// that case by overriding here.
		override := false
		if ourEntry.IsSpent() && !entry.IsSpent() {
			ourEntry = entry.Clone()
			override = true
		}

		// Store the entry we don't know.
		if err := s.addEntry(outpoint, ourEntry, override); err != nil {
			return err
		}
	}

	view.prune()
	return nil
}

// flush flushes the UTXO state to the database.
//
// This method should be called with the state lock held.
func (s *utxoCache) flush(bestState *BestState) error {
	// If we performed a flush in the current best state, we have nothing to do.
	// If the bestState hash is the zero hash then skip this check and continue with
	// the flush. The fastsync mode calls flush with the zerohash during initial utxo
	// set download so we don't want to prevent it from flushing in this case.
	if bestState.Hash == s.lastFlushHash && !bestState.Hash.IsEqual(&chainhash.Hash{}) {
		return nil
	}

	// Add one to round up the integer division.
	totalMiB := s.totalMemoryUsage()/(1024*1024) + 1
	log.Infof("Flushing UTXO cache of ~%v MiB to disk. For large sizes, "+
		"this can take up to several minutes...", totalMiB)

	// First update the database to indicate that a utxo state flush is started.
	// This allows us to recover when the node shuts down in the middle of this
	// method.
	err := s.db.Update(func(dbTx database.Tx) error {
		return dbPutUtxoStateConsistency(dbTx, ucsFlushOngoing, &s.lastFlushHash)
	})
	if err != nil {
		return err
	}

	// Store all entries in batches.
	flushBatch := func(dbTx database.Tx) error {
		var (
			// Form a batch by storing all entries to be put and deleted.
			nbBatchEntries = 0
			entriesPut     = make(map[wire.OutPoint]*UtxoEntry)
			entriesDelete  = make([]wire.OutPoint, 0)
		)
		for outpoint, entry := range s.cachedEntries {
			// Nil entries or unmodified entries can just be pruned.
			// They don't count for the batch size.
			if entry == nil || !entry.isModified() {
				s.totalEntryMemory -= entry.memoryUsage()
				delete(s.cachedEntries, outpoint)
				continue
			}

			if entry.IsSpent() {
				entriesDelete = append(entriesDelete, outpoint)
			} else {
				entriesPut[outpoint] = entry
			}
			nbBatchEntries++

			s.totalEntryMemory -= entry.memoryUsage()
			delete(s.cachedEntries, outpoint)

			// End this batch when the maximum number of entries per batch has
			// been reached.
			if nbBatchEntries >= utxoBatchSizeEntries {
				break
			}
		}

		// Apply the batched additions and deletions.
		if err := dbPutUtxoEntries(dbTx, entriesPut); err != nil {
			return err
		}

		return dbDeleteUtxoEntries(dbTx, entriesDelete)
	}
	s.flushInProgress = true
	defer func() { s.flushInProgress = false }()
	for len(s.cachedEntries) > 0 {
		log.Tracef("Flushing %d more entries...", len(s.cachedEntries))
		err := s.db.Update(func(dbTx database.Tx) error {
			return flushBatch(dbTx)
		})
		if err != nil {
			return err
		}
	}

	// When done, store the best state hash in the database to indicate the state
	// is consistent until that hash.
	err = s.db.Update(func(dbTx database.Tx) error {
		return dbPutUtxoStateConsistency(dbTx, ucsConsistent, &bestState.Hash)
	})
	if err != nil {
		return err
	}
	s.lastFlushHash = bestState.Hash
	log.Debug("Done flushing UTXO cache to disk")
	return nil
}

// Flush flushes the UTXO state to the database.
//
// This function is safe for concurrent access.
func (s *utxoCache) Flush(mode FlushMode, bestState *BestState) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	var threshold uint64
	switch mode {
	case FlushRequired:
		threshold = 0

	case FlushIfNeeded:
		threshold = s.maxTotalMemoryUsage

	case FlushPeriodic:
		threshold = (utxoFlushPeriodicThreshold * s.maxTotalMemoryUsage) / 100
	}

	if s.totalMemoryUsage() > threshold {
		return s.flush(bestState)
	}
	return nil
}

// rollBackBlock rolls back the effects of the block when the state was left in
// an inconsistent state.  This means that no errors will be raised when the
// state is invalid.
//
// This method should be called with the state lock held.
func (s *utxoCache) rollBackBlock(block *bchutil.Block, stxos []SpentTxOut) error {
	return disconnectTransactions(s, block, stxos)
}

// rollForwardBlock rolls forward the effects of the block when the state was
// left in an inconsistent state.  This means that no errors will be raised when
// the state is invalid.
//
// This method should be called with the state lock held.
func (s *utxoCache) rollForwardBlock(block *bchutil.Block) error {
	// We don't need the collect stxos and we allow overwriting existing entries.
	return connectTransactions(s, block, nil, true)
}

// InitConsistentState checks the consistency status of the utxo state and
// replays blocks if it lags behind the best state of the blockchain.
//
// It needs to be ensured that the chainView passed to this method does not
// get changed during the execution of this method.
func (s *utxoCache) InitConsistentState(tip *blockNode, fastSync bool, interrupt <-chan struct{}) error {
	// Load the consistency status from the database.
	var statusCode byte
	var statusHash *chainhash.Hash
	err := s.db.View(func(dbTx database.Tx) error {
		var err error
		statusCode, statusHash, err = dbFetchUtxoStateConsistency(dbTx)

		return err
	})
	if err != nil {
		return err
	}

	if fastSync {
		// If we're in fast sync mode and the status hash is not the zerohash then
		// we must have previously started the node not in fastsync mode which means
		// the UTXO set bucket will be dirty. In this case let's reset the UTXO
		// bucket so we can get a fresh start.
		if !statusHash.IsEqual(&chainhash.Hash{}) {
			err := s.db.Update(func(tx database.Tx) error {
				if err := tx.Metadata().DeleteBucket(utxoSetBucketName); err != nil {
					return err
				}

				if _, err := tx.Metadata().CreateBucket(utxoSetBucketName); err != nil {
					return err
				}

				return nil
			})

			return err
		}

		// If we're in fast sync mode and the status hash is the zero hash then
		// this is either a completely new node or a node that suffered a hard
		// shutdown during fast sync. In either case we don't need to repair
		// the UTXO set as we can just write over existing entries as we download
		// the UTXO set.
		return nil
	}

	log.Tracef("UTXO cache consistency status from disk: [%d] hash %v",
		statusCode, statusHash)

	// If no status was found, the database is old and didn't have a cached utxo
	// state yet. In that case, we set the status to the best state and write
	// this to the database.
	if statusCode == ucsEmpty {
		log.Debugf("Database didn't specify UTXO state consistency: consistent "+
			"to best chain tip (%v)", tip.hash)
		s.lastFlushHash = tip.hash
		err := s.db.Update(func(dbTx database.Tx) error {
			return dbPutUtxoStateConsistency(dbTx, ucsConsistent, &tip.hash)
		})

		return err
	}

	// If state is consistent, we are done.
	if statusCode == ucsConsistent && *statusHash == tip.hash {
		log.Debugf("UTXO state consistent (%d:%v)", tip.height, tip.hash)
		s.lastFlushHash = tip.hash
		return nil
	}

	s.lastFlushHash = *statusHash

	log.Info("Reconstructing UTXO state after unclean shutdown. This may take " +
		"a long time...")

	// Even though this should always be true, make sure the fetched hash is in
	// the best chain.
	var statusNode *blockNode
	var statusNodeNext *blockNode // the first one higher than the statusNode
	attachNodes := list.New()
	for node := tip; node.height >= 0; node = node.parent {
		if node.hash == *statusHash {
			statusNode = node
			break
		}
		attachNodes.PushFront(node)
		statusNodeNext = node
	}

	if statusNode == nil {
		return AssertError(fmt.Sprintf("last utxo consistency status contains "+
			"hash that is not in best chain: %v", statusHash))
	}

	// If data was in the middle of a flush, we have to roll back all blocks from
	// the last best block all the way back to the last consistent block.
	log.Debugf("Rolling back %d blocks to rebuild the UTXO state...", tip.height-statusNode.height)

	// Roll back blocks in batches.
	rollbackBatch := func(dbTx database.Tx, node *blockNode) (*blockNode, error) {
		nbBatchBlocks := 0
		for ; node.height > statusNode.height; node = node.parent {
			block, err := dbFetchBlockByNode(dbTx, node)
			if err != nil {
				return nil, err
			}

			stxos, err := dbFetchSpendJournalEntry(dbTx, block)
			if err != nil {
				return nil, err
			}

			if err := s.rollBackBlock(block, stxos); err != nil {
				return nil, err
			}

			nbBatchBlocks++

			if nbBatchBlocks >= utxoBatchSizeBlocks {
				break
			}
		}

		return node, nil
	}

	for node := tip; node.height > statusNode.height; {
		log.Tracef("Rolling back %d more blocks...",
			node.height-statusNode.height)
		err := s.db.Update(func(dbTx database.Tx) error {
			var err error
			node, err = rollbackBatch(dbTx, node)

			return err
		})
		if err != nil {
			return err
		}

		if interruptRequested(interrupt) {
			log.Warn("UTXO state reconstruction interrupted")

			return errInterruptRequested
		}
	}

	// Now we can flush and update the status to avoid redoing this work when
	// interrupted.
	if err := s.flush(&BestState{Hash: *statusHash}); err != nil {
		return err
	}

	log.Debugf("Replaying %d blocks to rebuild UTXO state...",
		tip.height-statusNodeNext.height+1)

	// Then we replay the blocks from the last consistent state up to the best
	// state. Iterate forward from the consistent node to the tip of the best
	// chain. After every batch, we can also update the consistency state to
	// avoid redoing the work when interrupted.
	rollforwardBatch := func(dbTx database.Tx, node *blockNode) (*blockNode, error) {
		nbBatchBlocks := 0
		toRemove := make([]*list.Element, 0, utxoBatchSizeBlocks)
		for e := attachNodes.Front(); e != nil; e = e.Next() {
			node = e.Value.(*blockNode)
			toRemove = append(toRemove, e)

			block, err := dbFetchBlockByNode(dbTx, node)
			if err != nil {
				return nil, err
			}

			if err := s.rollForwardBlock(block); err != nil {
				return nil, err
			}
			nbBatchBlocks++

			if nbBatchBlocks >= utxoBatchSizeBlocks {
				break
			}
		}
		for _, e := range toRemove {
			attachNodes.Remove(e)
		}

		return node, nil
	}

	for node := statusNodeNext; node.height <= tip.height; {
		log.Tracef("Replaying %d more blocks...", tip.height-node.height+1)
		err := s.db.Update(func(dbTx database.Tx) error {
			var err error
			node, err = rollforwardBatch(dbTx, node)

			return err
		})
		if err != nil {
			return err
		}

		// We can flush after each batch to avoid having to redo the work
		// when interrupted.
		if err := s.flush(&BestState{Hash: node.hash}); err != nil {
			return err
		}

		if interruptRequested(interrupt) {
			log.Warn("UTXO state reconstruction interrupted")

			return errInterruptRequested
		}
		if node.height == tip.height {
			break
		}
	}

	log.Debug("UTXO state reconstruction done")

	return nil
}
