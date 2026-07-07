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

func TestSubscribeOrderBookSendsDynamicSubscribeUpdate(t *testing.T) {
	gotFrame := make(chan map[string]any, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		for range 2 {
			frame, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read subscription frame: %v", err)
				return
			}
			gotFrame <- frame
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	if err := client.SubscribeOrderBook(ctx, []string{"asset-b"}); err != nil {
		t.Fatal(err)
	}

	initial := receiveMarketFrame(t, ctx, gotFrame)
	if got := initial["type"]; got != string(ChannelMarket) {
		t.Fatalf("initial type = %v, want %q", got, ChannelMarket)
	}
	if got := initial["initial_dump"]; got != true {
		t.Fatalf("initial_dump = %v, want true", got)
	}
	assertFrameAssets(t, initial, []string{"asset-a"})

	update := receiveMarketFrame(t, ctx, gotFrame)
	if got := update["operation"]; got != "subscribe" {
		t.Fatalf("operation = %v, want subscribe", got)
	}
	if got := update["initial_dump"]; got != true {
		t.Fatalf("dynamic orderbook initial_dump = %v, want true", got)
	}
	if _, ok := update["type"]; ok {
		t.Fatalf("dynamic subscribe update should omit type: %#v", update)
	}
	assertFrameAssets(t, update, []string{"asset-b"})
}

func TestSubscribeOrderBookCanDisableInitialDump(t *testing.T) {
	gotFrame := make(chan map[string]any, 2)
	server := newMarketFrameServer(t, gotFrame)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(
		WithHost(url),
		WithAutoReconnect(false),
		WithOrderBookInitialDump(false),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	if err := client.SubscribeOrderBook(ctx, []string{"asset-b"}); err != nil {
		t.Fatal(err)
	}

	initial := receiveMarketFrame(t, ctx, gotFrame)
	if got, ok := initial["initial_dump"]; !ok || got != false {
		t.Fatalf("initial_dump = %v, present = %v, want explicit false", got, ok)
	}
	assertFrameAssets(t, initial, []string{"asset-a"})

	update := receiveMarketFrame(t, ctx, gotFrame)
	if got, ok := update["initial_dump"]; !ok || got != false {
		t.Fatalf("dynamic initial_dump = %v, present = %v, want explicit false", got, ok)
	}
	assertFrameAssets(t, update, []string{"asset-b"})
}

func TestUnsubscribeOrderBookSendsUnsubscribeUpdate(t *testing.T) {
	gotFrame := make(chan map[string]any, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		for range 2 {
			frame, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read subscription frame: %v", err)
				return
			}
			gotFrame <- frame
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UnsubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}

	_ = receiveMarketFrame(t, ctx, gotFrame)
	update := receiveMarketFrame(t, ctx, gotFrame)
	if got := update["operation"]; got != "unsubscribe" {
		t.Fatalf("operation = %v, want unsubscribe", got)
	}
	if _, ok := update["type"]; ok {
		t.Fatalf("unsubscribe update should omit type: %#v", update)
	}
	assertFrameAssets(t, update, []string{"asset-a"})
}

func TestOrderBookPartialUnsubscribeUpdatesReplayState(t *testing.T) {
	gotInitial := make(chan map[string]any, 1)
	gotUnsubscribe := make(chan map[string]any, 1)
	gotReplay := make(chan map[string]any, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}

		ctx := context.Background()
		if connCount.Add(1) == 1 {
			initial, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read initial subscription: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotInitial <- initial
			unsub, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read unsubscribe update: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotUnsubscribe <- unsub
			_ = conn.CloseNow()
			return
		}

		replay, err := readMarketFrame(ctx, conn)
		if err != nil {
			t.Errorf("read replay subscription: %v", err)
			_ = conn.CloseNow()
			return
		}
		gotReplay <- replay
		<-ctx.Done()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0))
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a", "asset-b"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveMarketFrame(t, ctx, gotInitial)
	assertFrameAssets(t, initial, []string{"asset-a", "asset-b"})

	if err := client.UnsubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	unsub := receiveMarketFrame(t, ctx, gotUnsubscribe)
	if got := unsub["operation"]; got != "unsubscribe" {
		t.Fatalf("operation = %v, want unsubscribe", got)
	}
	assertFrameAssets(t, unsub, []string{"asset-a"})

	replay := receiveMarketFrame(t, ctx, gotReplay)
	if got := replay["type"]; got != string(ChannelMarket) {
		t.Fatalf("replay type = %v, want %q", got, ChannelMarket)
	}
	if _, ok := replay["operation"]; ok {
		t.Fatalf("replay should be an initial subscription, got: %#v", replay)
	}
	assertFrameAssets(t, replay, []string{"asset-b"})
}

