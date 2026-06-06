package gamma

import (
	"encoding/json"
	"net/url"

	pmtypes "github.com/bububa/polymarket-client/shared"
)

// Market describes a Gamma market.
type Market struct {
	ID                      pmtypes.Int          `json:"id"`
	Question                string               `json:"question"`
	ConditionID             string               `json:"conditionId"`
	Slug                    string               `json:"slug"`
	Description             string               `json:"description"`
	ResolutionSource        string               `json:"resolutionSource"`
	EndDate                 pmtypes.Time         `json:"endDate"`
	StartDate               pmtypes.Time         `json:"startDate"`
	Image                   string               `json:"image"`
	Icon                    string               `json:"icon"`
	Active                  bool                 `json:"active"`
	Closed                  bool                 `json:"closed"`
	Archived                bool                 `json:"archived"`
	Resolved                bool                 `json:"resolved"`
	New                     bool                 `json:"new"`
	Featured                bool                 `json:"featured"`
	Restricted              bool                 `json:"restricted"`
	Liquidity               pmtypes.Float64      `json:"liquidity"`
	Volume                  pmtypes.Float64      `json:"volume"`
	OpenInterest            pmtypes.Float64      `json:"openInterest"`
	VolumeNum               pmtypes.Float64      `json:"volumeNum"`
	LiquidityNum            pmtypes.Float64      `json:"liquidityNum"`
	EndDateISO              pmtypes.Time         `json:"endDateIso"`
	StartDateISO            pmtypes.Time         `json:"startDateIso"`
	CreatedAt               pmtypes.Time         `json:"createdAt"`
	UpdatedAt               pmtypes.Time         `json:"updatedAt"`
	CreationDate            pmtypes.Time         `json:"creationDate"`
	PublishedAt             pmtypes.Time         `json:"published_at"`
	CreatedBy               pmtypes.Int          `json:"createdBy"`
	UpdatedBy               pmtypes.Int          `json:"updatedBy"`
	Ready                   bool                 `json:"ready"`
	Funded                  bool                 `json:"funded"`
	AcceptingOrders         bool                 `json:"acceptingOrders"`
	AcceptingOrderTimestamp pmtypes.Time         `json:"acceptingOrderTimestamp"`
	EnableOrderBook         bool                 `json:"enableOrderBook"`
	MinimumOrderSize        pmtypes.Float64      `json:"minimumOrderSize"`
	MinimumTickSize         pmtypes.Float64      `json:"minimumTickSize"`
	QuestionID              pmtypes.String       `json:"questionID"`
	FPmm                    string               `json:"fpmm"`
	MakerBaseFee            pmtypes.Float64      `json:"makerBaseFee"`
	TakerBaseFee            pmtypes.Float64      `json:"takerBaseFee"`
	NotificationsEnabled    bool                 `json:"notificationsEnabled"`
	NegRisk                 bool                 `json:"negRisk"`
	NegRiskMarketID         pmtypes.String       `json:"negRiskMarketID"`
	NegRiskRequestID        pmtypes.String       `json:"negRiskRequestID"`
	Competitive             pmtypes.Float64      `json:"competitive"`
	RewardsMinSize          pmtypes.Float64      `json:"rewardsMinSize"`
	RewardsMaxSpread        pmtypes.Float64      `json:"rewardsMaxSpread"`
	Spread                  pmtypes.Float64      `json:"spread"`
	LastTradePrice          pmtypes.Float64      `json:"lastTradePrice"`
	BestBid                 pmtypes.Float64      `json:"bestBid"`
	BestAsk                 pmtypes.Float64      `json:"bestAsk"`
	OneDayPriceChange       pmtypes.Float64      `json:"oneDayPriceChange"`
	OneHourPriceChange      pmtypes.Float64      `json:"oneHourPriceChange"`
	OneWeekPriceChange      pmtypes.Float64      `json:"oneWeekPriceChange"`
	OneMonthPriceChange     pmtypes.Float64      `json:"oneMonthPriceChange"`
	OneYearPriceChange      pmtypes.Float64      `json:"oneYearPriceChange"`
	Outcomes                pmtypes.StringSlice  `json:"outcomes"`
	OutcomePrices           pmtypes.Float64Slice `json:"outcomePrices"`
	ClobTokenIDs            pmtypes.StringSlice  `json:"clobTokenIds"`
	Tags                    []Tag                `json:"tags"`
	Events                  []Event              `json:"events"`
	Rewards                 []Reward             `json:"rewards"`
	Category                string               `json:"category"`
	Subcategory             string               `json:"subcategory"`
	SortBy                  string               `json:"sortBy"`
	IsTemplate              bool                 `json:"isTemplate"`
	TemplateVariables       string               `json:"templateVariables"`
	MarketType              string               `json:"marketType"`
	GroupItemTitle          string               `json:"groupItemTitle"`
	GroupItemThreshold      pmtypes.String       `json:"groupItemThreshold"`
	OrderPriceMinTickSize   pmtypes.Float64      `json:"orderPriceMinTickSize"`
	OrderMinSize            pmtypes.Float64      `json:"orderMinSize"`
	Volume24hr              pmtypes.Float64      `json:"volume24hr"`
	Volume1wk               pmtypes.Float64      `json:"volume1wk"`
	Volume1mo               pmtypes.Float64      `json:"volume1mo"`
	Volume1yr               pmtypes.Float64      `json:"volume1yr"`
	LiquidityClob           pmtypes.Float64      `json:"liquidityClob"`
	VolumeClob              pmtypes.Float64      `json:"volumeClob"`
	Raw                     json.RawMessage      `json:"-"`
}

