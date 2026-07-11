package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/bububa/polymarket-client/combo"
	"github.com/bububa/polymarket-client/internal/polyauth"
	pmtypes "github.com/bububa/polymarket-client/shared"
	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/common"
)

const DefaultHost = "wss://combos-rfq-gateway-quoter.polymarket.com/ws/rfq"

type Client struct {
	host        string
	credentials *clob.Credentials
	identity    combo.Identity
	signer      *polyauth.Signer
	chainID     int64
	dialOptions *websocket.DialOptions
	authTimeout time.Duration
	ackTimeout  time.Duration
	eventBuffer int
	maxPending  int
}

type Option func(*Client)

func WithHost(host string) Option {
	return func(c *Client) {
		if host != "" {
			c.host = host
		}
	}
}
func WithCredentials(credentials clob.Credentials) Option {
	return func(c *Client) { c.credentials = &credentials }
}
func WithIdentity(identity combo.Identity) Option { return func(c *Client) { c.identity = identity } }
func WithSigner(signer *polyauth.Signer) Option   { return func(c *Client) { c.signer = signer } }
func WithChainID(chainID int64) Option            { return func(c *Client) { c.chainID = chainID } }
func WithDialOptions(options *websocket.DialOptions) Option {
	return func(c *Client) { c.dialOptions = options }
}
func WithAuthTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.authTimeout = timeout
		}
	}
}
func WithAckTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.ackTimeout = timeout
		}
	}
}
func WithEventBuffer(size int) Option {
	return func(c *Client) {
		if size > 0 {
			c.eventBuffer = size
		}
	}
}
func WithMaxPending(size int) Option {
	return func(c *Client) {
		if size > 0 {
			c.maxPending = size
		}
	}
}

