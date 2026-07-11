package clob

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/bububa/polymarket-client/relayer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BuildComboCollateralApprovalTx approves the Combo Router to spend pUSD.
func (c *Client) BuildComboCollateralApprovalTx(amount *big.Int, out *CTFTransaction) error {
	if err := validateCTFTransactionOutput(out); err != nil {
		return err
	}
	if amount == nil {
		return errors.New("polymarket: amount is required")
	}
	if amount.Sign() < 0 || amount.Cmp(uint256Max) > 0 {
		return errors.New("polymarket: amount must be a non-negative uint256")
	}
	contracts, err := c.comboContracts()
	if err != nil {
		return err
	}
	data, err := comboApprovalABI.Pack("approve", contracts.ComboRouter, amount)
	if err != nil {
		return fmt.Errorf("combo: pack approve: %w", err)
	}
	*out = CTFTransaction{To: contracts.Collateral, Data: data}
	return nil
}

// BuildComboPositionApprovalTx grants or revokes Router access to Combo ERC-1155 positions.
func (c *Client) BuildComboPositionApprovalTx(approved bool, out *CTFTransaction) error {
	if err := validateCTFTransactionOutput(out); err != nil {
		return err
	}
	contracts, err := c.comboContracts()
	if err != nil {
		return err
	}
	data, err := comboApprovalABI.Pack("setApprovalForAll", contracts.ComboRouter, approved)
	if err != nil {
		return fmt.Errorf("combo: pack setApprovalForAll: %w", err)
	}
	*out = CTFTransaction{To: contracts.ComboPositionManager, Data: data}
	return nil
}

// BuildComboSplitPositionTxs builds prepareCondition followed by Router.split.
func (c *Client) BuildComboSplitPositionTxs(req *ComboSplitPositionRequest, out *[]CTFTransaction) error {
	if req == nil {
		return errors.New("polymarket: nil combo split position request")
	}
	if out == nil {
		return errors.New("polymarket: combo transaction output is nil")
	}
	if err := validateBigIntRequired("amount", req.Amount); err != nil {
		return err
	}
	canonical, err := CanonicalizeComboLegs(req.LegPositionIDs)
	if err != nil {
		return err
	}
	combo, err := DeriveComboPositionContext(req.LegPositionIDs)
	if err != nil {
		return err
	}
	contracts, err := c.comboContracts()
	if err != nil {
		return err
	}
	prepareData, err := comboABI.Pack("prepareCondition", canonical)
	if err != nil {
		return fmt.Errorf("combo: pack prepareCondition: %w", err)
	}
	splitData, err := comboABI.Pack("split", combo.ConditionID, req.Amount)
	if err != nil {
		return fmt.Errorf("combo: pack split: %w", err)
	}
	*out = []CTFTransaction{
		{To: contracts.CombinatorialModule, Data: prepareData},
		{To: contracts.ComboRouter, Data: splitData},
	}
	return nil
}

// BuildComboMergePositionsTxs builds idempotent prepareCondition followed by Router.merge.
func (c *Client) BuildComboMergePositionsTxs(req *ComboMergePositionsRequest, out *[]CTFTransaction) error {
	if req == nil {
		return errors.New("polymarket: nil combo merge positions request")
	}
	if out == nil {
		return errors.New("polymarket: combo transaction output is nil")
	}
	if err := validateBigIntRequired("amount", req.Amount); err != nil {
		return err
	}
	canonical, err := CanonicalizeComboLegs(req.LegPositionIDs)
	if err != nil {
		return err
	}
	combo, err := DeriveComboPositionContext(req.LegPositionIDs)
	if err != nil {
		return err
	}
	contracts, err := c.comboContracts()
	if err != nil {
		return err
	}
	prepareData, err := comboABI.Pack("prepareCondition", canonical)
	if err != nil {
		return fmt.Errorf("combo: pack prepareCondition: %w", err)
	}
	mergeData, err := comboABI.Pack("merge", combo.ConditionID, req.Amount)
	if err != nil {
		return fmt.Errorf("combo: pack merge: %w", err)
	}
	*out = []CTFTransaction{
		{To: contracts.CombinatorialModule, Data: prepareData},
		{To: contracts.ComboRouter, Data: mergeData},
	}
	return nil
}

