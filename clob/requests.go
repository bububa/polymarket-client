package clob

// TradeParams filters GET /data/trades requests.
type TradeParams struct {
	// ID filters by trade identifier.
	ID string `url:"id,omitempty"`
	// TakerAddress filters by taker wallet address.
	TakerAddress string `url:"taker,omitempty"`
	// MakerAddress filters by maker wallet address.
	MakerAddress string `url:"maker,omitempty"`
	// Market filters by condition ID.
	Market string `url:"market,omitempty"`
	// AssetID filters by conditional token identifier.
	AssetID string `url:"asset_id,omitempty"`
	// Before filters trades before this Unix timestamp.
	Before int64 `url:"before,omitempty"`
	// After filters trades after this Unix timestamp.
	After int64 `url:"after,omitempty"`
}

// OpenOrderParams filters GET /data/orders requests.
type OpenOrderParams struct {
	// ID filters by order identifier.
	ID string `url:"id,omitempty"`
	// Market filters by condition ID.
	Market string `url:"market,omitempty"`
	// AssetID filters by conditional token identifier.
	AssetID string `url:"asset_id,omitempty"`
	// NextCursor Cursor for pagination (base64 encoded offset)
	NextCursor string `url:"next_cursor,omitempty" json:"next_cursor,omitempty"`
}

// PriceHistoryParams filters GET /prices-history requests.
type PriceHistoryParams struct {
	// Market is the condition ID.
	Market string `url:"market,omitempty"`
	// StartTS is the start Unix timestamp.
	StartTS int64 `url:"startTs,omitempty"`
	// EndTS is the end Unix timestamp.
	EndTS int64 `url:"endTs,omitempty"`
	// Fidelity controls the data point density.
	Fidelity int `url:"fidelity,omitempty"`
	// Interval sets the bucket size (e.g. "5m", "1h").
	Interval string `url:"interval,omitempty"`
}

// BatchPriceHistoryParams is the request body for POST /batch-prices-history.
type BatchPriceHistoryParams struct {
	// Markets lists the condition IDs to query.
	Markets []string `json:"markets"`
	// StartTS is the start Unix timestamp.
	StartTS int64 `json:"start_ts,omitempty"`
	// EndTS is the end Unix timestamp.
	EndTS int64 `json:"end_ts,omitempty"`
	// Fidelity controls the data point density.
	Fidelity int `json:"fidelity,omitempty"`
	// Interval sets the bucket size (e.g. "5m", "1h").
	Interval string `json:"interval,omitempty"`
}

// RebateParams filters current maker rebate requests.
type RebateParams struct {
	// Date filters by reward date.
	Date string `url:"date,omitempty"`
	// MakerAddress filters by maker wallet address.
	MakerAddress string `url:"maker_address,omitempty"`
}

// DropNotificationParams identifies notifications to dismiss.
type DropNotificationParams struct {
	// IDs is the list of notification identifiers to drop.
	IDs []string `url:"ids,omitempty"`
}

// OrderMarketCancelParams targets orders for partial cancellation.
type OrderMarketCancelParams struct {
	// Market is the condition ID to cancel orders for.
	Market string `json:"market,omitempty"`
	// AssetID is the conditional token to cancel.
	AssetID string `json:"asset_id,omitempty"`
}

// RewardsMarketsParams filters GET /rewards/markets requests.
type RewardsMarketsParams struct {
	// ConditionID filters by market ID.
	ConditionID string `url:"condition_id,omitempty"`
	// Date filters by reward date.
	Date string `url:"date,omitempty"`
	// NextCursor is the pagination cursor.
	NextCursor string `url:"next_cursor,omitempty"`
}

// EarningsParams filters GET /rewards/user/markets requests.
type EarningsParams struct {
	// Date filters by reward date.
	Date string `url:"date,omitempty"`
	// OrderBy sets the sort field.
	OrderBy string `url:"order_by,omitempty"`
	// Position sets the result offset.
	Position string `url:"position,omitempty"`
	// NoCompetition skips low-competition markets.
	NoCompetition bool `url:"no_competition,omitempty"`
	// NextCursor is the pagination cursor.
	NextCursor string `url:"next_cursor,omitempty"`
}