func New(opts ...Option) *Client {
	c := &Client{host: DefaultHost, chainID: clob.PolygonChainID, authTimeout: 30 * time.Second, ackTimeout: 30 * time.Second, eventBuffer: 64, maxPending: 256}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Open(ctx context.Context) (*Session, error) {
	if c.credentials == nil {
		return nil, errors.New("combo ws: credentials are required")
	}
	if c.signer == nil {
		return nil, errors.New("combo ws: signer is required")
	}
	if c.identity.SignerAddress == "" || c.identity.MakerAddress == "" {
		return nil, errors.New("combo ws: identity is required")
	}
	conn, _, err := websocket.Dial(ctx, c.host, c.dialOptions)
	if err != nil {
		return nil, fmt.Errorf("combo ws: dial: %w", err)
	}
	runCtx, cancel := context.WithCancel(context.Background())
	comboClient := combo.NewClient("", combo.WithCredentials(*c.credentials), combo.WithIdentity(c.identity), combo.WithSigner(c.signer), combo.WithAuthAddress(c.signer.Address().Hex()), combo.WithChainID(c.chainID))
	s := &Session{conn: conn, ctx: runCtx, cancel: cancel, events: make(chan Event, c.eventBuffer), errs: make(chan error, 1), done: make(chan struct{}), readDone: make(chan struct{}), pending: make(map[string][]chan pendingResult), ackTimeout: c.ackTimeout, maxPending: c.maxPending, builder: combo.NewQuoteBuilder(comboClient), identity: c.identity}
	authCh := make(chan error, 1)
	s.auth = authCh
	go s.readLoop()
	msg := authMessage{Type: "auth", Auth: authCredentials{APIKey: c.credentials.Key, Secret: c.credentials.Secret, Passphrase: c.credentials.Passphrase}, Identity: c.identity}
	if err := s.write(ctx, msg); err != nil {
		s.Close()
		return nil, err
	}
	timer := time.NewTimer(c.authTimeout)
	defer timer.Stop()
	select {
	case err := <-authCh:
		if err != nil {
			s.Close()
			return nil, err
		}
		return s, nil
	case <-timer.C:
		s.Close()
		return nil, errors.New("combo ws: authentication timed out")
	case <-ctx.Done():
		s.Close()
		return nil, ctx.Err()
	}
}

type pendingResult struct {
	value any
	err   error
}

type Session struct {
	conn       *websocket.Conn
	ctx        context.Context
	cancel     context.CancelFunc
	events     chan Event
	errs       chan error
	done       chan struct{}
	readDone   chan struct{}
	closeOnce  sync.Once
	writeMu    sync.Mutex
	pendingMu  sync.Mutex
	pending    map[string][]chan pendingResult
	auth       chan error
	ackTimeout time.Duration
	maxPending int
	builder    *combo.QuoteBuilder
	identity   combo.Identity
}

func (s *Session) Events() <-chan Event { return s.events }
func (s *Session) Errors() <-chan error { return s.errs }

func (s *Session) Quote(ctx context.Context, request QuoteRequestEvent, opts QuoteOptions) (QuoteReference, error) {
	if request.RFQID == "" {
		return QuoteReference{}, errors.New("combo ws: rfq id is required")
	}
	built, err := s.builder.BuildAndSign(request.QuoteRequest, opts)
	if err != nil {
		return QuoteReference{}, err
	}
	key := "quote:" + string(request.RFQID)
	wait, err := s.addPending(key)
	if err != nil {
		return QuoteReference{}, err
	}
	if err := s.write(ctx, quoteMessage{Type: "RFQ_QUOTE", RFQID: request.RFQID, PriceE6: built.PriceE6, SizeE6: built.SizeE6, SignedOrder: built.SignedOrder}); err != nil {
		s.removePending(key, wait)
		return QuoteReference{}, err
	}
	value, err := s.wait(ctx, key, wait)
	if err != nil {
		return QuoteReference{}, err
	}
	ref, ok := value.(QuoteReference)
	if !ok {
		return QuoteReference{}, errors.New("combo ws: invalid quote acknowledgement")
	}
	return ref, nil
}
func (s *Session) CancelQuote(ctx context.Context, ref QuoteReference) (CancelQuoteAck, error) {
	if ref.RFQID == "" || ref.QuoteID == "" {
		return CancelQuoteAck{}, errors.New("combo ws: rfq id and quote id are required")
	}
	key := "cancel:" + string(ref.RFQID) + ":" + string(ref.QuoteID)
	wait, err := s.addPending(key)
	if err != nil {
		return CancelQuoteAck{}, err
	}
	msg := cancelMessage{Type: "RFQ_QUOTE_CANCEL", RFQID: ref.RFQID, QuoteID: ref.QuoteID, SignerAddress: s.identity.SignerAddress, MakerAddress: s.identity.MakerAddress}
	if err := s.write(ctx, msg); err != nil {
		s.removePending(key, wait)
		return CancelQuoteAck{}, err
	}
	value, err := s.wait(ctx, key, wait)
	if err != nil {
		return CancelQuoteAck{}, err
	}
	ack, ok := value.(CancelQuoteAck)
	if !ok {
		return CancelQuoteAck{}, errors.New("combo ws: invalid cancel acknowledgement")
	}
	return ack, nil
}
func (s *Session) RespondConfirmation(ctx context.Context, rfqID combo.RFQID, quoteID combo.QuoteID, decision combo.ConfirmationDecision) (ConfirmationAck, error) {
	if rfqID == "" || quoteID == "" {
		return ConfirmationAck{}, errors.New("combo ws: rfq id and quote id are required")
	}
	if decision != combo.ConfirmationConfirm && decision != combo.ConfirmationDecline {
		return ConfirmationAck{}, errors.New("combo ws: invalid confirmation decision")
	}
	key := "confirm:" + string(rfqID) + ":" + string(quoteID)
	wait, err := s.addPending(key)
	if err != nil {
		return ConfirmationAck{}, err
	}
	if err := s.write(ctx, confirmationMessage{Type: "RFQ_CONFIRMATION_RESPONSE", RFQID: rfqID, QuoteID: quoteID, Decision: decision}); err != nil {
		s.removePending(key, wait)
		return ConfirmationAck{}, err
	}
	value, err := s.wait(ctx, key, wait)
	if err != nil {
		return ConfirmationAck{}, err
	}
	ack, ok := value.(ConfirmationAck)
	if !ok {
		return ConfirmationAck{}, errors.New("combo ws: invalid confirmation acknowledgement")
	}
	return ack, nil
}

func (s *Session) write(ctx context.Context, value any) error {
	select {
	case <-s.done:
		return errors.New("combo ws: session closed")
	default:
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("combo ws: encode message: %w", err)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.Write(ctx, websocket.MessageText, payload)
}
func (s *Session) addPending(key string) (chan pendingResult, error) {
	ch := make(chan pendingResult, 1)
	s.pendingMu.Lock()
	select {
	case <-s.done:
		s.pendingMu.Unlock()
		return nil, errors.New("combo ws: session closed")
	default:
	}
	if len(s.pending[key]) > 0 {
		s.pendingMu.Unlock()
		return nil, errors.New("combo ws: operation already pending for correlation key")
	}
	count := 0
	for _, items := range s.pending {
		count += len(items)
	}
	if count >= s.maxPending {
		s.pendingMu.Unlock()
		return nil, errors.New("combo ws: too many pending requests")
	}
	s.pending[key] = append(s.pending[key], ch)
	s.pendingMu.Unlock()
	return ch, nil
}
func (s *Session) removePending(key string, target chan pendingResult) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	items := s.pending[key]
	for i, item := range items {
		if item == target {
			items = append(items[:i], items[i+1:]...)
			break
		}
	}
	if len(items) == 0 {
		delete(s.pending, key)
	} else {
		s.pending[key] = items
	}
}
func (s *Session) resolve(key string, value any, err error) {
	s.pendingMu.Lock()
	items := s.pending[key]
	if len(items) > 0 {
		s.pending[key] = items[1:]
		if len(s.pending[key]) == 0 {
			delete(s.pending, key)
		}
	}
	s.pendingMu.Unlock()
	if len(items) > 0 {
		items[0] <- pendingResult{value: value, err: err}
	}
}
func (s *Session) wait(ctx context.Context, key string, ch chan pendingResult) (any, error) {
	timer := time.NewTimer(s.ackTimeout)
	defer timer.Stop()
	select {
	case result := <-ch:
		return result.value, result.err
	case <-timer.C:
		s.removePending(key, ch)
		return nil, errors.New("combo ws: acknowledgement timed out")
	case <-ctx.Done():
		s.removePending(key, ch)
		return nil, ctx.Err()
	case <-s.done:
		return nil, errors.New("combo ws: session closed")
	}
}

func (s *Session) readLoop() {
	defer func() {
		close(s.events)
		close(s.errs)
		close(s.readDone)
	}()
	for {
		_, payload, err := s.conn.Read(s.ctx)
		if err != nil {
			s.shutdown(fmt.Errorf("combo ws: read: %w", err))
			return
		}
		if err := s.handle(payload); err != nil {
			s.shutdown(err)
			return
		}
	}
}
func (s *Session) handle(payload []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &head); err != nil {
		return fmt.Errorf("combo ws: decode message: %w", err)
	}
	switch head.Type {
	case "auth":
		var msg authResponse
		if err := json.Unmarshal(payload, &msg); err != nil {
			return err
		}
		if !msg.Success {
			s.auth <- fmt.Errorf("combo ws: authentication failed: %s", msg.Error)
		} else {
			s.auth <- nil
		}
		return nil
	case "RFQ_REQUEST":
		var wire struct {
			Type string `json:"type"`
			combo.QuoteRequest
		}
		if err := json.Unmarshal(payload, &wire); err != nil {
			return fmt.Errorf("combo ws: decode quote request: %w", err)
		}
		if err := validateQuoteRequest(wire.QuoteRequest); err != nil {
			return err
		}
		return s.push(QuoteRequestEvent{wire.QuoteRequest})
	case "RFQ_CONFIRMATION_REQUEST":
		var event ConfirmationRequestEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("combo ws: decode confirmation request: %w", err)
		}
		if err := s.validateConfirmationRequest(event); err != nil {
			return err
		}
		return s.push(event)
	case "RFQ_EXECUTION_UPDATE":
		var event ExecutionUpdateEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("combo ws: decode execution update: %w", err)
		}
		if event.RFQID == "" || !validExecutionStatus(event.Status) {
			return errors.New("combo ws: malformed execution update")
		}
		return s.push(event)
	case "RFQ_TRADE":
		var event TradeEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("combo ws: decode trade: %w", err)
		}
		if err := validateTradeEvent(event); err != nil {
			return err
		}
		return s.push(event)
	case "ACK_RFQ_QUOTE":
		var ref QuoteReference
		if err := json.Unmarshal(payload, &ref); err != nil {
			return err
		}
		if ref.RFQID == "" || ref.QuoteID == "" {
			return errors.New("combo ws: malformed quote acknowledgement")
		}
		s.resolve("quote:"+string(ref.RFQID), ref, nil)
		return nil
	case "ACK_RFQ_QUOTE_CANCEL":
		var ref QuoteReference
		if err := json.Unmarshal(payload, &ref); err != nil {
			return err
		}
		if ref.RFQID == "" || ref.QuoteID == "" {
			return errors.New("combo ws: malformed quote cancellation acknowledgement")
		}
		s.resolve("cancel:"+string(ref.RFQID)+":"+string(ref.QuoteID), ref, nil)
		return nil
	case "ACK_RFQ_CONFIRMATION_RESPONSE":
		var ack ConfirmationAck
		if err := json.Unmarshal(payload, &ack); err != nil {
			return err
		}
		if ack.RFQID == "" || ack.QuoteID == "" ||
			(ack.Decision != combo.ConfirmationConfirm && ack.Decision != combo.ConfirmationDecline) {
			return errors.New("combo ws: malformed confirmation acknowledgement")
		}
		s.resolve("confirm:"+string(ack.RFQID)+":"+string(ack.QuoteID), ack, nil)
		return nil
	case "RFQ_ERROR":
		var rfqErr RFQError
		if err := json.Unmarshal(payload, &rfqErr); err != nil {
			return err
		}
		if rfqErr.Code == "" || rfqErr.Message == "" {
			return errors.New("combo ws: malformed rfq error")
		}
		key, err := errorKey(rfqErr)
		if err != nil {
			return err
		}
		if key != "" {
			s.resolve(key, nil, &rfqErr)
			return nil
		}
		s.reportError(&rfqErr)
		return nil
	default:
		return s.push(UnknownEvent{Type: head.Type, Raw: append(json.RawMessage(nil), payload...)})
	}
}