// Event describes a Gamma event.
type Event struct {
	ID                pmtypes.Int     `json:"id"`
	Ticker            string          `json:"ticker"`
	Slug              string          `json:"slug"`
	Title             string          `json:"title"`
	Subtitle          string          `json:"subtitle"`
	Description       string          `json:"description"`
	ResolutionSource  string          `json:"resolutionSource"`
	StartDate         pmtypes.Time    `json:"startDate"`
	CreationDate      pmtypes.Time    `json:"creationDate"`
	EndDate           pmtypes.Time    `json:"endDate"`
	Image             string          `json:"image"`
	Icon              string          `json:"icon"`
	Active            bool            `json:"active"`
	Closed            bool            `json:"closed"`
	Archived          bool            `json:"archived"`
	New               bool            `json:"new"`
	Featured          bool            `json:"featured"`
	Restricted        bool            `json:"restricted"`
	Liquidity         pmtypes.Float64 `json:"liquidity"`
	Volume            pmtypes.Float64 `json:"volume"`
	OpenInterest      pmtypes.Float64 `json:"openInterest"`
	SortBy            string          `json:"sortBy"`
	Category          string          `json:"category"`
	Subcategory       string          `json:"subcategory"`
	IsTemplate        bool            `json:"isTemplate"`
	TemplateVariables string          `json:"templateVariables"`
	PublishedAt       pmtypes.Time    `json:"published_at"`
	CreatedAt         pmtypes.Time    `json:"createdAt"`
	UpdatedAt         pmtypes.Time    `json:"updatedAt"`
	CreatedBy         pmtypes.Int     `json:"createdBy"`
	UpdatedBy         pmtypes.Int     `json:"updatedBy"`
	Competitive       pmtypes.Float64 `json:"competitive"`
	Volume24hr        pmtypes.Float64 `json:"volume24hr"`
	Volume1wk         pmtypes.Float64 `json:"volume1wk"`
	Volume1mo         pmtypes.Float64 `json:"volume1mo"`
	Volume1yr         pmtypes.Float64 `json:"volume1yr"`
	LiquidityClob     pmtypes.Float64 `json:"liquidityClob"`
	NegRisk           bool            `json:"negRisk"`
	CommentCount      pmtypes.Int     `json:"commentCount"`
	Markets           []Market        `json:"markets"`
	Tags              []Tag           `json:"tags"`
	Series            []Series        `json:"series"`
	Raw               json.RawMessage `json:"-"`
}

