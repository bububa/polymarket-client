package rtds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestClientSubscribesAndDecodesMessage(t *testing.T) {
	upgrader := websocket.Upgrader{}
	gotSub := make(chan SubscriptionRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		var sub SubscriptionRequest
		if err := conn.ReadJSON(&sub); err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}
		gotSub <- sub
		if err := conn.WriteJSON(Message{
			Topic:     "crypto_prices",
			Type:      "update",
			Timestamp: 1700000000,
			Payload:   []byte(`{"symbol":"BTCUSDT","timestamp":1700000000,"value":"65000"}`),
		}); err != nil {
			t.Errorf("write message: %v", err)
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client := NewClient(url).WithAutoReconnect(false)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.SubscribeCryptoPrices(ctx, []string{"BTCUSDT"}); err != nil {
		t.Fatal(err)
	}

	select {
	case sub := <-gotSub:
		if sub.Action != ActionSubscribe || len(sub.Subscriptions) != 1 || sub.Subscriptions[0].Topic != "crypto_prices" {
			t.Fatalf("unexpected subscription: %#v", sub)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for subscription")
	}

	select {
	case msg := <-client.Messages():
		var price CryptoPrice
		if err := msg.AsCryptoPrice(&price); err != nil {
			t.Fatal(err)
		}
		if price.Symbol != "BTCUSDT" || price.Value != "65000" {
			t.Fatalf("unexpected price: %#v", price)
		}
	case err := <-client.Errors():
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for message")
	}
}
