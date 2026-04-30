package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"go.uber.org/atomic"
)

const (
	// DefaultHost is the production CLOB WebSocket host.
	DefaultHost = "wss://ws-subscriptions-clob.polymarket.com"
)

type Callback func()

// Client is a reconnecting WebSocket client for CLOB market and user streams.
type Client struct {
	host string
	url  *atomic.String

	dialOpts *websocket.DialOptions
	creds    *clob.Credentials

	conn          *atomic.Pointer[websocket.Conn]
	closed        *atomic.Bool
	connected     *atomic.Bool
	ctx           context.Context
	cancel        context.CancelFunc
	events        chan Event
	errs          chan error
	autoReconnect bool
	reconnecting  *atomic.Bool

	onConnected    Callback
	onReconnected  Callback
	onDisconnected Callback

	subsMu sync.RWMutex
	subs   []subscription
}

type subscriptionTarget uint8

const (
	subscriptionTargetMarket subscriptionTarget = iota
	subscriptionTargetUser
)

type subscription struct {
	target               subscriptionTarget
	assetIDs             []string
	markets              []string
	initialDump          bool
	customFeatureEnabled bool
}

// New creates a CLOB WebSocket client.
func New(opts ...Option) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	clt := &Client{
		host:          DefaultHost,
		ctx:           ctx,
		cancel:        cancel,
		events:        make(chan Event, 256),
		errs:          make(chan error, 16),
		autoReconnect: true,

		url:          atomic.NewString(""),
		conn:         atomic.NewPointer[websocket.Conn](nil),
		connected:    atomic.NewBool(false),
		reconnecting: atomic.NewBool(false),
		closed:       atomic.NewBool(false),
	}
	for _, opt := range opts {
		opt(clt)
	}
	return clt
}

// Host returns the configured WebSocket host.
func (c *Client) Host() string { return c.host }

// MarketURL returns the market-channel WebSocket URL.
func (c *Client) MarketURL() string { return c.host + "/ws/market" }

// UserURL returns the user-channel WebSocket URL.
func (c *Client) UserURL() string { return c.host + "/ws/user" }

// SportsURL returns the public sports-channel WebSocket URL.
func (c *Client) SportsURL() string { return c.host + "/ws" }

// ConnectMarket opens the market-channel WebSocket.
func (c *Client) ConnectMarket(ctx context.Context) error {
	return c.connect(ctx, c.MarketURL())
}

// ConnectUser opens the authenticated user-channel WebSocket.
func (c *Client) ConnectUser(ctx context.Context) error {
	return c.connect(ctx, c.UserURL())
}

// ConnectSports opens the public sports-channel WebSocket.
func (c *Client) ConnectSports(ctx context.Context) error {
	return c.connect(ctx, c.SportsURL())
}

func (c *Client) connect(ctx context.Context, url string) error {
	if c.closed.Load() {
		return errors.New("polymarket: websocket client is closed")
	}
	c.url.Store(url)

	conn, _, err := websocket.Dial(ctx, url, c.dialOpts)
	if err != nil {
		return fmt.Errorf("polymarket: websocket dial: %w", err)
	}

	if oldConn := c.conn.Swap(conn); oldConn != nil {
		_ = oldConn.CloseNow()
	}

	if !c.connected.CompareAndSwap(false, true) {
		if c.onReconnected != nil {
			go c.onReconnected()
		}
	} else {
		if c.onConnected != nil {
			go c.onConnected()
		}
	}

	go c.readLoop(c.ctx, conn)
	c.replaySubscriptions(ctx)
	return nil
}

// Close closes the active connection and stops background loops.
func (c *Client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.cancel()
	conn := c.conn.Load()
	if conn != nil {
		_ = conn.CloseNow()
	}
	if c.connected.Load() && c.onDisconnected != nil {
		go c.onDisconnected()
	}
	return nil
}

// IsConnected reports whether a WebSocket connection is currently attached.
func (c *Client) IsConnected() bool {
	return !c.closed.Load() && c.conn.Load() != nil
}

// Events returns decoded CLOB WebSocket events.
func (c *Client) Events() <-chan Event { return c.events }

// Errors returns asynchronous connection and decode errors.
func (c *Client) Errors() <-chan error { return c.errs }

// SubscribeOrderBook subscribes to order book snapshots and deltas for asset IDs.
func (c *Client) SubscribeOrderBook(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: assetIDs, initialDump: true})
}

// UnsubscribeOrderBook unsubscribes from order book events for asset IDs.
func (c *Client) UnsubscribeOrderBook(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeLastTradePrice subscribes to last-trade-price events for asset IDs.
func (c *Client) SubscribeLastTradePrice(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: assetIDs})
}

