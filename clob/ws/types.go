package ws

import (
	"encoding/json"
	"fmt"

	"github.com/bububa/polymarket-client/clob"
)

// Channel identifies the WebSocket subscription target.
type Channel string

const (
	// ChannelMarket subscribes to order book updates.
	ChannelMarket Channel = "market"
	// ChannelUser subscribes to user-specific order and trade events.
	ChannelUser Channel = "user"
)

// EventType identifies the kind of WebSocket event.
type EventType string

const (
	// EventTypeBook is an order book snapshot or delta.
	EventTypeBook EventType = "book"
	// EventTypePriceChange signals a price update.
	EventTypePriceChange EventType = "price_change"
	// EventTypeTickSizeChange signals a tick size update.
	EventTypeTickSizeChange EventType = "tick_size_change"
	// EventTypeLastTradePrice signals a new trade.
	EventTypeLastTradePrice EventType = "last_trade_price"
	// EventTypeOrder is an order status change notification.
	EventTypeOrder EventType = "order"
	// EventTypeTrade is a trade confirmation.
	EventTypeTrade EventType = "trade"
	// EventTypeBestBidAsk is the top-of-book update.
	EventTypeBestBidAsk EventType = "best_bid_ask"
	// EventTypeNewMarket signals market listing.
	EventTypeNewMarket EventType = "new_market"
	// EventTypeMarketResolved signals market resolution.
	EventTypeMarketResolved EventType = "market_resolved"
)

// UserSubscription is sent to subscribe to user-channel events.
type UserSubscription struct {
	// Type must be ChannelUser.
	Type Channel `json:"type"`
	// Auth contains WebSocket credentials.
	Auth clob.WSAuth `json:"auth"`
	// Markets optionally scopes subscriptions.
	Markets []string `json:"markets,omitempty"`
	// Operation is "subscribe" or "unsubscribe".
	Operation string `json:"operation,omitempty"`
}

// MarketSubscription is sent to subscribe to market-channel events.
type MarketSubscription struct {
	// Type must be ChannelMarket.
	Type Channel `json:"type"`
	// Operation is "subscribe" or "unsubscribe".
	Operation string `json:"operation,omitempty"`
	// Markets optionally limits to specific condition IDs.
	Markets []string `json:"markets,omitempty"`
	// AssetIDs optionally limits to specific token IDs.
	AssetIDs []string `json:"assets_ids,omitempty"`
	// InitialDump requests a full order book on subscribe.
	InitialDump bool `json:"initial_dump,omitempty"`
	// CustomFeatureEnabled enables extended features.
	CustomFeatureEnabled bool `json:"custom_feature_enabled,omitempty"`
}

// BaseEvent is the common prefix of all WebSocket event payloads.
type BaseEvent struct {
	// EventType identifies the specific event variant.
	EventType EventType `json:"event_type"`
}

// Event is the discriminated-union interface for WebSocket events.
type Event interface{ isEvent() }

func (*BookEvent) isEvent()           {}
func (*PriceChangeEvent) isEvent()    {}
func (*TickSizeChangeEvent) isEvent() {}
func (*LastTradePriceEvent) isEvent() {}
func (*OrderEvent) isEvent()          {}
func (*TradeEvent) isEvent()          {}
func (*BestBidAskEvent) isEvent()     {}
func (*NewMarketEvent) isEvent()      {}
func (*MarketResolvedEvent) isEvent() {}

// BookEvent carries an order book update.
type BookEvent struct {
	BaseEvent
	// AssetID is the conditional token identifier.
	AssetID string `json:"asset_id"`
	// Market Condition ID of the market
	Market string `json:"market"`
	// Bids are buy order levels.
	Bids []clob.OrderSummary `json:"bids"`
	// Asks are sell order levels.
	Asks []clob.OrderSummary `json:"asks"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
	// Hash of the orderbook content
	Hash string `json:"hash"`
}

// PriceChangeBatchEvent carries batch trade price update.
type PriceChangeBatchEvent struct {
	BaseEvent
	// Market is the condition identifier.
	Market string `json:"market"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
	// PriceChanges
	PriceChanges []PriceChangeEvent `json:"price_changes"`
}

// PriceChangeEvent carries a single trade price update.
type PriceChangeEvent struct {
	BaseEvent
	// Market is the condition identifier.
	Market string `json:"market"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
	// AssetID is the conditional token identifier.
	AssetID string `json:"asset_id"`
	// BestBid is the highest bid price when included in price change messages.
	BestBid clob.Float64 `json:"best_bid"`
	// BestAsk is the lowest ask price when included in price change messages.
	BestAsk clob.Float64 `json:"best_ask"`
	// Price is the trade price.
	Price clob.Float64 `json:"price"`
	// Size is the trade quantity.
	Size clob.Float64 `json:"size"`
	// Side is the trade direction.
	Side clob.Side `json:"side"`
	// Hash of the orderbook content
	Hash string `json:"hash"`
}

// TickSizeChangeEvent carries a tick size update notification.
type TickSizeChangeEvent struct {
	BaseEvent
	// AssetID is the conditional token identifier.
	AssetID string `json:"asset_id"`
	// Market is the condition ID.
	Market string `json:"market"`
	// OldTickSize is the previous tick size.
	OldTickSize clob.TickSize `json:"old_tick_size"`
	// NewTickSize is the updated tick size.
	NewTickSize clob.TickSize `json:"new_tick_size"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
}

