---
outline: deep
---

# Changelog

## Unreleased

### Breaking Changes

- **Pre-allocation API**: All single-entity getters now accept an output pointer and return `error` only. Methods no longer allocate and return `*T`.

  ```go
  // Before
  market, err := client.GetMarket(ctx, "market-id")

  // After
  var market gamma.Market
  market.ID = "market-id"
  err := client.GetMarket(ctx, &market)
  ```

- **Gamma ID type**: `Market.ID`, `Event.ID`, `Series.ID`, and `Tag.ID` are now `shared.String` instead of `shared.Int`. Use string values in struct literals.

- **CTF helpers**: `BuildSplitPositionTx`, `BuildMergePositionsTx`, `BuildRedeemPositionsTx`, and `BuildRedeemNegRiskTx` now accept an output `*CTFTransaction` pointer.

- **Relayer SubmitTransaction**: Now accepts `*SubmitTransactionResponse` as third argument and returns only `error`.

- **RelayerSubmitter interface**: Method signature changed to `SubmitTransaction(ctx, req, *SubmitTransactionResponse) error`.

### Additions

- **`clob.OrderBuilder`** — high-level API for constructing and posting V2 orders
  with automatic tick-size validation, price-range checks, and neg-risk detection
- **`BuildOrderForToken` / `CreateAndPostOrderForToken`** — convenience methods
  that auto-fetch `tickSize` and `negRisk` from the CLOB API
- **`BuildMarketOrderForToken` / `CreateAndPostMarketOrderForToken`** — same for
  FOK/FAK market orders (amount = USDC for BUY, shares for SELL)
- Limit order price invariants: BUY implied price ≤ limit, SELL implied price ≥ limit
- Market order worst-price protection via ceiling takerAmount
- GTD expiration validation (must be ≥ now + 60s)
- `deferExec` + `FOK`/`FAK` local rejection (post-only only valid with GTC/GTD)
- `ValidateBytes32Hex` for builder code and metadata format validation
- Full Go doc comments on all exported types, fields, and methods
- VitePress documentation site with API reference for all packages
- API reference pages: relayer, data, gamma, bridge, types
- Gamma pre-allocation test coverage
