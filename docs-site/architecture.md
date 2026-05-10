# Architecture

Polymarket Client is a pure-Go SDK wrapping five distinct Polymarket HTTP APIs and one WebSocket API, all running on the Polygon blockchain. The library has no binaries — it is consumed as a Go module by applications.

## System Context (C4 Level 1)

```mermaid
graph TB
    subgraph "Your Application"
        APP["Your Go Application\n(uses polymarket-client)"]
    end

    subgraph "Polymarket Ecosystem"
        CLOB["CLOB v2 API\nclob.polymarket.com\nOrder matching"]
        RELAYER["Relayer API\nrelayer-v2.polymarket.com\nOn-chain tx submission"]
        DATA["Data API\ndata-api.polymarket.com\nPositions, trades, activity"]
        GAMMA["Gamma API\ngamma-api.polymarket.com\nMarkets, events, search"]
        BRIDGE["Bridge API\nbridge-api.polymarket.com\nCross-chain bridging"]
        WS["WebSocket\nws-subscriptions-clob.polymarket.com\nMarket and user streams"]
    end

    subgraph "Blockchain"
        POLY["Polygon\nChain ID 137\nERC-20 (USDC), CTF tokens"]
        RPC["Polygon RPC\npolygon-rpc.com\nOn-chain reads"]
    end

    subgraph "Auth Infrastructure"
        EIP["EIP-712 Signing\nWallet signatures\nHMAC secrets"]
    end

    APP -->|HTTP REST| CLOB
    APP -->|HTTP REST| RELAYER
    APP -->|HTTP REST| DATA
    APP -->|HTTP REST| GAMMA
    APP -->|HTTP REST| BRIDGE
    APP -->|WebSocket| WS

    APP -->|EIP-712 + HMAC| EIP
    EIP -->|Signatures| CLOB
    EIP -->|Signatures| RELAYER

    CLOB -->|CTF transactions| RELAYER
    RELAYER -->|Submit tx| POLY
    APP -->|On-chain reads| RPC
```

## Container Diagram (C4 Level 2)

```mermaid
graph TB
    subgraph "polymarket-client Library"
        subgraph "Public Packages"
            CLOBPKG["clob/\nCLOB v2 Client\n65 methods"]
            WSPKG["clob/ws/\nWebSocket Client\nLive streams"]
            RELAYERPKG["relayer/\nRelayer Client\n8 methods"]
            DATAPKG["data/\nData API Client\n14 methods"]
            GAMMAPKG["gamma/\nGamma API Client\n20 methods"]
            BRIDGEPKG["bridge/\nBridge API Client\n2 methods"]
            TYPESPKG["shared/\nShared scalar types\nString, Int, Float64, Time"]
        end

        subgraph "Internal Packages"
            POLYHTTP["polyhttp/\nShared HTTP client\nAuth level routing"]
            POLYAUTH["polyauth/\nEIP-712 signing\nL1/L2 header generation"]
        end

        CLOBPKG -->|uses| POLYHTTP
        CLOBPKG -->|uses| POLYAUTH
        CLOBPKG -->|submits via| RELAYERPKG
        WSPKG -->|uses| POLYAUTH
        RELAYERPKG -->|uses| POLYAUTH
        DATAPKG -->|uses| POLYHTTP
        GAMMAPKG -->|uses| POLYHTTP
        BRIDGEPKG -->|uses| POLYHTTP
        CLOBPKG -->|uses types| TYPESPKG
    end

    style POLYHTTP fill:#ffd,#stroke:#666
    style POLYAUTH fill:#ffd,#stroke:#666
    style TYPESPKG fill:#ffd,#stroke:#666
```

## Authentication Flow

Three-tier authentication with increasing privilege:

```mermaid
sequenceDiagram
    participant App as Your Application
    participant CLOB as CLOB Client
    participant Auth as polyauth
    participant API as Polymarket API

    %% AuthNone
    rect rgb(220, 255, 220)
    Note over App,API: AuthNone (0) — Public endpoints
    App->>CLOB: GetMarkets(ctx, cursor)
    CLOB->>API: GET /markets\n(no auth headers)
    API-->>CLOB: 200 OK
    CLOB-->>App: []Market
    end

    %% AuthL1
    rect rgb(255, 255, 200)
    Note over App,API: AuthL1 (1) — Wallet-signed (e.g. CreateAPIKey)
    App->>CLOB: CreateAPIKey(ctx, nonce)
    CLOB->>Auth: L1Headers(signer, chainID, ts, nonce)
    Auth-->>CLOB: POLY_ADDRESS, POLY_SIGNATURE, POLY_TIMESTAMP, POLY_NONCE
    CLOB->>API: POST /auth/api-key\n(EIP-712 signed headers)
    API-->>CLOB: 200 OK → {api_key, secret, passphrase}
    CLOB-->>App: Credentials
    end

    %% AuthL2
    rect rgb(255, 200, 200)
    Note over App,API: AuthL2 (2) — Full trading (orders, trades)
    App->>CLOB: PostOrder(ctx, req)
    CLOB->>Auth: L2Headers(signer, apiKey, secret, passphrase, ts, method, path, body)
    Auth-->>CLOB: POLY_ADDRESS, POLY_SIGNATURE, POLY_TIMESTAMP,\nPOLY_API_KEY, POLY_PASSPHRASE, POLY_NONCE
    CLOB->>API: POST /order\n(full auth headers)
    API-->>CLOB: 200 OK → {success}
    CLOB-->>App: PostOrderResponse
    end
```

