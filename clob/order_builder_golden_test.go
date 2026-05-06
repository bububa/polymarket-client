package clob

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type goldenFile struct {
	Schema      string         `json:"schema"`
	GeneratedBy map[string]any `json:"generatedBy"`
	Vectors     []goldenVector `json:"vectors"`

	// Single-vector files.
	Name     string         `json:"name"`
	Kind     string         `json:"kind"`
	Input    goldenInput    `json:"input"`
	Expected goldenExpected `json:"expected"`
}

type goldenVector struct {
	Name        string         `json:"name"`
	Kind        string         `json:"kind"`
	GeneratedBy map[string]any `json:"generatedBy"`
	Input       goldenInput    `json:"input"`
	Expected    goldenExpected `json:"expected"`
}

type goldenInput struct {
	PrivateKey    string  `json:"privateKey"`
	ChainID       int64   `json:"chainId"`
	TokenID       string  `json:"tokenId"`
	SignatureType int     `json:"signatureType"`
	Funder        string  `json:"funder"`
	Timestamp     int64   `json:"timestamp"`
	Owner         string  `json:"owner"`
	TickSize      string  `json:"tickSize"`
	NegRisk       bool    `json:"negRisk"`
	Side          string  `json:"side"`
	Price         float64 `json:"price"`
	Size          float64 `json:"size"`
	Amount        float64 `json:"amount"`
	Expiration    int64   `json:"expiration"`
	OrderType     string  `json:"orderType"`
}

type goldenExpected struct {
	Signer           string         `json:"signer"`
	SignedOrder      goldenOrder    `json:"signedOrder"`
	PostOrderRequest map[string]any `json:"postOrderRequest"`
}

type goldenOrder struct {
	Salt          string `json:"salt"`
	Maker         string `json:"maker"`
	Signer        string `json:"signer"`
	TokenID       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          int    `json:"side"`
	SignatureType int    `json:"signatureType"`
	Timestamp     string `json:"timestamp"`
	Metadata      string `json:"metadata"`
	Builder       string `json:"builder"`
	Expiration    string `json:"expiration"`
	Signature     string `json:"signature"`
}

// TestOrderBuilderV2_GoldenLimitOrderSignaturesAndPostOrderJSON verifies the
// byte-for-byte important CLOB v2 limit-order fields against py-clob-client-v2.
//
// Market orders are intentionally excluded here because this Go SDK keeps a
// more conservative market-order amount calculation than py-clob-client-v2.
// That difference changes maker/taker amounts and therefore the signature.
func TestOrderBuilderV2_GoldenLimitOrderSignaturesAndPostOrderJSON(t *testing.T) {
	for _, vector := range loadGoldenVectors(t) {
		if vector.Kind == "clob_order_v2_market" {
			continue
		}

		t.Run(vector.Name, func(t *testing.T) {
			signer, err := ParsePrivateKey(vector.Input.PrivateKey)
			if err != nil {
				t.Fatalf("ParsePrivateKey: %v", err)
			}

			expected := vector.Expected.SignedOrder
			order := signedOrderFromGolden(t, expected)

			if err := SignOrder(signer, vector.Input.ChainID, order, WithSignOrderNegRisk(vector.Input.NegRisk)); err != nil {
				t.Fatalf("SignOrder: %v", err)
			}

			assertSignedOrderMatchesGolden(t, expected, order)

			deferExec := boolFromMapDefault(vector.Expected.PostOrderRequest, "deferExec", false)
			actualPost := PostOrderRequest{
				Order:     *order,
				Owner:     stringFromMap(t, vector.Expected.PostOrderRequest, "owner"),
				OrderType: OrderType(stringFromMap(t, vector.Expected.PostOrderRequest, "orderType")),
				DeferExec: &deferExec,
			}

			assertPostOrderRequestEqual(t, vector.Expected.PostOrderRequest, actualPost)
			assertPostOrderSaltIsJSONNumber(t, actualPost)
		})
	}
}

