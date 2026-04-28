# polymarket-client

[![Go Reference](https://pkg.go.dev/badge/github.com/bububa/polymarket-client.svg)](https://pkg.go.dev/github.com/bububa/polymarket-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/bububa/polymarket-client)](https://goreportcard.com/report/github.com/bububa/polymarket-client)
[![CI](https://github.com/bububa/polymarket-client/actions/workflows/go.yml/badge.svg)](https://github.com/bububa/polymarket-client/actions/workflows/go.yml)

Go SDK for [Polymarket](https://polymarket.com) — the decentralized prediction market platform on Polygon.

## Features

- **Complete CLOB v2 coverage** — market data, order management, positions, RFQ (request-for-quote), rewards
- **WebSocket support** — live order book and order update streams
- **Three-tier auth** — public (no auth), L1 (EIP-712 signatures), L2 (API key + passphrase + wallet signature)
- **All Polymarket APIs** — CLOB, Relayer, Data, Gamma, Bridge
- **Zero live dependencies** — all tests use `httptest.NewServer`, run entirely offline
- **One external dependency** — `github.com/ethereum/go-ethereum` only

## Installation

```bash
go get github.com/bububa/polymarket-client
```

Requires **Go 1.23+** (CI uses `>=1.23.0`; `go.mod` declares 1.22).

## Quick Start

### Read-Only (No Auth Required)

```go
package main

import (
    "context"
    "fmt"

    "github.com/bububa/polymarket-client/clob"
)

func main() {
    client := clob.NewClient("") // defaults to CLOB v2 host

    // Fetch market data
    marketInfo := clob.ClobMarketInfo{ConditionID: "0xabc123"}
    if err := client.GetClobMarketInfo(context.Background(), &marketInfo); err != nil {
        panic(err)
    }
    fmt.Printf("Market: %s (negRisk=%v)\n", marketInfo.ConditionID, marketInfo.NegRisk)

    // Get order book
    book, err := client.GetOrderBook(context.Background(), "token-id-here")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Best bid: %v, Best ask: %v\n", book.Bids[0].Price, book.Asks[0].Price)
}
```

### Trading (L2 Authentication Required)

The easiest way to trade is using the `OrderBuilder`, which handles price
conversion, tick-size validation, and market option lookups automatically:

```go
package main

import (
    "context"
    "fmt"

    "github.com/ethereum/go-ethereum/crypto"

    "github.com/bububa/polymarket-client/clob"
    "github.com/bububa/polymarket-client/internal/polyauth"
)

func main() {
    privateKey, _ := crypto.HexToECDSA("your-private-key-hex")

    client := clob.NewClient("",
        clob.WithCredentials(clob.Credentials{
            Key:        "your-api-key",
            Secret:     "your-api-secret",
            Passphrase: "your-api-passphrase",
        }),
        clob.WithSigner(polyauth.NewSigner(privateKey)),
        clob.WithChainID(clob.PolygonChainID), // 137
    )

    b := clob.NewOrderBuilder(client)

    // Place a limit order (auto-fetches tickSize and negRisk)
    resp, err := b.CreateAndPostOrderForToken(context.Background(), clob.OrderArgsV2{
        TokenID: "your-token-id",
        Price:   "0.50",   // price per share
        Size:    "10.0",   // number of shares
        Side:    clob.SideBuy,
    }, clob.GTC, nil)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Order placed: %s (success=%v)\n", resp.OrderID, resp.Success)
}
```

For advanced use cases (pre-fetched market options, custom tick-size handling):

```go
b := clob.NewOrderBuilder(client)

// Advanced: manually supply tickSize and negRisk
order, err := b.BuildOrder(clob.OrderArgsV2{
    TokenID: "your-token-id",
    Price:   "0.50",
    Size:    "10.0",
    Side:    clob.SideBuy,
}, clob.CreateOrderOptions{TickSize: "0.01", NegRisk: false})

// Then post manually
resp, err := b.CreateAndPostOrder(ctx, args, opts, clob.GTC, nil)
```

> **Note**: `OrderBuilder` only constructs, validates, signs, and optionally
> submits orders. It does **not** check balance, allowance, or reserved
> open-order capacity. The caller is responsible for ensuring sufficient
> funds before posting.

### Using Other APIs

```go
// Data API — positions, trades, activity (no auth)
import "github.com/bububa/polymarket-client/data"

dataClient := data.New(data.Config{})
positions, _ := dataClient.GetPositions(ctx, data.PositionParams{User: "0x..."})

// Gamma API — events, markets search, tags (no auth)
import "github.com/bububa/polymarket-client/gamma"

gammaClient := gamma.New(gamma.Config{})
markets, _ := gammaClient.GetMarkets(ctx, gamma.MarketFilterParams{/* ... */})

// Relayer API — submit signed transactions (L1 auth via API key)
import "github.com/bububa/polymarket-client/relayer"

relayerClient := relayer.New(relayer.Config{
    Credentials: &relayer.Credentials{
        APIKey:  "...",
        Address: "0x...",
    },
})
```

## Package Overview

| Package | Purpose | Default Host | Auth Required |
|---|---|---|---|
| [`clob`](#clob-package) | CLOB v2 — orders, markets, positions, RFQ | `https://clob.polymarket.com` | Depends on endpoint |
| [`clob/ws`](#clobws-package) | WebSocket live order book & updates | `wss://ws-orderbook.clob.polymarket.com` | L2 |
| [`clob/ws/rtds`](#clobws-package) | WebSocket real-time data subscriptions | `wss://ws-data.clob.polymarket.com` | None |
| [`relayer`](#relayer-package) | Submit signed on-chain transactions | `https://relayer-v2.polymarket.com` | L1 |
| [`data`](#data-package) | Positions, trades, activity, leaderboard | `https://data-api.polymarket.com` | None |
| [`gamma`](#gamma-package) | Market search, events, tags, profiles | `https://gamma-api.polymarket.com` | None |
| [`bridge`](#bridge-package) | Bridge API | `https://bridge-api.polymarket.com` | None |
| [`shared`](#shared-package) | Shared scalar types (`String`, `Int`, `Float64`, `Time`) | — | — |

## Authentication

Polymarket uses three authentication levels:

| Level | Description | How It Works | Endpoints |
|---|---|---|---|
| **AuthNone (0)** | Public access | No headers | Market data, orderbook, prices |
| **AuthL1 (1)** | Wallet-signed | EIP-712 signature of timestamp + nonce | `CreateAPIKey`, `DeriveAPIKey` |
| **AuthL2 (2)** | Full trading | API key + HMAC-secret + wallet signature | Orders, trades, positions, RFQ |

L2 auth requires BOTH a `polyauth.Signer` (from your private key) AND `Credentials` (API key, secret, passphrase).

### Creating API Keys

```go
client := clob.NewClient("",
    clob.WithSigner(polyauth.NewSigner(privateKey)),
    clob.WithChainID(clob.PolygonChainID),
)

// Create new API key (L1 — wallet-signed)
creds, err := client.CreateAPIKey(ctx, nonce)
// Use returned credentials for L2 requests
```

## CLOB Package

All CLOB v2 endpoints:

### Market Data (No Auth)

| Method | Endpoint | Description |
|---|---|---|
| `GetOk` | `/ok` | Health check |
| `GetVersion` | `/version` | API version |
| `GetServerTime` | `/time` | Server timestamp |
| `GetMarkets` | `/markets` | Paginated markets |
| `GetClobMarketInfo` | `/clob-markets/:id` | Single market details |
| `GetOrderBook` | `/book` | Order book for token |
| `GetMidpoint` | `/midpoint` | Midpoint price |
| `GetPrice` | `/price` | Last price by side |
| `GetSpread` | `/spread` | Bid-ask spread |
| `GetLastTradePrice` | `/last-trade-price` | Most recent trade |
| `GetTickSize` | `/tick-size` | Minimum price increment |

### Orders & Trading (AuthL2)

| Method | Endpoint | Description |
|---|---|---|
| `PostOrder` | `/order` | Submit single order |
| `PostOrders` | `/orders` | Submit batch orders (supports `postOnly`, `deferExec`) |
| `CancelOrder` | `/order` | Cancel by order ID |
| `CancelOrders` | `/orders` | Cancel multiple orders |
| `CancelAll` | `/cancel-all` | Cancel all user orders |
| `CancelMarketOrders` | `/cancel-market-orders` | Cancel by market |
| `GetOrder` | `/data/order/:id` | Get order by ID |
| `GetOpenOrders` | `/data/orders` | List open orders |
| `GetTrades` | `/data/trades` | List user trades |
| `GetTickSize` | `/tick-size` | Minimum price increment |
| `GetTickSizeByTokenID` | `/tick-size/:id` | Minimum price increment by token |
| `GetNegRisk` | `/neg-risk` | Whether a token uses neg-risk resolution |

### OrderBuilder (Recommended for Trading)

The `OrderBuilder` provides a high-level API that automatically fetches
`tickSize` and `negRisk` from the CLOB API, so you don't need to supply them:

```go
b := clob.NewOrderBuilder(client)

// Limit order — auto-fetches tickSize + negRisk
resp, err := b.CreateAndPostOrderForToken(ctx, clob.OrderArgsV2{
    TokenID: "token-id",
    Price:   "0.50",
    Size:    "10.0",
    Side:    clob.SideBuy,
}, clob.GTC, nil)

// Market order — Amount is USDC for BUY, shares for SELL
resp, err := b.CreateAndPostMarketOrderForToken(ctx, clob.MarketOrderArgsV2{
    TokenID: "token-id",
    Price:   "0.50",   // worst-price limit
    Amount:  "100",    // BUY: USDC to spend / SELL: shares to sell
    Side:    clob.SideBuy,
}, clob.FOK, nil)
```

**Order types**: `GTC`, `GTD` for limit orders; `FOK`, `FAK` for market orders.

> `deferExec` (post-only) is only valid with `GTC`/`GTD`. Pairing it with
> `FOK`/`FAK` returns an error.

### RFQ (Request for Quote) (AuthL2)

| Method | Endpoint | Description |
|---|---|---|
| `CreateRFQRequest` | `/rfq/request` | Create RFQ |
| `GetRFQRequests` | `/rfq/data/requests` | List RFQs |
| `CreateRFQQuote` | `/rfq/quote` | Create RFQ quote |
| `AcceptRFQRequest` | `/rfq/request/accept` | Accept RFQ |
| `ApproveRFQQuote` | `/rfq/quote/approve` | Approve quote |

### Rewards (AuthL2 + Public)

| Method | Auth | Description |
|---|---|---|
| `GetEarningsForUserForDay` | L2 | User rewards for a date |
| `GetCurrentRewards` | None | Active reward campaigns |
| `GetRewardsForMarket` | None | Rewards for a market |
| `GetBuilderFeeRate` | None | Builder fee configuration |

### WebSocket (`clob/ws`)

```go
import "github.com/bububa/polymarket-client/clob/ws"

wsClient, err := ws.New(ws.Config{
    Host: "", // defaults to production
    // Optional: auth for order notifications
    Signer:      polyauth.NewSigner(privateKey),
    Credentials: &ws.Credentials{/* ... */},
    ChainID:     137,
})
defer wsClient.Close()

// Subscribe to order book
err = wsClient.SubscribeOrderBook("token-id")
// Subscribe to order updates (requires auth)
err = wsClient.SubscribeOrders()

// Read updates
for update := range wsClient.Channel {
    fmt.Printf("Update: %+v\n", update)
}
```

## Development

```bash
# Build
go build -v ./...

# Run tests (all offline — httptest.NewServer)
go test -v ./...

# Tidy dependencies
go mod tidy
```

### Test Files

| File | Coverage |
|---|---|
| `clob/auth_test.go` | Auth header generation, HMAC signatures |
| `clob/client_test.go` | CLOB v2 endpoints, flexible JSON parsing |
| `clob/ctf_test.go` | CTF relayer transaction submission |
| `clob/amount_test.go` | Amount calculation, tick validation, bytes32 checks |
| `clob/order_builder_test.go` | OrderBuilder price invariants, market order semantics, deferExec |
| `clob/sign_order_test.go` | V2 signing, domain hash, expiration handling |
| `relayer/client_test.go` | Relayer documented endpoints |
| `shared/flex_test.go` | Flexible JSON scalar serialization |

## License

[MIT](LICENSE)
