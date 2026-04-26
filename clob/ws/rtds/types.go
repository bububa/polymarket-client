// Package rtds implements the Polymarket real-time data stream client.
package rtds

import (
	"encoding/json"
	"fmt"
	"time"
)

// Action is an RTDS subscription action.
type Action string

const (
	// ActionSubscribe subscribes to one or more topics.
	ActionSubscribe Action = "subscribe"
	// ActionUnsubscribe unsubscribes from one or more topics.
	ActionUnsubscribe Action = "unsubscribe"
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

// Credentials is the CLOB API credential payload used by authenticated RTDS topics.
type Credentials struct {
	// Key is the CLOB API key.
	Key string `json:"key"`
	// Secret is the CLOB API secret.
	Secret string `json:"secret"`
	// Passphrase is the CLOB API passphrase.
	Passphrase string `json:"passphrase"`
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
	// CLOBAuth authenticates protected topics such as comments.
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
	if s.Topic == "crypto_prices_chainlink" && s.Filters != nil {
		data, err := json.Marshal(s.Filters)
		if err != nil {
			return nil, err
		}
		aux.Filters = string(data)
	}
	return json.Marshal(aux)
}

// Message is the top-level message received from RTDS.
type Message struct {
	// Topic is the source RTDS topic.
	Topic string `json:"topic"`
	// Type is the topic-specific event type.
	Type string `json:"type"`
	// Timestamp is the server event timestamp.
	Timestamp int64 `json:"timestamp"`
	// Payload is the raw topic-specific payload.
	Payload json.RawMessage `json:"payload"`
}

// AsCryptoPrice decodes Payload as a CryptoPrice.
func (m *Message) AsCryptoPrice(out *CryptoPrice) error {
	if m.Topic != "crypto_prices" {
		return fmt.Errorf("polymarket: RTDS topic is %q, not crypto_prices", m.Topic)
	}
	return json.Unmarshal(m.Payload, out)
}

// AsChainlinkPrice decodes Payload as a ChainlinkPrice.
func (m *Message) AsChainlinkPrice(out *ChainlinkPrice) error {
	if m.Topic != "crypto_prices_chainlink" {
		return fmt.Errorf("polymarket: RTDS topic is %q, not crypto_prices_chainlink", m.Topic)
	}
	return json.Unmarshal(m.Payload, out)
}

// AsComment decodes Payload as a Comment.
func (m *Message) AsComment(out *Comment) error {
	if m.Topic != "comments" {
		return fmt.Errorf("polymarket: RTDS topic is %q, not comments", m.Topic)
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
	Value string `json:"value"`
}

// ChainlinkPrice is a Chainlink price feed update.
type ChainlinkPrice struct {
	// Symbol is the feed symbol.
	Symbol string `json:"symbol"`
	// Timestamp is the source timestamp.
	Timestamp int64 `json:"timestamp"`
	// Value is the price value.
	Value string `json:"value"`
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
