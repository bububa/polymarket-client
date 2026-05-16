// Package rtds implements the Polymarket real-time data stream client.
package rtds

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bububa/polymarket-client/shared"
)

// Action is an RTDS subscription action.
type Action string

const (
	// ActionSubscribe subscribes to one or more topics.
	ActionSubscribe Action = "subscribe"
	// ActionUnsubscribe unsubscribes from one or more topics.
	ActionUnsubscribe Action = "unsubscribe"
)

const (
	// TopicCryptoPrices is the Binance crypto price RTDS topic.
	TopicCryptoPrices = "crypto_prices"
	// TopicCryptoPricesChainlink is the Chainlink crypto price RTDS topic.
	TopicCryptoPricesChainlink = "crypto_prices_chainlink"
	// TopicEquityPrices is the equity, ETF, forex, metals, and commodities RTDS topic.
	TopicEquityPrices = "equity_prices"
	// TopicComments is the comments RTDS topic.
	TopicComments = "comments"
)

const (
	// TypeUpdate is the live update message type.
	TypeUpdate = "update"
	// TypeSubscribe is the initial subscription snapshot message type.
	TypeSubscribe = "subscribe"
	// TypeAll subscribes to all message types for a topic.
	TypeAll = "*"
)

// CommentType identifies the RTDS comment event type.
type CommentType string

const (
	// CommentCreated is emitted when a comment is created.
	CommentCreated CommentType = "comment_created"
	// CommentRemoved is emitted when a comment is removed.
	CommentRemoved CommentType = "comment_removed"
	// ReactionCreated is emitted when a reaction is created.
	ReactionCreated CommentType = "reaction_created"
	// ReactionRemoved is emitted when a reaction is removed.
	ReactionRemoved CommentType = "reaction_removed"
)

const (
	// BinanceSymbolBTCUSDT is the Binance Bitcoin/USDT symbol.
	BinanceSymbolBTCUSDT = "btcusdt"
	// BinanceSymbolETHUSDT is the Binance Ethereum/USDT symbol.
	BinanceSymbolETHUSDT = "ethusdt"
	// BinanceSymbolSOLUSDT is the Binance Solana/USDT symbol.
	BinanceSymbolSOLUSDT = "solusdt"
	// BinanceSymbolXRPUSDT is the Binance XRP/USDT symbol.
	BinanceSymbolXRPUSDT = "xrpusdt"
)

const (
	// ChainlinkSymbolBTCUSD is the Chainlink Bitcoin/USD symbol.
	ChainlinkSymbolBTCUSD = "btc/usd"
	// ChainlinkSymbolETHUSD is the Chainlink Ethereum/USD symbol.
	ChainlinkSymbolETHUSD = "eth/usd"
	// ChainlinkSymbolSOLUSD is the Chainlink Solana/USD symbol.
	ChainlinkSymbolSOLUSD = "sol/usd"
	// ChainlinkSymbolXRPUSD is the Chainlink XRP/USD symbol.
	ChainlinkSymbolXRPUSD = "xrp/usd"
)

const (
	// EquitySymbolAAPL is the Apple stock symbol.
	EquitySymbolAAPL = "AAPL"
	// EquitySymbolTSLA is the Tesla stock symbol.
	EquitySymbolTSLA = "TSLA"
	// EquitySymbolMSFT is the Microsoft stock symbol.
	EquitySymbolMSFT = "MSFT"
	// EquitySymbolGOOGL is the Alphabet stock symbol.
	EquitySymbolGOOGL = "GOOGL"
	// EquitySymbolAMZN is the Amazon stock symbol.
	EquitySymbolAMZN = "AMZN"
	// EquitySymbolMETA is the Meta Platforms stock symbol.
	EquitySymbolMETA = "META"
	// EquitySymbolNVDA is the NVIDIA stock symbol.
	EquitySymbolNVDA = "NVDA"
	// EquitySymbolNFLX is the Netflix stock symbol.
	EquitySymbolNFLX = "NFLX"
	// EquitySymbolPLTR is the Palantir stock symbol.
	EquitySymbolPLTR = "PLTR"
	// EquitySymbolOPEN is the Opendoor stock symbol.
	EquitySymbolOPEN = "OPEN"
	// EquitySymbolRKLB is the Rocket Lab stock symbol.
	EquitySymbolRKLB = "RKLB"
	// EquitySymbolABNB is the Airbnb stock symbol.
	EquitySymbolABNB = "ABNB"
	// EquitySymbolCOIN is the Coinbase stock symbol.
	EquitySymbolCOIN = "COIN"
	// EquitySymbolHOOD is the Robinhood stock symbol.
	EquitySymbolHOOD = "HOOD"
)

