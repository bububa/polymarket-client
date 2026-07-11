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

// ComboPositionBalance returns an owner's protocol-v2 ERC-1155 balance.
func (c *Client) ComboPositionBalance(ctx context.Context, owner common.Address, positionID string) (*big.Int, error) {
	if owner == (common.Address{}) {
		return nil, errors.New("polymarket: combo position owner is required")
	}
	position, err := parseComboPositionID(positionID)
	if err != nil {
		return nil, err
	}
	data, err := comboPositionManagerABI.Pack("balanceOf", owner, position)
	if err != nil {
		return nil, fmt.Errorf("combo: pack balanceOf: %w", err)
	}
	values, err := c.callComboPositionManager(ctx, data, "balanceOf")
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, errors.New("combo: malformed balanceOf result")
	}
	balance, ok := values[0].(*big.Int)
	if !ok {
		return nil, errors.New("combo: malformed balanceOf result")
	}
	return new(big.Int).Set(balance), nil
}

// ComboMaxMergeAmount returns min(YES balance, NO balance) for derived legs.
func (c *Client) ComboMaxMergeAmount(ctx context.Context, owner common.Address, legs []string) (*big.Int, error) {
	if owner == (common.Address{}) {
		return nil, errors.New("polymarket: combo position owner is required")
	}
	combo, err := DeriveComboPositionContext(legs)
	if err != nil {
		return nil, err
	}
	yes, _ := parseComboPositionID(combo.PositionIDs[0])
	no, _ := parseComboPositionID(combo.PositionIDs[1])
	data, err := comboPositionManagerABI.Pack("balanceOfBatch", []common.Address{owner, owner}, []*big.Int{yes, no})
	if err != nil {
		return nil, fmt.Errorf("combo: pack balanceOfBatch: %w", err)
	}
	values, err := c.callComboPositionManager(ctx, data, "balanceOfBatch")
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, errors.New("combo: malformed balanceOfBatch result")
	}
	balances, ok := values[0].([]*big.Int)
	if !ok || len(balances) != 2 || balances[0] == nil || balances[1] == nil {
		return nil, errors.New("combo: malformed balanceOfBatch result")
	}
	amount := new(big.Int).Set(balances[0])
	if balances[1].Cmp(amount) < 0 {
		amount.Set(balances[1])
	}
	if amount.Sign() == 0 {
		return nil, errors.New("polymarket: combo position has no complementary balance to merge")
	}
	return amount, nil
}

func (c *Client) callComboPositionManager(ctx context.Context, data []byte, method string) ([]any, error) {
	if c.rpcURL == "" {
		return nil, errors.New("combo: rpc url is required")
	}
	contracts, err := c.comboContracts()
	if err != nil {
		return nil, err
	}
	client, err := ethclient.DialContext(ctx, c.rpcURL)
	if err != nil {
		return nil, fmt.Errorf("combo: dial rpc: %w", err)
	}
	defer client.Close()
	result, err := client.CallContract(ctx, ethereum.CallMsg{To: &contracts.ComboPositionManager, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("combo: call %s: %w", method, err)
	}
	values, err := comboPositionManagerABI.Unpack(method, result)
	if err != nil {
		return nil, fmt.Errorf("combo: unpack %s: %w", method, err)
	}
	return values, nil
}
