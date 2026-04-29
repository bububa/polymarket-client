package clob

import (
	"testing"
)

func TestComputeOrderAmounts_Buy(t *testing.T) {
	tests := []struct {
		name                 string
		price                string
		size                 string
		wantMaker, wantTaker string
	}{
		{"simple_0.5", "0.5", "10", "5000000", "10000000"},
		{"full_price", "1.0", "100", "100000000", "100000000"},
		{"fractional_price", "0.75", "4", "3000000", "4000000"},
		{"tiny_size", "0.10", "0.5", "50000", "500000"},
		{"large_order", "0.67", "10000", "6700000000", "10000000000"},
		{"many_decimals_price", "0.0073", "100", "730000", "100000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMaker, gotTaker, err := computeOrderAmounts(tt.price, tt.size, Buy)
			if err != nil {
				t.Fatalf("computeOrderAmounts(%s, %s, Buy) error = %v", tt.price, tt.size, err)
			}
			if gotMaker != tt.wantMaker {
				t.Errorf("makerAmount = %s, want %s", gotMaker, tt.wantMaker)
			}
			if gotTaker != tt.wantTaker {
				t.Errorf("takerAmount = %s, want %s", gotTaker, tt.wantTaker)
			}
		})
	}
}

func TestComputeOrderAmounts_Sell(t *testing.T) {
	tests := []struct {
		name                 string
		price                string
		size                 string
		wantMaker, wantTaker string
	}{
		{"simple_0.5", "0.5", "10", "10000000", "5000000"},
		{"full_price", "1.0", "100", "100000000", "100000000"},
		{"fractional_price", "0.75", "4", "4000000", "3000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMaker, gotTaker, err := computeOrderAmounts(tt.price, tt.size, Sell)
			if err != nil {
				t.Fatalf("computeOrderAmounts(%s, %s, Sell) error = %v", tt.price, tt.size, err)
			}
			if gotMaker != tt.wantMaker {
				t.Errorf("makerAmount = %s, want %s", gotMaker, tt.wantMaker)
			}
			if gotTaker != tt.wantTaker {
				t.Errorf("takerAmount = %s, want %s", gotTaker, tt.wantTaker)
			}
		})
	}
}

func TestComputeOrderAmounts_Errors(t *testing.T) {
	for _, tt := range []struct {
		name  string
		price string
		size  string
		side  Side
	}{
		{"invalid_price", "abc", "10", Buy},
		{"invalid_size", "0.5", "xyz", Buy},
		{"negative_price", "-0.1", "10", Buy},
		{"zero_size", "0.5", "0", Buy},
		{"negative_size", "0.5", "-1", Buy},
		{"invalid_side", "0.5", "10", Side("INVALID")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := computeOrderAmounts(tt.price, tt.size, tt.side)
			if err == nil {
				t.Fatalf("expected error for %v", tt)
			}
		})
	}
}

func TestComputeMarketOrderAmounts_BUY(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		price                string
		amount               string
		wantMaker, wantTaker string
	}{
		{"buy_100usdc_at_0.5", "0.5", "100", "100000000", "200000000"},
		{"buy_50usdc_at_0.25", "0.25", "50", "50000000", "200000000"},
		{"buy_10usdc_at_1.0", "1.0", "10", "10000000", "10000000"},
		{"buy_5usdc_at_0.99_rounds_shares_to_4dp", "0.99", "5", "5000000", "5050600"},
		{"buy_usdc_rounds_down_to_2dp", "0.50", "5.009", "5000000", "10000000"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotMaker, gotTaker, err := computeMarketOrderAmounts(tt.price, tt.amount, Buy)
			if err != nil {
				t.Fatalf("computeMarketOrderAmounts(%s, %s, Buy) error = %v", tt.price, tt.amount, err)
			}
			if gotMaker != tt.wantMaker {
				t.Errorf("makerAmount = %s, want %s", gotMaker, tt.wantMaker)
			}
			if gotTaker != tt.wantTaker {
				t.Errorf("takerAmount = %s, want %s", gotTaker, tt.wantTaker)
			}
		})
	}
}

