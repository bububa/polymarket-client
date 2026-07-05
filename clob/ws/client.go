package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"go.uber.org/atomic"

	"github.com/bububa/polymarket-client/clob"
)

const (
	// DefaultHost is the production CLOB WebSocket host.
	DefaultHost       = "wss://ws-subscriptions-clob.polymarket.com"
	DefaultSportsHost = "wss://sports-api.polymarket.com"
	// DefaultHeartbeatInterval is the documented Market/User channel heartbeat interval.
	DefaultHeartbeatInterval = 10 * time.Second
)

type Callback func()

// Client is a reconnecting WebSocket client for CLOB market and user streams.
type Client struct {
	host       string
	sportsHost string
	url        *atomic.String

	dialOpts *websocket.DialOptions
	creds    *clob.Credentials

	conn               *atomic.Pointer[websocket.Conn]
	closed             *atomic.Bool
	connected          *atomic.Bool
	ctx                context.Context
	cancel             context.CancelFunc
	events             chan Event
	errs               chan error
	autoReconnect      bool
	heartbeatInterval  time.Duration
	staleTimeout       time.Duration
	staleCheckInterval time.Duration
	lastDataAt         *atomic.Int64
	reconnecting       *atomic.Bool

	onConnected    Callback
	onReconnected  Callback
	onDisconnected Callback

	marketMu   sync.Mutex
	marketSubs marketSubscriptionState

	userSubsMu sync.RWMutex
	userSubs   []subscription
}

type subscription struct {
	markets []string
}

type marketSubscriptionFeature uint8

const (
	marketFeatureOrderBook marketSubscriptionFeature = iota
	marketFeaturePrices
	marketFeatureLastTradePrice
	marketFeatureTickSizeChange
	marketFeatureMidpoints
	marketFeatureBestBidAsk
	marketFeatureNewMarkets
	marketFeatureMarketResolutions
)

type marketSubscriptionState struct {
	assets map[string]map[marketSubscriptionFeature]int
}

type marketSubscriptionFrame struct {
	Type                 Channel  `json:"type,omitempty"`
	Operation            string   `json:"operation,omitempty"`
	AssetIDs             []string `json:"assets_ids,omitempty"`
	InitialDump          bool     `json:"initial_dump,omitempty"`
	CustomFeatureEnabled bool     `json:"custom_feature_enabled,omitempty"`
}

func newMarketSubscriptionState() marketSubscriptionState {
	return marketSubscriptionState{
		assets: make(map[string]map[marketSubscriptionFeature]int),
	}
}

func (f marketSubscriptionFeature) requiresInitialDump() bool {
	return f == marketFeatureOrderBook
}

func (f marketSubscriptionFeature) requiresCustom() bool {
	switch f {
	case marketFeatureBestBidAsk, marketFeatureNewMarkets, marketFeatureMarketResolutions:
		return true
	default:
		return false
	}
}

func (s *marketSubscriptionState) featureCount(assetID string, feature marketSubscriptionFeature) int {
	refs := s.assets[assetID]
	if refs == nil {
		return 0
	}
	return refs[feature]
}

func (s *marketSubscriptionState) increment(assetID string, feature marketSubscriptionFeature) {
	refs := s.assets[assetID]
	if refs == nil {
		refs = make(map[marketSubscriptionFeature]int)
		s.assets[assetID] = refs
	}
	refs[feature]++
}

func (s *marketSubscriptionState) decrement(assetID string, feature marketSubscriptionFeature) {
	refs := s.assets[assetID]
	if refs == nil || refs[feature] == 0 {
		return
	}
	if refs[feature] == 1 {
		delete(refs, feature)
	} else {
		refs[feature]--
	}
	if len(refs) == 0 {
		delete(s.assets, assetID)
	}
}

func (s *marketSubscriptionState) assetActive(assetID string) bool {
	return len(s.assets[assetID]) > 0
}

func (s *marketSubscriptionState) assetActiveExcludingFeature(assetID string, feature marketSubscriptionFeature) bool {
	for existingFeature, count := range s.assets[assetID] {
		if existingFeature != feature && count > 0 {
			return true
		}
	}
	return false
}

func (s *marketSubscriptionState) assetCustomActive(assetID string) bool {
	for feature, count := range s.assets[assetID] {
		if count > 0 && feature.requiresCustom() {
			return true
		}
	}
	return false
}

