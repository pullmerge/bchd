// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/wire"
)

// These variables are the chain proof-of-work limit parameters for each default
// network.
var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// mainPowLimit is the highest proof of work value a Bitcoin block can
	// have for the main network.  It is the value 2^224 - 1.
	mainPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)

	// regressionPowLimit is the highest proof of work value a Bitcoin block
	// can have for the regression test network.  It is the value 2^255 - 1.
	regressionPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// testNet3PowLimit is the highest proof of work value a Bitcoin block
	// can have for the test network (version 3).  It is the value
	// 2^224 - 1.
	testNet3PowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)

	// simNetPowLimit is the highest proof of work value a Bitcoin block
	// can have for the simulation test network.  It is the value 2^255 - 1.
	simNetPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
)

// Checkpoint identifies a known good point in the block chain.  Using
// checkpoints allows a few optimizations for old blocks during initial download
// and also prevents forks from old blocks.
//
// Each checkpoint is selected based upon several factors.  See the
// documentation for blockchain.IsCheckpointCandidate for details on the
// selection criteria.
type Checkpoint struct {
	Height         int32
	Hash           *chainhash.Hash
	UtxoSetHash    *chainhash.Hash
	UtxoSetSources []string
	UtxoSetSize    uint32
}

// DNSSeed identifies a DNS seed.
type DNSSeed struct {
	// Host defines the hostname of the seed.
	Host string

	// HasFiltering defines whether the seed supports filtering
	// by service flags (wire.ServiceFlag).
	HasFiltering bool
}

// ConsensusDeployment defines details related to a specific consensus rule
// change that is voted in.  This is part of BIP0009.
type ConsensusDeployment struct {
	// BitNumber defines the specific bit number within the block version
	// this particular soft-fork deployment refers to.
	BitNumber uint8

	// StartTime is the median block time after which voting on the
	// deployment starts.
	StartTime uint64

	// ExpireTime is the median block time after which the attempted
	// deployment expires.
	ExpireTime uint64
}

// Constants that define the deployment offset in the deployments field of the
// parameters for each deployment.  This is useful to be able to get the details
// of a specific deployment by name.
const (
	// DeploymentTestDummy defines the rule change deployment ID for testing
	// purposes.
	DeploymentTestDummy = iota

	// DeploymentCSV defines the rule change deployment ID for the CSV
	// soft-fork package. The CSV package includes the deployment of BIPS
	// 68, 112, and 113.
	DeploymentCSV

	// NOTE: DefinedDeployments must always come last since it is used to
	// determine how many defined deployments there currently are.

	// DefinedDeployments is the number of currently defined deployments.
	DefinedDeployments
)

type ABLAConstants struct {
	Epsilon0        uint64
	Beta0           uint64
	N0              uint64
	GammaReciprocal uint64
	ZetaXB7         uint64
	ThetaReciprocal uint64
	Delta           uint64
	FixedSize       bool
}

