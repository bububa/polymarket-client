# gamma Package

Go client for the Polymarket Gamma market-data API.

**Default Host:** `https://gamma-api.polymarket.com`  
**Auth:** None (all endpoints are public)

## Creating a Client

```go
client := gamma.New(gamma.Config{})
```

## Endpoints

| Method | Description |
|---|---|
| `GetMarket` | Market by ID or slug (pre-allocation) |
| `GetMarketBySlug` | Market by slug (pre-allocation) |
| `GetMarkets` | Filtered market list |
| `ListMarketsKeyset` | Cursor-paginated market list (`GET /markets/keyset`) |
| `GetEvent` | Event by ID or slug (pre-allocation) |
| `GetEventBySlug` | Event by slug (pre-allocation) |
| `GetEvents` | Filtered event list |
| `ListEventsKeyset` | Cursor-paginated event list (`GET /events/keyset`) |
| `Search` | Full-text search across markets, events, series, tags |
| `ListSeries` / `GetSeries` | Series listing and detail |
| `GetTags` / `GetTag` / `GetTagBySlug` / `GetEventTags` | Tag management |
| `GetRelatedTags` / `GetRelatedTagRelationships` | Related tag discovery |
| `GetSports` / `GetValidSportsMarketTypes` | Sports market metadata |
| `GetTeams` | Sports teams |
| `GetComments` / `GetComment` / `GetCommentsByUserAddress` | Comments |
| `GetPublicProfile` | Public user profile by wallet address |

## Pre-Allocation Pattern

Single-entity getters accept an output pointer:

```go
// Get by ID
market := gamma.Market{ID: "123"}
err := client.GetMarket(ctx, &market)

// Get by slug (ID left empty)
market := gamma.Market{Slug: "election-2024"}
err := client.GetMarket(ctx, &market)
```

## Usage Example

```go
// Search across all content
var results gamma.SearchResults
err := client.Search(ctx, "election", &results)

// Get tags
tags, err := client.GetTags(ctx)

// Get public profile
var profile gamma.PublicProfile
profile.Address = "0xYourAddress"
err := client.GetPublicProfile(ctx, &profile)
```
