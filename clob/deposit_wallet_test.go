package clob

import (
	"context"
	"testing"

	"github.com/bububa/polymarket-client/relayer"
)

type captureDepositWalletRelayer struct {
	submitted *relayer.SubmitTransactionRequest
}

func (m *captureDepositWalletRelayer) SubmitTransaction(_ context.Context, req *relayer.SubmitTransactionRequest, out *relayer.SubmitTransactionResponse) error {
	if req != nil {
		copied := *req
		m.submitted = &copied
	}
	if out != nil {
		*out = relayer.SubmitTransactionResponse{
			TransactionID: "wallet-create-1",
			State:         "submitted",
		}
	}
	return nil
}

func TestBuildDepositWalletCreateRelayerRequestUsesContractFactory(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
	)

	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	var req relayer.SubmitTransactionRequest
	if err := client.DepositWalletCreateRelayerRequest(&req); err != nil {
		t.Fatalf("BuildDepositWalletCreateRelayerRequest: %v", err)
	}

	if req.Type != relayer.NonceTypeWalletCreate {
		t.Fatalf("type = %s, want %s", req.Type, relayer.NonceTypeWalletCreate)
	}
	if req.From != signer.Address().Hex() {
		t.Fatalf("from = %s, want %s", req.From, signer.Address().Hex())
	}
	if req.To != contracts.DepositWalletFactory.Hex() {
		t.Fatalf("to = %s, want %s", req.To, contracts.DepositWalletFactory.Hex())
	}
	if req.ProxyWallet != "" || req.Data != "" || req.Nonce != "" || req.Signature != "" || req.SignatureParams != nil || req.DepositWalletParams != nil {
		t.Fatalf("unexpected non-empty WALLET-CREATE request: %+v", req)
	}
}

func TestDeployDepositWalletSubmitsContractFactoryRequest(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	mock := &captureDepositWalletRelayer{}
	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
		WithRelayerSubmitter(mock),
	)

	var out relayer.SubmitTransactionResponse
	if err := client.DeployDepositWallet(context.Background(), &out); err != nil {
		t.Fatalf("DeployDepositWallet: %v", err)
	}

	if out.TransactionID != "wallet-create-1" || out.State != "submitted" {
		t.Fatalf("unexpected response: %+v", out)
	}

	if mock.submitted == nil {
		t.Fatal("relayer submit was not called")
	}

	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	if mock.submitted.Type != relayer.NonceTypeWalletCreate {
		t.Fatalf("type = %s, want %s", mock.submitted.Type, relayer.NonceTypeWalletCreate)
	}
	if mock.submitted.From != signer.Address().Hex() {
		t.Fatalf("from = %s, want %s", mock.submitted.From, signer.Address().Hex())
	}
	if mock.submitted.To != contracts.DepositWalletFactory.Hex() {
		t.Fatalf("to = %s, want %s", mock.submitted.To, contracts.DepositWalletFactory.Hex())
	}
}

func TestBuildDepositWalletCreateRelayerRequestRejectsMissingSigner(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))

	var req relayer.SubmitTransactionRequest
	err := client.DepositWalletCreateRelayerRequest(&req)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "polymarket: signer is required" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestBuildDepositWalletCreateRelayerRequestRejectsNilOutput(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
	)

	err = client.DepositWalletCreateRelayerRequest(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "polymarket: submit transaction request output is nil" {
		t.Fatalf("error = %q", err.Error())
	}
}
