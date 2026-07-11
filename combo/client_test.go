package combo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bububa/polymarket-client/clob"
)

func TestGetMarketsNormalizesBinaryOutcomes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/rfq/combo-markets" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("exclude") != "a,b" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"markets":[{"id":1897034,"condition_id":"0xabc","position_ids":["1","2"],"slug":"s","title":"t","outcomes":["Yes","No"],"outcome_prices":["0.685",0.315],"image":"i","volume":"12.5","tags":["sports"]}],"next_cursor":"Mg"}`))
	}))
	defer server.Close()
	var out MarketPage
	if err := NewClient(server.URL).GetMarkets(context.Background(), MarketParams{Limit: 50, Exclude: []string{"a", "b"}}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Markets[0].Yes.Price != "0.685" || out.Markets[0].No.PositionID != "2" {
		t.Fatalf("market = %+v", out.Markets[0])
	}
}

func TestSubmitQuoteUsesComboHMACHeaders(t *testing.T) {
	identity := Identity{SignerAddress: "0x1111111111111111111111111111111111111111", MakerAddress: "0x1111111111111111111111111111111111111111", SignatureType: clob.SignatureTypeEOA}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, name := range []string{"POLY_ADDRESS", "POLY_API_KEY", "POLY_PASSPHRASE", "POLY_TIMESTAMP", "POLY_SIGNATURE"} {
			if r.Header.Get(name) == "" {
				t.Fatalf("missing %s", name)
			}
		}
		var body SubmitQuoteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.QuoteID != "q" {
			t.Fatalf("body = %+v", body)
		}
		_, _ = w.Write([]byte(`{"rfq_id":"r","quote_id":"q","success":true}`))
	}))
	defer server.Close()
	client := NewClient(server.URL, WithCredentials(clob.Credentials{Key: "key", Secret: "c2VjcmV0", Passphrase: "pass"}), WithAuthAddress(identity.SignerAddress), WithIdentity(identity))
	var out QuoteResponse
	err := client.SubmitQuote(context.Background(), validSubmitQuoteRequest(identity), &out)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticatedAddressIsSeparateFromOrderIdentity(t *testing.T) {
	signer, err := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	if err != nil {
		t.Fatal(err)
	}
	wallet := "0x2222222222222222222222222222222222222222"
	tests := []struct {
		name     string
		sigType  clob.SignatureType
		identity Identity
	}{
		{name: "eoa", sigType: clob.SignatureTypeEOA, identity: Identity{SignerAddress: signer.Address().Hex(), MakerAddress: signer.Address().Hex(), SignatureType: clob.SignatureTypeEOA}},
		{name: "proxy", sigType: clob.SignatureTypeProxy, identity: Identity{SignerAddress: signer.Address().Hex(), MakerAddress: wallet, SignatureType: clob.SignatureTypeProxy}},
		{name: "safe", sigType: clob.SignatureTypeGnosisSafe, identity: Identity{SignerAddress: signer.Address().Hex(), MakerAddress: wallet, SignatureType: clob.SignatureTypeGnosisSafe}},
		{name: "poly1271", sigType: clob.SignatureTypePoly1271, identity: Identity{SignerAddress: wallet, MakerAddress: wallet, SignatureType: clob.SignatureTypePoly1271}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("POLY_ADDRESS"); !strings.EqualFold(got, signer.Address().Hex()) {
					t.Errorf("POLY_ADDRESS = %q, want %q", got, signer.Address().Hex())
				}
				var body SubmitQuoteRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Identity != tt.identity {
					t.Errorf("identity = %+v, want %+v", body.Identity, tt.identity)
				}
				_, _ = w.Write([]byte(`{"rfq_id":"r","quote_id":"q","success":true}`))
			}))
			defer server.Close()
			client := NewClient(server.URL, WithCredentials(clob.Credentials{Key: "key", Secret: "c2VjcmV0", Passphrase: "pass"}), WithSigner(signer), WithIdentity(tt.identity))
			if !strings.EqualFold(client.AuthAddress(), signer.Address().Hex()) {
				t.Fatalf("AuthAddress = %q", client.AuthAddress())
			}
			var out QuoteResponse
			err := client.SubmitQuote(context.Background(), validSubmitQuoteRequest(tt.identity), &out)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAuthenticatedAddressMustMatchSigner(t *testing.T) {
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	identity := Identity{SignerAddress: signer.Address().Hex(), MakerAddress: signer.Address().Hex(), SignatureType: clob.SignatureTypeEOA}
	transportCalled := false
	client := NewClient("https://example.invalid",
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { transportCalled = true; return nil, nil })}),
		WithCredentials(clob.Credentials{Key: "key", Secret: "c2VjcmV0", Passphrase: "pass"}),
		WithSigner(signer), WithAuthAddress("0x2222222222222222222222222222222222222222"), WithIdentity(identity),
	)
	var out QuoteResponse
	err := client.SubmitQuote(context.Background(), validSubmitQuoteRequest(identity), &out)
	if err == nil || !strings.Contains(err.Error(), "does not match signer") {
		t.Fatalf("error = %v", err)
	}
	if transportCalled {
		t.Fatal("request was sent")
	}
}

