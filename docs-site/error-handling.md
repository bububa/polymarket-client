# Error Handling, Rate Limiting, and Retry Strategy

This SDK does **not** include built-in retry logic or timeout defaults. Each call returns a standard Go `error`. Defensive patterns are your responsibility.

## The `APIError` Type

Non-2xx responses return `*clob.APIError`:

```go
var apiErr *clob.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("status: %d\n", apiErr.StatusCode)
    fmt.Printf("message: %s\n", apiErr.Message)
    fmt.Printf("raw body: %s\n", apiErr.Body)
}
```

`polyhttp.APIError` (used by other packages) adds `RequestBody` for debugging and an `HTTPStatus()` method.

## Handle Status Codes by Category

```go
var apiErr *clob.APIError
if errors.As(err, &apiErr) {
    switch apiErr.StatusCode {
    case 429:
        // rate limited — back off and retry
    case 401, 403:
        // credentials invalid — re-authenticate
    case 503, 500, 502:
        // server error — retry with exponential backoff
    case 400:
        // bad request — check input, do NOT retry
    }
}
```

## Always Set a Timeout

The default HTTP client has **no timeout**. A stalled connection blocks forever.

```go
client := clob.NewClient("",
    clob.WithHTTPClient(&http.Client{Timeout: 15 * time.Second}),
    clob.WithCredentials(creds),
    clob.WithSigner(signer),
    clob.WithChainID(clob.PolygonChainID),
)
```

## Missing Identifier Errors

Some methods require an identifier field on the output struct. If empty, the SDK returns `errMissingIdentifier` without a network call. **This error is unexported** inside the `clob` package, so check via `errors.Is` against the known message prefix:

```go
if err != nil && strings.Contains(err.Error(), "missing identifier") {
    // ConditionID or AssetID was not set on the output struct
}
```

## Retry with Exponential Backoff

### No dependencies

```go
func retryWithBackoff(ctx context.Context, maxAttempts int, fn func() error) error {
    backoff := 100 * time.Millisecond
    var lastErr error
    for i := 0; i < maxAttempts; i++ {
        lastErr = fn()
        if lastErr == nil {
            return nil
        }
        var apiErr *clob.APIError
        if errors.As(lastErr, &apiErr) && apiErr.StatusCode == 400 {
            return lastErr // permanent — skip retry
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
        }
        backoff *= 2
    }
    return fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}
```

### Using `github.com/cenkalti/backoff/v4`

```go
b := backoff.NewExponentialBackOff()
b.MaxElapsedTime = 10 * time.Second

err := backoff.Retry(func() error {
    err := client.GetOrderBook(ctx, &book)
    if err == nil {
        return nil
    }
    var apiErr *clob.APIError
    if errors.As(err, &apiErr) && apiErr.StatusCode == 400 {
        return backoff.Permanent(err)
    }
    return err
}, b)
```

## Rate Limits

Rate limits are set server-side and may change. They are not currently published in a machine-readable format. On `429`, check `Retry-After` response headers and honor the value.

**Self-throttle** instead of waiting for the server. For order-book polling, 1.2 second intervals work well. For order submission, serialize calls rather than flooding.

## Production-Ready Example

```go
func main() {
    client := clob.NewClient("",
        clob.WithHTTPClient(&http.Client{Timeout: 15 * time.Second}),
    )

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    var book clob.OrderBookSummary
    book.AssetID = clob.Int{Value: 12345}

    err := retryWithBackoff(ctx, 3, func() error {
        return client.GetOrderBook(ctx, &book)
    })

    if strings.Contains(fmt.Sprint(err), "missing identifier") {
        fmt.Println("set AssetID before querying")
    } else if err != nil {
        fmt.Println("failed:", err)
    }
}
```
