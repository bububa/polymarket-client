package data

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"github.com/bububa/polymarket-client/internal/polyhttp"
	pmtypes "github.com/bububa/polymarket-client/shared"
)

type ComboPositionStatus string

const (
	ComboPositionOpen         ComboPositionStatus = "OPEN"
	ComboPositionPartial      ComboPositionStatus = "PARTIAL"
	ComboPositionResolvedWin  ComboPositionStatus = "RESOLVED_WIN"
	ComboPositionResolvedLoss ComboPositionStatus = "RESOLVED_LOSS"
)

type ComboPositionSort string

const (
	ComboPositionSortCurrentValueDesc ComboPositionSort = "current_value_desc"
	ComboPositionSortFirstEntryDesc   ComboPositionSort = "first_entry_desc"
	ComboPositionSortEntryCostDesc    ComboPositionSort = "entry_cost_desc"
	ComboPositionSortResolvedAtDesc   ComboPositionSort = "resolved_at_desc"
	ComboPositionSortUpdatedAsc       ComboPositionSort = "updated_asc"
)

type ComboPositionParams struct {
	User         string
	Status       ComboPositionStatus
	Sort         ComboPositionSort
	MarketIDs    []string
	Limit        int
	Offset       int
	Cursor       string
	UpdatedAfter int64
}

func (p ComboPositionParams) values() url.Values {
	q := url.Values{}
	setString(q, "user", p.User)
	setString(q, "status", string(p.Status))
	setString(q, "sort", string(p.Sort))
	setCommaList(q, "market_id", p.MarketIDs)
	setInt(q, "limit", p.Limit)
	if p.Offset > 0 {
		q.Set("offset", strconv.Itoa(p.Offset))
	}
	setString(q, "cursor", p.Cursor)
	setInt64(q, "updatedAfter", p.UpdatedAfter)
	return q
}

type ComboActivityParams struct {
	User      string
	MarketIDs []string
	Limit     int
	Offset    int
	Cursor    string
}

func (p ComboActivityParams) values() url.Values {
	q := url.Values{}
	setString(q, "user", p.User)
	setCommaList(q, "market_id", p.MarketIDs)
	setInt(q, "limit", p.Limit)
	if p.Offset > 0 {
		q.Set("offset", strconv.Itoa(p.Offset))
	}
	setString(q, "cursor", p.Cursor)
	return q
}

type ComboPagination struct {
	Limit      pmtypes.Int    `json:"limit"`
	Offset     pmtypes.Int    `json:"offset"`
	HasMore    bool           `json:"has_more"`
	NextCursor pmtypes.String `json:"next_cursor"`
}

type ComboPositionPage struct {
	Combos     []ComboPosition `json:"combos"`
	Pagination ComboPagination `json:"pagination"`
}

type ComboPosition struct {
	ConditionID       string              `json:"combo_condition_id"`
	PositionID        pmtypes.String      `json:"combo_position_id"`
	Outcome           string              `json:"side"`
	ModuleID          pmtypes.String      `json:"module_id"`
	Wallet            string              `json:"user_address"`
	Shares            pmtypes.String      `json:"shares_balance"`
	EntryAveragePrice pmtypes.String      `json:"entry_avg_price_usdc"`
	EntryCost         pmtypes.String      `json:"entry_cost_usdc"`
	RealizedPayout    pmtypes.String      `json:"realized_payout_usdc"`
	TotalCost         pmtypes.String      `json:"total_cost_usdc"`
	Status            ComboPositionStatus `json:"status"`
	Redeemable        bool                `json:"redeemable"`
	FirstEntryAt      pmtypes.Time        `json:"first_entry_at"`
	ResolvedAt        pmtypes.Time        `json:"resolved_at"`
	UpdatedAt         pmtypes.Time        `json:"updated_at"`
	LegsTotal         pmtypes.Int         `json:"legs_total"`
	LegsResolved      pmtypes.Int         `json:"legs_resolved"`
	LegsPending       pmtypes.Int         `json:"legs_pending"`
	Legs              []ComboPositionLeg  `json:"legs"`
}

type ComboPositionLeg struct {
	Index        pmtypes.Int          `json:"leg_index"`
	PositionID   pmtypes.String       `json:"leg_position_id"`
	ConditionID  string               `json:"leg_condition_id"`
	OutcomeIndex pmtypes.Int          `json:"leg_outcome_index"`
	OutcomeLabel string               `json:"leg_outcome_label"`
	Status       ComboPositionStatus  `json:"leg_status"`
	ResolvedAt   pmtypes.Time         `json:"leg_resolved_at"`
	CurrentPrice pmtypes.String       `json:"leg_current_price"`
	Market       *ComboPositionMarket `json:"market"`
}

