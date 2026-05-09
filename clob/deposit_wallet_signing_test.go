package clob

import (
	"encoding/binary"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const depositWalletTestPrivateKey = "0x59c6995e998f97a5a0044966f094538092e1db9e7b9c0e5a4e9e4e9e4e9e4e9e"

var (
	depositWalletTestAddress = common.HexToAddress("0x1111111111111111111111111111111111111111")
	depositWalletTestTokenID = "4738542302108129612856912335517660352849664845664685440963190764720214313804"
)

func TestSignDepositWalletOrder_FillsPoly1271Shape(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	order := depositWalletTestOrder()
	err = SignDepositWalletOrder(
		signer,
		PolygonChainID,
		order,
		depositWalletTestAddress,
		WithSignOrderSalt(big.NewInt(12345)),
		WithSignOrderTime(time.UnixMilli(1700000000000)),
	)
	if err != nil {
		t.Fatalf("SignDepositWalletOrder: %v", err)
	}

	if order.SignatureType != SignatureTypePoly1271 {
		t.Fatalf("signatureType = %d, want %d", order.SignatureType, SignatureTypePoly1271)
	}
	if !strings.EqualFold(order.Maker, depositWalletTestAddress.Hex()) {
		t.Fatalf("maker = %s, want %s", order.Maker, depositWalletTestAddress.Hex())
	}
	if !strings.EqualFold(order.Signer, depositWalletTestAddress.Hex()) {
		t.Fatalf("signer = %s, want %s", order.Signer, depositWalletTestAddress.Hex())
	}
	if order.Salt != Int64(12345) {
		t.Fatalf("salt = %d, want 12345", order.Salt)
	}
	if order.Timestamp.String() != "1700000000000" {
		t.Fatalf("timestamp = %s, want 1700000000000", order.Timestamp.String())
	}
	if order.Metadata != ZeroBytes32 {
		t.Fatalf("metadata = %s, want %s", order.Metadata, ZeroBytes32)
	}
	if order.Builder != ZeroBytes32 {
		t.Fatalf("builder = %s, want %s", order.Builder, ZeroBytes32)
	}
	if order.Expiration.String() != "0" {
		t.Fatalf("expiration = %s, want 0", order.Expiration.String())
	}
	if order.Signature == "" {
		t.Fatal("signature is empty")
	}

	assertDepositWalletWrappedSignatureShape(t, order)
}

func TestSignDepositWalletOrder_DeterministicWithFixedSaltAndTime(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	orderA := depositWalletTestOrder()
	orderB := depositWalletTestOrder()

	opts := []SignOrderOption{
		WithSignOrderSalt(big.NewInt(12345)),
		WithSignOrderTime(time.UnixMilli(1700000000000)),
	}

	if err := SignDepositWalletOrder(signer, PolygonChainID, orderA, depositWalletTestAddress, opts...); err != nil {
		t.Fatalf("SignDepositWalletOrder A: %v", err)
	}
	if err := SignDepositWalletOrder(signer, PolygonChainID, orderB, depositWalletTestAddress, opts...); err != nil {
		t.Fatalf("SignDepositWalletOrder B: %v", err)
	}

	if orderA.Signature != orderB.Signature {
		t.Fatalf("signature mismatch with fixed salt/time:\nA=%s\nB=%s", orderA.Signature, orderB.Signature)
	}
}

func TestSignDepositWalletOrder_WrappedSignatureContainsExpectedHashes(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	order := depositWalletTestOrder()
	if err := SignDepositWalletOrder(
		signer,
		PolygonChainID,
		order,
		depositWalletTestAddress,
		WithSignOrderSalt(big.NewInt(12345)),
		WithSignOrderTime(time.UnixMilli(1700000000000)),
	); err != nil {
		t.Fatalf("SignDepositWalletOrder: %v", err)
	}

	raw, err := hexutil.Decode(order.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	wantLen := 65 + 32 + 32 + len(depositWalletOrderTypeString) + 2
	if len(raw) != wantLen {
		t.Fatalf("wrapped signature length = %d, want %d", len(raw), wantLen)
	}

	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	wantDomainSeparator, err := ctfExchangeV2DomainSeparator(PolygonChainID, contracts.Exchange)
	if err != nil {
		t.Fatalf("ctfExchangeV2DomainSeparator: %v", err)
	}
	gotDomainSeparator := common.BytesToHash(raw[65 : 65+32])
	if gotDomainSeparator != wantDomainSeparator {
		t.Fatalf("domain separator mismatch: got %s, want %s", gotDomainSeparator.Hex(), wantDomainSeparator.Hex())
	}

	wantContentsHash, err := depositWalletOrderContentsHash(*order)
	if err != nil {
		t.Fatalf("depositWalletOrderContentsHash: %v", err)
	}
	gotContentsHash := common.BytesToHash(raw[65+32 : 65+32+32])
	if gotContentsHash != wantContentsHash {
		t.Fatalf("contents hash mismatch: got %s, want %s", gotContentsHash.Hex(), wantContentsHash.Hex())
	}

	typeStart := 65 + 32 + 32
	typeEnd := typeStart + len(depositWalletOrderTypeString)
	if string(raw[typeStart:typeEnd]) != depositWalletOrderTypeString {
		t.Fatalf("contents type mismatch: got %q, want %q", string(raw[typeStart:typeEnd]), depositWalletOrderTypeString)
	}

	gotTypeLen := binary.BigEndian.Uint16(raw[typeEnd : typeEnd+2])
	if int(gotTypeLen) != len(depositWalletOrderTypeString) {
		t.Fatalf("contents type length = %d, want %d", gotTypeLen, len(depositWalletOrderTypeString))
	}
}

func TestSignDepositWalletOrder_RejectsWrongSignatureType(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	order := depositWalletTestOrder()
	order.SignatureType = SignatureTypeProxy

	err = SignDepositWalletOrder(signer, PolygonChainID, order, depositWalletTestAddress)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "deposit wallet orders require signatureType") {
		t.Fatalf("error = %q, want signatureType error", err.Error())
	}
}

func TestSignDepositWalletOrder_RejectsMakerMismatch(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	order := depositWalletTestOrder()
	order.Maker = "0x2222222222222222222222222222222222222222"

	err = SignDepositWalletOrder(signer, PolygonChainID, order, depositWalletTestAddress)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "maker") || !strings.Contains(err.Error(), "does not match deposit wallet") {
		t.Fatalf("error = %q, want maker mismatch error", err.Error())
	}
}