// TestOrderBuilderV2_GoldenBuildLimitOrderAmountsAndDefaults validates that
// high-level BuildOrder matches py-clob-client-v2 for limit-order amount
// calculation and default fields.
//
// It does not compare salt/timestamp/signature because BuildOrder generates
// those at runtime.
func TestOrderBuilderV2_GoldenBuildLimitOrderAmountsAndDefaults(t *testing.T) {
	for _, vector := range loadGoldenVectors(t) {
		if vector.Kind == "clob_order_v2_market" {
			continue
		}

		t.Run(vector.Name, func(t *testing.T) {
			signer, err := ParsePrivateKey(vector.Input.PrivateKey)
			if err != nil {
				t.Fatalf("ParsePrivateKey: %v", err)
			}

			sigType := SignatureType(vector.Input.SignatureType)
			client := NewClient(
				"",
				WithSigner(signer),
				WithChainID(vector.Input.ChainID),
				WithDefaultSignatureType(sigType),
			)
			builder := NewOrderBuilder(client)

			order, err := builder.BuildOrder(OrderArgsV2{
				TokenID:       vector.Input.TokenID,
				Price:         formatFloatForTest(vector.Input.Price),
				Size:          formatFloatForTest(vector.Input.Size),
				Side:          Side(vector.Input.Side),
				Expiration:    strconv.FormatInt(vector.Input.Expiration, 10),
				SignatureType: &sigType,
				Maker:         vector.Input.Funder,
			}, CreateOrderOptions{
				TickSize: vector.Input.TickSize,
				NegRisk:  vector.Input.NegRisk,
			})
			if err != nil {
				t.Fatalf("BuildOrder: %v", err)
			}

			expected := vector.Expected.SignedOrder

			assertEqualFold(t, "maker", expected.Maker, order.Maker)
			assertEqualFold(t, "signer", expected.Signer, order.Signer)
			assertEqual(t, "tokenId", expected.TokenID, order.TokenID.String())
			assertEqual(t, "makerAmount", expected.MakerAmount, order.MakerAmount.String())
			assertEqual(t, "takerAmount", expected.TakerAmount, order.TakerAmount.String())
			assertEqual(t, "metadata", expected.Metadata, order.Metadata)
			assertEqual(t, "builder", expected.Builder, order.Builder)
			assertEqual(t, "expiration", expected.Expiration, order.Expiration.String())
			assertEqual(t, "signatureType", expected.SignatureType, int(order.SignatureType))

			expectedSide := Buy
			if expected.Side == 1 {
				expectedSide = Sell
			}
			assertEqual(t, "side", expectedSide, order.Side)

			if order.Salt <= 0 {
				t.Fatalf("salt = %d, want positive generated salt", order.Salt)
			}
			if order.Timestamp.String() == "" {
				t.Fatalf("timestamp is empty")
			}
			if order.Signature == "" {
				t.Fatalf("signature is empty")
			}
		})
	}
}

