package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/gorilla/websocket"
)

const (
	// DefaultHost is the production CLOB WebSocket host.
	DefaultHost = "wss://ws-subscriptions-clob.polymarket.com"

	defaultHeartbeatInterval = 50 * time.Second
	defaultHeartbeatTimeout  = 15 * time.Second
)

// Client is a reconnecting WebSocket client for CLOB market and user streams.
type Client struct {
	host string
	url  string

	dialer *websocket.Dialer
	header http.Header
	creds  *clob.Credentials

	mu      sync.Mutex
	writeMu sync.Mutex
	conn    *websocket.Conn
	closed  bool

	ctx    context.Context
	cancel context.CancelFunc

	events chan Event
	errs   chan error

	autoReconnect     bool
	reconnecting      bool
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration

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

// Config configures a CLOB WebSocket client.
type Config struct {
	// Host is the CLOB WebSocket host; it may also be a full /ws/... URL.
	Host string
	// URL overrides Host and is dialed directly by Connect.
	URL string
	// Dialer is used to open WebSocket connections.
	Dialer *websocket.Dialer
	// Header is sent during the WebSocket handshake.
	Header http.Header
	// Credentials are required for user-channel subscriptions.
	Credentials *clob.Credentials
}

// New creates a CLOB WebSocket client.
func New(config Config) *Client {
	host := strings.TrimRight(config.Host, "/")
	if host == "" {
		host = DefaultHost
	}
	dialURL := config.URL
	if dialURL == "" {
		if parsed, err := neturl.Parse(host); err == nil && parsed.Scheme != "" && parsed.Path != "" {
			dialURL = host
			parsed.Path = ""
			parsed.RawQuery = ""
			parsed.Fragment = ""
			host = strings.TrimRight(parsed.String(), "/")
		}
	}
	dialer := config.Dialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		host:              host,
		url:               dialURL,
		dialer:            dialer,
		header:            cloneHeader(config.Header),
		creds:             config.Credentials,
		ctx:               ctx,
		cancel:            cancel,
		events:            make(chan Event, 256),
		errs:              make(chan error, 16),
		autoReconnect:     true,
		heartbeatInterval: defaultHeartbeatInterval,
		heartbeatTimeout:  defaultHeartbeatTimeout,
	}
}

// NewClient creates a market-channel WebSocket client.
func NewClient(url string) *Client {
	config := Config{}
	if url != "" {
		config.URL = url
	}
	return New(config)
}

// NewAuthenticatedClient creates a user-channel WebSocket client.
func NewAuthenticatedClient(url string, creds clob.Credentials) *Client {
	if url == "" {
		url = DefaultHost + "/ws/user"
	}
	return New(Config{URL: url, Credentials: &creds})
}

// WithCredentials sets API credentials for future user-channel subscriptions.
func (c *Client) WithCredentials(creds clob.Credentials) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.creds = &creds
	return c
}

// WithAutoReconnect enables or disables automatic reconnect after read failures.
func (c *Client) WithAutoReconnect(enabled bool) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.autoReconnect = enabled
	return c
}

// WithHeartbeat configures plain-text PING/PONG heartbeat timing.
func (c *Client) WithHeartbeat(interval, timeout time.Duration) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	if interval > 0 {
		c.heartbeatInterval = interval
	}
	if timeout > 0 {
		c.heartbeatTimeout = timeout
	}
	return c
}

// Host returns the configured WebSocket host.
func (c *Client) Host() string { return c.host }

// MarketURL returns the market-channel WebSocket URL.
func (c *Client) MarketURL() string { return c.host + "/ws/market" }

// UserURL returns the user-channel WebSocket URL.
func (c *Client) UserURL() string { return c.host + "/ws/user" }

// SportsURL returns the public sports-channel WebSocket URL.
func (c *Client) SportsURL() string { return c.host + "/ws" }

// Connect opens the configured WebSocket URL and starts read and heartbeat loops.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	url := c.url
	if url == "" {
		url = c.MarketURL()
	}
	c.mu.Unlock()
	return c.connect(ctx, url)
}

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
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("polymarket: websocket client is closed")
	}
	dialer := c.dialer
	header := cloneHeader(c.header)
	c.url = url
	c.mu.Unlock()

	conn, _, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return fmt.Errorf("polymarket: websocket dial: %w", err)
	}

	c.mu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.conn = conn
	c.mu.Unlock()

	go c.readLoop(conn)
	go c.heartbeatLoop(conn)
	c.replaySubscriptions(ctx)
	return nil
}

// Close closes the active connection and stops background loops.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.cancel()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

// IsConnected reports whether a WebSocket connection is currently attached.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.closed && c.conn != nil
}

// Events returns decoded CLOB WebSocket events.
func (c *Client) Events() <-chan Event { return c.events }

// Errors returns asynchronous connection and decode errors.
func (c *Client) Errors() <-chan error { return c.errs }

// SubscribeOrderBook subscribes to order book snapshots and deltas for asset IDs.
func (c *Client) SubscribeOrderBook(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: cloneStrings(assetIDs), initialDump: true})
}

// UnsubscribeOrderBook unsubscribes from order book events for asset IDs.
func (c *Client) UnsubscribeOrderBook(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeLastTradePrice subscribes to last-trade-price events for asset IDs.
func (c *Client) SubscribeLastTradePrice(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: cloneStrings(assetIDs)})
}

// SubscribePrices subscribes to price change events for asset IDs.
func (c *Client) SubscribePrices(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: cloneStrings(assetIDs)})
}

