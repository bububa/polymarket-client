# AGENTS.md — polymarket-client

Go SDK for Polymarket (prediction market on Polygon). Library only — no binaries.

## Commands

```bash
go build -v ./...     # compile
go test -v ./...      # tests; no live API required
go mod tidy           # clean up go.mod
gofmt -w .            # format changed Go files
```

CI uses Go >=1.23.0. `go.mod` currently declares 1.22, but prefer 1.23+ when developing locally.

For golden vector maintenance:

```bash
make
```

Golden-vector generation uses Python and the official `py-clob-client-v2` reference implementation. Do not require this for normal unit tests.

## Package Map

| Package              | Purpose                                                                            | Auth Required       |
| -------------------- | ---------------------------------------------------------------------------------- | ------------------- |
| `clob/`              | CLOB v2 API — orders, markets, positions, RFQ, CTF helpers, deposit-wallet helpers | Mixed (0–2)         |
| `clob/ws/`           | WebSocket for live order book / order updates                                      | L2 for user channel |
| `clob/ws/rtds/`      | Real-time data subscriptions                                                       | None                |
| `relayer/`           | Relayer API wire client — submit signed SAFE/PROXY/WALLET requests                 | API key             |
| `data/`              | Read-only market data API                                                          | None                |
| `gamma/`             | Read-only gamma market-data API                                                    | None                |
| `bridge/`            | Bridge API                                                                         | None                |
| `shared/`            | Shared scalar types (`String`, `Int`, `Float64`, `Time`)                           | —                   |
| `internal/polyhttp/` | HTTP client with `AuthLevel` (`0=None`, `1=L1`, `2=L2`)                            | —                   |
| `internal/polyauth/` | EIP-712 signing, L1/L2 header generation, low-level hash signing                   | —                   |

## Client Construction Pattern

All public packages use `NewClient(host, opts...)` with functional options:

```go
// Minimal read-only CLOB client:
client := clob.NewClient("") // defaults to CLOB v2 host

// Data / Gamma clients:
dataClient := data.NewClient("")
gammaClient := gamma.NewClient("")

// Trading client:
signer, err := clob.ParsePrivateKey("0x...")
if err != nil {
    panic(err)
}

client := clob.NewClient("",
    clob.WithCredentials(clob.Credentials{
        Key:        "your-api-key",
        Secret:     "your-api-secret",
        Passphrase: "your-api-passphrase",
    }),
    clob.WithSigner(signer),
    clob.WithChainID(clob.PolygonChainID), // 137
)
```

Avoid importing `internal/polyauth` from README or external examples. Use exported `clob.ParsePrivateKey` or exported package-level helpers.

## Auth Levels

- **AuthNone (0)** — public endpoints such as market data, order book, prices.
- **AuthL1 (1)** — EIP-712 signed L1 headers, for example `CreateAPIKey` / `DeriveAPIKey`.
- **AuthL2 (2)** — requires both `Signer` and `Credentials` (API key, secret, passphrase). Used for order/trade/account endpoints.

L2 auth without credentials returns:

```text
api credentials are required for level 2 authenticated request
```

## Signature Types and Wallet Modes

CLOB v2 order signing supports these signature types:

| Signature type            | Value | Meaning                    |
| ------------------------- | ----: | -------------------------- |
| `SignatureTypeEOA`        |   `0` | Direct EOA signing         |
| `SignatureTypeProxy`      |   `1` | Polymarket proxy wallet    |
| `SignatureTypeGnosisSafe` |   `2` | Safe wallet                |
| `SignatureTypePoly1271`   |   `3` | Deposit wallet / POLY_1271 |

### Deposit wallet / POLY_1271

Deposit-wallet orders are not ordinary EOA order signatures. They use an ERC-7739 wrapped `POLY_1271` signature.

Required order shape:

```text
signatureType = 3
maker = deposit wallet address
signer = deposit wallet address
cryptographic signer = owner/session signer
```

High-level `OrderBuilder` supports this path when `SignatureTypePoly1271` is set and `Maker` is the deposit wallet address:

```go
sigType := clob.SignatureTypePoly1271

order, err := builder.BuildOrder(clob.OrderArgsV2{
    TokenID:       tokenID,
    Price:         "0.42",
    Size:          "10",
    Side:          clob.Buy,
    SignatureType: &sigType,
    Maker:         depositWallet.Hex(),
}, clob.CreateOrderOptions{TickSize: "0.01"})
```

`SignOrder` also dispatches to the deposit-wallet signing path when `SignatureTypePoly1271` is present. Do not route deposit-wallet orders through the legacy EOA/SAFE/PROXY signing assumptions.

## Balance / Allowance

`UpdateBalanceAllowance` is a `POST` endpoint and uses the existing `BalanceAllowanceResponse`.

