package clob

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

type OrderArgsV2 struct {
	TokenID       string
	Price         string
	Size          string
	Side          Side
	Expiration    string
	SignatureType SignatureType
	BuilderCode   string
	Metadata      string
}

type MarketOrderArgsV2 struct {
	Price string
	// Price is the worst-price limit (required for amount→shares conversion).
	// BUY: max price you're willing to pay. SELL: min price you'll accept.
	TokenID       string
	Amount        string // BUY: USDC (pUSD) to spend, SELL: shares to sell
	Side          Side
	SignatureType SignatureType
	BuilderCode   string
	Metadata      string
}

type CreateOrderOptions struct {
	TickSize string // required tick size; price will be validated, not auto-rounded
	NegRisk  bool
}

type OrderBuilder struct {
	client *Client
}

func NewOrderBuilder(client *Client) *OrderBuilder {
	return &OrderBuilder{client: client}
}

func (b *OrderBuilder) BuildOrder(args OrderArgsV2, opts CreateOrderOptions) (*SignedOrder, error) {
	if err := ValidateBytes32Hex("builder", args.BuilderCode); err != nil {
		return nil, err
	}
	if err := ValidateBytes32Hex("metadata", args.Metadata); err != nil {
		return nil, err
	}
	if err := validatePriceRange(args.Price, false); err != nil {
		return nil, err
	}
	if err := validatePriceTicks(args.Price, opts.TickSize); err != nil {
		return nil, err
	}

	maker, taker, err := computeOrderAmounts(args.Price, args.Size, args.Side)
	if err != nil {
		return nil, err
	}

	order := &SignedOrder{
		TokenID:       String(args.TokenID),
		MakerAmount:   String(maker),
		TakerAmount:   String(taker),
		Side:          args.Side,
		SignatureType: args.SignatureType,
		Builder:       args.BuilderCode,
		Metadata:      args.Metadata,
		Expiration:    String("0"),
	}

	if args.Expiration != "" {
		order.Expiration = String(args.Expiration)
	}

	if err := b.client.SignOrder(order, WithSignOrderNegRisk(opts.NegRisk)); err != nil {
		return nil, err
	}
	return order, nil
}

func (b *OrderBuilder) BuildMarketOrder(args MarketOrderArgsV2, opts CreateOrderOptions) (*SignedOrder, error) {
	err := ValidateBytes32Hex("builder", args.BuilderCode)
	if err != nil {
		return nil, err
	}
	err = ValidateBytes32Hex("metadata", args.Metadata)
	if err != nil {
		return nil, err
	}
	if err := validatePriceRange(args.Price, true); err != nil {
		return nil, err
	}
	if err := validatePriceTicks(args.Price, opts.TickSize); err != nil {
		return nil, err
	}

	maker, taker, err := computeMarketOrderAmounts(args.Price, args.Amount, args.Side)
	if err != nil {
		return nil, err
	}

	order := &SignedOrder{
		TokenID:       String(args.TokenID),
		MakerAmount:   String(maker),
		TakerAmount:   String(taker),
		Side:          args.Side,
		SignatureType: args.SignatureType,
		Expiration:    String("0"),
		Builder:       args.BuilderCode,
		Metadata:      args.Metadata,
	}

	if err := b.client.SignOrder(order, WithSignOrderNegRisk(opts.NegRisk)); err != nil {
		return nil, err
	}
	return order, nil
}

