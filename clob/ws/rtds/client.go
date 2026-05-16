package rtds

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"go.uber.org/atomic"
)

const (
	// DefaultHost is the production Polymarket RTDS WebSocket URL.
	DefaultHost = "wss://ws-live-data.polymarket.com"
	// DefaultHeartbeatInterval is the official RTDS keep-alive interval.
	DefaultHeartbeatInterval = 5 * time.Second
)

// Client is a reconnecting WebSocket client for Polymarket RTDS topics.
type Client struct {
	url      string
	dialOpts *websocket.DialOptions

	creds     *atomic.Pointer[Credentials]
	gammaAuth *atomic.Pointer[GammaAuth]
	conn      *atomic.Pointer[websocket.Conn]
	closed    *atomic.Bool
	connected *atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
	msgs      chan *Message
	errs      chan error

	autoReconnect      bool
	heartbeatInterval  time.Duration
	staleTimeout       time.Duration
	staleCheckInterval time.Duration
	lastDataAt         *atomic.Int64
	reconnecting       *atomic.Bool

	onConnected    func()
	onReconnected  func()
	onDisconnected func()

	subsMu sync.RWMutex
	subs   []Subscription
}

// Config configures an RTDS client.
type Config struct {
	// URL is the RTDS WebSocket URL.
	URL string
	// Header is sent during the WebSocket handshake.
	Header http.Header
	// Credentials are used when subscribing to authenticated topics.
	//
	// Deprecated: use GammaAuth.
	Credentials *Credentials
	// GammaAuth is used when subscribing to authenticated topics.
	GammaAuth *GammaAuth
	// AutoReconnect enables or disables automatic reconnect after read failures.
	AutoReconnect *bool
	// StaleTimeout forces reconnect when no message is received for the duration.
	StaleTimeout time.Duration
	// StaleCheckInterval sets how often stale detection runs.
	StaleCheckInterval time.Duration
	// OnConnected is fired when the WebSocket first connects.
	OnConnected func()
	// OnReconnected is fired when the WebSocket successfully reconnects.
	OnReconnected func()
	// OnDisconnected is fired when the connection drops and will not reconnect.
	OnDisconnected func()
}

// New creates an RTDS client.
func New(config Config) *Client {
	url := config.URL
	if url == "" {
		url = DefaultHost
	}
	ctx, cancel := context.WithCancel(context.Background())
	clt := &Client{
		url:                url,
		creds:              atomic.NewPointer[Credentials](config.Credentials),
		gammaAuth:          atomic.NewPointer[GammaAuth](config.GammaAuth),
		ctx:                ctx,
		cancel:             cancel,
		msgs:               make(chan *Message, 1024),
		errs:               make(chan error, 64),
		autoReconnect:      true,
		heartbeatInterval:  DefaultHeartbeatInterval,
		staleTimeout:       config.StaleTimeout,
		staleCheckInterval: config.StaleCheckInterval,
		onConnected:        config.OnConnected,
		onReconnected:      config.OnReconnected,
		onDisconnected:     config.OnDisconnected,

		conn:         atomic.NewPointer[websocket.Conn](nil),
		lastDataAt:   atomic.NewInt64(0),
		connected:    atomic.NewBool(false),
		reconnecting: atomic.NewBool(false),
		closed:       atomic.NewBool(false),
	}
	if config.AutoReconnect != nil {
		clt.autoReconnect = *config.AutoReconnect
	}
	if len(config.Header) > 0 {
		clt.dialOpts = &websocket.DialOptions{
			HTTPHeader: config.Header.Clone(),
		}
	}
	return clt
}

// NewClient creates an RTDS client for url.
func NewClient(url string) *Client {
	return New(Config{URL: url})
}

// WithCredentials sets legacy credentials for authenticated topic subscriptions.
//
// Deprecated: use WithGammaAuth.
func (c *Client) WithCredentials(creds *Credentials) *Client {
	c.creds.Store(creds)
	return c
}

// WithGammaAuth sets Gamma authentication for authenticated topic subscriptions.
func (c *Client) WithGammaAuth(auth *GammaAuth) *Client {
	c.gammaAuth.Store(auth)
	return c
}

// WithAutoReconnect enables or disables automatic reconnect after read failures.
func (c *Client) WithAutoReconnect(enabled bool) *Client {
	c.autoReconnect = enabled
	return c
}

// WithHeartbeatInterval sets the text PING interval for RTDS.
// Set interval <= 0 to disable heartbeat.
func (c *Client) WithHeartbeatInterval(interval time.Duration) *Client {
	c.heartbeatInterval = interval
	return c
}

// WithStaleTimeout enables stale stream detection.
// When enabled, the client forces reconnect if no message is received for the
// given duration. Set timeout <= 0 to disable it.
func (c *Client) WithStaleTimeout(timeout time.Duration) *Client {
	c.staleTimeout = timeout
	return c
}

