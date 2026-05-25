package data

import (
	"encoding/json"
	"net/url"

	pmtypes "github.com/bububa/polymarket-client/shared"
)

// Side is the trade side.
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// Health is the Data API health response.
type Health struct {
	// Data is the health status string.
	Data string `json:"data"`
}

// Position describes a current user position.
type Position struct {
	// ProxyWallet is the user's Polymarket proxy wallet.
	ProxyWallet string `json:"proxyWallet"`
	// Asset is the ERC-1155 token address.
	Asset string `json:"asset"`
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"conditionId"`
	// Size is the current position size.
	Size pmtypes.Float64 `json:"size"`
	// AvgPrice is the average entry price.
	AvgPrice pmtypes.Float64 `json:"avgPrice"`
	// InitialValue is the position cost in USDC.
	InitialValue pmtypes.Float64 `json:"initialValue"`
	// CurrentValue is the current market value.
	CurrentValue pmtypes.Float64 `json:"currentValue"`
	// CashPNL is the realized profit/loss in USDC.
	CashPNL pmtypes.Float64 `json:"cashPnl"`
	// PercentPNL is the realized profit/loss percentage.
	PercentPNL pmtypes.Float64 `json:"percentPnl"`
	// TotalBought is the total USDC spent.
	TotalBought pmtypes.Float64 `json:"totalBought"`
	// RealizedPNL is the realized profit/loss from sales.
	RealizedPNL pmtypes.Float64 `json:"realizedPnl"`
	// PercentRealizedPNL is the realized profit/loss percentage.
	PercentRealizedPNL pmtypes.Float64 `json:"percentRealizedPnl"`
	// CurPrice is the current market price.
	CurPrice pmtypes.Float64 `json:"curPrice"`
	// Redeemable is true when the position can be redeemed.
	Redeemable bool `json:"redeemable"`
	// Mergeable is true when positions can be merged.
	Mergeable bool `json:"mergeable"`
	// Title is the market question text.
	Title string `json:"title"`
	// Slug is the URL-friendly market slug.
	Slug string `json:"slug"`
	// Icon is the market icon URL.
	Icon string `json:"icon"`
	// EventSlug is the parent event slug.
	EventSlug string `json:"eventSlug"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// OutcomeIndex is the 0-based index of the outcome.
	OutcomeIndex pmtypes.Int `json:"outcomeIndex"`
	// OppositeOutcome is the complementary outcome label.
	OppositeOutcome string `json:"oppositeOutcome"`
	// OppositeAsset is the complementary token address.
	OppositeAsset string `json:"oppositeAsset"`
	// EndDate is the market resolution date.
	EndDate string `json:"endDate"`
	// NegativeRisk is true for neg-risk markets.
	NegativeRisk bool `json:"negativeRisk"`
	// Raw contains the unparsed JSON response.
	Raw json.RawMessage `json:"-"`
}

// ClosedPosition describes a closed user position.
type ClosedPosition struct {
	// ProxyWallet is the user's Polymarket proxy wallet.
	ProxyWallet string `json:"proxyWallet"`
	// Asset is the ERC-1155 token address.
	Asset string `json:"asset"`
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"conditionId"`
	// AvgPrice is the average entry price.
	AvgPrice pmtypes.Float64 `json:"avgPrice"`
	// RealizedPNL is the profit/loss.
	RealizedPNL pmtypes.Float64 `json:"realizedPnl"`
	// PercentRealizedPNL is the profit/loss percentage.
	PercentRealizedPNL pmtypes.Float64 `json:"percentRealizedPnl"`
	// CurPrice is the current market price.
	CurPrice pmtypes.Float64 `json:"curPrice"`
	// Timestamp is when the position was closed.
	Timestamp pmtypes.Int64 `json:"timestamp"`
	// Title is the market question text.
	Title string `json:"title"`
	// Slug is the URL-friendly market slug.
	Slug string `json:"slug"`
	// Icon is the market icon URL.
	Icon string `json:"icon"`
	// EventSlug is the parent event slug.
	EventSlug string `json:"eventSlug"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// OutcomeIndex is the 0-based index of the outcome.
	OutcomeIndex pmtypes.Int `json:"outcomeIndex"`
	// OppositeOutcome is the complementary outcome label.
	OppositeOutcome string `json:"oppositeOutcome"`
	// OppositeAsset is the complementary token address.
	OppositeAsset string `json:"oppositeAsset"`
	// EndDate is the market resolution date.
	EndDate string `json:"endDate"`
	// Raw contains the unparsed JSON response.
	Raw json.RawMessage `json:"-"`
}

