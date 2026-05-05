package clob

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/bububa/polymarket-client/relayer"
)

// BuildSplitPositionTx writes the destination and calldata for splitPosition into out.
func (c *Client) BuildSplitPositionTx(req *SplitPositionRequest, out *CTFTransaction) error {
	if req == nil {
		return errors.New("polymarket: nil split position request")
	}
	data, err := ctfABI.Pack("splitPosition", req.CollateralToken, req.ParentCollectionID, req.ConditionID, req.Partition, req.Amount)
	if err != nil {
		return fmt.Errorf("ctf: pack splitPosition: %w", err)
	}
	to, err := c.contractAddress(func(cc ContractConfig) common.Address { return cc.ConditionalTokens })
	if err != nil {
		return err
	}
	*out = CTFTransaction{To: to, Data: data}
	return nil
}

// BuildMergePositionsTx writes the destination and calldata for mergePositions into out.
func (c *Client) BuildMergePositionsTx(req *MergePositionsRequest, out *CTFTransaction) error {
	if req == nil {
		return errors.New("polymarket: nil merge position request")
	}
	data, err := ctfABI.Pack("mergePositions", req.CollateralToken, req.ParentCollectionID, req.ConditionID, req.Partition, req.Amount)
	if err != nil {
		return fmt.Errorf("ctf: pack mergePositions: %w", err)
	}
	to, err := c.contractAddress(func(cc ContractConfig) common.Address { return cc.ConditionalTokens })
	if err != nil {
		return err
	}
	*out = CTFTransaction{To: to, Data: data}
	return nil
}

// BuildRedeemPositionsTx writes the destination and calldata for redeemPositions into out.
func (c *Client) BuildRedeemPositionsTx(req *RedeemPositionsRequest, out *CTFTransaction) error {
	if req == nil {
		return errors.New("polymarket: nil redeem position request")
	}
	data, err := ctfABI.Pack("redeemPositions", req.CollateralToken, req.ParentCollectionID, req.ConditionID, req.IndexSets)
	if err != nil {
		return fmt.Errorf("ctf: pack redeemPositions: %w", err)
	}
	to, err := c.contractAddress(func(cc ContractConfig) common.Address { return cc.ConditionalTokens })
	if err != nil {
		return err
	}
	*out = CTFTransaction{To: to, Data: data}
	return nil
}

// BuildRedeemNegRiskTx writes the destination and calldata for neg-risk adapter redemption into out.
func (c *Client) BuildRedeemNegRiskTx(req *RedeemNegRiskRequest, out *CTFTransaction) error {
	if req == nil {
		return errors.New("polymarket: nil neg-risk redeem request")
	}
	data, err := negRiskABI.Pack("redeemPositions", req.ConditionID, req.Amounts)
	if err != nil {
		return fmt.Errorf("ctf: pack neg-risk redeemPositions: %w", err)
	}
	to, err := c.contractAddress(func(cc ContractConfig) common.Address { return cc.NegRiskAdapter })
	if err != nil {
		return err
	}
	*out = CTFTransaction{To: to, Data: data}
	return nil
}

// SplitPosition submits a splitPosition transaction and writes its receipt into out.
func (c *Client) SplitPosition(ctx context.Context, req *SplitPositionRequest, out *TxReceipt) error {
	var tx CTFTransaction
	if err := c.BuildSplitPositionTx(req, &tx); err != nil {
		return err
	}
	return c.sendCTFTxAndWait(ctx, &tx, out)
}

// MergePositions submits a mergePositions transaction and writes its receipt into out.
func (c *Client) MergePositions(ctx context.Context, req *MergePositionsRequest, out *TxReceipt) error {
	var tx CTFTransaction
	if err := c.BuildMergePositionsTx(req, &tx); err != nil {
		return err
	}
	return c.sendCTFTxAndWait(ctx, &tx, out)
}

// RedeemPositions submits a redeemPositions transaction and writes its receipt into out.
func (c *Client) RedeemPositions(ctx context.Context, req *RedeemPositionsRequest, out *TxReceipt) error {
	var tx CTFTransaction
	if err := c.BuildRedeemPositionsTx(req, &tx); err != nil {
		return err
	}
	return c.sendCTFTxAndWait(ctx, &tx, out)
}

// RedeemNegRisk submits a neg-risk adapter redemption transaction and writes its receipt into out.
func (c *Client) RedeemNegRisk(ctx context.Context, req *RedeemNegRiskRequest, out *TxReceipt) error {
	var tx CTFTransaction
	if err := c.BuildRedeemNegRiskTx(req, &tx); err != nil {
		return err
	}
	return c.sendCTFTxAndWait(ctx, &tx, out)
}