// BuildComboRedeemPositionTx builds Router.redeem for a Combo position ID.
func (c *Client) BuildComboRedeemPositionTx(req *ComboRedeemPositionRequest, out *CTFTransaction) error {
	if req == nil {
		return errors.New("polymarket: nil combo redeem position request")
	}
	if err := validateCTFTransactionOutput(out); err != nil {
		return err
	}
	if err := validateBigIntRequired("amount", req.Amount); err != nil {
		return err
	}
	condition, outcome, err := DecodeComboPositionID(req.PositionID)
	if err != nil {
		return err
	}
	contracts, err := c.comboContracts()
	if err != nil {
		return err
	}
	data, err := comboABI.Pack("redeem", condition, new(big.Int).SetUint64(uint64(outcome)), req.Amount)
	if err != nil {
		return fmt.Errorf("combo: pack redeem: %w", err)
	}
	*out = CTFTransaction{To: contracts.ComboRouter, Data: data}
	return nil
}

// ComboSplitPosition executes prepareCondition then split using the configured EOA.
func (c *Client) ComboSplitPosition(ctx context.Context, req *ComboSplitPositionRequest, out *ComboOperationReceipt) error {
	var txs []CTFTransaction
	if err := c.BuildComboSplitPositionTxs(req, &txs); err != nil {
		return err
	}
	return c.sendComboTransactions(ctx, txs, out)
}

// ComboMergePositions executes prepareCondition then merge using the configured EOA.
func (c *Client) ComboMergePositions(ctx context.Context, req *ComboMergePositionsRequest, out *ComboOperationReceipt) error {
	resolved, err := c.resolveComboMergeAmount(ctx, req, c.signerAddress())
	if err != nil {
		return err
	}
	var txs []CTFTransaction
	if err := c.BuildComboMergePositionsTxs(resolved, &txs); err != nil {
		return err
	}
	return c.sendComboTransactions(ctx, txs, out)
}

// ComboRedeemPosition redeems an explicit Combo position amount using the configured EOA.
func (c *Client) ComboRedeemPosition(ctx context.Context, req *ComboRedeemPositionRequest, out *TxReceipt) error {
	resolved, err := c.resolveComboRedeemAmount(ctx, req, c.signerAddress())
	if err != nil {
		return err
	}
	var tx CTFTransaction
	if err := c.BuildComboRedeemPositionTx(resolved, &tx); err != nil {
		return err
	}
	return c.sendCTFTxAndWait(ctx, &tx, out)
}

// ComboSplitPositionWithDepositWallet submits an atomic WALLET batch.
func (c *Client) ComboSplitPositionWithDepositWallet(ctx context.Context, req *ComboSplitPositionRequest, args *DepositWalletCTFArgs, out *relayer.SubmitTransactionResponse) error {
	var txs []CTFTransaction
	if err := c.BuildComboSplitPositionTxs(req, &txs); err != nil {
		return err
	}
	return c.submitComboDepositWallet(ctx, txs, args, out)
}

// ComboMergePositionsWithDepositWallet submits an atomic WALLET batch.
func (c *Client) ComboMergePositionsWithDepositWallet(ctx context.Context, req *ComboMergePositionsRequest, args *DepositWalletCTFArgs, out *relayer.SubmitTransactionResponse) error {
	if args == nil || !common.IsHexAddress(args.DepositWallet) {
		return errors.New("polymarket: valid deposit wallet is required")
	}
	resolved, err := c.resolveComboMergeAmount(ctx, req, common.HexToAddress(args.DepositWallet))
	if err != nil {
		return err
	}
	var txs []CTFTransaction
	if err := c.BuildComboMergePositionsTxs(resolved, &txs); err != nil {
		return err
	}
	return c.submitComboDepositWallet(ctx, txs, args, out)
}