func TestOrderBookDynamicSubscribeUpdatesReplayState(t *testing.T) {
	gotInitial := make(chan map[string]any, 1)
	gotSubscribe := make(chan map[string]any, 1)
	gotUnsubscribe := make(chan map[string]any, 1)
	gotReplay := make(chan map[string]any, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}

		ctx := context.Background()
		if connCount.Add(1) == 1 {
			initial, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read initial subscription: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotInitial <- initial
			subscribe, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read subscribe update: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotSubscribe <- subscribe
			unsubscribe, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read unsubscribe update: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotUnsubscribe <- unsubscribe
			_ = conn.CloseNow()
			return
		}

		replay, err := readMarketFrame(ctx, conn)
		if err != nil {
			t.Errorf("read replay subscription: %v", err)
			_ = conn.CloseNow()
			return
		}
		gotReplay <- replay
		<-ctx.Done()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0))
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveMarketFrame(t, ctx, gotInitial)
	assertFrameAssets(t, initial, []string{"asset-a"})

	if err := client.SubscribeOrderBook(ctx, []string{"asset-b"}); err != nil {
		t.Fatal(err)
	}
	subscribe := receiveMarketFrame(t, ctx, gotSubscribe)
	if got := subscribe["operation"]; got != "subscribe" {
		t.Fatalf("operation = %v, want subscribe", got)
	}
	assertFrameAssets(t, subscribe, []string{"asset-b"})

	if err := client.UnsubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	unsubscribe := receiveMarketFrame(t, ctx, gotUnsubscribe)
	if got := unsubscribe["operation"]; got != "unsubscribe" {
		t.Fatalf("operation = %v, want unsubscribe", got)
	}
	assertFrameAssets(t, unsubscribe, []string{"asset-a"})

	replay := receiveMarketFrame(t, ctx, gotReplay)
	if got := replay["type"]; got != string(ChannelMarket) {
		t.Fatalf("replay type = %v, want %q", got, ChannelMarket)
	}
	if _, ok := replay["operation"]; ok {
		t.Fatalf("replay should be an initial subscription, got: %#v", replay)
	}
	assertFrameAssets(t, replay, []string{"asset-b"})
}

func TestOrderBookInitialDumpDisabledReplaysFalse(t *testing.T) {
	gotInitial := make(chan map[string]any, 1)
	gotReplay := make(chan map[string]any, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}

		ctx := context.Background()
		if connCount.Add(1) == 1 {
			initial, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read initial subscription: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotInitial <- initial
			_ = conn.CloseNow()
			return
		}

		replay, err := readMarketFrame(ctx, conn)
		if err != nil {
			t.Errorf("read replay subscription: %v", err)
			_ = conn.CloseNow()
			return
		}
		gotReplay <- replay
		<-ctx.Done()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(
		WithHost(url),
		WithHeartbeatInterval(0),
		WithOrderBookInitialDump(false),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveMarketFrame(t, ctx, gotInitial)
	if got, ok := initial["initial_dump"]; !ok || got != false {
		t.Fatalf("initial_dump = %v, present = %v, want explicit false", got, ok)
	}

	replay := receiveMarketFrame(t, ctx, gotReplay)
	if got, ok := replay["initial_dump"]; !ok || got != false {
		t.Fatalf("replay initial_dump = %v, present = %v, want explicit false", got, ok)
	}
	assertFrameAssets(t, replay, []string{"asset-a"})
}

func TestDuplicateOrderBookSubscribeReplaysOnce(t *testing.T) {
	gotInitial := make(chan map[string]any, 1)
	gotUnexpected := make(chan map[string]any, 1)
	gotReplay := make(chan map[string]any, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}

		ctx := context.Background()
		if connCount.Add(1) == 1 {
			initial, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read initial subscription: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotInitial <- initial

			readCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
			defer cancel()
			if frame, err := readMarketFrame(readCtx, conn); err == nil {
				gotUnexpected <- frame
			}
			_ = conn.CloseNow()
			return
		}

		replay, err := readMarketFrame(ctx, conn)
		if err != nil {
			t.Errorf("read replay subscription: %v", err)
			_ = conn.CloseNow()
			return
		}
		gotReplay <- replay
		<-ctx.Done()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0))
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveMarketFrame(t, ctx, gotInitial)
	assertFrameAssets(t, initial, []string{"asset-a"})

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}

	select {
	case frame := <-gotUnexpected:
		t.Fatalf("duplicate subscribe sent unexpected frame: %#v", frame)
	case <-time.After(250 * time.Millisecond):
	case <-ctx.Done():
		t.Fatal("timed out waiting for duplicate subscribe check")
	}

	replay := receiveMarketFrame(t, ctx, gotReplay)
	assertFrameAssets(t, replay, []string{"asset-a"})
}

