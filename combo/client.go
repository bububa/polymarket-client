package combo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/bububa/polymarket-client/internal/polyauth"
	"github.com/bububa/polymarket-client/internal/polyhttp"
	pmtypes "github.com/bububa/polymarket-client/shared"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Client struct {
	host          string
	httpClient    *http.Client
	credentials   *clob.Credentials
	signer        *polyauth.Signer
	authAddress   string
	identity      Identity
	chainID       int64
	useServerTime bool
	userAgent     string
}

type Option func(*Client)

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}
func WithCredentials(credentials clob.Credentials) Option {
	return func(c *Client) { c.credentials = &credentials }
}
func WithSigner(signer *polyauth.Signer) Option { return func(c *Client) { c.signer = signer } }

// WithAuthAddress sets the EOA that owns the configured CLOB API credentials.
// It is distinct from the order identity for POLY_1271 deposit wallets.
func WithAuthAddress(address string) Option { return func(c *Client) { c.authAddress = address } }
func WithIdentity(identity Identity) Option { return func(c *Client) { c.identity = identity } }
func WithChainID(chainID int64) Option      { return func(c *Client) { c.chainID = chainID } }
func WithServerTime(enabled bool) Option    { return func(c *Client) { c.useServerTime = enabled } }

func NewClient(host string, opts ...Option) *Client {
	if host == "" {
		host = DefaultHost
	}
	c := &Client{host: strings.TrimRight(host, "/"), httpClient: polyhttp.NewDefaultHTTPClient(), chainID: clob.PolygonChainID, userAgent: "polymarket-client-go/combo"}
	for _, opt := range opts {
		opt(c)
	}
	if c.authAddress == "" && c.signer != nil {
		c.authAddress = c.signer.Address().Hex()
	}
	return c
}

func (c *Client) Host() string                   { return c.host }
func (c *Client) Identity() Identity             { return c.identity }
func (c *Client) Signer() *polyauth.Signer       { return c.signer }
func (c *Client) Credentials() *clob.Credentials { return c.credentials }
func (c *Client) AuthAddress() string            { return c.authAddress }
func (c *Client) ChainID() int64                 { return c.chainID }

func (c *Client) GetMarkets(ctx context.Context, params MarketParams, out *MarketPage) error {
	if out == nil {
		return errors.New("combo: output is required")
	}
	if params.Limit < 0 || params.Limit > 100 {
		return errors.New("combo: market limit must be between 1 and 100")
	}
	q := url.Values{}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}
	if len(params.Exclude) > 0 {
		q.Set("exclude", strings.Join(params.Exclude, ","))
	}
	return c.do(ctx, http.MethodGet, "/v1/rfq/combo-markets", q, nil, false, out)
}

func (c *Client) SubmitQuote(ctx context.Context, req SubmitQuoteRequest, out *QuoteResponse) error {
	if out == nil {
		return errors.New("combo: output is required")
	}
	if req.QuoteID == "" || req.RFQID == "" {
		return errors.New("combo: quote id and rfq id are required")
	}
	if err := c.validateIdentity(req.Identity); err != nil {
		return err
	}
	if c.signer != nil {
		if err := validateOrderIdentity(req.Identity, c.signer); err != nil {
			return err
		}
	}
	if err := validateSubmitQuote(req); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, "/v1/maker/quotes", nil, req, true, out)
}

func validateSubmitQuote(req SubmitQuoteRequest) error {
	price, err := parseUint256Field("price_e6", req.PriceE6, true)
	if err != nil {
		return err
	}
	if price.Cmp(big.NewInt(1_000_000)) >= 0 {
		return errors.New("combo: price_e6 must be less than 1000000")
	}
	if _, err := parseUint256Field("size_e6", req.SizeE6, true); err != nil {
		return err
	}
	order := req.SignedOrder
	if !strings.EqualFold(order.Maker, req.MakerAddress) || !strings.EqualFold(order.Signer, req.SignerAddress) || order.SignatureType != req.SignatureType {
		return errors.New("combo: signed order identity does not match request identity")
	}
	if !common.IsHexAddress(order.Maker) || !common.IsHexAddress(order.Signer) ||
		common.HexToAddress(order.Maker) == (common.Address{}) || common.HexToAddress(order.Signer) == (common.Address{}) {
		return errors.New("combo: signed order signer and maker must be valid non-zero addresses")
	}
	for name, value := range map[string]pmtypes.String{
		"tokenId": order.TokenID, "makerAmount": order.MakerAmount, "takerAmount": order.TakerAmount, "timestamp": order.Timestamp,
	} {
		if _, err := parseUint256Field(name, value, true); err != nil {
			return err
		}
	}
	if _, err := parseUint256Field("salt", order.Salt, false); err != nil {
		return err
	}
	if order.Expiration != "" {
		if _, err := parseUint256Field("expiration", order.Expiration, false); err != nil {
			return err
		}
	}
	if order.Side != 0 && order.Side != 1 {
		return errors.New("combo: signed order side must be 0 or 1")
	}
	if err := validateBytes32Field("metadata", order.Metadata); err != nil {
		return err
	}
	if err := validateBytes32Field("builder", order.Builder); err != nil {
		return err
	}
	signature, err := hexutil.Decode(order.Signature)
	if err != nil {
		return fmt.Errorf("combo: signed order signature must be hex: %w", err)
	}
	switch order.SignatureType {
	case clob.SignatureTypeEOA:
		if !strings.EqualFold(order.Signer, order.Maker) {
			return errors.New("combo: EOA signed order signer and maker must match")
		}
		if len(signature) != 65 {
			return errors.New("combo: signed order signature must be 65 bytes")
		}
	case clob.SignatureTypeProxy, clob.SignatureTypeGnosisSafe:
		if len(signature) != 65 {
			return errors.New("combo: signed order signature must be 65 bytes")
		}
	case clob.SignatureTypePoly1271:
		if !strings.EqualFold(order.Signer, order.Maker) {
			return errors.New("combo: POLY_1271 signed order signer and maker must match")
		}
		const wrapperFixedLength = 65 + 32 + 32 + 2
		if len(signature) != wrapperFixedLength+len(orderType) {
			return errors.New("combo: POLY_1271 signed order signature has invalid wrapper length")
		}
	default:
		return fmt.Errorf("combo: unsupported signed order signature type %d", order.SignatureType)
	}
	return nil
}