For deposit wallet allowance refresh, set `SignatureType` to `SignatureTypePoly1271`:

```go
var out clob.BalanceAllowanceResponse

err := client.UpdateBalanceAllowance(ctx, clob.BalanceAllowanceParams{
    AssetType:     clob.AssetCollateral,
    SignatureType: clob.SignatureTypePoly1271,
}, &out)
```

For conditional tokens:

```go
err := client.UpdateBalanceAllowance(ctx, clob.BalanceAllowanceParams{
    AssetType:     clob.AssetConditional,
    TokenID:       tokenID,
    SignatureType: clob.SignatureTypePoly1271,
}, &out)
```

Keep `SignatureType` as a value field, not a pointer, unless the public API is intentionally changed.

## Relayer and Deposit Wallets

`relayer/` owns the wire types and raw `/submit` / `/nonce` API interaction. Chain-aware contract defaults belong in `clob/contracts.go`.

Do not hard-code deposit-wallet factory addresses in `relayer/`. Use `Contracts(chainID).DepositWalletFactory` from the `clob` package.

Current relayer transaction types:

```text
PROXY
SAFE
WALLET-CREATE
WALLET
```

`SubmitTransactionRequest.SignatureParams` should be a pointer:

```go
SignatureParams *relayer.SignatureParams `json:"signatureParams,omitempty"`
```

This is important because `WALLET-CREATE` and `WALLET` requests must omit `signatureParams`.

### Deploy a deposit wallet

Use the CLOB wrapper so the factory comes from `Contracts(chainID)`:

```go
var out relayer.SubmitTransactionResponse

err := client.DeployDepositWallet(ctx, &out)
```

### Generic deposit-wallet batch

For non-CTF wallet actions, use the deposit-wallet batch helper:

```go
var out relayer.SubmitTransactionResponse

err := client.DepositWalletBatch(ctx, &clob.DepositWalletBatchArgs{
    DepositWallet: depositWallet.Hex(),
    Deadline:      "1760000000",
    Calls: []relayer.DepositWalletCall{
        {
            Target: someContract.Hex(),
            Value:  "0",
            Data:   "0x...",
        },
    },
}, &out)
```

For CTF split / merge / redeem, prefer the CTF convenience methods below instead of manually building a generic batch.

## CTF Helpers

The CTF helpers have three execution modes:

| Mode                        | API                                                                                                                                       |
| --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Direct RPC transaction      | `SplitPosition`, `MergePositions`, `RedeemPositions`, `RedeemNegRisk`                                                                     |
| Legacy relayer SAFE/PROXY   | `SplitPositionRelayer`, `MergePositionsRelayer`, `RedeemPositionsRelayer`, `RedeemNegRiskRelayer`                                         |
| Deposit wallet WALLET batch | `SplitPositionWithDepositWallet`, `MergePositionsWithDepositWallet`, `RedeemPositionsWithDepositWallet`, `RedeemNegRiskWithDepositWallet` |

### Prefer convenience methods for deposit-wallet CTF operations

```go
var out relayer.SubmitTransactionResponse

err := client.SplitPositionWithDepositWallet(ctx,
    &clob.SplitPositionRequest{
        CollateralToken:    clob.Collateral,
        ParentCollectionID: common.Hash{},
        ConditionID:        conditionID,
        Partition:          clob.BinaryPartition(),
        Amount:             big.NewInt(1_000_000),
    },
    &clob.DepositWalletCTFArgs{
        DepositWallet: depositWallet.Hex(),
        Deadline:      "1760000000",
    },
    &out,
)
```

Merge:

```go
err := client.MergePositionsWithDepositWallet(ctx,
    &clob.MergePositionsRequest{
        CollateralToken:    clob.Collateral,
        ParentCollectionID: common.Hash{},
        ConditionID:        conditionID,
        Partition:          clob.BinaryPartition(),
        Amount:             big.NewInt(1_000_000),
    },
    &clob.DepositWalletCTFArgs{
        DepositWallet: depositWallet.Hex(),
        Deadline:      "1760000000",
    },
    &out,
)
```

Redeem resolved positions:

```go
err := client.RedeemPositionsWithDepositWallet(ctx,
    &clob.RedeemPositionsRequest{
        CollateralToken:    clob.Collateral,
        ParentCollectionID: common.Hash{},
        ConditionID:        conditionID,
        IndexSets:          clob.BinaryPartition(),
    },
    &clob.DepositWalletCTFArgs{
        DepositWallet: depositWallet.Hex(),
        Deadline:      "1760000000",
    },
    &out,
)
```

Neg-risk redeem:

```go
err := client.RedeemNegRiskWithDepositWallet(ctx,
    &clob.RedeemNegRiskRequest{
        ConditionID: conditionID,
        Amounts: []*big.Int{
            big.NewInt(1_000_000),
            big.NewInt(2_000_000),
        },
    },
    &clob.DepositWalletCTFArgs{
        DepositWallet: depositWallet.Hex(),
        Deadline:      "1760000000",
    },
    &out,
)
```