func TestComputeMarketOrderAmounts_SELL(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		price                string
		amount               string
		wantMaker, wantTaker string
	}{
		{"sell_200shares_at_0.45", "0.45", "200", "200000000", "90000000"},
		{"sell_50shares_at_0.75", "0.75", "50", "50000000", "37500000"},
		{"sell_10shares_at_1.0", "1.0", "10", "10000000", "10000000"},
		{"sell_100shares_at_0.333333_rounds_usdc_to_4dp", "0.333333", "100", "100000000", "33333300"},
		{"sell_shares_rounds_down_to_2dp", "0.50", "5.00009", "5000000", "2500000"},
		{"sell_fractional_dust_rounds_down_to_2dp", "0.01", "4.6153", "4610000", "46100"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotMaker, gotTaker, err := computeMarketOrderAmounts(tt.price, tt.amount, Sell)
			if err != nil {
				t.Fatalf("computeMarketOrderAmounts(%s, %s, Sell) error = %v", tt.price, tt.amount, err)
			}
			if gotMaker != tt.wantMaker {
				t.Errorf("makerAmount = %s, want %s", gotMaker, tt.wantMaker)
			}
			if gotTaker != tt.wantTaker {
				t.Errorf("takerAmount = %s, want %s", gotTaker, tt.wantTaker)
			}
		})
	}
}

func TestComputeMarketOrderAmounts_Errors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		price  string
		amount string
		side   Side
	}{
		{"zero_price", "0", "100", Buy},
		{"zero_amount", "0.5", "0", Buy},
		{"negative_amount", "0.5", "-10", Sell},
		{"invalid_price", "abc", "100", Buy},
		{"invalid_amount", "0.5", "xyz", Sell},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := computeMarketOrderAmounts(tt.price, tt.amount, tt.side)
			if err == nil {
				t.Fatalf("expected error for %v", tt)
			}
		})
	}
}

func TestValidatePriceTicks_Aligned(t *testing.T) {
	valid := []struct {
		price    string
		tickSize string
	}{
		{"0.67", "0.01"},
		{"0.80", "0.1"},
		{"0.125", "0.001"},
		{"0.50", "0.01"},
		{"0.33", "0.01"},
		{"0.032", "0.001"},
		{"0.1234", "0.0001"},
	}
	for _, tt := range valid {
		t.Run(tt.price+"_"+tt.tickSize, func(t *testing.T) {
			if err := validatePriceTicks(tt.price, tt.tickSize); err != nil {
				t.Errorf("validatePriceTicks(%s, %s) error = %v", tt.price, tt.tickSize, err)
			}
		})
	}
	misaligned := []struct {
		price    string
		tickSize string
	}{
		{"0.673", "0.01"},
		{"0.889", "0.1"},
		{"0.3333", "0.01"},
		{"0.0326", "0.001"},
	}
	for _, tt := range misaligned {
		t.Run(tt.price+"_"+tt.tickSize, func(t *testing.T) {
			if err := validatePriceTicks(tt.price, tt.tickSize); err == nil {
				t.Errorf("validatePriceTicks(%s, %s) expected error", tt.price, tt.tickSize)
			}
		})
	}
}

func TestValidatePriceTicks_RejectsNonPositive(t *testing.T) {
	invalid := []struct {
		price    string
		tickSize string
	}{
		{"0.50", "0"},
		{"0.50", "-0.01"},
		{"0.50", "0.0"},
	}
	for _, tt := range invalid {
		t.Run(tt.price+"_"+tt.tickSize, func(t *testing.T) {
			if err := validatePriceTicks(tt.price, tt.tickSize); err == nil {
				t.Errorf("validatePriceTicks(%s, %s) expected error for non-positive tickSize", tt.price, tt.tickSize)
			}
		})
	}
	if err := validatePriceTicks("0.503", ""); err != nil {
		t.Errorf("empty tickSize should skip validation, got %v", err)
	}
}

func TestValidateBytes32Hex(t *testing.T) {
	valid := []string{
		"0x0000000000000000000000000000000000000000000000000000000000000000",
		"0xABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
	}
	for _, v := range valid {
		if err := ValidateBytes32Hex("test", v); err != nil {
			t.Errorf("ValidateBytes32Hex(%q) error = %v", v, err)
		}
	}

	for _, tt := range []struct {
		value string
		name  string
	}{
		{"not-hex", "no-prefix"},
		{"0xabc", "too-short"},
		{"0x" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeff", "too-long"},
		{"0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG", "bad-char"},
	} {
		if err := ValidateBytes32Hex("test", tt.value); err == nil {
			t.Errorf("ValidateBytes32Hex(%q) expected error for %s", tt.value, tt.name)
		}
	}

	if err := ValidateBytes32Hex("test", ""); err != nil {
		t.Errorf("empty string should be valid, got %v", err)
	}
}
