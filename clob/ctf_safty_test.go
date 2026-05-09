package clob

import (
	"context"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/bububa/polymarket-client/internal/polyauth"
	"github.com/bububa/polymarket-client/relayer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const ctfSafetyTestPrivateKey = "0x59c6995e998f97a5a0044966f094538092e1db9e7b9c0e5a4e9e4e9e4e9e4e9e"

var (
	ctfSafetyConditionID                 = common.HexToHash("0x0102030405060708091011121314151617181920212223242526272829303132")
	ctfSafetyParentCollection            = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	ctfSafetyCollateral                  = mustContractConfigForTest(PolygonChainID).Collateral
	ctfSafetyCTFCollateralAdapter        = mustContractConfigForTest(PolygonChainID).CTFCollateralAdapter
	ctfSafetyNegRiskCTFCollateralAdapter = mustContractConfigForTest(PolygonChainID).NegRiskCTFCollateralAdapter
)

func TestBuildCTFTransactions_ABIRoundTrip(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))

	t.Run("splitPosition", func(t *testing.T) {
		req := &SplitPositionRequest{
			CollateralToken:    ctfSafetyCollateral,
			ParentCollectionID: ctfSafetyParentCollection,
			ConditionID:        ctfSafetyConditionID,
			Partition:          BinaryPartition(),
			Amount:             big.NewInt(1_000_000),
		}

		var tx CTFTransaction
		if err := client.BuildSplitPositionTx(req, &tx); err != nil {
			t.Fatalf("BuildSplitPositionTx: %v", err)
		}

		assertAddressEqualCTF(t, "to", ctfSafetyCTFCollateralAdapter, tx.To)
		assertMethodSelectorCTF(t, "splitPosition", ctfABI.Methods["splitPosition"].ID, tx.Data)

		values, err := ctfABI.Methods["splitPosition"].Inputs.Unpack(tx.Data[4:])
		if err != nil {
			t.Fatalf("unpack splitPosition calldata: %v", err)
		}

		assertAddressEqualCTF(t, "collateralToken", req.CollateralToken, values[0].(common.Address))
		assertHashEqualCTF(t, "parentCollectionID", req.ParentCollectionID, values[1].([32]byte))
		assertHashEqualCTF(t, "conditionID", req.ConditionID, values[2].([32]byte))
		assertBigIntSliceEqualCTF(t, "partition", req.Partition, values[3].([]*big.Int))
		assertBigIntEqualCTF(t, "amount", req.Amount, values[4].(*big.Int))
	})

	t.Run("mergePositions", func(t *testing.T) {
		req := &MergePositionsRequest{
			CollateralToken:    ctfSafetyCollateral,
			ParentCollectionID: ctfSafetyParentCollection,
			ConditionID:        ctfSafetyConditionID,
			Partition:          BinaryPartition(),
			Amount:             big.NewInt(2_000_000),
		}

		var tx CTFTransaction
		if err := client.BuildMergePositionsTx(req, &tx); err != nil {
			t.Fatalf("BuildMergePositionsTx: %v", err)
		}

		assertAddressEqualCTF(t, "to", ctfSafetyCTFCollateralAdapter, tx.To)
		assertMethodSelectorCTF(t, "mergePositions", ctfABI.Methods["mergePositions"].ID, tx.Data)

		values, err := ctfABI.Methods["mergePositions"].Inputs.Unpack(tx.Data[4:])
		if err != nil {
			t.Fatalf("unpack mergePositions calldata: %v", err)
		}

		assertAddressEqualCTF(t, "collateralToken", req.CollateralToken, values[0].(common.Address))
		assertHashEqualCTF(t, "parentCollectionID", req.ParentCollectionID, values[1].([32]byte))
		assertHashEqualCTF(t, "conditionID", req.ConditionID, values[2].([32]byte))
		assertBigIntSliceEqualCTF(t, "partition", req.Partition, values[3].([]*big.Int))
		assertBigIntEqualCTF(t, "amount", req.Amount, values[4].(*big.Int))
	})

	t.Run("redeemPositions", func(t *testing.T) {
		req := &RedeemPositionsRequest{
			CollateralToken:    ctfSafetyCollateral,
			ParentCollectionID: ctfSafetyParentCollection,
			ConditionID:        ctfSafetyConditionID,
			IndexSets:          BinaryPartition(),
		}

		var tx CTFTransaction
		if err := client.BuildRedeemPositionsTx(req, &tx); err != nil {
			t.Fatalf("BuildRedeemPositionsTx: %v", err)
		}

		assertAddressEqualCTF(t, "to", ctfSafetyCTFCollateralAdapter, tx.To)
		assertMethodSelectorCTF(t, "redeemPositions", ctfABI.Methods["redeemPositions"].ID, tx.Data)

		values, err := ctfABI.Methods["redeemPositions"].Inputs.Unpack(tx.Data[4:])
		if err != nil {
			t.Fatalf("unpack redeemPositions calldata: %v", err)
		}

		assertAddressEqualCTF(t, "collateralToken", req.CollateralToken, values[0].(common.Address))
		assertHashEqualCTF(t, "parentCollectionID", req.ParentCollectionID, values[1].([32]byte))
		assertHashEqualCTF(t, "conditionID", req.ConditionID, values[2].([32]byte))
		assertBigIntSliceEqualCTF(t, "indexSets", req.IndexSets, values[3].([]*big.Int))
	})

	t.Run("redeemNegRisk", func(t *testing.T) {
		req := &RedeemNegRiskRequest{
			ConditionID: ctfSafetyConditionID,
		}

		var tx CTFTransaction
		if err := client.BuildRedeemNegRiskTx(req, &tx); err != nil {
			t.Fatalf("BuildRedeemNegRiskTx: %v", err)
		}

		assertAddressEqualCTF(t, "to", ctfSafetyNegRiskCTFCollateralAdapter, tx.To)
		assertMethodSelectorCTF(t, "redeemPositions", ctfABI.Methods["redeemPositions"].ID, tx.Data)

		values, err := ctfABI.Methods["redeemPositions"].Inputs.Unpack(tx.Data[4:])
		if err != nil {
			t.Fatalf("unpack neg-risk redeemPositions calldata: %v", err)
		}

		assertAddressEqualCTF(t, "collateralToken placeholder", common.Address{}, values[0].(common.Address))
		assertHashEqualCTF(t, "parentCollectionID placeholder", common.Hash{}, values[1].([32]byte))
		assertHashEqualCTF(t, "conditionID", req.ConditionID, values[2].([32]byte))
		assertBigIntSliceEqualCTF(t, "indexSets placeholder", BinaryPartition(), values[3].([]*big.Int))
	})
}

