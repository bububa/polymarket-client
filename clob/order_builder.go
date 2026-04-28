package clob

import "context"

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
	TokenID       string
	WorstPrice    string
	Size          string
	Side          Side
	SignatureType SignatureType
}

type OrderBuilder struct {
	client *Client
}

func NewOrderBuilder(client *Client) *OrderBuilder {
	return &OrderBuilder{client: client}
}

func (b *OrderBuilder) BuildOrder(args OrderArgsV2) (*SignedOrder, error) {
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

	if err := b.client.SignOrder(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (b *OrderBuilder) BuildMarketOrder(args MarketOrderArgsV2) (*SignedOrder, error) {
	maker, taker, err := computeMarketOrderAmounts(args.WorstPrice, args.Size, args.Side)
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
		Builder:       ZeroBytes32,
		Metadata:      ZeroBytes32,
	}

	if err := b.client.SignOrder(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (b *OrderBuilder) CreateAndPostOrder(ctx context.Context, args OrderArgsV2, orderType OrderType, deferExec bool) (*PostOrderResponse, error) {
	order, err := b.BuildOrder(args)
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

func (b *OrderBuilder) CreateAndPostMarketOrder(ctx context.Context, args MarketOrderArgsV2, deferExec bool) (*PostOrderResponse, error) {
	order, err := b.BuildMarketOrder(args)
	if err != nil {
		return nil, err
	}
	req := PostOrderRequest{
		Order:     *order,
		Owner:     order.Maker,
		OrderType: GTC,
		DeferExec: deferExec,
	}
	out := &PostOrderResponse{}
	if err := b.client.PostOrder(ctx, req, out); err != nil {
		return nil, err
	}
	return out, nil
}
