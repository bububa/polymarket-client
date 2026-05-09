package clob

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/bububa/polymarket-client/internal/polyauth"
	"github.com/bububa/polymarket-client/relayer"
)

const (
	depositWalletBatchDomainName    = "DepositWallet"
	depositWalletBatchDomainVersion = "1"
)

// DepositWalletBatchRelayerRequest builds and signs a WALLET relayer submit request.
//
// This is the deposit-wallet batch path. The signature is a normal 65-byte
// EIP-712 signature over the DepositWallet Batch typed data. It is not the
// ERC-7739 wrapped CLOB order signature used by SignatureTypePoly1271 orders.
func (c *Client) DepositWalletBatchRelayerRequest(
	ctx context.Context,
	args *DepositWalletBatchArgs,
	out *relayer.SubmitTransactionRequest,
) error {
	if out == nil {
		return errors.New("polymarket: submit transaction request output is nil")
	}
	if c.auth.Signer == nil {
		return errors.New("polymarket: signer is required")
	}
	if c.auth.ChainID <= 0 {
		return errors.New("polymarket: chain id is required")
	}
	if args == nil {
		return errors.New("polymarket: submit transaction request args is nil")
	}

	from := strings.TrimSpace(args.From)
	if from == "" {
		from = c.auth.Signer.Address().Hex()
	}
	if !common.IsHexAddress(from) {
		return fmt.Errorf("polymarket: from must be a valid hex address")
	}
	from = common.HexToAddress(from).Hex()

	factory, err := c.depositWalletFactoryAddress(args.Factory)
	if err != nil {
		return err
	}

	depositWallet := strings.TrimSpace(args.DepositWallet)
	if depositWallet == "" {
		return errors.New("polymarket: deposit wallet is required")
	}
	if !common.IsHexAddress(depositWallet) {
		return fmt.Errorf("polymarket: deposit wallet must be a valid hex address")
	}
	depositWallet = common.HexToAddress(depositWallet).Hex()

	deadline := strings.TrimSpace(args.Deadline)
	if _, err := parseUint256Big("deadline", deadline); err != nil {
		return err
	}

	calls, err := normalizeDepositWalletBatchCalls(args.Calls)
	if err != nil {
		return err
	}

	nonce := strings.TrimSpace(args.Nonce)
	if nonce == "" {
		nonce, err = c.fetchDepositWalletBatchNonce(ctx, from)
		if err != nil {
			return err
		}
	}
	if _, err := parseUint256Big("nonce", nonce); err != nil {
		return err
	}

	typedData, err := BuildDepositWalletBatchTypedData(
		c.auth.ChainID,
		depositWallet,
		nonce,
		deadline,
		calls,
	)
	if err != nil {
		return err
	}

	signature, err := polyauth.SignTypedData(c.auth.Signer, typedData)
	if err != nil {
		return fmt.Errorf("polymarket: sign deposit wallet batch: %w", err)
	}

	*out = relayer.SubmitTransactionRequest{
		Type:      relayer.NonceTypeWallet,
		From:      from,
		To:        factory.Hex(),
		Nonce:     nonce,
		Signature: signature,
		DepositWalletParams: &relayer.DepositWalletParams{
			DepositWallet: depositWallet,
			Deadline:      deadline,
			Calls:         calls,
		},
		Metadata: args.Metadata,
	}

	return nil
}

// DepositWalletBatch builds, signs, and submits a WALLET batch through the configured relayer.
func (c *Client) DepositWalletBatch(
	ctx context.Context,
	args *DepositWalletBatchArgs,
	out *relayer.SubmitTransactionResponse,
) error {
	var req relayer.SubmitTransactionRequest
	if err := c.DepositWalletBatchRelayerRequest(ctx, args, &req); err != nil {
		return err
	}
	return c.SubmitRelayerTransaction(ctx, &req, out)
}

