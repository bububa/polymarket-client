# Troubleshooting

## Authentication Errors

### "signer is required for authenticated request"

You're trying to access an L1 or L2 endpoint without setting a signer.

```go
client := clob.NewClient("",
    clob.WithSigner(polyauth.NewSigner(privateKey)),
)
```

### "api credentials are required for level 2 authenticated request"

L2 endpoints need both signer AND credentials:

```go
client := clob.NewClient("",
    clob.WithCredentials(clob.Credentials{Key: "...", Secret: "...", Passphrase: "..."}),
    clob.WithSigner(polyauth.NewSigner(privateKey)),
)
```

## Network Issues

### Connection refused

Verify the host URL. The CLOB client defaults to `https://clob.polymarket.com` if host is empty string.

### Timeout errors

Set a custom HTTP client with longer timeout:

```go
client := clob.NewClient("",
    clob.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
)
```

## JSON Parsing

### "cannot parse server time from map[string]any"

The `/time` endpoint response format changed. Update to the latest version.

### Float values are 0

Polymarket returns prices as decimal strings (`"0.50"`). Use `shared.Float64` or `shared.String` instead of raw `float64`.

## WebSocket

### Connection drops and reconnects

`clob/ws` auto-reconnects by default and replays stored subscriptions. Read asynchronous errors from `client.Errors()` so reconnect problems are visible to your application.

Disable reconnect only if you want to own the lifecycle yourself:

```go
client := ws.New(ws.WithAutoReconnect(false))
```

### Socket stays open but no messages arrive

Enable stale detection if your application prefers a forced reconnect after a quiet period:

```go
client := ws.New(
    ws.WithStaleTimeout(2*time.Minute),
    ws.WithStaleCheckInterval(10*time.Second),
)
```

Any non-empty message, including heartbeat messages, refreshes the stale timer.

## Build Issues

### "go.mod requires 1.22 but I'm using 1.21"

The CI uses Go 1.23+. Upgrade your Go toolchain.

### Import not found

Run `go mod tidy` and ensure the module path is `github.com/bububa/polymarket-client`.