// Params defines a Bitcoin network by its parameters.  These parameters may be
// used by Bitcoin applications to differentiate networks as well as addresses
// and keys for one network from those intended for use on another network.
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name string

	// Net defines the magic bytes used to identify the network.
	Net wire.BitcoinNet

	// DefaultPort defines the default peer-to-peer port for the network.
	DefaultPort string

	// DNSSeeds defines a list of DNS seeds for the network that are used
	// as one method to discover peers.
	DNSSeeds []DNSSeed

	// GenesisBlock defines the first block of the chain.
	GenesisBlock *wire.MsgBlock

	// GenesisHash is the starting block hash.
	GenesisHash *chainhash.Hash

	// PowLimit defines the highest allowed proof of work value for a block
	// as a uint256.
	PowLimit *big.Int

	// PowLimitBits defines the highest allowed proof of work value for a
	// block in compact form.
	PowLimitBits uint32

	// These fields define the block heights at which the specified softfork
	// BIP became active.
	BIP0034Height int32
	BIP0065Height int32
	BIP0066Height int32

	// Only testnet4 and chipnet uses CSV activation by height. All the others use the
	// deployment schedule. If this value is set to anything other than zero
	// then it will activate at this height.
	CSVHeight int32

	// The following are the heights at which the Bitcoin Cash specific forks
	// became active.
	UahfForkHeight                int32  // August 1, 2017 hardfork
	DaaForkHeight                 int32  // November 13, 2017 hardfork
	MagneticAnonomalyForkHeight   int32  // November 15, 2018 hardfork
	GreatWallForkHeight           int32  // May 15, 2019 hardfork
	GravitonForkHeight            int32  // Nov 15, 2019 hardfork
	PhononForkHeight              int32  // May 15, 2020 hardfork
	AxionActivationHeight         int32  // Nov 15, 2020 hardfork
	CosmicInflationActivationTime uint64 // May 15, 2022 hardfork
	Upgrade9ForkHeight            int32  // May 15, 2023 hardfork
	ABLAForkHeight                int32  // May 15, 2024 hardfork
	Upgrade11ActivationTime       uint64 // May 15, 2025 hardfork

	// The ABLA algorithm constants
	ABLAConfig ABLAConstants

	// CoinbaseMaturity is the number of blocks required before newly mined
	// coins (coinbase transactions) can be spent.
	CoinbaseMaturity uint16

	// SubsidyReductionInterval is the interval of blocks before the subsidy
	// is reduced.
	SubsidyReductionInterval int32

	// TargetTimespan is the desired amount of time that should elapse
	// before the block difficulty requirement is examined to determine how
	// it should be changed in order to maintain the desired block
	// generation rate.
	TargetTimespan time.Duration

	// TargetTimePerBlock is the desired amount of time to generate each
	// block.
	TargetTimePerBlock time.Duration

	// RetargetAdjustmentFactor is the adjustment factor used to limit
	// the minimum and maximum amount of adjustment that can occur between
	// difficulty retargets.
	RetargetAdjustmentFactor int64

	// ReduceMinDifficulty defines whether the network should reduce the
	// minimum required difficulty after a long enough period of time has
	// passed without finding a block.  This is really only useful for test
	// networks and should not be set on a main network.
	ReduceMinDifficulty bool

	// NoDifficultyAdjustment defines whether the network should skip the
	// normal difficulty adjustment and keep the current difficulty.
	NoDifficultyAdjustment bool

	// MinDiffReductionTime is the amount of time after which the minimum
	// required difficulty should be reduced when a block hasn't been found.
	//
	// NOTE: This only applies if ReduceMinDifficulty is true.
	MinDiffReductionTime time.Duration

	// AsertDifficultyHalflife is the halflife parameter used by the asert
	// difficulty adjustment algorithm for the given network.
	AsertDifficultyHalflife int64

	// AsertDifficultyAnchorHeight is the height of the asert difficulty
	// anchor block.
	AsertDifficultyAnchorHeight int32

	// AsertDifficultyAnchorParentTimestamp is the timestamp of the asert difficulty
	// anchor block's parent.
	AsertDifficultyAnchorParentTimestamp int64

	// AsertDifficultyAnchorBits is the bits of the asert difficulty
	// anchor block.
	AsertDifficultyAnchorBits uint32

	// GenerateSupported specifies whether or not CPU mining is allowed.
	GenerateSupported bool

	// Checkpoints ordered from oldest to newest.
	Checkpoints []Checkpoint

	// These fields are related to voting on consensus rule changes as
	// defined by BIP0009.
	//
	// RuleChangeActivationThreshold is the number of blocks in a threshold
	// state retarget window for which a positive vote for a rule change
	// must be cast in order to lock in a rule change. It should typically
	// be 95% for the main network and 75% for test networks.
	//
	// MinerConfirmationWindow is the number of blocks in each threshold
	// state retarget window.
	//
	// Deployments define the specific consensus rule changes to be voted
	// on.
	RuleChangeActivationThreshold uint32
	MinerConfirmationWindow       uint32
	Deployments                   [DefinedDeployments]ConsensusDeployment

	// Mempool parameters
	RelayNonStdTxs bool

	// The prefix used for the cashaddress. This is different for each network.
	CashAddressPrefix string

	// Address encoding magics
	LegacyPubKeyHashAddrID byte // First byte of a P2PKH address
	LegacyScriptHashAddrID byte // First byte of a P2SH address
	PrivateKeyID           byte // First byte of a WIF private key

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID [4]byte
	HDPublicKeyID  [4]byte

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType uint32

	// slp indexer parameters
	SlpIndexStartHeight int32
	SlpIndexStartHash   *chainhash.Hash
	SlpAddressPrefix    string
}

