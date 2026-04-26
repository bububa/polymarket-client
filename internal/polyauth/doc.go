// Package polyauth provides EIP-712 signing and authentication header generation
// for the Polymarket CLOB API.
//
// # Signer
//
// Signer wraps a secp256k1 ECDSA private key and produces EIP-712 typed-data
// signatures. Create one via ParsePrivateKey(hexKey) or from an existing
// *ecdsa.PrivateKey. Use Address() to get the associated wallet address.
//
// # L1 Headers — API Key Creation
//
// L1Headers returns POLY_ADDRESS, POLY_SIGNATURE, POLY_TIMESTAMP, and
// POLY_NONCE headers for wallet-signed requests used to create or derive
// API keys (CreateAPIKey / DeriveAPIKey endpoints).
//
// # L2 Headers — Full Trading Auth
//
// L2Headers returns headers for all order/trade endpoints. It requires an
// API key, a decoded HMAC secret (see DecodeAPISecret), a passphrase, and
// the request method/path/body to compute the HMAC signature.
//
// # Key Handling
//
// GenerateKey produces a new hex-encoded secp256k1 private key.
// ParsePrivateKey parses a hex-encoded key (with or without "0x" prefix)
// into a Signer. DecodeAPISecret decodes a URL-safe base64 Polymarket
// API secret into raw bytes for HMAC signing.
package polyauth