// BuilderTradeParams filters GET /builder/trades requests.
type BuilderTradeParams struct {
	// ID filters by trade identifier.
	ID string `url:"id,omitempty"`
	// MakerAddress filters by maker wallet address.
	MakerAddress string `url:"maker,omitempty"`
	// Market filters by condition ID.
	Market string `url:"market,omitempty"`
	// AssetID filters by conditional token identifier.
	AssetID string `url:"asset_id,omitempty"`
	// Before filters trades before this Unix timestamp.
	Before int64 `url:"before,omitempty"`
	// After filters trades after this Unix timestamp.
	After int64 `url:"after,omitempty"`
	// BuilderCode filters by builder attribution code.
	BuilderCode string `url:"builder_code,omitempty"`
	// NextCursor is the pagination cursor.
	NextCursor string `url:"next_cursor,omitempty"`
}

// CreateRFQRequest is the body for POST /rfq/request.
type CreateRFQRequest struct {
	// AssetIn is the ERC-20 address of the asset being sent.
	AssetIn string `json:"assetIn"`
	// AssetOut is the ERC-20 address of the asset being received.
	AssetOut string `json:"assetOut"`
	// AmountIn is the input quantity.
	AmountIn Float64 `json:"amountIn"`
	// AmountOut is the desired output quantity.
	AmountOut Float64 `json:"amountOut"`
	// UserType identifies the signature method.
	UserType SignatureType `json:"userType"`
}

// CancelRFQRequest is the body for DELETE /rfq/request.
type CancelRFQRequest struct {
	// RequestID identifies the RFQ to cancel.
	RequestID string `json:"requestId"`
}

// CreateRFQQuoteRequest is the body for POST /rfq/quote.
type CreateRFQQuoteRequest struct {
	// RequestID references the original RFQ request.
	RequestID string `json:"requestId"`
	// AssetIn is the ERC-20 address being sent.
	AssetIn string `json:"assetIn"`
	// AssetOut is the ERC-20 address being received.
	AssetOut string `json:"assetOut"`
	// AmountIn is the input quantity.
	AmountIn Float64 `json:"amountIn"`
	// AmountOut is the quoted output quantity.
	AmountOut Float64 `json:"amountOut"`
	// UserType identifies the signature method.
	UserType SignatureType `json:"userType"`
}

// CancelRFQQuoteRequest is the body for DELETE /rfq/quote.
type CancelRFQQuoteRequest struct {
	// QuoteID identifies the quote to cancel.
	QuoteID string `json:"quoteId"`
}

// RFQListParams filters GET /rfq/data/* listing endpoints.
type RFQListParams struct {
	// Offset is the start index.
	Offset string `url:"offset,omitempty"`
	// Limit sets the maximum returned results.
	Limit int `url:"limit,omitempty"`
	// State filters by quote/request state.
	State string `url:"state,omitempty"`
	// RequestIDs filters by request IDs.
	RequestIDs []string `url:"requestIds,omitempty"`
	// QuoteIDs filters by quote IDs.
	QuoteIDs []string `url:"quoteIds,omitempty"`
	// Markets filters by condition IDs.
	Markets []string `url:"markets,omitempty"`
	// SizeMin filters by minimum trade size.
	SizeMin Float64 `url:"sizeMin,omitempty"`
	// SizeMax filters by maximum trade size.
	SizeMax Float64 `url:"sizeMax,omitempty"`
	// SizeUSDCMin filters by minimum USDC value.
	SizeUSDCMin Float64 `url:"sizeUsdcMin,omitempty"`
	// SizeUSDCMax filters by maximum USDC value.
	SizeUSDCMax Float64 `url:"sizeUsdcMax,omitempty"`
	// PriceMin filters by minimum price.
	PriceMin Float64 `url:"priceMin,omitempty"`
	// PriceMax filters by maximum price.
	PriceMax Float64 `url:"priceMax,omitempty"`
	// SortBy sets the sort field.
	SortBy string `url:"sortBy,omitempty"`
	// SortDir sets the sort direction (asc/desc).
	SortDir string `url:"sortDir,omitempty"`
}