func TestSubmitCTFRelayerTransaction_UsesSignedRelayerToAndData(t *testing.T) {
	mock := &captureCTFRelayer{}
	client := NewClient("", WithChainID(PolygonChainID), WithRelayerSubmitter(mock))

	rawTx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	req := &RelayerCTFRequest{
		From:        "0x0000000000000000000000000000000000000001",
		To:          "0x00000000000000000000000000000000000000bb",
		ProxyWallet: "0x0000000000000000000000000000000000000002",
		Data:        "0x01020304",
		Nonce:       "123",
		Signature:   "0xabcdef",
		Type:        relayer.NonceTypeSafe,
		Metadata:    "signed-payload",
		Value:       "0",
		SignatureParams: &relayer.SignatureParams{
			GasPrice:       "0",
			Operation:      "0",
			SafeTxGas:      "0",
			BaseGas:        "0",
			GasToken:       ZeroAddress,
			RefundReceiver: ZeroAddress,
		},
	}

	var out relayer.SubmitTransactionResponse
	if err := client.SubmitCTFRelayerTransaction(context.Background(), rawTx, req, &out); err != nil {
		t.Fatalf("SubmitCTFRelayerTransaction: %v", err)
	}

	if mock.submitted == nil {
		t.Fatal("relayer submit was not called")
	}

	assertStringEqualCTF(t, "submitted.To", req.To, mock.submitted.To)
	assertStringEqualCTF(t, "submitted.Data", req.Data, mock.submitted.Data)
	assertStringEqualCTF(t, "submitted.ProxyWallet", req.ProxyWallet, mock.submitted.ProxyWallet)
	assertStringEqualCTF(t, "submitted.Nonce", req.Nonce, mock.submitted.Nonce)
	assertStringEqualCTF(t, "submitted.Signature", req.Signature, mock.submitted.Signature)
	assertStringEqualCTF(t, "submitted.Metadata", req.Metadata, mock.submitted.Metadata)

	if mock.submitted.To == rawTx.To.Hex() {
		t.Fatalf("submitted.To used raw CTF tx target %s instead of signed relayer target %s", rawTx.To.Hex(), req.To)
	}
	if mock.submitted.Data == hexutil.Encode(rawTx.Data) {
		t.Fatalf("submitted.Data used raw CTF calldata %s instead of signed relayer data %s", hexutil.Encode(rawTx.Data), req.Data)
	}
}