### API boundary

Keep these entry points separate:

```text
BuildCTFRelayerRequest / SubmitCTFRelayerTransaction
  SAFE / PROXY only

CTFDepositWalletTransactionRequest / SubmitCTFDepositWalletTransaction
  deposit-wallet WALLET only
```

Do not add `WALLET` support to `CTFRelayerArgs`. If a user tries `NonceTypeWallet` or `NonceTypeWalletCreate` on the legacy CTF relayer path, return a clear error directing them to the deposit-wallet CTF API.

### CTF validation

CTF transaction builders must validate inputs before ABI packing. Never let invalid requests reach `abi.Pack` and trigger reflect panics.

Validate at least:

```text
out != nil
collateral token required where applicable
condition id required
partition / indexSets / amounts non-empty
amounts non-nil and positive
```

## OrderBuilder

Use `OrderBuilder` for trading whenever possible. It handles price conversion, tick-size validation, default signature type resolution, and signing.

```go
b := clob.NewOrderBuilder(client)

resp, err := b.CreateAndPostOrderForToken(ctx, clob.OrderArgsV2{
    TokenID: "token-id",
    Price:   "0.50",
    Size:    "10.0",
    Side:    clob.Buy,
}, clob.GTC, nil)
```

Manual advanced build:

```go
args := clob.OrderArgsV2{
    TokenID: "token-id",
    Price:   "0.50",
    Size:    "10.0",
    Side:    clob.Buy,
}
opts := clob.CreateOrderOptions{TickSize: "0.01", NegRisk: false}

order, err := b.BuildOrder(args, opts)
```

Market order semantics:

```text
BUY  market order Amount = USDC to spend
SELL market order Amount = shares to sell
```

The Go market-order amount calculation intentionally may diverge from `py-clob-client-v2` where tests document that divergence. Do not blindly change `computeMarketOrderAmounts` just to match Python without checking the intended Go semantics.

## Pagination

Pagination uses `Page[T]` with `next_cursor` and `limit`.

`GetOpenOrdersPage` returns one page. Legacy `GetOpenOrders` should be treated as current-page convenience behavior only if documented that way in code comments.

## Key Dependencies

- `github.com/ethereum/go-ethereum` — EIP-712 signing, hash signing, Polygon RPC, ABI encoding.
- No code generation.
- No migrations.
- No binaries.

## Tests

Run all tests offline:

```bash
go test -v ./...
```

Important test areas:

| Area                               | Files / coverage                                                                          |
| ---------------------------------- | ----------------------------------------------------------------------------------------- |
| Auth                               | `clob/auth_test.go`                                                                       |
| Client endpoints and flexible JSON | `clob/client_test.go`, `shared/flex_test.go`                                              |
| Order building and signing         | `clob/order_builder_test.go`, `clob/sign_order_test.go`, golden vector tests              |
| Deposit wallet order signing       | `clob/deposit_wallet_signing_test.go`                                                     |
| Deposit wallet relayer / CTF       | `clob/deposit_wallet*_test.go`, `relayer/types_test.go`, `relayer/deposit_wallet_test.go` |
| CTF calldata and safety            | `clob/ctf*_test.go`                                                                       |
| Relayer SAFE/PROXY                 | `relayer/client_test.go`, `relayer/proxy_test.go`, `relayer/safe_test.go`                 |

All unit tests should use mocks or `httptest.NewServer`; do not require live Polymarket APIs or live RPC for normal CI.

## Golden Vectors

Golden vectors live under:

```text
testdata/golden/py-clob-client-v2/
```

They are generated from the official `py-clob-client-v2` reference implementation.

Use them for:

```text
CLOB v2 order signatures
SAFE / PROXY relayer signing
POLY_1271 deposit-wallet order signatures
```

Deposit-wallet WALLET batch signing should also get a golden vector if an official reference generator is available. Until then, tests only prove local structure and signing stability, not byte-for-byte parity with an official SDK.

## Conventions

- JSON serialization uses shared scalar types to handle Polymarket's string-encoded decimals.
- URL query parameters use struct tags `` `url:"param_name,omitempty"` `` and reflection helpers.
- Do not export or recommend imports from `internal/` packages in docs or examples.
- Keep chain-specific contract addresses in `clob/contracts.go`.
- Keep relayer wire schema in `relayer/types.go`.
- Prefer CTF deposit-wallet convenience methods over raw generic batch examples.
- Avoid changing public API shapes from values to pointers unless the semantic difference matters.
- Every public package should have a `doc.go` with a one-line description.

## Release

GoReleaser is configured with `builds: skip: true`. Tagging a release creates versioned module archives only.