func parseUint256Field(name string, value pmtypes.String, positive bool) (*big.Int, error) {
	n, ok := new(big.Int).SetString(string(value), 10)
	if !ok || n.Sign() < 0 || n.BitLen() > 256 || (positive && n.Sign() == 0) {
		return nil, fmt.Errorf("combo: signed quote %s must be a valid uint256", name)
	}
	return n, nil
}

func validateBytes32Field(name, value string) error {
	decoded, err := hexutil.Decode(value)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("combo: signed order %s must be bytes32", name)
	}
	return nil
}

func (c *Client) CancelQuote(ctx context.Context, req CancelQuoteRequest, out *CancelQuoteResponse) error {
	if out == nil {
		return errors.New("combo: output is required")
	}
	if req.QuoteID == "" || req.RFQID == "" {
		return errors.New("combo: quote id and rfq id are required")
	}
	if err := c.validateIdentity(req.Identity); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, "/v1/maker/quotes/cancel", nil, req, true, out)
}

func (c *Client) RespondLastLook(ctx context.Context, req LastLookRequest, out *LastLookResponse) error {
	if out == nil {
		return errors.New("combo: output is required")
	}
	if req.QuoteID == "" || req.RFQID == "" {
		return errors.New("combo: quote id and rfq id are required")
	}
	if req.Decision != ConfirmationConfirm && req.Decision != ConfirmationDecline {
		return errors.New("combo: decision must be CONFIRM or DECLINE")
	}
	if err := c.validateIdentity(req.Identity); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, "/v1/maker/confirmations", nil, req, true, out)
}

func (c *Client) validateIdentity(identity Identity) error {
	if identity.SignerAddress == "" || identity.MakerAddress == "" {
		return errors.New("combo: signer and maker addresses are required")
	}
	if !common.IsHexAddress(identity.SignerAddress) || !common.IsHexAddress(identity.MakerAddress) {
		return errors.New("combo: signer and maker addresses must be valid evm addresses")
	}
	if !strings.EqualFold(identity.SignerAddress, c.identity.SignerAddress) || !strings.EqualFold(identity.MakerAddress, c.identity.MakerAddress) || identity.SignatureType != c.identity.SignatureType {
		return errors.New("combo: request identity does not match client identity")
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, authenticated bool, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("combo: marshal request: %w", err)
	}
	if body == nil {
		payload = nil
	}
	fullURL := c.host + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	var reader io.Reader
	if len(payload) > 0 {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return fmt.Errorf("combo: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		if c.credentials == nil {
			return errors.New("combo: api credentials are required")
		}
		if c.authAddress == "" {
			return errors.New("combo: authenticated address is required")
		}
		if !common.IsHexAddress(c.authAddress) {
			return errors.New("combo: authenticated address must be a valid evm address")
		}
		if c.signer != nil && !strings.EqualFold(c.authAddress, c.signer.Address().Hex()) {
			return errors.New("combo: authenticated address does not match signer")
		}
		ts := time.Now().Unix()
		if c.useServerTime {
			serverTS, err := c.serverTime(ctx)
			if err != nil {
				return err
			}
			ts = serverTS
		}
		sig, err := clob.BuildHMACSignature(c.credentials.Secret, ts, method, path, payload)
		if err != nil {
			return fmt.Errorf("combo: sign request: %w", err)
		}
		req.Header.Set("POLY_ADDRESS", c.authAddress)
		req.Header.Set("POLY_API_KEY", c.credentials.Key)
		req.Header.Set("POLY_PASSPHRASE", c.credentials.Passphrase)
		req.Header.Set("POLY_TIMESTAMP", strconv.FormatInt(ts, 10))
		req.Header.Set("POLY_SIGNATURE", sig)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("combo: perform request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("combo: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return &polyhttp.APIError{StatusCode: resp.StatusCode, Message: string(bytes.TrimSpace(data)), Body: data, RequestBody: payload}
	}
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("combo: decode response: %w", err)
	}
	return nil
}

func (c *Client) serverTime(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clob.MainnetHost+"/time", nil)
	if err != nil {
		return 0, fmt.Errorf("combo: create server-time request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("combo: fetch server time: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("combo: read server time: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return 0, fmt.Errorf("combo: fetch server time: status %d", resp.StatusCode)
	}
	value := strings.Trim(strings.TrimSpace(string(payload)), `"`)
	ts, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("combo: parse server time: %w", err)
	}
	return ts, nil
}