func TestSignDepositWalletOrder_RejectsSignerMismatch(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	order := depositWalletTestOrder()
	order.Signer = "0x2222222222222222222222222222222222222222"

	err = SignDepositWalletOrder(signer, PolygonChainID, order, depositWalletTestAddress)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "signer") || !strings.Contains(err.Error(), "does not match deposit wallet") {
		t.Fatalf("error = %q, want signer mismatch error", err.Error())
	}
}

func TestSignDepositWalletOrder_RejectsZeroDepositWallet(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	err = SignDepositWalletOrder(signer, PolygonChainID, depositWalletTestOrder(), common.Address{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "deposit wallet address is required") {
		t.Fatalf("error = %q, want deposit wallet address error", err.Error())
	}
}

func TestBuildDepositWalletOrderTypedData_UsesNormalOrderTypedData(t *testing.T) {
	order := depositWalletTestOrder()
	order.Maker = depositWalletTestAddress.Hex()
	order.Signer = depositWalletTestAddress.Hex()
	order.Salt = Int64(12345)
	order.Timestamp = String("1700000000000")
	order.Metadata = ZeroBytes32
	order.Builder = ZeroBytes32

	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	typedData := BuildDepositWalletOrderTypedData(PolygonChainID, contracts.Exchange, *order)

	if typedData.PrimaryType != "Order" {
		t.Fatalf("primaryType = %s, want Order", typedData.PrimaryType)
	}
	if typedData.Domain.Name != orderProtocolName {
		t.Fatalf("domain name = %s, want %s", typedData.Domain.Name, orderProtocolName)
	}
	if typedData.Domain.Version != orderProtocolVersion {
		t.Fatalf("domain version = %s, want %s", typedData.Domain.Version, orderProtocolVersion)
	}
	if typedData.Domain.VerifyingContract != contracts.Exchange.Hex() {
		t.Fatalf("verifyingContract = %s, want %s", typedData.Domain.VerifyingContract, contracts.Exchange.Hex())
	}
	if got := typedData.Message["signatureType"]; got != "3" {
		t.Fatalf("message signatureType = %v, want 3", got)
	}
	if got := typedData.Message["maker"]; !strings.EqualFold(got.(string), depositWalletTestAddress.Hex()) {
		t.Fatalf("message maker = %v, want %s", got, depositWalletTestAddress.Hex())
	}
	if got := typedData.Message["signer"]; !strings.EqualFold(got.(string), depositWalletTestAddress.Hex()) {
		t.Fatalf("message signer = %v, want %s", got, depositWalletTestAddress.Hex())
	}
}

func TestSignDepositWalletOrder_GoldenPyClobClientV2(t *testing.T) {
	vector := loadDepositWalletGoldenVector(t)

	signer, err := ParsePrivateKey(vector.Input.PrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	depositWallet := common.HexToAddress(vector.Input.DepositWallet)
	expected := vector.Expected.SignedOrder

	order := signedOrderFromDepositWalletGolden(t, expected)

	if err := SignDepositWalletOrder(
		signer,
		vector.Input.ChainID,
		order,
		depositWallet,
		WithSignOrderSalt(mustBigIntFromDecimal(t, vector.Input.Salt)),
		WithSignOrderTime(time.UnixMilli(mustInt64FromDecimal(t, vector.Input.Timestamp))),
	); err != nil {
		t.Fatalf("SignDepositWalletOrder: %v", err)
	}

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
	assertEqualFold(t, "signature", expected.Signature, order.Signature)

	expectedSide := Buy
	if expected.Side == 1 {
		expectedSide = Sell
	}
	assertEqual(t, "side", expectedSide, order.Side)

	deferExec := boolFromMapDefault(vector.Expected.PostOrderRequest, "deferExec", false)
	actualPost := PostOrderRequest{
		Order:     *order,
		Owner:     stringFromMap(t, vector.Expected.PostOrderRequest, "owner"),
		OrderType: OrderType(stringFromMap(t, vector.Expected.PostOrderRequest, "orderType")),
		DeferExec: &deferExec,
	}

	assertPostOrderRequestEqual(t, vector.Expected.PostOrderRequest, actualPost)
	assertPostOrderSaltIsJSONNumber(t, actualPost)
}

func TestSignOrder_SupportsDepositWalletPoly1271(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	order := depositWalletTestOrder()
	order.Maker = depositWalletTestAddress.Hex()
	order.Signer = depositWalletTestAddress.Hex()

	err = SignOrder(
		signer,
		PolygonChainID,
		order,
		WithSignOrderSalt(big.NewInt(12345)),
		WithSignOrderTime(time.UnixMilli(1700000000000)),
	)
	if err != nil {
		t.Fatalf("SignOrder: %v", err)
	}

	if order.SignatureType != SignatureTypePoly1271 {
		t.Fatalf("signatureType = %d, want %d", order.SignatureType, SignatureTypePoly1271)
	}
	if !strings.EqualFold(order.Maker, depositWalletTestAddress.Hex()) {
		t.Fatalf("maker = %s, want %s", order.Maker, depositWalletTestAddress.Hex())
	}
	if !strings.EqualFold(order.Signer, depositWalletTestAddress.Hex()) {
		t.Fatalf("signer = %s, want %s", order.Signer, depositWalletTestAddress.Hex())
	}
	assertDepositWalletWrappedSignatureShape(t, order)
}

func TestSignOrder_DepositWalletMatchesExplicitSignDepositWalletOrder(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	orderA := depositWalletTestOrder()
	orderA.Maker = depositWalletTestAddress.Hex()
	orderA.Signer = depositWalletTestAddress.Hex()

	orderB := depositWalletTestOrder()

	opts := []SignOrderOption{
		WithSignOrderSalt(big.NewInt(12345)),
		WithSignOrderTime(time.UnixMilli(1700000000000)),
	}

	if err := SignOrder(signer, PolygonChainID, orderA, opts...); err != nil {
		t.Fatalf("SignOrder: %v", err)
	}

	if err := SignDepositWalletOrder(signer, PolygonChainID, orderB, depositWalletTestAddress, opts...); err != nil {
		t.Fatalf("SignDepositWalletOrder: %v", err)
	}

	if orderA.Signature != orderB.Signature {
		t.Fatalf("signature mismatch:\nSignOrder=%s\nSignDepositWalletOrder=%s", orderA.Signature, orderB.Signature)
	}
	if orderA.Maker != orderB.Maker {
		t.Fatalf("maker mismatch: SignOrder=%s SignDepositWalletOrder=%s", orderA.Maker, orderB.Maker)
	}
	if orderA.Signer != orderB.Signer {
		t.Fatalf("signer mismatch: SignOrder=%s SignDepositWalletOrder=%s", orderA.Signer, orderB.Signer)
	}
}

func TestSignOrder_DepositWalletRequiresMakerOrSigner(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	order := depositWalletTestOrder()
	order.Maker = ""
	order.Signer = ""

	err = SignOrder(signer, PolygonChainID, order)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires maker or signer deposit wallet address") {
		t.Fatalf("error = %q, want deposit wallet inference error", err.Error())
	}
}

func TestBuildOrder_Poly1271UsesMakerAsSigner(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
	)
	builder := NewOrderBuilder(client)

	sigType := SignatureTypePoly1271
	order, err := builder.BuildOrder(OrderArgsV2{
		TokenID:       depositWalletTestTokenID,
		Price:         "0.42",
		Size:          "10",
		Side:          Buy,
		SignatureType: &sigType,
		Maker:         depositWalletTestAddress.Hex(),
	}, CreateOrderOptions{TickSize: "0.01"})
	if err != nil {
		t.Fatalf("BuildOrder: %v", err)
	}

	if order.SignatureType != SignatureTypePoly1271 {
		t.Fatalf("signatureType = %d, want %d", order.SignatureType, SignatureTypePoly1271)
	}
	if !strings.EqualFold(order.Maker, depositWalletTestAddress.Hex()) {
		t.Fatalf("maker = %s, want %s", order.Maker, depositWalletTestAddress.Hex())
	}
	if !strings.EqualFold(order.Signer, depositWalletTestAddress.Hex()) {
		t.Fatalf("signer = %s, want %s", order.Signer, depositWalletTestAddress.Hex())
	}
	assertDepositWalletWrappedSignatureShape(t, order)
}

