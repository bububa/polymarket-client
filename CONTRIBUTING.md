# Contributing to polymarket-client

Thank you for contributing to the Polymarket Go SDK. This project is a **library only** -- no binaries, no `main` package. All contributions should keep it that way.

## Quick Start

```bash
go build -v ./...
go test -v ./...
go mod tidy
```

CI uses Go >= 1.23.0. `go.mod` declares 1.22, but prefer 1.23+ locally.

## Testing Conventions

- All 5 test files use `httptest.NewServer` -- no live API calls required.
- Run the full suite offline: `go test -v ./...`
- New tests **must** follow the same pattern. Use `httptest` to mock HTTP, never hit production endpoints.
- Table-driven tests are preferred. Use subtests for per-case isolation.

## Coding Standards

### Client Construction

All public packages (`clob/`, `data/`, `gamma/`, `relayer/`, `bridge/`) use `NewClient(host, opts...)` with functional options. New clients must follow the same pattern:

```go
func NewClient(host string, opts ...Option) *Client { ... }
```

### Auth Levels

- **AuthNone (0)** -- public endpoints (market data, orderbook snapshots)
- **AuthL1 (1)** -- EIP-712 signed L1 headers (e.g. `CreateAPIKey`)
- **AuthL2 (2)** -- requires both `Signer` **and** `Credentials` (API key, secret, passphrase). All order/trade endpoints.

L2 auth without `Credentials` must return `"api credentials are required for level 2 authenticated request"`.

### JSON Serialization

Polymarket returns decimals as JSON strings. Use the custom scalar types from `shared/` (`shared.Float64`, `shared.Int`, `shared.String`, `shared.Time`). Never use raw `float64` or `string` for numeric API fields.

### Pagination

All paginated responses use `Page[T]` with `next_cursor` and `limit` fields. Match this shape.

### Query Parameters

Struct fields used as URL query params use `` `url:"param,omitempty"` `` tags and are serialized via reflection (`clob/client.go:values()`). Follow this existing pattern rather than adding new serialization libraries.

### Documentation

- Every **exported** type, function, method, and constant must have a godoc comment.
- Each public package must have a `doc.go` with a one-line package description.
- Examples in `doc.go` are strongly preferred.

## Adding New Endpoints

Mirror the existing patterns in `clob/client.go`:

1. Define request/response structs in the appropriate `*_types.go` file.
2. Add a method on `*Client` with a clear verb-noun name (`GetMarket`, `PlaceOrder`, etc.).
3. Use the appropriate `AuthLevel` in the `polyhttp.RequestConfig`.
4. Add a table-driven test in the corresponding `*_test.go` file.
5. No new dependencies.

## Adding New Packages

1. Create the package directory at the repo root (or under `internal/` if private).
2. Add `doc.go` with a one-line description.
3. Implement `NewClient(host, opts...)` with functional options.
4. All public types and functions must be documented.
5. Include at least one test using `httptest.NewServer`.

## Dependencies

- The only external dependency is `github.com/ethereum/go-ethereum`.
- Adding new dependencies requires justification in the PR description.
- Prefer the standard library over third-party packages.

## Pull Request Checklist

- [ ] `go build -v ./...` passes
- [ ] `go test -v ./...` passes
- [ ] `go mod tidy` is clean (no unused imports, go.sum is up to date)
- [ ] All exported symbols have godoc comments
- [ ] New tests use `httptest.NewServer` (no live API)
- [ ] Followed existing code style (gofmt, functional options, shared scalars)
- [ ] PR description explains *what* and *why*

## Branch Naming

Use descriptive, kebab-case names:

```
feature/add-rfq-endpoint
fix/orderbook-pagination
refactor/auth-header-generation
```

## Commit Messages

Use conventional commits:

```
feat(clob): add GetRFQPrices endpoint
fix(data): handle negative string-encoded decimals
docs(gamma): add package examples to doc.go
```

Scope in parentheses should match the package name (`clob`, `data`, `gamma`, `relayer`, `bridge`, `shared`, `internal`).
