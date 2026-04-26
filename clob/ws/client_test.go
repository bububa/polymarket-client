package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/gorilla/websocket"
)

func TestClientSubscribesAndDecodesEvents(t *testing.T) {
	upgrader := websocket.Upgrader{}
	gotSub := make(chan MarketSubscription, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		var sub MarketSubscription
		if err := conn.ReadJSON(&sub); err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}
		gotSub <- sub
		event := BookEvent{
			BaseEvent: BaseEvent{EventType: EventTypeBook},
			AssetID:   "asset-1",
			Bids:      []clob.OrderSummary{{Price: clob.Float64(0.45), Size: clob.Float64(10)}},
			Asks:      []clob.OrderSummary{{Price: clob.Float64(0.55), Size: clob.Float64(20)}},
			Timestamp: "1700000000000",
		}
		if err := conn.WriteJSON([]BookEvent{event}); err != nil {
			t.Errorf("write event: %v", err)
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := NewClient(url).WithAutoReconnect(false).WithHeartbeat(time.Hour, time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
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
	client := NewClient("").WithAutoReconnect(false)
	err := client.SubscribeOrders(context.Background(), []string{"condition"})
	if err == nil {
		t.Fatal("expected missing credentials error")
	}
}

func TestDecodeEventArrayError(t *testing.T) {
	events := decodeEvents([]byte(`[{"event_type":"unknown"}]`))
	if len(events) != 1 || events[0].err == nil {
		data, _ := json.Marshal(events)
		t.Fatalf("expected decode error, got %s", data)
	}
}
