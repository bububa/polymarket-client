package relayer

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/bububa/polymarket-client/internal/polyauth"
)

// WalletCreateSubmitTransactionRequest builds a WALLET-CREATE relayer submit request.
//
// WALLET-CREATE deploys a deterministic deposit wallet and does not require
// nonce, user signature, signatureParams, proxyWallet, or calldata.
func WalletCreateSubmitTransactionRequest(
	signer *polyauth.Signer,
	args *WalletCreateSubmitTransactionArgs,
	out *SubmitTransactionRequest,
) error {
	if signer == nil {
		return errors.New("relayer: signer is required")
	}
	if out == nil {
		return errors.New("relayer: submit transaction request output is nil")
	}

	var from string
	var factory string

	if args != nil {
		from = args.From
		factory = args.Factory
	}
	if from == "" {
		from = signer.Address().Hex()
	}
	if !common.IsHexAddress(from) {
		return fmt.Errorf("relayer: from must be a valid hex address")
	}

	if factory == "" {
		return errors.New("relayer: deposit wallet factory is required")
	}
	if !common.IsHexAddress(factory) {
		return fmt.Errorf("relayer: factory must be a valid hex address")
	}

	*out = SubmitTransactionRequest{
		Type: NonceTypeWalletCreate,
		From: common.HexToAddress(from).Hex(),
		To:   common.HexToAddress(factory).Hex(),
	}
	return nil
}

// WalletCreateSubmitTransactionRequest builds a WALLET-CREATE relayer submit request.
func (c *Client) WalletCreateSubmitTransactionRequest(
	_ context.Context,
	signer *polyauth.Signer,
	args *WalletCreateSubmitTransactionArgs,
	out *SubmitTransactionRequest,
) error {
	return WalletCreateSubmitTransactionRequest(signer, args, out)
}

// DeployDepositWallet submits a WALLET-CREATE request through the relayer.
func (c *Client) DeployDepositWallet(
	ctx context.Context,
	signer *polyauth.Signer,
	args *WalletCreateSubmitTransactionArgs,
	out *SubmitTransactionResponse,
) error {
	var req SubmitTransactionRequest
	if err := c.WalletCreateSubmitTransactionRequest(ctx, signer, args, &req); err != nil {
		return err
	}
	return c.SubmitTransaction(ctx, &req, out)
}
