package ws

import (
	"encoding/json"
	"fmt"

	"github.com/bububa/polymarket-client/clob"
	"github.com/bububa/polymarket-client/combo"
	pmtypes "github.com/bububa/polymarket-client/shared"
)

type Event interface{ isEvent() }

type QuoteRequestEvent struct{ combo.QuoteRequest }

func (QuoteRequestEvent) isEvent() {}

type ConfirmationRequestEvent struct {
	RFQID          combo.RFQID            `json:"rfq_id"`
	QuoteID        combo.QuoteID          `json:"quote_id"`
	SignerAddress  string                 `json:"signer_address"`
	MakerAddress   string                 `json:"maker_address"`
	SignatureType  clob.SignatureType     `json:"signature_type"`
	LegPositionIDs []pmtypes.String       `json:"leg_position_ids"`
	ConditionID    combo.ComboConditionID `json:"condition_id"`
	YesPositionID  pmtypes.String         `json:"yes_position_id"`
	NoPositionID   pmtypes.String         `json:"no_position_id"`
	Direction      combo.Direction        `json:"direction"`
	Side           combo.Side             `json:"side"`
	FillSizeE6     pmtypes.String         `json:"fill_size_e6"`
	PriceE6        pmtypes.String         `json:"price_e6"`
	ConfirmBy      pmtypes.Int64          `json:"confirm_by"`
}

func (ConfirmationRequestEvent) isEvent() {}

type ExecutionUpdateEvent struct {
	RFQID           combo.RFQID           `json:"rfq_id"`
	Status          combo.ExecutionStatus `json:"status"`
	TransactionHash string                `json:"tx_hash"`
}

func (ExecutionUpdateEvent) isEvent() {}

type TradeEvent struct {
	RFQID          combo.RFQID             `json:"rfq_id"`
	RequesterID    combo.RequestorPublicID `json:"requester_id"`
	ConditionID    combo.ComboConditionID  `json:"condition_id"`
	LegPositionIDs []pmtypes.String        `json:"leg_position_ids"`
	Direction      combo.Direction         `json:"direction"`
	Side           combo.Side              `json:"side"`
	PriceE6        pmtypes.String          `json:"price_e6"`
	SizeE6         pmtypes.String          `json:"size_e6"`
	ExecutedAt     pmtypes.Int64           `json:"executed_at"`
}

func (TradeEvent) isEvent() {}

type UnknownEvent struct {
	Type string
	Raw  json.RawMessage
}

func (UnknownEvent) isEvent() {}

type QuoteReference struct {
	RFQID   combo.RFQID   `json:"rfq_id"`
	QuoteID combo.QuoteID `json:"quote_id"`
}
type CancelQuoteAck = QuoteReference
type ConfirmationAck struct {
	RFQID    combo.RFQID                `json:"rfq_id"`
	QuoteID  combo.QuoteID              `json:"quote_id"`
	Decision combo.ConfirmationDecision `json:"decision"`
}

type RFQErrorCode string

type RFQError struct {
	Code        RFQErrorCode  `json:"code"`
	ErrorID     string        `json:"error_id"`
	RequestType string        `json:"request_type"`
	RFQID       combo.RFQID   `json:"rfq_id"`
	QuoteID     combo.QuoteID `json:"quote_id"`
	Message     string        `json:"error"`
}

func (e *RFQError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("combo rfq: %s: %s", e.Code, e.Message)
	}
	return "combo rfq: " + e.Message
}

type QuoteOptions = combo.QuoteOptions

type authMessage struct {
	Type     string          `json:"type"`
	Auth     authCredentials `json:"auth"`
	Identity combo.Identity  `json:"identity"`
}
type authCredentials struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}
type authResponse struct {
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Address string `json:"address"`
	Role    string `json:"role"`
	Error   string `json:"error"`
}
type quoteMessage struct {
	Type        string            `json:"type"`
	RFQID       combo.RFQID       `json:"rfq_id"`
	PriceE6     pmtypes.String    `json:"price_e6"`
	SizeE6      pmtypes.String    `json:"size_e6"`
	SignedOrder combo.SignedOrder `json:"signed_order"`
}
type cancelMessage struct {
	Type          string        `json:"type"`
	RFQID         combo.RFQID   `json:"rfq_id"`
	QuoteID       combo.QuoteID `json:"quote_id"`
	SignerAddress string        `json:"signer_address"`
	MakerAddress  string        `json:"maker_address"`
}
type confirmationMessage struct {
	Type     string                     `json:"type"`
	RFQID    combo.RFQID                `json:"rfq_id"`
	QuoteID  combo.QuoteID              `json:"quote_id"`
	Decision combo.ConfirmationDecision `json:"decision"`
}
