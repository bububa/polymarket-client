# clob Package

Go client for Polymarket CLOB v2 API.

**Default Host:** `https://clob.polymarket.com`  
**Methods:** 65+  
**Auth:** Depends on endpoint (0, 1, or 2)

## Market Data (AuthNone)

| Method | Description |
|---|---|
| `GetOk` | Health check (`GET /ok`) |
| `GetVersion` | API version (`GET /version`) |
| `GetServerTime` | Server timestamp (`GET /time`) |
| `GetMarkets` | Paginated markets (`GET /markets`) |
| `GetSimplifiedMarkets` | Simplified market list (`GET /simplified-markets`) |
| `GetClobMarketInfo` | Single market details (`GET /clob-markets/:id`) |
| `GetMarketByToken` | Market by token ID (`GET /markets-by-token/:id`) |
| `GetOrderBook` | Order book for a token (`GET /book`) |
| `GetOrderBooks` | Batch order books (`POST /books`) |
| `GetMidpoint` | Midpoint price (`GET /midpoint`) |
| `GetMidpoints` | Batch midpoints (`POST /midpoints`) |
| `GetPrice` | Price by side (`GET /price`) |
| `GetPrices` | Batch prices (`POST /prices`) |
| `GetSpread` | Bid-ask spread (`GET /spread`) |
| `GetSpreads` | Batch spreads (`POST /spreads`) |
| `GetLastTradePrice` | Last trade (`GET /last-trade-price`) |
| `GetLastTradesPrices` | Batch last trades (`POST /last-trades-prices`) |
| `GetTickSize` | Tick size (`GET /tick-size`) |
| `GetNegRisk` | Neg-risk flag (`GET /neg-risk`) |
| `GetFeeRate` | Fee rate (`GET /fee-rate`) |

## Auth & Keys

| Method | Auth | Description |
|---|---|---|
| `CreateAPIKey` | L1 | Create new API key |
| `DeriveAPIKey` | L1 | Derive existing API key |
| `GetAPIKeys` | L2 | List all API keys |
| `DeleteAPIKey` | L2 | Delete current API key |
| `GetClosedOnlyMode` | L2 | Check ban status |

## Orders & Trading (AuthL2)

| Method | Description |
|---|---|
| `PostOrder` | Place single order (`POST /order`) |
| `PostOrders` | Place batch orders (`POST /orders`) |
| `CancelOrder` | Cancel by ID (`DELETE /order`) |
| `CancelOrders` | Cancel multiple (`DELETE /orders`) |
| `CancelAll` | Cancel all (`DELETE /cancel-all`) |
| `CancelMarketOrders` | Cancel by market (`DELETE /cancel-market-orders`) |
| `GetOrder` | Get order by ID |
| `GetOpenOrders` | List open orders |
| `GetPreMigrationOrders` | Pre-migration orders |
| `GetTrades` | User trade history |
| `IsOrderScoring` | Check order scoring |
| `AreOrdersScoring` | Batch scoring check |
| `SendHeartbeat` | Send heartbeat (`POST /heartbeats`) |

## RFQ (AuthL2)

| Method | Description |
|---|---|
| `CreateRFQRequest` | Create RFQ |
| `CancelRFQRequest` | Cancel RFQ request |
| `GetRFQRequests` | List RFQs |
| `CreateRFQQuote` | Create RFQ quote |
| `CancelRFQQuote` | Cancel RFQ quote |
| `GetRFQuoterQuotes` | Quoter's quotes |
| `GetRFQRequesterQuotes` | Requester's quotes |
| `GetRFQBestQuote` | Best quote |
| `AcceptRFQRequest` | Accept RFQ |
| `ApproveRFQQuote` | Approve quote |

## Builder API (AuthL2)

| Method | Description |
|---|---|
| `CreateBuilderAPIKey` | Create builder key |
| `GetBuilderAPIKeys` | List builder keys |
| `RevokeBuilderAPIKey` | Revoke builder key |
| `GetBuilderTrades` | Builder trade history |
| `GetBuilderFeeRate` | Builder fee rate (public) |
