package clob

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetClobMarketInfoUsesV2EndpointAndFlexibleFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/clob-markets/0xabc" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		_, _ = w.Write([]byte(`{
			"c":"0xabc",
			"mts":"0.01",
			"mos":5,
			"nr":true,
			"fd":{"r":"0.02","e":"2","to":true},
			"t":[{"t":123,"o":"Yes"}],
			"rfqe":true
		}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	got := ClobMarketInfo{ConditionID: "0xabc"}
	if err := client.GetClobMarketInfo(context.Background(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ConditionID != "0xabc" || !got.NegRisk || !got.RFQEnabled {
		t.Fatalf("unexpected market info: %+v", got)
	}
	if got.Tokens[0].TokenID.String() != "123" || float64(got.FeeDetails.Exponent) != 2 {
		t.Fatalf("unexpected token/fee details: %+v", got)
	}
}

func TestCLOBAdditionalMarketDataEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/batch-prices-history":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"history":{"token-1":[{"t":"1713398400","p":"0.42"}]}}`))
		case "/rebates/current":
			if r.URL.Query().Get("date") != "2026-02-27" || r.URL.Query().Get("maker_address") != "0xmaker" {
				t.Fatalf("rebate query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[{"date":"2026-02-27","condition_id":"0xcond","asset_address":"0xasset","maker_address":"0xmaker","rebated_fees_usdc":"0.237519"}]`))
		case "/fee-rate/token-1":
			_, _ = w.Write([]byte(`{"base_fee":30}`))
		case "/tick-size/token-1":
			_, _ = w.Write([]byte(`{"minimum_tick_size":0.01}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	var history BatchPriceHistoryResponse
	if err := client.GetBatchPricesHistory(context.Background(), BatchPriceHistoryParams{Markets: []string{"token-1"}}, &history); err != nil {
		t.Fatal(err)
	}
	if len(history.History["token-1"]) != 1 || float64(history.History["token-1"][0].P) != 0.42 {
		t.Fatalf("unexpected history: %+v", history)
	}
	rebates, err := client.GetCurrentRebates(context.Background(), RebateParams{Date: "2026-02-27", MakerAddress: "0xmaker"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rebates) != 1 || float64(rebates[0].RebatedFeesUSDC) != 0.237519 {
		t.Fatalf("unexpected rebates: %+v", rebates)
	}
	var fee FeeRateResponse
	if err := client.GetFeeRateByTokenID(context.Background(), "token-1", &fee); err != nil || int(fee.BaseFee) != 30 {
		t.Fatalf("fee=%+v err=%v", fee, err)
	}
	var tick TickSizeResponse
	if err := client.GetTickSizeByTokenID(context.Background(), "token-1", &tick); err != nil || tick.MinimumTickSize != TickSizeHundredth {
		t.Fatalf("tick=%+v err=%v", tick, err)
	}
}

func TestGetOpenOrdersAcceptsPagedAndArrayResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "data page",
			body: `{"data":[{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}],"next_cursor":"LTE="}`,
		},
		{
			name: "orders page",
			body: `{"orders":[{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}],"next_cursor":"LTE="}`,
		},
		{
			name: "array",
			body: `[{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/data/orders" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				if r.URL.Query().Get("market") != "0xabc" {
					t.Fatalf("market query = %s", r.URL.RawQuery)
				}

				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := NewClient(srv.URL,
				WithSigner(testKey()),
				WithCredentials(Credentials{
					Key:        "test-key",
					Secret:     "c2VjcmV0",
					Passphrase: "test-passphrase",
				}),
			)

			got, err := client.GetOpenOrders(context.Background(), OpenOrderParams{Market: "0xabc"})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 {
				t.Fatalf("len(got) = %d, want 1", len(got))
			}
			if got[0].ID != "order-1" {
				t.Fatalf("ID = %q, want %q", got[0].ID, "order-1")
			}
			if got[0].AssetID.String() != "123" {
				t.Fatalf("AssetID = %q, want %q", got[0].AssetID.String(), "123")
			}
		})
	}
}

func TestGetOpenOrdersPageDecodesPaginationMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/data/orders" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("market") != "0xabc" {
			t.Fatalf("market query = %s", r.URL.RawQuery)
		}

		_, _ = w.Write([]byte(`{
			"data": [
				{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}
			],
			"limit": 100,
			"count": 1,
			"next_cursor": "LTE="
		}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
	)

	page, err := client.GetOpenOrdersPage(context.Background(), OpenOrderParams{Market: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	if page == nil {
		t.Fatal("page is nil")
	}

	if page.Limit != 100 {
		t.Fatalf("Limit = %d, want 100", page.Limit)
	}
	if page.Count != 1 {
		t.Fatalf("Count = %d, want 1", page.Count)
	}
	if page.NextCursor != "LTE=" {
		t.Fatalf("NextCursor = %q, want %q", page.NextCursor, "LTE=")
	}

	if len(page.Data) != 1 {
		t.Fatalf("len(page.Data) = %d, want 1", len(page.Data))
	}
	if page.Data[0].ID != "order-1" {
		t.Fatalf("ID = %q, want %q", page.Data[0].ID, "order-1")
	}
	if page.Data[0].Market != "0xabc" {
		t.Fatalf("Market = %q, want %q", page.Data[0].Market, "0xabc")
	}
	if page.Data[0].AssetID.String() != "123" {
		t.Fatalf("AssetID = %q, want %q", page.Data[0].AssetID.String(), "123")
	}
	if page.Data[0].Price != 0.42 {
		t.Fatalf("Price = %v, want %q", page.Data[0].Price, "0.42")
	}
}

func TestGetOpenOrdersPageDecodesPaginationMetadataFromWrappedShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "data wrapper",
			body: `{
				"data": [{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}],
				"limit": 100,
				"count": 1,
				"next_cursor": "data-cursor"
			}`,
		},
		{
			name: "orders wrapper",
			body: `{
				"orders": [{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}],
				"limit": 50,
				"count": 1,
				"next_cursor": "orders-cursor"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/data/orders" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := NewClient(srv.URL,
				WithSigner(testKey()),
				WithCredentials(Credentials{
					Key:        "test-key",
					Secret:     "c2VjcmV0",
					Passphrase: "test-passphrase",
				}),
			)

			page, err := client.GetOpenOrdersPage(context.Background(), OpenOrderParams{})
			if err != nil {
				t.Fatal(err)
			}
			if page == nil {
				t.Fatal("page is nil")
			}
			if len(page.Data) != 1 {
				t.Fatalf("len(page.Data) = %d, want 1", len(page.Data))
			}
			if page.Count != 1 {
				t.Fatalf("Count = %d, want 1", page.Count)
			}
			if page.NextCursor == "" {
				t.Fatal("NextCursor is empty")
			}

			switch tt.name {
			case "data wrapper":
				if page.Limit != 100 {
					t.Fatalf("Limit = %d, want 100", page.Limit)
				}
				if page.NextCursor != "data-cursor" {
					t.Fatalf("NextCursor = %q, want data-cursor", page.NextCursor)
				}
			case "orders wrapper":
				if page.Limit != 50 {
					t.Fatalf("Limit = %d, want 50", page.Limit)
				}
				if page.NextCursor != "orders-cursor" {
					t.Fatalf("NextCursor = %q, want orders-cursor", page.NextCursor)
				}
			}
		})
	}
}

func TestGetOpenOrdersPageSendsNextCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/data/orders" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("market") != "0xabc" {
			t.Fatalf("market query = %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("next_cursor") != "LTE=" {
			t.Fatalf("next_cursor query = %q, want %q", r.URL.Query().Get("next_cursor"), "LTE=")
		}

		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
	)

	page, err := client.GetOpenOrdersPage(context.Background(), OpenOrderParams{
		Market:     "0xabc",
		NextCursor: "LTE=",
	})
	if err != nil {
		t.Fatal(err)
	}
	if page == nil {
		t.Fatal("page is nil")
	}
	if len(page.Data) != 0 {
		t.Fatalf("len(page.Data) = %d, want 0", len(page.Data))
	}
}

func TestBuildOrder_UsesDefaultSignatureType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/data/orders" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("market") != "0xabc" {
			t.Fatalf("market query = %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("next_cursor") != "LTE=" {
			t.Fatalf("next_cursor query = %q, want %q", r.URL.Query().Get("next_cursor"), "LTE=")
		}

		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	client := NewClient(srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
		WithDefaultSignatureType(SignatureTypeProxy),
	)
	builder := NewOrderBuilder(client)

	order, err := builder.BuildOrder(OrderArgsV2{
		TokenID: "123",
		Price:   "0.50",
		Size:    "10",
		Side:    Buy,
		// SignatureType intentionally nil.
	}, CreateOrderOptions{
		TickSize: "0.01",
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.SignatureType != SignatureTypeProxy {
		t.Fatalf("SignatureType = %d, want %d", order.SignatureType, SignatureTypeProxy)
	}
}

func TestBuildOrder_ExplicitSignatureTypeOverridesDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/data/orders" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("market") != "0xabc" {
			t.Fatalf("market query = %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("next_cursor") != "LTE=" {
			t.Fatalf("next_cursor query = %q, want %q", r.URL.Query().Get("next_cursor"), "LTE=")
		}

		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	client := NewClient(srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
		WithDefaultSignatureType(SignatureTypeProxy),
	)
	builder := NewOrderBuilder(client)

	order, err := builder.BuildOrder(OrderArgsV2{
		TokenID:       "123",
		Price:         "0.50",
		Size:          "10",
		Side:          Buy,
		SignatureType: new(SignatureTypeEOA),
	}, CreateOrderOptions{
		TickSize: "0.01",
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.SignatureType != SignatureTypeEOA {
		t.Fatalf("SignatureType = %d, want %d", order.SignatureType, SignatureTypeEOA)
	}
}

func TestBuildMarketOrder_UsesDefaultSignatureType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/data/orders" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("market") != "0xabc" {
			t.Fatalf("market query = %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("next_cursor") != "LTE=" {
			t.Fatalf("next_cursor query = %q, want %q", r.URL.Query().Get("next_cursor"), "LTE=")
		}

		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	client := NewClient(srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
		WithDefaultSignatureType(SignatureTypeProxy),
	)
	builder := NewOrderBuilder(client)

	order, err := builder.BuildMarketOrder(MarketOrderArgsV2{
		TokenID: "123",
		Price:   "0.50",
		Amount:  "100",
		Side:    Buy,
		// SignatureType intentionally nil.
	}, CreateOrderOptions{
		TickSize: "0.01",
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.SignatureType != SignatureTypeProxy {
		t.Fatalf("SignatureType = %d, want %d", order.SignatureType, SignatureTypeProxy)
	}
}

func TestBuildMarketOrder_ExplicitSignatureTypeOverridesDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/data/orders" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("market") != "0xabc" {
			t.Fatalf("market query = %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("next_cursor") != "LTE=" {
			t.Fatalf("next_cursor query = %q, want %q", r.URL.Query().Get("next_cursor"), "LTE=")
		}

		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	client := NewClient(srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
		WithDefaultSignatureType(SignatureTypeProxy),
	)
	builder := NewOrderBuilder(client)

	order, err := builder.BuildMarketOrder(MarketOrderArgsV2{
		TokenID:       "123",
		Price:         "0.50",
		Amount:        "100",
		Side:          Buy,
		SignatureType: new(SignatureTypeEOA),
	}, CreateOrderOptions{
		TickSize: "0.01",
	})
	if err != nil {
		t.Fatal(err)
	}

	if order.SignatureType != SignatureTypeEOA {
		t.Fatalf("SignatureType = %d, want %d", order.SignatureType, SignatureTypeEOA)
	}
}
