package clob

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/bububa/polymarket-client/internal/polyauth"
)

func TestSignOrderFillsV2Defaults(t *testing.T) {
	signer, err := polyauth.ParsePrivateKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient("", WithSigner(signer))
	order := SignedOrder{
		TokenID:     "123",
		MakerAmount: "1000000",
		TakerAmount: "500000",
		Side:        Buy,
	}

	err = client.SignOrder(
		&order,
		WithSignOrderSalt(big.NewInt(42)),
		WithSignOrderTime(time.UnixMilli(1700000000000)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if order.Salt != 42 {
		t.Fatalf("salt = %d, want 42", order.Salt)
	}
	if order.Timestamp.String() != "1700000000000" {
		t.Fatalf("timestamp = %s, want 1700000000000", order.Timestamp)
	}
	if order.Metadata != ZeroBytes32 || order.Builder != ZeroBytes32 {
		t.Fatalf("unexpected metadata/builder defaults: %q %q", order.Metadata, order.Builder)
	}
	if order.Signature == "" {
		t.Fatal("signature is empty")
	}
}

func TestSignOrderRejectsSignerMismatch(t *testing.T) {
	signer, err := polyauth.ParsePrivateKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	order := SignedOrder{
		Maker:       signer.Address().Hex(),
		Signer:      "0x0000000000000000000000000000000000000001",
		TokenID:     "123",
		MakerAmount: "1000000",
		TakerAmount: "500000",
		Side:        Sell,
	}
	if err := SignOrder(signer, PolygonChainID, &order); err == nil {
		t.Fatal("expected signer mismatch error")
	}
}

func TestSignatureTypeEnumValues(t *testing.T) {
	if SignatureTypeEOA != 0 {
		t.Fatalf("SignatureTypeEOA = %d, want 0", SignatureTypeEOA)
	}
	if SignatureTypeProxy != 1 {
		t.Fatalf("SignatureTypeProxy = %d, want 1", SignatureTypeProxy)
	}
	if SignatureTypeGnosisSafe != 2 {
		t.Fatalf("SignatureTypeGnosisSafe = %d, want 2", SignatureTypeGnosisSafe)
	}
	if SignatureTypePoly1271 != 3 {
		t.Fatalf("SignatureTypePoly1271 = %d, want 3", SignatureTypePoly1271)
	}
}

func TestSignedOrderJSONMarshal_NoV1Fields(t *testing.T) {
	order := SignedOrder{
		Salt:          42,
		Maker:         "0x0000000000000000000000000000000000000001",
		Signer:        "0x0000000000000000000000000000000000000002",
		TokenID:       "123",
		MakerAmount:   "1000000",
		TakerAmount:   "500000",
		Side:          Buy,
		SignatureType: SignatureTypeEOA,
		Timestamp:     "1700000000000",
		Metadata:      ZeroBytes32,
		Builder:       ZeroBytes32,
		Signature:     "0xdead",
	}
	b, err := json.Marshal(order)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, key := range []string{`"taker"`, `"nonce"`, `"feeRateBps"`} {
		if strings.Contains(s, key) {
			t.Fatalf("v1 field %q found in JSON: %s", key, s)
		}
	}
}

func TestSignedOrderJSON_ExpirationOmitEmpty(t *testing.T) {
	gtc := SignedOrder{
		Salt:          42,
		Maker:         "0x0000000000000000000000000000000000000001",
		Signer:        "0x0000000000000000000000000000000000000002",
		TokenID:       "123",
		MakerAmount:   "1000000",
		TakerAmount:   "500000",
		Side:          Buy,
		SignatureType: SignatureTypeEOA,
		Timestamp:     "1700000000000",
		Metadata:      ZeroBytes32,
		Builder:       ZeroBytes32,
		Signature:     "0xdead",
	}
	b, _ := json.Marshal(gtc)
	if strings.Contains(string(b), `"expiration"`) {
		t.Fatal("GTC order should not include expiration field")
	}

	gtd := gtc
	gtd.Expiration = "1735689600"
	b, _ = json.Marshal(gtd)
	if !strings.Contains(string(b), `"expiration"`) {
		t.Fatal("GTD order should include expiration field")
	}
	if !strings.Contains(string(b), `"1735689600"`) {
		t.Fatal("GTD order expiration should have correct value")
	}
}

func TestEIP712TypedData_ExcludesV2ForbiddenFields(t *testing.T) {
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	td := buildOrderTypedData(PolygonChainID, contracts.Exchange, SignedOrder{
		Salt:          42,
		Maker:         "0x0000000000000000000000000000000000000001",
		Signer:        "0x0000000000000000000000000000000000000002",
		TokenID:       "123",
		MakerAmount:   "1000000",
		TakerAmount:   "500000",
		Expiration:    "1735689600",
		Side:          Buy,
		SignatureType: SignatureTypeEOA,
		Timestamp:     "1700000000000",
		Metadata:      ZeroBytes32,
		Builder:       ZeroBytes32,
	})
	orderTypes := td.Types["Order"]
	names := make(map[string]bool)
	for _, f := range orderTypes {
		names[f.Name] = true
	}

	for _, excluded := range []string{"taker", "expiration", "nonce", "feeRateBps"} {
		if names[excluded] {
			t.Fatalf("EIP-712 Order type must not contain %q", excluded)
		}
	}
	for _, required := range []string{"salt", "maker", "signer", "tokenId", "makerAmount", "takerAmount", "side", "signatureType", "timestamp", "metadata", "builder"} {
		if !names[required] {
			t.Fatalf("EIP-712 Order type must contain %q", required)
		}
	}
	if td.Domain.Version != "2" {
		t.Fatalf("EIP-712 domain version = %q, want '2'", td.Domain.Version)
	}
	if td.Domain.Name != "Polymarket CTF Exchange" {
		t.Fatalf("EIP-712 domain name = %q, want 'Polymarket CTF Exchange'", td.Domain.Name)
	}
}

func TestEIP712TypedData_HashMatchesV2Schema(t *testing.T) {
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	order := SignedOrder{
		Salt:          42,
		Maker:         "0x0000000000000000000000000000000000000001",
		Signer:        "0x0000000000000000000000000000000000000001",
		TokenID:       "123",
		MakerAmount:   "1000000",
		TakerAmount:   "500000",
		Side:          Buy,
		SignatureType: SignatureTypeEOA,
		Timestamp:     "1700000000000",
		Metadata:      ZeroBytes32,
		Builder:       ZeroBytes32,
	}
	td := buildOrderTypedData(PolygonChainID, contracts.Exchange, order)
	hash, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		t.Fatal(err)
	}
	if hexutil.Encode(hash) == "0x0000000000000000000000000000000000000000000000000000000000000000" {
		t.Fatal("EIP-712 hash should not be zero")
	}
}

func TestEIP712TypedData_WithExpirationExcluded(t *testing.T) {
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	order := SignedOrder{
		Salt:          42,
		Maker:         "0x0000000000000000000000000000000000000001",
		Signer:        "0x0000000000000000000000000000000000000001",
		TokenID:       "123",
		MakerAmount:   "1000000",
		TakerAmount:   "500000",
		Expiration:    "9999999999",
		Side:          Buy,
		SignatureType: SignatureTypeEOA,
		Timestamp:     "1700000000000",
		Metadata:      ZeroBytes32,
		Builder:       ZeroBytes32,
	}
	td := buildOrderTypedData(PolygonChainID, contracts.Exchange, order)
	msg := td.Message
	if _, hasExpiration := msg["expiration"]; hasExpiration {
		t.Fatal("EIP-712 Message must not contain expiration even when set on order")
	}
	for _, key := range []string{"timestamp", "metadata", "builder"} {
		if _, ok := msg[key]; !ok {
			t.Fatalf("EIP-712 Message must contain %q", key)
		}
	}
}

func TestPostOrderRequestJSON(t *testing.T) {
	req := PostOrderRequest{
		Order: SignedOrder{
			Salt:          1,
			Maker:         "0x0000000000000000000000000000000000000001",
			Signer:        "0x0000000000000000000000000000000000000001",
			TokenID:       "123",
			MakerAmount:   "1000000",
			TakerAmount:   "500000",
			Side:          Buy,
			SignatureType: SignatureTypeEOA,
			Timestamp:     "1700000000000",
			Metadata:      ZeroBytes32,
			Builder:       ZeroBytes32,
			Signature:     "0xdead",
		},
		Owner:     "0x0000000000000000000000000000000000000001",
		OrderType: GTC,
	}
	b, _ := json.Marshal(req)
	s := string(b)

	for _, want := range []string{`"order"`, `"owner"`, `"orderType"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("PostOrderRequest JSON missing %q: %s", want, s)
		}
	}
	for _, excluded := range []string{`"taker"`, `"nonce"`, `"feeRateBps"`} {
		if strings.Contains(s, excluded) {
			t.Fatalf("PostOrderRequest JSON contains v1 field %q: %s", excluded, s)
		}
	}
}

func TestNewClientDefaultHost(t *testing.T) {
	c := NewClient("")
	if c.Host() != MainnetHost {
		t.Fatalf("NewClient(\"\").Host() = %q, want %q", c.Host(), MainnetHost)
	}
}
