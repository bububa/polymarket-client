package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/bububa/polymarket-client/combo"
	"github.com/coder/websocket"
)

func TestSessionQuoteAndAck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.CloseNow()
		ctx := context.Background()
		_, payload, err := conn.Read(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		var auth map[string]any
		if err := json.Unmarshal(payload, &auth); err != nil || auth["type"] != "auth" {
			t.Errorf("auth = %s, err = %v", payload, err)
			return
		}
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"auth","success":true,"address":"0x1","role":"maker"}`))
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"RFQ_REQUEST","rfq_id":"r1","requestor_public_id":"p1","leg_position_ids":["1","2"],"condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","yes_position_id":"11","no_position_id":"22","direction":"SELL","side":"YES","requested_size":{"unit":"shares","value_e6":"1000000"},"submission_deadline":1780575184000}`))
		_, payload, err = conn.Read(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		var quote map[string]any
		_ = json.Unmarshal(payload, &quote)
		if quote["type"] != "RFQ_QUOTE" {
			t.Errorf("quote = %s", payload)
			return
		}
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ACK_RFQ_QUOTE","rfq_id":"r1","quote_id":"q1"}`))
		_, _, _ = conn.Read(ctx)
	}))
	defer server.Close()
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	identity := combo.Identity{SignerAddress: signer.Address().Hex(), MakerAddress: signer.Address().Hex(), SignatureType: clob.SignatureTypeEOA}
	host := "ws" + strings.TrimPrefix(server.URL, "http")
	client := New(WithHost(host), WithCredentials(clob.Credentials{Key: "key", Secret: "secret", Passphrase: "pass"}), WithSigner(signer), WithIdentity(identity), WithAckTimeout(time.Second))
	session, err := client.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	select {
	case raw := <-session.Events():
		event, ok := raw.(QuoteRequestEvent)
		if !ok {
			t.Fatalf("event = %T", raw)
		}
		ref, err := session.Quote(context.Background(), event, QuoteOptions{Price: "0.45"})
		if err != nil {
			t.Fatal(err)
		}
		if ref.QuoteID != "q1" {
			t.Fatalf("ref = %+v", ref)
		}
	case err := <-session.Errors():
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("event timeout")
	}
}

func TestExecutionStatusTerminal(t *testing.T) {
	if !combo.ExecutionConfirmed.Terminal() || !combo.ExecutionFailed.Terminal() || combo.ExecutionRetrying.Terminal() {
		t.Fatal("unexpected terminal status")
	}
}

func TestSessionOperationsRejectMissingIDsBeforeRegisteringPending(t *testing.T) {
	session := &Session{pending: make(map[string][]chan pendingResult)}
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "quote missing rfq id",
			call: func() error {
				_, err := session.Quote(context.Background(), QuoteRequestEvent{}, QuoteOptions{})
				return err
			},
		},
		{
			name: "cancel missing rfq id",
			call: func() error {
				_, err := session.CancelQuote(context.Background(), QuoteReference{QuoteID: "q1"})
				return err
			},
		},
		{
			name: "cancel missing quote id",
			call: func() error {
				_, err := session.CancelQuote(context.Background(), QuoteReference{RFQID: "r1"})
				return err
			},
		},
		{
			name: "confirmation missing rfq id",
			call: func() error {
				_, err := session.RespondConfirmation(context.Background(), "", "q1", combo.ConfirmationConfirm)
				return err
			},
		},
		{
			name: "confirmation missing quote id",
			call: func() error {
				_, err := session.RespondConfirmation(context.Background(), "r1", "", combo.ConfirmationConfirm)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil || !strings.Contains(err.Error(), "id") {
				t.Fatalf("error = %v", err)
			}
			if len(session.pending) != 0 {
				t.Fatalf("pending = %+v", session.pending)
			}
		})
	}
}