// Series describes a Gamma series.
type Series struct {
	ID          pmtypes.Int     `json:"id"`
	Ticker      string          `json:"ticker"`
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Subtitle    string          `json:"subtitle"`
	Description string          `json:"description"`
	Image       string          `json:"image"`
	Icon        string          `json:"icon"`
	Active      bool            `json:"active"`
	Closed      bool            `json:"closed"`
	Archived    bool            `json:"archived"`
	Volume      pmtypes.Float64 `json:"volume"`
	Liquidity   pmtypes.Float64 `json:"liquidity"`
	StartDate   pmtypes.Time    `json:"startDate"`
	EndDate     pmtypes.Time    `json:"endDate"`
	CreatedAt   pmtypes.Time    `json:"createdAt"`
	UpdatedAt   pmtypes.Time    `json:"updatedAt"`
	Events      []Event         `json:"events"`
	Tags        []Tag           `json:"tags"`
	Raw         json.RawMessage `json:"-"`
}

// Tag describes a Gamma tag.
type Tag struct {
	ID          pmtypes.Int     `json:"id"`
	Label       string          `json:"label"`
	Slug        string          `json:"slug"`
	ForceShow   bool            `json:"forceShow"`
	PublishedAt pmtypes.Time    `json:"publishedAt"`
	CreatedBy   pmtypes.Int     `json:"createdBy"`
	UpdatedBy   pmtypes.Int     `json:"updatedBy"`
	CreatedAt   pmtypes.Time    `json:"createdAt"`
	UpdatedAt   pmtypes.Time    `json:"updatedAt"`
	ForceHide   bool            `json:"forceHide"`
	IsCarousel  bool            `json:"isCarousel"`
	Raw         json.RawMessage `json:"-"`
}

// TagRelationship describes a ranked relationship between two tags.
type TagRelationship struct {
	ID           pmtypes.Int `json:"id"`
	TagID        pmtypes.Int `json:"tagID"`
	RelatedTagID pmtypes.Int `json:"relatedTagID"`
	Rank         pmtypes.Int `json:"rank"`
}

// Reward describes a Gamma market reward configuration.
type Reward struct {
	ID           pmtypes.Int     `json:"id"`
	AssetAddress string          `json:"assetAddress"`
	StartDate    pmtypes.Time    `json:"startDate"`
	EndDate      pmtypes.Time    `json:"endDate"`
	RatePerDay   pmtypes.Float64 `json:"ratePerDay"`
	TotalRewards pmtypes.Float64 `json:"totalRewards"`
	Raw          json.RawMessage `json:"-"`
}

// SearchResults contains public search matches.
type SearchResults struct {
	Markets []Market `json:"markets"`
	Events  []Event  `json:"events"`
	Series  []Series `json:"series"`
	Tags    []Tag    `json:"tags"`
}

// SportsMetadata describes a sport.
type SportsMetadata struct {
	ID        pmtypes.Int  `json:"id"`
	Name      string       `json:"name"`
	Slug      string       `json:"slug"`
	Label     string       `json:"label"`
	TagID     pmtypes.Int  `json:"tagId"`
	CreatedAt pmtypes.Time `json:"createdAt"`
	Image     string       `json:"image"`
}

// Team describes a sports team.
type Team struct {
	ID           pmtypes.Int  `json:"id"`
	Name         string       `json:"name"`
	Sport        string       `json:"sport"`
	Logo         string       `json:"logo"`
	Abbreviation string       `json:"abbreviation"`
	Image        string       `json:"image"`
	CreatedAt    pmtypes.Time `json:"createdAt"`
	UpdatedAt    pmtypes.Time `json:"updatedAt"`
}

// Comment describes a Gamma comment.
type Comment struct {
	ID              pmtypes.Int     `json:"id"`
	Body            string          `json:"body"`
	User            string          `json:"user"`
	Address         string          `json:"address"`
	CreatedAt       pmtypes.Time    `json:"createdAt"`
	UpdatedAt       pmtypes.Time    `json:"updatedAt"`
	Deleted         bool            `json:"deleted"`
	ParentID        pmtypes.Int     `json:"parentId"`
	RootID          pmtypes.Int     `json:"rootId"`
	Depth           pmtypes.Int     `json:"depth"`
	Market          string          `json:"market"`
	EventID         pmtypes.Int     `json:"eventId"`
	ReportCount     pmtypes.Int     `json:"reportCount"`
	ReactionCount   pmtypes.Int     `json:"reactionCount"`
	ProfileImage    string          `json:"profileImage"`
	ProfileImageURL string          `json:"profileImageUrl"`
	Raw             json.RawMessage `json:"-"`
}