// MainNetParams defines the network parameters for the main Bitcoin network.
var MainNetParams = Params{
	Name:        "mainnet",
	Net:         wire.MainNet,
	DefaultPort: "8333",
	DNSSeeds: []DNSSeed{ // TODO change this to proper dns seed
		{"seed.bchd.cash", true},
		{"btccash-seeder.bitcoinunlimited.info", true},
		{"seed.bch.loping.net", true},
		{"dnsseed.electroncash.de", true},
	},

	// Chain parameters
	GenesisBlock:  &genesisBlock,
	GenesisHash:   &genesisHash,
	PowLimit:      mainPowLimit,
	PowLimitBits:  0x1d00ffff,
	BIP0034Height: 227931, // 000000000000024b89b42a942fe0d9fea3bb44ab7bd1b19115dd6a759c0808b8
	BIP0065Height: 388381, // 000000000000000004c2b624ed5d7756c508d90fd0da2c7c679febfa6c4735f0
	BIP0066Height: 363725, // 00000000000000000379eaa19dce8c9b722d46ae6a57c2f1a988119488b50931

	UahfForkHeight:              478558, // 0000000000000000011865af4122fe3b144e2cbeea86142e8ff2fb4107352d43
	DaaForkHeight:               504031, // 0000000000000000011ebf65b60d0a3de80b8175be709d653b4c1a1beeb6ab9c
	MagneticAnonomalyForkHeight: 556766, // 00000000000000000102d94fde9bd0807a2cc7582fe85dd6349b73ce4e8d9322
	GreatWallForkHeight:         582679, // 0000000000000000018596bdfd350a9fbc7297a62a3f510b74565d992d63d2ef
	GravitonForkHeight:          609135, // 0000000000000000026f7ec9e79be2f5bb839f29ebcf734066d4bb9a13f6ea83
	PhononForkHeight:            635258, // 000000000000000003302c47d01e78f1c86aa3b0e96b066761a5059bc8f5781a
	AxionActivationHeight:       661647, // 00000000000000000083ed4b7a780d59e3983513215518ad75654bb02deee62f

	CosmicInflationActivationTime: 1652616000,

	Upgrade9ForkHeight: 792772, // 000000000000000002B678C471841C3E404EC7AE9CA9C32026FE27EB6E3A1ED1
	ABLAForkHeight:     845890,

	Upgrade11ActivationTime: 1747310400,

	// Reference for the following constant values: https://gitlab.com/0353F40E/ebaa/-/blob/main/README.md#mainnet
	ABLAConfig: ABLAConstants{
		Epsilon0:        16000000,
		Beta0:           16000000,
		N0:              845890,
		GammaReciprocal: 37938,
		ZetaXB7:         192,
		ThetaReciprocal: 37938,
		Delta:           10,
		FixedSize:       false,
	},

	CoinbaseMaturity:                     100,
	SubsidyReductionInterval:             210000,
	TargetTimespan:                       time.Hour * 24 * 14, // 14 days
	TargetTimePerBlock:                   time.Minute * 10,    // 10 minutes
	RetargetAdjustmentFactor:             4,                   // 25% less, 400% more
	ReduceMinDifficulty:                  false,
	NoDifficultyAdjustment:               false,
	MinDiffReductionTime:                 0,
	AsertDifficultyHalflife:              2 * 24 * 3600, // 2 days in seconds
	AsertDifficultyAnchorHeight:          661647,
	AsertDifficultyAnchorParentTimestamp: 1605447844,
	AsertDifficultyAnchorBits:            402971390,
	GenerateSupported:                    false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{
		{Height: 11111, Hash: newHashFromStr("0000000069e244f73d78e8fd29ba2fd2ed618bd6fa2ee92559f542fdb26e7c1d")},
		{Height: 33333, Hash: newHashFromStr("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")},
		{Height: 74000, Hash: newHashFromStr("0000000000573993a3c9e41ce34471c079dcf5f52a0e824a81e7f953b8661a20")},
		{Height: 105000, Hash: newHashFromStr("00000000000291ce28027faea320c8d2b054b2e0fe44a773f3eefb151d6bdc97")},
		{Height: 134444, Hash: newHashFromStr("00000000000005b12ffd4cd315cd34ffd4a594f430ac814c91184a0d42d2b0fe")},
		{Height: 168000, Hash: newHashFromStr("000000000000099e61ea72015e79632f216fe6cb33d7899acb35b75c8303b763")},
		{Height: 193000, Hash: newHashFromStr("000000000000059f452a5f7340de6682a977387c17010ff6e6c3bd83ca8b1317")},
		{Height: 210000, Hash: newHashFromStr("000000000000048b95347e83192f69cf0366076336c639f9b7228e9ba171342e")},
		{Height: 216116, Hash: newHashFromStr("00000000000001b4f4b433e81ee46494af945cf96014816a4e2370f11b23df4e")},
		{Height: 225430, Hash: newHashFromStr("00000000000001c108384350f74090433e7fcf79a606b8e797f065b130575932")},
		{Height: 250000, Hash: newHashFromStr("000000000000003887df1f29024b06fc2200b55f8af8f35453d7be294df2d214")},
		{Height: 267300, Hash: newHashFromStr("000000000000000a83fbd660e918f218bf37edd92b748ad940483c7c116179ac")},
		{Height: 279000, Hash: newHashFromStr("0000000000000001ae8c72a0b0c301f67e3afca10e819efa9041e458e9bd7e40")},
		{Height: 300255, Hash: newHashFromStr("0000000000000000162804527c6e9b9f0563a280525f9d08c12041def0a0f3b2")},
		{Height: 319400, Hash: newHashFromStr("000000000000000021c6052e9becade189495d1c539aa37c58917305fd15f13b")},
		{Height: 343185, Hash: newHashFromStr("0000000000000000072b8bf361d01a6ba7d445dd024203fafc78768ed4368554")},
		{Height: 352940, Hash: newHashFromStr("000000000000000010755df42dba556bb72be6a32f3ce0b6941ce4430152c9ff")},
		{Height: 382320, Hash: newHashFromStr("00000000000000000a8dc6ed5b133d0eb2fd6af56203e4159789b092defd8ab2")},
		{Height: 400000, Hash: newHashFromStr("000000000000000004ec466ce4732fe6f1ed1cddc2ed4b328fff5224276e3f6f")},
		{Height: 430000, Hash: newHashFromStr("000000000000000001868b2bb3a285f3cc6b33ea234eb70facf4dcdf22186b87")},
		{Height: 470000, Hash: newHashFromStr("0000000000000000006c539c722e280a0769abd510af0073430159d71e6d7589")},
		{Height: 510000, Hash: newHashFromStr("00000000000000000367922b6457e21d591ef86b360d78a598b14c2f1f6b0e04")},
		{Height: 552979, Hash: newHashFromStr("0000000000000000015648768ac1b788a83187d706f858919fcc5c096b76fbf2")},
		{
			Height:      556767,
			Hash:        newHashFromStr("0000000000000000004626ff6e3b936941d341c5932ece4357eeccac44e6d56c"),
			UtxoSetHash: newHashFromStr("352a96fdd4dd229ced594b6bae5c658da514eb76e9b04eb469a7effaada6a14e"),
			UtxoSetSize: 3102076686,
			UtxoSetSources: []string{
				"http://localhost:8080/ipfs/QmQECm1yUgKh7oQg2syCFo5BLvhFSSH9m95fw41b3GTrXV",
				"https://ipfs.greyh.at/ipfs/QmQECm1yUgKh7oQg2syCFo5BLvhFSSH9m95fw41b3GTrXV",
				"https://ipfs.io/ipfs/QmQECm1yUgKh7oQg2syCFo5BLvhFSSH9m95fw41b3GTrXV",
			},
		},
		{
			Height:      582680,
			Hash:        newHashFromStr("000000000000000001b4b8e36aec7d4f9671a47872cb9a74dc16ca398c7dcc18"),
			UtxoSetHash: newHashFromStr("2406189cf8e15140c2004d4e9aa012711f48183689a305223c9bafdd3af8887e"),
			UtxoSetSize: 3178882199,
			UtxoSetSources: []string{
				"http://localhost:8080/ipfs/QmXkBQJrMKkCKNbwv4m5xtnqwU9Sq7kucPigvZW8mWxcrv",
				"https://ipfs.greyh.at/ipfs/QmXkBQJrMKkCKNbwv4m5xtnqwU9Sq7kucPigvZW8mWxcrv",
				"https://ipfs.io/ipfs/QmXkBQJrMKkCKNbwv4m5xtnqwU9Sq7kucPigvZW8mWxcrv",
			},
		},
		{
			Height:      609136,
			Hash:        newHashFromStr("000000000000000000b48bb207faac5ac655c313e41ac909322eaa694f5bc5b1"),
			UtxoSetHash: newHashFromStr("105b794be6370014bef629b61db50716eea29e2d6c712e11cad9fa9c2aaeee30"),
			UtxoSetSize: 3214790044,
			UtxoSetSources: []string{
				"http://localhost:8080/ipfs/QmZQFi5kiY1cAu6hEpLCEaAZ3FX1CtmFVtE8DiLvkYNg62",
				"https://ipfs.greyh.at/ipfs/QmZQFi5kiY1cAu6hEpLCEaAZ3FX1CtmFVtE8DiLvkYNg62",
				"https://ipfs.io/ipfs/QmZQFi5kiY1cAu6hEpLCEaAZ3FX1CtmFVtE8DiLvkYNg62",
			},
		},
		{
			Height:      635259,
			Hash:        newHashFromStr("00000000000000000033dfef1fc2d6a5d5520b078c55193a9bf498c5b27530f7"),
			UtxoSetHash: newHashFromStr("45dc882e48bf4b6d894db29512f35296ffda5ea32c2b82920aa5ae1ba056689a"),
			UtxoSetSize: 3282796875,
			UtxoSetSources: []string{
				"http://localhost:8080/ipfs/QmYhcrsLgGfRTuxoZUCPCEj5xzZx5sAgV32Z7p1qPerJBr",
				"https://ipfs.greyh.at/ipfs/QmYhcrsLgGfRTuxoZUCPCEj5xzZx5sAgV32Z7p1qPerJBr",
				"https://ipfs.io/ipfs/QmYhcrsLgGfRTuxoZUCPCEj5xzZx5sAgV32Z7p1qPerJBr",
			},
		},
		{
			Height:      661648,
			Hash:        newHashFromStr("0000000000000000029e471c41818d24b8b74c911071c4ef0b4a0509f9b5a8ce"),
			UtxoSetHash: newHashFromStr("fff228b2f788d2be35868fc2517d2557f856cbb9d6e2dad7310ab6054a29ef67"),
			UtxoSetSize: 2931107971,
			UtxoSetSources: []string{
				"http://localhost:8080/ipfs/QmY9Anst9NB42RVSGZehNCF52B2DxAzAYXEPrLrar75VMT",
				"https://ipfs.greyh.at/ipfs/QmY9Anst9NB42RVSGZehNCF52B2DxAzAYXEPrLrar75VMT",
				"https://ipfs.io/ipfs/QmY9Anst9NB42RVSGZehNCF52B2DxAzAYXEPrLrar75VMT",
			},
		},
	},

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1916, // 95% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016, //
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  1199145601, // January 1, 2008 UTC
			ExpireTime: 1230767999, // December 31, 2008 UTC
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  1462060800, // May 1st, 2016
			ExpireTime: 1493596800, // May 1st, 2017
		},
	},

	// Mempool parameters
	RelayNonStdTxs: false,

	// The prefix for the cashaddress
	CashAddressPrefix: "bitcoincash", // always bitcoincash for mainnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x00, // starts with 1
	LegacyScriptHashAddrID: 0x05, // starts with 3
	PrivateKeyID:           0x80, // starts with 5 (uncompressed) or K (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e}, // starts with xpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 145,

	// slp indexer parameters
	SlpIndexStartHeight: 543374,
	SlpIndexStartHash:   newHashFromStr("0000000000000000020322dc9d6da5bb55c4c12aa5040a7c4c2f673e28a5b9f0"),
	SlpAddressPrefix:    "simpleledger",
}