func TestSubmitCTFRelayerTransaction_FallsBackToRawCTFToAndData(t *testing.T) {
	mock := &captureCTFRelayer{}
	client := NewClient("", WithChainID(PolygonChainID), WithRelayerSubmitter(mock))

	rawTx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	req := &RelayerCTFRequest{
		From:        "0x0000000000000000000000000000000000000001",
		ProxyWallet: "0x0000000000000000000000000000000000000002",
		Nonce:       "123",
		Signature:   "0xabcdef",
		Type:        relayer.NonceTypeProxy,
		Value:       "0",
	}

	if err := client.SubmitCTFRelayerTransaction(context.Background(), rawTx, req, &relayer.SubmitTransactionResponse{}); err != nil {
		t.Fatalf("SubmitCTFRelayerTransaction: %v", err)
	}

	if mock.submitted == nil {
		t.Fatal("relayer submit was not called")
	}

	assertStringEqualCTF(t, "submitted.To", rawTx.To.Hex(), mock.submitted.To)
	assertStringEqualCTF(t, "submitted.Data", hexutil.Encode(rawTx.Data), mock.submitted.Data)
}

func TestSubmitCTFRelayerTransaction_RejectsInvalidInputs(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID), WithRelayerSubmitter(&captureCTFRelayer{}))

	validTx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0x01},
	}

	tests := []struct {
		name string
		tx   *CTFTransaction
		req  *RelayerCTFRequest
		want string
	}{
		{
			name: "nil tx",
			tx:   nil,
			req:  &RelayerCTFRequest{ProxyWallet: "0x0000000000000000000000000000000000000002"},
			want: "nil CTF transaction",
		},
		{
			name: "nil request",
			tx:   validTx,
			req:  nil,
			want: "nil relayer CTF request",
		},
		{
			name: "empty proxy wallet",
			tx:   validTx,
			req:  &RelayerCTFRequest{},
			want: "proxy wallet is required",
		},
		{
			name: "invalid proxy wallet",
			tx:   validTx,
			req:  &RelayerCTFRequest{ProxyWallet: "not-an-address"},
			want: "proxy wallet must be a valid hex address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SubmitCTFRelayerTransaction(context.Background(), tt.tx, tt.req, &relayer.SubmitTransactionResponse{})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestBuildCTFRelayerRequest_PreservesSignedProxyPayload(t *testing.T) {
	signer, err := ParsePrivateKey(ctfSafetyTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	mock := &captureCTFRelayer{
		proxySubmit: relayer.SubmitTransactionRequest{
			From:        signer.Address().Hex(),
			To:          "0x0000000000000000000000000000000000000f01",
			ProxyWallet: "0x0000000000000000000000000000000000000002",
			Data:        "0xfeedface",
			Nonce:       "456",
			Signature:   "0xproxy",
			Type:        relayer.NonceTypeProxy,
			Metadata:    "proxy-metadata",
			Value:       "0",
			SignatureParams: &relayer.SignatureParams{
				GasPrice:   "0",
				GasLimit:   "3000000",
				RelayerFee: "0",
				RelayHub:   "0x0000000000000000000000000000000000000003",
				Relay:      "0x0000000000000000000000000000000000000004",
			},
		},
	}
	client := NewClient("", WithSigner(signer), WithChainID(PolygonChainID), WithRelayerSubmitter(mock))

	rawTx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	var out RelayerCTFRequest
	err = client.BuildCTFRelayerRequest(context.Background(), rawTx, &CTFRelayerArgs{
		Type:        relayer.NonceTypeProxy,
		ProxyWallet: "0x0000000000000000000000000000000000000002",
		Metadata:    "proxy-metadata",
		GasLimit:    "3000000",
	}, &out)
	if err != nil {
		t.Fatalf("BuildCTFRelayerRequest: %v", err)
	}

	if mock.proxyArgs == nil {
		t.Fatal("ProxySubmitTransactionRequest was not called")
	}

	assertStringEqualCTF(t, "proxyArgs.ProxyWallet", "0x0000000000000000000000000000000000000002", mock.proxyArgs.ProxyWallet)
	assertStringEqualCTF(t, "proxyArgs.Metadata", "proxy-metadata", mock.proxyArgs.Metadata)
	assertStringEqualCTF(t, "proxyArgs.GasLimit", "3000000", mock.proxyArgs.GasLimit)

	assertStringEqualCTF(t, "out.To", mock.proxySubmit.To, out.To)
	assertStringEqualCTF(t, "out.Data", mock.proxySubmit.Data, out.Data)
	assertStringEqualCTF(t, "out.ProxyWallet", mock.proxySubmit.ProxyWallet, out.ProxyWallet)
	assertStringEqualCTF(t, "out.Nonce", mock.proxySubmit.Nonce, out.Nonce)
	assertStringEqualCTF(t, "out.Signature", mock.proxySubmit.Signature, out.Signature)
	assertStringEqualCTF(t, "out.Metadata", mock.proxySubmit.Metadata, out.Metadata)

	if out.To == rawTx.To.Hex() {
		t.Fatalf("BuildCTFRelayerRequest lost signed proxy To and fell back to raw CTF tx target")
	}
	if out.Data == hexutil.Encode(rawTx.Data) {
		t.Fatalf("BuildCTFRelayerRequest lost signed proxy Data and fell back to raw CTF calldata")
	}
}

func TestBuildCTFRelayerRequest_PreservesSignedSafePayload(t *testing.T) {
	signer, err := ParsePrivateKey(ctfSafetyTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	mock := &captureCTFRelayer{
		safeSubmit: relayer.SubmitTransactionRequest{
			From:        signer.Address().Hex(),
			To:          "0x0000000000000000000000000000000000000f02",
			ProxyWallet: "0x0000000000000000000000000000000000000005",
			Data:        "0xcafebabe",
			Nonce:       "789",
			Signature:   "0xsafe",
			Type:        relayer.NonceTypeSafe,
			Metadata:    "safe-metadata",
			Value:       "0",
			SignatureParams: &relayer.SignatureParams{
				GasPrice:       "0",
				Operation:      "0",
				SafeTxGas:      "0",
				BaseGas:        "0",
				GasToken:       ZeroAddress,
				RefundReceiver: ZeroAddress,
			},
		},
	}
	client := NewClient("", WithSigner(signer), WithChainID(PolygonChainID), WithRelayerSubmitter(mock))

	rawTx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0xde, 0xad, 0xbe, 0xef},
	}

	var out RelayerCTFRequest
	err = client.BuildCTFRelayerRequest(context.Background(), rawTx, &CTFRelayerArgs{
		Type:        relayer.NonceTypeSafe,
		ProxyWallet: "0x0000000000000000000000000000000000000005",
		Metadata:    "safe-metadata",
	}, &out)
	if err != nil {
		t.Fatalf("BuildCTFRelayerRequest: %v", err)
	}

	if mock.safeArgs == nil {
		t.Fatal("SafeSubmitTransactionRequest was not called")
	}

	assertStringEqualCTF(t, "safeArgs.ProxyWallet", "0x0000000000000000000000000000000000000005", mock.safeArgs.ProxyWallet)
	assertStringEqualCTF(t, "safeArgs.Metadata", "safe-metadata", mock.safeArgs.Metadata)
	if mock.safeArgs.ChainID != PolygonChainID {
		t.Fatalf("safeArgs.ChainID = %d, want %d", mock.safeArgs.ChainID, PolygonChainID)
	}
	if len(mock.safeArgs.Transactions) != 1 {
		t.Fatalf("safeArgs.Transactions length = %d, want 1", len(mock.safeArgs.Transactions))
	}
	assertStringEqualCTF(t, "safe tx.to", rawTx.To.Hex(), mock.safeArgs.Transactions[0].To)
	assertStringEqualCTF(t, "safe tx.data", hexutil.Encode(rawTx.Data), mock.safeArgs.Transactions[0].Data)
	if mock.safeArgs.Transactions[0].Operation != relayer.OperationCall {
		t.Fatalf("safe tx operation = %d, want %d", mock.safeArgs.Transactions[0].Operation, relayer.OperationCall)
	}

	assertStringEqualCTF(t, "out.To", mock.safeSubmit.To, out.To)
	assertStringEqualCTF(t, "out.Data", mock.safeSubmit.Data, out.Data)
	assertStringEqualCTF(t, "out.ProxyWallet", mock.safeSubmit.ProxyWallet, out.ProxyWallet)
	assertStringEqualCTF(t, "out.Nonce", mock.safeSubmit.Nonce, out.Nonce)
	assertStringEqualCTF(t, "out.Signature", mock.safeSubmit.Signature, out.Signature)
	assertStringEqualCTF(t, "out.Metadata", mock.safeSubmit.Metadata, out.Metadata)

	if out.To == rawTx.To.Hex() {
		t.Fatalf("BuildCTFRelayerRequest lost signed safe To and fell back to raw CTF tx target")
	}
	if out.Data == hexutil.Encode(rawTx.Data) {
		t.Fatalf("BuildCTFRelayerRequest lost signed safe Data and fell back to raw CTF calldata")
	}
}

func TestBuildCTFRelayerRequest_RejectsInvalidInputs(t *testing.T) {
	signer, err := ParsePrivateKey(ctfSafetyTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	validTx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0x01},
	}

	tests := []struct {
		name   string
		client *Client
		tx     *CTFTransaction
		args   *CTFRelayerArgs
		out    *RelayerCTFRequest
		want   string
	}{
		{
			name:   "nil tx",
			client: NewClient("", WithSigner(signer), WithChainID(PolygonChainID), WithRelayerSubmitter(&captureCTFRelayer{})),
			tx:     nil,
			args:   &CTFRelayerArgs{Type: relayer.NonceTypeProxy, ProxyWallet: "0x0000000000000000000000000000000000000002"},
			out:    &RelayerCTFRequest{},
			want:   "nil CTF transaction",
		},
		{
			name:   "nil output",
			client: NewClient("", WithSigner(signer), WithChainID(PolygonChainID), WithRelayerSubmitter(&captureCTFRelayer{})),
			tx:     validTx,
			args:   &CTFRelayerArgs{Type: relayer.NonceTypeProxy, ProxyWallet: "0x0000000000000000000000000000000000000002"},
			out:    nil,
			want:   "nil relayer CTF request output",
		},
		{
			name:   "missing signer",
			client: NewClient("", WithChainID(PolygonChainID), WithRelayerSubmitter(&captureCTFRelayer{})),
			tx:     validTx,
			args:   &CTFRelayerArgs{Type: relayer.NonceTypeProxy, ProxyWallet: "0x0000000000000000000000000000000000000002"},
			out:    &RelayerCTFRequest{},
			want:   "signer is required",
		},
		{
			name:   "missing relayer",
			client: NewClient("", WithSigner(signer), WithChainID(PolygonChainID)),
			tx:     validTx,
			args:   &CTFRelayerArgs{Type: relayer.NonceTypeProxy, ProxyWallet: "0x0000000000000000000000000000000000000002"},
			out:    &RelayerCTFRequest{},
			want:   "relayer client is required",
		},
		{
			name:   "unsupported type",
			client: NewClient("", WithSigner(signer), WithChainID(PolygonChainID), WithRelayerSubmitter(&captureCTFRelayer{})),
			tx:     validTx,
			args:   &CTFRelayerArgs{Type: relayer.NonceType("BOGUS"), ProxyWallet: "0x0000000000000000000000000000000000000002"},
			out:    &RelayerCTFRequest{},
			want:   "unsupported relayer type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.BuildCTFRelayerRequest(context.Background(), tt.tx, tt.args, tt.out)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestBuildCTFRelayerRequest_NilArgsDoesNotPanic(t *testing.T) {
	signer, err := ParsePrivateKey(ctfSafetyTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient("", WithSigner(signer), WithChainID(PolygonChainID), WithRelayerSubmitter(&captureCTFRelayer{}))

	var out RelayerCTFRequest
	err = client.BuildCTFRelayerRequest(context.Background(), &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0x01},
	}, nil, &out)
	if err == nil {
		t.Fatal("expected unsupported relayer type error for nil args")
	}
	if !strings.Contains(err.Error(), "unsupported relayer type") {
		t.Fatalf("error = %q, want unsupported relayer type", err.Error())
	}
}

func TestBuildCTFRelayerRequestRejectsDepositWalletTypes(t *testing.T) {
	signer, err := ParsePrivateKey(ctfSafetyTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
		WithRelayerSubmitter(&captureCTFRelayer{}),
	)

	tx := &CTFTransaction{
		To:   common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		Data: []byte{0x01},
	}

	tests := []struct {
		name string
		typ  relayer.NonceType
	}{
		{
			name: "wallet",
			typ:  relayer.NonceTypeWallet,
		},
		{
			name: "wallet create",
			typ:  relayer.NonceTypeWalletCreate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out RelayerCTFRequest
			err := client.BuildCTFRelayerRequest(context.Background(), tx, &CTFRelayerArgs{
				Type:        tt.typ,
				ProxyWallet: "0x0000000000000000000000000000000000000002",
			}, &out)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "deposit wallet CTF transactions require CTFDepositWalletTransactionRequest") {
				t.Fatalf("error = %q", err.Error())
			}
		})
	}
}

