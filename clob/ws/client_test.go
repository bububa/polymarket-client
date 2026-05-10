package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/bububa/polymarket-client/shared"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestClientSubscribesAndDecodesEvents(t *testing.T) {
	gotSub := make(chan MarketSubscription, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		var sub MarketSubscription
		if err := wsjson.Read(ctx, conn, &sub); err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}
		gotSub <- sub

		event := BookEvent{
			BaseEvent: BaseEvent{EventType: EventTypeBook},
			AssetID:   "asset-1",
			Bids:      []clob.OrderSummary{{Price: clob.Float64(0.45), Size: clob.Float64(10)}},
			Asks:      []clob.OrderSummary{{Price: clob.Float64(0.55), Size: clob.Float64(20)}},
			Timestamp: shared.TimeFromUnixMilli(1700000000000),
		}
		if err := wsjson.Write(ctx, conn, []BookEvent{event}); err != nil {
			t.Errorf("write event: %v", err)
		}

		// Keep connection alive until test completes
		<-ctx.Done()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(
		WithHost(url),
		WithAutoReconnect(false),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case sub := <-gotSub:
		if sub.Type != ChannelMarket || len(sub.AssetIDs) != 1 || sub.AssetIDs[0] != "asset-1" || !sub.InitialDump {
			t.Fatalf("unexpected subscription: %#v", sub)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for subscription")
	}

	select {
	case raw := <-client.Events():
		event, ok := raw.(*BookEvent)
		if !ok {
			t.Fatalf("event type = %T, want *BookEvent", raw)
		}
		if event.AssetID != "asset-1" || len(event.Bids) != 1 || len(event.Asks) != 1 {
			t.Fatalf("unexpected event: %#v", event)
		}
	case err := <-client.Errors():
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for event")
	}
}

func TestUserSubscriptionRequiresCredentials(t *testing.T) {
	client := New(WithAutoReconnect(false))
	err := client.SubscribeOrders(context.Background(), []string{"condition"})
	if err == nil {
		t.Fatal("expected missing credentials error")
	}
}