// RegressionNetParams defines the network parameters for the regression test
// Bitcoin network.  Not to be confused with the test Bitcoin network (version
// 3), this network is sometimes simply called "testnet".
var RegressionNetParams = Params{
	Name:        "regtest",
	Net:         wire.TestNet,
	DefaultPort: "18444",
	DNSSeeds:    []DNSSeed{},

	// Chain parameters
	GenesisBlock:     &regTestGenesisBlock,
	GenesisHash:      &regTestGenesisHash,
	PowLimit:         regressionPowLimit,
	PowLimitBits:     0x207fffff,
	CoinbaseMaturity: 100,
	BIP0034Height:    100000000, // Not active - Permit ver 1 blocks
	BIP0065Height:    1351,      // Used by regression tests
	BIP0066Height:    1251,      // Used by regression tests

	UahfForkHeight:              0, // Always active on regtest
	DaaForkHeight:               0, // Always active on regtest
	MagneticAnonomalyForkHeight: 1000,
	PhononForkHeight:            1000,
	AxionActivationHeight:       0, // Always active on regtest

	CosmicInflationActivationTime: 1652616000,

	Upgrade9ForkHeight:      100000000,
	ABLAForkHeight:          100000000,
	Upgrade11ActivationTime: 0, // Always active on regtest

	ABLAConfig: ABLAConstants{
		Epsilon0:        500000,
		Beta0:           500000,
		N0:              845890,
		GammaReciprocal: 37938,
		ZetaXB7:         192,
		ThetaReciprocal: 37938,
		Delta:           10,
		FixedSize:       true,
	},

	SubsidyReductionInterval:             150,
	TargetTimespan:                       time.Hour * 24 * 14, // 14 days
	TargetTimePerBlock:                   time.Minute * 10,    // 10 minutes
	RetargetAdjustmentFactor:             4,                   // 25% less, 400% more
	ReduceMinDifficulty:                  true,
	NoDifficultyAdjustment:               true,
	MinDiffReductionTime:                 time.Minute * 20, // TargetTimePerBlock * 2
	AsertDifficultyHalflife:              3600,             // 1 hour
	AsertDifficultyAnchorHeight:          0,
	AsertDifficultyAnchorParentTimestamp: regTestGenesisBlock.Header.Timestamp.Unix(),
	AsertDifficultyAnchorBits:            regTestGenesisBlock.Header.Bits,
	GenerateSupported:                    true,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: nil,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 108, // 75%  of MinerConfirmationWindow
	MinerConfirmationWindow:       144,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
	},

	// Mempool parameters
	RelayNonStdTxs: true,

	// The prefix for the cashaddress
	CashAddressPrefix: "bchreg", // always bchreg for reg testnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x6f, // starts with m or n
	LegacyScriptHashAddrID: 0xc4, // starts with 2
	PrivateKeyID:           0xef, // starts with 9 (uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1, // all coins use 1

	// slp indexer parameters
	SlpIndexStartHeight: -1,
	SlpIndexStartHash:   &chainhash.Hash{},
	SlpAddressPrefix:    "slpreg",
}

