package clob

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/bububa/polymarket-client/relayer"
)

// CTFDepositWalletTransactionRequest wraps a CTF transaction into a
// signed deposit-wallet WALLET batch relayer request.
func (c *Client) CTFDepositWalletTransactionRequest(
	ctx context.Context,
	tx *CTFTransaction,
	args *DepositWalletCTFArgs,
	out *relayer.SubmitTransactionRequest,
) error {
	if tx == nil {
		return errors.New("polymarket: nil CTF transaction")
	}
	if tx.To == (common.Address{}) {
		return errors.New("polymarket: CTF transaction target is required")
	}
	if len(tx.Data) == 0 {
		return errors.New("polymarket: CTF transaction data is required")
	}
	if args == nil {
		return errors.New("polymarket: submit transaction request args is nil")
	}

	return c.DepositWalletBatchRelayerRequest(ctx, &DepositWalletBatchArgs{
		From:          args.From,
		Factory:       args.Factory,
		DepositWallet: args.DepositWallet,
		Nonce:         args.Nonce,
		Deadline:      args.Deadline,
		Calls: []relayer.DepositWalletCall{
			{
				Target: tx.To.Hex(),
				Value:  "0",
				Data:   hexutil.Encode(tx.Data),
			},
		},
		Metadata: args.Metadata,
	}, out)
}

// SubmitCTFDepositWalletTransaction builds, signs, and submits a CTF transaction as a
// deposit-wallet WALLET batch through the configured relayer.
func (c *Client) SubmitCTFDepositWalletTransaction(
	ctx context.Context,
	tx *CTFTransaction,
	args *DepositWalletCTFArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var req relayer.SubmitTransactionRequest
	if err := c.CTFDepositWalletTransactionRequest(ctx, tx, args, &req); err != nil {
		return err
	}
	return c.SubmitRelayerTransaction(ctx, &req, out)
}
