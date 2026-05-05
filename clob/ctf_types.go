package clob

import (
	"math/big"

	"github.com/bububa/polymarket-client/relayer"
	"github.com/ethereum/go-ethereum/common"
)

// BinaryPartition returns the standard CTF partition for binary Polymarket markets.
//
// The official CTF docs define index set 1 as the first outcome and index set 2
// as the second outcome, so split and merge calls for binary markets use [1, 2].
func BinaryPartition() []*big.Int {
	return []*big.Int{big.NewInt(1), big.NewInt(2)}
}

// SplitPositionRequest describes a ConditionalTokens splitPosition call.
type SplitPositionRequest struct {
	// CollateralToken is the pUSD token address supplied as collateral.
	CollateralToken common.Address
	// ParentCollectionID is zero for top-level Polymarket markets.
	ParentCollectionID common.Hash
	// ConditionID is the market condition ID.
	ConditionID common.Hash
	// Partition is the list of index sets to mint; binary markets use [1, 2].
	Partition []*big.Int
	// Amount is the pUSD amount, in base units, to split into a full outcome set.
	Amount *big.Int
}

// SplitBinary returns a splitPosition request for a standard binary Polymarket market.
func SplitBinary(collateral common.Address, conditionID common.Hash, amount *big.Int) SplitPositionRequest {
	return SplitPositionRequest{
		CollateralToken:    collateral,
		ParentCollectionID: common.Hash{},
		ConditionID:        conditionID,
		Partition:          BinaryPartition(),
		Amount:             amount,
	}
}

// MergePositionsRequest describes a ConditionalTokens mergePositions call.
type MergePositionsRequest struct {
	// CollateralToken is the pUSD token address received after merging a full set.
	CollateralToken common.Address
	// ParentCollectionID is zero for top-level Polymarket markets.
	ParentCollectionID common.Hash
	// ConditionID is the market condition ID.
	ConditionID common.Hash
	// Partition is the list of index sets to burn; binary markets use [1, 2].
	Partition []*big.Int
	// Amount is the number of full outcome sets to merge, in base units.
	Amount *big.Int
}

// MergeBinary returns a mergePositions request for a standard binary Polymarket market.
func MergeBinary(collateral common.Address, conditionID common.Hash, amount *big.Int) MergePositionsRequest {
	return MergePositionsRequest{
		CollateralToken:    collateral,
		ParentCollectionID: common.Hash{},
		ConditionID:        conditionID,
		Partition:          BinaryPartition(),
		Amount:             amount,
	}
}

// RedeemPositionsRequest describes a ConditionalTokens redeemPositions call.
type RedeemPositionsRequest struct {
	// CollateralToken is the pUSD token address redeemed after market resolution.
	CollateralToken common.Address
	// ParentCollectionID is zero for top-level Polymarket markets.
	ParentCollectionID common.Hash
	// ConditionID is the resolved market condition ID.
	ConditionID common.Hash
	// IndexSets are the outcome index sets to redeem; binary markets use [1, 2].
	IndexSets []*big.Int
}

// RedeemBinary returns a redeemPositions request for a resolved binary Polymarket market.
func RedeemBinary(collateral common.Address, conditionID common.Hash) RedeemPositionsRequest {
	return RedeemPositionsRequest{
		CollateralToken:    collateral,
		ParentCollectionID: common.Hash{},
		ConditionID:        conditionID,
		IndexSets:          BinaryPartition(),
	}
}

// RedeemNegRiskRequest describes a neg-risk adapter redemption call.
type RedeemNegRiskRequest struct {
	// ConditionID is the resolved neg-risk condition ID.
	ConditionID common.Hash
	// Amounts contains the per-index-set amounts expected by the neg-risk adapter.
	Amounts []*big.Int
}

// TxReceipt is the minimal transaction receipt returned by CTF helpers.
type TxReceipt struct {
	// Hash is the transaction hash.
	Hash common.Hash
	// BlockNumber is the block that included the transaction.
	BlockNumber uint64
}

// CTFTransaction contains the destination contract and calldata for a CTF operation.
type CTFTransaction struct {
	// To is the contract address that should receive the transaction.
	To common.Address
	// Data is the ABI-encoded calldata.
	Data []byte
}

// RelayerCTFRequest contains the relayer metadata needed to submit CTF calldata.
type RelayerCTFRequest struct {
	// To is the contract address that should receive the transaction.
	To string
	// From is the signer address.
	From string
	// ProxyWallet is the user's Polymarket proxy wallet.
	ProxyWallet string
	// Data is the ABI-encoded calldata.
	Data string
	// Nonce is the relayer transaction nonce.
	Nonce string
	// Signature is the 0x-prefixed Safe or proxy transaction signature.
	Signature string
	// SignatureParams are Safe transaction parameters.
	SignatureParams relayer.SignatureParams
	// Type is the relayer transaction type, typically SAFE or PROXY.
	Type relayer.NonceType
	// Metadata is optional caller-provided transaction metadata.
	Metadata string
	// Value is the native token value for the call when required.
	Value string
}

// CTFRelayerArgs contains high-level relayer options for CTF calls.
type CTFRelayerArgs struct {
	// Type selects SAFE or PROXY relayer signing.
	Type relayer.NonceType
	// From is the EOA signer address. If empty, client signer address is used.
	From string
	// ProxyWallet is the user's Polymarket proxy wallet / funder address.
	ProxyWallet string
	// Metadata is optional relayer metadata.
	Metadata string
	// GasLimit optionally overrides the computed proxy gas limit.
	GasLimit string
}