// TestNet3Params defines the network parameters for the test Bitcoin network
// (version 3).  Not to be confused with the regression test network, this
// network is sometimes simply called "testnet".
var TestNet3Params = Params{
	Name:        "testnet3",
	Net:         wire.TestNet3,
	DefaultPort: "18333",
	DNSSeeds: []DNSSeed{
		{"testnet-seed.bchd.cash", true},
		{"testnet-seed-bch.bitcoinforks.org", true},
		{"seed.tbch.loping.net", true},
	},

	// Chain parameters
	GenesisBlock:  &testNet3GenesisBlock,
	GenesisHash:   &testNet3GenesisHash,
	PowLimit:      testNet3PowLimit,
	PowLimitBits:  0x1d00ffff,
	BIP0034Height: 21111,  // 0000000023b3a96d3484e5abb3755c413e7d41500f8e2a5c3f0dd01299cd8ef8
	BIP0065Height: 581885, // 00000000007f6655f22f98e72ed80d8b06dc761d5da09df0fa1dc4be4f861eb6
	BIP0066Height: 330776, // 000000002104c8c45e99a8853285a3b592602a3ccde2b832481da85e9e4ba182

	UahfForkHeight:              1155875, // 00000000f17c850672894b9a75b63a1e72830bbd5f4c8889b5c1a80e7faef138
	DaaForkHeight:               1188697, // 0000000000170ed0918077bde7b4d36cc4c91be69fa09211f748240dabe047fb
	MagneticAnonomalyForkHeight: 1267996, // 00000000000001fae0095cd4bea16f1ce8ab63f3f660a03c6d8171485f484b24
	GreatWallForkHeight:         1303884, // 00000000000001a749d7aa418c582a0e234ebc15643bf23a4f3107fa55120388
	GravitonForkHeight:          1341711, // 00000000c678f67ea16d5bf803f68ce42991839d13849f77332d6f586f62d421
	PhononForkHeight:            1378460, // 0000000070f33c64cb94629680fbc57d17bea354a73e693affcb366d023db324
	AxionActivationHeight:       1421481, // 00000000062c7f32591d883c99fc89ebe74a83287c0f2b7ffeef72e62217d40b

	CosmicInflationActivationTime: 1652616000,

	Upgrade9ForkHeight: 1552787,
	ABLAForkHeight:     1605520,

	Upgrade11ActivationTime: 1747310400,

	ABLAConfig: ABLAConstants{
		Epsilon0:        16000000,
		Beta0:           16000000,
		N0:              845890,
		GammaReciprocal: 37938,
		ZetaXB7:         192,
		ThetaReciprocal: 37938,
		Delta:           10,
		FixedSize:       true,
	},

	CoinbaseMaturity:                     100,
	SubsidyReductionInterval:             210000,
	TargetTimespan:                       time.Hour * 24 * 14, // 14 days
	TargetTimePerBlock:                   time.Minute * 10,    // 10 minutes
	RetargetAdjustmentFactor:             4,                   // 25% less, 400% more
	ReduceMinDifficulty:                  true,
	NoDifficultyAdjustment:               false,
	MinDiffReductionTime:                 time.Minute * 20, // TargetTimePerBlock * 2
	AsertDifficultyHalflife:              3600,             // 1 hour
	AsertDifficultyAnchorHeight:          1421481,
	AsertDifficultyAnchorParentTimestamp: 1605445400,
	AsertDifficultyAnchorBits:            486604799,
	GenerateSupported:                    false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{
		{Height: 546, Hash: newHashFromStr("000000002a936ca763904c3c35fce2f3556c559c0214345d31b1bcebf76acb70")},
		{Height: 100000, Hash: newHashFromStr("00000000009e2958c15ff9290d571bf9459e93b19765c6801ddeccadbb160a1e")},
		{Height: 200000, Hash: newHashFromStr("0000000000287bffd321963ef05feab753ebe274e1d78b2fd4e2bfe9ad3aa6f2")},
		{Height: 300001, Hash: newHashFromStr("0000000000004829474748f3d1bc8fcf893c88be255e6d7f571c548aff57abf4")},
		{Height: 400002, Hash: newHashFromStr("0000000005e2c73b8ecb82ae2dbc2e8274614ebad7172b53528aba7501f5a089")},
		{Height: 500011, Hash: newHashFromStr("00000000000929f63977fbac92ff570a9bd9e7715401ee96f2848f7b07750b02")},
		{Height: 600002, Hash: newHashFromStr("000000000001f471389afd6ee94dcace5ccc44adc18e8bff402443f034b07240")},
		{Height: 700000, Hash: newHashFromStr("000000000000406178b12a4dea3b27e13b3c4fe4510994fd667d7c1e6a3f4dc1")},
		{Height: 800010, Hash: newHashFromStr("000000000017ed35296433190b6829db01e657d80631d43f5983fa403bfdb4c1")},
		{Height: 900000, Hash: newHashFromStr("0000000000356f8d8924556e765b7a94aaebc6b5c8685dcfa2b1ee8b41acd89b")},
		{Height: 1000007, Hash: newHashFromStr("00000000001ccb893d8a1f25b70ad173ce955e5f50124261bbbc50379a612ddf")},
		{Height: 1341712, Hash: newHashFromStr("00000000fffc44ea2e202bd905a9fbbb9491ef9e9d5a9eed4039079229afa35b")},
		{Height: 1378461, Hash: newHashFromStr("0000000099f5509b5f36b1926bcf82b21d936ebeadee811030dfbbb7fae915d7")},
		{Height: 1421482, Hash: newHashFromStr("0000000023e0680a8a062b3cc289a4a341124ce7fcb6340ede207e194d73b60a")},
	},

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1512, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  1199145601, // January 1, 2008 UTC
			ExpireTime: 1230767999, // December 31, 2008 UTC
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  1456790400, // March 1st, 2016
			ExpireTime: 1493596800, // May 1st, 2017
		},
	},

	// Mempool parameters
	RelayNonStdTxs: true,

	// The prefix for the cashaddress
	CashAddressPrefix: "bchtest", // always bchtest for testnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x6f, // starts with m or n
	LegacyScriptHashAddrID: 0xc4, // starts with 2
	PrivateKeyID:           0xef, // starts with 9 (uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1, // all coins use 1

	// slp indexer parameters
	SlpIndexStartHeight: 1253800,
	SlpIndexStartHash:   newHashFromStr("000000000000e8d3d53ad5dcb555d1119b650c8f3e1cb7d106764b455a87b10a"),
	SlpAddressPrefix:    "slptest",
}

