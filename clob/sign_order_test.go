package clob

import (
	"math/big"
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
	if order.Taker != ZeroAddress || order.Expiration.String() != "0" || order.Nonce.String() != "0" || order.FeeRateBps.String() != "0" {
		t.Fatalf("unexpected wire compatibility defaults: %#v", order)
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
