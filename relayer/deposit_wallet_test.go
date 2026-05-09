package relayer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bububa/polymarket-client/internal/polyauth"
)

const walletCreateTestPrivateKey = "0x59c6995e998f97a5a0044966f094538092e1db9e7b9c0e5a4e9e4e9e4e9e4e9e"

const walletCreateTestFactory = "0x00000000000Fb5C9ADea0298D729A0CB3823Cc07"

func TestWalletCreateSubmitTransactionRequestRequiresFactory(t *testing.T) {
	signer := walletCreateTestSigner(t)

	var req SubmitTransactionRequest
	err := WalletCreateSubmitTransactionRequest(signer, nil, &req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "deposit wallet factory is required") {
		t.Fatalf("error = %q, want factory required error", err.Error())
	}
}

func TestWalletCreateSubmitTransactionRequestBuildsWalletCreate(t *testing.T) {
	signer := walletCreateTestSigner(t)

	var req SubmitTransactionRequest
	if err := WalletCreateSubmitTransactionRequest(signer, &WalletCreateSubmitTransactionArgs{
		Factory: walletCreateTestFactory,
	}, &req); err != nil {
		t.Fatalf("WalletCreateSubmitTransactionRequest: %v", err)
	}

	if req.Type != NonceTypeWalletCreate {
		t.Fatalf("type = %s, want %s", req.Type, NonceTypeWalletCreate)
	}
	if req.From != signer.Address().Hex() {
		t.Fatalf("from = %s, want %s", req.From, signer.Address().Hex())
	}
	if req.To != walletCreateTestFactory {
		t.Fatalf("to = %s, want %s", req.To, walletCreateTestFactory)
	}
	if req.ProxyWallet != "" || req.Data != "" || req.Nonce != "" || req.Signature != "" || req.SignatureParams != nil || req.DepositWalletParams != nil {
		t.Fatalf("unexpected non-empty WALLET-CREATE request: %+v", req)
	}
}

func TestWalletCreateSubmitTransactionRequestOverridesFrom(t *testing.T) {
	signer := walletCreateTestSigner(t)

	var req SubmitTransactionRequest
	if err := WalletCreateSubmitTransactionRequest(signer, &WalletCreateSubmitTransactionArgs{
		From:    "0x0000000000000000000000000000000000000001",
		Factory: walletCreateTestFactory,
	}, &req); err != nil {
		t.Fatalf("WalletCreateSubmitTransactionRequest: %v", err)
	}

	if req.From != "0x0000000000000000000000000000000000000001" {
		t.Fatalf("from = %s", req.From)
	}
	if req.To != walletCreateTestFactory {
		t.Fatalf("to = %s, want %s", req.To, walletCreateTestFactory)
	}
}

func TestDeployDepositWalletSubmitsWalletCreate(t *testing.T) {
	signer := walletCreateTestSigner(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/submit" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}

		var req SubmitTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if req.Type != NonceTypeWalletCreate {
			t.Fatalf("type = %s, want %s", req.Type, NonceTypeWalletCreate)
		}
		if req.From != signer.Address().Hex() {
			t.Fatalf("from = %s, want %s", req.From, signer.Address().Hex())
		}
		if req.To != walletCreateTestFactory {
			t.Fatalf("to = %s, want %s", req.To, walletCreateTestFactory)
		}
		if req.ProxyWallet != "" || req.Data != "" || req.Nonce != "" || req.Signature != "" || req.SignatureParams != nil || req.DepositWalletParams != nil {
			t.Fatalf("unexpected non-empty WALLET-CREATE request: %+v", req)
		}

		_, _ = w.Write([]byte(`{"transactionID":"tx-1","state":"submitted"}`))
	}))
	defer srv.Close()

	client := New(Config{Host: srv.URL})

	var out SubmitTransactionResponse
	if err := client.DeployDepositWallet(context.Background(), signer, &WalletCreateSubmitTransactionArgs{
		Factory: walletCreateTestFactory,
	}, &out); err != nil {
		t.Fatalf("DeployDepositWallet: %v", err)
	}

	if out.TransactionID != "tx-1" || out.State != "submitted" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestWalletCreateSubmitTransactionRequestRejectsInvalidInputs(t *testing.T) {
	signer := walletCreateTestSigner(t)

	tests := []struct {
		name   string
		signer *polyauth.Signer
		args   *WalletCreateSubmitTransactionArgs
		out    *SubmitTransactionRequest
		want   string
	}{
		{
			name:   "nil signer",
			signer: nil,
			args:   &WalletCreateSubmitTransactionArgs{Factory: walletCreateTestFactory},
			out:    &SubmitTransactionRequest{},
			want:   "signer is required",
		},
		{
			name:   "nil out",
			signer: signer,
			args:   &WalletCreateSubmitTransactionArgs{Factory: walletCreateTestFactory},
			out:    nil,
			want:   "output is nil",
		},
		{
			name:   "missing factory",
			signer: signer,
			args:   nil,
			out:    &SubmitTransactionRequest{},
			want:   "deposit wallet factory is required",
		},
		{
			name:   "invalid from",
			signer: signer,
			args: &WalletCreateSubmitTransactionArgs{
				From:    "bad",
				Factory: walletCreateTestFactory,
			},
			out:  &SubmitTransactionRequest{},
			want: "from must be a valid hex address",
		},
		{
			name:   "invalid factory",
			signer: signer,
			args:   &WalletCreateSubmitTransactionArgs{Factory: "bad"},
			out:    &SubmitTransactionRequest{},
			want:   "factory must be a valid hex address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WalletCreateSubmitTransactionRequest(tt.signer, tt.args, tt.out)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func walletCreateTestSigner(t *testing.T) *polyauth.Signer {
	t.Helper()

	signer, err := polyauth.ParsePrivateKey(walletCreateTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}
	return signer
}
