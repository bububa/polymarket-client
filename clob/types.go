package clob

const (
	MainnetHost = "https://clob.polymarket.com"
	V2Host      = "https://clob-v2.polymarket.com"
	AmoyHost    = "https://clob-v2.polymarket.com"

	PolygonChainID = 137
	AmoyChainID    = 80002

	ZeroBytes32 = "0x0000000000000000000000000000000000000000000000000000000000000000"
	ZeroAddress = "0x0000000000000000000000000000000000000000"
)

type Side string

const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

type OrderType string

const (
	GTC OrderType = "GTC"
	FOK OrderType = "FOK"
	GTD OrderType = "GTD"
	FAK OrderType = "FAK"
)

type SignatureType int

const (
	SignatureTypeEOA SignatureType = iota
	SignatureTypeProxy
	SignatureTypeGnosisSafe
	// SignatureTypePoly1271 is for EIP-1271 contract wallet validation.
	// Supported by the exchange contract, but not listed on the simplified
	// Create Order documentation page.
	SignatureTypePoly1271
)

type AssetType string

const (
	AssetCollateral  AssetType = "COLLATERAL"
	AssetConditional AssetType = "CONDITIONAL"
)

type TickSize string

const (
	TickSizeTenth       TickSize = "0.1"
	TickSizeHundredth   TickSize = "0.01"
	TickSizeThousandth  TickSize = "0.001"
	TickSizeTenThousand TickSize = "0.0001"
)

// Credentials holds Polymarket API authentication material returned by CreateAPIKey.
type Credentials struct {
	// Key is the API key identifier.
	Key string `json:"apiKey"`
	// Secret is the HMAC secret used to sign requests.
	Secret string `json:"secret"`
	// Passphrase protects the API key secret.
	Passphrase string `json:"passphrase"`
}

// WSAuth holds WebSocket authentication material.
type WSAuth struct {
	// APIKey is the WebSocket API key.
	APIKey string `json:"apiKey"`
	// Secret is the WebSocket HMAC secret.
	Secret string `json:"secret"`
	// Passphrase protects the WebSocket API key.
	Passphrase string `json:"passphrase"`
}

// apiKeysResponse is the envelope returned by GET /auth/api-keys.
type apiKeysResponse struct {
	APIKeys []Credentials `json:"apiKeys"`
}

// Page wraps a paginated CLOB API response.
type Page[T any] struct {
	// Limit is the requested page size.
	Limit Int `json:"limit"`
	// Count is the number of results on this page.
	Count Int `json:"count"`
	// NextCursor is the pagination cursor for the next page, empty when exhausted.
	NextCursor string `json:"next_cursor"`
	// Data contains the page results.
	Data []T `json:"data"`
}

// BookParams identifies a token/side combination for bulk price queries.
type BookParams struct {
	// TokenID is the conditional token identifier.
	TokenID string `json:"token_id" url:"token_id"`
	// Side filters by buy or sell, empty means both.
	Side Side `json:"side,omitempty" url:"side,omitempty"`
}

// OrderSummary represents a single price/size level in an order book.
type OrderSummary struct {
	// Price is the limit order price.
	Price Float64 `json:"price"`
	// Size is the available quantity at this price.
	Size Float64 `json:"size"`
}

// OrderBookSummary is a snapshot of the order book for a single asset.
type OrderBookSummary struct {
	// Market is the condition ID.
	Market string `json:"market"`
	// AssetID is the token ID.
	AssetID String `json:"asset_id"`
	// Timestamp is when the snapshot was taken.
	Timestamp Time `json:"timestamp"`
	// Bids are buy orders sorted best-first.
	Bids []OrderSummary `json:"bids"`
	// Asks are sell orders sorted best-first.
	Asks []OrderSummary `json:"asks"`
	// MinOrderSize is the minimum order quantity.
	MinOrderSize Float64 `json:"min_order_size"`
	// NegRisk indicates a negative-risk market.
	NegRisk bool `json:"neg_risk"`
	// TickSize is the minimum price increment.
	TickSize Float64 `json:"tick_size"`
	// LastTradePrice is the price of the most recent trade.
	LastTradePrice Float64 `json:"last_trade_price"`
	// Hash is the order book snapshot hash.
	Hash string `json:"hash"`
}

