package clob

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// validatePriceTicks checks that price is an exact multiple of the given tick size.
// Empty tickSize means "no validation". Non-empty tickSize must be positive.
func validatePriceTicks(price, tickSize string) error {
	if tickSize == "" || price == "" {
		return nil
	}
	p, err := parseRat(price, "price")
	if err != nil {
		return err
	}
	t, err := parseRat(tickSize, "tickSize")
	if err != nil {
		return err
	}
	if t.Sign() <= 0 {
		return fmt.Errorf("polymarket: tickSize must be positive, got %q", tickSize)
	}
	if !isExactMultiple(p, t) {
		return fmt.Errorf("polymarket: price %q is not aligned to tick size %q", price, tickSize)
	}
	return nil
}

func isExactMultiple(p, t *big.Rat) bool {
	div := new(big.Rat).Quo(p, t)
	return div.IsInt()
}

// ValidateBytes32Hex checks that v is empty or a valid 0x-prefixed bytes32 hex string.
func ValidateBytes32Hex(name, v string) error {
	if v == "" {
		return nil
	}
	if !strings.HasPrefix(v, "0x") && !strings.HasPrefix(v, "0X") {
		return fmt.Errorf("polymarket: %s must be 0x-prefixed hex, got %q", name, v)
	}
	hex := v[2:]
	if len(hex) != 64 {
		return fmt.Errorf("polymarket: %s must be 64 hex chars (bytes32), got %d chars", name, len(hex))
	}
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("polymarket: %s contains invalid hex char %q", name, c)
		}
	}
	return nil
}

func ValidateHexAddress(name, value string, allowEmpty bool) error {
	if value == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("polymarket: %s is required", name)
	}

	if !common.IsHexAddress(value) {
		return fmt.Errorf("polymarket: %s must be a valid hex address", name)
	}

	return nil
}

func validateUint256String(name string, value String, required bool, positive bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("polymarket: %s is required", name)
		}
		return nil
	}

	n, ok := new(big.Int).SetString(string(value), 10)
	if !ok {
		return fmt.Errorf("polymarket: %s must be a base-10 integer", name)
	}
	if n.Sign() < 0 {
		return fmt.Errorf("polymarket: %s must be non-negative", name)
	}
	if positive && n.Sign() == 0 {
		return fmt.Errorf("polymarket: %s must be positive", name)
	}

	// 2^256 - 1
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	if n.Cmp(max) > 0 {
		return fmt.Errorf("polymarket: %s overflows uint256", name)
	}

	return nil
}