func validExecutionStatus(status combo.ExecutionStatus) bool {
	switch status {
	case combo.ExecutionMatched, combo.ExecutionMined, combo.ExecutionConfirmed, combo.ExecutionRetrying, combo.ExecutionFailed:
		return true
	default:
		return false
	}
}

func validateQuoteRequest(request combo.QuoteRequest) error {
	if request.RFQID == "" || request.RequestorPublicID == "" || len(request.LegPositionIDs) == 0 || request.SubmissionDeadline <= 0 {
		return errors.New("combo ws: malformed quote request")
	}
	if _, err := combo.NormalizeConditionID(string(request.ConditionID)); err != nil {
		return fmt.Errorf("combo ws: malformed quote request: %w", err)
	}
	if err := validatePositionIDs(request.LegPositionIDs, request.YesPositionID, request.NoPositionID); err != nil {
		return fmt.Errorf("combo ws: malformed quote request: %w", err)
	}
	if request.Direction != combo.DirectionBuy && request.Direction != combo.DirectionSell {
		return errors.New("combo ws: malformed quote request direction")
	}
	if request.Side != combo.SideYes {
		return errors.New("combo ws: malformed quote request side")
	}
	if request.RequestedSize.Unit != combo.RequestedSizeNotional && request.RequestedSize.Unit != combo.RequestedSizeShares {
		return errors.New("combo ws: malformed quote request size unit")
	}
	if _, ok := positiveWireInteger(string(request.RequestedSize.ValueE6)); !ok {
		return errors.New("combo ws: malformed quote request size")
	}
	return nil
}

