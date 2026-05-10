# WebSocket Streams

The `clob/ws` package provides a reconnecting WebSocket client for real-time order book, price, and order notification streams.

## Quick Start

```go
import "github.com/bububa/polymarket-client/clob/ws"

client := ws.New()
defer client.Close()

ctx := context.Background()
if err := client.ConnectMarket(ctx); err != nil {
    log.Fatal(err)
}

// Subscribe to an order book
if err := client.SubscribeOrderBook(ctx, []string{"asset-id"}); err != nil {
    log.Fatal(err)
}

// Read events
for event := range client.Events() {
    if book, ok := event.(*ws.BookEvent); ok {
        fmt.Printf("bids: %d, asks: %d\n", len(book.Bids), len(book.Asks))
    }
}
```

## Authentication

Market-channel subscriptions (order book, prices, etc.) require no auth.  
User-channel subscriptions (order fills, trade events) require L2 credentials:

```go
client := ws.New(
    ws.WithCredentials(&clob.Credentials{
        Key:        "api-key",
        Secret:     "api-secret",
        Passphrase: "passphrase",
    }),
)
client.ConnectUser(ctx)
client.SubscribeOrders(ctx, []string{"condition-id"})
```

## Connection Lifecycle

The client handles three connection scenarios automatically:

```go
client := ws.New(
    ws.WithOnConnected(func() {
        // Fired once — first successful connection
        log.Println("connected")
    }),
    ws.WithOnReconnected(func() {
        // Fired on each reconnect — use to log downtime or trigger alerts
        log.Println("reconnected")
    }),
    ws.WithOnDisconnected(func() {
        // Fired only when no further reconnect will happen
        // (autoReconnect=false or client.Close() called)
        log.Println("disconnected permanently")
    }),
)
```

`WithOnDisconnected` is not fired for reconnectable drops while auto-reconnect is enabled. Use `WithOnReconnected` and `Errors()` for reconnect monitoring.

## Auto-Reconnect

By default the client reconnects with exponential backoff:

| Attempt | Delay |
|---|---|
| 1 | 1 s |
| 2 | 2 s |
| 3 | 4 s |
| 4 | 8 s |
| n | min(2ⁿ⁻¹ s, 60 s) |

After each reconnect, all subscriptions are replayed automatically.

Disable auto-reconnect:

```go
client := ws.New(ws.WithAutoReconnect(false))
```

## Stale Detection

Auto-reconnect handles read failures. To also recover from a socket that remains open but stops receiving messages, enable stale detection:

```go
client := ws.New(
    ws.WithStaleTimeout(2*time.Minute),
    ws.WithStaleCheckInterval(10*time.Second),
)
```

Stale detection is disabled by default. Any non-empty message read from the socket, including heartbeat messages, refreshes the stale timer. If no message arrives before the timeout, the client closes the active connection and uses the normal reconnect path.

## Custom Dial Options

Pass headers or subprotocols via `websocket.DialOptions`:

```go
client := ws.New(
    ws.WithDialOptions(&websocket.DialOptions{
        HTTPHeader: http.Header{"X-Custom": []string{"value"}},
    }),
)
```

## Multiple Channels

The same client can connect to different channels sequentially (each `Connect*` replaces the previous connection):

```go
client.ConnectMarket(ctx)    // market events
client.ConnectUser(ctx)       // user events  (market connection replaced)
```
For simultaneous market + user streams, create two clients with independent contexts.

## Error Handling

Asynchronous errors (read failures, reconnect errors) arrive on `client.Errors()`:

```go
for err := range client.Errors() {
    log.Printf("ws error: %v", err)
}
```

Subscribe to this channel in a separate goroutine so errors don't block the read loop.

## Closing

`Close()` is idempotent and cancels the internal context:

```go
client.Close()  // stops read loop, closes connection, no further reconnects
```
