package rtds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// DefaultHost is the production Polymarket RTDS WebSocket URL.
	DefaultHost = "wss://rtds.polymarket.com"

	defaultHeartbeatInterval = 50 * time.Second
	defaultHeartbeatTimeout  = 15 * time.Second
)

// Client is a reconnecting WebSocket client for Polymarket RTDS topics.
type Client struct {
	url string

	dialer *websocket.Dialer
	header http.Header
	creds  *Credentials

	mu      sync.Mutex
	writeMu sync.Mutex
	conn    *websocket.Conn
	closed  bool

	ctx    context.Context
	cancel context.CancelFunc

	msgs chan *Message
	errs chan error

	autoReconnect     bool
	reconnecting      bool
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration

	subsMu sync.RWMutex
	subs   []Subscription
}

// Config configures an RTDS client.
type Config struct {
	// URL is the RTDS WebSocket URL.
	URL string
	// Dialer is used to open WebSocket connections.
	Dialer *websocket.Dialer
	// Header is sent during the WebSocket handshake.
	Header http.Header
	// Credentials are used when subscribing to authenticated topics.
	Credentials *Credentials
}

// New creates an RTDS client.
func New(config Config) *Client {
	url := config.URL
	if url == "" {
		url = DefaultHost
	}
	dialer := config.Dialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		url:               url,
		dialer:            dialer,
		header:            cloneHeader(config.Header),
		creds:             config.Credentials,
		ctx:               ctx,
		cancel:            cancel,
		msgs:              make(chan *Message, 1024),
		errs:              make(chan error, 64),
		autoReconnect:     true,
		heartbeatInterval: defaultHeartbeatInterval,
		heartbeatTimeout:  defaultHeartbeatTimeout,
	}
}

// NewClient creates an RTDS client for url.
func NewClient(url string) *Client {
	return New(Config{URL: url})
}

// WithCredentials sets credentials for authenticated topic subscriptions.
func (c *Client) WithCredentials(creds *Credentials) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.creds = creds
	return c
}

// WithAutoReconnect enables or disables automatic reconnect after read failures.
func (c *Client) WithAutoReconnect(enabled bool) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.autoReconnect = enabled
	return c
}

// Connect opens the RTDS WebSocket and starts read and heartbeat loops.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("polymarket: RTDS client is closed")
	}
	url := c.url
	dialer := c.dialer
	header := cloneHeader(c.header)
	c.mu.Unlock()

	conn, _, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return fmt.Errorf("polymarket: RTDS dial: %w", err)
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

// Close closes the active RTDS connection and stops background loops.
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

// Messages returns decoded RTDS messages.
func (c *Client) Messages() <-chan *Message { return c.msgs }

// Errors returns asynchronous RTDS connection and decode errors.
func (c *Client) Errors() <-chan error { return c.errs }

// Subscribe sends and records a topic subscription.
func (c *Client) Subscribe(ctx context.Context, sub Subscription) error {
	c.subsMu.Lock()
	c.subs = append(c.subs, sub)
	c.subsMu.Unlock()
	return c.sendJSON(ctx, SubscriptionRequest{Action: ActionSubscribe, Subscriptions: []Subscription{sub}})
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
		filters = symbols
	}
	return c.Subscribe(ctx, Subscription{Topic: "crypto_prices", Type: "update", Filters: filters})
}

// SubscribeChainlinkPrices subscribes to Chainlink crypto price updates.
func (c *Client) SubscribeChainlinkPrices(ctx context.Context, symbol string) error {
	var filters any
	if symbol != "" {
		filters = map[string]string{"symbol": symbol}
	}
	return c.Subscribe(ctx, Subscription{Topic: "crypto_prices_chainlink", Type: "*", Filters: filters})
}

// SubscribeComments subscribes to comment events.
func (c *Client) SubscribeComments(ctx context.Context, commentType CommentType, creds *Credentials) error {
	if creds == nil {
		c.mu.Lock()
		creds = c.creds
		c.mu.Unlock()
	}
	eventType := string(commentType)
	if eventType == "" {
		eventType = "*"
	}
	return c.Subscribe(ctx, Subscription{Topic: "comments", Type: eventType, CLOBAuth: creds})
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

func (c *Client) readLoop(conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if c.ctx.Err() != nil {
				return
			}
			c.sendErr(fmt.Errorf("polymarket: RTDS read: %w", err))
			c.scheduleReconnect(conn)
			return
		}
		if string(data) == "PONG" {
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

func (c *Client) heartbeatLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.writeMu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(c.heartbeatTimeout))
			err := conn.WriteMessage(websocket.TextMessage, []byte("PING"))
			_ = conn.SetWriteDeadline(time.Time{})
			c.writeMu.Unlock()
			if err != nil {
				c.sendErr(fmt.Errorf("polymarket: RTDS heartbeat: %w", err))
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
		return errors.New("polymarket: RTDS websocket is not connected")
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
			err := c.Connect(ctx)
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

func (c *Client) removeSubscriptionLocked(target Subscription) {
	for idx := len(c.subs) - 1; idx >= 0; idx-- {
		sub := c.subs[idx]
		if sub.Topic == target.Topic && sub.Type == target.Type {
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

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return nil
	}
	return header.Clone()
}