type ComboPositionMarket struct {
	MarketID    pmtypes.String            `json:"market_id"`
	Slug        string                    `json:"slug"`
	Title       string                    `json:"title"`
	Outcome     string                    `json:"outcome"`
	ImageURL    string                    `json:"image_url"`
	IconURL     string                    `json:"icon_url"`
	Category    string                    `json:"category"`
	Subcategory string                    `json:"subcategory"`
	Tags        []string                  `json:"tags"`
	EndDate     pmtypes.Time              `json:"end_date"`
	Event       *ComboPositionMarketEvent `json:"event"`
}

type ComboPositionMarketEvent struct {
	EventID    pmtypes.String `json:"event_id"`
	EventSlug  string         `json:"event_slug"`
	EventTitle string         `json:"event_title"`
	EventImage string         `json:"event_image"`
}

type ComboActivityType string

const (
	ComboActivitySplit    ComboActivityType = "SPLIT"
	ComboActivityMerge    ComboActivityType = "MERGE"
	ComboActivityConvert  ComboActivityType = "CONVERT"
	ComboActivityCompress ComboActivityType = "COMPRESS"
	ComboActivityWrap     ComboActivityType = "WRAP"
	ComboActivityUnwrap   ComboActivityType = "UNWRAP"
	ComboActivityRedeem   ComboActivityType = "REDEEM"
)

type ComboActivityPage struct {
	Activities []ComboActivity `json:"activity"`
	Pagination ComboPagination `json:"pagination"`
}

// UnknownComboActivity preserves a lifecycle event introduced by the server
// before this client has a typed representation for it.
type UnknownComboActivity struct {
	Type ComboActivityType
	Raw  json.RawMessage
}

type ComboActivity struct {
	ID              pmtypes.String     `json:"id"`
	Type            ComboActivityType  `json:"type"`
	Wallet          string             `json:"user_address"`
	ConditionID     string             `json:"combo_condition_id"`
	ModuleID        pmtypes.String     `json:"module_id"`
	Amount          pmtypes.String     `json:"amount_usdc"`
	Timestamp       pmtypes.Time       `json:"timestamp"`
	TransactionAt   pmtypes.Time       `json:"tx_dttm"`
	TransactionHash string             `json:"tx_hash"`
	LogIndex        pmtypes.Int        `json:"log_index"`
	BlockNumber     pmtypes.String     `json:"block_number"`
	Legs            []ComboPositionLeg `json:"legs"`
	PositionID      pmtypes.String     `json:"combo_position_id"`
	Payout          pmtypes.String     `json:"payout_usdc"`
	Raw             json.RawMessage    `json:"-"`
}

func (a *ComboActivity) UnmarshalJSON(data []byte) error {
	type alias ComboActivity
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*a = ComboActivity(out)
	a.Raw = append(a.Raw[:0], data...)
	return nil
}

func (a ComboActivity) IsKnown() bool {
	switch a.Type {
	case ComboActivitySplit, ComboActivityMerge, ComboActivityConvert, ComboActivityCompress,
		ComboActivityWrap, ComboActivityUnwrap, ComboActivityRedeem:
		return true
	default:
		return false
	}
}

func (a ComboActivity) Unknown() *UnknownComboActivity {
	if a.IsKnown() {
		return nil
	}
	return &UnknownComboActivity{Type: a.Type, Raw: append(json.RawMessage(nil), a.Raw...)}
}

func (c *Client) GetComboPositions(ctx context.Context, params ComboPositionParams) (*ComboPositionPage, error) {
	if params.User == "" {
		return nil, errors.New("data: user is required")
	}
	var out ComboPositionPage
	return &out, c.http.GetJSON(ctx, "/v1/positions/combos", params.values(), polyhttp.AuthNone, &out)
}

func (c *Client) GetComboActivity(ctx context.Context, params ComboActivityParams) (*ComboActivityPage, error) {
	if params.User == "" {
		return nil, errors.New("data: user is required")
	}
	var out ComboActivityPage
	return &out, c.http.GetJSON(ctx, "/v1/activity/combos", params.values(), polyhttp.AuthNone, &out)
}