// SubscribePrices subscribes to price change events for asset IDs.
func (c *Client) SubscribePrices(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: assetIDs})
}

// UnsubscribePrices unsubscribes from price change events for asset IDs.
func (c *Client) UnsubscribePrices(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeTickSizeChange subscribes to tick-size-change events for asset IDs.
func (c *Client) SubscribeTickSizeChange(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: assetIDs})
}

// UnsubscribeTickSizeChange unsubscribes from tick-size-change events for asset IDs.
func (c *Client) UnsubscribeTickSizeChange(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeMidpoints subscribes to midpoint events for asset IDs.
func (c *Client) SubscribeMidpoints(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: assetIDs})
}

// UnsubscribeMidpoints unsubscribes from midpoint events for asset IDs.
func (c *Client) UnsubscribeMidpoints(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeBestBidAsk subscribes to best bid/ask events for asset IDs.
func (c *Client) SubscribeBestBidAsk(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: assetIDs, customFeatureEnabled: true})
}

// UnsubscribeBestBidAsk unsubscribes from best bid/ask events for asset IDs.
func (c *Client) UnsubscribeBestBidAsk(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// UnsubscribeLastTradePrice unsubscribes from last-trade-price events for asset IDs.
func (c *Client) UnsubscribeLastTradePrice(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeNewMarkets subscribes to new market listing events.
func (c *Client) SubscribeNewMarkets(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: assetIDs, customFeatureEnabled: true})
}

// UnsubscribeNewMarkets unsubscribes from new market listing events.
func (c *Client) UnsubscribeNewMarkets(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeMarketResolutions subscribes to market resolution events.
func (c *Client) SubscribeMarketResolutions(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: assetIDs, customFeatureEnabled: true})
}

// UnsubscribeMarketResolutions unsubscribes from market resolution events.
func (c *Client) UnsubscribeMarketResolutions(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeUserEvents subscribes to all user order and trade events for markets.
func (c *Client) SubscribeUserEvents(ctx context.Context, markets []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetUser, markets: markets})
}

// UnsubscribeUserEvents unsubscribes from user events for markets.
func (c *Client) UnsubscribeUserEvents(ctx context.Context, markets []string) error {
	return c.removeAndSend(ctx, subscriptionTargetUser, markets)
}

// SubscribeOrders subscribes to user order status events for markets.
func (c *Client) SubscribeOrders(ctx context.Context, markets []string) error {
	return c.SubscribeUserEvents(ctx, markets)
}

// UnsubscribeOrders unsubscribes from user order events for markets.
func (c *Client) UnsubscribeOrders(ctx context.Context, markets []string) error {
	return c.UnsubscribeUserEvents(ctx, markets)
}

// SubscribeTrades subscribes to user trade events for markets.
func (c *Client) SubscribeTrades(ctx context.Context, markets []string) error {
	return c.SubscribeUserEvents(ctx, markets)
}

// UnsubscribeTrades unsubscribes from user trade events for markets.
func (c *Client) UnsubscribeTrades(ctx context.Context, markets []string) error {
	return c.UnsubscribeUserEvents(ctx, markets)
}

func (c *Client) addAndSend(ctx context.Context, sub subscription) error {
	c.subsMu.Lock()
	c.subs = append(c.subs, sub)
	c.subsMu.Unlock()
	if err := c.sendSubscription(ctx, sub, ""); err != nil {
		c.removeMatchingSubscription(sub)
		return err
	}
	return nil
}

func (c *Client) removeAndSend(ctx context.Context, target subscriptionTarget, ids []string) error {
	sub := subscription{target: target}
	if target == subscriptionTargetUser {
		sub.markets = ids
	} else {
		sub.assetIDs = ids
	}
	c.removeMatchingSubscription(sub)
	return c.sendSubscription(ctx, sub, "unsubscribe")
}

func (c *Client) replaySubscriptions(ctx context.Context) {
	c.subsMu.RLock()
	subs := append([]subscription(nil), c.subs...)
	c.subsMu.RUnlock()
	for _, sub := range subs {
		if err := c.sendSubscription(ctx, sub, ""); err != nil {
			c.sendErr(fmt.Errorf("polymarket: replay websocket subscription: %w", err))
		}
	}
}

func (c *Client) sendSubscription(ctx context.Context, sub subscription, operation string) error {
	conn := c.conn.Load()
	if conn == nil {
		return errors.New("polymarket: websocket is not connected")
	}
	if sub.target == subscriptionTargetUser {
		auth, err := c.wsAuth()
		if err != nil {
			return err
		}
		return wsjson.Write(ctx, conn, UserSubscription{
			Type:      ChannelUser,
			Auth:      auth,
			Markets:   sub.markets,
			Operation: operation,
		})
	}
	return wsjson.Write(ctx, conn, MarketSubscription{
		Type:                 ChannelMarket,
		Operation:            operation,
		AssetIDs:             sub.assetIDs,
		InitialDump:          sub.initialDump,
		CustomFeatureEnabled: sub.customFeatureEnabled,
	})
}

