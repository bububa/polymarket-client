package clob

import (
	"context"

	"github.com/bububa/polymarket-client/relayer"
)

// SplitPositionWithDepositWallet builds, signs, and submits a
// splitPosition call through a deposit-wallet WALLET transaction.
func (c *Client) SplitPositionWithDepositWallet(
	ctx context.Context,
	req *SplitPositionRequest,
	args *DepositWalletCTFArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var tx CTFTransaction
	if err := c.BuildSplitPositionTx(req, &tx); err != nil {
		return err
	}
	return c.SubmitCTFDepositWalletTransaction(ctx, &tx, args, out)
}

// MergePositionsWithDepositWallet builds, signs, and submits a
// mergePositions call through a deposit-wallet WALLET transaction.
func (c *Client) MergePositionsWithDepositWallet(
	ctx context.Context,
	req *MergePositionsRequest,
	args *DepositWalletCTFArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var tx CTFTransaction
	if err := c.BuildMergePositionsTx(req, &tx); err != nil {
		return err
	}
	return c.SubmitCTFDepositWalletTransaction(ctx, &tx, args, out)
}

// RedeemPositionsWithDepositWallet builds, signs, and submits a
// redeemPositions call through a deposit-wallet WALLET transaction.
func (c *Client) RedeemPositionsWithDepositWallet(
	ctx context.Context,
	req *RedeemPositionsRequest,
	args *DepositWalletCTFArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var tx CTFTransaction
	if err := c.BuildRedeemPositionsTx(req, &tx); err != nil {
		return err
	}
	return c.SubmitCTFDepositWalletTransaction(ctx, &tx, args, out)
}

// RedeemNegRiskWithDepositWallet builds, signs, and submits a
// neg-risk redeemPositions call through a deposit-wallet WALLET transaction.
func (c *Client) RedeemNegRiskWithDepositWallet(
	ctx context.Context,
	req *RedeemNegRiskRequest,
	args *DepositWalletCTFArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var tx CTFTransaction
	if err := c.BuildRedeemNegRiskTx(req, &tx); err != nil {
		return err
	}
	return c.SubmitCTFDepositWalletTransaction(ctx, &tx, args, out)
}