var ChipNetParams = Params{
	Name:        "chipnet",
	Net:         wire.ChipNet,
	DefaultPort: "48333",
	DNSSeeds: []DNSSeed{
		{"chipnet.bitjson.com", true},
	},

	// Chain parameters
	GenesisBlock:  &testNet4GenesisBlock, // Same value as testnet4
	GenesisHash:   &testNet4GenesisHash,  // Same value as testnet4
	PowLimit:      testNet3PowLimit,      // Same value as testnet3
	PowLimitBits:  0x1d00ffff,
	BIP0034Height: 2,
	BIP0065Height: 3,
	BIP0066Height: 4,
	CSVHeight:     5,

	UahfForkHeight:              5,
	DaaForkHeight:               3000,
	MagneticAnonomalyForkHeight: 3999,
	GreatWallForkHeight:         0,
	GravitonForkHeight:          4999,
	PhononForkHeight:            0,
	AxionActivationHeight:       16844,

	CosmicInflationActivationTime: 1637694000,

	Upgrade9ForkHeight: 121956,
	ABLAForkHeight:     174519,

	Upgrade11ActivationTime: 1731672000,

	//	Reference for the following constant values: https://gitlab.com/0353F40E/ebaa/-/blob/main/README.md#testnets
	ABLAConfig: ABLAConstants{
		Epsilon0:        1000000,
		Beta0:           1000000,
		N0:              174519,
		GammaReciprocal: 37938,
		ZetaXB7:         192,
		ThetaReciprocal: 37938,
		Delta:           10,
		FixedSize:       false,
	},

	CoinbaseMaturity:                     100,
	SubsidyReductionInterval:             210000,
	TargetTimespan:                       time.Hour * 24 * 14, // 14 days
	TargetTimePerBlock:                   time.Minute * 10,    // 10 minutes
	RetargetAdjustmentFactor:             4,                   // 25% less, 400% more
	ReduceMinDifficulty:                  true,
	NoDifficultyAdjustment:               false,
	MinDiffReductionTime:                 time.Minute * 20, // TargetTimePerBlock * 2
	AsertDifficultyHalflife:              3600,             // 1 hour
	AsertDifficultyAnchorHeight:          16844,
	AsertDifficultyAnchorParentTimestamp: 1605451779,
	AsertDifficultyAnchorBits:            0x1d00ffff,
	GenerateSupported:                    false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{},

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1512, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  1199145601, // January 1, 2008 UTC
			ExpireTime: 1230767999, // December 31, 2008 UTC
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  1456790400, // March 1st, 2016
			ExpireTime: 1493596800, // May 1st, 2017
		},
	},

	// Mempool parameters
	RelayNonStdTxs: false,

	// The prefix for the cashaddress
	CashAddressPrefix: "bchtest", // always bchtest for testnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x6f, // starts with m or n
	LegacyScriptHashAddrID: 0xc4, // starts with 2
	PrivateKeyID:           0xef, // starts with 9 (uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1, // all coins use 1

	// slp indexer parameters
	SlpIndexStartHeight: 0,
	SlpIndexStartHash:   newHashFromStr("000000001dd410c49a788668ce26751718cc797474d3152a5fc073dd44fd9f7b"),
	SlpAddressPrefix:    "slptest",
}