// Token represents a conditional outcome token.
type Token struct {
	// TokenID is the ERC-1155 token identifier.
	TokenID String `json:"token_id"`
	// Outcome is the outcome label (e.g. "Yes", "No").
	Outcome string `json:"outcome"`
	// Price is the current market price.
	Price Float64 `json:"price"`
	// Winner is true when this outcome has been resolved as correct.
	Winner bool `json:"winner,omitempty"`
}

// RewardRate describes daily rewards for a specific asset.
type RewardRate struct {
	// AssetAddress is the reward asset contract address.
	AssetAddress string `json:"asset_address"`
	// RewardsDailyRate is the daily reward amount.
	RewardsDailyRate Float64 `json:"rewards_daily_rate"`
}

// Rewards contains reward parameters for a market.
type Rewards struct {
	// Rates is the list of asset-specific reward rates.
	Rates []RewardRate `json:"rates"`
	// MinSize is the minimum order size to qualify for rewards.
	MinSize Float64 `json:"min_size"`
	// MaxSpread is the maximum bid-ask spread to qualify for rewards.
	MaxSpread Float64 `json:"max_spread"`
}

// Market is the full CLOB market data structure.
type Market struct {
	// EnableOrderBook indicates order book functionality is available.
	EnableOrderBook bool `json:"enable_order_book"`
	// Active is true when the market is accepting trades.
	Active bool `json:"active"`
	// Closed is true when the market has closed.
	Closed bool `json:"closed"`
	// Archived is true when the market is archived.
	Archived bool `json:"archived"`
	// AcceptingOrders indicates the order book is accepting new orders.
	AcceptingOrders bool `json:"accepting_orders"`
	// AcceptingOrderTimestamp is when the market started accepting orders.
	AcceptingOrderTimestamp Time `json:"accepting_order_timestamp"`
	// MinimumOrderSize is the smallest allowed order.
	MinimumOrderSize Float64 `json:"minimum_order_size"`
	// MinimumTickSize is the price increment granularity.
	MinimumTickSize Float64 `json:"minimum_tick_size"`
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"condition_id"`
	// QuestionID identifies the underlying Polymarket question.
	QuestionID string `json:"question_id"`
	// Question is the market question text.
	Question string `json:"question"`
	// Description is the market description.
	Description string `json:"description"`
	// MarketSlug is the URL-friendly market slug.
	MarketSlug string `json:"market_slug"`
	// EndDateISO is the market resolution date.
	EndDateISO Time `json:"end_date_iso"`
	// GameStartTime is the event start time.
	GameStartTime Time `json:"game_start_time"`
	// SecondsDelay is the delay period before order book acceptance.
	SecondsDelay Uint64 `json:"seconds_delay"`
	// FPMM is the Fixed Product Market Maker address.
	FPMM string `json:"fpmm"`
	// MakerBaseFee is the maker fee rate.
	MakerBaseFee Float64 `json:"maker_base_fee"`
	// TakerBaseFee is the taker fee rate.
	TakerBaseFee Float64 `json:"taker_base_fee"`
	// NotificationsEnabled indicates push notifications are active.
	NotificationsEnabled bool `json:"notifications_enabled"`
	// NegRisk indicates a negative-risk market resolution.
	NegRisk bool `json:"neg_risk"`
	// NegRiskMarketID identifies the neg-risk grouping.
	NegRiskMarketID string `json:"neg_risk_market_id"`
	// NegRiskRequestID is the original request identifier.
	NegRiskRequestID string `json:"neg_risk_request_id"`
	// Icon is the market icon URL.
	Icon string `json:"icon"`
	// Image is the market banner image URL.
	Image string `json:"image"`
	// Rewards contains reward parameters.
	Rewards Rewards `json:"rewards"`
	// Is5050Outcome is true for binary 50/50 markets.
	Is5050Outcome bool `json:"is_50_50_outcome"`
	// Tokens lists the conditional outcome tokens.
	Tokens []Token `json:"tokens"`
	// Tags categorize the market.
	Tags []string `json:"tags"`
}

// SimplifiedMarket is a lightweight market record.
type SimplifiedMarket struct {
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"condition_id"`
	// Tokens lists the conditional outcome tokens.
	Tokens []Token `json:"tokens"`
	// Rewards contains reward parameters.
	Rewards Rewards `json:"rewards"`
	// Active is true when the market is accepting trades.
	Active bool `json:"active"`
	// Closed is true when the market has closed.
	Closed bool `json:"closed"`
	// Archived is true when the market is archived.
	Archived bool `json:"archived"`
	// AcceptingOrders indicates the order book is accepting new orders.
	AcceptingOrders bool `json:"accepting_orders"`
}