const (
	// EquitySymbolQQQ is the Invesco QQQ ETF symbol.
	EquitySymbolQQQ = "QQQ"
	// EquitySymbolSPY is the S&P 500 ETF symbol.
	EquitySymbolSPY = "SPY"
	// EquitySymbolEWY is the iShares MSCI South Korea ETF symbol.
	EquitySymbolEWY = "EWY"
	// EquitySymbolVXX is the Barclays iPath Series B S&P 500 VIX symbol.
	EquitySymbolVXX = "VXX"
)

const (
	// EquitySymbolEURUSD is the Euro/USD forex symbol.
	EquitySymbolEURUSD = "EURUSD"
	// EquitySymbolGBPUSD is the British Pound/USD forex symbol.
	EquitySymbolGBPUSD = "GBPUSD"
	// EquitySymbolUSDCAD is the USD/Canadian Dollar forex symbol.
	EquitySymbolUSDCAD = "USDCAD"
	// EquitySymbolUSDJPY is the USD/Japanese Yen forex symbol.
	EquitySymbolUSDJPY = "USDJPY"
	// EquitySymbolUSDKRW is the USD/South Korean Won forex symbol.
	EquitySymbolUSDKRW = "USDKRW"
)

const (
	// EquitySymbolXAUUSD is the gold spot symbol.
	EquitySymbolXAUUSD = "XAUUSD"
	// EquitySymbolXAGUSD is the silver spot symbol.
	EquitySymbolXAGUSD = "XAGUSD"
)

const (
	// EquitySymbolWTI is the crude oil WTI rolling front-month futures symbol.
	EquitySymbolWTI = "WTI"
	// EquitySymbolCC is the cocoa rolling front-month futures symbol.
	EquitySymbolCC = "CC"
	// EquitySymbolNGD is the natural gas rolling front-month futures symbol.
	EquitySymbolNGD = "NGD"
)

var (
	supportedBinanceSymbols = []string{
		BinanceSymbolBTCUSDT,
		BinanceSymbolETHUSDT,
		BinanceSymbolSOLUSDT,
		BinanceSymbolXRPUSDT,
	}
	supportedChainlinkSymbols = []string{
		ChainlinkSymbolBTCUSD,
		ChainlinkSymbolETHUSD,
		ChainlinkSymbolSOLUSD,
		ChainlinkSymbolXRPUSD,
	}
	supportedEquityStockSymbols = []string{
		EquitySymbolAAPL,
		EquitySymbolTSLA,
		EquitySymbolMSFT,
		EquitySymbolGOOGL,
		EquitySymbolAMZN,
		EquitySymbolMETA,
		EquitySymbolNVDA,
		EquitySymbolNFLX,
		EquitySymbolPLTR,
		EquitySymbolOPEN,
		EquitySymbolRKLB,
		EquitySymbolABNB,
		EquitySymbolCOIN,
		EquitySymbolHOOD,
	}
	supportedEquityETFSymbols = []string{
		EquitySymbolQQQ,
		EquitySymbolSPY,
		EquitySymbolEWY,
		EquitySymbolVXX,
	}
	supportedEquityForexSymbols = []string{
		EquitySymbolEURUSD,
		EquitySymbolGBPUSD,
		EquitySymbolUSDCAD,
		EquitySymbolUSDJPY,
		EquitySymbolUSDKRW,
	}
	supportedEquityMetalSymbols = []string{
		EquitySymbolXAUUSD,
		EquitySymbolXAGUSD,
	}
	supportedEquityCommoditySymbols = []string{
		EquitySymbolWTI,
		EquitySymbolCC,
		EquitySymbolNGD,
	}
)