func (s *marketSubscriptionState) activeCount() int {
	return len(s.assets)
}

func (s *marketSubscriptionState) activeAssetIDs() []string {
	ids := make([]string, 0, len(s.assets))
	for id, refs := range s.assets {
		if len(refs) > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (s *marketSubscriptionState) initialDumpActive() bool {
	for _, refs := range s.assets {
		if refs[marketFeatureOrderBook] > 0 {
			return true
		}
	}
	return false
}

func (s *marketSubscriptionState) customActive() bool {
	for _, refs := range s.assets {
		for feature, count := range refs {
			if count > 0 && feature.requiresCustom() {
				return true
			}
		}
	}
	return false
}

// New creates a CLOB WebSocket client.
func New(opts ...Option) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	clt := &Client{
		host:              DefaultHost,
		sportsHost:        DefaultSportsHost,
		ctx:               ctx,
		cancel:            cancel,
		events:            make(chan Event, 256),
		errs:              make(chan error, 16),
		autoReconnect:     true,
		heartbeatInterval: DefaultHeartbeatInterval,

		url:          atomic.NewString(""),
		conn:         atomic.NewPointer[websocket.Conn](nil),
		lastDataAt:   atomic.NewInt64(0),
		connected:    atomic.NewBool(false),
		reconnecting: atomic.NewBool(false),
		closed:       atomic.NewBool(false),

		marketSubs: newMarketSubscriptionState(),
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
func (c *Client) SportsURL() string { return c.sportsHost + "/ws" }

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
	return c.subscribeMarketAssets(ctx, marketFeatureOrderBook, assetIDs)
}

// UnsubscribeOrderBook unsubscribes from order book events for asset IDs.
func (c *Client) UnsubscribeOrderBook(ctx context.Context, assetIDs []string) error {
	return c.unsubscribeMarketAssets(ctx, marketFeatureOrderBook, assetIDs)
}

// SubscribeLastTradePrice subscribes to last-trade-price events for asset IDs.
func (c *Client) SubscribeLastTradePrice(ctx context.Context, assetIDs []string) error {
	return c.subscribeMarketAssets(ctx, marketFeatureLastTradePrice, assetIDs)
}

// SubscribePrices subscribes to price change events for asset IDs.
func (c *Client) SubscribePrices(ctx context.Context, assetIDs []string) error {
	return c.subscribeMarketAssets(ctx, marketFeaturePrices, assetIDs)
}

// UnsubscribePrices unsubscribes from price change events for asset IDs.
func (c *Client) UnsubscribePrices(ctx context.Context, assetIDs []string) error {
	return c.unsubscribeMarketAssets(ctx, marketFeaturePrices, assetIDs)
}

// SubscribeTickSizeChange subscribes to tick-size-change events for asset IDs.
func (c *Client) SubscribeTickSizeChange(ctx context.Context, assetIDs []string) error {
	return c.subscribeMarketAssets(ctx, marketFeatureTickSizeChange, assetIDs)
}

// UnsubscribeTickSizeChange unsubscribes from tick-size-change events for asset IDs.
func (c *Client) UnsubscribeTickSizeChange(ctx context.Context, assetIDs []string) error {
	return c.unsubscribeMarketAssets(ctx, marketFeatureTickSizeChange, assetIDs)
}

// SubscribeMidpoints subscribes to midpoint events for asset IDs.
func (c *Client) SubscribeMidpoints(ctx context.Context, assetIDs []string) error {
	return c.subscribeMarketAssets(ctx, marketFeatureMidpoints, assetIDs)
}

// UnsubscribeMidpoints unsubscribes from midpoint events for asset IDs.
func (c *Client) UnsubscribeMidpoints(ctx context.Context, assetIDs []string) error {
	return c.unsubscribeMarketAssets(ctx, marketFeatureMidpoints, assetIDs)
}

// SubscribeBestBidAsk subscribes to best bid/ask events for asset IDs.
func (c *Client) SubscribeBestBidAsk(ctx context.Context, assetIDs []string) error {
	return c.subscribeMarketAssets(ctx, marketFeatureBestBidAsk, assetIDs)
}

// UnsubscribeBestBidAsk unsubscribes from best bid/ask events for asset IDs.
func (c *Client) UnsubscribeBestBidAsk(ctx context.Context, assetIDs []string) error {
	return c.unsubscribeMarketAssets(ctx, marketFeatureBestBidAsk, assetIDs)
}

// UnsubscribeLastTradePrice unsubscribes from last-trade-price events for asset IDs.
func (c *Client) UnsubscribeLastTradePrice(ctx context.Context, assetIDs []string) error {
	return c.unsubscribeMarketAssets(ctx, marketFeatureLastTradePrice, assetIDs)
}

// SubscribeNewMarkets subscribes to new market listing events.
func (c *Client) SubscribeNewMarkets(ctx context.Context, assetIDs []string) error {
	return c.subscribeMarketAssets(ctx, marketFeatureNewMarkets, assetIDs)
}

// UnsubscribeNewMarkets unsubscribes from new market listing events.
func (c *Client) UnsubscribeNewMarkets(ctx context.Context, assetIDs []string) error {
	return c.unsubscribeMarketAssets(ctx, marketFeatureNewMarkets, assetIDs)
}

// SubscribeMarketResolutions subscribes to market resolution events.
func (c *Client) SubscribeMarketResolutions(ctx context.Context, assetIDs []string) error {
	return c.subscribeMarketAssets(ctx, marketFeatureMarketResolutions, assetIDs)
}

// UnsubscribeMarketResolutions unsubscribes from market resolution events.
func (c *Client) UnsubscribeMarketResolutions(ctx context.Context, assetIDs []string) error {
	return c.unsubscribeMarketAssets(ctx, marketFeatureMarketResolutions, assetIDs)
}

// SubscribeUserEvents subscribes to all user order and trade events for markets.
func (c *Client) SubscribeUserEvents(ctx context.Context, markets []string) error {
	return c.addUserAndSend(ctx, subscription{markets: markets})
}

// UnsubscribeUserEvents unsubscribes from user events for markets.
func (c *Client) UnsubscribeUserEvents(ctx context.Context, markets []string) error {
	return c.removeUserAndSend(ctx, markets)
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

func (c *Client) replaySubscriptions(ctx context.Context) {
	if err := c.replayMarketSubscription(ctx); err != nil {
		c.sendErr(fmt.Errorf("polymarket: replay websocket market subscription: %w", err))
	}

	c.userSubsMu.RLock()
	subs := append([]subscription(nil), c.userSubs...)
	c.userSubsMu.RUnlock()
	for _, sub := range subs {
		if err := c.sendUserSubscription(ctx, sub, ""); err != nil {
			c.sendErr(fmt.Errorf("polymarket: replay websocket user subscription: %w", err))
		}
	}
}

func (c *Client) subscribeMarketAssets(ctx context.Context, feature marketSubscriptionFeature, assetIDs []string) error {
	ids := canonicalStrings(assetIDs)
	if len(ids) == 0 {
		return nil
	}

	c.marketMu.Lock()
	defer c.marketMu.Unlock()

	firstSubscribe := c.marketSubs.activeCount() == 0
	newIDs := make([]string, 0, len(ids))
	initialDumpIDs := make([]string, 0, len(ids))
	customEnableIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		assetActive := c.marketSubs.assetActive(id)
		if !assetActive {
			newIDs = append(newIDs, id)
		}
		if feature.requiresInitialDump() && c.marketSubs.featureCount(id, feature) == 0 {
			initialDumpIDs = append(initialDumpIDs, id)
		}
		if feature.requiresCustom() && !c.marketSubs.assetCustomActive(id) {
			customEnableIDs = append(customEnableIDs, id)
		}
	}

	operation := "subscribe"
	frameIDs := unionSortedStrings(unionSortedStrings(newIDs, customEnableIDs), initialDumpIDs)
	initialDump := feature.requiresInitialDump() && len(initialDumpIDs) > 0
	customFeatureEnabled := feature.requiresCustom() && len(frameIDs) > 0
	if firstSubscribe {
		operation = ""
		frameIDs = ids
		initialDump = feature.requiresInitialDump()
		customFeatureEnabled = feature.requiresCustom()
	}
	if len(frameIDs) > 0 {
		if err := c.sendMarketSubscription(ctx, frameIDs, operation, initialDump, customFeatureEnabled); err != nil {
			return err
		}
	}
	for _, id := range ids {
		c.marketSubs.increment(id, feature)
	}
	return nil
}

func (c *Client) unsubscribeMarketAssets(ctx context.Context, feature marketSubscriptionFeature, assetIDs []string) error {
	ids := canonicalStrings(assetIDs)
	if len(ids) == 0 {
		return nil
	}

	c.marketMu.Lock()
	defer c.marketMu.Unlock()

	removedIDs := make([]string, 0, len(ids))
	decrementIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		count := c.marketSubs.featureCount(id, feature)
		if count == 0 {
			continue
		}
		decrementIDs = append(decrementIDs, id)
		if count == 1 {
			if !c.marketSubs.assetActiveExcludingFeature(id, feature) {
				removedIDs = append(removedIDs, id)
			}
		}
	}
	if len(decrementIDs) == 0 {
		return nil
	}

	if len(removedIDs) > 0 {
		if err := c.sendMarketSubscription(ctx, removedIDs, "unsubscribe", false, false); err != nil {
			return err
		}
	}
	for _, id := range decrementIDs {
		c.marketSubs.decrement(id, feature)
	}
	return nil
}

func (c *Client) replayMarketSubscription(ctx context.Context) error {
	c.marketMu.Lock()
	defer c.marketMu.Unlock()

	ids := c.marketSubs.activeAssetIDs()
	if len(ids) == 0 {
		return nil
	}
	return c.sendMarketSubscription(ctx, ids, "", c.marketSubs.initialDumpActive(), c.marketSubs.customActive())
}

func (c *Client) sendMarketSubscription(ctx context.Context, assetIDs []string, operation string, initialDump bool, customFeatureEnabled bool) error {
	conn := c.conn.Load()
	if conn == nil {
		return errors.New("polymarket: websocket is not connected")
	}
	frame := marketSubscriptionFrame{
		Operation:            operation,
		AssetIDs:             assetIDs,
		InitialDump:          initialDump,
		CustomFeatureEnabled: customFeatureEnabled,
	}
	if operation == "" {
		frame.Type = ChannelMarket
	}
	return wsjson.Write(ctx, conn, frame)
}

func (c *Client) addUserAndSend(ctx context.Context, sub subscription) error {
	c.userSubsMu.Lock()
	c.userSubs = append(c.userSubs, sub)
	c.userSubsMu.Unlock()
	if err := c.sendUserSubscription(ctx, sub, ""); err != nil {
		c.removeMatchingUserSubscription(sub)
		return err
	}
	return nil
}

func (c *Client) removeUserAndSend(ctx context.Context, markets []string) error {
	sub := subscription{markets: markets}
	c.removeMatchingUserSubscription(sub)
	return c.sendUserSubscription(ctx, sub, "unsubscribe")
}

func (c *Client) sendUserSubscription(ctx context.Context, sub subscription, operation string) error {
	conn := c.conn.Load()
	if conn == nil {
		return errors.New("polymarket: websocket is not connected")
	}
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
		if len(bytes.TrimSpace(data)) > 0 {
			c.markDataActive()
		}
		if bytes.EqualFold(data, []byte("PONG")) {
			continue
		} else if bytes.EqualFold(data, []byte("PING")) {
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, []byte("pong"))
			cancel()
			if err != nil {
				if c.ctx.Err() != nil || websocket.CloseStatus(err) != -1 {
					return
				}
				c.sendErr(fmt.Errorf("polymarket: websocket pong: %w", err))
				c.scheduleReconnect(conn)
				return
			}
			continue
		}
		events := decodeEvents(data)
		for _, event := range events {
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
				c.sendErr(fmt.Errorf("polymarket: websocket stale: no messages for %s", c.staleTimeout))
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
				c.sendErr(fmt.Errorf("polymarket: websocket ping: %w", err))
				c.scheduleReconnect(conn)
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

func (c *Client) removeMatchingUserSubscription(target subscription) {
	c.userSubsMu.Lock()
	defer c.userSubsMu.Unlock()
	for idx := len(c.userSubs) - 1; idx >= 0; idx-- {
		sub := c.userSubs[idx]
		if sameStrings(sub.markets, target.markets) {
			c.userSubs = append(c.userSubs[:idx], c.userSubs[idx+1:]...)
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
	var batch PriceChangeBatchEvent
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
			if change.Timestamp.IsZero() {
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

func canonicalStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func unionSortedStrings(a, b []string) []string {
	if len(a) == 0 {
		return canonicalStrings(b)
	}
	if len(b) == 0 {
		return canonicalStrings(a)
	}
	merged := make([]string, 0, len(a)+len(b))
	merged = append(merged, a...)
	merged = append(merged, b...)
	return canonicalStrings(merged)
}
