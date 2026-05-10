package ws

import (
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/coder/websocket"
)

type Option func(*Client)

func WithHost(host string) Option {
	return func(clt *Client) {
		clt.host = host
	}
}

func WithSportsHost(host string) Option {
	return func(clt *Client) {
		clt.sportsHost = host
	}
}

// WithDialOptions sets custom dial options for the WebSocket connection.
func WithDialOptions(opts *websocket.DialOptions) Option {
	return func(clt *Client) {
		clt.dialOpts = opts
	}
}

func WithCredentials(cred *clob.Credentials) Option {
	return func(clt *Client) {
		clt.creds = cred
	}
}

func WithAutoReconnect(v bool) Option {
	return func(clt *Client) {
		clt.autoReconnect = v
	}
}

// WithHeartbeatInterval sets the text PING interval for Market/User channels.
// Set interval <= 0 to disable heartbeat.
func WithHeartbeatInterval(interval time.Duration) Option {
	return func(clt *Client) {
		clt.heartbeatInterval = interval
	}
}

// WithStaleTimeout enables stale stream detection.
// When enabled, the client forces a reconnect if no message is received for the
// given duration. Set timeout <= 0 to disable it.
func WithStaleTimeout(timeout time.Duration) Option {
	return func(clt *Client) {
		clt.staleTimeout = timeout
	}
}

// WithStaleCheckInterval sets how often stale stream detection runs.
// Set interval <= 0 to use a default derived from WithStaleTimeout.
func WithStaleCheckInterval(interval time.Duration) Option {
	return func(clt *Client) {
		clt.staleCheckInterval = interval
	}
}

// WithOnConnected sets a callback fired when the WebSocket first connects.
func WithOnConnected(fn func()) Option {
	return func(clt *Client) {
		clt.onConnected = fn
	}
}

// WithOnReconnected sets a callback fired when the WebSocket successfully reconnects.
func WithOnReconnected(fn func()) Option {
	return func(clt *Client) {
		clt.onReconnected = fn
	}
}

// WithOnDisconnected sets a callback fired when the connection drops and will not reconnect
// (because autoReconnect is disabled or the client is closed).
func WithOnDisconnected(fn func()) Option {
	return func(clt *Client) {
		clt.onDisconnected = fn
	}
}