func TestClientSendsTextKeepAlive(t *testing.T) {
	gotPing := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			t.Errorf("read ping: %v", err)
			return
		}
		if msgType != websocket.MessageText || string(data) != "PING" {
			t.Errorf("message = %v %q, want text PING", msgType, data)
			return
		}
		gotPing <- struct{}{}
		if err := conn.Write(ctx, websocket.MessageText, []byte("PONG")); err != nil {
			t.Errorf("write pong: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(
		WithHost(url),
		WithAutoReconnect(false),
		WithHeartbeatInterval(20*time.Millisecond),
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	select {
	case <-gotPing:
	case <-ctx.Done():
		t.Fatal("timed out waiting for PING")
	}

	select {
	case ev := <-client.Events():
		t.Fatalf("unexpected event from PONG: %#v", ev)
	case err := <-client.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestStaleDetectionForcesReconnectAndReplaysSubscriptions(t *testing.T) {
	gotInitialSub := make(chan MarketSubscription, 1)
	gotReplaySub := make(chan MarketSubscription, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		var sub MarketSubscription
		if err := wsjson.Read(ctx, conn, &sub); err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}

		if connCount.Add(1) == 1 {
			gotInitialSub <- sub
			time.Sleep(200 * time.Millisecond)
			return
		}

		gotReplaySub <- sub
		event := BookEvent{
			BaseEvent: BaseEvent{EventType: EventTypeBook},
			AssetID:   "asset-1",
			Bids:      []clob.OrderSummary{{Price: clob.Float64(0.45), Size: clob.Float64(10)}},
			Asks:      []clob.OrderSummary{{Price: clob.Float64(0.55), Size: clob.Float64(20)}},
			Timestamp: shared.TimeFromUnixMilli(1700000000000),
		}
		if err := wsjson.Write(ctx, conn, []BookEvent{event}); err != nil {
			t.Errorf("write event: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(
		WithHost(url),
		WithHeartbeatInterval(0),
		WithStaleTimeout(40*time.Millisecond),
		WithStaleCheckInterval(10*time.Millisecond),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case sub := <-gotInitialSub:
		if len(sub.AssetIDs) != 1 || sub.AssetIDs[0] != "asset-1" {
			t.Fatalf("unexpected initial subscription: %#v", sub)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for initial subscription")
	}

	select {
	case sub := <-gotReplaySub:
		if len(sub.AssetIDs) != 1 || sub.AssetIDs[0] != "asset-1" {
			t.Fatalf("unexpected replay subscription: %#v", sub)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for replay subscription")
	}

	for {
		select {
		case raw := <-client.Events():
			event, ok := raw.(*BookEvent)
			if !ok {
				t.Fatalf("event type = %T, want *BookEvent", raw)
			}
			if event.AssetID != "asset-1" {
				t.Fatalf("AssetID = %q, want asset-1", event.AssetID)
			}
			return
		case <-client.Errors():
		case <-ctx.Done():
			t.Fatal("timed out waiting for replayed event")
		}
	}
}

func TestMarketSubscriptionUsesAssetsIDsWireField(t *testing.T) {
	payload, err := json.Marshal(MarketSubscription{
		Type:        ChannelMarket,
		AssetIDs:    []string{"asset-1"},
		InitialDump: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["assets_ids"]; !ok {
		t.Fatalf("payload missing assets_ids: %s", payload)
	}
	if _, ok := raw["asset_ids"]; ok {
		t.Fatalf("payload should not use asset_ids: %s", payload)
	}
}

func TestDecodeNewMarketAcceptsAssetIDsVariant(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"event_type":"new_market","id":"m1","asset_ids":["asset-1"]}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*NewMarketEvent)
	if len(got.AssetIDs) != 1 || got.AssetIDs[0] != "asset-1" {
		t.Fatalf("AssetIDs = %#v", got.AssetIDs)
	}
}

func TestDecodeMarketResolvedAcceptsAssetIDsVariant(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"event_type":"market_resolved","id":"m1","asset_ids":["asset-1"],"winning_asset_id":"asset-1"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*MarketResolvedEvent)
	if len(got.AssetIDs) != 1 || got.AssetIDs[0] != "asset-1" {
		t.Fatalf("AssetIDs = %#v", got.AssetIDs)
	}
}

func TestDecodeOrderUsesDocumentedIDField(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"event_type":"order","id":"order-1","asset_id":"asset-1","market":"0xabc","price":"0.42","size":"10","side":"BUY","status":"LIVE"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*OrderEvent)
	if got.OrderID != "order-1" {
		t.Fatalf("OrderID = %q, want order-1", got.OrderID)
	}
}

func TestDecodeOrderAcceptsOrderIDCompat(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"event_type":"order","order_id":"order-1","asset_id":"asset-1","market":"0xabc","price":"0.42","size":"10","side":"BUY","status":"LIVE"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*OrderEvent)
	if got.OrderID != "order-1" {
		t.Fatalf("OrderID = %q, want order-1", got.OrderID)
	}
}

func TestMarshalOrderUsesDocumentedIDField(t *testing.T) {
	payload, err := json.Marshal(OrderEvent{OrderID: "order-1"})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["id"] != "order-1" {
		t.Fatalf("id = %v, want order-1", raw["id"])
	}
	if _, ok := raw["order_id"]; ok {
		t.Fatalf("payload should not include order_id: %s", payload)
	}
}

func TestDecodePriceChangeBatch(t *testing.T) {
	events := decodeEvents([]byte(`{
		"event_type": "price_change",
		"market": "0xabc",
		"timestamp": "1700000000000",
		"price_changes": [
			{
				"asset_id": "asset-1",
				"price": "0.42",
				"size": "10",
				"side": "BUY",
				"best_bid": "0.41",
				"best_ask": "0.43"
			},
			{
				"asset_id": "asset-2",
				"price": "0.58",
				"size": "20",
				"side": "SELL"
			}
		]
	}`))
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}

	first, ok := events[0].event.(*PriceChangeEvent)
	if !ok {
		t.Fatalf("event type = %T, want *PriceChangeEvent", events[0].event)
	}
	if first.EventType != EventTypePriceChange {
		t.Fatalf("first.EventType = %q, want %q", first.EventType, EventTypePriceChange)
	}
	if first.Market != "0xabc" {
		t.Fatalf("first.Market = %q, want %q", first.Market, "0xabc")
	}
	if first.Timestamp.Time().UnixMilli() != 1700000000000 {
		t.Fatalf("first.Timestamp = %v, want %q", first.Timestamp, "1700000000000")
	}
	if first.AssetID != "asset-1" ||
		first.Price != 0.42 ||
		first.Size != 10 ||
		first.Side != clob.Buy ||
		first.BestBid != 0.41 ||
		first.BestAsk != 0.43 {
		t.Fatalf("unexpected first price change: %+v", first)
	}

	second, ok := events[1].event.(*PriceChangeEvent)
	if !ok {
		t.Fatalf("event type = %T, want *PriceChangeEvent", events[1].event)
	}
	if second.EventType != EventTypePriceChange {
		t.Fatalf("second.EventType = %q, want %q", second.EventType, EventTypePriceChange)
	}
	if second.Market != "0xabc" {
		t.Fatalf("second.Market = %q, want %q", second.Market, "0xabc")
	}
	if second.Timestamp.Time().UnixMilli() != 1700000000000 {
		t.Fatalf("second.Timestamp = %v, want %q", second.Timestamp, "1700000000000")
	}
	if second.AssetID != "asset-2" ||
		second.Price != 0.58 ||
		second.Size != 20 ||
		second.Side != clob.Sell {
		t.Fatalf("unexpected second price change: %+v", second)
	}
}

func TestDecodePriceChangeBatchKeepsChildMarketAndTimestamp(t *testing.T) {
	events := decodeEvents([]byte(`{
		"event_type": "price_change",
		"market": "0xbatch",
		"timestamp": "1700000000000",
		"price_changes": [
			{
				"asset_id": "asset-1",
				"market": "0xchild",
				"timestamp": "1700000000001",
				"price": "0.42",
				"size": "10",
				"side": "BUY"
			}
		]
	}`))
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	got, ok := events[0].event.(*PriceChangeEvent)
	if !ok {
		t.Fatalf("event type = %T, want *PriceChangeEvent", events[0].event)
	}

	if got.Market != "0xchild" {
		t.Fatalf("Market = %q, want %q", got.Market, "0xchild")
	}
	if got.Timestamp.Time().UnixMilli() != 1700000000001 {
		t.Fatalf("Timestamp = %v, want %q", got.Timestamp, "1700000000001")
	}
}

func TestDecodeEventArrayError(t *testing.T) {
	events := decodeEvents([]byte(`[{"event_type":"unknown"}]`))
	if len(events) != 1 || events[0].err == nil {
		data, _ := json.Marshal(events)
		t.Fatalf("expected decode error, got %s", data)
	}
}

func TestClientRespondsToServerPingWithPong(t *testing.T) {
	gotPong := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		if err := conn.Write(ctx, websocket.MessageText, []byte("ping")); err != nil {
			t.Errorf("write ping: %v", err)
			return
		}

		readCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		msgType, data, err := conn.Read(readCtx)
		if err != nil {
			t.Errorf("read pong: %v", err)
			return
		}
		if msgType != websocket.MessageText || !bytes.EqualFold(data, []byte("pong")) {
			t.Errorf("message = %v %q, want text pong", msgType, data)
			return
		}

		gotPong <- struct{}{}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(
		WithSportsHost(url),
		WithAutoReconnect(false),
		WithHeartbeatInterval(0), // isolate server-ping -> client-pong behavior
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.ConnectSports(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	select {
	case <-gotPong:
	case <-ctx.Done():
		t.Fatal("timed out waiting for pong")
	}

	select {
	case ev := <-client.Events():
		t.Fatalf("unexpected event from ping/pong frame: %#v", ev)
	case err := <-client.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
}