// Value describes user portfolio value for a market.
type Value struct {
	// User is the wallet address.
	User string `json:"user"`
	// Market is the condition ID.
	Market string `json:"market"`
	// Value is the total portfolio value.
	Value pmtypes.Float64 `json:"value"`
	// Cash is the USDC balance.
	Cash pmtypes.Float64 `json:"cash"`
	// Tokens is the token position value.
	Tokens pmtypes.Float64 `json:"tokens"`
}

// Trade describes a Data API trade.
type Trade struct {
	// ProxyWallet is the user's Polymarket proxy wallet.
	ProxyWallet string `json:"proxyWallet"`
	// User is the wallet address.
	User string `json:"user"`
	// Name is the user's display name.
	Name string `json:"name"`
	// Pseudonym is the user's pseudonym.
	Pseudonym string `json:"pseudonym"`
	// Bio is the user's profile bio.
	Bio string `json:"bio"`
	// ProfileImage is the user's profile image URL.
	ProfileImage string `json:"profileImage"`
	// ProfileImageOptimized is the optimized profile image URL.
	ProfileImageOptimized string `json:"profileImageOptimized"`
	// Side is the trade direction.
	Side Side `json:"side"`
	// Asset is the ERC-1155 token address.
	Asset string `json:"asset"`
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"conditionId"`
	// Size is the trade quantity.
	Size pmtypes.Float64 `json:"size"`
	// Price is the execution price.
	Price pmtypes.Float64 `json:"price"`
	// Timestamp is when the trade occurred.
	Timestamp pmtypes.Int64 `json:"timestamp"`
	// Title is the market question text.
	Title string `json:"title"`
	// Slug is the URL-friendly market slug.
	Slug string `json:"slug"`
	// Icon is the market icon URL.
	Icon string `json:"icon"`
	// EventSlug is the parent event slug.
	EventSlug string `json:"eventSlug"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// OutcomeIndex is the 0-based index of the outcome.
	OutcomeIndex pmtypes.Int `json:"outcomeIndex"`
	// TransactionHash is the on-chain transaction hash.
	TransactionHash string `json:"transactionHash"`
	// Raw contains the unparsed JSON response.
	Raw json.RawMessage `json:"-"`
}

// Activity describes a user activity event.
type Activity struct {
	// ProxyWallet is the user's Polymarket proxy wallet.
	ProxyWallet string `json:"proxyWallet"`
	// User is the wallet address.
	User string `json:"user"`
	// Name is the user's display name.
	Name string `json:"name"`
	// Pseudonym is the user's pseudonym.
	Pseudonym string `json:"pseudonym"`
	// Bio is the user's profile bio.
	Bio string `json:"bio"`
	// ProfileImage is the user's profile image URL.
	ProfileImage string `json:"profileImage"`
	// ProfileImageOptimized is the optimized profile image URL.
	ProfileImageOptimized string `json:"profileImageOptimized"`
	// Timestamp is when the activity occurred.
	Timestamp pmtypes.Int64 `json:"timestamp"`
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"conditionId"`
	// Type is the activity category.
	Type string `json:"type"`
	// Size is the token quantity.
	Size pmtypes.Float64 `json:"size"`
	// USDCSize is the USDC value.
	USDCSize pmtypes.Float64 `json:"usdcSize"`
	// TransactionHash is the on-chain transaction hash.
	TransactionHash string `json:"transactionHash"`
	// Price is the activity price.
	Price pmtypes.Float64 `json:"price"`
	// Asset is the ERC-1155 token address.
	Asset string `json:"asset"`
	// Side is the trade direction.
	Side string `json:"side"`
	// Title is the market question text.
	Title string `json:"title"`
	// Slug is the URL-friendly market slug.
	Slug string `json:"slug"`
	// Icon is the market icon URL.
	Icon string `json:"icon"`
	// EventSlug is the parent event slug.
	EventSlug string `json:"eventSlug"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// OutcomeIndex is the 0-based index of the outcome.
	OutcomeIndex pmtypes.Int `json:"outcomeIndex"`
	// Raw contains the unparsed JSON response.
	Raw json.RawMessage `json:"-"`
}

