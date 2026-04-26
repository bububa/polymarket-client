package bridge

import (
	"context"
	"net/http"
	"time"

	"github.com/bububa/polymarket-client/internal/polyhttp"
)

const DefaultHost = "https://bridge.polymarket.com"

type Client struct {
	host string
	http *polyhttp.Client
}

// Config configures a Bridge API client.
type Config struct {
	Host       string
	HTTPClient *http.Client
	UserAgent  string
}

// New creates a Bridge API client.
func New(config Config) *Client {
	if config.Host == "" {
		config.Host = DefaultHost
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	if config.UserAgent == "" {
		config.UserAgent = "polymarket-client-go/bridge"
	}
	return &Client{
		host: config.Host,
		http: &polyhttp.Client{BaseURL: config.Host, HTTPClient: config.HTTPClient, UserAgent: config.UserAgent},
	}
}

// Host returns the configured Bridge API host.
func (c *Client) Host() string { return c.host }

// GetSupportedAssets lists all assets supported by the Polymarket bridge.
func (c *Client) GetSupportedAssets(ctx context.Context, out *SupportedAssetsResponse) error {
	return c.http.GetJSON(ctx, "/supported-assets", nil, polyhttp.AuthNone, out)
}

// CreateDepositAddress generates a deposit address for the given wallet address
// to bridge assets into the Polymarket ecosystem.
func (c *Client) CreateDepositAddress(ctx context.Context, address string, out *DepositResponse) error {
	return c.http.PostJSON(ctx, "/deposit", DepositRequest{Address: address}, polyhttp.AuthNone, out)
}

// GetStatus checks the current bridging status for a given deposit address.
func (c *Client) GetStatus(ctx context.Context, address string, out *StatusResponse) error {
	return c.http.GetJSON(ctx, "/status/"+address, nil, polyhttp.AuthNone, out)
}

// GetQuote returns an estimated quote for bridging a specified amount of an asset.
func (c *Client) GetQuote(ctx context.Context, req QuoteRequest, out *QuoteResponse) error {
	return c.http.PostJSON(ctx, "/quote", req, polyhttp.AuthNone, out)
}

// Withdraw initiates a withdrawal (bridge-out) request for the specified asset and amount.
func (c *Client) Withdraw(ctx context.Context, req WithdrawRequest, out *WithdrawResponse) error {
	return c.http.PostJSON(ctx, "/withdraw", req, polyhttp.AuthNone, out)
}