func TestSessionRejectsDuplicatePendingCorrelationKey(t *testing.T) {
	session := &Session{
		done:       make(chan struct{}),
		pending:    make(map[string][]chan pendingResult),
		maxPending: 2,
	}
	const key = "quote:r1"
	first, err := session.addPending(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.addPending(key); err == nil || !strings.Contains(err.Error(), "already pending") {
		t.Fatalf("duplicate error = %v", err)
	}
	if got := len(session.pending[key]); got != 1 {
		t.Fatalf("pending count = %d", got)
	}

	session.removePending(key, first)
	second, err := session.addPending(key)
	if err != nil {
		t.Fatalf("register after removal: %v", err)
	}
	session.removePending(key, second)
}

func TestSessionTerminalErrorReplacesBufferedAsyncError(t *testing.T) {
	session := &Session{
		cancel:  func() {},
		errs:    make(chan error, 1),
		done:    make(chan struct{}),
		pending: make(map[string][]chan pendingResult),
	}
	session.reportError(errors.New("older asynchronous error"))
	session.shutdown(errors.New("terminal protocol error"))

	select {
	case err := <-session.Errors():
		if err == nil || err.Error() != "terminal protocol error" {
			t.Fatalf("error = %v", err)
		}
	default:
		t.Fatal("terminal error was not delivered")
	}
}

func TestHandleRejectsUncorrelatedAcknowledgementsAndErrors(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "quote ack missing rfq", payload: `{"type":"ACK_RFQ_QUOTE","quote_id":"q1"}`, want: "malformed quote acknowledgement"},
		{name: "quote ack missing quote", payload: `{"type":"ACK_RFQ_QUOTE","rfq_id":"r1"}`, want: "malformed quote acknowledgement"},
		{name: "cancel ack missing quote", payload: `{"type":"ACK_RFQ_QUOTE_CANCEL","rfq_id":"r1"}`, want: "malformed quote cancellation acknowledgement"},
		{name: "confirmation ack missing rfq", payload: `{"type":"ACK_RFQ_CONFIRMATION_RESPONSE","quote_id":"q1"}`, want: "malformed confirmation acknowledgement"},
		{name: "confirmation ack missing decision", payload: `{"type":"ACK_RFQ_CONFIRMATION_RESPONSE","rfq_id":"r1","quote_id":"q1"}`, want: "malformed confirmation acknowledgement"},
		{name: "confirmation ack invalid decision", payload: `{"type":"ACK_RFQ_CONFIRMATION_RESPONSE","rfq_id":"r1","quote_id":"q1","decision":"MAYBE"}`, want: "malformed confirmation acknowledgement"},
		{name: "error missing code", payload: `{"type":"RFQ_ERROR","request_type":"UNKNOWN","error":"invalid"}`, want: "malformed rfq error"},
		{name: "error missing message", payload: `{"type":"RFQ_ERROR","request_type":"UNKNOWN","code":"INVALID_QUOTE"}`, want: "malformed rfq error"},
		{name: "quote error missing rfq", payload: `{"type":"RFQ_ERROR","request_type":"RFQ_QUOTE","code":"INVALID_QUOTE","error":"invalid"}`, want: "uncorrelated quote error"},
		{name: "cancel error missing quote", payload: `{"type":"RFQ_ERROR","request_type":"RFQ_QUOTE_CANCEL","rfq_id":"r1","code":"INVALID_QUOTE","error":"invalid"}`, want: "uncorrelated quote cancellation error"},
		{name: "confirmation error missing quote", payload: `{"type":"RFQ_ERROR","request_type":"RFQ_CONFIRMATION_RESPONSE","rfq_id":"r1","code":"INVALID_CONFIRMATION","error":"invalid"}`, want: "uncorrelated confirmation error"},
		{name: "quote request missing requester", payload: `{"type":"RFQ_REQUEST","rfq_id":"r1","leg_position_ids":["1","2"],"condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","yes_position_id":"11","no_position_id":"22","direction":"SELL","side":"YES","requested_size":{"unit":"shares","value_e6":"1000000"},"submission_deadline":1780575184000}`, want: "malformed quote request"},
		{name: "quote request invalid direction", payload: `{"type":"RFQ_REQUEST","rfq_id":"r1","requestor_public_id":"p1","leg_position_ids":["1","2"],"condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","yes_position_id":"11","no_position_id":"22","direction":"HOLD","side":"YES","requested_size":{"unit":"shares","value_e6":"1000000"},"submission_deadline":1780575184000}`, want: "malformed quote request direction"},
		{name: "quote request zero size", payload: `{"type":"RFQ_REQUEST","rfq_id":"r1","requestor_public_id":"p1","leg_position_ids":["1","2"],"condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","yes_position_id":"11","no_position_id":"22","direction":"SELL","side":"YES","requested_size":{"unit":"shares","value_e6":"0"},"submission_deadline":1780575184000}`, want: "malformed quote request size"},
		{name: "execution update unknown status", payload: `{"type":"RFQ_EXECUTION_UPDATE","rfq_id":"r1","status":"CANCELLED"}`, want: "malformed execution update"},
		{name: "trade missing requester", payload: `{"type":"RFQ_TRADE","rfq_id":"r1","condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","leg_position_ids":["1"],"direction":"SELL","side":"YES","price_e6":"450000","size_e6":"1000000","executed_at":1780575184000}`, want: "malformed trade"},
		{name: "trade invalid condition", payload: `{"type":"RFQ_TRADE","rfq_id":"r1","requester_id":"p1","condition_id":"0x03aa","leg_position_ids":["1"],"direction":"SELL","side":"YES","price_e6":"450000","size_e6":"1000000","executed_at":1780575184000}`, want: "invalid condition id"},
		{name: "trade invalid position", payload: `{"type":"RFQ_TRADE","rfq_id":"r1","requester_id":"p1","condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","leg_position_ids":["not-a-number"],"direction":"SELL","side":"YES","price_e6":"450000","size_e6":"1000000","executed_at":1780575184000}`, want: "malformed trade position id"},
		{name: "trade invalid direction", payload: `{"type":"RFQ_TRADE","rfq_id":"r1","requester_id":"p1","condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","leg_position_ids":["1"],"direction":"HOLD","side":"YES","price_e6":"450000","size_e6":"1000000","executed_at":1780575184000}`, want: "malformed trade direction"},
		{name: "trade invalid side", payload: `{"type":"RFQ_TRADE","rfq_id":"r1","requester_id":"p1","condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","leg_position_ids":["1"],"direction":"SELL","side":"NO","price_e6":"450000","size_e6":"1000000","executed_at":1780575184000}`, want: "malformed trade side"},
		{name: "trade zero price", payload: `{"type":"RFQ_TRADE","rfq_id":"r1","requester_id":"p1","condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","leg_position_ids":["1"],"direction":"SELL","side":"YES","price_e6":"0","size_e6":"1000000","executed_at":1780575184000}`, want: "malformed trade price"},
		{name: "trade zero size", payload: `{"type":"RFQ_TRADE","rfq_id":"r1","requester_id":"p1","condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","leg_position_ids":["1"],"direction":"SELL","side":"YES","price_e6":"450000","size_e6":"0","executed_at":1780575184000}`, want: "malformed trade size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{pending: make(map[string][]chan pendingResult)}
			err := session.handle([]byte(tt.payload))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestHandlePreservesConfirmationAcknowledgementDecision(t *testing.T) {
	wait := make(chan pendingResult, 1)
	session := &Session{pending: map[string][]chan pendingResult{"confirm:r1:q1": {wait}}}
	if err := session.handle([]byte(`{"type":"ACK_RFQ_CONFIRMATION_RESPONSE","rfq_id":"r1","quote_id":"q1","decision":"CONFIRM"}`)); err != nil {
		t.Fatal(err)
	}
	result := <-wait
	if result.err != nil {
		t.Fatal(result.err)
	}
	ack, ok := result.value.(ConfirmationAck)
	if !ok {
		t.Fatalf("ack = %T", result.value)
	}
	if ack.RFQID != "r1" || ack.QuoteID != "q1" || ack.Decision != combo.ConfirmationConfirm {
		t.Fatalf("ack = %+v", ack)
	}
}

func TestHandleRejectsConfirmationIdentityMismatch(t *testing.T) {
	identity := combo.Identity{SignerAddress: "0x1111111111111111111111111111111111111111", MakerAddress: "0x1111111111111111111111111111111111111111", SignatureType: clob.SignatureTypeEOA}
	session := &Session{identity: identity}
	payload := `{"type":"RFQ_CONFIRMATION_REQUEST","rfq_id":"r1","quote_id":"q1","signer_address":"0x2222222222222222222222222222222222222222","maker_address":"0x2222222222222222222222222222222222222222","signature_type":0,"leg_position_ids":["1","2"],"condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","yes_position_id":"11","no_position_id":"22","direction":"SELL","side":"YES","fill_size_e6":"1000000","price_e6":"450000","confirm_by":1780575184000}`
	err := session.handle([]byte(payload))
	if err == nil || !strings.Contains(err.Error(), "identity does not match session") {
		t.Fatalf("error = %v", err)
	}
}

func TestHandleAcceptsValidExecutionAndTradeEvents(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "execution update", payload: `{"type":"RFQ_EXECUTION_UPDATE","rfq_id":"r1","status":"CONFIRMED","tx_hash":"0xabc"}`},
		{name: "trade", payload: `{"type":"RFQ_TRADE","rfq_id":"r1","requester_id":"p1","condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","leg_position_ids":["1","2"],"direction":"SELL","side":"YES","price_e6":"450000","size_e6":"1000000","executed_at":1780575184000}`},
		{name: "quote request", payload: `{"type":"RFQ_REQUEST","rfq_id":"r1","requestor_public_id":"p1","leg_position_ids":["1","2"],"condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","yes_position_id":"11","no_position_id":"22","direction":"SELL","side":"YES","requested_size":{"unit":"shares","value_e6":"1000000"},"submission_deadline":1780575184000}`},
		{name: "confirmation request", payload: `{"type":"RFQ_CONFIRMATION_REQUEST","rfq_id":"r1","quote_id":"q1","signer_address":"0x1111111111111111111111111111111111111111","maker_address":"0x1111111111111111111111111111111111111111","signature_type":0,"leg_position_ids":["1","2"],"condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","yes_position_id":"11","no_position_id":"22","direction":"SELL","side":"YES","fill_size_e6":"1000000","price_e6":"450000","confirm_by":1780575184000}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{events: make(chan Event, 1), done: make(chan struct{}), identity: combo.Identity{SignerAddress: "0x1111111111111111111111111111111111111111", MakerAddress: "0x1111111111111111111111111111111111111111", SignatureType: clob.SignatureTypeEOA}}
			if err := session.handle([]byte(tt.payload)); err != nil {
				t.Fatal(err)
			}
			select {
			case <-session.events:
			default:
				t.Fatal("event was not delivered")
			}
		})
	}
}

func TestSessionCloseRejectsPendingAndClosesOutputs(t *testing.T) {
	session, closeServer := openIdleSession(t)
	defer closeServer()

	keys := []string{"quote:r1", "cancel:r1:q1", "confirm:r1:q1"}
	waits := make([]chan pendingResult, 0, len(keys))
	for _, key := range keys {
		wait, err := session.addPending(key)
		if err != nil {
			t.Fatal(err)
		}
		waits = append(waits, wait)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	for i, wait := range waits {
		select {
		case result := <-wait:
			if result.err == nil || !strings.Contains(result.err.Error(), "session closed") {
				t.Fatalf("pending %d error = %v", i, result.err)
			}
		default:
			t.Fatalf("pending %d was not rejected", i)
		}
	}
	if _, ok := <-session.Events(); ok {
		t.Fatal("events channel is open")
	}
	for range session.Errors() {
	}
}

func TestSessionCloseWhileDispatching(t *testing.T) {
	payloads := []struct {
		name    string
		payload string
	}{
		{name: "quote request", payload: `{"type":"RFQ_REQUEST","rfq_id":"r1","requestor_public_id":"p1","leg_position_ids":["1","2"],"condition_id":"0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","yes_position_id":"11","no_position_id":"22","direction":"SELL","side":"YES","requested_size":{"unit":"shares","value_e6":"1000000"},"submission_deadline":1780575184000}`},
		{name: "unknown event", payload: `{"type":"RFQ_FUTURE_EVENT","value":1}`},
		{name: "uncorrelated error", payload: `{"type":"RFQ_ERROR","code":"RATE_LIMITED","error":"rate limited","error_id":"e1","request_type":"UNKNOWN"}`},
	}
	for _, tt := range payloads {
		t.Run(tt.name, func(t *testing.T) {
			for range 20 {
				sent := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					conn, err := websocket.Accept(w, r, nil)
					if err != nil {
						return
					}
					defer conn.CloseNow()
					ctx := context.Background()
					if _, _, err := conn.Read(ctx); err != nil {
						return
					}
					_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"auth","success":true}`))
					_ = conn.Write(ctx, websocket.MessageText, []byte(tt.payload))
					close(sent)
					_, _, _ = conn.Read(ctx)
				}))
				session := openSession(t, server.URL)
				<-sent
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				server.Close()
			}
		})
	}
}

func openIdleSession(t *testing.T) (*Session, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ctx := context.Background()
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"auth","success":true}`))
		_, _, _ = conn.Read(ctx)
	}))
	return openSession(t, server.URL), server.Close
}

func openSession(t *testing.T, serverURL string) *Session {
	t.Helper()
	signer, err := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	if err != nil {
		t.Fatal(err)
	}
	identity := combo.Identity{SignerAddress: signer.Address().Hex(), MakerAddress: signer.Address().Hex(), SignatureType: clob.SignatureTypeEOA}
	host := "ws" + strings.TrimPrefix(serverURL, "http")
	session, err := New(WithHost(host), WithCredentials(clob.Credentials{Key: "key", Secret: "secret", Passphrase: "pass"}), WithSigner(signer), WithIdentity(identity), WithAckTimeout(time.Second)).Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return session
}
