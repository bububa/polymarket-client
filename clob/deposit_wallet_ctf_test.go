package clob

import (
	"context"
	"strings"
	"testing"

	"github.com/bububa/polymarket-client/relayer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestCTFDepositWalletTransactionRequestWrapsCTFTx(t *testing.T) {
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

	tx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	var req relayer.SubmitTransactionRequest
	err = client.CTFDepositWalletTransactionRequest(context.Background(), tx, &DepositWalletCTFArgs{
		DepositWallet: "0x0000000000000000000000000000000000000002",
		Deadline:      "1760000000",
		Metadata:      "split-position",
	}, &req)
	if err != nil {
		t.Fatalf("CTFDepositWalletTransactionRequest: %v", err)
	}

	assertDepositWalletBatchSubmitRequest(t, req, signer.Address().Hex(), "7", tx.To.Hex())

	if req.Metadata != "split-position" {
		t.Fatalf("metadata = %q, want split-position", req.Metadata)
	}
	if req.DepositWalletParams == nil {
		t.Fatal("depositWalletParams is nil")
	}
	if len(req.DepositWalletParams.Calls) != 1 {
		t.Fatalf("calls len = %d, want 1", len(req.DepositWalletParams.Calls))
	}

	call := req.DepositWalletParams.Calls[0]
	if !strings.EqualFold(call.Target, tx.To.Hex()) {
		t.Fatalf("call target = %s, want %s", call.Target, tx.To.Hex())
	}
	if call.Value != "0" {
		t.Fatalf("call value = %s, want 0", call.Value)
	}
	if call.Data != hexutil.Encode(tx.Data) {
		t.Fatalf("call data = %s, want %s", call.Data, hexutil.Encode(tx.Data))
	}
}

func TestSubmitCTFDepositWalletTransactionSubmitsWrappedCTFTx(t *testing.T) {
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

	tx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	var out relayer.SubmitTransactionResponse
	err = client.SubmitCTFDepositWalletTransaction(context.Background(), tx, &DepositWalletCTFArgs{
		DepositWallet: "0x0000000000000000000000000000000000000002",
		Deadline:      "1760000000",
	}, &out)
	if err != nil {
		t.Fatalf("SubmitCTFDepositWalletTransaction: %v", err)
	}

	if out.TransactionID != "wallet-batch-1" || out.State != "submitted" {
		t.Fatalf("unexpected response: %+v", out)
	}
	if mock.submitted == nil {
		t.Fatal("relayer submit was not called")
	}
	if mock.submitted.DepositWalletParams == nil {
		t.Fatal("submitted depositWalletParams is nil")
	}
	if len(mock.submitted.DepositWalletParams.Calls) != 1 {
		t.Fatalf("submitted calls len = %d, want 1", len(mock.submitted.DepositWalletParams.Calls))
	}

	call := mock.submitted.DepositWalletParams.Calls[0]
	if !strings.EqualFold(call.Target, tx.To.Hex()) {
		t.Fatalf("call target = %s, want %s", call.Target, tx.To.Hex())
	}
	if call.Value != "0" {
		t.Fatalf("call value = %s, want 0", call.Value)
	}
	if call.Data != hexutil.Encode(tx.Data) {
		t.Fatalf("call data = %s, want %s", call.Data, hexutil.Encode(tx.Data))
	}
}

func TestCTFDepositWalletTransactionRequestRejectsInvalidInputs(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
		WithRelayerSubmitter(&captureDepositWalletBatchRelayer{nonce: "7"}),
	)

	validArgs := &DepositWalletCTFArgs{
		DepositWallet: "0x0000000000000000000000000000000000000002",
		Deadline:      "1760000000",
	}

	tests := []struct {
		name       string
		tx         *CTFTransaction
		args       *DepositWalletCTFArgs
		out        *relayer.SubmitTransactionRequest
		wantSubstr string
	}{
		{
			name:       "nil tx",
			tx:         nil,
			args:       validArgs,
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "nil CTF transaction",
		},
		{
			name: "zero target",
			tx: &CTFTransaction{
				Data: []byte{0x01},
			},
			args:       validArgs,
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "CTF transaction target is required",
		},
		{
			name: "empty data",
			tx: &CTFTransaction{
				To: common.HexToAddress("0x00000000000000000000000000000000000000aa"),
			},
			args:       validArgs,
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "CTF transaction data is required",
		},
		{
			name: "nil args",
			tx: &CTFTransaction{
				To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
				Data: []byte{0x01},
			},
			args:       nil,
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "submit transaction request args is nil",
		},
		{
			name: "nil output",
			tx: &CTFTransaction{
				To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
				Data: []byte{0x01},
			},
			args:       validArgs,
			out:        nil,
			wantSubstr: "output is nil",
		},
		{
			name: "missing deposit wallet",
			tx: &CTFTransaction{
				To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
				Data: []byte{0x01},
			},
			args:       &DepositWalletCTFArgs{Deadline: "1760000000"},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "deposit wallet is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.CTFDepositWalletTransactionRequest(context.Background(), tt.tx, tt.args, tt.out)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}
