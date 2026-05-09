package polyauth

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const clobAuthMessage = "This message attests that I control the given wallet"

type Signer struct {
	key     *ecdsa.PrivateKey
	address common.Address
}

// GenerateKey creates a new secp256k1 private key and returns it as hex.
func GenerateKey() (string, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(crypto.FromECDSA(key)), nil
}

// ParsePrivateKey parses a hex-encoded secp256k1 private key.
func ParsePrivateKey(raw string) (*Signer, error) {
	raw = strings.TrimPrefix(strings.TrimSpace(raw), "0x")
	key, err := crypto.HexToECDSA(raw)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return &Signer{key: key, address: crypto.PubkeyToAddress(key.PublicKey)}, nil
}

// Address returns the Ethereum address for the signer.
func (s *Signer) Address() common.Address { return s.address }

// PrivateKey returns the underlying ECDSA private key.
func (s *Signer) PrivateKey() *ecdsa.PrivateKey { return s.key }

// SignTypedData signs EIP-712 typed data with signer.
func SignTypedData(signer *Signer, typedData apitypes.TypedData) (string, error) {
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return "", fmt.Errorf("build typed data digest: %w", err)
	}
	return SignHash(signer, digest)
}

// SignHash signs a 32-byte digest and returns a 0x-prefixed Ethereum signature.
func SignHash(signer *Signer, digest []byte) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("signer is nil")
	}
	if len(digest) != 32 {
		return "", fmt.Errorf("digest must be 32 bytes, got %d", len(digest))
	}
	signature, err := crypto.Sign(digest, signer.key)
	if err != nil {
		return "", fmt.Errorf("sign hash: %w", err)
	}
	signature[64] += 27
	return "0x" + hex.EncodeToString(signature), nil
}

// L1Headers returns Polymarket L1 headers for API-key creation and derivation.
func L1Headers(signer *Signer, chainID, timestamp, nonce int64) (map[string]string, error) {
	signature, err := signer.signClobAuth(chainID, timestamp, nonce)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"POLY_ADDRESS":   signer.address.Hex(),
		"POLY_SIGNATURE": signature,
		"POLY_TIMESTAMP": strconv.FormatInt(timestamp, 10),
		"POLY_NONCE":     strconv.FormatInt(nonce, 10),
	}, nil
}

// L2Headers returns Polymarket L2 headers for API-key authenticated requests.
func L2Headers(signer *Signer, key string, secret []byte, passphrase string, timestamp int64, method, path string, body []byte) (map[string]string, error) {
	signature := HMACSignatureBytes(secret, timestamp, method, path, body)
	return map[string]string{
		"POLY_ADDRESS":    signer.address.Hex(),
		"POLY_SIGNATURE":  signature,
		"POLY_TIMESTAMP":  strconv.FormatInt(timestamp, 10),
		"POLY_API_KEY":    key,
		"POLY_PASSPHRASE": passphrase,
	}, nil
}

func (s *Signer) signClobAuth(chainID, timestamp, nonce int64) (string, error) {
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"ClobAuth": {
				{Name: "address", Type: "address"},
				{Name: "timestamp", Type: "string"},
				{Name: "nonce", Type: "uint256"},
				{Name: "message", Type: "string"},
			},
		},
		PrimaryType: "ClobAuth",
		Domain: apitypes.TypedDataDomain{
			Name:    "ClobAuthDomain",
			Version: "1",
			ChainId: ethmath.NewHexOrDecimal256(chainID),
		},
		Message: apitypes.TypedDataMessage{
			"address":   s.address.Hex(),
			"timestamp": strconv.FormatInt(timestamp, 10),
			"nonce":     strconv.FormatInt(nonce, 10),
			"message":   clobAuthMessage,
		},
	}
	return SignTypedData(s, typedData)
}

// DecodeAPISecret decodes a URL-safe base64 Polymarket API secret.
func DecodeAPISecret(secret string) ([]byte, error) {
	normalized, err := normalizeBase64URL(secret)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.URLEncoding.DecodeString(normalized)
	if err == nil {
		return decoded, nil
	}
	std := strings.NewReplacer("-", "+", "_", "/").Replace(normalized)
	decoded, err = base64.StdEncoding.DecodeString(std)
	if err != nil {
		return nil, fmt.Errorf("decode API secret: %w", err)
	}
	return decoded, nil
}

// HMACSignature signs a request with a base64-encoded Polymarket API secret.
func HMACSignature(secret string, timestamp int64, method, requestPath string, body []byte) (string, error) {
	decoded, err := DecodeAPISecret(secret)
	if err != nil {
		return "", err
	}
	return HMACSignatureBytes(decoded, timestamp, method, requestPath, body), nil
}

// HMACSignatureBytes signs a request with a decoded Polymarket API secret.
func HMACSignatureBytes(secret []byte, timestamp int64, method, requestPath string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	var buf [24]byte
	_, _ = mac.Write(strconv.AppendInt(buf[:0], timestamp, 10))
	_, _ = io.WriteString(mac, method)
	_, _ = io.WriteString(mac, requestPath)
	if len(body) > 0 {
		_, _ = mac.Write(bytes.ReplaceAll(body, []byte("'"), []byte("\"")))
	}
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

func normalizeBase64URL(value string) (string, error) {
	value = strings.TrimSpace(value)
	switch len(value) % 4 {
	case 1:
		return "", fmt.Errorf("invalid base64url secret: length mod 4 == 1")
	case 2:
		value += "=="
	case 3:
		value += "="
	}
	return value, nil
}