// PublicProfile describes a public user profile.
type PublicProfile struct {
	Address               string          `json:"address"`
	Name                  string          `json:"name"`
	Username              string          `json:"username"`
	Pseudonym             string          `json:"pseudonym"`
	Bio                   string          `json:"bio"`
	ProfileImage          string          `json:"profileImage"`
	ProfileImageOptimized string          `json:"profileImageOptimized"`
	DisplayUsernamePublic bool            `json:"displayUsernamePublic"`
	ProxyWallet           string          `json:"proxyWallet"`
	Wallet                string          `json:"wallet"`
	CreatedAt             pmtypes.Time    `json:"createdAt"`
	UpdatedAt             pmtypes.Time    `json:"updatedAt"`
	VolumeTraded          pmtypes.Float64 `json:"volumeTraded"`
	ProfitLoss            pmtypes.Float64 `json:"profitLoss"`
	PositionsValue        pmtypes.Float64 `json:"positionsValue"`
	Verified              bool            `json:"verified"`
	VerifiedBadge         bool            `json:"verifiedBadge"`
	XUsername             string          `json:"xUsername"`
	Raw                   json.RawMessage `json:"-"`
}

func (m *Market) UnmarshalJSON(data []byte) error {
	type alias Market
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*m = Market(out)
	m.Raw = append(m.Raw[:0], data...)
	return nil
}

func (e *Event) UnmarshalJSON(data []byte) error {
	type alias Event
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*e = Event(out)
	e.Raw = append(e.Raw[:0], data...)
	return nil
}

func (s *Series) UnmarshalJSON(data []byte) error {
	type alias Series
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*s = Series(out)
	s.Raw = append(s.Raw[:0], data...)
	return nil
}

func (t *Tag) UnmarshalJSON(data []byte) error {
	type alias Tag
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*t = Tag(out)
	t.Raw = append(t.Raw[:0], data...)
	return nil
}

func (c *Comment) UnmarshalJSON(data []byte) error {
	type alias Comment
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*c = Comment(out)
	c.Raw = append(c.Raw[:0], data...)
	return nil
}

func (p *PublicProfile) UnmarshalJSON(data []byte) error {
	type alias PublicProfile
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*p = PublicProfile(out)
	p.Raw = append(p.Raw[:0], data...)
	return nil
}

// MarketFilterParams filters GET /markets requests.
type MarketFilterParams struct {
	// Active filters by active status.
	Active *bool
	// Closed filters by closed status.
	Closed *bool
	// Archived filters by archived status.
	Archived *bool
	// Resolved filters by resolution status.
	Resolved *bool
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
	// Order sets the sort field.
	Order string
	// Ascending sets the sort direction.
	Ascending *bool
	// TagID filters by tag ID.
	TagID int
	// EventID filters by event ID.
	EventID int
	// Slug filters by market slug.
	Slug string
	// NegativeRisk filters by neg-risk status.
	NegativeRisk *bool
	// AcceptingOrders filters by order book status.
	AcceptingOrders *bool
	// ClobTokenIDs filters by conditional token IDs.
	ClobTokenIDs []string
	// ConditionIDs filters by condition IDs.
	ConditionIDs []string
	// MarketMakerAddress filters by FPMM contract address.
	MarketMakerAddress []string
}

func (p MarketFilterParams) appendQuery(q url.Values) {
	setBool(q, "active", p.Active)
	setBool(q, "closed", p.Closed)
	setBool(q, "archived", p.Archived)
	setBool(q, "resolved", p.Resolved)
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	setString(q, "order", p.Order)
	setBool(q, "ascending", p.Ascending)
	setInt(q, "tag_id", p.TagID)
	setInt(q, "event_id", p.EventID)
	setString(q, "slug", p.Slug)
	setBool(q, "negative_risk", p.NegativeRisk)
	setBool(q, "accepting_orders", p.AcceptingOrders)
	for _, id := range p.ClobTokenIDs {
		q.Add("clob_token_ids", id)
	}
	for _, id := range p.ConditionIDs {
		q.Add("condition_ids", id)
	}
	for _, addr := range p.MarketMakerAddress {
		q.Add("market_maker_address", addr)
	}
}