// WithStaleCheckInterval sets how often stale stream detection runs.
// Set interval <= 0 to use a default derived from WithStaleTimeout.
func (c *Client) WithStaleCheckInterval(interval time.Duration) *Client {
	c.staleCheckInterval = interval
	return c
}

// WithOnConnected sets a callback fired when the WebSocket first connects.
func (c *Client) WithOnConnected(fn func()) *Client {
	c.onConnected = fn
	return c
}

// WithOnReconnected sets a callback fired when the WebSocket successfully reconnects.
func (c *Client) WithOnReconnected(fn func()) *Client {
	c.onReconnected = fn
	return c
}

// WithOnDisconnected sets a callback fired when the connection drops and will not reconnect.
func (c *Client) WithOnDisconnected(fn func()) *Client {
	c.onDisconnected = fn
	return c
}

// Connect opens the RTDS WebSocket and starts read loop.
func (c *Client) Connect(ctx context.Context) error {
	return c.connect(ctx)
}

func (c *Client) connect(ctx context.Context) error {
	if c.closed.Load() {
		return errors.New("polymarket: RTDS client is closed")
	}

	conn, _, err := websocket.Dial(ctx, c.url, c.dialOpts)
	if err != nil {
		return fmt.Errorf("polymarket: RTDS dial: %w", err)
	}

	if oldConn := c.conn.Swap(conn); oldConn != nil {
		_ = oldConn.CloseNow()
	}

	if !c.connected.CompareAndSwap(false, true) {
		if c.onReconnected != nil {
			go c.onReconnected()
		}
	} else if c.onConnected != nil {
		go c.onConnected()
	}

	go c.readLoop(c.ctx, conn)
	if c.heartbeatInterval > 0 {
		go c.heartbeat(c.ctx, conn)
	}
	if c.staleTimeout > 0 {
		c.markDataActive()
		go c.staleWatchdog(c.ctx, conn)
	}
	c.replaySubscriptions(ctx)
	return nil
}

// Close closes the active RTDS connection and stops background loops.
func (c *Client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.cancel()
	if conn := c.conn.Load(); conn != nil {
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

// Messages returns decoded RTDS messages.
func (c *Client) Messages() <-chan *Message { return c.msgs }

// Errors returns asynchronous RTDS connection and decode errors.
func (c *Client) Errors() <-chan error { return c.errs }

// Subscribe sends and records a topic subscription.
func (c *Client) Subscribe(ctx context.Context, sub Subscription) error {
	c.subsMu.Lock()
	c.subs = append(c.subs, sub)
	c.subsMu.Unlock()
	if err := c.sendJSON(ctx, SubscriptionRequest{Action: ActionSubscribe, Subscriptions: []Subscription{sub}}); err != nil {
		c.removeSubscription(sub)
		return err
	}
	return nil
}

// Unsubscribe sends and removes a topic subscription.
func (c *Client) Unsubscribe(ctx context.Context, sub Subscription) error {
	c.subsMu.Lock()
	c.removeSubscriptionLocked(sub)
	c.subsMu.Unlock()
	return c.sendJSON(ctx, SubscriptionRequest{Action: ActionUnsubscribe, Subscriptions: []Subscription{sub}})
}

// SubscribeCryptoPrices subscribes to Binance crypto price updates.
func (c *Client) SubscribeCryptoPrices(ctx context.Context, symbols []string) error {
	var filters any
	if len(symbols) > 0 {
		filters = strings.ToLower(strings.Join(symbols, ","))
	}
	return c.Subscribe(ctx, Subscription{Topic: TopicCryptoPrices, Type: TypeUpdate, Filters: filters})
}

// SubscribeChainlinkPrices subscribes to Chainlink crypto price updates.
func (c *Client) SubscribeChainlinkPrices(ctx context.Context, symbol string) error {
	var filters any
	if symbol != "" {
		filters = map[string]string{"symbol": strings.ToLower(symbol)}
	} else {
		filters = ""
	}
	return c.Subscribe(ctx, Subscription{Topic: TopicCryptoPricesChainlink, Type: TypeAll, Filters: filters})
}

// SubscribeEquityPrices subscribes to equity price updates for symbols.
func (c *Client) SubscribeEquityPrices(ctx context.Context, symbols []string) error {
	if len(symbols) == 0 {
		return c.Subscribe(ctx, Subscription{Topic: TopicEquityPrices, Type: TypeUpdate})
	}
	subs := make([]Subscription, 0, len(symbols))
	for _, symbol := range symbols {
		subs = append(subs, Subscription{
			Topic:   TopicEquityPrices,
			Type:    TypeUpdate,
			Filters: map[string]string{"symbol": strings.ToUpper(symbol)},
		})
	}
	return c.subscribeMany(ctx, subs)
}

// SubscribeEquityPrice subscribes to equity price updates for symbol.
func (c *Client) SubscribeEquityPrice(ctx context.Context, symbol string) error {
	return c.SubscribeEquityPrices(ctx, []string{symbol})
}

// SubscribeComments subscribes to comment events.
func (c *Client) SubscribeComments(ctx context.Context, commentType CommentType, creds *Credentials) error {
	var gammaAuth *GammaAuth
	if creds == nil {
		creds = c.creds.Load()
		gammaAuth = c.gammaAuth.Load()
	}
	eventType := string(commentType)
	if eventType == "" {
		eventType = TypeAll
	}
	return c.Subscribe(ctx, Subscription{Topic: TopicComments, Type: eventType, GammaAuth: gammaAuth, CLOBAuth: creds})
}

func (c *Client) subscribeMany(ctx context.Context, subs []Subscription) error {
	if len(subs) == 0 {
		return nil
	}
	c.subsMu.Lock()
	c.subs = append(c.subs, subs...)
	c.subsMu.Unlock()
	if err := c.sendJSON(ctx, SubscriptionRequest{Action: ActionSubscribe, Subscriptions: subs}); err != nil {
		for _, sub := range subs {
			c.removeSubscription(sub)
		}
		return err
	}
	return nil
}

func (c *Client) replaySubscriptions(ctx context.Context) {
	c.subsMu.RLock()
	subs := append([]Subscription(nil), c.subs...)
	c.subsMu.RUnlock()
	if len(subs) == 0 {
		return
	}
	if err := c.sendJSON(ctx, SubscriptionRequest{Action: ActionSubscribe, Subscriptions: subs}); err != nil {
		c.sendErr(fmt.Errorf("polymarket: replay RTDS subscriptions: %w", err))
	}
}

func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if c.ctx.Err() != nil || websocket.CloseStatus(err) != -1 {
				return
			}
			c.sendErr(fmt.Errorf("polymarket: RTDS read: %w", err))
			c.scheduleReconnect(conn)
			return
		}
		if len(bytes.TrimSpace(data)) > 0 {
			c.markDataActive()
		}
		if bytes.EqualFold(data, []byte("PONG")) {
			continue
		}
		if bytes.EqualFold(data, []byte("PING")) {
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, []byte("pong"))
			cancel()
			if err != nil {
				if c.ctx.Err() != nil || websocket.CloseStatus(err) != -1 {
					return
				}
				c.sendErr(fmt.Errorf("polymarket: RTDS pong: %w", err))
				c.scheduleReconnect(conn)
				return
			}
			continue
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.sendErr(fmt.Errorf("polymarket: decode RTDS message: %w", err))
			continue
		}
		select {
		case c.msgs <- &msg:
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) staleWatchdog(ctx context.Context, conn *websocket.Conn) {
	if c.staleTimeout <= 0 {
		return
	}
	interval := c.staleCheckInterval
	if interval <= 0 {
		interval = defaultStaleCheckInterval(c.staleTimeout)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.conn.Load() != conn {
				return
			}
			last := c.lastDataAt.Load()
			if last == 0 {
				continue
			}
			if age := time.Since(time.Unix(0, last)); age > c.staleTimeout {
				c.sendErr(fmt.Errorf("polymarket: RTDS websocket stale: no messages for %s", c.staleTimeout))
				_ = conn.CloseNow()
				c.scheduleReconnect(conn)
				return
			}
		}
	}
}