// TestBuildMarketOrder_IntentionallyDivergesFromPyClobClientV2 documents the
// deliberate market-order rounding difference.
//
// py-clob-client-v2 and this Go SDK can produce different market-order
// maker/taker amounts. That also means the final EIP-712 signature differs.
// This test keeps the difference visible while still validating stable fields.
func TestBuildMarketOrder_IntentionallyDivergesFromPyClobClientV2(t *testing.T) {
	found := false

	for _, vector := range loadGoldenVectors(t) {
		if vector.Kind != "clob_order_v2_market" {
			continue
		}
		found = true

		t.Run(vector.Name, func(t *testing.T) {
			signer, err := ParsePrivateKey(vector.Input.PrivateKey)
			if err != nil {
				t.Fatalf("ParsePrivateKey: %v", err)
			}

			sigType := SignatureType(vector.Input.SignatureType)
			client := NewClient(
				"",
				WithSigner(signer),
				WithChainID(vector.Input.ChainID),
				WithDefaultSignatureType(sigType),
			)
			builder := NewOrderBuilder(client)

			order, err := builder.BuildMarketOrder(MarketOrderArgsV2{
				TokenID:       vector.Input.TokenID,
				Price:         formatFloatForTest(vector.Input.Price),
				Amount:        formatFloatForTest(vector.Input.Amount),
				Side:          Side(vector.Input.Side),
				SignatureType: &sigType,
				Maker:         vector.Input.Funder,
			}, CreateOrderOptions{
				TickSize: vector.Input.TickSize,
				NegRisk:  vector.Input.NegRisk,
			})
			if err != nil {
				t.Fatalf("BuildMarketOrder: %v", err)
			}

			expected := vector.Expected.SignedOrder

			assertEqualFold(t, "maker", expected.Maker, order.Maker)
			assertEqualFold(t, "signer", expected.Signer, order.Signer)
			assertEqual(t, "tokenId", expected.TokenID, order.TokenID.String())
			assertEqual(t, "metadata", expected.Metadata, order.Metadata)
			assertEqual(t, "builder", expected.Builder, order.Builder)
			assertEqual(t, "expiration", expected.Expiration, order.Expiration.String())
			assertEqual(t, "signatureType", expected.SignatureType, int(order.SignatureType))

			expectedSide := Buy
			if expected.Side == 1 {
				expectedSide = Sell
			}
			assertEqual(t, "side", expectedSide, order.Side)

			if order.MakerAmount.String() == expected.MakerAmount &&
				order.TakerAmount.String() == expected.TakerAmount {
				t.Fatalf("%s no longer diverges from py-clob-client-v2; consider enabling strict market-order golden comparison", vector.Name)
			}

			t.Logf(
				"%s intentionally diverges from py-clob-client-v2 market amounts: official maker/taker=%s/%s, go maker/taker=%s/%s",
				vector.Name,
				expected.MakerAmount,
				expected.TakerAmount,
				order.MakerAmount.String(),
				order.TakerAmount.String(),
			)
		})
	}

	if !found {
		t.Fatal("no market order golden vectors found")
	}
}

func loadGoldenVectors(t *testing.T) []goldenVector {
	t.Helper()

	dir := filepath.Join("..", "testdata", "golden", "py-clob-client-v2")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read golden dir %s: %v", dir, err)
	}

	var vectors []goldenVector
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		// Deposit-wallet POLY_1271 vectors have a different schema and are tested
		// by deposit_wallet_signing_test.go.
		if entry.Name() == "clob_order_v2_deposit_wallet.json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read golden file %s: %v", path, err)
		}

		var file goldenFile
		if err := json.Unmarshal(data, &file); err != nil {
			t.Fatalf("decode golden file %s: %v", path, err)
		}

		if len(file.Vectors) > 0 {
			for _, v := range file.Vectors {
				if v.Name == "" {
					t.Fatalf("golden file %s contains vector without name", path)
				}
				vectors = append(vectors, v)
			}
			continue
		}

		if file.Name == "" {
			t.Fatalf("golden file %s contains neither vectors nor single vector name", path)
		}

		vectors = append(vectors, goldenVector{
			Name:        file.Name,
			Kind:        file.Kind,
			GeneratedBy: file.GeneratedBy,
			Input:       file.Input,
			Expected:    file.Expected,
		})
	}

	if len(vectors) == 0 {
		t.Fatalf("no golden vectors found in %s", dir)
	}
	return vectors
}

func signedOrderFromGolden(t *testing.T, expected goldenOrder) *SignedOrder {
	t.Helper()

	salt, err := strconv.ParseInt(expected.Salt, 10, 64)
	if err != nil {
		t.Fatalf("parse salt %q: %v", expected.Salt, err)
	}

	side := Buy
	if expected.Side == 1 {
		side = Sell
	}

	return &SignedOrder{
		Salt:          Int64(salt),
		Maker:         expected.Maker,
		Signer:        expected.Signer,
		TokenID:       String(expected.TokenID),
		MakerAmount:   String(expected.MakerAmount),
		TakerAmount:   String(expected.TakerAmount),
		Side:          side,
		SignatureType: SignatureType(expected.SignatureType),
		Timestamp:     String(expected.Timestamp),
		Metadata:      expected.Metadata,
		Builder:       expected.Builder,
		Expiration:    String(expected.Expiration),
	}
}

