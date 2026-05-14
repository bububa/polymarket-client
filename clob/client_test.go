package clob

import (
	"context"
	"encoding/json"
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

			client := NewClient(
				srv.URL,
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

	client := NewClient(
		srv.URL,
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

			client := NewClient(
				srv.URL,
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

	client := NewClient(
		srv.URL,
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
	client := NewClient(
		srv.URL,
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
	client := NewClient(
		srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
		WithDefaultSignatureType(SignatureTypeProxy),
	)
	builder := NewOrderBuilder(client)
	SignatureType := SignatureTypeEOA
	order, err := builder.BuildOrder(OrderArgsV2{
		TokenID:       "123",
		Price:         "0.50",
		Size:          "10",
		Side:          Buy,
		SignatureType: &SignatureType,
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
	client := NewClient(
		srv.URL,
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
	client := NewClient(
		srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
		WithDefaultSignatureType(SignatureTypeProxy),
	)
	builder := NewOrderBuilder(client)
	SignatureType := SignatureTypeEOA
	order, err := builder.BuildMarketOrder(MarketOrderArgsV2{
		TokenID:       "123",
		Price:         "0.50",
		Amount:        "100",
		Side:          Buy,
		SignatureType: &SignatureType,
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

func TestUpdateBalanceAllowanceSendsPoly1271SignatureType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/balance-allowance/update" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}

		var body BalanceAllowanceParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if body.AssetType != AssetCollateral {
			t.Fatalf("asset_type = %q, want %q", body.AssetType, AssetCollateral)
		}
		if body.TokenID != "" {
			t.Fatalf("token_id = %q, want empty", body.TokenID)
		}
		if body.SignatureType != SignatureTypePoly1271 {
			t.Fatalf("signature_type = %d, want %d", body.SignatureType, SignatureTypePoly1271)
		}

		_, _ = w.Write([]byte(`{"balance":"100","allowance":"100"}`))
	}))
	defer srv.Close()

	client := NewClient(
		srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
	)

	var out BalanceAllowanceResponse
	if err := client.UpdateBalanceAllowance(context.Background(), BalanceAllowanceParams{
		AssetType:     AssetCollateral,
		SignatureType: SignatureTypePoly1271,
	}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateBalanceAllowanceOmitsDefaultSignatureType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/balance-allowance/update" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}

		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if raw["asset_type"] != string(AssetCollateral) {
			t.Fatalf("asset_type = %v, want %q", raw["asset_type"], AssetCollateral)
		}
		if _, ok := raw["signature_type"]; ok {
			t.Fatalf("signature_type should be omitted for default EOA, got %v", raw["signature_type"])
		}

		_, _ = w.Write([]byte(`{"balance":"100","allowance":"100"}`))
	}))
	defer srv.Close()

	client := NewClient(
		srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
	)

	var out BalanceAllowanceResponse
	if err := client.UpdateBalanceAllowance(context.Background(), BalanceAllowanceParams{
		AssetType: AssetCollateral,
	}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestRewardsUserEndpointsIncludeOfficialParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Query().Get("signature_type") != "2" {
			t.Fatalf("signature_type query = %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("maker_address") != "0xmaker" {
			t.Fatalf("maker_address query = %s", r.URL.RawQuery)
		}

		switch r.URL.Path {
		case "/rewards/user":
			if r.URL.Query().Get("date") != "2026-05-13" || r.URL.Query().Get("next_cursor") != "abc" {
				t.Fatalf("user rewards query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"date":"2026-05-13 00:00:00+00","condition_id":"0xcond","asset_address":"0xasset","maker_address":"0xmaker","earnings":"0.12","asset_rate":"1"}],"next_cursor":"LTE="}`))
		case "/rewards/user/total":
			if r.URL.Query().Get("date") != "2026-05-13" {
				t.Fatalf("total rewards query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[{"date":"2026-05-13 00:00:00+00","asset_address":"0xasset","maker_address":"0xmaker","earnings":"0.32","asset_rate":"1"}]`))
		case "/rewards/user/percentages":
			_, _ = w.Write([]byte(`{"0xcond":"0.38"}`))
		case "/rewards/user/markets":
			if r.URL.Query().Get("order_by") != "earning_percentage" || r.URL.Query().Get("page_size") != "500" {
				t.Fatalf("markets query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"condition_id":"0xcond","question":"Q","market_slug":"m","event_slug":"e","rewards_max_spread":"0.035","rewards_min_size":"100","market_competitiveness":"4.2","spread":"0.01","volume_24hr":"1234","tokens":[{"token_id":"1","outcome":"YES","price":"0.5"}],"rewards_config":[{"asset_address":"0xasset","start_date":"2026-05-13","end_date":"2026-05-14","rate_per_day":"300","total_rewards":"300","total_days":"1"}],"maker_address":"0xmaker","earning_percentage":"0.38","earnings":[{"asset_address":"0xasset","earnings":"0.12","asset_rate":"1"}]}],"next_cursor":"LTE="}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(
		srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
	)

	var user Page[UserEarning]
	if err := client.GetEarningsForUserForDayWithParams(context.Background(), UserRewardsParams{
		Date:          "2026-05-13",
		SignatureType: SignatureTypeGnosisSafe,
		MakerAddress:  "0xmaker",
		NextCursor:    "abc",
	}, &user); err != nil {
		t.Fatal(err)
	}
	if len(user.Data) != 1 || user.Data[0].ConditionID != "0xcond" {
		t.Fatalf("unexpected user rewards: %+v", user)
	}

	var total []TotalUserEarning
	if err := client.GetTotalEarningsForUserForDayWithParams(context.Background(), UserRewardsParams{
		Date:          "2026-05-13",
		SignatureType: SignatureTypeGnosisSafe,
		MakerAddress:  "0xmaker",
	}, &total); err != nil {
		t.Fatal(err)
	}
	if len(total) != 1 || float64(total[0].Earnings) != 0.32 {
		t.Fatalf("unexpected total rewards: %+v", total)
	}

	pct, err := client.GetRewardPercentagesWithParams(context.Background(), UserRewardsParams{
		SignatureType: SignatureTypeGnosisSafe,
		MakerAddress:  "0xmaker",
	})
	if err != nil {
		t.Fatal(err)
	}
	if float64(pct["0xcond"]) != 0.38 {
		t.Fatalf("unexpected percentages: %+v", pct)
	}

	var markets Page[UserRewardsEarning]
	if err := client.GetUserEarningsAndMarketsConfig(context.Background(), EarningsParams{
		OrderBy:      "earning_percentage",
		PageSize:     500,
		MakerAddress: "0xmaker",
	}, SignatureTypeGnosisSafe, &markets); err != nil {
		t.Fatal(err)
	}
	if len(markets.Data) != 1 || float64(markets.Data[0].Spread) != 0.01 || float64(markets.Data[0].Volume24h) != 1234 {
		t.Fatalf("unexpected markets: %+v", markets)
	}
}

func TestRewardsMarketsMultiUsesOfficialParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rewards/markets/multi" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		q := r.URL.Query()
		if q.Get("order_by") != "rate_per_day" ||
			q.Get("position") != "DESC" ||
			q.Get("page_size") != "50" ||
			q.Get("max_spread") != "0.1" ||
			q.Get("min_price") != "0.1" ||
			q.Get("max_price") != "0.9" ||
			q.Get("min_volume_24hr") != "1000" ||
			q.Get("next_cursor") != "abc" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"data":[{"condition_id":"0xcond","question":"Q","market_slug":"m","event_slug":"e","rewards_max_spread":"0.035","rewards_min_size":"100","market_competitiveness":"4.2","neg_risk":true,"spread":"0.01","volume_24hr":"1234","end_date":"2026-05-14 00:00:00+00","sponsored_daily_rate":"200","native_daily_rate":"100","total_daily_rate":"300","sponsors_count":2,"tokens":[{"token_id":"1","outcome":"YES","price":"0.5","winner":false}],"rewards_config":[{"asset_address":"0xasset","start_date":"2026-05-13","end_date":"2026-05-14","rate_per_day":"300","total_rewards":"300","total_days":"1"}]}],"next_cursor":"LTE="}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	var out Page[MarketReward]
	if err := client.GetRewardsMarketsMulti(context.Background(), RewardsMarketsParams{
		OrderBy:      "rate_per_day",
		Position:     "DESC",
		PageSize:     50,
		MaxSpread:    "0.1",
		MinPrice:     "0.1",
		MaxPrice:     "0.9",
		MinVolume24h: "1000",
		NextCursor:   "abc",
	}, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 ||
		out.Data[0].ConditionID != "0xcond" ||
		!out.Data[0].NegRisk ||
		float64(out.Data[0].TotalDailyRate) != 300 ||
		int(out.Data[0].SponsorsCount) != 2 ||
		float64(out.Data[0].RewardsConfig[0].TotalDays) != 1 {
		t.Fatalf("unexpected rewards markets: %+v", out)
	}
}

func TestGetTotalEarningsForUserForDayAggregatesLegacyOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rewards/user/total" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("date") != "2026-05-13" || r.URL.Query().Get("signature_type") != "2" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[
			{"date":"2026-05-13 00:00:00+00","asset_address":"0xasset1","maker_address":"0xmaker","earnings":"0.30","asset_rate":"1"},
			{"date":"2026-05-13 00:00:00+00","asset_address":"0xasset2","maker_address":"0xmaker","earnings":"2","asset_rate":"0.50"}
		]`))
	}))
	defer srv.Close()

	client := NewClient(
		srv.URL,
		WithSigner(testKey()),
		WithCredentials(Credentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0",
			Passphrase: "test-passphrase",
		}),
	)

	var out UserEarning
	if err := client.GetTotalEarningsForUserForDay(context.Background(), "2026-05-13", SignatureTypeGnosisSafe, &out); err != nil {
		t.Fatal(err)
	}
	if float64(out.Earnings) != 1.3 {
		t.Fatalf("earnings = %v, want 1.3", out.Earnings)
	}
	if out.AssetAddress != "" {
		t.Fatalf("asset_address = %q, want empty for multi-asset aggregate", out.AssetAddress)
	}
	if float64(out.AssetRate) != 1 {
		t.Fatalf("asset_rate = %v, want 1", out.AssetRate)
	}
}
