package clob

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/bububa/polymarket-client/relayer"
)

// SplitPositionRelayer builds a splitPosition transaction, signs a PROXY relayer request,
// and submits it through the configured relayer.
func (c *Client) SplitPositionRelayer(
	ctx context.Context,
	req *SplitPositionRequest,
	relayerReq *CTFRelayerArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var tx CTFTransaction
	if err := c.BuildSplitPositionTx(req, &tx); err != nil {
		return err
	}

	var submitReq RelayerCTFRequest
	if err := c.BuildCTFRelayerRequest(ctx, &tx, relayerReq, &submitReq); err != nil {
		return err
	}

	return c.SubmitCTFRelayerTransaction(ctx, &tx, &submitReq, out)
}

// MergePositionsRelayer builds a mergePositions transaction, signs a PROXY relayer request,
// and submits it through the configured relayer.
func (c *Client) MergePositionsRelayer(
	ctx context.Context,
	req *MergePositionsRequest,
	relayerReq *CTFRelayerArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var tx CTFTransaction
	if err := c.BuildMergePositionsTx(req, &tx); err != nil {
		return err
	}

	var submitReq RelayerCTFRequest
	if err := c.BuildCTFRelayerRequest(ctx, &tx, relayerReq, &submitReq); err != nil {
		return err
	}

	return c.SubmitCTFRelayerTransaction(ctx, &tx, &submitReq, out)
}

// RedeemPositionsRelayer builds a redeemPositions transaction, signs a PROXY relayer request,
// and submits it through the configured relayer.
func (c *Client) RedeemPositionsRelayer(
	ctx context.Context,
	req *RedeemPositionsRequest,
	relayerReq *CTFRelayerArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var tx CTFTransaction
	if err := c.BuildRedeemPositionsTx(req, &tx); err != nil {
		return err
	}

	var submitReq RelayerCTFRequest
	if err := c.BuildCTFRelayerRequest(ctx, &tx, relayerReq, &submitReq); err != nil {
		return err
	}

	return c.SubmitCTFRelayerTransaction(ctx, &tx, &submitReq, out)
}

// RedeemNegRiskRelayer builds a neg-risk redeemPositions transaction, signs a PROXY relayer request,
// and submits it through the configured relayer.
func (c *Client) RedeemNegRiskRelayer(
	ctx context.Context,
	req *RedeemNegRiskRequest,
	relayerReq *CTFRelayerArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var tx CTFTransaction
	if err := c.BuildRedeemNegRiskTx(req, &tx); err != nil {
		return err
	}

	var submitReq RelayerCTFRequest
	if err := c.BuildCTFRelayerRequest(ctx, &tx, relayerReq, &submitReq); err != nil {
		return err
	}

	return c.SubmitCTFRelayerTransaction(ctx, &tx, &submitReq, out)
}

// BuildCTFRelayerRequest builds and signs a PROXY relayer request for built CTF calldata.
func (c *Client) BuildCTFRelayerRequest(
	ctx context.Context,
	tx *CTFTransaction,
	req *CTFRelayerArgs,
	out *RelayerCTFRequest,
) error {
	if tx == nil {
		return errors.New("polymarket: nil CTF transaction")
	}
	if out == nil {
		return errors.New("polymarket: nil relayer CTF request output")
	}
	if c.auth.Signer == nil {
		return errors.New("polymarket: signer is required")
	}
	if c.relayerClient == nil {
		return errors.New("polymarket: relayer client is required")
	}
	if req == nil {
		req = new(CTFRelayerArgs)
	}

	from := strings.TrimSpace(req.From)
	if from == "" {
		from = c.auth.Signer.Address().Hex()
	}
	if !common.IsHexAddress(from) {
		return fmt.Errorf("polymarket: relayer from must be a valid hex address")
	}

	var submit relayer.SubmitTransactionRequest
	switch req.Type {
	case relayer.NonceTypeSafe:
		builder, ok := c.relayerClient.(SafeRelayerBuilder)
		if !ok {
			return errors.New("polymarket: relayer client does not support safe request signing")
		}
		err := builder.SafeSubmitTransactionRequest(ctx, c.auth.Signer, &relayer.SafeSubmitTransactionArgs{
			From:        from,
			ProxyWallet: req.ProxyWallet,
			ChainID:     c.auth.ChainID,
			Transactions: []relayer.SafeTransaction{
				{
					To:        tx.To.Hex(),
					Operation: relayer.OperationCall,
					Data:      hexutil.Encode(tx.Data),
					Value:     "0",
				},
			},
			Metadata: req.Metadata,
		}, &submit)
		if err != nil {
			return err
		}
	case relayer.NonceTypeProxy:
		builder, ok := c.relayerClient.(ProxyRelayerBuilder)
		if !ok {
			return errors.New("polymarket: relayer client does not support proxy request signing")
		}
		encodedData, err := relayer.EncodeProxyTransactionData([]relayer.ProxyTransaction{
			{
				To:       tx.To.Hex(),
				TypeCode: relayer.CallTypeCall,
				Data:     hexutil.Encode(tx.Data),
				Value:    "0",
			},
		})
		if err != nil {
			return err
		}
		err = builder.ProxySubmitTransactionRequest(ctx, c.auth.Signer, &relayer.ProxySubmitTransactionArgs{
			From:        from,
			ProxyWallet: req.ProxyWallet,
			Data:        encodedData,
			Metadata:    req.Metadata,
			GasLimit:    req.GasLimit,
		}, &submit)
		if err != nil {
			return err
		}
	case relayer.NonceTypeWallet, relayer.NonceTypeWalletCreate:
		return errors.New("polymarket: deposit wallet CTF transactions require CTFDepositWalletTransactionRequest")
	default:
		return fmt.Errorf("polymarket: unsupported relayer type %q", req.Type)
	}

	*out = RelayerCTFRequest{
		To:              submit.To,
		From:            submit.From,
		ProxyWallet:     submit.ProxyWallet,
		Data:            submit.Data,
		Nonce:           submit.Nonce,
		Signature:       submit.Signature,
		SignatureParams: submit.SignatureParams,
		Type:            submit.Type,
		Metadata:        submit.Metadata,
		Value:           submit.Value,
	}
	return nil
}