// SubmitCTFRelayerTransaction submits built CTF calldata through the configured relayer.
func (c *Client) SubmitCTFRelayerTransaction(ctx context.Context, tx *CTFTransaction, req *RelayerCTFRequest, out *relayer.SubmitTransactionResponse) error {
	if tx == nil {
		return errors.New("polymarket: nil CTF transaction")
	}
	if req == nil {
		return errors.New("polymarket: nil relayer CTF request")
	}
	if strings.TrimSpace(req.ProxyWallet) == "" {
		return errors.New("polymarket: proxy wallet is required")
	}
	if !common.IsHexAddress(req.ProxyWallet) {
		return fmt.Errorf("polymarket: proxy wallet must be a valid hex address")
	}
	to := strings.TrimSpace(req.To)
	if to == "" {
		to = tx.To.Hex()
	}
	data := strings.TrimSpace(req.Data)
	if data == "" {
		data = hexutil.Encode(tx.Data)
	}
	return c.SubmitRelayerTransaction(ctx, &relayer.SubmitTransactionRequest{
		From:            req.From,
		To:              to,
		ProxyWallet:     req.ProxyWallet,
		Data:            data,
		Nonce:           req.Nonce,
		Signature:       req.Signature,
		SignatureParams: req.SignatureParams,
		Type:            req.Type,
		Metadata:        req.Metadata,
		Value:           req.Value,
	}, out)
}

// ConditionID computes the CTF condition ID for oracle, question ID, and slot count.
func ConditionID(oracle common.Address, questionID common.Hash, outcomeSlotCount uint) common.Hash {
	buf := make([]byte, 0, 84)
	buf = append(buf, oracle.Bytes()...)
	buf = append(buf, questionID.Bytes()...)
	n := new(big.Int).SetUint64(uint64(outcomeSlotCount))
	slot := make([]byte, 32)
	n.FillBytes(slot)
	buf = append(buf, slot...)
	return crypto.Keccak256Hash(buf)
}

// CollectionID computes the CTF collection ID for a condition and index set.
func CollectionID(parentCollectionID common.Hash, conditionID common.Hash, indexSet *big.Int) common.Hash {
	inner := make([]byte, 64)
	copy(inner[:32], conditionID.Bytes())
	indexSet.FillBytes(inner[32:64])
	h := crypto.Keccak256Hash(inner)

	var result [32]byte
	parentBytes := parentCollectionID.Bytes()
	hashBytes := h.Bytes()
	for i := range result {
		result[i] = parentBytes[i] ^ hashBytes[i]
	}
	return common.BytesToHash(result[:])
}

// PositionID computes the CTF ERC1155 position ID for a collateral token and collection.
func PositionID(collateralToken common.Address, collectionID common.Hash) *big.Int {
	buf := make([]byte, 0, 52)
	buf = append(buf, collateralToken.Bytes()...)
	buf = append(buf, collectionID.Bytes()...)
	h := crypto.Keccak256Hash(buf)
	return new(big.Int).SetBytes(h.Bytes())
}

func (c *Client) contractAddress(field func(ContractConfig) common.Address) (common.Address, error) {
	config, err := Contracts(c.auth.ChainID)
	if err != nil {
		return common.Address{}, err
	}
	return field(config), nil
}

func (c *Client) sendCTFTxAndWait(ctx context.Context, txRequest *CTFTransaction, out *TxReceipt) error {
	if c.auth.Signer == nil {
		return errors.New("ctf: signer is required")
	}
	if c.rpcURL == "" {
		return errors.New("ctf: rpc url is required")
	}
	ec, err := ethclient.DialContext(ctx, c.rpcURL)
	if err != nil {
		return fmt.Errorf("ctf: dial rpc: %w", err)
	}
	defer ec.Close()

	key := c.auth.Signer.PrivateKey()
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := ec.PendingNonceAt(ctx, from)
	if err != nil {
		return fmt.Errorf("ctf: get nonce: %w", err)
	}

	chainID := big.NewInt(c.auth.ChainID)
	gasTipCap, err := ec.SuggestGasTipCap(ctx)
	if err != nil {
		return fmt.Errorf("ctf: suggest gas tip cap: %w", err)
	}
	head, err := ec.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("ctf: get latest header: %w", err)
	}
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(head.BaseFee, big.NewInt(2)))
	msg := ethereum.CallMsg{From: from, To: &txRequest.To, Data: txRequest.Data, GasFeeCap: gasFeeCap, GasTipCap: gasTipCap}
	gasLimit, err := ec.EstimateGas(ctx, msg)
	if err != nil {
		return fmt.Errorf("ctf: estimate gas: %w", err)
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Gas:       gasLimit,
		To:        &txRequest.To,
		Value:     big.NewInt(0),
		Data:      txRequest.Data,
	})
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		return fmt.Errorf("ctf: sign tx: %w", err)
	}
	if err := ec.SendTransaction(ctx, signed); err != nil {
		return fmt.Errorf("ctf: send tx: %w", err)
	}
	receipt, err := waitForReceipt(ctx, ec, signed.Hash())
	if err != nil {
		return err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		return fmt.Errorf("ctf: transaction %s reverted", signed.Hash().Hex())
	}
	*out = TxReceipt{Hash: receipt.TxHash, BlockNumber: receipt.BlockNumber.Uint64()}
	return nil
}

func waitForReceipt(ctx context.Context, ec *ethclient.Client, txHash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ctf: waiting for receipt %s: %w", txHash.Hex(), ctx.Err())
		case <-ticker.C:
			receipt, err := ec.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
			if !errors.Is(err, ethereum.NotFound) {
				return nil, fmt.Errorf("ctf: get receipt %s: %w", txHash.Hex(), err)
			}
		}
	}
}

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
