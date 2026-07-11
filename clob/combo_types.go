package clob

import "math/big"

const MaxComboLegs = 50

// ComboConditionID is the protocol-v2 bytes31 condition identifier.
type ComboConditionID [31]byte

// ComboPositionContext contains a derived Combo condition and its YES/NO IDs.
type ComboPositionContext struct {
	ConditionID ComboConditionID
	PositionIDs [2]string
}

// ComboSplitPositionRequest describes a Combo inventory split.
type ComboSplitPositionRequest struct {
	LegPositionIDs []string
	Amount         *big.Int
}

// ComboMergePositionsRequest describes a complementary Combo inventory merge.
type ComboMergePositionsRequest struct {
	LegPositionIDs []string
	// Amount is explicit for builders; high-level methods resolve nil to max.
	Amount *big.Int
}

// ComboRedeemPositionRequest describes redemption of one resolved Combo outcome.
type ComboRedeemPositionRequest struct {
	PositionID string
	// Amount is explicit for builders; high-level methods resolve nil to balance.
	Amount *big.Int
}

// ComboOperationReceipt contains the receipts for an ordered Combo operation.
type ComboOperationReceipt struct {
	Transactions []TxReceipt
}