// TestNet4Params defines the network parameters for the test Bitcoin network
// (version 4).  Not to be confused with the regression test network, this
// network is sometimes simply called "testnet".
var TestNet4Params = Params{
	Name:        "testnet4",
	Net:         wire.TestNet4,
	DefaultPort: "28333",
	DNSSeeds: []DNSSeed{
		{"testnet4-seed.bchd.cash", true},
		{"testnet4.imaginary.cash", true},
		{"testnet4-seed-bch.toom.im", true},
		{"seed.tbch4.loping.net", true},
		{"testnet4-seed.flowee.cash", true},
	},

	// Chain parameters
	GenesisBlock:  &testNet4GenesisBlock,
	GenesisHash:   &testNet4GenesisHash,
	PowLimit:      testNet3PowLimit,
	PowLimitBits:  0x1d00ffff,
	BIP0034Height: 2,
	BIP0065Height: 3,
	BIP0066Height: 4,
	CSVHeight:     5,

	UahfForkHeight:              5,
	DaaForkHeight:               3000,
	MagneticAnonomalyForkHeight: 3999,
	GreatWallForkHeight:         0,
	GravitonForkHeight:          4999,
	PhononForkHeight:            0,
	AxionActivationHeight:       16844,

	CosmicInflationActivationTime: 1637694000,

	Upgrade9ForkHeight: 148043,
	ABLAForkHeight:     200740,

	Upgrade11ActivationTime: 1747310400,

	ABLAConfig: ABLAConstants{
		Epsilon0:        1000000,
		Beta0:           1000000,
		N0:              845890,
		GammaReciprocal: 37938,
		ZetaXB7:         192,
		ThetaReciprocal: 37938,
		Delta:           10,
		FixedSize:       true,
	},

	CoinbaseMaturity:                     100,
	SubsidyReductionInterval:             210000,
	TargetTimespan:                       time.Hour * 24 * 14, // 14 days
	TargetTimePerBlock:                   time.Minute * 10,    // 10 minutes
	RetargetAdjustmentFactor:             4,                   // 25% less, 400% more
	ReduceMinDifficulty:                  true,
	NoDifficultyAdjustment:               false,
	MinDiffReductionTime:                 time.Minute * 20, // TargetTimePerBlock * 2
	AsertDifficultyHalflife:              3600,             // 1 hour
	AsertDifficultyAnchorHeight:          16844,
	AsertDifficultyAnchorParentTimestamp: 1605451779,
	AsertDifficultyAnchorBits:            0x1d00ffff,
	GenerateSupported:                    false,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{},

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1512, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  1199145601, // January 1, 2008 UTC
			ExpireTime: 1230767999, // December 31, 2008 UTC
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  1456790400, // March 1st, 2016
			ExpireTime: 1493596800, // May 1st, 2017
		},
	},

	// Mempool parameters
	RelayNonStdTxs: false,

	// The prefix for the cashaddress
	CashAddressPrefix: "bchtest", // always bchtest for testnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x6f, // starts with m or n
	LegacyScriptHashAddrID: 0xc4, // starts with 2
	PrivateKeyID:           0xef, // starts with 9 (uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1, // all coins use 1

	// slp indexer parameters
	SlpIndexStartHeight: 0,
	SlpIndexStartHash:   newHashFromStr("000000001dd410c49a788668ce26751718cc797474d3152a5fc073dd44fd9f7b"),
	SlpAddressPrefix:    "slptest",
}

// SimNetParams defines the network parameters for the simulation test Bitcoin
// network.  This network is similar to the normal test network except it is
// intended for private use within a group of individuals doing simulation
// testing.  The functionality is intended to differ in that the only nodes
// which are specifically specified are used to create the network rather than
// following normal discovery rules.  This is important as otherwise it would
// just turn into another public testnet.
var SimNetParams = Params{
	Name:        "simnet",
	Net:         wire.SimNet,
	DefaultPort: "18555",
	DNSSeeds:    []DNSSeed{}, // NOTE: There must NOT be any seeds.

	// Chain parameters
	GenesisBlock:                         &simNetGenesisBlock,
	GenesisHash:                          &simNetGenesisHash,
	PowLimit:                             simNetPowLimit,
	PowLimitBits:                         0x207fffff,
	BIP0034Height:                        0, // Always active on simnet
	BIP0065Height:                        0, // Always active on simnet
	BIP0066Height:                        0, // Always active on simnet
	UahfForkHeight:                       0, // Always active on simnet
	DaaForkHeight:                        2000,
	MagneticAnonomalyForkHeight:          3000,
	GreatWallForkHeight:                  0,
	AxionActivationHeight:                4000,
	CosmicInflationActivationTime:        0,
	CoinbaseMaturity:                     100,
	SubsidyReductionInterval:             210000,
	TargetTimespan:                       time.Hour * 24 * 14, // 14 days
	TargetTimePerBlock:                   time.Minute * 10,    // 10 minutes
	RetargetAdjustmentFactor:             4,                   // 25% less, 400% more
	ReduceMinDifficulty:                  true,
	NoDifficultyAdjustment:               true,
	MinDiffReductionTime:                 time.Minute * 20, // TargetTimePerBlock * 2
	AsertDifficultyHalflife:              3600,             // 1 hour
	AsertDifficultyAnchorHeight:          0,
	AsertDifficultyAnchorParentTimestamp: simNetGenesisBlock.Header.Timestamp.Unix(),
	AsertDifficultyAnchorBits:            simNetGenesisBlock.Header.Bits,
	GenerateSupported:                    true,

	Upgrade9ForkHeight:      100000000,
	ABLAForkHeight:          100000000,
	Upgrade11ActivationTime: 0, // Always active on simnet

	ABLAConfig: ABLAConstants{
		Epsilon0:        16000000,
		Beta0:           16000000,
		N0:              845890,
		GammaReciprocal: 37938,
		ZetaXB7:         192,
		ThetaReciprocal: 37938,
		Delta:           10,
		FixedSize:       true,
	},

	// Checkpoints ordered from oldest to newest.
	Checkpoints: nil,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 75, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       100,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
	},

	// Mempool parameters
	RelayNonStdTxs: true,

	// The prefix for the cashaddress
	CashAddressPrefix: "bchsim", // always bchsim for simnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x3f, // starts with S
	LegacyScriptHashAddrID: 0x7b, // starts with s
	PrivateKeyID:           0x64, // starts with 4 (uncompressed) or F (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x20, 0xb9, 0x00}, // starts with sprv
	HDPublicKeyID:  [4]byte{0x04, 0x20, 0xbd, 0x3a}, // starts with spub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 115, // ASCII for s
}

