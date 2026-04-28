package clob

import (
	"fmt"
	"math/big"
	"strings"
)

const orderScale = 1_000_000

// computeOrderAmounts converts price and size into makerAmount/takerAmount strings
// at 6-decimal precision (Polymarket CLOB integer format).
//
// BUY:  makerAmount = price x size x 1e6 (USDC), takerAmount = size x 1e6 (tokens)
// SELL: makerAmount = size x 1e6 (tokens),   takerAmount = price x size x 1e6 (USDC)
func computeOrderAmounts(price, size string, side Side) (makerAmount, takerAmount string, err error) {
	p, err := parseRat(price, "price")
	if err != nil {
		return "", "", err
	}
	s, err := parseRat(size, "size")
	if err != nil {
		return "", "", err
	}

	if side != Buy && side != Sell {
		return "", "", fmt.Errorf("polymarket: invalid side %q", side)
	}
	if p.Sign() < 0 {
		return "", "", fmt.Errorf("polymarket: price must be >= 0, got %v", price)
	}
	if s.Sign() <= 0 {
		return "", "", fmt.Errorf("polymarket: size must be > 0, got %v", size)
	}

	monetary := new(big.Rat).Mul(p, s)
	scale := new(big.Int).SetInt64(orderScale)

	monetaryScaled := new(big.Rat).Mul(monetary, new(big.Rat).SetInt(scale))
	sizeScaled := new(big.Rat).Mul(s, new(big.Rat).SetInt(scale))

	makerInt := truncRat(monetaryScaled)
	takerInt := truncRat(sizeScaled)

	if side == Buy {
		return makerInt.String(), takerInt.String(), nil
	}
	return takerInt.String(), makerInt.String(), nil
}

// computeMarketOrderAmounts computes makerAmount/takerAmount for FOK/FAK market orders.
//
// BUY market order: amount is USDC (pUSD) the user wants to spend.
//
//	makerAmount = amount x 1e6         (USDC offered)
//	takerAmount = ceil(amount / price x 1e6)  (tokens wanted at worstPrice)
//
// SELL market order: amount is shares the user wants to sell.
//
//	makerAmount = amount x 1e6         (tokens offered)
//	takerAmount = ceil(amount x price x 1e6)  (USDC wanted)
//
// We ceil the takerAmount so the implied execution price never worsens the user's worstPrice.
func computeMarketOrderAmounts(price, amount string, side Side) (makerAmount, takerAmount string, err error) {
	p, err := parseRat(price, "price")
	if err != nil {
		return "", "", err
	}
	a, err := parseRat(amount, "amount")
	if err != nil {
		return "", "", err
	}

	if side != Buy && side != Sell {
		return "", "", fmt.Errorf("polymarket: invalid side %q", side)
	}
	if p.Sign() <= 0 {
		return "", "", fmt.Errorf("polymarket: market order price must be > 0, got %v", price)
	}
	if a.Sign() <= 0 {
		return "", "", fmt.Errorf("polymarket: market order amount must be > 0, got %v", amount)
	}

	scale := new(big.Int).SetInt64(orderScale)
	amtScaled := new(big.Rat).Mul(a, new(big.Rat).SetInt(scale))

	if side == Buy {
		// makerAmount = USDC (floor), takerAmount = USDC / price (ceil → protects worst price)
		makerInt := truncRat(amtScaled)
		takerAmt := new(big.Rat).Quo(a, p)
		takerScaled := new(big.Rat).Mul(takerAmt, new(big.Rat).SetInt(scale))
		return makerInt.String(), ceilRat(takerScaled).String(), nil
	}
	// seller: makerAmount = shares (floor), takerAmount = shares x price (ceil → protects worst price)
	makerInt := truncRat(amtScaled)
	takerAmt := new(big.Rat).Mul(a, p)
	takerScaled := new(big.Rat).Mul(takerAmt, new(big.Rat).SetInt(scale))
	return makerInt.String(), ceilRat(takerScaled).String(), nil
}

func parseRat(s, name string) (*big.Rat, error) {
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return nil, fmt.Errorf("polymarket: invalid %s %q", name, s)
	}
	return r, nil
}

func truncRat(r *big.Rat) *big.Int {
	return new(big.Int).Div(r.Num(), r.Denom())
}

func ceilRat(r *big.Rat) *big.Int {
	num := r.Num()
	den := r.Denom()
	q := new(big.Int).Div(num, den)
	rem := new(big.Int).Mod(num, den)
	if rem.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

// validatePriceTicks checks that price is an exact multiple of the given tick size.
// Empty tickSize means "no validation".
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
		return nil
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
