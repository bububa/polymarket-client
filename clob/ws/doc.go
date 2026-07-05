// Package ws provides WebSocket clients for live Polymarket CLOB order book and user updates.
//
// # Connecting
//
// Basic market stream:
//
//	wsClient := ws.New()
//	err := wsClient.ConnectMarket(ctx)
//
// User stream with CLOB API credentials:
//
//	wsClient := ws.New(ws.Config{
//	    Credentials: &clob.Credentials{
//	        Key:        "...",
//	        Secret:     "...",
//	        Passphrase: "...",
//	    },
//	})
//	err := wsClient.ConnectUser(ctx)
//
// # Subscriptions
//
//	err = wsClient.SubscribeOrderBook(ctx, []string{"token-id"})
//	err = wsClient.SubscribeOrders(ctx, []string{"condition-id"})
//
// Market subscriptions share one connection-level asset set. New market assets
// are sent with subscription update frames, and reconnect replay sends the
// current canonical asset set once.
//
// # Reading Updates
//
//	for event := range wsClient.Events() {
//	    _ = event
//	}
package ws
