package gamma

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bububa/polymarket-client/internal/polyhttp"
)

var errMissingIdentifier = errors.New("polymarket: missing identifier on output value")

const DefaultHost = "https://gamma-api.polymarket.com"

type Client struct {
	host string
	http *polyhttp.Client
}

// Config configures a Gamma API client.
type Config struct {
	Host       string
	HTTPClient *http.Client
	UserAgent  string
}

// New creates a Gamma API client.
func New(config Config) *Client {
	if config.Host == "" {
		config.Host = DefaultHost
	}
	if config.HTTPClient == nil {
		config.HTTPClient = polyhttp.NewDefaultHTTPClient()
	}
	if config.UserAgent == "" {
		config.UserAgent = "polymarket-client-go/gamma"
	}
	return &Client{
		host: config.Host,
		http: &polyhttp.Client{BaseURL: config.Host, HTTPClient: config.HTTPClient, UserAgent: config.UserAgent},
	}
}

// Host returns the configured Gamma API host.
func (c *Client) Host() string { return c.host }

// GetMarket writes a market into out using out.ID, or out.Slug when ID is empty.
func (c *Client) GetMarket(ctx context.Context, out *Market) error {
	if out == nil {
		return errMissingIdentifier
	}
	if out.ID != 0 {
		return c.http.GetJSON(ctx, "/markets/"+strconv.Itoa(int(out.ID)), nil, polyhttp.AuthNone, out)
	}
	if out.Slug != "" {
		return c.GetMarketBySlug(ctx, out)
	}
	return errMissingIdentifier
}

// GetMarketBySlug writes a market by out.Slug into out.
func (c *Client) GetMarketBySlug(ctx context.Context, out *Market) error {
	if out == nil || out.Slug == "" {
		return errMissingIdentifier
	}
	return c.http.GetJSON(ctx, "/markets/slug/"+out.Slug, nil, polyhttp.AuthNone, out)
}

// GetMarkets returns markets matching params.
func (c *Client) GetMarkets(ctx context.Context, params MarketFilterParams) ([]Market, error) {
	var out []Market
	return out, c.http.GetJSON(ctx, "/markets", filterValues(params), polyhttp.AuthNone, &out)
}

// ListMarketsKeyset returns markets using cursor-based keyset pagination.
func (c *Client) ListMarketsKeyset(ctx context.Context, params MarketKeysetParams) (MarketKeysetPage, error) {
	var out MarketKeysetPage
	return out, c.http.GetJSON(ctx, "/markets/keyset", filterValues(params), polyhttp.AuthNone, &out)
}

// GetEvent writes an event into out using out.ID, or out.Slug when ID is empty.
func (c *Client) GetEvent(ctx context.Context, out *Event) error {
	if out == nil {
		return errMissingIdentifier
	}
	if out.ID != 0 {
		return c.http.GetJSON(ctx, "/events/"+strconv.Itoa(int(out.ID)), nil, polyhttp.AuthNone, out)
	}
	if out.Slug != "" {
		return c.GetEventBySlug(ctx, out)
	}
	return errMissingIdentifier
}

// GetEventBySlug writes an event by out.Slug into out.
func (c *Client) GetEventBySlug(ctx context.Context, out *Event) error {
	if out == nil || out.Slug == "" {
		return errMissingIdentifier
	}
	return c.http.GetJSON(ctx, "/events/slug/"+out.Slug, nil, polyhttp.AuthNone, out)
}

// GetEvents returns events matching params.
func (c *Client) GetEvents(ctx context.Context, params EventFilterParams) ([]Event, error) {
	var out []Event
	return out, c.http.GetJSON(ctx, "/events", filterValues(params), polyhttp.AuthNone, &out)
}

// ListEventsKeyset returns events using cursor-based keyset pagination.
func (c *Client) ListEventsKeyset(ctx context.Context, params EventKeysetParams) (EventKeysetPage, error) {
	var out EventKeysetPage
	return out, c.http.GetJSON(ctx, "/events/keyset", filterValues(params), polyhttp.AuthNone, &out)
}

// Search writes market, event, and public-profile matches into out.
func (c *Client) Search(ctx context.Context, query string, out *SearchResults) error {
	return c.http.GetJSON(ctx, "/public-search", url.Values{"q": []string{query}}, polyhttp.AuthNone, out)
}

// PublicSearch writes market, event, and public-profile matches into out.
func (c *Client) PublicSearch(ctx context.Context, query string, out *SearchResults) error {
	return c.Search(ctx, query, out)
}

// ListSeries returns series matching params.
func (c *Client) ListSeries(ctx context.Context, params SeriesFilterParams) ([]Series, error) {
	var out []Series
	return out, c.http.GetJSON(ctx, "/series", filterValues(params), polyhttp.AuthNone, &out)
}

// GetSeries writes a series by out.ID into out.
func (c *Client) GetSeries(ctx context.Context, out *Series) error {
	if out == nil || out.ID == 0 {
		return errMissingIdentifier
	}
	return c.http.GetJSON(ctx, "/series/"+strconv.Itoa(int(out.ID)), nil, polyhttp.AuthNone, out)
}

// GetTags returns all tags.
func (c *Client) GetTags(ctx context.Context) ([]Tag, error) {
	var out []Tag
	return out, c.http.GetJSON(ctx, "/tags", nil, polyhttp.AuthNone, &out)
}

// GetEventTags returns tags attached to an event.
func (c *Client) GetEventTags(ctx context.Context, eventID int) ([]Tag, error) {
	if eventID == 0 {
		return nil, errMissingIdentifier
	}
	var out []Tag
	return out, c.http.GetJSON(ctx, "/events/"+strconv.Itoa(eventID)+"/tags", nil, polyhttp.AuthNone, &out)
}