// MarketByToken maps a primary and secondary token to a condition.
type MarketByToken struct {
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"condition_id"`
	// PrimaryTokenID is the winning outcome token ID.
	PrimaryTokenID String `json:"primary_token_id"`
	// SecondaryTokenID is the losing outcome token ID.
	SecondaryTokenID String `json:"secondary_token_id"`
}

// FeeInfo describes the fee structure for a CLOB market.
type FeeInfo struct {
	// Rate is the fee rate.
	Rate Float64 `json:"r"`
	// Exponent is the fee exponent.
	Exponent Float64 `json:"e"`
	// TakerOnly is true when fees apply only to takers.
	TakerOnly bool `json:"to"`
}

// ClobMarketToken is a lightweight token in ClobMarketInfo.
type ClobMarketToken struct {
	// TokenID is the conditional token identifier.
	TokenID String `json:"t"`
	// Outcome is the outcome label.
	Outcome string `json:"o"`
}

// ClobMarketInfo is the compact market info from GET /clob-markets.
type ClobMarketInfo struct {
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"c"`
	// MinimumTickSize is the price increment granularity.
	MinimumTickSize Float64 `json:"mts"`
	// MinimumOrderSize is the smallest allowed order.
	MinimumOrderSize Float64 `json:"mos"`
	// NegRisk indicates a negative-risk market resolution.
	NegRisk bool `json:"nr"`
	// FeeDetails describes the fee structure.
	FeeDetails FeeInfo `json:"fd"`
	// Tokens lists the conditional outcome tokens.
	Tokens []ClobMarketToken `json:"t"`
	// RFQEnabled indicates request-for-quote is active.
	RFQEnabled bool `json:"rfqe"`
	// MakerBaseFee is the maker fee rate.
	MakerBaseFee Float64 `json:"mbf"`
	// TakerBaseFee is the taker fee rate.
	TakerBaseFee Float64 `json:"tbf"`
}

// MidpointResponse contains the order book midpoint price.
type MidpointResponse struct {
	// Mid is the midpoint between best bid and best ask.
	Mid Float64 `json:"mid"`
}

// PriceResponse contains the current price for a token/side.
type PriceResponse struct {
	// Price is the current price.
	Price Float64 `json:"price"`
}

// SpreadResponse contains the current bid-ask spread.
type SpreadResponse struct {
	// Spread is the difference between best ask and best bid.
	Spread Float64 `json:"spread"`
}

// TickSizeResponse contains the minimum price increment.
type TickSizeResponse struct {
	// MinimumTickSize is the minimum price increment.
	MinimumTickSize TickSize `json:"minimum_tick_size"`
}

// NegRiskResponse indicates whether a market uses neg-risk resolution.
type NegRiskResponse struct {
	// NegRisk is true for neg-risk markets.
	NegRisk bool `json:"neg_risk"`
}

// FeeRateResponse contains the base fee rate for a market.
type FeeRateResponse struct {
	// BaseFee is the base fee in basis points.
	BaseFee Int `json:"base_fee"`
}

