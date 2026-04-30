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
		{name: "data page", body: `{"data":[{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}],"next_cursor":"LTE="}`},
		{name: "orders page", body: `{"orders":[{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}],"next_cursor":"LTE="}`},
		{name: "array", body: `[{"id":"order-1","market":"0xabc","asset_id":"123","price":"0.42"}]`},
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

			client := NewClient(srv.URL, WithSigner(testKey()), WithCredentials(Credentials{
				Key:        "test-key",
				Secret:     "c2VjcmV0",
				Passphrase: "test-passphrase",
			}))
			got, err := client.GetOpenOrders(context.Background(), OpenOrderParams{Market: "0xabc"})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].ID != "order-1" || got[0].AssetID.String() != "123" {
				t.Fatalf("unexpected orders: %+v", got)
			}
		})
	}
}