## Package Dependency Graph

```mermaid
graph LR
    subgraph "Public API Surface"
        A["clob/"]
        B["clob/ws/"]
        C["relayer/"]
        D["data/"]
        E["gamma/"]
        F["bridge/"]
        G["shared/"]
    end

    subgraph "Internal"
        H["polyhttp/"]
        I["polyauth/"]
    end

    subgraph "External"
        J["go-ethereum\ncrypto, eip712"]
    end

    A --> C
    A --> H
    A --> I
    B --> I
    C --> H
    C --> I
    D --> H
    E --> H
    F --> H
    A --> G
    I --> J
```

## Request Processing Pipeline

Every HTTP request flows through this pipeline:

```mermaid
flowchart LR
    A["Client method\n(e.g. GetMarkets)"] --> B["Build URL + query params"]
    B --> C["Marshal body\n(if POST/DELETE)"]
    C --> D["Create http.Request"]
    D --> E{"Auth level?"}

    E -->|0 (None)| F["No auth headers"]
    E -->|1 (L1)| G["polyauth.L1Headers()\nEIP-712 signature"]
    E -->|2 (L2)| H["polyauth.L2Headers()\nAPI key + HMAC + wallet sig"]

    F --> I["Execute request"]
    G --> I
    H --> I
    I --> J{"Status OK?"}
    J -->|2xx| K["Unmarshal JSON\n→ out value"]
    J -->|4xx/5xx| L["Return APIError\nwith status + body"]
```

## Key Type System

Polymarket returns numeric values as both raw JSON numbers **and** decimal strings. The SDK handles this via custom types:

```mermaid
classDiagram
    class Float64 {
        +float64 Value
        +UnmarshalJSON(data)
        +MarshalJSON()
    }

    class String {
        +string Value
        +UnmarshalJSON(data)
        +MarshalJSON()
    }

    class Page~T~ {
        +[]T Results
        +string NextCursor
        +*int64 Limit
    }

    class APISecret {
        +string Encoded
        +DecodeAPISecret() []byte
    }

    Float64 --|> json.Marshaler
    String --|> json.Marshaler
    Page : Generic pagination
    APISecret : Base64-decode for HMAC
```

- **`Float64`** — always unmarshals as `float64`, accepts both `"0.50"` and `0.50`
- **`String`** — accepts strings, numbers, and booleans while preserving a stable string form
- **`Page[T]`** — generic pagination with `NextCursor`, used throughout CLOB and rewards APIs

## WebSocket Architecture

```mermaid
stateDiagram-v2
    [*] --> Initializing
    Initializing --> Connecting: ws.Dial()
    Connecting --> Authenticate: connection opened
    Authenticate --> Subscribing: auth success
    Authenticate --> Disconnected: auth failed
    Subscribing --> Receiving: subscription confirmed
    Receiving --> Receiving: order book update
    Receiving --> Receiving: order fill
    Receiving --> Receiving: order status change
    Receiving --> Reconnecting: connection lost
    Receiving --> Reconnecting: stale timeout
    Reconnecting --> Subscribing: reconnected
    Disconnected --> [*]
```

The WebSocket client (`clob/ws`) uses `github.com/coder/websocket`, maintains a read loop in a goroutine, and sends decoded updates through `client.Events()`. Reconnection is automatic by default with exponential backoff, and stored subscriptions are replayed after each successful reconnect. Optional stale detection can force reconnect when the socket stays open but no messages arrive before the configured timeout.

## CLOB–Relayer Integration

The CLOB client can optionally be configured with a Relayer client for on-chain CTF (Conditional Token Framework) operations:

```mermaid
sequenceDiagram
    participant User as User App
    participant CLOB as CLOB Client
    participant Relayer as Relayer Client
    participant Polygon as Polygon Network

    User->>CLOB: SubmitRelayerTransaction(req)
    CLOB->>CLOB: Check relay client is configured
    CLOB->>Relayer: SubmitTransaction(ctx, req)
    Relayer->>Relayer: Add RELAYER_API_KEY header
    Relayer->>Polygon: POST /submit
    Polygon-->>Relayer: {hash, success}
    Relayer-->>CLOB: SubmitTransactionResponse
    CLOB-->>User: Response
```

This is configured via `clob.WithRelayerClient(relayerClient)` or `clob.WithRelayerSubmitter(submitter)`. Without it, `SubmitRelayerTransaction` returns `"polymarket: relayer client is not configured"`.