// PricePoint is a single timestamped price record.
type PricePoint struct {
	// T is the Unix timestamp.
	T Int64 `json:"t"`
	// P is the price at time T.
	P Float64 `json:"p"`
}

// PriceHistoryResponse contains a time series of price points.
type PriceHistoryResponse struct {
	// History is the ordered list of price points.
	History []PricePoint `json:"history"`
}

// BatchPriceHistoryResponse contains historical price points keyed by market asset ID.
type BatchPriceHistoryResponse struct {
	History map[string][]PricePoint `json:"history"`
}

// Rebate documents current rebated fees for a maker on one market asset.
type Rebate struct {
	// Date is the rebate date.
	Date Date `json:"date"`
	// ConditionID is the market condition identifier.
	ConditionID string `json:"condition_id"`
	// AssetAddress is the reward asset contract address.
	AssetAddress string `json:"asset_address"`
	// MakerAddress is the maker wallet address.
	MakerAddress string `json:"maker_address"`
	// RebatedFeesUSDC is the total rebated fee in USDC.
	RebatedFeesUSDC Float64 `json:"rebated_fees_usdc"`
}

// LastTradePriceResponse contains the most recent trade price.
type LastTradePriceResponse struct {
	// Price is the trade price.
	Price Float64 `json:"price"`
	// Side is the trade direction.
	Side Side `json:"side"`
}

// LastTradesPricesResponse contains per-token last trade prices.
type LastTradesPricesResponse struct {
	// TokenID is the conditional token identifier.
	TokenID String `json:"token_id"`
	// Price is the last trade price.
	Price Float64 `json:"price"`
	// Side is the trade direction.
	Side Side `json:"side"`
}

// SignedOrder is an EIP-712 signed order ready for submission.
type SignedOrder struct {
	// Salt is the order uniqueness salt. Marshalled as a JSON number
	// (not string) — Polymarket's CLOB v2 backend rejects salt-as-string
	// with the generic "Invalid order payload" error. Bug discovered
	// 2026-04-28 against production. The salt generator is bounded to
	// the current ms timestamp (≤ ~1.78e12) so int64 safely fits.
	Salt String `json:"salt"`
	// Maker is the order creator address.
	Maker string `json:"maker"`
	// Signer is the signing authority address.
	Signer string `json:"signer"`
	// TokenID is the conditional token identifier.
	TokenID String `json:"tokenId"`
	// MakerAmount is the amount the maker offers.
	MakerAmount String `json:"makerAmount"`
	// TakerAmount is the amount the maker wants.
	TakerAmount String `json:"takerAmount"`
	// Expiration is the order expiry Unix timestamp (GTD orders only).
	// Present in REST wire format but excluded from EIP-712 signing.
	Expiration String `json:"expiration,omitempty"`
	// Side is the order direction.
	Side Side `json:"side"`
	// SignatureType identifies the signature method.
	SignatureType SignatureType `json:"signatureType"`
	// Timestamp is when the order was created.
	Timestamp String `json:"timestamp"`
	// Metadata is optional order metadata.
	Metadata string `json:"metadata"`
	// Builder is the builder code for fee splits.
	Builder string `json:"builder"`
	// Signature is the EIP-712 signature bytes.
	Signature string `json:"signature"`
}

// PostOrderRequest wraps a SignedOrder for POST /order.
type PostOrderRequest struct {
	// Order is the signed order payload.
	Order SignedOrder `json:"order"`
	// Owner is the order owner address.
	Owner string `json:"owner"`
	// OrderType is the execution type (GTC, FOK, GTD).
	OrderType OrderType `json:"orderType"`
	// DeferExec defers order execution for later processing.
	DeferExec *bool `json:"deferExec,omitempty"`
}