var (
	// ErrDuplicateNet describes an error where the parameters for a Bitcoin
	// network could not be set due to the network already being a standard
	// network or previously-registered into this package.
	ErrDuplicateNet = errors.New("duplicate Bitcoin network")

	// ErrUnknownHDKeyID describes an error where the provided id which
	// is intended to identify the network for a hierarchical deterministic
	// private extended key is not registered.
	ErrUnknownHDKeyID = errors.New("unknown hd private extended key bytes")
)

var (
	registeredNets      = make(map[wire.BitcoinNet]struct{})
	pubKeyHashAddrIDs   = make(map[byte]struct{})
	scriptHashAddrIDs   = make(map[byte]struct{})
	cashAddressPrefixes = make(map[string]struct{})
	hdPrivToPubKeyIDs   = make(map[[4]byte][]byte)
)

// String returns the hostname of the DNS seed in human-readable form.
func (d DNSSeed) String() string {
	return d.Host
}

// Register registers the network parameters for a Bitcoin network.  This may
// error with ErrDuplicateNet if the network is already registered (either
// due to a previous Register call, or the network being one of the default
// networks).
//
// Network parameters should be registered into this package by a main package
// as early as possible.  Then, library packages may lookup networks or network
// parameters based on inputs and work regardless of the network being standard
// or not.
func Register(params *Params) error {
	if _, ok := registeredNets[params.Net]; ok {
		return ErrDuplicateNet
	}
	registeredNets[params.Net] = struct{}{}
	pubKeyHashAddrIDs[params.LegacyPubKeyHashAddrID] = struct{}{}
	scriptHashAddrIDs[params.LegacyScriptHashAddrID] = struct{}{}
	hdPrivToPubKeyIDs[params.HDPrivateKeyID] = params.HDPublicKeyID[:]

	// A valid cashaddress prefix for the given net followed by ':'.
	cashAddressPrefixes[params.CashAddressPrefix+":"] = struct{}{}
	return nil
}

// mustRegister performs the same function as Register except it panics if there
// is an error.  This should only be called from package init functions.
func mustRegister(params *Params) {
	if err := Register(params); err != nil {
		panic("failed to register network: " + err.Error())
	}
}

// IsPubKeyHashAddrID returns whether the id is an identifier known to prefix a
// pay-to-pubkey-hash address on any default or registered network.  This is
// used when decoding an address string into a specific address type.  It is up
// to the caller to check both this and IsScriptHashAddrID and decide whether an
// address is a pubkey hash address, script hash address, neither, or
// undeterminable (if both return true).
func IsPubKeyHashAddrID(id byte) bool {
	_, ok := pubKeyHashAddrIDs[id]
	return ok
}

// IsScriptHashAddrID returns whether the id is an identifier known to prefix a
// pay-to-script-hash address on any default or registered network.  This is
// used when decoding an address string into a specific address type.  It is up
// to the caller to check both this and IsPubKeyHashAddrID and decide whether an
// address is a pubkey hash address, script hash address, neither, or
// undeterminable (if both return true).
func IsScriptHashAddrID(id byte) bool {
	_, ok := scriptHashAddrIDs[id]
	return ok
}

// IsCashAddressPrefix returns whether the prefix is a known prefix for the
// cashaddress on any default or registered network.  This is used when decoding
// an address string into a specific address type.
func IsCashAddressPrefix(prefix string) bool {
	prefix = strings.ToLower(prefix)
	_, ok := cashAddressPrefixes[prefix]
	return ok
}

// HDPrivateKeyToPublicKeyID accepts a private hierarchical deterministic
// extended key id and returns the associated public key id.  When the provided
// id is not registered, the ErrUnknownHDKeyID error will be returned.
func HDPrivateKeyToPublicKeyID(id []byte) ([]byte, error) {
	if len(id) != 4 {
		return nil, ErrUnknownHDKeyID
	}

	var key [4]byte
	copy(key[:], id)
	pubBytes, ok := hdPrivToPubKeyIDs[key]
	if !ok {
		return nil, ErrUnknownHDKeyID
	}

	return pubBytes, nil
}

// newHashFromStr converts the passed big-endian hex string into a
// chainhash.Hash.  It only differs from the one available in chainhash in that
// it panics on an error since it will only (and must only) be called with
// hard-coded, and therefore known good, hashes.
func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
		// Ordinarily I don't like panics in library code since it
		// can take applications down without them having a chance to
		// recover which is extremely annoying, however an exception is
		// being made in this case because the only way this can panic
		// is if there is an error in the hard-coded hashes.  Thus it
		// will only ever potentially panic on init and therefore is
		// 100% predictable.
		panic(err)
	}
	return hash
}

func init() {
	// Register all default networks when the package is initialized.
	mustRegister(&MainNetParams)
	mustRegister(&TestNet3Params)
	mustRegister(&RegressionNetParams)
	mustRegister(&SimNetParams)
}