// LastTradePriceEvent carries the most recent trade price.
type LastTradePriceEvent struct {
	BaseEvent
	// AssetID is the conditional token identifier.
	AssetID string `json:"asset_id"`
	// Market is the condition ID.
	Market string `json:"market"`
	// Price is the trade price.
	Price clob.Float64 `json:"price"`
	// Size is the trade quantity.
	Size clob.Float64 `json:"size"`
	// Side is the trade direction.
	Side clob.Side `json:"side"`
	// FeeRateBps is the fee rate in basis points.
	FeeRateBps clob.Float64 `json:"fee_rate_bps"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
	// TransactionHash transaction_hash
	TransactionHash string `json:"transaction_hash"`
}

// OrderEvent carries an order status change (fill, cancel, expiry).
type OrderEvent struct {
	BaseEvent
	// OrderID is the affected order identifier.
	OrderID string `json:"order_id"`
	// AssetID is the conditional token identifier.
	AssetID string `json:"asset_id"`
	// Market is the condition ID.
	Market string `json:"market"`
	// Price is the order price.
	Price clob.Float64 `json:"price"`
	// Size is the order quantity.
	Size clob.Float64 `json:"size"`
	// Side is the order direction.
	Side clob.Side `json:"side"`
	// Status is the updated order status.
	Status OrderStatus `json:"status"`
	// Reason explains why the order changed state.
	Reason string `json:"reason,omitempty"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
}

// TradeEvent carries a matched trade notification.
type TradeEvent struct {
	BaseEvent
	// TradeID is the trade identifier.
	TradeID string `json:"trade_id"`
	// AssetID is the conditional token identifier.
	AssetID string `json:"asset_id"`
	// Market is the condition ID.
	Market string `json:"market"`
	// Price is the execution price.
	Price clob.Float64 `json:"price"`
	// Size is the matched quantity.
	Size clob.Float64 `json:"size"`
	// Side is the trade direction.
	Side clob.Side `json:"side"`
	// Status is the trade processing status.
	Status TradeStatus `json:"status"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
}

// BestBidAskEvent carries top-of-book bid/ask update.
type BestBidAskEvent struct {
	BaseEvent
	// Market is the condition ID.
	Market string `json:"market"`
	// AssetID is the conditional token identifier.
	AssetID string `json:"asset_id"`
	// BestBid is the highest bid price.
	BestBid clob.Float64 `json:"best_bid"`
	// BestAsk is the lowest ask price.
	BestAsk clob.Float64 `json:"best_ask"`
	// Spread is the difference.
	Spread clob.Float64 `json:"spread"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
}

// EventMessage is the embedded sports event metadata.
type EventMessage struct {
	// ID is the message identifier.
	ID string `json:"id"`
	// Ticker is the sports ticker symbol.
	Ticker string `json:"ticker"`
	// Slug is the event slug.
	Slug string `json:"slug"`
	// Title is the event title.
	Title string `json:"title"`
	// Description is the event description.
	Description string `json:"description"`
}

// NewMarketEvent signals a new market listing via WebSocket.
type NewMarketEvent struct {
	BaseEvent
	// ID is the market condition identifier.
	ID string `json:"id"`
	// Question is the market question text.
	Question string `json:"question"`
	// Market is the market name.
	Market string `json:"market"`
	// Slug is the URL-friendly slug.
	Slug string `json:"slug"`
	// Description is the market description.
	Description string `json:"description"`
	// AssetIDs lists the conditional token identifiers.
	AssetIDs []string `json:"assets_ids"`
	// AssetIDsAlt accepts the asset_ids variant sometimes emitted by CLOB.
	AssetIDsAlt []string `json:"asset_ids,omitempty"`
	// Outcomes lists the outcome labels.
	Outcomes []string `json:"outcomes"`
	// EventMessage is optional embedded event metadata.
	EventMessage *EventMessage `json:"event_message,omitempty"`
	// Timestamp is the event time.
	Timestamp             clob.Time     `json:"timestamp"`
	Tags                  []string      `json:"tags,omitempty"`
	ConditionID           string        `json:"condition_id,omitempty"`
	Active                *bool         `json:"active,omitempty"`
	CLOBTokenIDs          []string      `json:"clob_token_ids,omitempty"`
	SportsMarketType      string        `json:"sports_market_type,omitempty"`
	Line                  string        `json:"line,omitempty"`
	GameStartTime         clob.Time     `json:"game_start_time,omitzero"` // 如果可能为空字符串，可能要用 string 更稳
	OrderPriceMinTickSize clob.TickSize `json:"order_price_min_tick_size,omitempty"`
	GroupItemTitle        string        `json:"group_item_title,omitempty"`
}

