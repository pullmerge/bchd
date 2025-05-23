// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"math/rand"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gcash/bchd/wire"
)

// genTestTx creates a random transaction for uses within test cases.
func genTestTx() (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(2)
	tx.Version = rand.Int31()

	numTxins := rand.Intn(11)
	for i := 0; i < numTxins; i++ {
		randTxIn := wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Index: uint32(rand.Int31()),
			},
			Sequence: uint32(rand.Int31()),
		}
		_, err := rand.Read(randTxIn.PreviousOutPoint.Hash[:])
		if err != nil {
			return nil, err
		}

		tx.TxIn = append(tx.TxIn, &randTxIn)
	}

	numTxouts := rand.Intn(11)
	for i := 0; i < numTxouts; i++ {
		randTxOut := wire.TxOut{
			Value:    rand.Int63(),
			PkScript: make([]byte, rand.Intn(30)),
		}
		if _, err := rand.Read(randTxOut.PkScript); err != nil {
			return nil, err
		}
		tx.TxOut = append(tx.TxOut, &randTxOut)
	}

	return tx, nil
}

// TestHashCacheAddContainsHashes tests that after items have been added to the
// hash cache, the ContainsHashes method returns true for all the items
// inserted.  Conversely, ContainsHashes should return false for any items
// _not_ in the hash cache.
func TestHashCacheAddContainsHashes(t *testing.T) {
	t.Parallel()

	rand.Seed(time.Now().Unix())

	cache := NewHashCache(10)

	var err error

	// First, we'll generate 10 random transactions for use within our
	// tests.
	const numTxns = 10
	txns := make([]*wire.MsgTx, numTxns)
	for i := 0; i < numTxns; i++ {
		txns[i], err = genTestTx()
		if err != nil {
			t.Fatalf("unable to generate test tx: %v", err)
		}
	}

	// With the transactions generated, we'll add each of them to the hash
	// cache.
	for _, tx := range txns {
		cache.AddSigHashes(tx)
	}

	// Next, we'll ensure that each of the transactions inserted into the
	// cache are properly located by the ContainsHashes method.
	for _, tx := range txns {
		txid := tx.TxHash()
		if ok := cache.ContainsHashes(&txid); !ok {
			t.Fatalf("txid %v not found in cache but should be: ",
				txid)
		}
	}

	randTx, err := genTestTx()
	if err != nil {
		t.Fatalf("unable to generate tx: %v", err)
	}

	// Finally, we'll assert that a transaction that wasn't added to the
	// cache won't be reported as being present by the ContainsHashes
	// method.
	randTxid := randTx.TxHash()
	if ok := cache.ContainsHashes(&randTxid); ok {
		t.Fatalf("txid %v wasn't inserted into cache but was found",
			randTxid)
	}
}

// TestHashCacheAddGet tests that the sighashes for a particular transaction
// are properly retrieved by the GetSigHashes function.
func TestHashCacheAddGet(t *testing.T) {
	t.Parallel()

	rand.Seed(time.Now().Unix())

	cache := NewHashCache(10)

	// To start, we'll generate a random transaction and compute the set of
	// sighashes for the transaction.
	randTx, err := genTestTx()
	if err != nil {
		t.Fatalf("unable to generate tx: %v", err)
	}
	sigHashes := NewTxSigHashes(randTx)

	// Next, add the transaction to the hash cache.
	cache.AddSigHashes(randTx)

	// The transaction inserted into the cache above should be found.
	txid := randTx.TxHash()
	cacheHashes, ok := cache.GetSigHashes(&txid)
	if !ok {
		t.Fatalf("tx %v wasn't found in cache", txid)
	}

	// Finally, the sighashes retrieved should exactly match the sighash
	// originally inserted into the cache.
	// TODO TODO
	if sigHashes.HashOutputs != cacheHashes.HashOutputs ||
		sigHashes.HashPrevOuts != cacheHashes.HashPrevOuts ||
		sigHashes.HashSequence != cacheHashes.HashSequence {
		t.Fatalf("sighashes don't match: expected %v, got %v",
			spew.Sdump(sigHashes), spew.Sdump(cacheHashes))
	}
}

// TestHashCachePurge tests that items are able to be properly removed from the
// hash cache.
func TestHashCachePurge(t *testing.T) {
	t.Parallel()

	rand.Seed(time.Now().Unix())

	cache := NewHashCache(10)

	var err error

	// First we'll start by inserting numTxns transactions into the hash cache.
	const numTxns = 10
	txns := make([]*wire.MsgTx, numTxns)
	for i := 0; i < numTxns; i++ {
		txns[i], err = genTestTx()
		if err != nil {
			t.Fatalf("unable to generate test tx: %v", err)
		}
	}
	for _, tx := range txns {
		cache.AddSigHashes(tx)
	}

	// Once all the transactions have been inserted, we'll purge them from
	// the hash cache.
	for _, tx := range txns {
		txid := tx.TxHash()
		cache.PurgeSigHashes(&txid)
	}

	// At this point, none of the transactions inserted into the hash cache
	// should be found within the cache.
	for _, tx := range txns {
		txid := tx.TxHash()
		if ok := cache.ContainsHashes(&txid); ok {
			t.Fatalf("tx %v found in cache but should have "+
				"been purged: ", txid)
		}
	}
}
