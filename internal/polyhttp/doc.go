// Package polyhttp provides the shared HTTP client used by all Polymarket API
// packages (clob, data, gamma, relayer, bridge).
//
// # Authentication Levels
//
// Polymarket endpoints use three auth tiers:
//
//	AuthNone (0) — public endpoints, no headers
//	AuthL1   (1) — EIP-712 wallet-signed headers (e.g. CreateAPIKey)
//	AuthL2   (2) — full trading: requires both a Signer and Credentials
//
// The caller passes the desired AuthLevel to GetJSON, PostJSON, DeleteJSON,
// or DoJSON. If Client.Headers (a HeaderFunc) is non-nil, it is invoked with the
// level to produce auth headers injected into every request.
//
// # Client Usage
//
// The Client struct holds BaseURL, an http.Client, UserAgent, and the optional
// HeaderFunc callback. Construct it directly and pass it to higher-level packages.
//
// # Request Pipeline
//
// GetJSON/PostJSON/DeleteJSON → DoJSON → marshal body (JSON/[]byte/string) →
// build request → set Accept/Content-Type/User-Agent → call HeaderFunc for auth →
// execute → on non-2xx return *APIError → otherwise unmarshal JSON into out (with
// special handling for *int64 and *string responses).
//
// # Query Parameters
//
// Callers pass url.Values to DoJSON/GetJSON. At the package level, query structs
// use `url:"param"` tags and are reflected into values by the caller before
// constructing the final query map.
package polyhttp
