package clob

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bububa/polymarket-client/relayer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ComboSplitPositionRelayer submits an atomic SAFE/PROXY Combo split batch.
func (c *Client) ComboSplitPositionRelayer(ctx context.Context, req *ComboSplitPositionRequest, args *CTFRelayerArgs, out *relayer.SubmitTransactionResponse) error {
	var txs []CTFTransaction
	if err := c.BuildComboSplitPositionTxs(req, &txs); err != nil {
		return err
	}
	return c.submitComboRelayer(ctx, txs, args, out)
}

// ComboMergePositionsRelayer submits an atomic SAFE/PROXY Combo merge batch.
func (c *Client) ComboMergePositionsRelayer(ctx context.Context, req *ComboMergePositionsRequest, args *CTFRelayerArgs, out *relayer.SubmitTransactionResponse) error {
	if args == nil || !common.IsHexAddress(args.ProxyWallet) {
		return errors.New("polymarket: valid proxy wallet is required")
	}
	resolved, err := c.resolveComboMergeAmount(ctx, req, common.HexToAddress(args.ProxyWallet))
	if err != nil {
		return err
	}
	var txs []CTFTransaction
	if err := c.BuildComboMergePositionsTxs(resolved, &txs); err != nil {
		return err
	}
	return c.submitComboRelayer(ctx, txs, args, out)
}

// ComboRedeemPositionRelayer submits a SAFE/PROXY Combo redemption.
func (c *Client) ComboRedeemPositionRelayer(ctx context.Context, req *ComboRedeemPositionRequest, args *CTFRelayerArgs, out *relayer.SubmitTransactionResponse) error {
	if args == nil || !common.IsHexAddress(args.ProxyWallet) {
		return errors.New("polymarket: valid proxy wallet is required")
	}
	resolved, err := c.resolveComboRedeemAmount(ctx, req, common.HexToAddress(args.ProxyWallet))
	if err != nil {
		return err
	}
	var tx CTFTransaction
	if err := c.BuildComboRedeemPositionTx(resolved, &tx); err != nil {
		return err
	}
	return c.submitComboRelayer(ctx, []CTFTransaction{tx}, args, out)
}

func (c *Client) submitComboRelayer(ctx context.Context, txs []CTFTransaction, args *CTFRelayerArgs, out *relayer.SubmitTransactionResponse) error {
	if c.auth.Signer == nil {
		return errors.New("polymarket: signer is required")
	}
	if c.relayerClient == nil {
		return errors.New("polymarket: relayer client is required")
	}
	if args == nil {
		args = new(CTFRelayerArgs)
	}
	if len(txs) == 0 {
		return errors.New("polymarket: combo relayer transactions are required")
	}
	from := strings.TrimSpace(args.From)
	if from == "" {
		from = c.auth.Signer.Address().Hex()
	}
	if !common.IsHexAddress(from) || !common.IsHexAddress(args.ProxyWallet) {
		return errors.New("polymarket: relayer from and proxy wallet must be valid hex addresses")
	}
	var submit relayer.SubmitTransactionRequest
	switch args.Type {
	case relayer.NonceTypeSafe:
		builder, ok := c.relayerClient.(SafeRelayerBuilder)
		if !ok {
			return errors.New("polymarket: relayer client does not support safe request signing")
		}
		safeTxs := make([]relayer.SafeTransaction, len(txs))
		for i, tx := range txs {
			safeTxs[i] = relayer.SafeTransaction{To: tx.To.Hex(), Operation: relayer.OperationCall, Data: hexutil.Encode(tx.Data), Value: "0"}
		}
		if err := builder.SafeSubmitTransactionRequest(ctx, c.auth.Signer, &relayer.SafeSubmitTransactionArgs{
			From: from, ProxyWallet: args.ProxyWallet, ChainID: c.auth.ChainID,
			Transactions: safeTxs, Metadata: args.Metadata,
		}, &submit); err != nil {
			return err
		}
	case relayer.NonceTypeProxy:
		builder, ok := c.relayerClient.(ProxyRelayerBuilder)
		if !ok {
			return errors.New("polymarket: relayer client does not support proxy request signing")
		}
		proxyTxs := make([]relayer.ProxyTransaction, len(txs))
		for i, tx := range txs {
			proxyTxs[i] = relayer.ProxyTransaction{To: tx.To.Hex(), TypeCode: relayer.CallTypeCall, Data: hexutil.Encode(tx.Data), Value: "0"}
		}
		data, err := relayer.EncodeProxyTransactionData(proxyTxs)
		if err != nil {
			return fmt.Errorf("polymarket: encode combo proxy transactions: %w", err)
		}
		if err := builder.ProxySubmitTransactionRequest(ctx, c.auth.Signer, &relayer.ProxySubmitTransactionArgs{
			From: from, ProxyWallet: args.ProxyWallet, Data: data,
			Metadata: args.Metadata, GasLimit: args.GasLimit,
		}, &submit); err != nil {
			return err
		}
	case relayer.NonceTypeWallet, relayer.NonceTypeWalletCreate:
		return errors.New("polymarket: deposit wallet combo transactions require the deposit-wallet combo API")
	default:
		return fmt.Errorf("polymarket: unsupported relayer type %q", args.Type)
	}
	return c.SubmitRelayerTransaction(ctx, &submit, out)
}