// Holder describes a market holder.
type Holder struct {
	// ProxyWallet is the holder's proxy wallet.
	ProxyWallet string `json:"proxyWallet"`
	// Name is the display name.
	Name string `json:"name"`
	// Pseudonym is the user's pseudonym.
	Pseudonym string `json:"pseudonym"`
	// Bio is the user's profile bio.
	Bio string `json:"bio"`
	// ProfileImage is the profile image URL.
	ProfileImage string `json:"profileImage"`
	// ProfileImageOptimized is the optimized profile image URL.
	ProfileImageOptimized string `json:"profileImageOptimized"`
	// DisplayUsernamePublic indicates whether the display username is public.
	DisplayUsernamePublic bool `json:"displayUsernamePublic"`
	// Asset is the ERC-1155 token address.
	Asset string `json:"asset"`
	// Amount is the token balance.
	Amount pmtypes.Float64 `json:"amount"`
	// OutcomeIndex is the 0-based index of the outcome.
	OutcomeIndex pmtypes.Int `json:"outcomeIndex"`
	// Verified is true for verified accounts.
	Verified bool `json:"verified"`
	// Raw contains the unparsed JSON response.
	Raw json.RawMessage `json:"-"`
}

// Traded contains the total markets traded response.
type Traded struct {
	// User is the wallet address.
	User string `json:"user"`
	// Traded is the number of unique markets traded.
	Traded pmtypes.Int `json:"traded"`
}

// OpenInterest describes market open interest.
type OpenInterest struct {
	// Market is the condition ID.
	Market string `json:"market"`
	// Value is the open interest in USDC.
	Value pmtypes.Float64 `json:"value"`
}

// LiveVolume describes live volume for an event or market.
type LiveVolume struct {
	// Market is the condition ID.
	Market string `json:"market"`
	// Volume is the total traded volume in USDC.
	Volume pmtypes.Float64 `json:"volume"`
}

// MarketPositions groups user positions for a single outcome token.
type MarketPositions struct {
	// Token is the outcome token identifier.
	Token string `json:"token"`
	// Positions lists the user positions.
	Positions []MarketPosition `json:"positions"`
	// Raw contains the unparsed JSON response.
	Raw json.RawMessage `json:"-"`
}

// MarketPosition describes a user's position in a market outcome token.
type MarketPosition struct {
	// ProxyWallet is the user's proxy wallet.
	ProxyWallet string `json:"proxyWallet"`
	// Name is the display name.
	Name string `json:"name"`
	// ProfileImage is the profile image URL.
	ProfileImage string `json:"profileImage"`
	// Verified is true for verified accounts.
	Verified bool `json:"verified"`
	// Asset is the ERC-1155 token address.
	Asset string `json:"asset"`
	// ConditionID is the CTF condition identifier.
	ConditionID string `json:"conditionId"`
	// AvgPrice is the average entry price.
	AvgPrice pmtypes.Float64 `json:"avgPrice"`
	// Size is the position size.
	Size pmtypes.Float64 `json:"size"`
	// CurrPrice is the current market price.
	CurrPrice pmtypes.Float64 `json:"currPrice"`
	// CurrentValue is the current market value.
	CurrentValue pmtypes.Float64 `json:"currentValue"`
	// CashPNL is the realized profit/loss in USDC.
	CashPNL pmtypes.Float64 `json:"cashPnl"`
	// TotalBought is the total USDC spent.
	TotalBought pmtypes.Float64 `json:"totalBought"`
	// RealizedPNL is the realized profit/loss.
	RealizedPNL pmtypes.Float64 `json:"realizedPnl"`
	// TotalPNL is the total profit/loss.
	TotalPNL pmtypes.Float64 `json:"totalPnl"`
	// Outcome is the outcome label.
	Outcome string `json:"outcome"`
	// OutcomeIndex is the 0-based index of the outcome.
	OutcomeIndex pmtypes.Int `json:"outcomeIndex"`
	// Raw contains the unparsed JSON response.
	Raw json.RawMessage `json:"-"`
}