type captureCTFRelayer struct {
	submitted *relayer.SubmitTransactionRequest

	proxyArgs   *relayer.ProxySubmitTransactionArgs
	proxySubmit relayer.SubmitTransactionRequest

	safeArgs   *relayer.SafeSubmitTransactionArgs
	safeSubmit relayer.SubmitTransactionRequest
}

func (m *captureCTFRelayer) SubmitTransaction(_ context.Context, req *relayer.SubmitTransactionRequest, out *relayer.SubmitTransactionResponse) error {
	if req != nil {
		copied := *req
		m.submitted = &copied
	}
	if out != nil {
		*out = relayer.SubmitTransactionResponse{
			TransactionID: "test-relayer-tx",
			State:         "submitted",
		}
	}
	return nil
}

func (m *captureCTFRelayer) ProxySubmitTransactionRequest(
	_ context.Context,
	_ *polyauth.Signer,
	args *relayer.ProxySubmitTransactionArgs,
	out *relayer.SubmitTransactionRequest,
) error {
	if args != nil {
		copied := *args
		m.proxyArgs = &copied
	}
	if out != nil {
		*out = m.proxySubmit
	}
	return nil
}

func (m *captureCTFRelayer) SafeSubmitTransactionRequest(
	_ context.Context,
	_ *polyauth.Signer,
	args *relayer.SafeSubmitTransactionArgs,
	out *relayer.SubmitTransactionRequest,
) error {
	if args != nil {
		copied := *args
		m.safeArgs = &copied
	}
	if out != nil {
		*out = m.safeSubmit
	}
	return nil
}