func (c *Client) markDataActive() {
	c.lastDataAt.Store(time.Now().UnixNano())
}

func defaultStaleCheckInterval(timeout time.Duration) time.Duration {
	interval := timeout / 2
	if interval <= 0 {
		return timeout
	}
	if interval < time.Second {
		return interval
	}
	if interval > 30*time.Second {
		return 30 * time.Second
	}
	return interval
}

func (c *Client) heartbeat(ctx context.Context, conn *websocket.Conn) {
	if c.heartbeatInterval <= 0 {
		return
	}
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.conn.Load() != conn {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, []byte("PING"))
			cancel()
			if err != nil {
				if c.ctx.Err() != nil || websocket.CloseStatus(err) != -1 {
					return
				}
				c.sendErr(fmt.Errorf("polymarket: RTDS websocket ping: %w", err))
				c.scheduleReconnect(conn)
				return
			}
		}
	}
}

func (c *Client) sendJSON(ctx context.Context, v any) error {
	conn := c.conn.Load()
	if conn == nil {
		return errors.New("polymarket: RTDS websocket is not connected")
	}
	return wsjson.Write(ctx, conn, v)
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
			err := c.connect(dialCtx)
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

func (c *Client) removeSubscription(target Subscription) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	c.removeSubscriptionLocked(target)
}

func (c *Client) removeSubscriptionLocked(target Subscription) {
	for idx := len(c.subs) - 1; idx >= 0; idx-- {
		sub := c.subs[idx]
		if sameSubscription(sub, target) {
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

func sameSubscription(a, b Subscription) bool {
	aKey, aOK := subscriptionIdentity(a)
	bKey, bOK := subscriptionIdentity(b)
	return aOK && bOK && aKey == bKey
}

func subscriptionIdentity(sub Subscription) (string, bool) {
	data, err := json.Marshal(sub)
	if err != nil {
		return "", false
	}
	return string(data), true
}