// PostOrderResponse is returned after order submission.
type PostOrderResponse struct {
	// Success is true when the order was accepted.
	Success bool `json:"success"`
	// ErrorMsg contains the error description on failure.
	ErrorMsg string `json:"errorMsg"`
	// OrderID is the assigned order identifier.
	OrderID string `json:"orderID"`
	// TransactionHashes lists on-chain transaction hashes.
	TransactionHashes []string `json:"transactionsHashes"`
	// Status is the order processing status.
	Status string `json:"status"`
	// TakingAmount is the amount filled from the order.
	TakingAmount Float64 `json:"takingAmount"`
	// MakingAmount is the remaining order amount.
	MakingAmount Float64 `json:"makingAmount"`
	// TradeIDs lists the resulting trade identifiers.
	TradeIDs []string `json:"trade_ids"`
}

// OpenOrder represents an active order returned by GET /data/orders.
type OpenOrder struct {
	// ID is the order identifier.
	ID string `json:"id"`
	// Status is the current order status.
	Status string `json:"status"`
	// Owner is the order owner address.
	Owner string `json:"owner"`
	// MakerAddress is the maker wallet address.
	MakerAddress string `json:"maker_address"`
	// Market is the condition ID the order belongs to.
	Market string `json:"market"`
	// AssetID is the conditional token identifier.
	AssetID String `json:"asset_id"`
	// Side is the order direction.
	Side Side `json:"side"`
	// OriginalSize is the initial order quantity.
	OriginalSize Float64 `json:"original_size"`
	// SizeMatched is the quantity already filled.
	SizeMatched Float64 `json:"size_matched"`
	// Price is the limit price.
	Price Float64 `json:"price"`
	// AssociateTrades lists the trade IDs from fills.
	AssociateTrades []string `json:"associate_trades"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// CreatedAt is when the order was placed.
	CreatedAt Time `json:"created_at"`
	// Expiration is the order expiry time.
	Expiration Time `json:"expiration"`
	// OrderType is the execution type.
	OrderType OrderType `json:"order_type"`
}

// MakerOrder represents a single maker fill in a trade.
type MakerOrder struct {
	// OrderID is the order that was filled.
	OrderID string `json:"order_id"`
	// Owner is the order owner address.
	Owner string `json:"owner"`
	// MakerAddress is the maker wallet address.
	MakerAddress string `json:"maker_address"`
	// MatchedAmount is the quantity matched.
	MatchedAmount Float64 `json:"matched_amount"`
	// Price is the fill price.
	Price Float64 `json:"price"`
	// FeeRateBps is the fee rate in basis points.
	FeeRateBps Float64 `json:"fee_rate_bps"`
	// AssetID is the conditional token identifier.
	AssetID String `json:"asset_id"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// Side is the order direction.
	Side Side `json:"side"`
}