// BuildDepositWalletBatchTypedData returns the EIP-712 typed data used by deposit-wallet WALLET batches.
func BuildDepositWalletBatchTypedData(
	chainID int64,
	depositWallet string,
	nonce string,
	deadline string,
	calls []relayer.DepositWalletCall,
) (apitypes.TypedData, error) {
	if chainID <= 0 {
		return apitypes.TypedData{}, errors.New("polymarket: chain id is required")
	}
	if !common.IsHexAddress(depositWallet) {
		return apitypes.TypedData{}, fmt.Errorf("polymarket: deposit wallet must be a valid hex address")
	}

	nonceInt, err := parseUint256Big("nonce", nonce)
	if err != nil {
		return apitypes.TypedData{}, err
	}
	deadlineInt, err := parseUint256Big("deadline", deadline)
	if err != nil {
		return apitypes.TypedData{}, err
	}

	normalizedCalls, err := normalizeDepositWalletBatchCalls(calls)
	if err != nil {
		return apitypes.TypedData{}, err
	}

	callMessages := make([]any, 0, len(normalizedCalls))
	for _, call := range normalizedCalls {
		valueInt, err := parseUint256Big("call value", call.Value)
		if err != nil {
			return apitypes.TypedData{}, err
		}

		callMessages = append(callMessages, map[string]any{
			"target": common.HexToAddress(call.Target).Hex(),
			"value":  valueInt.String(),
			"data":   call.Data,
		})
	}

	depositWallet = common.HexToAddress(depositWallet).Hex()

	return apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Call": []apitypes.Type{
				{Name: "target", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "data", Type: "bytes"},
			},
			"Batch": []apitypes.Type{
				{Name: "wallet", Type: "address"},
				{Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
				{Name: "calls", Type: "Call[]"},
			},
		},
		PrimaryType: "Batch",
		Domain: apitypes.TypedDataDomain{
			Name:              depositWalletBatchDomainName,
			Version:           depositWalletBatchDomainVersion,
			ChainId:           ethmath.NewHexOrDecimal256(chainID),
			VerifyingContract: depositWallet,
		},
		Message: apitypes.TypedDataMessage{
			"wallet":   depositWallet,
			"nonce":    nonceInt.String(),
			"deadline": deadlineInt.String(),
			"calls":    callMessages,
		},
	}, nil
}

func (c *Client) depositWalletFactoryAddress(override string) (common.Address, error) {
	if strings.TrimSpace(override) != "" {
		if !common.IsHexAddress(override) {
			return common.Address{}, fmt.Errorf("polymarket: deposit wallet factory must be a valid hex address")
		}
		return common.HexToAddress(override), nil
	}

	contracts, err := Contracts(c.auth.ChainID)
	if err != nil {
		return common.Address{}, err
	}
	if contracts.DepositWalletFactory == (common.Address{}) {
		return common.Address{}, fmt.Errorf("polymarket: deposit wallet factory is not configured for chain %d", c.auth.ChainID)
	}
	return contracts.DepositWalletFactory, nil
}

func (c *Client) fetchDepositWalletBatchNonce(ctx context.Context, from string) (string, error) {
	if c.relayerClient == nil {
		return "", errors.New("polymarket: relayer client is not configured")
	}

	nonceGetter, ok := c.relayerClient.(RelayerNonceGetter)
	if !ok {
		return "", errors.New("polymarket: relayer client does not support nonce lookup")
	}

	nonceResp := relayer.NonceResponse{Address: from}
	if err := nonceGetter.GetNonce(ctx, &nonceResp, relayer.NonceTypeWallet); err != nil {
		return "", err
	}

	nonce := strings.TrimSpace(nonceResp.Nonce.String())
	if nonce == "" {
		return "", errors.New("polymarket: empty deposit wallet batch nonce")
	}
	return nonce, nil
}

func normalizeDepositWalletBatchCalls(calls []relayer.DepositWalletCall) ([]relayer.DepositWalletCall, error) {
	if len(calls) == 0 {
		return nil, errors.New("polymarket: deposit wallet batch calls are required")
	}

	out := make([]relayer.DepositWalletCall, 0, len(calls))
	for i, call := range calls {
		target := strings.TrimSpace(call.Target)
		if target == "" {
			return nil, fmt.Errorf("polymarket: deposit wallet call %d target is required", i)
		}
		if !common.IsHexAddress(target) {
			return nil, fmt.Errorf("polymarket: deposit wallet call %d target must be a valid hex address", i)
		}

		value := strings.TrimSpace(call.Value)
		if value == "" {
			value = "0"
		}
		if _, err := parseUint256Big("call value", value); err != nil {
			return nil, fmt.Errorf("polymarket: deposit wallet call %d: %w", i, err)
		}

		data := strings.TrimSpace(call.Data)
		if data == "" {
			data = "0x"
		}
		if _, err := hexutil.Decode(data); err != nil {
			return nil, fmt.Errorf("polymarket: deposit wallet call %d data must be 0x-prefixed hex: %w", i, err)
		}

		out = append(out, relayer.DepositWalletCall{
			Target: common.HexToAddress(target).Hex(),
			Value:  value,
			Data:   data,
		})
	}

	return out, nil
}
