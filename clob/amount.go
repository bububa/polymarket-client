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

func computeMarketOrderAmounts(worstPrice, size string, side Side) (makerAmount, takerAmount string, err error) {
	return computeOrderAmounts(worstPrice, size, side)
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

func roundToTickSize(price, tickSize string) (string, error) {
	p, err := parseRat(price, "price")
	if err != nil {
		return "", err
	}
	t, err := parseRat(tickSize, "tickSize")
	if err != nil {
		return "", err
	}
	if t.Sign() <= 0 {
		return p.FloatString(6), nil
	}

	div := new(big.Rat).Quo(p, t)
	floorDiv := new(big.Int).Div(div.Num(), div.Denom())
	rounded := new(big.Rat).Mul(new(big.Rat).SetInt(floorDiv), t)
	s := rounded.FloatString(6)
	// strip unnecessary trailing zeros (keep at least 2 decimal places for tick-like prices)
	s = strings.TrimRight(s, "0")
	if dot := strings.LastIndex(s, "."); dot >= 0 && len(s)-dot <= 2 {
		s += "0" // keep "0.50" not "0.5"
	}
	return s, nil
}