// EventFilterParams is an alias for MarketFilterParams.
type EventFilterParams = MarketFilterParams

// MarketKeysetPage contains a cursor-paginated market page.
type MarketKeysetPage struct {
	Markets    []Market `json:"markets"`
	NextCursor string   `json:"next_cursor"`
}

// MarketKeysetParams filters GET /markets/keyset requests.
type MarketKeysetParams struct {
	// Limit sets the maximum number of results, up to 100.
	Limit int
	// Order is a comma-separated list of JSON field names to order by.
	Order string
	// Ascending sets the sort direction when Order is set.
	Ascending *bool
	// AfterCursor is the cursor returned as next_cursor by the previous page.
	AfterCursor string
	// IDs filters by market IDs.
	IDs []int
	// Slugs filters by market slugs.
	Slugs []string
	// Closed filters by closed status.
	Closed *bool
	// Decimalized filters by decimalized status.
	Decimalized *bool
	// ClobTokenIDs filters by conditional token IDs.
	ClobTokenIDs []string
	// ConditionIDs filters by condition IDs.
	ConditionIDs []string
	// QuestionIDs filters by question IDs.
	QuestionIDs []string
	// MarketMakerAddress filters by FPMM contract address.
	MarketMakerAddress []string
	// LiquidityNumMin filters by minimum numeric liquidity.
	LiquidityNumMin *float64
	// LiquidityNumMax filters by maximum numeric liquidity.
	LiquidityNumMax *float64
	// VolumeNumMin filters by minimum numeric volume.
	VolumeNumMin *float64
	// VolumeNumMax filters by maximum numeric volume.
	VolumeNumMax *float64
	// StartDateMin filters by minimum start date.
	StartDateMin string
	// StartDateMax filters by maximum start date.
	StartDateMax string
	// EndDateMin filters by minimum end date.
	EndDateMin string
	// EndDateMax filters by maximum end date.
	EndDateMax string
	// TagIDs filters by tag IDs.
	TagIDs []int
	// RelatedTags includes markets related to the supplied tags.
	RelatedTags *bool
	// TagMatch controls tag matching semantics.
	TagMatch string
	// CYOM filters by create-your-own-market status.
	CYOM *bool
	// RFQEnabled filters by RFQ status.
	RFQEnabled *bool
	// UMAResolutionStatus filters by UMA resolution status.
	UMAResolutionStatus string
	// GameID filters by game ID.
	GameID string
	// SportsMarketTypes filters by sports market type identifiers.
	SportsMarketTypes []string
	// IncludeTag includes Tags relation on each market.
	IncludeTag *bool
	// Locale sets the localized response language.
	Locale string
}

func (p MarketKeysetParams) appendQuery(q url.Values) {
	setInt(q, "limit", p.Limit)
	setString(q, "order", p.Order)
	setBool(q, "ascending", p.Ascending)
	setString(q, "after_cursor", p.AfterCursor)
	setIntSlice(q, "id", p.IDs)
	setStringSlice(q, "slug", p.Slugs)
	setBool(q, "closed", p.Closed)
	setBool(q, "decimalized", p.Decimalized)
	setStringSlice(q, "clob_token_ids", p.ClobTokenIDs)
	setStringSlice(q, "condition_ids", p.ConditionIDs)
	setStringSlice(q, "question_ids", p.QuestionIDs)
	setStringSlice(q, "market_maker_address", p.MarketMakerAddress)
	setFloat(q, "liquidity_num_min", p.LiquidityNumMin)
	setFloat(q, "liquidity_num_max", p.LiquidityNumMax)
	setFloat(q, "volume_num_min", p.VolumeNumMin)
	setFloat(q, "volume_num_max", p.VolumeNumMax)
	setString(q, "start_date_min", p.StartDateMin)
	setString(q, "start_date_max", p.StartDateMax)
	setString(q, "end_date_min", p.EndDateMin)
	setString(q, "end_date_max", p.EndDateMax)
	setIntSlice(q, "tag_id", p.TagIDs)
	setBool(q, "related_tags", p.RelatedTags)
	setString(q, "tag_match", p.TagMatch)
	setBool(q, "cyom", p.CYOM)
	setBool(q, "rfq_enabled", p.RFQEnabled)
	setString(q, "uma_resolution_status", p.UMAResolutionStatus)
	setString(q, "game_id", p.GameID)
	setStringSlice(q, "sports_market_types", p.SportsMarketTypes)
	setBool(q, "include_tag", p.IncludeTag)
	setString(q, "locale", p.Locale)
}