// GetTag writes a tag into out using out.ID, or out.Slug when ID is empty.
func (c *Client) GetTag(ctx context.Context, out *Tag) error {
	if out == nil {
		return errMissingIdentifier
	}
	if out.ID != 0 {
		return c.http.GetJSON(ctx, "/tags/"+strconv.Itoa(int(out.ID)), nil, polyhttp.AuthNone, out)
	}
	if out.Slug != "" {
		return c.GetTagBySlug(ctx, out)
	}
	return errMissingIdentifier
}

// GetTagBySlug writes a tag by out.Slug into out.
func (c *Client) GetTagBySlug(ctx context.Context, out *Tag) error {
	if out == nil || out.Slug == "" {
		return errMissingIdentifier
	}
	return c.http.GetJSON(ctx, "/tags/slug/"+out.Slug, nil, polyhttp.AuthNone, out)
}

// GetRelatedTagRelationships returns ranked related-tag relationships for a tag ID.
func (c *Client) GetRelatedTagRelationships(ctx context.Context, id int, params RelatedTagParams) ([]TagRelationship, error) {
	var out []TagRelationship
	return out, c.http.GetJSON(ctx, "/tags/"+strconv.Itoa(id)+"/related-tags", filterValues(params), polyhttp.AuthNone, &out)
}

// GetRelatedTagRelationshipsBySlug returns ranked related-tag relationships for a tag slug.
func (c *Client) GetRelatedTagRelationshipsBySlug(ctx context.Context, slug string, params RelatedTagParams) ([]TagRelationship, error) {
	var out []TagRelationship
	return out, c.http.GetJSON(ctx, "/tags/slug/"+slug+"/related-tags", filterValues(params), polyhttp.AuthNone, &out)
}

// GetRelatedTags returns related Tag records for a tag ID.
func (c *Client) GetRelatedTags(ctx context.Context, id int, params RelatedTagParams) ([]Tag, error) {
	var out []Tag
	return out, c.http.GetJSON(ctx, "/tags/"+strconv.Itoa(id)+"/related-tags/tags", filterValues(params), polyhttp.AuthNone, &out)
}

// GetRelatedTagsBySlug returns related Tag records for a tag slug.
func (c *Client) GetRelatedTagsBySlug(ctx context.Context, slug string, params RelatedTagParams) ([]Tag, error) {
	var out []Tag
	return out, c.http.GetJSON(ctx, "/tags/slug/"+slug+"/related-tags/tags", filterValues(params), polyhttp.AuthNone, &out)
}

// GetSports returns sports metadata.
func (c *Client) GetSports(ctx context.Context) ([]SportsMetadata, error) {
	var out []SportsMetadata
	return out, c.http.GetJSON(ctx, "/sports", nil, polyhttp.AuthNone, &out)
}

// GetValidSportsMarketTypes writes valid sports market types into out.
func (c *Client) GetValidSportsMarketTypes(ctx context.Context, out *SportsMarketTypesResponse) error {
	return c.http.GetJSON(ctx, "/sports/market-types", nil, polyhttp.AuthNone, out)
}

// GetTeams returns sports teams.
func (c *Client) GetTeams(ctx context.Context) ([]Team, error) {
	var out []Team
	return out, c.http.GetJSON(ctx, "/teams", nil, polyhttp.AuthNone, &out)
}

// GetComments returns comments matching params.
func (c *Client) GetComments(ctx context.Context, params CommentFilterParams) ([]Comment, error) {
	var out []Comment
	return out, c.http.GetJSON(ctx, "/comments", filterValues(params), polyhttp.AuthNone, &out)
}

// GetComment returns comments for a comment ID.
func (c *Client) GetComment(ctx context.Context, id int) ([]Comment, error) {
	var out []Comment
	return out, c.http.GetJSON(ctx, "/comments/"+strconv.Itoa(id), nil, polyhttp.AuthNone, &out)
}

// GetCommentsByUserAddress returns comments for a user wallet address.
func (c *Client) GetCommentsByUserAddress(ctx context.Context, address string) ([]Comment, error) {
	var out []Comment
	return out, c.http.GetJSON(ctx, "/comments/user/"+address, nil, polyhttp.AuthNone, &out)
}

// GetPublicProfile writes a public profile by out.Address into out.
func (c *Client) GetPublicProfile(ctx context.Context, out *PublicProfile) error {
	if out == nil || out.Address == "" {
		return errMissingIdentifier
	}
	return c.http.GetJSON(ctx, "/public-profile/"+out.Address, nil, polyhttp.AuthNone, out)
}

func filterValues(params interface{ appendQuery(url.Values) }) url.Values {
	q := url.Values{}
	params.appendQuery(q)
	return q
}

func setBool(q url.Values, key string, val *bool) {
	if val != nil {
		q.Set(key, strconv.FormatBool(*val))
	}
}

func setInt(q url.Values, key string, val int) {
	if val > 0 {
		q.Set(key, strconv.Itoa(val))
	}
}

func setIntSlice(q url.Values, key string, vals []int) {
	for _, val := range vals {
		q.Add(key, strconv.Itoa(val))
	}
}

func setString(q url.Values, key, val string) {
	if val != "" {
		q.Set(key, val)
	}
}

func setStringSlice(q url.Values, key string, vals []string) {
	for _, val := range vals {
		if val != "" {
			q.Add(key, val)
		}
	}
}

func setFloat(q url.Values, key string, val *float64) {
	if val != nil {
		q.Set(key, strconv.FormatFloat(*val, 'f', -1, 64))
	}
}
