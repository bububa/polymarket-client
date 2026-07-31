package ws

import (
	"context"
	"time"

	"github.com/coder/websocket"
)

// handleConnectionFailure linearizes transport failure ownership. Read,
// heartbeat, and stale-detection workers can observe the same failure, but
// only the worker that detaches the active connection publishes it and
// schedules recovery.
func (c *Client) handleConnectionFailure(conn *websocket.Conn, operation string, err error) {
	if err == nil || c.closed.Load() || c.ctx.Err() != nil {
		return
	}
	c.lifecycleMu.Lock()
	if c.closed.Load() || c.ctx.Err() != nil {
		c.lifecycleMu.Unlock()
		return
	}
	if !c.conn.CompareAndSwap(conn, nil) {
		c.lifecycleMu.Unlock()
		return
	}
	if !c.autoReconnect {
		c.notifyDisconnectedLocked()
	}
	c.lifecycleMu.Unlock()

	_ = conn.CloseNow()
	c.sendErr(newConnectionError(operation, err))
	if !c.autoReconnect {
		return
	}
	c.requestReconnect()
}

// notifyDisconnectedLocked records the terminal callback for the connection
// generation currently protected by lifecycleMu.
func (c *Client) notifyDisconnectedLocked() {
	if !c.connected.Load() || c.onDisconnected == nil ||
		!c.disconnectedNotified.CompareAndSwap(false, true) {
		return
	}
	go c.onDisconnected()
}

func (c *Client) requestReconnect() {
	c.reconnectRequested.Store(true)
	if !c.reconnecting.CompareAndSwap(false, true) {
		return
	}
	go c.reconnectLoop()
}

func (c *Client) reconnectLoop() {
	backoff := time.Second
	for {
		if c.closed.Load() || c.ctx.Err() != nil {
			c.reconnecting.Store(false)
			return
		}
		c.reconnectRequested.Store(false)
		timer := time.NewTimer(backoff)
		select {
		case <-c.ctx.Done():
			timer.Stop()
			c.reconnecting.Store(false)
			return
		case <-timer.C:
		}

		dialCtx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
		err := c.connect(dialCtx, c.url.Load())
		cancel()
		if err != nil {
			if c.closed.Load() || c.ctx.Err() != nil {
				c.reconnecting.Store(false)
				return
			}
			c.sendErr(err)
			if backoff < time.Minute {
				backoff *= 2
			}
			continue
		}
		if c.reconnectRequested.Load() {
			backoff = time.Second
			continue
		}

		// Hand off reconnect ownership without losing a failure that races
		// with a successful dial returning.
		c.reconnecting.Store(false)
		if !c.reconnectRequested.Swap(false) ||
			!c.reconnecting.CompareAndSwap(false, true) {
			return
		}
		backoff = time.Second
	}
}
