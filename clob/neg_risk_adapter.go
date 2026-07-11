package clob

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// GetFeeBps returns the fee, in basis points, configured for a neg-risk market.
// The underlying NegRiskAdapter contract calls this value feeBips.
func (c *Client) GetFeeBps(ctx context.Context, marketID common.Hash) (*big.Int, error) {
	return c.callNegRiskAdapterUint256(ctx, "getFeeBips", marketID)
}

// GetQuestionCount returns the number of questions prepared for a neg-risk market.
func (c *Client) GetQuestionCount(ctx context.Context, marketID common.Hash) (*big.Int, error) {
	return c.callNegRiskAdapterUint256(ctx, "getQuestionCount", marketID)
}

func (c *Client) callNegRiskAdapterUint256(ctx context.Context, method string, marketID common.Hash) (*big.Int, error) {
	if err := validateHashRequired("market id", marketID); err != nil {
		return nil, err
	}
	if c.rpcURL == "" {
		return nil, errors.New("ctf: rpc url is required")
	}
	to, err := c.contractAddress(func(config ContractConfig) common.Address { return config.NegRiskAdapter })
	if err != nil {
		return nil, err
	}
	data, err := negRiskABI.Pack(method, marketID)
	if err != nil {
		return nil, fmt.Errorf("ctf: pack neg-risk adapter %s: %w", method, err)
	}
	ec, err := ethclient.DialContext(ctx, c.rpcURL)
	if err != nil {
		return nil, fmt.Errorf("ctf: dial rpc: %w", err)
	}
	defer ec.Close()

	result, err := ec.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("ctf: call neg-risk adapter %s: %w", method, err)
	}
	values, err := negRiskABI.Unpack(method, result)
	if err != nil {
		return nil, fmt.Errorf("ctf: decode neg-risk adapter %s: %w", method, err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("ctf: decode neg-risk adapter %s: expected one result", method)
	}
	value, ok := values[0].(*big.Int)
	if !ok || value == nil {
		return nil, fmt.Errorf("ctf: decode neg-risk adapter %s: invalid uint256 result", method)
	}
	return new(big.Int).Set(value), nil
}