func assertSignedOrderMatchesGolden(t *testing.T, expected goldenOrder, order *SignedOrder) {
	t.Helper()

	assertEqualFold(t, "signature", expected.Signature, order.Signature)
	assertEqualFold(t, "maker", expected.Maker, order.Maker)
	assertEqualFold(t, "signer", expected.Signer, order.Signer)
	assertEqual(t, "tokenId", expected.TokenID, order.TokenID.String())
	assertEqual(t, "makerAmount", expected.MakerAmount, order.MakerAmount.String())
	assertEqual(t, "takerAmount", expected.TakerAmount, order.TakerAmount.String())
	assertEqual(t, "timestamp", expected.Timestamp, order.Timestamp.String())
	assertEqual(t, "metadata", expected.Metadata, order.Metadata)
	assertEqual(t, "builder", expected.Builder, order.Builder)
	assertEqual(t, "expiration", expected.Expiration, order.Expiration.String())
	assertEqual(t, "signatureType", expected.SignatureType, int(order.SignatureType))

	expectedSide := Buy
	if expected.Side == 1 {
		expectedSide = Sell
	}
	assertEqual(t, "side", expectedSide, order.Side)
}

func canonicalPostOrderRequest(t *testing.T, v any) map[string]any {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal post order request: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal post order request: %v", err)
	}

	// py-clob-client-v2 includes postOnly as a top-level alias for deferExec.
	// This Go SDK currently uses deferExec only. postOnly does not affect the
	// signed order payload, so canonical golden comparison ignores it.
	delete(m, "postOnly")

	return m
}

func assertPostOrderRequestEqual(t *testing.T, expected any, actual any) {
	t.Helper()

	expectedCanonical := canonicalPostOrderRequest(t, expected)
	actualCanonical := canonicalPostOrderRequest(t, actual)

	if !reflect.DeepEqual(expectedCanonical, actualCanonical) {
		expectedPretty, _ := json.MarshalIndent(expectedCanonical, "", "  ")
		actualPretty, _ := json.MarshalIndent(actualCanonical, "", "  ")
		t.Fatalf("postOrderRequest mismatch\nexpected:\n%s\nactual:\n%s", expectedPretty, actualPretty)
	}
}

func assertPostOrderSaltIsJSONNumber(t *testing.T, req PostOrderRequest) {
	t.Helper()

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal post order request: %v", err)
	}

	var raw struct {
		Order map[string]json.RawMessage `json:"order"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal post order request: %v", err)
	}

	salt := strings.TrimSpace(string(raw.Order["salt"]))
	if salt == "" {
		t.Fatalf("order.salt missing in JSON: %s", data)
	}
	if strings.HasPrefix(salt, `"`) {
		t.Fatalf("order.salt must be a JSON number, got %s in %s", salt, data)
	}
}

func boolFromMapDefault(m map[string]any, key string, fallback bool) bool {
	v, ok := m[key]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}

func stringFromMap(t *testing.T, m map[string]any, key string) string {
	t.Helper()

	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in map", key)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("key %q = %T(%v), want string", key, v, v)
	}
	return s
}

func formatFloatForTest(v float64) string {
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func assertEqual[T comparable](t *testing.T, name string, expected, actual T) {
	t.Helper()

	if expected != actual {
		t.Fatalf("%s mismatch: expected %v, got %v", name, expected, actual)
	}
}

func assertEqualFold(t *testing.T, name string, expected, actual string) {
	t.Helper()

	if !strings.EqualFold(expected, actual) {
		t.Fatalf("%s mismatch: expected %s, got %s", name, expected, actual)
	}
}