func (p *Position) UnmarshalJSON(data []byte) error {
	type alias Position
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*p = Position(out)
	p.Raw = append(p.Raw[:0], data...)
	return nil
}

func (p *ClosedPosition) UnmarshalJSON(data []byte) error {
	type alias ClosedPosition
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*p = ClosedPosition(out)
	p.Raw = append(p.Raw[:0], data...)
	return nil
}

func (t *Trade) UnmarshalJSON(data []byte) error {
	type alias Trade
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*t = Trade(out)
	t.Raw = append(t.Raw[:0], data...)
	return nil
}

func (a *Activity) UnmarshalJSON(data []byte) error {
	type alias Activity
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*a = Activity(out)
	a.Raw = append(a.Raw[:0], data...)
	return nil
}

func (h *Holder) UnmarshalJSON(data []byte) error {
	type alias Holder
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*h = Holder(out)
	h.Raw = append(h.Raw[:0], data...)
	return nil
}

func (p *MarketPositions) UnmarshalJSON(data []byte) error {
	type alias MarketPositions
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*p = MarketPositions(out)
	p.Raw = append(p.Raw[:0], data...)
	return nil
}

func (p *MarketPosition) UnmarshalJSON(data []byte) error {
	type alias MarketPosition
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*p = MarketPosition(out)
	p.Raw = append(p.Raw[:0], data...)
	return nil
}

type LeaderboardEntry struct {
	// Rank is the leaderboard rank.
	Rank pmtypes.String `json:"rank"`
	// ProxyWallet is the user's proxy wallet address.
	ProxyWallet string `json:"proxyWallet"`
	// UserName is the display username.
	UserName string `json:"userName"`
	// Vol is the leaderboard volume.
	Vol pmtypes.Float64 `json:"vol"`
	// PNL is the leaderboard profit and loss.
	PNL pmtypes.Float64 `json:"pnl"`
	// ProfileImage is the user's profile image URL.
	ProfileImage string `json:"profileImage"`
	// XUsername is the user's X username.
	XUsername string `json:"xUsername"`
	// VerifiedBadge indicates whether the user has a verified badge.
	VerifiedBadge bool `json:"verifiedBadge"`
}

// BuilderLeaderboardEntry describes an aggregated builder leaderboard row.
type BuilderLeaderboardEntry struct {
	Builder     string          `json:"builder"`
	Volume      pmtypes.Float64 `json:"volume"`
	Trades      pmtypes.Int     `json:"trades"`
	Rank        pmtypes.String  `json:"rank"`
	DisplayName string          `json:"displayName"`
}

// BuilderVolume describes a daily builder volume time-series row.
type BuilderVolume struct {
	Date    string          `json:"date"`
	Builder string          `json:"builder"`
	Volume  pmtypes.Float64 `json:"volume"`
	Trades  pmtypes.Int     `json:"trades"`
}

// PositionParams filters GET /positions requests.
type PositionParams struct {
	// User is the wallet address to query.
	User string
	// Markets filters by condition IDs.
	Markets []string
	// EventIDs filters by event IDs.
	EventIDs []int
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
	// SortBy sets the sort field.
	SortBy string
	// SortDirection sets the sort order.
	SortDirection string
	// Title filters by market title.
	Title string
	// Redeemable filters by redeemable status.
	Redeemable *bool
	// Mergeable filters by mergeable status.
	Mergeable *bool
	// SizeThreshold filters by minimum position size.
	SizeThreshold string
}

func (p PositionParams) values() url.Values {
	q := url.Values{}
	setString(q, "user", p.User)
	setCommaList(q, "market", p.Markets)
	setIntList(q, "eventId", p.EventIDs)
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	setString(q, "sortBy", p.SortBy)
	setString(q, "sortDirection", p.SortDirection)
	setString(q, "title", p.Title)
	setString(q, "sizeThreshold", p.SizeThreshold)
	setBool(q, "redeemable", p.Redeemable)
	setBool(q, "mergeable", p.Mergeable)
	return q
}

// ClosedPositionParams is an alias for PositionParams.
type ClosedPositionParams = PositionParams

// MarketPositionsParams filters GET /v1/market-positions requests.
type MarketPositionsParams struct {
	// Market is the condition ID.
	Market string
	// User is the wallet address.
	User string
	// Status filters by position status.
	Status string
	// SortBy sets the sort field.
	SortBy string
	// SortDirection sets the sort order.
	SortDirection string
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
}