// EventKeysetPage contains a cursor-paginated event page.
type EventKeysetPage struct {
	Events     []Event `json:"events"`
	NextCursor string  `json:"next_cursor"`
}

// EventKeysetParams filters GET /events/keyset requests.
type EventKeysetParams struct {
	// Limit sets the maximum number of results, up to 500.
	Limit int
	// Order is a comma-separated list of JSON field names to order by.
	Order string
	// Ascending sets the sort direction when Order is set.
	Ascending *bool
	// AfterCursor is the cursor returned as next_cursor by the previous page.
	AfterCursor string
	// IDs filters by event IDs.
	IDs []int
	// Slugs filters by event slugs.
	Slugs []string
	// Closed filters by closed status.
	Closed *bool
	// Live filters by live status.
	Live *bool
	// Featured filters by featured status.
	Featured *bool
	// CYOM filters by create-your-own-market status.
	CYOM *bool
	// TitleSearch filters by title search text.
	TitleSearch string
	// LiquidityMin filters by minimum liquidity.
	LiquidityMin *float64
	// LiquidityMax filters by maximum liquidity.
	LiquidityMax *float64
	// VolumeMin filters by minimum volume.
	VolumeMin *float64
	// VolumeMax filters by maximum volume.
	VolumeMax *float64
	// StartDateMin filters by minimum start date.
	StartDateMin string
	// StartDateMax filters by maximum start date.
	StartDateMax string
	// EndDateMin filters by minimum end date.
	EndDateMin string
	// EndDateMax filters by maximum end date.
	EndDateMax string
	// StartTimeMin filters by minimum start time.
	StartTimeMin string
	// StartTimeMax filters by maximum start time.
	StartTimeMax string
	// TagIDs filters by tag IDs.
	TagIDs []int
	// TagSlug filters by tag slug.
	TagSlug string
	// ExcludeTagIDs excludes matching tag IDs.
	ExcludeTagIDs []int
	// RelatedTags includes events related to the supplied tags.
	RelatedTags *bool
	// TagMatch controls tag matching semantics.
	TagMatch string
	// SeriesIDs filters by series IDs.
	SeriesIDs []int
	// GameIDs filters by game IDs.
	GameIDs []int
	// EventDate filters by event date.
	EventDate string
	// EventWeek filters by event week.
	EventWeek int
	// FeaturedOrder filters by featured-order status.
	FeaturedOrder *bool
	// Recurrence filters by recurrence.
	Recurrence string
	// CreatedBy filters by creator IDs.
	CreatedBy []string
	// ParentEventID filters by parent event ID.
	ParentEventID int
	// IncludeChildren includes child events.
	IncludeChildren *bool
	// PartnerSlug attaches external partners to matching events.
	PartnerSlug string
	// IncludeChat includes Chats and Series.Chats relations.
	IncludeChat *bool
	// IncludeTemplate includes Templates relation.
	IncludeTemplate *bool
	// IncludeBestLines includes BestLines relation.
	IncludeBestLines *bool
	// Locale sets the localized response language.
	Locale string
}