// UnsubscribePrices unsubscribes from price change events for asset IDs.
func (c *Client) UnsubscribePrices(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeTickSizeChange subscribes to tick-size-change events for asset IDs.
func (c *Client) SubscribeTickSizeChange(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: cloneStrings(assetIDs)})
}

// UnsubscribeTickSizeChange unsubscribes from tick-size-change events for asset IDs.
func (c *Client) UnsubscribeTickSizeChange(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeMidpoints subscribes to midpoint events for asset IDs.
func (c *Client) SubscribeMidpoints(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: cloneStrings(assetIDs)})
}

// UnsubscribeMidpoints unsubscribes from midpoint events for asset IDs.
func (c *Client) UnsubscribeMidpoints(ctx context.Context, assetIDs []string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, assetIDs)
}

// SubscribeBestBidAsk subscribes to best bid/ask events for asset IDs.
func (c *Client) SubscribeBestBidAsk(ctx context.Context, assetIDs []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: cloneStrings(assetIDs), customFeatureEnabled: true})
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
func (c *Client) SubscribeNewMarkets(ctx context.Context, assetIDs ...[]string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: optionalStrings(assetIDs), customFeatureEnabled: true})
}

// UnsubscribeNewMarkets unsubscribes from new market listing events.
func (c *Client) UnsubscribeNewMarkets(ctx context.Context, assetIDs ...[]string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, optionalStrings(assetIDs))
}

// SubscribeMarketResolutions subscribes to market resolution events.
func (c *Client) SubscribeMarketResolutions(ctx context.Context, assetIDs ...[]string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetMarket, assetIDs: optionalStrings(assetIDs), customFeatureEnabled: true})
}

// UnsubscribeMarketResolutions unsubscribes from market resolution events.
func (c *Client) UnsubscribeMarketResolutions(ctx context.Context, assetIDs ...[]string) error {
	return c.removeAndSend(ctx, subscriptionTargetMarket, optionalStrings(assetIDs))
}

// SubscribeUserEvents subscribes to all user order and trade events for markets.
func (c *Client) SubscribeUserEvents(ctx context.Context, markets []string) error {
	return c.addAndSend(ctx, subscription{target: subscriptionTargetUser, markets: cloneStrings(markets)})
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
		c.subsMu.Lock()
		c.removeMatchingSubscriptionLocked(sub)
		c.subsMu.Unlock()
		return err
	}
	return nil
}

func (c *Client) removeAndSend(ctx context.Context, target subscriptionTarget, ids []string) error {
	sub := subscription{target: target}
	if target == subscriptionTargetUser {
		sub.markets = cloneStrings(ids)
	} else {
		sub.assetIDs = cloneStrings(ids)
	}

	c.subsMu.Lock()
	c.removeMatchingSubscriptionLocked(sub)
	c.subsMu.Unlock()
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
	if sub.target == subscriptionTargetUser {
		auth, err := c.wsAuth()
		if err != nil {
			return err
		}
		return c.sendJSON(ctx, UserSubscription{
			Type:      ChannelUser,
			Auth:      auth,
			Markets:   sub.markets,
			Operation: operation,
		})
	}
	return c.sendJSON(ctx, MarketSubscription{
		Type:                 ChannelMarket,
		Operation:            operation,
		AssetIDs:             sub.assetIDs,
		InitialDump:          sub.initialDump,
		CustomFeatureEnabled: sub.customFeatureEnabled,
	})
}

func (c *Client) wsAuth() (clob.WSAuth, error) {
	c.mu.Lock()
	creds := c.creds
	c.mu.Unlock()
	if creds == nil {
		return clob.WSAuth{}, errors.New("polymarket: websocket user subscriptions require credentials")
	}
	return clob.WSAuth{
		APIKey:     creds.Key,
		Secret:     creds.Secret,
		Passphrase: creds.Passphrase,
	}, nil
}

func (c *Client) readLoop(conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if c.ctx.Err() != nil {
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

func (c *Client) heartbeatLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			deadline := time.Now().Add(c.heartbeatTimeout)
			c.writeMu.Lock()
			_ = conn.SetWriteDeadline(deadline)
			err := conn.WriteMessage(websocket.TextMessage, []byte("PING"))
			_ = conn.SetWriteDeadline(time.Time{})
			c.writeMu.Unlock()
			if err != nil {
				c.sendErr(fmt.Errorf("polymarket: websocket heartbeat: %w", err))
				c.scheduleReconnect(conn)
				return
			}
		}
	}
}

func (c *Client) sendJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errors.New("polymarket: websocket is not connected")
	}
	done := make(chan error, 1)
	go func() {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		done <- conn.WriteMessage(websocket.TextMessage, data)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (c *Client) scheduleReconnect(conn *websocket.Conn) {
	c.mu.Lock()
	if c.closed || !c.autoReconnect || c.reconnecting || c.conn != conn {
		c.mu.Unlock()
		return
	}
	c.reconnecting = true
	url := c.url
	c.conn = nil
	c.mu.Unlock()

	go func() {
		defer func() {
			c.mu.Lock()
			c.reconnecting = false
			c.mu.Unlock()
		}()
		backoff := time.Second
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(backoff):
			}
			ctx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
			err := c.connect(ctx, url)
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

func (c *Client) removeMatchingSubscriptionLocked(target subscription) {
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
	event, err := DecodeEvent(trimmed)
	return []decodedEvent{{event: event, err: err}}
}

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return nil
	}
	return header.Clone()
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func optionalStrings(values [][]string) []string {
	if len(values) == 0 {
		return nil
	}
	return cloneStrings(values[0])
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