func mustContractConfigForTest(chainID int64) ContractConfig {
	cfg, err := Contracts(chainID)
	if err != nil {
		panic(err)
	}
	return cfg
}

func assertMethodSelectorCTF(t *testing.T, name string, expected []byte, data []byte) {
	t.Helper()

	if len(data) < 4 {
		t.Fatalf("%s calldata too short: %x", name, data)
	}
	if !reflect.DeepEqual(expected, data[:4]) {
		t.Fatalf("%s selector mismatch: expected 0x%x, got 0x%x", name, expected, data[:4])
	}
}

func assertAddressEqualCTF(t *testing.T, name string, expected, actual common.Address) {
	t.Helper()

	if expected != actual {
		t.Fatalf("%s mismatch: expected %s, got %s", name, expected.Hex(), actual.Hex())
	}
}

func assertHashEqualCTF(t *testing.T, name string, expected common.Hash, actual [32]byte) {
	t.Helper()

	got := common.BytesToHash(actual[:])
	if expected != got {
		t.Fatalf("%s mismatch: expected %s, got %s", name, expected.Hex(), got.Hex())
	}
}

func assertBigIntEqualCTF(t *testing.T, name string, expected, actual *big.Int) {
	t.Helper()

	if expected == nil || actual == nil {
		if expected != actual {
			t.Fatalf("%s mismatch: expected %v, got %v", name, expected, actual)
		}
		return
	}
	if expected.Cmp(actual) != 0 {
		t.Fatalf("%s mismatch: expected %s, got %s", name, expected.String(), actual.String())
	}
}

func assertBigIntSliceEqualCTF(t *testing.T, name string, expected, actual []*big.Int) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("%s length mismatch: expected %d, got %d", name, len(expected), len(actual))
	}
	for i := range expected {
		assertBigIntEqualCTF(t, name+"["+string(rune('0'+i))+"]", expected[i], actual[i])
	}
}

func assertStringEqualCTF(t *testing.T, name string, expected, actual string) {
	t.Helper()

	if expected != actual {
		t.Fatalf("%s mismatch: expected %s, got %s", name, expected, actual)
	}
}