func (s *Session) validateConfirmationRequest(event ConfirmationRequestEvent) error {
	if event.RFQID == "" || event.QuoteID == "" || event.ConfirmBy <= 0 {
		return errors.New("combo ws: malformed confirmation request")
	}
	if !common.IsHexAddress(event.SignerAddress) || !common.IsHexAddress(event.MakerAddress) ||
		common.HexToAddress(event.SignerAddress) == (common.Address{}) || common.HexToAddress(event.MakerAddress) == (common.Address{}) {
		return errors.New("combo ws: malformed confirmation request identity")
	}
	if !strings.EqualFold(event.SignerAddress, s.identity.SignerAddress) ||
		!strings.EqualFold(event.MakerAddress, s.identity.MakerAddress) || event.SignatureType != s.identity.SignatureType {
		return errors.New("combo ws: confirmation request identity does not match session")
	}
	if _, err := combo.NormalizeConditionID(string(event.ConditionID)); err != nil {
		return fmt.Errorf("combo ws: malformed confirmation request: %w", err)
	}
	if err := validatePositionIDs(event.LegPositionIDs, event.YesPositionID, event.NoPositionID); err != nil {
		return fmt.Errorf("combo ws: malformed confirmation request: %w", err)
	}
	if event.Direction != combo.DirectionBuy && event.Direction != combo.DirectionSell {
		return errors.New("combo ws: malformed confirmation request direction")
	}
	if event.Side != combo.SideYes {
		return errors.New("combo ws: malformed confirmation request side")
	}
	price, ok := positiveWireInteger(string(event.PriceE6))
	if !ok || price.Cmp(big.NewInt(1_000_000)) >= 0 {
		return errors.New("combo ws: malformed confirmation request price")
	}
	if _, ok := positiveWireInteger(string(event.FillSizeE6)); !ok {
		return errors.New("combo ws: malformed confirmation request fill size")
	}
	return nil
}

