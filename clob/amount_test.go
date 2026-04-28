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
	tests := []struct {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := computeOrderAmounts(tt.price, tt.size, tt.side)
			if err == nil {
				t.Fatalf("expected error for %v", tt)
			}
		})
	}
}

func TestComputeMarketOrderAmounts(t *testing.T) {
	maker, taker, err := computeMarketOrderAmounts("0.99", "50", Buy)
	if err != nil {
		t.Fatal(err)
	}
	if maker != "49500000" {
		t.Errorf("market BUY makerAmount = %s, want 49500000", maker)
	}
	if taker != "50000000" {
		t.Errorf("market BUY takerAmount = %s, want 50000000", taker)
	}
}

func TestRoundToTickSize(t *testing.T) {
	tests := []struct {
		price    string
		tickSize string
		want     string
	}{
		{"0.673", "0.01", "0.67"},
		{"0.889", "0.1", "0.80"},
		{"0.125", "0.001", "0.125"},
		{"0.5", "0.01", "0.50"},
		{"0.3333", "0.01", "0.33"},
		{"0.0326", "0.001", "0.032"},
		{"0.123456", "0.0001", "0.1234"},
	}

	for _, tt := range tests {
		t.Run(tt.price+"_"+tt.tickSize, func(t *testing.T) {
			got, err := roundToTickSize(tt.price, tt.tickSize)
			if err != nil {
				t.Fatalf("roundToTickSize(%s, %s) error = %v", tt.price, tt.tickSize, err)
			}
			if got != tt.want {
				t.Errorf("roundToTickSize(%s, %s) = %q, want %q", tt.price, tt.tickSize, got, tt.want)
			}
		})
	}
}

func TestRoundToTickSize_ZeroTick(t *testing.T) {
	got, err := roundToTickSize("0.673", "0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.673000" {
		t.Errorf("zero tick size result = %q, want 0.673000", got)
	}
}