// SupportedBinanceSymbols returns the officially documented Binance RTDS symbols.
func SupportedBinanceSymbols() []string {
	return cloneStrings(supportedBinanceSymbols)
}

// SupportedChainlinkSymbols returns the officially documented Chainlink RTDS symbols.
func SupportedChainlinkSymbols() []string {
	return cloneStrings(supportedChainlinkSymbols)
}

// SupportedEquitySymbols returns all officially documented equity RTDS symbols.
func SupportedEquitySymbols() []string {
	out := make([]string, 0,
		len(supportedEquityStockSymbols)+
			len(supportedEquityETFSymbols)+
			len(supportedEquityForexSymbols)+
			len(supportedEquityMetalSymbols)+
			len(supportedEquityCommoditySymbols))
	out = append(out, supportedEquityStockSymbols...)
	out = append(out, supportedEquityETFSymbols...)
	out = append(out, supportedEquityForexSymbols...)
	out = append(out, supportedEquityMetalSymbols...)
	out = append(out, supportedEquityCommoditySymbols...)
	return out
}

// SupportedEquityStockSymbols returns the officially documented equity stock RTDS symbols.
func SupportedEquityStockSymbols() []string {
	return cloneStrings(supportedEquityStockSymbols)
}

// SupportedEquityETFSymbols returns the officially documented equity ETF RTDS symbols.
func SupportedEquityETFSymbols() []string {
	return cloneStrings(supportedEquityETFSymbols)
}

// SupportedEquityForexSymbols returns the officially documented equity forex RTDS symbols.
func SupportedEquityForexSymbols() []string {
	return cloneStrings(supportedEquityForexSymbols)
}

// SupportedEquityMetalSymbols returns the officially documented precious metals RTDS symbols.
func SupportedEquityMetalSymbols() []string {
	return cloneStrings(supportedEquityMetalSymbols)
}

// SupportedEquityCommoditySymbols returns the officially documented commodity RTDS symbols.
func SupportedEquityCommoditySymbols() []string {
	return cloneStrings(supportedEquityCommoditySymbols)
}

// IsSupportedBinanceSymbol reports whether symbol is in the documented Binance symbol set.
func IsSupportedBinanceSymbol(symbol string) bool {
	return containsFold(supportedBinanceSymbols, symbol)
}

// IsSupportedChainlinkSymbol reports whether symbol is in the documented Chainlink symbol set.
func IsSupportedChainlinkSymbol(symbol string) bool {
	return containsFold(supportedChainlinkSymbols, symbol)
}

// IsSupportedEquitySymbol reports whether symbol is in the documented equity symbol set.
func IsSupportedEquitySymbol(symbol string) bool {
	return containsFold(SupportedEquitySymbols(), symbol)
}

// Credentials is the legacy CLOB API credential payload for authenticated RTDS topics.
//
// Deprecated: official RTDS documentation uses GammaAuth for authenticated streams.
type Credentials struct {
	// Key is the CLOB API key.
	Key string `json:"key"`
	// Secret is the CLOB API secret.
	Secret string `json:"secret"`
	// Passphrase is the CLOB API passphrase.
	Passphrase string `json:"passphrase"`
}

// GammaAuth authenticates protected RTDS topics with a wallet address.
type GammaAuth struct {
	// Address is the wallet address used for Gamma authentication.
	Address string `json:"address"`
}

// SubscriptionRequest is the top-level RTDS subscription message.
type SubscriptionRequest struct {
	// Action is subscribe or unsubscribe.
	Action Action `json:"action"`
	// Subscriptions lists the topic subscriptions to update.
	Subscriptions []Subscription `json:"subscriptions"`
}

