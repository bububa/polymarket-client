package combo

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bububa/polymarket-client/clob"
	pmtypes "github.com/bububa/polymarket-client/shared"
)

const (
	DefaultHost = "https://combos-rfq-api.polymarket.com"
	ZeroBytes32 = clob.ZeroBytes32
)

type RFQID string
type QuoteID string
type RequestorPublicID string
type ComboConditionID string

type Direction string

const (
	DirectionBuy  Direction = "BUY"
	DirectionSell Direction = "SELL"
)

type Side string

const SideYes Side = "YES"

type RequestedSizeUnit string

const (
	RequestedSizeNotional RequestedSizeUnit = "notional"
	RequestedSizeShares   RequestedSizeUnit = "shares"
)

type ConfirmationDecision string

const (
	ConfirmationConfirm ConfirmationDecision = "CONFIRM"
	ConfirmationDecline ConfirmationDecision = "DECLINE"
)

type ExecutionStatus string

const (
	ExecutionMatched   ExecutionStatus = "MATCHED"
	ExecutionMined     ExecutionStatus = "MINED"
	ExecutionConfirmed ExecutionStatus = "CONFIRMED"
	ExecutionRetrying  ExecutionStatus = "RETRYING"
	ExecutionFailed    ExecutionStatus = "FAILED"
)

func (s ExecutionStatus) Terminal() bool { return s == ExecutionConfirmed || s == ExecutionFailed }

type QuoteSource string

const (
	QuoteSourceCollateral QuoteSource = "collateral"
	QuoteSourceInventory  QuoteSource = "inventory"
)

type Identity struct {
	SignerAddress string             `json:"signer_address"`
	MakerAddress  string             `json:"maker_address"`
	SignatureType clob.SignatureType `json:"signature_type"`
}

type MarketParams struct {
	Limit   int
	Cursor  string
	Exclude []string
}

type MarketPage struct {
	Markets    []Market       `json:"markets"`
	NextCursor pmtypes.String `json:"next_cursor"`
}

type Market struct {
	ID          pmtypes.String  `json:"id"`
	ConditionID string          `json:"condition_id"`
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Yes         MarketOutcome   `json:"yes"`
	No          MarketOutcome   `json:"no"`
	Image       string          `json:"image"`
	Volume      pmtypes.Float64 `json:"volume"`
	Tags        []string        `json:"tags"`
}

type MarketOutcome struct {
	Label      string
	PositionID pmtypes.String
	Price      pmtypes.String
}

func (m *Market) UnmarshalJSON(data []byte) error {
	var wire struct {
		ID            pmtypes.String   `json:"id"`
		ConditionID   string           `json:"condition_id"`
		PositionIDs   []pmtypes.String `json:"position_ids"`
		Slug          string           `json:"slug"`
		Title         string           `json:"title"`
		Outcomes      []string         `json:"outcomes"`
		OutcomePrices []pmtypes.String `json:"outcome_prices"`
		Image         string           `json:"image"`
		Volume        pmtypes.Float64  `json:"volume"`
		Tags          []string         `json:"tags"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if len(wire.Outcomes) != 2 || len(wire.PositionIDs) != 2 || len(wire.OutcomePrices) != 2 {
		return fmt.Errorf("combo: market %s is not binary", wire.ID)
	}
	*m = Market{
		ID: wire.ID, ConditionID: wire.ConditionID, Slug: wire.Slug, Title: wire.Title,
		Yes:   MarketOutcome{Label: wire.Outcomes[0], PositionID: wire.PositionIDs[0], Price: wire.OutcomePrices[0]},
		No:    MarketOutcome{Label: wire.Outcomes[1], PositionID: wire.PositionIDs[1], Price: wire.OutcomePrices[1]},
		Image: wire.Image, Volume: wire.Volume, Tags: wire.Tags,
	}
	return nil
}

type RequestedSize struct {
	Unit    RequestedSizeUnit `json:"unit"`
	ValueE6 pmtypes.String    `json:"value_e6"`
}

type QuoteRequest struct {
	RFQID              RFQID             `json:"rfq_id"`
	RequestorPublicID  RequestorPublicID `json:"requestor_public_id"`
	LegPositionIDs     []pmtypes.String  `json:"leg_position_ids"`
	ConditionID        ComboConditionID  `json:"condition_id"`
	YesPositionID      pmtypes.String    `json:"yes_position_id"`
	NoPositionID       pmtypes.String    `json:"no_position_id"`
	Direction          Direction         `json:"direction"`
	Side               Side              `json:"side"`
	RequestedSize      RequestedSize     `json:"requested_size"`
	SubmissionDeadline pmtypes.Int64     `json:"submission_deadline"`
}

type SignedOrder struct {
	Salt          pmtypes.String     `json:"salt"`
	Maker         string             `json:"maker"`
	Signer        string             `json:"signer"`
	TokenID       pmtypes.String     `json:"tokenId"`
	MakerAmount   pmtypes.String     `json:"makerAmount"`
	TakerAmount   pmtypes.String     `json:"takerAmount"`
	Side          int                `json:"side"`
	SignatureType clob.SignatureType `json:"signatureType"`
	Timestamp     pmtypes.String     `json:"timestamp"`
	Expiration    pmtypes.String     `json:"expiration,omitempty"`
	Metadata      string             `json:"metadata,omitempty"`
	Builder       string             `json:"builder,omitempty"`
	Signature     string             `json:"signature"`
}

type SubmitQuoteRequest struct {
	QuoteID QuoteID `json:"quote_id"`
	RFQID   RFQID   `json:"rfq_id"`
	Identity
	PriceE6     pmtypes.String `json:"price_e6"`
	SizeE6      pmtypes.String `json:"size_e6"`
	SignedOrder SignedOrder    `json:"signed_order"`
}

type QuoteResponse struct {
	RFQID   RFQID   `json:"rfq_id"`
	QuoteID QuoteID `json:"quote_id"`
	Success bool    `json:"success"`
	Status  string  `json:"status"`
}

type CancelQuoteRequest struct {
	RFQID   RFQID   `json:"rfq_id"`
	QuoteID QuoteID `json:"quote_id"`
	Identity
}

type CancelQuoteResponse struct {
	RFQID   RFQID   `json:"rfq_id"`
	QuoteID QuoteID `json:"quote_id"`
	Success bool    `json:"success"`
}

type LastLookRequest struct {
	RFQID   RFQID   `json:"rfq_id"`
	QuoteID QuoteID `json:"quote_id"`
	Identity
	Decision ConfirmationDecision `json:"decision"`
}

type LastLookResponse struct {
	RFQID   RFQID   `json:"rfq_id"`
	QuoteID QuoteID `json:"quote_id"`
	Success bool    `json:"success"`
}

func NormalizeConditionID(value string) (ComboConditionID, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	if len(v) == 66 && strings.HasPrefix(v, "0x03") && (strings.HasSuffix(v, "00") || strings.HasSuffix(v, "01")) {
		v = v[:64]
	}
	if len(v) != 64 || !strings.HasPrefix(v, "0x03") {
		return "", fmt.Errorf("combo: invalid condition id %q", value)
	}
	for _, ch := range v[2:] {
		if !strings.ContainsRune("0123456789abcdef", ch) {
			return "", fmt.Errorf("combo: invalid condition id %q", value)
		}
	}
	return ComboConditionID(v), nil
}
