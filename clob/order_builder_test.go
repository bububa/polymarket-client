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
	}, CreateOrderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if order.MakerAmount.String() != "6700000" {
		t.Errorf("makerAmount = %s, want 6700000", order.MakerAmount)
	}
	if order.TakerAmount.String() != "10000000" {
		t.Errorf("takerAmount = %s, want 10000000", order.TakerAmount)
	}
	if order.Signature == "" {
		t.Fatal("signature is empty")
	}
	if order.Expiration.String() != "0" {
		t.Errorf("expiration = %s, want 0", order.Expiration)
	}
}

func TestBuildOrder_WithTickRounding(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	order, err := b.BuildOrder(OrderArgsV2{
		TokenID: "123456",
		Price:   "0.673",
		Size:    "10",
		Side:    Buy,
	}, CreateOrderOptions{TickSize: "0.01"})
	if err != nil {
		t.Fatal(err)
	}
	// 0.673 rounded down to 0.01 tick = 0.67
	if order.MakerAmount.String() != "6700000" {
		t.Errorf("makerAmount = %s, want 6700000 (tick-rounded)", order.MakerAmount)
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
	}, CreateOrderOptions{})
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
	}, CreateOrderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if order.Expiration.String() != "9999999999" {
		t.Errorf("expiration = %s, want 9999999999", order.Expiration)
	}
}

func TestBuildOrder_WithNegRisk(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	_, err := b.BuildOrder(OrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Size:    "10",
		Side:    Buy,
	}, CreateOrderOptions{NegRisk: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBuildOrder_InvalidBytes32(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	_, err := b.BuildOrder(OrderArgsV2{
		TokenID:     "123456",
		Price:       "0.50",
		Size:        "10",
		Side:        Buy,
		BuilderCode: "not-valid-hex",
	}, CreateOrderOptions{})
	if err == nil {
		t.Fatal("expected error for invalid builder code")
	}

	_, err = b.BuildOrder(OrderArgsV2{
		TokenID:  "123456",
		Price:    "0.50",
		Size:     "10",
		Side:     Buy,
		Metadata: "0xabc",
	}, CreateOrderOptions{})
	if err == nil {
		t.Fatal("expected error for invalid metadata")
	}
}

func TestBuildMarketOrder_HappyPath_BUY(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	order, err := b.BuildMarketOrder(MarketOrderArgsV2{
		TokenID: "789",
		Price:   "0.5",
		Amount:  "100",
		Side:    Buy,
	}, CreateOrderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// BUY: makerAmount = 100 USDC, takerAmount = 100/0.5 = 200 shares
	if order.MakerAmount.String() != "100000000" {
		t.Errorf("makerAmount = %s, want 100000000", order.MakerAmount)
	}
	if order.TakerAmount.String() != "200000000" {
		t.Errorf("takerAmount = %s, want 200000000", order.TakerAmount)
	}
	if order.Builder != ZeroBytes32 {
		t.Errorf("market order builder = %q, want %q", order.Builder, ZeroBytes32)
	}
}

func TestBuildMarketOrder_HappyPath_SELL(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	order, err := b.BuildMarketOrder(MarketOrderArgsV2{
		TokenID: "789",
		Price:   "0.45",
		Amount:  "200",
		Side:    Sell,
	}, CreateOrderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// SELL: makerAmount = 200 shares, takerAmount = 200*0.45 = 90 USDC
	if order.MakerAmount.String() != "200000000" {
		t.Errorf("makerAmount = %s, want 200000000", order.MakerAmount)
	}
	if order.TakerAmount.String() != "90000000" {
		t.Errorf("takerAmount = %s, want 90000000", order.TakerAmount)
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

	client := NewClient(srv.URL, WithSigner(signer), WithChainID(PolygonChainID), WithCredentials(Credentials{Key: "test-key"}))
	b := NewOrderBuilder(client)

	resp, err := b.CreateAndPostOrder(context.Background(), OrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Size:    "20",
		Side:    Buy,
	}, CreateOrderOptions{}, GTC, false)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
	if gotReq.OrderType != GTC {
		t.Errorf("posted orderType = %s, want GTC", gotReq.OrderType)
	}
}

func TestCreateAndPostMarketOrder_FOK(t *testing.T) {
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

	client := NewClient(srv.URL, WithSigner(signer), WithChainID(PolygonChainID), WithCredentials(Credentials{Key: "test-key"}))
	b := NewOrderBuilder(client)

	resp, err := b.CreateAndPostMarketOrder(context.Background(), MarketOrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Amount:  "100",
		Side:    Buy,
	}, CreateOrderOptions{}, FOK, false)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
	if gotReq.OrderType != FOK {
		t.Errorf("market post orderType = %s, want FOK", gotReq.OrderType)
	}
}

func TestCreateAndPostMarketOrder_RejectGTC(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	_, err := b.CreateAndPostMarketOrder(context.Background(), MarketOrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Amount:  "100",
		Side:    Buy,
	}, CreateOrderOptions{}, GTC, false)
	if err == nil {
		t.Fatal("expected error for GTC market order")
	}
}

func TestCreateAndPostMarketOrder_RejectGTD(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	_, err := b.CreateAndPostMarketOrder(context.Background(), MarketOrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Amount:  "100",
		Side:    Buy,
	}, CreateOrderOptions{}, GTD, false)
	if err == nil {
		t.Fatal("expected error for GTD market order")
	}
}
