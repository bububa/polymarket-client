package clob

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestBuildOrder_TickSizeRejectsMisaligned(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	_, err := b.BuildOrder(OrderArgsV2{
		TokenID: "123456",
		Price:   "0.673",
		Size:    "10",
		Side:    Buy,
	}, CreateOrderOptions{TickSize: "0.01"})
	if err == nil {
		t.Fatal("expected error for price not aligned to tick size")
	}
}

func TestBuildOrder_TickSizeAcceptsAligned(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	_, err := b.BuildOrder(OrderArgsV2{
		TokenID: "123456",
		Price:   "0.67",
		Size:    "10",
		Side:    Buy,
	}, CreateOrderOptions{TickSize: "0.01"})
	if err != nil {
		t.Fatal(err)
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
	// SELL: takerAmount = ceil(price * size * 1e6) = ceil(25000000) = 25000000
	if order.TakerAmount.String() != "25000000" {
		t.Errorf("takerAmount = %s, want 25000000", order.TakerAmount)
	}
}

func TestBuildOrder_SellPriceInvariant(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	// SELL with non-integer takerAmount to verify ceil protection
	// price=0.3333333, size=1.0:
	// raw takerAmount = 333333.3 → ceil → 333334
	// implied price = 333334 / 1000000 = 0.333334 >= 0.3333333
	order, err := b.BuildOrder(OrderArgsV2{
		TokenID: "123456",
		Price:   "0.3333333",
		Size:    "1.0",
		Side:    Sell,
	}, CreateOrderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	takerVal, err := strconv.ParseInt(order.TakerAmount.String(), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	makerVal, err := strconv.ParseInt(order.MakerAmount.String(), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if takerVal == 0 || makerVal == 0 {
		t.Fatal("zero amount")
	}
	impliedPrice := float64(takerVal) / float64(makerVal)
	if impliedPrice < 0.3333333 {
		t.Errorf("SELL implied price = %f, must be >= 0.3333333 (limit price)", impliedPrice)
	}
}

func TestBuildOrder_BuyPriceInvariant(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	// BUY with non-integer takerAmount
	// price=0.3333333, size=0.7:
	// makerAmount = floor(233333.31) = 233333
	// takerAmount = floor(700000) = 700000
	// implied price = 233333 / 700000 = 0.333332... <= 0.3333333
	order, err := b.BuildOrder(OrderArgsV2{
		TokenID: "123456",
		Price:   "0.3333333",
		Size:    "0.7",
		Side:    Buy,
	}, CreateOrderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	makerVal, err := strconv.ParseInt(order.MakerAmount.String(), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	takerVal, err := strconv.ParseInt(order.TakerAmount.String(), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	if makerVal == 0 || takerVal == 0 {
		t.Fatal("zero amount")
	}
	impliedPrice := float64(makerVal) / float64(takerVal)
	if impliedPrice > 0.3333333 {
		t.Errorf("BUY implied price = %f, must be <= 0.3333333 (limit price)", impliedPrice)
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

func TestBuildOrder_NegRiskSignatureDiverges(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	args := OrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Size:    "10",
		Side:    Buy,
	}

	orderNormal, err := b.BuildOrder(args, CreateOrderOptions{NegRisk: false})
	if err != nil {
		t.Fatal(err)
	}

	orderNegRisk, err := b.BuildOrder(args, CreateOrderOptions{NegRisk: true})
	if err != nil {
		t.Fatal(err)
	}

	if orderNormal.Signature == orderNegRisk.Signature {
		t.Fatal("signatures with NegRisk=false and NegRisk=true must differ")
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
	}, CreateOrderOptions{}, GTC, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
	if gotReq.OrderType != GTC {
		t.Errorf("posted orderType = %s, want GTC", gotReq.OrderType)
	}
	if gotReq.DeferExec != nil {
		t.Errorf("deferExec should be nil when not set, got %v", *gotReq.DeferExec)
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
	}, CreateOrderOptions{}, FOK, nil)
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
	}, CreateOrderOptions{}, GTC, nil)
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
	}, CreateOrderOptions{}, GTD, nil)
	if err == nil {
		t.Fatal("expected error for GTD market order")
	}
}

func TestCreateAndPostMarketOrder_RejectDeferExec(t *testing.T) {
	signer := testKey()
	client := NewClient("", WithSigner(signer))
	b := NewOrderBuilder(client)

	yes := true
	_, err := b.CreateAndPostMarketOrder(context.Background(), MarketOrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Amount:  "100",
		Side:    Buy,
	}, CreateOrderOptions{}, FOK, &yes)
	if err == nil {
		t.Fatal("expected error for deferExec=true + FOK")
	}

	_, err = b.CreateAndPostMarketOrder(context.Background(), MarketOrderArgsV2{
		TokenID: "123456",
		Price:   "0.50",
		Amount:  "100",
		Side:    Buy,
	}, CreateOrderOptions{}, FAK, &yes)
	if err == nil {
		t.Fatal("expected error for deferExec=true + FAK")
	}
}

func TestValidatePriceRange(t *testing.T) {
	for _, tt := range []struct {
		price    string
		allowOne bool
		wantErr  bool
	}{
		{"0.50", false, false},
		{"0.99", false, false},
		{"0.0001", false, false},
		{"0", false, true},
		{"-0.1", false, true},
		{"1.0", false, true},
		{"1.2", false, true},
		{"1.0", true, false},
		{"1.1", true, true},
		{"not-a-number", false, true},
		{"", false, true},
	} {
		t.Run("limit_"+tt.price, func(t *testing.T) {
			err := validatePriceRange(tt.price, tt.allowOne)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePriceRange(%q, %v) err = %v, wantErr = %v", tt.price, tt.allowOne, err, tt.wantErr)
			}
		})
	}
}

func TestComputeMarketOrderAmounts_CeilProtectsWorstPrice_BUY(t *testing.T) {
	maker, taker, err := computeMarketOrderAmounts("0.333333", "100", Buy)
	if err != nil {
		t.Fatal(err)
	}
	// implied price = makerAmount / takerAmount should be <= worstPrice
	// floor takerAmount would give implied price > worstPrice
	takerVal, err := strconv.ParseInt(taker, 10, 64)
	makerVal, _ := strconv.ParseInt(maker, 10, 64)
	if takerVal == 0 {
		t.Fatal("taker is zero")
	}
	impliedPrice := float64(makerVal) / float64(takerVal)
	if impliedPrice > 0.333334 {
		t.Errorf("implied price %f exceeds worst price 0.333333", impliedPrice)
	}
}

func TestComputeMarketOrderAmounts_CeilProtectsWorstPrice_SELL(t *testing.T) {
	maker, taker, err := computeMarketOrderAmounts("0.333333", "100", Sell)
	if err != nil {
		t.Fatal(err)
	}
	makerVal, _ := strconv.ParseInt(maker, 10, 64)
	takerVal, err := strconv.ParseInt(taker, 10, 64)
	if takerVal == 0 {
		t.Fatal("taker is zero")
	}
	impliedPrice := float64(takerVal) / float64(makerVal)
	if impliedPrice < 0.333332 {
		t.Errorf("implied price %f below worst price 0.333333", impliedPrice)
	}
}
