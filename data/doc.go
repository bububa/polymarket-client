// Package data provides a read-only client for the Polymarket Data API.
//
// # Overview
//
// The Data API exposes aggregated market analytics — positions, trades, activity,
// open interest, leaderboard rankings, and accounting snapshots. No authentication
// required; all endpoints use AuthNone.
//
// # Usage
//
//	client := data.New(data.Config{})  // defaults to data-api.polymarket.com
//
//	// User positions
//	positions, _ := client.GetPositions(ctx, data.PositionParams{User: "0x..."})
//
//	// Trades
//	trades, _ := client.GetTrades(ctx, data.TradeParams{User: "0x...", Limit: 100})
//
//	// Open interest
//	oi, _ := client.GetOpenInterest(ctx, []string{"market1", "market2"})
//
//	// Combo positions and lifecycle activity
//	comboPositions, _ := client.GetComboPositions(ctx, data.ComboPositionParams{User: "0x..."})
//	comboActivity, _ := client.GetComboActivity(ctx, data.ComboActivityParams{User: "0x..."})
//
//	// Download accounting snapshot (ZIP)
//	zipData, _ := client.DownloadAccountingSnapshot(ctx, "0x...")
//
// Default host: https://data-api.polymarket.com
package data