// Subscription is a single RTDS topic subscription.
type Subscription struct {
	// Topic is the RTDS topic name, such as crypto_prices or comments.
	Topic string `json:"topic"`
	// Type filters the event type, or "*" for all types.
	Type string `json:"type"`
	// Filters applies topic-specific filters.
	Filters any `json:"filters,omitempty"`
	// GammaAuth authenticates protected topics such as comments.
	GammaAuth *GammaAuth `json:"gamma_auth,omitempty"`
	// CLOBAuth authenticates protected topics for older RTDS deployments.
	//
	// Deprecated: use GammaAuth.
	CLOBAuth *Credentials `json:"clob_auth,omitempty"`
}

// MarshalJSON serializes topic-specific RTDS filters in the shape expected by Polymarket.
func (s Subscription) MarshalJSON() ([]byte, error) {
	type subscriptionAlias Subscription
	aux := struct {
		subscriptionAlias
		Filters any `json:"filters,omitempty"`
	}{
		subscriptionAlias: subscriptionAlias(s),
		Filters:           s.Filters,
	}
	filters, ok, err := subscriptionWireFilters(s)
	if err != nil {
		return nil, err
	}
	if ok {
		aux.Filters = filters
	}
	return json.Marshal(aux)
}

// Message is the top-level message received from RTDS.
type Message struct {
	// Topic is the source RTDS topic.
	Topic string `json:"topic"`
	// Type is the topic-specific event type.
	Type string `json:"type"`
	// Timestamp is the server event timestamp in Unix milliseconds.
	Timestamp int64 `json:"timestamp"`
	// Payload is the raw topic-specific payload.
	Payload json.RawMessage `json:"payload"`
}

// AsCryptoPrice decodes Payload as a CryptoPrice.
func (m *Message) AsCryptoPrice(out *CryptoPrice) error {
	if m.Topic != TopicCryptoPrices {
		return fmt.Errorf("polymarket: RTDS topic is %q, not %s", m.Topic, TopicCryptoPrices)
	}
	return json.Unmarshal(m.Payload, out)
}

// AsChainlinkPrice decodes Payload as a ChainlinkPrice.
func (m *Message) AsChainlinkPrice(out *ChainlinkPrice) error {
	if m.Topic != TopicCryptoPricesChainlink {
		return fmt.Errorf("polymarket: RTDS topic is %q, not %s", m.Topic, TopicCryptoPricesChainlink)
	}
	return json.Unmarshal(m.Payload, out)
}

// AsEquityPrice decodes Payload as an EquityPrice live update.
func (m *Message) AsEquityPrice(out *EquityPrice) error {
	if m.Topic != TopicEquityPrices {
		return fmt.Errorf("polymarket: RTDS topic is %q, not %s", m.Topic, TopicEquityPrices)
	}
	if m.Type != TypeUpdate {
		return fmt.Errorf("polymarket: RTDS type is %q, not %s", m.Type, TypeUpdate)
	}
	return json.Unmarshal(m.Payload, out)
}

// AsEquityPriceSnapshot decodes Payload as an EquityPriceSnapshot.
func (m *Message) AsEquityPriceSnapshot(out *EquityPriceSnapshot) error {
	if m.Topic != TopicEquityPrices {
		return fmt.Errorf("polymarket: RTDS topic is %q, not %s", m.Topic, TopicEquityPrices)
	}
	if m.Type != TypeSubscribe {
		return fmt.Errorf("polymarket: RTDS type is %q, not %s", m.Type, TypeSubscribe)
	}
	return json.Unmarshal(m.Payload, out)
}

// AsComment decodes Payload as a Comment.
func (m *Message) AsComment(out *Comment) error {
	if m.Topic != TopicComments {
		return fmt.Errorf("polymarket: RTDS topic is %q, not %s", m.Topic, TopicComments)
	}
	return json.Unmarshal(m.Payload, out)
}

// CryptoPrice is a Binance crypto price update.
type CryptoPrice struct {
	// Symbol is the asset symbol.
	Symbol string `json:"symbol"`
	// Timestamp is the source timestamp.
	Timestamp int64 `json:"timestamp"`
	// Value is the price value.
	Value shared.Float64 `json:"value"`
}

