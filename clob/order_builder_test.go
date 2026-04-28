package clob

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bububa/polymarket-client/internal/polyauth"
)

func testKey() *polyauth.Signer {
	s, err := polyauth.ParsePrivateKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		panic(err)
	}
	return s
}

func TestBuildOrder_HappyPath(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	order, err := b.BuildOrder(OrderArgsV2{
		TokenID: "123456",
		Price:   "0.67",
		Size:    "10",
		Side:    Buy,
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.MakerAmount.String() != "6700000" {
		t.Errorf("makerAmount = %s, want 6700000", order.MakerAmount)
	}
	if order.TakerAmount.String() != "10000000" {
		t.Errorf("takerAmount = %s, want 10000000", order.TakerAmount)
	}
	if order.Side != Buy {
		t.Errorf("side = %s, want BUY", order.Side)
	}
	if order.TokenID.String() != "123456" {
		t.Errorf("tokenId = %s", order.TokenID)
	}
	if order.Signature == "" {
		t.Fatal("signature is empty")
	}
	if order.Expiration.String() != "0" {
		t.Errorf("expiration = %s, want 0", order.Expiration)
	}
}

func TestBuildOrder_Sell(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	order, err := b.BuildOrder(OrderArgsV2{
		TokenID: "123456",
		Price:   "0.25",
		Size:    "100",
		Side:    Sell,
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.MakerAmount.String() != "100000000" {
		t.Errorf("makerAmount = %s, want 100000000", order.MakerAmount)
	}
	if order.TakerAmount.String() != "25000000" {
		t.Errorf("takerAmount = %s, want 25000000", order.TakerAmount)
	}
}

func TestBuildOrder_InvalidParams(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	_, err := b.BuildOrder(OrderArgsV2{TokenID: "1", Price: "bad", Size: "10", Side: Buy})
	if err == nil {
		t.Fatal("expected error for invalid price")
	}

	_, err = b.BuildOrder(OrderArgsV2{TokenID: "1", Price: "0.5", Size: "xyz", Side: Buy})
	if err == nil {
		t.Fatal("expected error for invalid size")
	}
}

func TestBuildOrder_WithCustomExpiration(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	order, err := b.BuildOrder(OrderArgsV2{
		TokenID:    "123456",
		Price:      "0.50",
		Size:       "10",
		Side:       Buy,
		Expiration: "9999999999",
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Expiration.String() != "9999999999" {
		t.Errorf("expiration = %s, want 9999999999", order.Expiration)
	}
}

func TestBuildMarketOrder_HappyPath(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	order, err := b.BuildMarketOrder(MarketOrderArgsV2{
		TokenID:    "789",
		WorstPrice: "0.99",
		Size:       "50",
		Side:       Buy,
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.MakerAmount.String() != "49500000" {
		t.Errorf("makerAmount = %s, want 49500000", order.MakerAmount)
	}
	if order.TakerAmount.String() != "50000000" {
		t.Errorf("takerAmount = %s, want 50000000", order.TakerAmount)
	}
	if order.Builder != ZeroBytes32 {
		t.Errorf("market order builder = %q, want %q", order.Builder, ZeroBytes32)
	}
}

func TestCreateAndPostOrder_WithServer(t *testing.T) {
	signer := testKey()

	var gotReq PostOrderRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/order" {
			http.Error(w, "not found", 404)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PostOrderResponse{Success: true, OrderID: "order-123"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, WithSigner(signer), WithChainID(PolygonChainID), WithCredentials(Credentials{Key: "test-key", Secret: "test-secret", Passphrase: "test-passphrase"}))
	b := NewOrderBuilder(client)

	resp, err := b.CreateAndPostOrder(context.Background(), OrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Size:    "20",
		Side:    Buy,
	}, GTC, false)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.OrderID != "order-123" {
		t.Errorf("orderID = %s, want order-123", resp.OrderID)
	}

	if gotReq.Order.Signature == "" {
		t.Fatal("posted order has no signature")
	}
	if gotReq.Order.MakerAmount.String() != "10000000" {
		t.Errorf("posted makerAmount = %s, want 10000000", gotReq.Order.MakerAmount)
	}
	if gotReq.Order.TakerAmount.String() != "20000000" {
		t.Errorf("posted takerAmount = %s, want 20000000", gotReq.Order.TakerAmount)
	}
	if gotReq.OrderType != GTC {
		t.Errorf("posted orderType = %s, want GTC", gotReq.OrderType)
	}
}

func TestCreateAndPostMarketOrder_WithServer(t *testing.T) {
	signer := testKey()

	var gotReq PostOrderRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PostOrderResponse{Success: true, OrderID: "mkt-456"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, WithSigner(signer), WithChainID(PolygonChainID), WithCredentials(Credentials{Key: "test-key", Secret: "test-secret", Passphrase: "test-passphrase"}))
	b := NewOrderBuilder(client)

	resp, err := b.CreateAndPostMarketOrder(context.Background(), MarketOrderArgsV2{
		TokenID:    "789012",
		WorstPrice: "0.95",
		Size:       "30",
		Side:       Buy,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
	if gotReq.OrderType != GTC {
		t.Errorf("market post orderType = %s, want GTC", gotReq.OrderType)
	}
}
