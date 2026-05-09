package clob

import (
	"context"
	"strings"
	"testing"

	"github.com/bububa/polymarket-client/relayer"
)

type captureDepositWalletBatchRelayer struct {
	submitted *relayer.SubmitTransactionRequest

	nonceAddress string
	nonceType    relayer.NonceType
	nonce        string
}

func (m *captureDepositWalletBatchRelayer) SubmitTransaction(_ context.Context, req *relayer.SubmitTransactionRequest, out *relayer.SubmitTransactionResponse) error {
	if req != nil {
		copied := *req
		m.submitted = &copied
	}
	if out != nil {
		*out = relayer.SubmitTransactionResponse{
			TransactionID: "wallet-batch-1",
			State:         "submitted",
		}
	}
	return nil
}

func (m *captureDepositWalletBatchRelayer) GetNonce(_ context.Context, out *relayer.NonceResponse, nonceType ...relayer.NonceType) error {
	if out != nil {
		m.nonceAddress = out.Address
	}

	if len(nonceType) > 0 {
		m.nonceType = nonceType[0]
	}

	nonce := m.nonce
	if nonce == "" {
		nonce = "7"
	}

	if out != nil {
		out.Nonce = String(nonce)
	}
	return nil
}

func TestBuildDepositWalletBatchTypedData(t *testing.T) {
	typedData, err := BuildDepositWalletBatchTypedData(
		PolygonChainID,
		"0x0000000000000000000000000000000000000002",
		"7",
		"1760000000",
		[]relayer.DepositWalletCall{
			{
				Target: "0x0000000000000000000000000000000000000003",
				Value:  "0",
				Data:   "0x01020304",
			},
		},
	)
	if err != nil {
		t.Fatalf("BuildDepositWalletBatchTypedData: %v", err)
	}

	if typedData.PrimaryType != "Batch" {
		t.Fatalf("primaryType = %s, want Batch", typedData.PrimaryType)
	}
	if typedData.Domain.Name != "DepositWallet" {
		t.Fatalf("domain name = %s, want DepositWallet", typedData.Domain.Name)
	}
	if typedData.Domain.Version != "1" {
		t.Fatalf("domain version = %s, want 1", typedData.Domain.Version)
	}
	if typedData.Domain.VerifyingContract != "0x0000000000000000000000000000000000000002" {
		t.Fatalf("verifyingContract = %s", typedData.Domain.VerifyingContract)
	}
	if typedData.Message["wallet"] != "0x0000000000000000000000000000000000000002" {
		t.Fatalf("wallet = %v", typedData.Message["wallet"])
	}
	if typedData.Message["nonce"] != "7" {
		t.Fatalf("nonce = %v", typedData.Message["nonce"])
	}
	if typedData.Message["deadline"] != "1760000000" {
		t.Fatalf("deadline = %v", typedData.Message["deadline"])
	}

	calls, ok := typedData.Message["calls"].([]any)
	if !ok {
		t.Fatalf("calls = %T(%v), want []any", typedData.Message["calls"], typedData.Message["calls"])
	}
	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}

	call, ok := calls[0].(map[string]any)
	if !ok {
		t.Fatalf("call = %T(%v), want map", calls[0], calls[0])
	}
	if !strings.EqualFold(call["target"].(string), "0x0000000000000000000000000000000000000003") {
		t.Fatalf("call target = %v", call["target"])
	}
	if call["value"] != "0" {
		t.Fatalf("call value = %v", call["value"])
	}
	if call["data"] != "0x01020304" {
		t.Fatalf("call data = %v", call["data"])
	}
}

func TestBuildDepositWalletBatchRelayerRequestFetchesNonceAndSigns(t *testing.T) {
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

	var req relayer.SubmitTransactionRequest
	err = client.DepositWalletBatchRelayerRequest(context.Background(), &DepositWalletBatchArgs{
		DepositWallet: "0x0000000000000000000000000000000000000002",
		Deadline:      "1760000000",
		Calls: []relayer.DepositWalletCall{
			{
				Target: "0x0000000000000000000000000000000000000003",
				Value:  "0",
				Data:   "0x01020304",
			},
		},
		Metadata: "approve-pusd",
	}, &req)
	if err != nil {
		t.Fatalf("DepositWalletBatchRelayerRequest: %v", err)
	}

	if !strings.EqualFold(mock.nonceAddress, signer.Address().Hex()) {
		t.Fatalf("nonce address = %s, want %s", mock.nonceAddress, signer.Address().Hex())
	}
	if mock.nonceType != relayer.NonceTypeWallet {
		t.Fatalf("nonce type = %s, want %s", mock.nonceType, relayer.NonceTypeWallet)
	}

	assertDepositWalletBatchSubmitRequest(
		t,
		req,
		signer.Address().Hex(),
		"7",
		"0x0000000000000000000000000000000000000003",
	)

	if req.Metadata != "approve-pusd" {
		t.Fatalf("metadata = %q, want approve-pusd", req.Metadata)
	}
}