func (b *OrderBuilder) CreateAndPostOrder(ctx context.Context, args OrderArgsV2, opts CreateOrderOptions, orderType OrderType, deferExec *bool) (*PostOrderResponse, error) {
	if err := validateDeferExec(orderType, deferExec); err != nil {
		return nil, err
	}
	order, err := b.BuildOrder(args, opts)
	if err != nil {
		return nil, err
	}
	if err := validateExpiration(orderType, order, time.Now); err != nil {
		return nil, err
	}
	req := PostOrderRequest{
		Order:     *order,
		Owner:     order.Maker,
		OrderType: orderType,
		DeferExec: deferExec,
	}
	out := &PostOrderResponse{}
	if err := b.client.PostOrder(ctx, req, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (b *OrderBuilder) CreateAndPostMarketOrder(ctx context.Context, args MarketOrderArgsV2, opts CreateOrderOptions, orderType OrderType, deferExec *bool) (*PostOrderResponse, error) {
	if orderType != FOK && orderType != FAK {
		return nil, fmt.Errorf("polymarket: market order type must be FOK or FAK, got %s", orderType)
	}
	if err := validateDeferExec(orderType, deferExec); err != nil {
		return nil, err
	}
	order, err := b.BuildMarketOrder(args, opts)
	if err != nil {
		return nil, err
	}
	req := PostOrderRequest{
		Order:     *order,
		Owner:     order.Maker,
		OrderType: orderType,
		DeferExec: deferExec,
	}
	out := &PostOrderResponse{}
	if err := b.client.PostOrder(ctx, req, out); err != nil {
		return nil, err
	}
	return out, nil
}

type OrderBuilderConfig struct {
	TickSize string
	NegRisk  bool
}

type GetMarketOptionsResponse struct {
	TickSize string
	NegRisk  bool
}

func (b *OrderBuilder) GetMarketOptions(ctx context.Context, tokenID string) (*GetMarketOptionsResponse, error) {
	var tick TickSizeResponse
	if err := b.client.GetTickSizeByTokenID(ctx, tokenID, &tick); err != nil {
		return nil, fmt.Errorf("polymarket: get tick size: %w", err)
	}
	var neg NegRiskResponse
	if err := b.client.GetNegRisk(ctx, tokenID, &neg); err != nil {
		return nil, fmt.Errorf("polymarket: get neg risk: %w", err)
	}
	return &GetMarketOptionsResponse{
		TickSize: string(tick.MinimumTickSize),
		NegRisk:  neg.NegRisk,
	}, nil
}

func (b *OrderBuilder) BuildOrderForToken(ctx context.Context, args OrderArgsV2, builderCode, metadata string) (*SignedOrder, error) {
	opts, err := b.GetMarketOptions(ctx, args.TokenID)
	if err != nil {
		return nil, err
	}
	return b.BuildOrder(OrderArgsV2{
		TokenID:       args.TokenID,
		Price:         args.Price,
		Size:          args.Size,
		Side:          args.Side,
		Expiration:    args.Expiration,
		SignatureType: args.SignatureType,
		BuilderCode:   builderCode,
		Metadata:      metadata,
	}, CreateOrderOptions{TickSize: opts.TickSize, NegRisk: opts.NegRisk})
}

func (b *OrderBuilder) CreateAndPostOrderForToken(ctx context.Context, args OrderArgsV2, orderType OrderType, deferExec *bool, builderCode, metadata string) (*PostOrderResponse, error) {
	opts, err := b.GetMarketOptions(ctx, args.TokenID)
	if err != nil {
		return nil, err
	}
	return b.CreateAndPostOrder(ctx, OrderArgsV2{
		TokenID:       args.TokenID,
		Price:         args.Price,
		Size:          args.Size,
		Side:          args.Side,
		Expiration:    args.Expiration,
		SignatureType: args.SignatureType,
		BuilderCode:   builderCode,
		Metadata:      metadata,
	}, CreateOrderOptions{TickSize: opts.TickSize, NegRisk: opts.NegRisk}, orderType, deferExec)
}

// validateDeferExec ensures deferExec (post-only) is only used with GTC/GTD.
func validateDeferExec(orderType OrderType, deferExec *bool) error {
	if deferExec != nil && *deferExec && (orderType == FOK || orderType == FAK) {
		return fmt.Errorf("polymarket: deferExec (post-only) is not compatible with %s; use GTC or GTD", orderType)
	}
	return nil
}

// validateExpiration checks GTD expiration locally to reduce INVALID_ORDER_EXPIRATION rejections.
func validateExpiration(orderType OrderType, order *SignedOrder, nowFn func() time.Time) error {
	exp := order.Expiration.String()
	if orderType == GTD {
		if exp == "" || exp == "0" {
			return fmt.Errorf("polymarket: GTD orders require a non-zero expiration")
		}
		expInt, err := strconv.ParseInt(exp, 10, 64)
		if err != nil {
			return fmt.Errorf("polymarket: GTD expiration must be a numeric Unix timestamp: %w", err)
		}
		if expInt <= 0 {
			return fmt.Errorf("polymarket: GTD expiration must be a positive Unix timestamp")
		}
		threshold := nowFn().Unix() + 60
		if expInt < threshold {
			return fmt.Errorf("polymarket: GTD expiration must be at least 60 seconds in the future (got %d, need >= %d)", expInt, threshold)
		}
	}
	return nil
}

// validatePriceRange checks that the price is in a valid range for Polymarket binary tokens.
// limitOrder: 0 < price < 1
// marketOrder: 0 < price <= 1 (1.00 is allowed as worstPrice)
func validatePriceRange(price string, allowOne bool) error {
	if price == "" {
		return fmt.Errorf("polymarket: price must not be empty")
	}
	pr, _, err := big.ParseFloat(price, 10, 64, big.ToZero)
	if err != nil {
		return fmt.Errorf("polymarket: invalid price %q", price)
	}
	if pr.Sign() <= 0 {
		return fmt.Errorf("polymarket: price must be > 0, got %s", price)
	}
	if pr.Cmp(big.NewFloat(1)) >= 0 && !allowOne {
		return fmt.Errorf("polymarket: price must be < 1 for limit orders, got %s", price)
	}
	if pr.Cmp(big.NewFloat(1)) > 0 {
		return fmt.Errorf("polymarket: price must be <= 1, got %s", price)
	}
	return nil
}