func (p MarketPositionsParams) values() url.Values {
	q := url.Values{}
	setString(q, "market", p.Market)
	setString(q, "user", p.User)
	setString(q, "status", p.Status)
	setString(q, "sortBy", p.SortBy)
	setString(q, "sortDirection", p.SortDirection)
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	return q
}

// TradeParams filters GET /trades requests.
type TradeParams struct {
	// User is the wallet address.
	User string
	// Markets filters by condition IDs.
	Markets []string
	// EventIDs filters by event IDs.
	EventIDs []int
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
	// TakerOnly filters to taker-side trades.
	TakerOnly *bool
	// Side filters by BUY or SELL.
	Side Side
	// FilterType selects an amount comparator such as CASH, TOKENS, or SHARES.
	FilterType string
	// FilterAmount is the comparator threshold amount.
	FilterAmount string
}

func (p TradeParams) values() url.Values {
	q := url.Values{}
	setString(q, "user", p.User)
	setCommaList(q, "market", p.Markets)
	setIntList(q, "eventId", p.EventIDs)
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	setBool(q, "takerOnly", p.TakerOnly)
	setString(q, "side", string(p.Side))
	setString(q, "filterType", p.FilterType)
	setString(q, "filterAmount", p.FilterAmount)
	return q
}

// ActivityParams filters GET /activity requests.
type ActivityParams struct {
	// User is the wallet address.
	User string
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
	// Start is the start Unix timestamp.
	Start int64
	// End is the end Unix timestamp.
	End int64
	// SortBy sets the sort field.
	SortBy string
	// SortDirection sets the sort order.
	SortDirection string
	// Side filters by trade direction.
	Side Side
	// ActivityTypes filters by event categories.
	ActivityTypes []string
}

func (p ActivityParams) values() url.Values {
	q := url.Values{}
	setString(q, "user", p.User)
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	setInt64(q, "start", p.Start)
	setInt64(q, "end", p.End)
	setString(q, "sortBy", p.SortBy)
	setString(q, "sortDirection", p.SortDirection)
	setString(q, "side", string(p.Side))
	setCommaList(q, "type", p.ActivityTypes)
	return q
}

// HoldersParams filters GET /holders requests.
type HoldersParams struct {
	// Markets filters by condition IDs.
	Markets []string
	// Limit sets the maximum results.
	Limit int
	// MinBalance filters by minimum token balance.
	MinBalance int
}

func (p HoldersParams) values() url.Values {
	q := url.Values{}
	setCommaList(q, "market", p.Markets)
	setInt(q, "limit", p.Limit)
	setInt(q, "minBalance", p.MinBalance)
	return q
}

// LiveVolumeParams filters GET /live-volume requests.
type LiveVolumeParams struct {
	// Markets filters by condition IDs.
	Markets []string
}

func (p LiveVolumeParams) values() url.Values {
	q := url.Values{}
	setCommaList(q, "market", p.Markets)
	return q
}

// LeaderboardParams filters GET /v1/leaderboard requests.
type LeaderboardParams struct {
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
	// Category filters by leaderboard category.
	Category string
	// TimePeriod filters by time range.
	TimePeriod string
	// OrderBy sets the sort field.
	OrderBy string
}

func (p LeaderboardParams) values() url.Values {
	q := url.Values{}
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	setString(q, "category", p.Category)
	setString(q, "timePeriod", p.TimePeriod)
	setString(q, "orderBy", p.OrderBy)
	return q
}

// BuilderLeaderboardParams filters GET /v1/builders/leaderboard requests.
type BuilderLeaderboardParams struct {
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
}

func (p BuilderLeaderboardParams) values() url.Values {
	q := url.Values{}
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	return q
}

// BuilderVolumeParams filters GET /v1/builders/volume requests.
type BuilderVolumeParams struct {
	// Builder is the builder code.
	Builder string
	// Start is the start date.
	Start string
	// End is the end date.
	End string
}

func (p BuilderVolumeParams) values() url.Values {
	q := url.Values{}
	setString(q, "builder", p.Builder)
	setString(q, "start", p.Start)
	setString(q, "end", p.End)
	return q
}