func TestSubscribePricesSendsDynamicSubscribeUpdate(t *testing.T) {
	gotFrame := make(chan map[string]any, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		for range 2 {
			frame, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read subscription frame: %v", err)
				return
			}
			gotFrame <- frame
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribePrices(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	if err := client.SubscribePrices(ctx, []string{"asset-b"}); err != nil {
		t.Fatal(err)
	}

	initial := receiveMarketFrame(t, ctx, gotFrame)
	if got := initial["type"]; got != string(ChannelMarket) {
		t.Fatalf("initial type = %v, want %q", got, ChannelMarket)
	}
	if _, ok := initial["initial_dump"]; ok {
		t.Fatalf("prices initial subscription should not request initial_dump: %#v", initial)
	}
	assertFrameAssets(t, initial, []string{"asset-a"})

	update := receiveMarketFrame(t, ctx, gotFrame)
	if got := update["operation"]; got != "subscribe" {
		t.Fatalf("operation = %v, want subscribe", got)
	}
	if _, ok := update["type"]; ok {
		t.Fatalf("dynamic subscribe update should omit type: %#v", update)
	}
	assertFrameAssets(t, update, []string{"asset-b"})
}

func TestMarketSubscribeTrimsAndSkipsBlankAssetIDs(t *testing.T) {
	gotFrame := make(chan map[string]any, 1)
	server := newMarketFrameServer(t, gotFrame)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribePrices(ctx, []string{" asset-b ", "", "asset-a", " ", "asset-b"}); err != nil {
		t.Fatal(err)
	}

	initial := receiveMarketFrame(t, ctx, gotFrame)
	assertFrameAssets(t, initial, []string{"asset-a", "asset-b"})
}

func TestOrderBookThenPricesUsesExistingConnectionUpdate(t *testing.T) {
	gotFrame := make(chan map[string]any, 2)
	var connCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		connCount.Add(1)

		ctx := context.Background()
		for range 2 {
			frame, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read subscription frame: %v", err)
				return
			}
			gotFrame <- frame
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	if err := client.SubscribePrices(ctx, []string{"asset-b"}); err != nil {
		t.Fatal(err)
	}

	initial := receiveMarketFrame(t, ctx, gotFrame)
	assertFrameAssets(t, initial, []string{"asset-a"})
	if got := initial["initial_dump"]; got != true {
		t.Fatalf("initial_dump = %v, want true", got)
	}

	update := receiveMarketFrame(t, ctx, gotFrame)
	if got := update["operation"]; got != "subscribe" {
		t.Fatalf("operation = %v, want subscribe", got)
	}
	if _, ok := update["type"]; ok {
		t.Fatalf("dynamic subscribe update should omit type: %#v", update)
	}
	assertFrameAssets(t, update, []string{"asset-b"})
	if got := connCount.Load(); got != 1 {
		t.Fatalf("connection count = %d, want 1", got)
	}
}

func TestMarketHelperOwnershipKeepsAssetSubscribedUntilLastRef(t *testing.T) {
	gotFrame := make(chan map[string]any, 4)
	server := newMarketFrameServer(t, gotFrame)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribePrices(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveMarketFrame(t, ctx, gotFrame)
	assertFrameAssets(t, initial, []string{"asset-a"})

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	orderbookSubscribe := receiveMarketFrame(t, ctx, gotFrame)
	if got := orderbookSubscribe["operation"]; got != "subscribe" {
		t.Fatalf("operation = %v, want subscribe", got)
	}
	if got := orderbookSubscribe["initial_dump"]; got != true {
		t.Fatalf("orderbook upgrade initial_dump = %v, want true", got)
	}
	assertFrameAssets(t, orderbookSubscribe, []string{"asset-a"})

	if err := client.UnsubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	assertNoMarketFrame(t, gotFrame, 150*time.Millisecond)

	if err := client.UnsubscribePrices(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	unsubscribe := receiveMarketFrame(t, ctx, gotFrame)
	if got := unsubscribe["operation"]; got != "unsubscribe" {
		t.Fatalf("operation = %v, want unsubscribe", got)
	}
	assertFrameAssets(t, unsubscribe, []string{"asset-a"})
}

func TestCustomHelperEnablesCustomFeatureForExistingAsset(t *testing.T) {
	gotFrame := make(chan map[string]any, 3)
	server := newMarketFrameServer(t, gotFrame)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribePrices(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveMarketFrame(t, ctx, gotFrame)
	assertFrameAssets(t, initial, []string{"asset-a"})
	if _, ok := initial["custom_feature_enabled"]; ok {
		t.Fatalf("prices initial subscription should not enable custom feature: %#v", initial)
	}

	if err := client.SubscribeBestBidAsk(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	update := receiveMarketFrame(t, ctx, gotFrame)
	if got := update["operation"]; got != "subscribe" {
		t.Fatalf("operation = %v, want subscribe", got)
	}
	if got := update["custom_feature_enabled"]; got != true {
		t.Fatalf("custom_feature_enabled = %v, want true", got)
	}
	if _, ok := update["type"]; ok {
		t.Fatalf("custom feature update should omit type: %#v", update)
	}
	assertFrameAssets(t, update, []string{"asset-a"})
}

func TestCustomHelperInitialSubscribeEnablesCustomFeature(t *testing.T) {
	gotFrame := make(chan map[string]any, 1)
	server := newMarketFrameServer(t, gotFrame)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeBestBidAsk(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveMarketFrame(t, ctx, gotFrame)
	if got := initial["type"]; got != string(ChannelMarket) {
		t.Fatalf("initial type = %v, want %q", got, ChannelMarket)
	}
	if got := initial["custom_feature_enabled"]; got != true {
		t.Fatalf("custom_feature_enabled = %v, want true", got)
	}
	assertFrameAssets(t, initial, []string{"asset-a"})
}

func TestCustomHelperUnsubscribeKeepsNormalAssetSubscription(t *testing.T) {
	gotFrame := make(chan map[string]any, 4)
	server := newMarketFrameServer(t, gotFrame)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0), WithAutoReconnect(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribePrices(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	_ = receiveMarketFrame(t, ctx, gotFrame)
	if err := client.SubscribeBestBidAsk(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	_ = receiveMarketFrame(t, ctx, gotFrame)

	if err := client.UnsubscribeBestBidAsk(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	assertNoMarketFrame(t, gotFrame, 150*time.Millisecond)

	if err := client.UnsubscribePrices(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	unsubscribe := receiveMarketFrame(t, ctx, gotFrame)
	if got := unsubscribe["operation"]; got != "unsubscribe" {
		t.Fatalf("operation = %v, want unsubscribe", got)
	}
	assertFrameAssets(t, unsubscribe, []string{"asset-a"})
}

func TestMixedMarketHelpersReconnectReplayCanonicalState(t *testing.T) {
	gotInitial := make(chan map[string]any, 1)
	gotPriceSubscribe := make(chan map[string]any, 1)
	gotCustomSubscribe := make(chan map[string]any, 1)
	gotPriceUnsubscribe := make(chan map[string]any, 1)
	gotReplay := make(chan map[string]any, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}

		ctx := context.Background()
		if connCount.Add(1) == 1 {
			initial, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read initial subscription: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotInitial <- initial
			priceSubscribe, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read price subscribe update: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotPriceSubscribe <- priceSubscribe
			customSubscribe, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read custom subscribe update: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotCustomSubscribe <- customSubscribe
			priceUnsubscribe, err := readMarketFrame(ctx, conn)
			if err != nil {
				t.Errorf("read price unsubscribe update: %v", err)
				_ = conn.CloseNow()
				return
			}
			gotPriceUnsubscribe <- priceUnsubscribe
			_ = conn.CloseNow()
			return
		}

		replay, err := readMarketFrame(ctx, conn)
		if err != nil {
			t.Errorf("read replay subscription: %v", err)
			_ = conn.CloseNow()
			return
		}
		gotReplay <- replay
		<-ctx.Done()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(url), WithHeartbeatInterval(0))
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.ConnectMarket(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveMarketFrame(t, ctx, gotInitial)
	if got := initial["initial_dump"]; got != true {
		t.Fatalf("initial_dump = %v, want true", got)
	}
	assertFrameAssets(t, initial, []string{"asset-a"})

	if err := client.SubscribePrices(ctx, []string{"asset-b"}); err != nil {
		t.Fatal(err)
	}
	priceSubscribe := receiveMarketFrame(t, ctx, gotPriceSubscribe)
	if got := priceSubscribe["operation"]; got != "subscribe" {
		t.Fatalf("operation = %v, want subscribe", got)
	}
	assertFrameAssets(t, priceSubscribe, []string{"asset-b"})

	if err := client.SubscribeBestBidAsk(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	customSubscribe := receiveMarketFrame(t, ctx, gotCustomSubscribe)
	if got := customSubscribe["custom_feature_enabled"]; got != true {
		t.Fatalf("custom_feature_enabled = %v, want true", got)
	}
	assertFrameAssets(t, customSubscribe, []string{"asset-a"})

	if err := client.UnsubscribeOrderBook(ctx, []string{"asset-a"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UnsubscribePrices(ctx, []string{"asset-b"}); err != nil {
		t.Fatal(err)
	}
	priceUnsubscribe := receiveMarketFrame(t, ctx, gotPriceUnsubscribe)
	if got := priceUnsubscribe["operation"]; got != "unsubscribe" {
		t.Fatalf("operation = %v, want unsubscribe", got)
	}
	assertFrameAssets(t, priceUnsubscribe, []string{"asset-b"})

	replay := receiveMarketFrame(t, ctx, gotReplay)
	if got := replay["type"]; got != string(ChannelMarket) {
		t.Fatalf("replay type = %v, want %q", got, ChannelMarket)
	}
	if _, ok := replay["operation"]; ok {
		t.Fatalf("replay should be an initial subscription, got: %#v", replay)
	}
	if _, ok := replay["initial_dump"]; ok {
		t.Fatalf("replay should omit initial_dump after orderbook unsubscribe: %#v", replay)
	}
	if got := replay["custom_feature_enabled"]; got != true {
		t.Fatalf("replay custom_feature_enabled = %v, want true", got)
	}
	assertFrameAssets(t, replay, []string{"asset-a"})
}

func TestUserSubscriptionsStillReplayAfterReconnect(t *testing.T) {
	gotInitial := make(chan UserSubscription, 1)
	gotReplay := make(chan UserSubscription, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		var sub UserSubscription
		if err := wsjson.Read(ctx, conn, &sub); err != nil {
			t.Errorf("read user subscription: %v", err)
			_ = conn.CloseNow()
			return
		}
		if connCount.Add(1) == 1 {
			gotInitial <- sub
			_ = conn.CloseNow()
			return
		}

		gotReplay <- sub
		<-ctx.Done()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(
		WithHost(url),
		WithHeartbeatInterval(0),
		WithCredentials(&clob.Credentials{Key: "key", Secret: "secret", Passphrase: "passphrase"}),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.ConnectUser(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.SubscribeOrders(ctx, []string{"condition-a"}); err != nil {
		t.Fatal(err)
	}
	initial := receiveUserSubscription(t, ctx, gotInitial)
	if initial.Type != ChannelUser || initial.Operation != "" || !sameStrings(initial.Markets, []string{"condition-a"}) {
		t.Fatalf("unexpected initial user subscription: %#v", initial)
	}

	replay := receiveUserSubscription(t, ctx, gotReplay)
	if replay.Type != ChannelUser || replay.Operation != "" || !sameStrings(replay.Markets, []string{"condition-a"}) {
		t.Fatalf("unexpected replay user subscription: %#v", replay)
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

func readMarketFrame(ctx context.Context, conn *websocket.Conn) (map[string]any, error) {
	var frame map[string]any
	if err := wsjson.Read(ctx, conn, &frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func newMarketFrameServer(t *testing.T, frames chan<- map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		for {
			frame, err := readMarketFrame(ctx, conn)
			if err != nil {
				return
			}
			frames <- frame
		}
	}))
}

func receiveMarketFrame(t *testing.T, ctx context.Context, frames <-chan map[string]any) map[string]any {
	t.Helper()
	select {
	case frame := <-frames:
		return frame
	case <-ctx.Done():
		t.Fatal("timed out waiting for market subscription frame")
		return nil
	}
}

func receiveUserSubscription(t *testing.T, ctx context.Context, subs <-chan UserSubscription) UserSubscription {
	t.Helper()
	select {
	case sub := <-subs:
		return sub
	case <-ctx.Done():
		t.Fatal("timed out waiting for user subscription")
		return UserSubscription{}
	}
}

func assertNoMarketFrame(t *testing.T, frames <-chan map[string]any, timeout time.Duration) {
	t.Helper()
	select {
	case frame := <-frames:
		t.Fatalf("unexpected market subscription frame: %#v", frame)
	case <-time.After(timeout):
	}
}

func assertFrameAssets(t *testing.T, frame map[string]any, want []string) {
	t.Helper()
	raw, ok := frame["assets_ids"].([]any)
	if !ok {
		t.Fatalf("assets_ids missing or wrong type: %#v", frame)
	}
	if len(raw) != len(want) {
		t.Fatalf("assets_ids length = %d, want %d: %#v", len(raw), len(want), frame)
	}
	for idx, value := range raw {
		got, ok := value.(string)
		if !ok {
			t.Fatalf("assets_ids[%d] = %T, want string: %#v", idx, value, frame)
		}
		if got != want[idx] {
			t.Fatalf("assets_ids[%d] = %q, want %q: %#v", idx, got, want[idx], frame)
		}
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
	event, err := DecodeEvent([]byte(`{"event_type":"order","id":"order-1","asset_id":"asset-1","market":"0xabc","price":"0.42","side":"BUY","status":"LIVE"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*OrderEvent)
	if got.OrderID != "order-1" {
		t.Fatalf("OrderID = %q, want order-1", got.OrderID)
	}
}

func TestDecodeOrderPreservesDocumentedUserFields(t *testing.T) {
	event, err := DecodeEvent([]byte(`{
		"event_type": "order",
		"id": "order-1",
		"owner": "owner-1",
		"market": "0xmarket",
		"asset_id": "asset-1",
		"side": "SELL",
		"order_owner": "owner-2",
		"original_size": "10",
		"size_matched": "2.5",
		"price": "0.57",
		"associate_trades": ["trade-1"],
		"outcome": "YES",
		"type": "PLACEMENT",
		"created_at": "1672290687",
		"expiration": "1672299999",
		"order_type": "GTD",
		"status": "LIVE",
		"maker_address": "0x1234",
		"timestamp": "1672290687"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*OrderEvent)
	if got.OrderID != "order-1" || got.Owner != "owner-1" || got.OrderOwner != "owner-2" {
		t.Fatalf("unexpected ownership fields: %#v", got)
	}
	if got.OriginalSize != clob.Float64(10) || got.SizeMatched != clob.Float64(2.5) {
		t.Fatalf("sizes = %v/%v, want 10/2.5", got.OriginalSize, got.SizeMatched)
	}
	if len(got.AssociateTrades) != 1 || got.AssociateTrades[0] != "trade-1" {
		t.Fatalf("AssociateTrades = %#v", got.AssociateTrades)
	}
	if got.Outcome != "YES" || got.Type != "PLACEMENT" || got.OrderType != clob.GTD || got.MakerAddress != "0x1234" {
		t.Fatalf("documented fields not preserved: %#v", got)
	}
}

func TestDecodeOrderDoesNotMapUndocumentedOrderIDAlias(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"event_type":"order","order_id":"order-1","asset_id":"asset-1","market":"0xabc","price":"0.42","size":"10","side":"BUY","status":"LIVE"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*OrderEvent)
	if got.OrderID != "" {
		t.Fatalf("OrderID = %q, want empty for undocumented order_id alias", got.OrderID)
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

func TestDecodeTradePreservesDocumentedUserFields(t *testing.T) {
	event, err := DecodeEvent([]byte(`{
		"event_type": "trade",
		"type": "TRADE",
		"id": "trade-1",
		"taker_order_id": "taker-order-1",
		"market": "0xmarket",
		"asset_id": "asset-1",
		"side": "BUY",
		"size": "10",
		"price": "0.57",
		"fee_rate_bps": "0",
		"status": "MATCHED",
		"matchtime": "1672290701",
		"last_update": "1672290702",
		"outcome": "YES",
		"owner": "owner-1",
		"trade_owner": "owner-2",
		"maker_address": "0x1234",
		"transaction_hash": "0xhash",
		"bucket_index": 3,
		"maker_orders": [
			{
				"order_id": "maker-order-1",
				"owner": "maker-owner",
				"maker_address": "0x5678",
				"matched_amount": "10",
				"price": "0.57",
				"fee_rate_bps": "0",
				"asset_id": "asset-1",
				"outcome": "YES",
				"side": "SELL"
			}
		],
		"trader_side": "TAKER",
		"timestamp": "1672290701"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*TradeEvent)
	if got.TradeID != "trade-1" || got.TakerOrderID != "taker-order-1" || got.Type != "TRADE" {
		t.Fatalf("unexpected identifiers: %#v", got)
	}
	if got.Size != clob.Float64(10) || got.Price != clob.Float64(0.57) || got.FeeRateBps != 0 {
		t.Fatalf("unexpected price/size fields: %#v", got)
	}
	if got.Status != TradeStatusMatched || got.Owner != "owner-1" || got.TradeOwner != "owner-2" {
		t.Fatalf("unexpected status/owner fields: %#v", got)
	}
	if got.TransactionHash != "0xhash" || got.BucketIndex != clob.Int(3) || got.TraderSide != "TAKER" {
		t.Fatalf("unexpected settlement fields: %#v", got)
	}
	if len(got.MakerOrders) != 1 || got.MakerOrders[0].OrderID != "maker-order-1" || got.MakerOrders[0].MatchedAmount != clob.Float64(10) {
		t.Fatalf("MakerOrders = %#v", got.MakerOrders)
	}
}

func TestDecodeTradeDoesNotMapUndocumentedAliases(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"event_type":"trade","trade_id":"trade-1","match_time":"1672290701"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := event.(*TradeEvent)
	if got.TradeID != "" {
		t.Fatalf("TradeID = %q, want empty for undocumented trade_id alias", got.TradeID)
	}
	if !got.MatchTime.IsZero() {
		t.Fatalf("MatchTime = %s, want zero for undocumented match_time alias", got.MatchTime.Time())
	}
}

func TestMarshalTradeUsesDocumentedIDField(t *testing.T) {
	payload, err := json.Marshal(TradeEvent{TradeID: "trade-1"})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["id"] != "trade-1" {
		t.Fatalf("id = %v, want trade-1", raw["id"])
	}
	if _, ok := raw["trade_id"]; ok {
		t.Fatalf("payload should not include trade_id: %s", payload)
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