func TestRespondLastLookRejectsMissingIDsBeforeSending(t *testing.T) {
	identity := Identity{SignerAddress: "0x1111111111111111111111111111111111111111", MakerAddress: "0x1111111111111111111111111111111111111111", SignatureType: clob.SignatureTypeEOA}
	tests := []struct {
		name string
		req  LastLookRequest
	}{
		{name: "missing rfq id", req: LastLookRequest{QuoteID: "q1", Identity: identity, Decision: ConfirmationConfirm}},
		{name: "missing quote id", req: LastLookRequest{RFQID: "r1", Identity: identity, Decision: ConfirmationConfirm}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transportCalled := false
			client := NewClient("https://example.invalid",
				WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					transportCalled = true
					return nil, nil
				})}),
				WithIdentity(identity),
			)
			var out LastLookResponse
			err := client.RespondLastLook(context.Background(), tt.req, &out)
			if err == nil || !strings.Contains(err.Error(), "quote id and rfq id are required") {
				t.Fatalf("error = %v", err)
			}
			if transportCalled {
				t.Fatal("request was sent")
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestSubmitQuoteRejectsInvalidPayload(t *testing.T) {
	identity := Identity{SignerAddress: "0x1111111111111111111111111111111111111111", MakerAddress: "0x1111111111111111111111111111111111111111", SignatureType: clob.SignatureTypeEOA}
	tests := []struct {
		name   string
		mutate func(*SubmitQuoteRequest)
		want   string
	}{
		{name: "missing price", mutate: func(req *SubmitQuoteRequest) { req.PriceE6 = "" }, want: "price_e6"},
		{name: "zero size", mutate: func(req *SubmitQuoteRequest) { req.SizeE6 = "0" }, want: "size_e6"},
		{name: "identity mismatch", mutate: func(req *SubmitQuoteRequest) { req.SignedOrder.Maker = "0x2222222222222222222222222222222222222222" }, want: "identity does not match"},
		{name: "missing token", mutate: func(req *SubmitQuoteRequest) { req.SignedOrder.TokenID = "" }, want: "tokenId"},
		{name: "invalid side", mutate: func(req *SubmitQuoteRequest) { req.SignedOrder.Side = 2 }, want: "side must be 0 or 1"},
		{name: "invalid metadata", mutate: func(req *SubmitQuoteRequest) { req.SignedOrder.Metadata = "0x01" }, want: "metadata must be bytes32"},
		{name: "missing signature", mutate: func(req *SubmitQuoteRequest) { req.SignedOrder.Signature = "" }, want: "signature must be hex"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validSubmitQuoteRequest(identity)
			tt.mutate(&req)
			client := NewClient("https://example.invalid", WithIdentity(identity), WithAuthAddress(identity.SignerAddress), WithCredentials(clob.Credentials{Key: "key", Secret: "c2VjcmV0", Passphrase: "pass"}))
			var out QuoteResponse
			err := client.SubmitQuote(context.Background(), req, &out)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func validSubmitQuoteRequest(identity Identity) SubmitQuoteRequest {
	signature := "0x" + strings.Repeat("11", 65)
	if identity.SignatureType == clob.SignatureTypePoly1271 {
		signature = "0x" + strings.Repeat("11", 65+32+32+len(orderType)+2)
	}
	return SubmitQuoteRequest{
		QuoteID: "q", RFQID: "r", Identity: identity, PriceE6: "450000", SizeE6: "1000000",
		SignedOrder: SignedOrder{
			Salt: "1", Maker: identity.MakerAddress, Signer: identity.SignerAddress, TokenID: "11",
			MakerAmount: "450000", TakerAmount: "1000000", Side: 0, SignatureType: identity.SignatureType,
			Timestamp: "1780575184", Metadata: ZeroBytes32, Builder: ZeroBytes32, Signature: signature,
		},
	}
}