// ComboRedeemPositionWithDepositWallet submits a WALLET redemption call.
func (c *Client) ComboRedeemPositionWithDepositWallet(ctx context.Context, req *ComboRedeemPositionRequest, args *DepositWalletCTFArgs, out *relayer.SubmitTransactionResponse) error {
	if args == nil || !common.IsHexAddress(args.DepositWallet) {
		return errors.New("polymarket: valid deposit wallet is required")
	}
	resolved, err := c.resolveComboRedeemAmount(ctx, req, common.HexToAddress(args.DepositWallet))
	if err != nil {
		return err
	}
	var tx CTFTransaction
	if err := c.BuildComboRedeemPositionTx(resolved, &tx); err != nil {
		return err
	}
	return c.submitComboDepositWallet(ctx, []CTFTransaction{tx}, args, out)
}

func (c *Client) comboContracts() (ContractConfig, error) {
	contracts, err := Contracts(c.auth.ChainID)
	if err != nil {
		return ContractConfig{}, err
	}
	if contracts.CombinatorialModule == (common.Address{}) || contracts.ComboRouter == (common.Address{}) || contracts.ComboPositionManager == (common.Address{}) {
		return ContractConfig{}, fmt.Errorf("polymarket: combo position contracts are not configured for chain %d", c.auth.ChainID)
	}
	return contracts, nil
}

func (c *Client) sendComboTransactions(ctx context.Context, txs []CTFTransaction, out *ComboOperationReceipt) error {
	if out == nil {
		return errors.New("polymarket: combo operation receipt output is nil")
	}
	out.Transactions = make([]TxReceipt, 0, len(txs))
	for i := range txs {
		var receipt TxReceipt
		if err := c.sendCTFTxAndWait(ctx, &txs[i], &receipt); err != nil {
			return fmt.Errorf("combo: transaction %d: %w", i, err)
		}
		out.Transactions = append(out.Transactions, receipt)
	}
	return nil
}

func (c *Client) submitComboDepositWallet(ctx context.Context, txs []CTFTransaction, args *DepositWalletCTFArgs, out *relayer.SubmitTransactionResponse) error {
	if args == nil {
		return errors.New("polymarket: deposit wallet CTF args are required")
	}
	calls := make([]relayer.DepositWalletCall, len(txs))
	for i, tx := range txs {
		calls[i] = relayer.DepositWalletCall{Target: tx.To.Hex(), Value: "0", Data: hexutil.Encode(tx.Data)}
	}
	return c.DepositWalletBatch(ctx, &DepositWalletBatchArgs{
		From: args.From, Factory: args.Factory, DepositWallet: args.DepositWallet,
		Nonce: args.Nonce, Deadline: args.Deadline, Calls: calls, Metadata: args.Metadata,
	}, out)
}

func (c *Client) signerAddress() common.Address {
	if c.auth.Signer == nil {
		return common.Address{}
	}
	return c.auth.Signer.Address()
}

func (c *Client) resolveComboMergeAmount(ctx context.Context, req *ComboMergePositionsRequest, owner common.Address) (*ComboMergePositionsRequest, error) {
	if req == nil {
		return nil, errors.New("polymarket: nil combo merge positions request")
	}
	resolved := *req
	if resolved.Amount != nil {
		if err := validateBigIntRequired("amount", resolved.Amount); err != nil {
			return nil, err
		}
	}
	maxAmount, err := c.ComboMaxMergeAmount(ctx, owner, resolved.LegPositionIDs)
	if err != nil {
		return nil, err
	}
	if resolved.Amount == nil {
		resolved.Amount = maxAmount
	} else if resolved.Amount.Cmp(maxAmount) > 0 {
		return nil, fmt.Errorf("polymarket: combo merge amount %s exceeds available amount %s", resolved.Amount, maxAmount)
	}
	return &resolved, nil
}

func (c *Client) resolveComboRedeemAmount(ctx context.Context, req *ComboRedeemPositionRequest, owner common.Address) (*ComboRedeemPositionRequest, error) {
	if req == nil {
		return nil, errors.New("polymarket: nil combo redeem position request")
	}
	resolved := *req
	if resolved.Amount == nil {
		balance, err := c.ComboPositionBalance(ctx, owner, resolved.PositionID)
		if err != nil {
			return nil, err
		}
		if balance.Sign() == 0 {
			return nil, errors.New("polymarket: combo position has no balance to redeem")
		}
		resolved.Amount = balance
	}
	return &resolved, nil
}