// Trade represents a matched fill between maker and taker.
type Trade struct {
	// ID is the trade identifier.
	ID string `json:"id"`
	// TakerOrderID is the taker's order ID.
	TakerOrderID string `json:"taker_order_id"`
	// Market is the condition ID.
	Market string `json:"market"`
	// AssetID is the conditional token identifier.
	AssetID String `json:"asset_id"`
	// Side is the trade direction.
	Side Side `json:"side"`
	// Size is the matched quantity.
	Size Float64 `json:"size"`
	// FeeRateBps is the fee rate in basis points.
	FeeRateBps Float64 `json:"fee_rate_bps"`
	// Price is the execution price.
	Price Float64 `json:"price"`
	// Status is the trade processing status.
	Status string `json:"status"`
	// MatchTime is when the trade was matched.
	MatchTime Time `json:"match_time"`
	// LastUpdate is the last status change.
	LastUpdate Time `json:"last_update"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// BucketIndex is the price bucket index.
	BucketIndex Int `json:"bucket_index"`
	// Owner is the trade owner address.
	Owner string `json:"owner"`
	// MakerAddress is the maker wallet address.
	MakerAddress string `json:"maker_address"`
	// MakerOrders lists the maker orders that were filled.
	MakerOrders []MakerOrder `json:"maker_orders"`
	// TransactionHash is the on-chain transaction hash.
	TransactionHash string `json:"transaction_hash"`
	// TraderSide identifies taker or maker role.
	TraderSide string `json:"trader_side"`
	// ErrorMsg contains error details on failure.
	ErrorMsg string `json:"error_msg"`
}

// CancelOrdersResponse reports cancellation results.
type CancelOrdersResponse struct {
	// Canceled lists successfully canceled order IDs.
	Canceled []string `json:"canceled"`
	// NotCanceled maps failed order IDs to error messages.
	NotCanceled map[string]string `json:"not_canceled"`
}

// BalanceAllowanceParams filters balance and allowance queries.
type BalanceAllowanceParams struct {
	// AssetType specifies collateral or conditional.
	AssetType AssetType `json:"asset_type" url:"asset_type"`
	// TokenID is the conditional token (required for conditional assets).
	TokenID string `json:"token_id,omitempty" url:"token_id,omitempty"`
	// SignatureType identifies the signature method.
	SignatureType SignatureType `json:"signature_type,omitempty" url:"signature_type,omitempty"`
}

// BalanceAllowanceResponse contains user balance and allowance info.
type BalanceAllowanceResponse struct {
	// Balance is the current token balance.
	Balance Float64 `json:"balance"`
	// Allowance is the approved amount for the exchange contract.
	Allowance Float64 `json:"allowance,omitempty"`
	// Allowances maps spender addresses to approved amounts.
	Allowances map[string]string `json:"allowances,omitempty"`
}

// Notification is a user notification from the CLOB.
type Notification struct {
	// Type identifies the notification category.
	Type Int `json:"type"`
	// Owner is the recipient address.
	Owner string `json:"owner"`
	// Payload contains the notification data.
	Payload jsonRawObject `json:"payload"`
}

type jsonRawObject map[string]any

// BanStatus reports whether a user is in closed-only mode.
type BanStatus struct {
	// ClosedOnly is true when the account can only reduce positions.
	ClosedOnly bool `json:"closed_only"`
}

// OrderScoring reports whether an order is actively scoring.
type OrderScoring struct {
	// Scoring is true when the order is in the scoring queue.
	Scoring bool `json:"scoring"`
}

// GeoblockResponse reports geographic access restrictions.
type GeoblockResponse struct {
	// Blocked is true when the IP is in a restricted jurisdiction.
	Blocked bool `json:"blocked"`
	// IP is the detected client IP.
	IP string `json:"ip"`
	// Country is the ISO country code.
	Country string `json:"country"`
	// Region is the geographic region.
	Region string `json:"region"`
}

// UserEarning records a maker's reward earning for one market asset.
type UserEarning struct {
	// Date is the earning date.
	Date Date `json:"date"`
	// ConditionID is the market condition identifier.
	ConditionID string `json:"condition_id"`
	// AssetAddress is the reward asset contract address.
	AssetAddress string `json:"asset_address"`
	// MakerAddress is the maker wallet address.
	MakerAddress string `json:"maker_address"`
	// Earnings is the total reward earned in USDC.
	Earnings Float64 `json:"earnings"`
	// AssetRate is the reward asset exchange rate.
	AssetRate Float64 `json:"asset_rate"`
}

type UserRewardsEarning struct {
	ConditionID           string          `json:"condition_id"`
	Question              string          `json:"question"`
	MarketSlug            string          `json:"market_slug"`
	EventSlug             string          `json:"event_slug"`
	Image                 string          `json:"image"`
	RewardsMaxSpread      Float64         `json:"rewards_max_spread"`
	RewardsMinSize        Float64         `json:"rewards_min_size"`
	MarketCompetitiveness Float64         `json:"market_competitiveness"`
	Tokens                []Token         `json:"tokens"`
	RewardsConfig         []RewardsConfig `json:"rewards_config"`
	MakerAddress          string          `json:"maker_address"`
	EarningPercentage     Float64         `json:"earning_percentage"`
	Earnings              []Earning       `json:"earnings"`
}

type RewardsConfig struct {
	AssetAddress string  `json:"asset_address"`
	StartDate    Date    `json:"start_date"`
	EndDate      Date    `json:"end_date"`
	RatePerDay   Float64 `json:"rate_per_day"`
	TotalRewards Float64 `json:"total_rewards"`
}

// MarketRewardsConfig describes the reward period for one market asset.
type MarketRewardsConfig struct {
	// ID is the config identifier.
	ID String `json:"id"`
	// AssetAddress is the reward asset contract address.
	AssetAddress string `json:"asset_address"`
	// StartDate is the reward start date.
	StartDate Date `json:"start_date"`
	// EndDate is the reward end date.
	EndDate Date `json:"end_date"`
	// RatePerDay is the daily reward rate.
	RatePerDay Float64 `json:"rate_per_day"`
	// TotalRewards is the total reward pool.
	TotalRewards Float64 `json:"total_rewards"`
	// TotalDays is the duration in days.
	TotalDays Float64 `json:"total_days"`
}

// Earning records a single reward earning entry.
type Earning struct {
	// AssetAddress is the reward asset contract address.
	AssetAddress string `json:"asset_address"`
	// Earnings is the total reward amount.
	Earnings Float64 `json:"earnings"`
	// AssetRate is the reward asset exchange rate.
	AssetRate Float64 `json:"asset_rate"`
}

// CurrentReward describes active reward configuration for a market.
type CurrentReward struct {
	// ConditionID is the market condition identifier.
	ConditionID string `json:"condition_id"`
	// RewardsConfig lists the active reward periods.
	RewardsConfig []RewardsConfig `json:"rewards_config"`
	// RewardsMaxSpread is the max spread to qualify for rewards.
	RewardsMaxSpread Float64 `json:"rewards_max_spread"`
	// RewardsMinSize is the minimum order size to qualify.
	RewardsMinSize Float64 `json:"rewards_min_size"`
}

// MarketReward describes reward eligibility for a specific market.
type MarketReward struct {
	// ConditionID is the market condition identifier.
	ConditionID string `json:"condition_id"`
	// Question is the market question text.
	Question string `json:"question"`
	// MarketSlug is the URL-friendly market slug.
	MarketSlug string `json:"market_slug"`
	// EventSlug is the parent event slug.
	EventSlug string `json:"event_slug"`
	// Image is the market banner image URL.
	Image string `json:"image"`
	// RewardsMaxSpread is the max spread to qualify.
	RewardsMaxSpread Float64 `json:"rewards_max_spread"`
	// RewardsMinSize is the minimum order size to qualify.
	RewardsMinSize Float64 `json:"rewards_min_size"`
	// MarketCompetitiveness indicates market saturation.
	MarketCompetitiveness Float64 `json:"market_competitiveness"`
	// Tokens lists the conditional outcome tokens.
	Tokens []Token `json:"tokens"`
	// RewardsConfig lists the active reward periods.
	RewardsConfig []MarketRewardsConfig `json:"rewards_config"`
}

// BuilderAPIKey describes a builder API key.
type BuilderAPIKey struct {
	// Key is the API key value.
	Key string `json:"key"`
	// CreatedAt is when the key was created.
	CreatedAt Time `json:"createdAt"`
	// RevokedAt is when the key was revoked, zero if active.
	RevokedAt Time `json:"revokedAt"`
}

// BuilderTrade represents a trade attributed to a builder.
type BuilderTrade struct {
	// ID is the trade identifier.
	ID string `json:"id"`
	// TradeType identifies the trade category.
	TradeType string `json:"tradeType"`
	// TakerOrderHash is the taker order hash.
	TakerOrderHash string `json:"takerOrderHash"`
	// Builder is the builder code.
	Builder string `json:"builder"`
	// Market is the condition ID.
	Market string `json:"market"`
	// AssetID is the conditional token identifier.
	AssetID String `json:"assetId"`
	// Side is the trade direction.
	Side Side `json:"side"`
	// Size is the matched quantity.
	Size Float64 `json:"size"`
	// SizeUSDC is the quantity in USDC.
	SizeUSDC Float64 `json:"sizeUsdc"`
	// Price is the execution price.
	Price Float64 `json:"price"`
	// Status is the trade processing status.
	Status string `json:"status"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// OutcomeIndex is the 0-based index of the outcome.
	OutcomeIndex Int `json:"outcomeIndex"`
	// Owner is the trade owner address.
	Owner string `json:"owner"`
	// Maker is the maker wallet address.
	Maker string `json:"maker"`
	// TransactionHash is the on-chain transaction hash.
	TransactionHash string `json:"transactionHash"`
	// MatchTime is when the trade was matched.
	MatchTime Time `json:"matchTime"`
	// BucketIndex is the price bucket index.
	BucketIndex Int `json:"bucketIndex"`
	// Fee is the fee amount.
	Fee Float64 `json:"fee"`
	// FeeUSDC is the fee in USDC.
	FeeUSDC Float64 `json:"feeUsdc"`
	// ErrMsg contains error details on failure.
	ErrMsg string `json:"errMsg"`
	// CreatedAt is when the trade record was created.
	CreatedAt Time `json:"createdAt"`
	// UpdatedAt is the last record update.
	UpdatedAt Time `json:"updatedAt"`
}

