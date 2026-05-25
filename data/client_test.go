package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTradesUsesDocumentedCoreParams(t *testing.T) {
	takerOnly := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/trades" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		q := r.URL.Query()
		if q.Get("user") != "0xuser" ||
			q.Get("market") != "0xmarket-a,0xmarket-b" ||
			q.Get("eventId") != "101,202" ||
			q.Get("limit") != "50" ||
			q.Get("offset") != "10" ||
			q.Get("takerOnly") != "true" ||
			q.Get("side") != "BUY" ||
			q.Get("filterType") != "CASH" ||
			q.Get("filterAmount") != "100" {
			t.Fatalf("trade query = %s", r.URL.RawQuery)
		}

		_, _ = w.Write([]byte(`[{
			"proxyWallet":"0xproxy",
			"user":"0xuser",
			"name":"Alice",
			"pseudonym":"alice",
			"bio":"bio",
			"profileImage":"https://example.com/p.png",
			"profileImageOptimized":"https://example.com/op.png",
			"side":"BUY",
			"asset":"123",
			"conditionId":"0xmarket-a",
			"size":10,
			"price":0.42,
			"timestamp":1714564800,
			"title":"Q",
			"slug":"q",
			"eventSlug":"e",
			"outcome":"Yes",
			"outcomeIndex":0,
			"transactionHash":"0xtx"
		}]`))
	}))
	defer srv.Close()

	client := New(Config{Host: srv.URL})
	trades, err := client.GetTrades(context.Background(), TradeParams{
		User:         "0xuser",
		Markets:      []string{"0xmarket-a", "0xmarket-b"},
		EventIDs:     []int{101, 202},
		Limit:        50,
		Offset:       10,
		TakerOnly:    &takerOnly,
		Side:         SideBuy,
		FilterType:   "CASH",
		FilterAmount: "100",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) != 1 {
		t.Fatalf("len(trades) = %d, want 1", len(trades))
	}
	if trades[0].Name != "Alice" || trades[0].ProfileImageOptimized == "" || len(trades[0].Raw) == 0 {
		t.Fatalf("unexpected trade: %+v", trades[0])
	}
}
