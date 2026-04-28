# Trading Orders

The recommended way to place orders is using the `OrderBuilder`, which handles
price/size conversion, tick-size validation, neg-risk detection, and EIP-712
signing automatically.

## OrderBuilder — Limit Orders

```go
b := clob.NewOrderBuilder(client)

// Auto-fetches tickSize and negRisk from the CLOB API
resp, err := b.CreateAndPostOrderForToken(ctx, clob.OrderArgsV2{
    TokenID: "0xtoken...",
    Price:   "0.65",
    Size:    "50.0",
    Side:    clob.SideBuy, // or clob.SideSell
}, clob.GTC, nil)
```

Order types: `clob.GTC` (Good Till Cancel), `clob.GTD` (Good Till Date).

For GTD orders, the expiration must be at least 60 seconds in the future:

```go
resp, err := b.CreateAndPostOrderForToken(ctx, clob.OrderArgsV2{
    TokenID:    "0xtoken...",
    Price:      "0.65",
    Size:       "50.0",
    Side:       clob.SideBuy,
    Expiration: "1735689600", // Unix timestamp
}, clob.GTD, nil)
```

## OrderBuilder — Market Orders

```go
resp, err := b.CreateAndPostMarketOrderForToken(ctx, clob.MarketOrderArgsV2{
    TokenID:    "0xtoken...",
    Price:      "0.50",  // worst-price limit (slippage protection)
    Amount:     "100",    // BUY: USDC to spend / SELL: shares to sell
    Side:       clob.SideBuy,
}, clob.FOK, nil)
```

Market order types: `clob.FOK` (Fill Or Kill), `clob.FAK` (Fill And Kill).

> `deferExec` (post-only) is only valid with GTC/GTD. Using it with FOK/FAK
> returns an error.

## Manual Signing (Advanced)

If you need full control, you can build and post orders manually:

```go
order, err := b.BuildOrder(clob.OrderArgsV2{
    TokenID: "0xtoken...",
    Price:   "0.65",
    Size:    "50.0",
    Side:    clob.SideBuy,
}, clob.CreateOrderOptions{
    TickSize: "0.01",
    NegRisk:  false,
})

resp, err := b.CreateAndPostOrder(ctx, clob.OrderArgsV2{...}, opts, clob.GTC, nil)
```

## Cancelling Orders

```go
// Single order
resp, err := client.CancelOrder(ctx, "order-id")

// Multiple orders
resp, err := client.CancelOrders(ctx, []string{"id1", "id2"})

// All orders for the user
resp, err := client.CancelAll(ctx)

// Cancel orders for a specific market
resp, err := client.CancelMarketOrders(ctx, clob.OrderMarketCancelParams{
    Market: "0xconditionID",
})
```

## Querying Orders

```go
// Specific order
order, err := client.GetOrder(ctx, "order-id")

// All open orders
orders, err := client.GetOpenOrders(ctx, clob.OpenOrderParams{
    Market:  "0xconditionID",
    AssetID: "0xtokenID",
})

// User's trade history
trades, err := client.GetTrades(ctx, clob.TradeParams{
    Market: "0xconditionID",
    Limit:  100,
})
```

## Checking Order Scoring

```go
// Single order
scoring, err := client.IsOrderScoring(ctx, "order-id")

// Multiple orders
scores, err := client.AreOrdersScoring(ctx, []string{"id1", "id2"})
```