// BuilderFeeRate contains the fee rates for a builder.
type BuilderFeeRate struct {
	// BuilderMakerFeeRateBps is the maker fee in basis points.
	BuilderMakerFeeRateBps Int `json:"builderMakerFeeRateBps"`
	// BuilderTakerFeeRateBps is the taker fee in basis points.
	BuilderTakerFeeRateBps Int `json:"builderTakerFeeRateBps"`
}

// HeartbeatResponse is returned by the heartbeat endpoint.
type HeartbeatResponse struct {
	// HeartbeatID is the unique heartbeat identifier.
	HeartbeatID string `json:"heartbeat_id"`
	// Error contains error details on failure.
	Error string `json:"error"`
}

// ReadonlyAPIKey describes a read-only API key.
type ReadonlyAPIKey struct {
	// APIKey is the read-only key value.
	APIKey string `json:"apiKey"`
}

// RfqRequest represents an RFQ (request for quote) submission.
type RfqRequest struct {
	// RequestID is the unique request identifier.
	RequestID string `json:"requestId"`
	// UserAddress is the requester wallet address.
	UserAddress string `json:"userAddress"`
	// ProxyAddress is the proxy wallet address.
	ProxyAddress string `json:"proxyAddress"`
	// Condition is the condition the request applies to.
	Condition string `json:"condition"`
	// Token is the asset being offered.
	Token String `json:"token"`
	// Complement is the complementary asset.
	Complement String `json:"complement"`
	// Side is the requested trade direction.
	Side Side `json:"side"`
	// SizeIn is the input quantity.
	SizeIn Float64 `json:"sizeIn"`
	// SizeOut is the output quantity.
	SizeOut Float64 `json:"sizeOut"`
	// Price is the requested price.
	Price Float64 `json:"price"`
	// Expiry is the request expiry Unix timestamp.
	Expiry Int64 `json:"expiry"`
}

// RfqQuote represents a quote generated in response to an RFQ.
type RfqQuote struct {
	// QuoteID is the unique quote identifier.
	QuoteID string `json:"quoteId"`
	// RequestID references the original RFQ request.
	RequestID string `json:"requestId"`
	// UserAddress is the requester wallet address.
	UserAddress string `json:"userAddress"`
	// ProxyAddress is the proxy wallet address.
	ProxyAddress string `json:"proxyAddress"`
	// Condition is the condition the quote applies to.
	Condition string `json:"condition"`
	// Token is the asset being offered.
	Token String `json:"token"`
	// Complement is the complementary asset.
	Complement String `json:"complement"`
	// Side is the trade direction.
	Side Side `json:"side"`
	// SizeIn is the input quantity.
	SizeIn Float64 `json:"sizeIn"`
	// SizeOut is the output quantity.
	SizeOut Float64 `json:"sizeOut"`
	// Price is the quoted price.
	Price Float64 `json:"price"`
}