func (p EventKeysetParams) appendQuery(q url.Values) {
	setInt(q, "limit", p.Limit)
	setString(q, "order", p.Order)
	setBool(q, "ascending", p.Ascending)
	setString(q, "after_cursor", p.AfterCursor)
	setIntSlice(q, "id", p.IDs)
	setStringSlice(q, "slug", p.Slugs)
	setBool(q, "closed", p.Closed)
	setBool(q, "live", p.Live)
	setBool(q, "featured", p.Featured)
	setBool(q, "cyom", p.CYOM)
	setString(q, "title_search", p.TitleSearch)
	setFloat(q, "liquidity_min", p.LiquidityMin)
	setFloat(q, "liquidity_max", p.LiquidityMax)
	setFloat(q, "volume_min", p.VolumeMin)
	setFloat(q, "volume_max", p.VolumeMax)
	setString(q, "start_date_min", p.StartDateMin)
	setString(q, "start_date_max", p.StartDateMax)
	setString(q, "end_date_min", p.EndDateMin)
	setString(q, "end_date_max", p.EndDateMax)
	setString(q, "start_time_min", p.StartTimeMin)
	setString(q, "start_time_max", p.StartTimeMax)
	setIntSlice(q, "tag_id", p.TagIDs)
	setString(q, "tag_slug", p.TagSlug)
	setIntSlice(q, "exclude_tag_id", p.ExcludeTagIDs)
	setBool(q, "related_tags", p.RelatedTags)
	setString(q, "tag_match", p.TagMatch)
	setIntSlice(q, "series_id", p.SeriesIDs)
	setIntSlice(q, "game_id", p.GameIDs)
	setString(q, "event_date", p.EventDate)
	setInt(q, "event_week", p.EventWeek)
	setBool(q, "featured_order", p.FeaturedOrder)
	setString(q, "recurrence", p.Recurrence)
	setStringSlice(q, "created_by", p.CreatedBy)
	setInt(q, "parent_event_id", p.ParentEventID)
	setBool(q, "include_children", p.IncludeChildren)
	setString(q, "partner_slug", p.PartnerSlug)
	setBool(q, "include_chat", p.IncludeChat)
	setBool(q, "include_template", p.IncludeTemplate)
	setBool(q, "include_best_lines", p.IncludeBestLines)
	setString(q, "locale", p.Locale)
}

// SeriesFilterParams filters GET /series requests.
type SeriesFilterParams struct {
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
	// Active filters by active status.
	Active *bool
	// Closed filters by closed status.
	Closed *bool
	// Archived filters by archived status.
	Archived *bool
	// Slug filters by series slug.
	Slug string
	// TagID filters by tag ID.
	TagID int
	// Ascending sets the sort direction.
	Ascending *bool
	// Order sets the sort field.
	Order string
}

func (p SeriesFilterParams) appendQuery(q url.Values) {
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	setBool(q, "active", p.Active)
	setBool(q, "closed", p.Closed)
	setBool(q, "archived", p.Archived)
	setString(q, "slug", p.Slug)
	setInt(q, "tag_id", p.TagID)
	setBool(q, "ascending", p.Ascending)
	setString(q, "order", p.Order)
}

// RelatedTagParams filters GET /tags/:id/related-tags requests.
type RelatedTagParams struct {
	// OmitEmpty removes tags with no related items.
	OmitEmpty *bool
	// Status filters by tag status.
	Status string
}

func (p RelatedTagParams) appendQuery(q url.Values) {
	setBool(q, "omit_empty", p.OmitEmpty)
	setString(q, "status", p.Status)
}

// SportsMarketTypesResponse contains valid sports market types.
type SportsMarketTypesResponse struct {
	// MarketTypes lists the valid market type identifiers.
	MarketTypes []string `json:"marketTypes"`
}

// CommentFilterParams filters GET /comments requests.
type CommentFilterParams struct {
	// Limit sets the maximum results.
	Limit int
	// Offset sets the start index.
	Offset int
	// Market filters by condition ID.
	Market string
	// EventID filters by event ID.
	EventID int
	// User filters by user wallet address.
	User string
}

func (p CommentFilterParams) appendQuery(q url.Values) {
	setInt(q, "limit", p.Limit)
	setInt(q, "offset", p.Offset)
	setString(q, "market", p.Market)
	setInt(q, "event_id", p.EventID)
	setString(q, "user", p.User)
}
