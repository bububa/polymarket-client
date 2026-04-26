# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.0.1] - 2026-04-25

### Changed

- Refactor to pre-allocation API and update CTF helpers

## [v1.0.0] - 2026-04-25

### Added

- Initial Polymarket Go SDK with full API coverage

#### Packages

- `clob/` — CLOB v2 API: orders, markets, positions, RFQ
- `clob/ws/` — WebSocket for live order book / order updates
- `relayer/` — Relayer API: submit signed transactions
- `data/` — Read-only market data API
- `gamma/` — Read-only gamma market-data API
- `bridge/` — Bridge API
- `shared/` — Shared scalar types (`String`, `Int`, `Float64`, `Time`)
- `internal/polyhttp/` — HTTP client with auth-level routing
- `internal/polyauth/` — EIP-712 signing, L1/L2 header generation

### Changed

- Update go-ethereum dependency (1.14.12 → 1.17.0)
- Update golang.org/x/crypto (0.35.0 → 0.45.0)
- Update consensys/gnark-crypto dependency
- Update go.opentelemetry.io/otel (1.39.0 → 1.41.0)