func TestBuildDepositWalletBatchRelayerRequestUsesProvidedNonce(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	mock := &captureDepositWalletBatchRelayer{}
	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
		WithRelayerSubmitter(mock),
	)

	var req relayer.SubmitTransactionRequest
	err = client.DepositWalletBatchRelayerRequest(context.Background(), &DepositWalletBatchArgs{
		DepositWallet: "0x0000000000000000000000000000000000000002",
		Nonce:         "11",
		Deadline:      "1760000000",
		Calls: []relayer.DepositWalletCall{
			{
				Target: "0x0000000000000000000000000000000000000003",
				Value:  "",
				Data:   "",
			},
		},
	}, &req)
	if err != nil {
		t.Fatalf("DepositWalletBatchRelayerRequest: %v", err)
	}

	if mock.nonceAddress != "" {
		t.Fatalf("nonce should not have been fetched, got address %s", mock.nonceAddress)
	}

	assertDepositWalletBatchSubmitRequest(
		t,
		req,
		signer.Address().Hex(),
		"11",
		"0x0000000000000000000000000000000000000003",
	)

	if req.DepositWalletParams.Calls[0].Value != "0" {
		t.Fatalf("default call value = %q, want 0", req.DepositWalletParams.Calls[0].Value)
	}
	if req.DepositWalletParams.Calls[0].Data != "0x" {
		t.Fatalf("default call data = %q, want 0x", req.DepositWalletParams.Calls[0].Data)
	}
}

func TestBuildDepositWalletBatchRelayerRequestAllowsFactoryOverride(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
	)

	var req relayer.SubmitTransactionRequest
	err = client.DepositWalletBatchRelayerRequest(context.Background(), &DepositWalletBatchArgs{
		Factory:       "0x00000000000000000000000000000000000000aa",
		DepositWallet: "0x0000000000000000000000000000000000000002",
		Nonce:         "1",
		Deadline:      "1760000000",
		Calls: []relayer.DepositWalletCall{
			{
				Target: "0x0000000000000000000000000000000000000003",
				Value:  "0",
				Data:   "0x",
			},
		},
	}, &req)
	if err != nil {
		t.Fatalf("DepositWalletBatchRelayerRequest: %v", err)
	}

	if !strings.EqualFold(req.To, "0x00000000000000000000000000000000000000aa") {
		t.Fatalf("to = %s, want factory override", req.To)
	}
}

func TestExecuteDepositWalletBatchSubmitsWalletRequest(t *testing.T) {
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

	var out relayer.SubmitTransactionResponse
	err = client.DepositWalletBatch(context.Background(), &DepositWalletBatchArgs{
		DepositWallet: "0x0000000000000000000000000000000000000002",
		Deadline:      "1760000000",
		Calls: []relayer.DepositWalletCall{
			{
				Target: "0x0000000000000000000000000000000000000003",
				Value:  "0",
				Data:   "0x01020304",
			},
		},
	}, &out)
	if err != nil {
		t.Fatalf("DepositWalletBatch: %v", err)
	}

	if out.TransactionID != "wallet-batch-1" || out.State != "submitted" {
		t.Fatalf("unexpected response: %+v", out)
	}

	if mock.submitted == nil {
		t.Fatal("relayer submit was not called")
	}

	assertDepositWalletBatchSubmitRequest(
		t,
		*mock.submitted,
		signer.Address().Hex(),
		"7",
		"0x0000000000000000000000000000000000000003",
	)
}