func validatePositionIDs(legs []pmtypes.String, yes, no pmtypes.String) error {
	if _, ok := positiveWireInteger(string(yes)); !ok {
		return errors.New("invalid YES position id")
	}
	if _, ok := positiveWireInteger(string(no)); !ok {
		return errors.New("invalid NO position id")
	}
	if yes == no {
		return errors.New("YES and NO position ids must differ")
	}
	for _, positionID := range legs {
		if _, ok := positiveWireInteger(string(positionID)); !ok {
			return errors.New("invalid leg position id")
		}
	}
	return nil
}

func validateTradeEvent(event TradeEvent) error {
	if event.RFQID == "" || event.RequesterID == "" || len(event.LegPositionIDs) == 0 || event.ExecutedAt <= 0 {
		return errors.New("combo ws: malformed trade")
	}
	if _, err := combo.NormalizeConditionID(string(event.ConditionID)); err != nil {
		return fmt.Errorf("combo ws: malformed trade: %w", err)
	}
	for _, positionID := range event.LegPositionIDs {
		if _, ok := positiveWireInteger(string(positionID)); !ok {
			return errors.New("combo ws: malformed trade position id")
		}
	}
	if event.Direction != combo.DirectionBuy && event.Direction != combo.DirectionSell {
		return errors.New("combo ws: malformed trade direction")
	}
	if event.Side != combo.SideYes {
		return errors.New("combo ws: malformed trade side")
	}
	price, ok := positiveWireInteger(string(event.PriceE6))
	if !ok || price.Cmp(big.NewInt(1_000_000)) >= 0 {
		return errors.New("combo ws: malformed trade price")
	}
	if _, ok := positiveWireInteger(string(event.SizeE6)); !ok {
		return errors.New("combo ws: malformed trade size")
	}
	return nil
}

func positiveWireInteger(value string) (*big.Int, bool) {
	n, ok := new(big.Int).SetString(value, 10)
	return n, ok && n.Sign() > 0
}

func errorKey(e RFQError) (string, error) {
	switch e.RequestType {
	case "RFQ_QUOTE":
		if e.RFQID == "" {
			return "", errors.New("combo ws: uncorrelated quote error")
		}
		return "quote:" + string(e.RFQID), nil
	case "RFQ_QUOTE_CANCEL":
		if e.RFQID == "" || e.QuoteID == "" {
			return "", errors.New("combo ws: uncorrelated quote cancellation error")
		}
		return "cancel:" + string(e.RFQID) + ":" + string(e.QuoteID), nil
	case "RFQ_CONFIRMATION_RESPONSE":
		if e.RFQID == "" || e.QuoteID == "" {
			return "", errors.New("combo ws: uncorrelated confirmation error")
		}
		return "confirm:" + string(e.RFQID) + ":" + string(e.QuoteID), nil
	}
	return "", nil
}
func (s *Session) push(event Event) error {
	select {
	case <-s.done:
		return errors.New("combo ws: session closed")
	default:
	}
	select {
	case <-s.done:
		return errors.New("combo ws: session closed")
	case s.events <- event:
		return nil
	default:
		return errors.New("combo ws: event buffer is full")
	}
}
func (s *Session) reportError(err error) {
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case <-s.done:
		return
	case s.errs <- err:
	default:
	}
}

// reportTerminalError makes the reason for closing the session observable even
// when a previous asynchronous error occupies the bounded error buffer. The
// read loop is the sole non-terminal error producer, so an existing buffered
// item is necessarily older than this terminal failure.
func (s *Session) reportTerminalError(err error) {
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case s.errs <- err:
		return
	default:
	}
	select {
	case <-s.errs:
	default:
	}
	select {
	case s.errs <- err:
	case <-s.done:
	}
}

func (s *Session) shutdown(err error) {
	s.closeOnce.Do(func() {
		s.cancel()
		if err != nil {
			s.reportTerminalError(err)
		}
		s.pendingMu.Lock()
		pending := s.pending
		s.pending = make(map[string][]chan pendingResult)
		s.pendingMu.Unlock()
		for _, items := range pending {
			for _, ch := range items {
				ch <- pendingResult{err: errors.New("combo ws: session closed")}
			}
		}
		close(s.done)
		if s.conn != nil {
			_ = s.conn.Close(websocket.StatusNormalClosure, "closed")
		}
	})
}
func (s *Session) Close() error {
	s.shutdown(nil)
	<-s.readDone
	return nil
}
