package clob

import (
	"context"
	"errors"
	"fmt"

	"github.com/bububa/polymarket-client/relayer"
	"github.com/ethereum/go-ethereum/common"
)

func (c *Client) DepositWalletCreateRelayerRequest(out *relayer.SubmitTransactionRequest) error {
	if c.auth.Signer == nil {
		return errors.New("polymarket: signer is required")
	}
	if out == nil {
		return errors.New("polymarket: submit transaction request output is nil")
	}

	contracts, err := Contracts(c.auth.ChainID)
	if err != nil {
		return err
	}
	if contracts.DepositWalletFactory == (common.Address{}) {
		return fmt.Errorf("polymarket: deposit wallet factory is not configured for chain %d", c.auth.ChainID)
	}

	return relayer.WalletCreateSubmitTransactionRequest(c.auth.Signer, &relayer.WalletCreateSubmitTransactionArgs{
		Factory: contracts.DepositWalletFactory.Hex(),
	}, out)
}

// DeployDepositWallet submits a WALLET-CREATE request through the configured relayer.
func (c *Client) DeployDepositWallet(ctx context.Context, out *relayer.SubmitTransactionResponse) error {
	var req relayer.SubmitTransactionRequest
	if err := c.DepositWalletCreateRelayerRequest(&req); err != nil {
		return err
	}
	return c.SubmitRelayerTransaction(ctx, &req, out)
}
