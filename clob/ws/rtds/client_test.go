package rtds

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

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestSupportedSymbolsReturnCopies(t *testing.T) {
	binance := SupportedBinanceSymbols()
	if len(binance) != 4 || binance[0] != BinanceSymbolBTCUSDT {
		t.Fatalf("unexpected Binance symbols: %#v", binance)
	}
	binance[0] = "mutated"
	if SupportedBinanceSymbols()[0] != BinanceSymbolBTCUSDT {
		t.Fatal("SupportedBinanceSymbols returned mutable backing storage")
	}

	equity := SupportedEquitySymbols()
	if len(equity) != 28 {
		t.Fatalf("len(SupportedEquitySymbols) = %d, want 28", len(equity))
	}
	if !IsSupportedBinanceSymbol("BTCUSDT") ||
		!IsSupportedChainlinkSymbol("ETH/USD") ||
		!IsSupportedEquitySymbol("aapl") {
		t.Fatal("supported symbol lookup should be case-insensitive")
	}
}

func TestSubscriptionWireShapeMatchesOfficialDocs(t *testing.T) {
	tests := []struct {
		name string
		sub  Subscription
		want string
	}{
		{
			name: "binance comma separated filters",
			sub: Subscription{
				Topic:   TopicCryptoPrices,
				Type:    TypeUpdate,
				Filters: []string{BinanceSymbolSOLUSDT, BinanceSymbolBTCUSDT, BinanceSymbolETHUSDT},
			},
			want: `{"topic":"crypto_prices","type":"update","filters":"solusdt,btcusdt,ethusdt"}`,
		},
		{
			name: "chainlink json string filters",
			sub: Subscription{
				Topic:   TopicCryptoPricesChainlink,
				Type:    TypeAll,
				Filters: map[string]string{"symbol": ChainlinkSymbolETHUSD},
			},
			want: `{"topic":"crypto_prices_chainlink","type":"*","filters":"{\"symbol\":\"eth/usd\"}"}`,
		},
		{
			name: "chainlink all symbols",
			sub: Subscription{
				Topic:   TopicCryptoPricesChainlink,
				Type:    TypeAll,
				Filters: "",
			},
			want: `{"topic":"crypto_prices_chainlink","type":"*","filters":""}`,
		},
		{
			name: "equity json string filters",
			sub: Subscription{
				Topic:   TopicEquityPrices,
				Type:    TypeUpdate,
				Filters: map[string]string{"symbol": EquitySymbolAAPL},
			},
			want: `{"topic":"equity_prices","type":"update","filters":"{\"symbol\":\"AAPL\"}"}`,
		},
		{
			name: "gamma auth",
			sub: Subscription{
				Topic:     TopicComments,
				Type:      string(CommentCreated),
				GammaAuth: &GammaAuth{Address: "0xabc"},
			},
			want: `{"topic":"comments","type":"comment_created","gamma_auth":{"address":"0xabc"}}`,
		},
		{
			name: "deprecated clob auth",
			sub: Subscription{
				Topic:    TopicComments,
				Type:     string(CommentCreated),
				CLOBAuth: &Credentials{Key: "key", Secret: "secret", Passphrase: "pass"},
			},
			want: `{"topic":"comments","type":"comment_created","clob_auth":{"key":"key","secret":"secret","passphrase":"pass"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.sub)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, got, []byte(tt.want))
		})
	}
}

func TestClientSubscribesAndDecodesMessage(t *testing.T) {
	gotSub := make(chan SubscriptionRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := context.Background()
		var sub SubscriptionRequest
		if err := wsjson.Read(ctx, conn, &sub); err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}
		gotSub <- sub

		if err := wsjson.Write(ctx, conn, Message{
			Topic:     TopicCryptoPrices,
			Type:      TypeUpdate,
			Timestamp: 1700000000000,
			Payload:   []byte(`{"symbol":"btcusdt","timestamp":1700000000000,"value":65000}`),
		}); err != nil {
			t.Errorf("write message: %v", err)
		}

		<-ctx.Done()
	}))
	defer server.Close()

	client := NewClient(wsURL(server)).WithAutoReconnect(false).WithHeartbeatInterval(0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.SubscribeCryptoPrices(ctx, []string{BinanceSymbolBTCUSDT}); err != nil {
		t.Fatal(err)
	}

	select {
	case sub := <-gotSub:
		if sub.Action != ActionSubscribe ||
			len(sub.Subscriptions) != 1 ||
			sub.Subscriptions[0].Topic != TopicCryptoPrices ||
			sub.Subscriptions[0].Filters != BinanceSymbolBTCUSDT {
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
		if price.Symbol != BinanceSymbolBTCUSDT || float64(price.Value) != 65000 {
			t.Fatalf("unexpected price: %#v", price)
		}
	case err := <-client.Errors():
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for message")
	}
}

func TestEquityPayloadDecoding(t *testing.T) {
	update := Message{
		Topic:   TopicEquityPrices,
		Type:    TypeUpdate,
		Payload: []byte(`{"symbol":"aapl","value":198.45,"full_accuracy_value":"198.4523","timestamp":1711382400000,"received_at":1711382400005,"is_carried_forward":true}`),
	}
	var price EquityPrice
	if err := update.AsEquityPrice(&price); err != nil {
		t.Fatal(err)
	}
	if price.Symbol != "aapl" ||
		float64(price.Value) != 198.45 ||
		price.FullAccuracyValue != "198.4523" ||
		!price.IsCarriedForward {
		t.Fatalf("unexpected equity price: %#v", price)
	}

	snapshotMsg := Message{
		Topic:   TopicEquityPrices,
		Type:    TypeSubscribe,
		Payload: []byte(`{"symbol":"aapl","data":[{"timestamp":1711382280000,"value":198.30},{"timestamp":1711382340000,"value":"198.41"}]}`),
	}
	var snapshot EquityPriceSnapshot
	if err := snapshotMsg.AsEquityPriceSnapshot(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Symbol != "aapl" || len(snapshot.Data) != 2 || float64(snapshot.Data[1].Value) != 198.41 {
		t.Fatalf("unexpected equity snapshot: %#v", snapshot)
	}
}

func TestConnectContextDoesNotStopReadLoop(t *testing.T) {
	writeMessage := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		<-writeMessage
		if err := wsjson.Write(context.Background(), conn, Message{
			Topic:   TopicCryptoPrices,
			Type:    TypeUpdate,
			Payload: []byte(`{"symbol":"btcusdt","timestamp":1,"value":1}`),
		}); err != nil {
			t.Errorf("write message: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(wsURL(server)).WithAutoReconnect(false).WithHeartbeatInterval(0)
	connectCtx, cancelConnect := context.WithTimeout(context.Background(), time.Second)
	if err := client.Connect(connectCtx); err != nil {
		t.Fatal(err)
	}
	cancelConnect()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	close(writeMessage)

	select {
	case msg := <-client.Messages():
		var price CryptoPrice
		if err := msg.AsCryptoPrice(&price); err != nil {
			t.Fatal(err)
		}
	case err := <-client.Errors():
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for message after connect context cancellation")
	}
}

func TestClientSendsHeartbeatPing(t *testing.T) {
	gotPing := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		msgType, data, err := conn.Read(context.Background())
		if err != nil {
			t.Errorf("read heartbeat: %v", err)
			return
		}
		if msgType != websocket.MessageText || string(data) != "PING" {
			t.Errorf("message = %v %q, want text PING", msgType, data)
			return
		}
		gotPing <- struct{}{}
	}))
	defer server.Close()

	client := NewClient(wsURL(server)).WithAutoReconnect(false).WithHeartbeatInterval(20 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	select {
	case <-gotPing:
	case <-ctx.Done():
		t.Fatal("timed out waiting for heartbeat PING")
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
		msgType, data, err := conn.Read(ctx)
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

	client := NewClient(wsURL(server)).WithAutoReconnect(false).WithHeartbeatInterval(0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	select {
	case <-gotPong:
	case <-ctx.Done():
		t.Fatal("timed out waiting for pong")
	}

	select {
	case msg := <-client.Messages():
		t.Fatalf("unexpected message from ping/pong frame: %#v", msg)
	case err := <-client.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestReconnectReplaysSubscriptions(t *testing.T) {
	gotInitialSub := make(chan SubscriptionRequest, 1)
	gotReplaySub := make(chan SubscriptionRequest, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		var sub SubscriptionRequest
		if err := wsjson.Read(context.Background(), conn, &sub); err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}
		if connCount.Add(1) == 1 {
			gotInitialSub <- sub
			_ = conn.CloseNow()
			return
		}
		gotReplaySub <- sub
		if err := wsjson.Write(context.Background(), conn, Message{
			Topic:   TopicCryptoPrices,
			Type:    TypeUpdate,
			Payload: []byte(`{"symbol":"btcusdt","timestamp":1,"value":2}`),
		}); err != nil {
			t.Errorf("write message: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(wsURL(server)).WithHeartbeatInterval(0)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.SubscribeCryptoPrices(ctx, []string{BinanceSymbolBTCUSDT}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-gotInitialSub:
	case <-ctx.Done():
		t.Fatal("timed out waiting for initial subscription")
	}
	select {
	case sub := <-gotReplaySub:
		if len(sub.Subscriptions) != 1 || sub.Subscriptions[0].Filters != BinanceSymbolBTCUSDT {
			t.Fatalf("unexpected replay subscription: %#v", sub)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for replay subscription")
	}
	for {
		select {
		case <-client.Messages():
			return
		case <-client.Errors():
		case <-ctx.Done():
			t.Fatal("timed out waiting for replayed message")
		}
	}
}

func TestStaleDetectionForcesReconnectAndReplaysSubscriptions(t *testing.T) {
	gotReplaySub := make(chan SubscriptionRequest, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		var sub SubscriptionRequest
		if err := wsjson.Read(context.Background(), conn, &sub); err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}
		if connCount.Add(1) == 1 {
			time.Sleep(200 * time.Millisecond)
			return
		}
		gotReplaySub <- sub
	}))
	defer server.Close()

	client := NewClient(wsURL(server)).
		WithHeartbeatInterval(0).
		WithStaleTimeout(40 * time.Millisecond).
		WithStaleCheckInterval(10 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.SubscribeCryptoPrices(ctx, []string{BinanceSymbolBTCUSDT}); err != nil {
		t.Fatal(err)
	}

	select {
	case sub := <-gotReplaySub:
		if len(sub.Subscriptions) != 1 || sub.Subscriptions[0].Topic != TopicCryptoPrices {
			t.Fatalf("unexpected replay subscription: %#v", sub)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for stale reconnect replay")
	}
}

func TestSubscribeFailureRollsBack(t *testing.T) {
	client := NewClient("")
	err := client.SubscribeCryptoPrices(context.Background(), []string{BinanceSymbolBTCUSDT})
	if err == nil {
		t.Fatal("expected subscribe to fail without connection")
	}
	if len(client.subs) != 0 {
		t.Fatalf("len(subs) = %d, want 0", len(client.subs))
	}
}

func TestRemoveSubscriptionMatchesFiltersExactly(t *testing.T) {
	client := NewClient("")
	aapl := Subscription{Topic: TopicEquityPrices, Type: TypeUpdate, Filters: map[string]string{"symbol": EquitySymbolAAPL}}
	tsla := Subscription{Topic: TopicEquityPrices, Type: TypeUpdate, Filters: map[string]string{"symbol": EquitySymbolTSLA}}
	client.subs = []Subscription{aapl, tsla}

	client.removeSubscription(aapl)
	if len(client.subs) != 1 {
		t.Fatalf("len(subs) = %d, want 1", len(client.subs))
	}
	if !sameSubscription(client.subs[0], tsla) {
		t.Fatalf("remaining subscription = %#v, want TSLA", client.subs[0])
	}
}

func TestLifecycleCallbacks(t *testing.T) {
	connected := make(chan struct{}, 1)
	reconnected := make(chan struct{}, 1)
	disconnected := make(chan struct{}, 1)
	var connCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		if connCount.Add(1) == 1 {
			_ = conn.CloseNow()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	client := NewClient(wsURL(server)).
		WithHeartbeatInterval(0).
		WithOnConnected(func() { connected <- struct{}{} }).
		WithOnReconnected(func() { reconnected <- struct{}{} }).
		WithOnDisconnected(func() { disconnected <- struct{}{} })
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case <-connected:
	case <-ctx.Done():
		t.Fatal("timed out waiting for connected callback")
	}
	select {
	case <-reconnected:
	case <-ctx.Done():
		t.Fatal("timed out waiting for reconnected callback")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-disconnected:
	case <-ctx.Done():
		t.Fatal("timed out waiting for disconnected callback")
	}
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var gotAny any
	if err := json.Unmarshal(got, &gotAny); err != nil {
		t.Fatalf("unmarshal got: %v\n%s", err, got)
	}
	var wantAny any
	if err := json.Unmarshal(want, &wantAny); err != nil {
		t.Fatalf("unmarshal want: %v\n%s", err, want)
	}
	gotJSON, _ := json.Marshal(gotAny)
	wantJSON, _ := json.Marshal(wantAny)
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("json = %s, want %s", gotJSON, wantJSON)
	}
}

func wsURL(server *httptest.Server) string {
	return "ws" + strings.TrimPrefix(server.URL, "http")
}
