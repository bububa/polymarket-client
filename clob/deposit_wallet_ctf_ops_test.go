package clob

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/bububa/polymarket-client/internal/polyauth"
	"github.com/bububa/polymarket-client/relayer"
	"github.com/ethereum/go-ethereum/common"
)

func TestSplitPositionWithDepositWalletSubmitsSplitTx(t *testing.T) {
	client, signer, mock := newDepositWalletCTFOpsTestClient(t)

	var out relayer.SubmitTransactionResponse
	err := client.SplitPositionWithDepositWallet(
		context.Background(),
		&SplitPositionRequest{
			CollateralToken:    ctfSafetyCollateral,
			ParentCollectionID: common.Hash{},
			ConditionID:        ctfSafetyConditionID,
			Partition:          BinaryPartition(),
			Amount:             big.NewInt(1_000_000),
		},
		validDepositWalletCTFArgs(),
		&out,
	)
	if err != nil {
		t.Fatalf("SplitPositionWithDepositWallet: %v", err)
	}

	assertDepositWalletCTFOpSubmitted(t, out, mock)
	assertDepositWalletBatchSubmitRequest(t, *mock.submitted, signer.Address().Hex(), "7", ctfSafetyCTFCollateralAdapter.Hex())
	assertDepositWalletCallSelector(t, *mock.submitted, ctfABI.Methods["splitPosition"].ID)
}

func TestMergePositionsWithDepositWalletSubmitsMergeTx(t *testing.T) {
	client, signer, mock := newDepositWalletCTFOpsTestClient(t)

	var out relayer.SubmitTransactionResponse
	err := client.MergePositionsWithDepositWallet(
		context.Background(),
		&MergePositionsRequest{
			CollateralToken:    ctfSafetyCollateral,
			ParentCollectionID: common.Hash{},
			ConditionID:        ctfSafetyConditionID,
			Partition:          BinaryPartition(),
			Amount:             big.NewInt(1_000_000),
		},
		validDepositWalletCTFArgs(),
		&out,
	)
	if err != nil {
		t.Fatalf("MergePositionsWithDepositWallet: %v", err)
	}

	assertDepositWalletCTFOpSubmitted(t, out, mock)
	assertDepositWalletBatchSubmitRequest(t, *mock.submitted, signer.Address().Hex(), "7", ctfSafetyCTFCollateralAdapter.Hex())
	assertDepositWalletCallSelector(t, *mock.submitted, ctfABI.Methods["mergePositions"].ID)
}

func TestRedeemPositionsWithDepositWalletSubmitsRedeemTx(t *testing.T) {
	client, signer, mock := newDepositWalletCTFOpsTestClient(t)

	var out relayer.SubmitTransactionResponse
	err := client.RedeemPositionsWithDepositWallet(
		context.Background(),
		&RedeemPositionsRequest{
			CollateralToken:    ctfSafetyCollateral,
			ParentCollectionID: common.Hash{},
			ConditionID:        ctfSafetyConditionID,
			IndexSets:          BinaryPartition(),
		},
		validDepositWalletCTFArgs(),
		&out,
	)
	if err != nil {
		t.Fatalf("RedeemPositionsWithDepositWallet: %v", err)
	}

	assertDepositWalletCTFOpSubmitted(t, out, mock)
	assertDepositWalletBatchSubmitRequest(t, *mock.submitted, signer.Address().Hex(), "7", ctfSafetyCTFCollateralAdapter.Hex())
	assertDepositWalletCallSelector(t, *mock.submitted, ctfABI.Methods["redeemPositions"].ID)
}

func TestRedeemNegRiskWithDepositWalletSubmitsNegRiskRedeemTx(t *testing.T) {
	client, signer, mock := newDepositWalletCTFOpsTestClient(t)

	var out relayer.SubmitTransactionResponse
	err := client.RedeemNegRiskWithDepositWallet(
		context.Background(),
		&RedeemNegRiskRequest{
			ConditionID: ctfSafetyConditionID,
		},
		validDepositWalletCTFArgs(),
		&out,
	)
	if err != nil {
		t.Fatalf("RedeemNegRiskWithDepositWallet: %v", err)
	}

	assertDepositWalletCTFOpSubmitted(t, out, mock)
	assertDepositWalletBatchSubmitRequest(t, *mock.submitted, signer.Address().Hex(), "7", ctfSafetyNegRiskCTFCollateralAdapter.Hex())
	assertDepositWalletCallSelector(t, *mock.submitted, ctfABI.Methods["redeemPositions"].ID)
}

func TestDepositWalletCTFOpsPropagateBuildErrors(t *testing.T) {
	client, _, _ := newDepositWalletCTFOpsTestClient(t)

	var out relayer.SubmitTransactionResponse
	err := client.SplitPositionWithDepositWallet(
		context.Background(),
		&SplitPositionRequest{},
		validDepositWalletCTFArgs(),
		&out,
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func newDepositWalletCTFOpsTestClient(t *testing.T) (*Client, *polyauth.Signer, *captureDepositWalletBatchRelayer) {
	t.Helper()

	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	mock := &captureDepositWalletBatchRelayer{nonce: "7"}
	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
		WithRelayerSubmitter(mock),
	)

	return client, signer, mock
}

func validDepositWalletCTFArgs() *DepositWalletCTFArgs {
	return &DepositWalletCTFArgs{
		DepositWallet: "0x0000000000000000000000000000000000000002",
		Deadline:      "1760000000",
	}
}

func assertDepositWalletCTFOpSubmitted(
	t *testing.T,
	out relayer.SubmitTransactionResponse,
	mock *captureDepositWalletBatchRelayer,
) {
	t.Helper()

	if out.TransactionID != "wallet-batch-1" || out.State != "submitted" {
		t.Fatalf("unexpected response: %+v", out)
	}
	if mock.submitted == nil {
		t.Fatal("relayer submit was not called")
	}
}

func assertDepositWalletCallSelector(t *testing.T, req relayer.SubmitTransactionRequest, selector []byte) {
	t.Helper()

	if req.DepositWalletParams == nil {
		t.Fatal("depositWalletParams is nil")
	}
	if len(req.DepositWalletParams.Calls) != 1 {
		t.Fatalf("calls len = %d, want 1", len(req.DepositWalletParams.Calls))
	}

	data := req.DepositWalletParams.Calls[0].Data
	if !strings.HasPrefix(data, "0x") {
		t.Fatalf("call data = %q, want 0x-prefixed hex", data)
	}
	if len(data) < 10 {
		t.Fatalf("call data too short: %s", data)
	}

	gotSelector := data[:10]
	wantSelector := "0x" + common.Bytes2Hex(selector)
	if !strings.EqualFold(gotSelector, wantSelector) {
		t.Fatalf("selector = %s, want %s", gotSelector, wantSelector)
	}
}
