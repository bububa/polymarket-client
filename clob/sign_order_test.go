package clob

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

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
	if order.Salt.String() != "42" {
		t.Fatalf("salt = %s, want 42", order.Salt)
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
		Salt:          "42",
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
		Salt:          "42",
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