func TestBuildDepositWalletBatchRelayerRequestRejectsInvalidInputs(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	validArgs := func() *DepositWalletBatchArgs {
		return &DepositWalletBatchArgs{
			DepositWallet: "0x0000000000000000000000000000000000000002",
			Nonce:         "1",
			Deadline:      "1760000000",
			Calls: []relayer.DepositWalletCall{
				{
					Target: "0x0000000000000000000000000000000000000003",
					Value:  "0",
					Data:   "0x",
				},
			},
		}
	}

	tests := []struct {
		name       string
		client     *Client
		args       func() *DepositWalletBatchArgs
		out        *relayer.SubmitTransactionRequest
		wantSubstr string
	}{
		{
			name: "nil output",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args:       validArgs,
			out:        nil,
			wantSubstr: "output is nil",
		},
		{
			name: "missing signer",
			client: NewClient(
				"",
				WithChainID(PolygonChainID),
			),
			args:       validArgs,
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "signer is required",
		},
		{
			name: "missing chain id",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(0),
			),
			args:       validArgs,
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "chain id is required",
		},
		{
			name: "invalid from",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.From = "bad"
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "from must be a valid hex address",
		},
		{
			name: "invalid factory override",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.Factory = "bad"
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "deposit wallet factory must be a valid hex address",
		},
		{
			name: "missing deposit wallet",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.DepositWallet = ""
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "deposit wallet is required",
		},
		{
			name: "invalid deposit wallet",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.DepositWallet = "bad"
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "deposit wallet must be a valid hex address",
		},
		{
			name: "missing deadline",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.Deadline = ""
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "deadline is required",
		},
		{
			name: "missing calls",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.Calls = nil
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "deposit wallet batch calls are required",
		},
		{
			name: "invalid call target",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.Calls[0].Target = "bad"
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "target must be a valid hex address",
		},
		{
			name: "invalid call data",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.Calls[0].Data = "bad"
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "data must be 0x-prefixed hex",
		},
		{
			name: "missing relayer for nonce fetch",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.Nonce = ""
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "relayer client is not configured",
		},
		{
			name: "relayer without nonce lookup",
			client: NewClient(
				"",
				WithSigner(signer),
				WithChainID(PolygonChainID),
				WithRelayerSubmitter(&submitOnlyDepositWalletRelayer{}),
			),
			args: func() *DepositWalletBatchArgs {
				a := validArgs()
				a.Nonce = ""
				return a
			},
			out:        &relayer.SubmitTransactionRequest{},
			wantSubstr: "does not support nonce lookup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.DepositWalletBatchRelayerRequest(context.Background(), tt.args(), tt.out)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}

type submitOnlyDepositWalletRelayer struct{}

func (m *submitOnlyDepositWalletRelayer) SubmitTransaction(_ context.Context, _ *relayer.SubmitTransactionRequest, _ *relayer.SubmitTransactionResponse) error {
	return nil
}

func assertDepositWalletBatchSubmitRequest(
	t *testing.T,
	req relayer.SubmitTransactionRequest,
	from string,
	nonce string,
	expectedCallTarget string,
) {
	t.Helper()

	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	if req.Type != relayer.NonceTypeWallet {
		t.Fatalf("type = %s, want %s", req.Type, relayer.NonceTypeWallet)
	}
	if !strings.EqualFold(req.From, from) {
		t.Fatalf("from = %s, want %s", req.From, from)
	}
	if !strings.EqualFold(req.To, contracts.DepositWalletFactory.Hex()) {
		t.Fatalf("to = %s, want %s", req.To, contracts.DepositWalletFactory.Hex())
	}
	if req.Nonce != nonce {
		t.Fatalf("nonce = %s, want %s", req.Nonce, nonce)
	}
	if req.Signature == "" {
		t.Fatal("signature is empty")
	}
	if req.SignatureParams != nil {
		t.Fatalf("signatureParams should be nil for WALLET batch, got %+v", req.SignatureParams)
	}
	if req.ProxyWallet != "" || req.Data != "" {
		t.Fatalf("legacy proxy fields should be empty: %+v", req)
	}
	if req.DepositWalletParams == nil {
		t.Fatal("depositWalletParams is nil")
	}
	if !strings.EqualFold(req.DepositWalletParams.DepositWallet, "0x0000000000000000000000000000000000000002") {
		t.Fatalf("depositWallet = %s", req.DepositWalletParams.DepositWallet)
	}
	if req.DepositWalletParams.Deadline != "1760000000" {
		t.Fatalf("deadline = %s", req.DepositWalletParams.Deadline)
	}
	if len(req.DepositWalletParams.Calls) != 1 {
		t.Fatalf("calls len = %d, want 1", len(req.DepositWalletParams.Calls))
	}
	if !strings.EqualFold(req.DepositWalletParams.Calls[0].Target, expectedCallTarget) {
		t.Fatalf("call target = %s, want %s", req.DepositWalletParams.Calls[0].Target, expectedCallTarget)
	}
}