func (c *Client) wsAuth() (clob.WSAuth, error) {
	if c.creds == nil {
		return clob.WSAuth{}, errors.New("polymarket: websocket user subscriptions require credentials")
	}
	return clob.WSAuth{
		APIKey:     c.creds.Key,
		Secret:     c.creds.Secret,
		Passphrase: c.creds.Passphrase,
	}, nil
}

func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if c.ctx.Err() != nil || websocket.CloseStatus(err) != -1 {
				return
			}
			c.sendErr(fmt.Errorf("polymarket: websocket read: %w", err))
			c.scheduleReconnect(conn)
			return
		}
		if bytes.Equal(data, []byte("PONG")) {
			continue
		}
		for _, event := range decodeEvents(data) {
			if event.err != nil {
				c.sendErr(event.err)
				continue
			}
			select {
			case c.events <- event.event:
			case <-c.ctx.Done():
				return
			}
		}
	}
}

func (c *Client) scheduleReconnect(conn *websocket.Conn) {
	if c.closed.Load() || !c.autoReconnect {
		if !c.autoReconnect && c.onDisconnected != nil {
			go c.onDisconnected()
		}
		return
	}
	if !c.reconnecting.CompareAndSwap(false, true) {
		return
	}
	if !c.conn.CompareAndSwap(conn, nil) {
		_ = conn.CloseNow()
		c.reconnecting.CompareAndSwap(true, false)
		return
	}
	go func() {
		defer func() {
			c.reconnecting.CompareAndSwap(true, false)
		}()
		backoff := time.Second
		for {
			if c.closed.Load() {
				return
			}
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(backoff):
			}
			dialCtx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
			err := c.connect(dialCtx, c.url.Load())
			cancel()
			if err == nil {
				return
			}
			c.sendErr(err)
			if backoff < time.Minute {
				backoff *= 2
			}
		}
	}()
}

func (c *Client) removeMatchingSubscription(target subscription) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	for idx := len(c.subs) - 1; idx >= 0; idx-- {
		sub := c.subs[idx]
		if sub.target != target.target {
			continue
		}
		if target.target == subscriptionTargetUser && sameStrings(sub.markets, target.markets) {
			c.subs = append(c.subs[:idx], c.subs[idx+1:]...)
			return
		}
		if target.target == subscriptionTargetMarket && sameStrings(sub.assetIDs, target.assetIDs) {
			c.subs = append(c.subs[:idx], c.subs[idx+1:]...)
			return
		}
	}
}

func (c *Client) sendErr(err error) {
	select {
	case c.errs <- err:
	default:
	}
}

type decodedEvent struct {
	event Event
	err   error
}

func decodeEvents(data []byte) []decodedEvent {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || !bytes.Contains(trimmed, []byte(`"event_type"`)) {
		return nil
	}
	if trimmed[0] == '[' {
		var raw []json.RawMessage
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return []decodedEvent{{err: fmt.Errorf("polymarket: decode websocket event array: %w", err)}}
		}
		out := make([]decodedEvent, 0, len(raw))
		for _, msg := range raw {
			event, err := DecodeEvent(msg)
			out = append(out, decodedEvent{event: event, err: err})
		}
		return out
	}
	var batch struct {
		BaseEvent
		Market       string             `json:"market"`
		Timestamp    string             `json:"timestamp"`
		PriceChanges []PriceChangeEvent `json:"price_changes"`
	}

	if err := json.Unmarshal(trimmed, &batch); err == nil &&
		batch.EventType == EventTypePriceChange &&
		len(batch.PriceChanges) > 0 {

		out := make([]decodedEvent, 0, len(batch.PriceChanges))
		for i := range batch.PriceChanges {
			change := batch.PriceChanges[i]
			change.BaseEvent = batch.BaseEvent

			if change.Market == "" {
				change.Market = batch.Market
			}
			if change.Timestamp == "" {
				change.Timestamp = batch.Timestamp
			}

			out = append(out, decodedEvent{event: &change})
		}
		return out
	}
	event, err := DecodeEvent(trimmed)
	return []decodedEvent{{event: event, err: err}}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, value := range a {
		seen[value]++
	}
	for _, value := range b {
		if seen[value] == 0 {
			return false
		}
		seen[value]--
	}
	return true
}
