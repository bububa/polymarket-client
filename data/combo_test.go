package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetComboPositions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/positions/combos" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("market_id"); got != "0x03aa,0x03bb" {
			t.Fatalf("market_id = %q", got)
		}
		_, _ = w.Write([]byte(`{"combos":[{"combo_condition_id":"0x03aa","combo_position_id":123456789012345678901234567890,"side":"YES","module_id":"7","user_address":"0xabc","shares_balance":"1.25","status":"OPEN","redeemable":false,"first_entry_at":"2026-05-01T00:00:00Z","legs_total":"2","legs_resolved":0,"legs_pending":2,"legs":[]}],"pagination":{"limit":"20","offset":0,"has_more":true,"next_cursor":"next"}}`))
	}))
	defer server.Close()
	client := New(Config{Host: server.URL})
	page, err := client.GetComboPositions(context.Background(), ComboPositionParams{User: "0xabc", MarketIDs: []string{"0x03aa", "0x03bb"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(page.Combos[0].PositionID); got != "123456789012345678901234567890" {
		t.Fatalf("position id = %s", got)
	}
	if got := string(page.Combos[0].Shares); got != "1.25" {
		t.Fatalf("shares = %s", got)
	}
}

func TestGetComboActivityPreservesUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"activity":[{"id":1,"type":"REBALANCE","user_address":"0xabc","combo_condition_id":"0x03aa","module_id":7,"amount_usdc":1.5,"timestamp":1713398400,"tx_dttm":"2026-05-01T00:00:00Z","tx_hash":"0xhash","log_index":"2","block_number":999,"legs":[]}],"pagination":{"limit":50,"offset":0,"has_more":false,"next_cursor":null}}`))
	}))
	defer server.Close()
	page, err := New(Config{Host: server.URL}).GetComboActivity(context.Background(), ComboActivityParams{User: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	unknown := page.Activities[0].Unknown()
	if unknown == nil || unknown.Type != "REBALANCE" || len(unknown.Raw) == 0 {
		t.Fatalf("unknown = %+v", unknown)
	}
}
