// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
Package mempool provides a policy-enforced pool of unmined bitcoin cash transactions.

A key responsibility of the bitcoin cash network is mining user-generated transactions
into blocks.  In order to facilitate this, the mining process relies on having a
readily-available source of transactions to include in a block that is being
solved.

At a high level, this package satisfies that requirement by providing an
in-memory pool of fully validated transactions that can also optionally be
further filtered based upon a configurable policy.

One of the policy configuration options controls whether or not "standard"
transactions are accepted.  In essence, a "standard" transaction is one that
satisfies a fairly strict set of requirements that are largely intended to help
provide fair use of the system to all users.  It is important to note that what
is considered a "standard" transaction changes over time.  For some insight, at
the time of this writing, an example of SOME of the criteria that are required
for a transaction to be considered standard are that it is of the most-recently
supported version, finalized, does not exceed a specific size, and only consists
of specific script forms.

Since this package does not deal with other bitcoin cash specifics such as network
communication and transaction relay, it returns a list of transactions that were
accepted which gives the caller a high level of flexibility in how they want to
proceed.  Typically, this will involve things such as relaying the transactions
to other peers on the network and notifying the mining process that new
transactions are available.

# Feature Overview

The following is a quick overview of the major features.  It is not intended to
be an exhaustive list.

  - Maintain a pool of fully validated transactions
  - Reject non-fully-spent duplicate transactions
  - Reject coinbase transactions
  - Reject double spends (both from the chain and other transactions in pool)
  - Reject invalid transactions according to the network consensus rules
  - Full script execution and validation with signature cache support
  - Individual transaction query support
  - Orphan transaction support (transactions that spend from unknown outputs)
  - Configurable limits (see transaction acceptance policy)
  - Automatic addition of orphan transactions that are no longer orphans as new
    transactions are added to the pool
  - Individual orphan transaction query support
  - Configurable transaction acceptance policy
  - Option to accept or reject standard transactions
  - Option to accept or reject transactions based on priority calculations
  - Rate limiting of low-fee and free transactions
  - Non-zero fee threshold
  - Max signature operations per transaction
  - Max orphan transaction size
  - Max number of orphan transactions allowed
  - Additional metadata tracking for each transaction
  - Timestamp when the transaction was added to the pool
  - Most recent block height when the transaction was added to the pool
  - The fee the transaction pays
  - The starting priority for the transaction
  - Manual control of transaction removal
  - Recursive removal of all dependent transactions

# Errors

Errors returned by this package are either the raw errors provided by underlying
calls or of type mempool.RuleError.  Since there are two classes of rules
(mempool acceptance rules and blockchain (consensus) acceptance rules), the
mempool.RuleError type contains a single Err field which will, in turn, either
be a mempool.TxRuleError or a blockchain.RuleError.  The first indicates a
violation of mempool acceptance rules while the latter indicates a violation of
consensus acceptance rules.  This allows the caller to easily differentiate
between unexpected errors, such as database errors, versus errors due to rule
violations through type assertions.  In addition, callers can programmatically
determine the specific rule violation by type asserting the Err field to one of
the aforementioned types and examining their underlying ErrorCode field.
*/
package mempool