// ChainlinkPrice is a Chainlink price feed update.
type ChainlinkPrice struct {
	// Symbol is the feed symbol.
	Symbol string `json:"symbol"`
	// Timestamp is the source timestamp.
	Timestamp int64 `json:"timestamp"`
	// Value is the price value.
	Value shared.Float64 `json:"value"`
}

// EquityPrice is an equity_prices live update payload.
type EquityPrice struct {
	// Symbol is the lower-case symbol identifier.
	Symbol string `json:"symbol"`
	// Value is the spot price.
	Value shared.Float64 `json:"value"`
	// FullAccuracyValue is the full-precision price.
	FullAccuracyValue string `json:"full_accuracy_value"`
	// Timestamp is the price measurement timestamp in Unix milliseconds.
	Timestamp int64 `json:"timestamp"`
	// ReceivedAt is when the system received the price in Unix milliseconds.
	ReceivedAt int64 `json:"received_at,omitempty"`
	// IsCarriedForward reports whether the value is carried forward while a market is closed.
	IsCarriedForward bool `json:"is_carried_forward,omitempty"`
}

// EquityPriceSnapshot is an equity_prices initial subscription snapshot payload.
type EquityPriceSnapshot struct {
	// Symbol is the lower-case symbol identifier.
	Symbol string `json:"symbol"`
	// Data is the historical price backfill.
	Data []EquityPricePoint `json:"data"`
}

// EquityPricePoint is one historical equity price point.
type EquityPricePoint struct {
	// Timestamp is the price measurement timestamp in Unix milliseconds.
	Timestamp int64 `json:"timestamp"`
	// Value is the spot price.
	Value shared.Float64 `json:"value"`
}

// Comment is a comment event payload.
type Comment struct {
	// ID is the comment identifier.
	ID string `json:"id"`
	// Body is the comment text.
	Body string `json:"body"`
	// CreatedAt is the creation time.
	CreatedAt time.Time `json:"createdAt"`
	// ParentCommentID is the parent comment when this is a reply.
	ParentCommentID *string `json:"parentCommentID,omitempty"`
	// ParentEntityID is the commented entity identifier.
	ParentEntityID int64 `json:"parentEntityID"`
	// ParentEntityType is the commented entity type.
	ParentEntityType string `json:"parentEntityType"`
	// Profile contains author profile fields.
	Profile CommentProfile `json:"profile"`
	// ReactionCount is the current reaction count.
	ReactionCount int64 `json:"reactionCount"`
	// ReplyAddress is the reply target address.
	ReplyAddress *string `json:"replyAddress,omitempty"`
	// ReportCount is the current report count.
	ReportCount int64 `json:"reportCount"`
	// UserAddress is the author address.
	UserAddress string `json:"userAddress"`
}

// CommentProfile contains the RTDS comment author profile.
type CommentProfile struct {
	// BaseAddress is the author's base wallet.
	BaseAddress string `json:"baseAddress"`
	// DisplayUsernamePublic reports whether the profile name is public.
	DisplayUsernamePublic bool `json:"displayUsernamePublic"`
	// Name is the profile display name.
	Name string `json:"name"`
	// ProxyWallet is the Polymarket proxy wallet.
	ProxyWallet *string `json:"proxyWallet,omitempty"`
	// Pseudonym is the optional public pseudonym.
	Pseudonym *string `json:"pseudonym,omitempty"`
}

func cloneStrings(in []string) []string {
	return append([]string(nil), in...)
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func subscriptionWireFilters(s Subscription) (any, bool, error) {
	if s.Filters == nil {
		if s.Topic == TopicCryptoPricesChainlink {
			return "", true, nil
		}
		return nil, false, nil
	}
	switch s.Topic {
	case TopicCryptoPrices:
		switch filters := s.Filters.(type) {
		case []string:
			return strings.Join(filters, ","), true, nil
		case string:
			return filters, true, nil
		}
	case TopicCryptoPricesChainlink, TopicEquityPrices:
		switch filters := s.Filters.(type) {
		case string:
			return filters, true, nil
		default:
			data, err := json.Marshal(filters)
			if err != nil {
				return nil, false, err
			}
			return string(data), true, nil
		}
	}
	return s.Filters, true, nil
}