// MarketResolvedEvent signals a market has been resolved.
type MarketResolvedEvent struct {
	BaseEvent
	// ID is the market condition identifier.
	ID string `json:"id"`
	// Question is the market question text.
	Question string `json:"question,omitempty"`
	// Market is the market name.
	Market string `json:"market"`
	// Slug is the URL-friendly slug.
	Slug string `json:"slug,omitempty"`
	// Description is the market description.
	Description string `json:"description,omitempty"`
	// AssetIDs lists the conditional token identifiers.
	AssetIDs []string `json:"assets_ids"`
	// AssetIDsAlt accepts the asset_ids variant sometimes emitted by CLOB.
	AssetIDsAlt []string `json:"asset_ids,omitempty"`
	// Outcomes lists the outcome labels.
	Outcomes []string `json:"outcomes,omitempty"`
	// WinningAssetID is the resolved winning token ID.
	WinningAssetID string `json:"winning_asset_id"`
	// WinningOutcome is the resolved outcome label.
	WinningOutcome string `json:"winning_outcome"`
	// EventMessage is optional embedded event metadata.
	EventMessage *EventMessage `json:"event_message,omitempty"`
	// Timestamp is the event time.
	Timestamp clob.Time `json:"timestamp"`
	Tags      []string  `json:"tags,omitempty"`
}

// OrderStatus identifies the lifecycle state of an order.
type OrderStatus string

const (
	// OrderStatusOpen is an active unfilled order.
	OrderStatusOpen OrderStatus = "OPEN"
	// OrderStatusCanceled has been canceled by the user.
	OrderStatusCanceled OrderStatus = "CANCELED"
	// OrderStatusFilled is fully matched.
	OrderStatusFilled OrderStatus = "FILLED"
	// OrderStatusExpired has passed its expiry time.
	OrderStatusExpired OrderStatus = "EXPIRED"
	// OrderStatusRetrying is being re-submitted after failure.
	OrderStatusRetrying OrderStatus = "RETRYING"
	// OrderStatusFailed was rejected or failed processing.
	OrderStatusFailed OrderStatus = "FAILED"
)

// TradeStatus identifies the processing state of a trade.
type TradeStatus string

const (
	// TradeStatusMatched is a completed trade.
	TradeStatusMatched TradeStatus = "matched"
	// TradeStatusRetrying is being re-submitted.
	TradeStatusRetrying TradeStatus = "RETRYING"
	// TradeStatusFailed was rejected or failed processing.
	TradeStatusFailed TradeStatus = "FAILED"
)

// SportsResultUpdate is a real-time sports match update from the sports channel.
type SportsResultUpdate struct {
	// Slug is the sports match slug.
	Slug string `json:"slug"`
	// Live is true when the match is live.
	Live bool `json:"live"`
	// Ended is true when the match has ended.
	Ended bool `json:"ended"`
	// Score is the current score.
	Score string `json:"score"`
	// Period is the current match period.
	Period string `json:"period"`
	// Elapsed is the elapsed match time.
	Elapsed string `json:"elapsed"`
	// LastUpdate is the last update timestamp.
	LastUpdate clob.Time `json:"last_update"`
}

// DecodeEvent decodes a raw CLOB WebSocket event payload into a typed Event.
func DecodeEvent(data []byte) (Event, error) {
	var base BaseEvent
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}
	var out Event
	switch base.EventType {
	case EventTypeBook:
		out = &BookEvent{}
	case EventTypePriceChange:
		out = &PriceChangeEvent{}
	case EventTypeTickSizeChange:
		out = &TickSizeChangeEvent{}
	case EventTypeLastTradePrice:
		out = &LastTradePriceEvent{}
	case EventTypeOrder:
		out = &OrderEvent{}
	case EventTypeTrade:
		out = &TradeEvent{}
	case EventTypeBestBidAsk:
		out = &BestBidAskEvent{}
	case EventTypeNewMarket:
		out = &NewMarketEvent{}
	case EventTypeMarketResolved:
		out = &MarketResolvedEvent{}
	default:
		return nil, fmt.Errorf("unknown websocket event type %q", base.EventType)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return nil, err
	}
	switch ev := out.(type) {
	case *NewMarketEvent:
		if len(ev.AssetIDs) == 0 && len(ev.AssetIDsAlt) > 0 {
			ev.AssetIDs = ev.AssetIDsAlt
		}
	case *MarketResolvedEvent:
		if len(ev.AssetIDs) == 0 && len(ev.AssetIDsAlt) > 0 {
			ev.AssetIDs = ev.AssetIDsAlt
		}
	}
	return out, nil
}