func TestBuildMarketOrder_Poly1271UsesMakerAsSigner(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
	)
	builder := NewOrderBuilder(client)

	sigType := SignatureTypePoly1271
	order, err := builder.BuildMarketOrder(MarketOrderArgsV2{
		TokenID:       depositWalletTestTokenID,
		Price:         "0.42",
		Amount:        "25",
		Side:          Buy,
		SignatureType: &sigType,
		Maker:         depositWalletTestAddress.Hex(),
	}, CreateOrderOptions{TickSize: "0.01"})
	if err != nil {
		t.Fatalf("BuildMarketOrder: %v", err)
	}

	if order.SignatureType != SignatureTypePoly1271 {
		t.Fatalf("signatureType = %d, want %d", order.SignatureType, SignatureTypePoly1271)
	}
	if !strings.EqualFold(order.Maker, depositWalletTestAddress.Hex()) {
		t.Fatalf("maker = %s, want %s", order.Maker, depositWalletTestAddress.Hex())
	}
	if !strings.EqualFold(order.Signer, depositWalletTestAddress.Hex()) {
		t.Fatalf("signer = %s, want %s", order.Signer, depositWalletTestAddress.Hex())
	}
	assertDepositWalletWrappedSignatureShape(t, order)
}

func TestBuildOrder_Poly1271RequiresMakerDepositWallet(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	client := NewClient(
		"",
		WithSigner(signer),
		WithChainID(PolygonChainID),
	)
	builder := NewOrderBuilder(client)

	sigType := SignatureTypePoly1271
	_, err = builder.BuildOrder(OrderArgsV2{
		TokenID:       depositWalletTestTokenID,
		Price:         "0.42",
		Size:          "10",
		Side:          Buy,
		SignatureType: &sigType,
		// Maker intentionally empty.
	}, CreateOrderOptions{TickSize: "0.01"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires maker or signer deposit wallet address") {
		t.Fatalf("error = %q, want deposit wallet maker error", err.Error())
	}
}

type depositWalletGoldenFile struct {
	Schema   string                    `json:"schema"`
	Name     string                    `json:"name"`
	Kind     string                    `json:"kind"`
	Input    depositWalletGoldenInput  `json:"input"`
	Expected depositWalletGoldenOutput `json:"expected"`
}

type depositWalletGoldenInput struct {
	PrivateKey    string `json:"privateKey"`
	ChainID       int64  `json:"chainId"`
	TokenID       string `json:"tokenId"`
	DepositWallet string `json:"depositWallet"`
	SignatureType int    `json:"signatureType"`
	Owner         string `json:"owner"`
	Timestamp     string `json:"timestamp"`
	Salt          string `json:"salt"`
	Side          string `json:"side"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Expiration    string `json:"expiration"`
	Metadata      string `json:"metadata"`
	Builder       string `json:"builder"`
	Exchange      string `json:"exchange"`
}

type depositWalletGoldenOutput struct {
	OwnerSigner      string         `json:"ownerSigner"`
	DepositWallet    string         `json:"depositWallet"`
	OrderHash        string         `json:"orderHash"`
	TypedData        map[string]any `json:"typedData"`
	SignedOrder      goldenOrder    `json:"signedOrder"`
	PostOrderRequest map[string]any `json:"postOrderRequest"`
}

func loadDepositWalletGoldenVector(t *testing.T) depositWalletGoldenFile {
	t.Helper()

	path := filepath.Join("..", "testdata", "golden", "py-clob-client-v2", "clob_order_v2_deposit_wallet.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read deposit wallet golden file %s: %v", path, err)
	}

	var vector depositWalletGoldenFile
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatalf("decode deposit wallet golden file %s: %v", path, err)
	}

	if vector.Kind != "clob_order_v2_deposit_wallet" {
		t.Fatalf("golden kind = %q, want clob_order_v2_deposit_wallet", vector.Kind)
	}
	if vector.Input.SignatureType != int(SignatureTypePoly1271) {
		t.Fatalf("golden signatureType = %d, want %d", vector.Input.SignatureType, SignatureTypePoly1271)
	}
	if !strings.EqualFold(vector.Input.DepositWallet, vector.Expected.DepositWallet) {
		t.Fatalf("deposit wallet mismatch: input=%s expected=%s", vector.Input.DepositWallet, vector.Expected.DepositWallet)
	}

	return vector
}

func signedOrderFromDepositWalletGolden(t *testing.T, expected goldenOrder) *SignedOrder {
	t.Helper()

	salt := mustInt64FromDecimal(t, expected.Salt)

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

func mustBigIntFromDecimal(t *testing.T, value string) *big.Int {
	t.Helper()

	n, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	if !ok {
		t.Fatalf("parse decimal big.Int %q", value)
	}
	return n
}

func mustInt64FromDecimal(t *testing.T, value string) int64 {
	t.Helper()

	n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		t.Fatalf("parse decimal int64 %q: %v", value, err)
	}
	return n
}

func depositWalletTestOrder() *SignedOrder {
	return &SignedOrder{
		TokenID:       String(depositWalletTestTokenID),
		MakerAmount:   String("4200000"),
		TakerAmount:   String("10000000"),
		Side:          Buy,
		SignatureType: SignatureTypePoly1271,
	}
}

func assertDepositWalletWrappedSignatureShape(t *testing.T, order *SignedOrder) {
	t.Helper()

	raw, err := hexutil.Decode(order.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	wantLen := 65 + 32 + 32 + len(depositWalletOrderTypeString) + 2
	if len(raw) != wantLen {
		t.Fatalf("wrapped signature length = %d, want %d", len(raw), wantLen)
	}

	if raw[64] != 27 && raw[64] != 28 {
		t.Fatalf("inner signature v = %d, want 27 or 28", raw[64])
	}

	typeStart := 65 + 32 + 32
	typeEnd := typeStart + len(depositWalletOrderTypeString)

	gotType := string(raw[typeStart:typeEnd])
	if gotType != depositWalletOrderTypeString {
		t.Fatalf("contents type mismatch: got %q, want %q", gotType, depositWalletOrderTypeString)
	}

	gotTypeLen := binary.BigEndian.Uint16(raw[typeEnd : typeEnd+2])
	if int(gotTypeLen) != len(depositWalletOrderTypeString) {
		t.Fatalf("contents type length = %d, want %d", gotTypeLen, len(depositWalletOrderTypeString))
	}
}
