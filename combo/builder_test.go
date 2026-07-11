package combo

import (
	"strings"
	"testing"

	"github.com/bububa/polymarket-client/clob"
	pmtypes "github.com/bububa/polymarket-client/shared"
	"github.com/ethereum/go-ethereum/common"
)

func TestQuoteBuilderBuyCollateral(t *testing.T) {
	signer, err := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	if err != nil {
		t.Fatal(err)
	}
	identity := Identity{SignerAddress: signer.Address().Hex(), MakerAddress: signer.Address().Hex(), SignatureType: clob.SignatureTypeEOA}
	client := NewClient("", WithSigner(signer), WithIdentity(identity))
	request := QuoteRequest{RFQID: "r", ConditionID: "0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", YesPositionID: "11", NoPositionID: "22", Direction: DirectionBuy, Side: SideYes, RequestedSize: RequestedSize{Unit: RequestedSizeNotional, ValueE6: "1000000"}}
	quote, err := NewQuoteBuilder(client).BuildAndSign(request, QuoteOptions{Price: "0.45", Source: QuoteSourceCollateral})
	if err != nil {
		t.Fatal(err)
	}
	if quote.SizeE6 != "2222222" || quote.SignedOrder.TokenID != pmtypes.String("22") || quote.SignedOrder.MakerAmount != "1222223" || quote.SignedOrder.TakerAmount != "2222222" {
		t.Fatalf("quote = %+v", quote)
	}
	if quote.SignedOrder.Side != 0 || !strings.HasPrefix(quote.SignedOrder.Signature, "0x") {
		t.Fatalf("order = %+v", quote.SignedOrder)
	}
}

func TestQuoteBuilderUsesConfiguredComboExchange(t *testing.T) {
	contracts, err := clob.Contracts(clob.PolygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	order := SignedOrder{}
	typed := buildTypedData(clob.PolygonChainID, contracts.ComboExchange, order)
	if !strings.EqualFold(typed.Domain.VerifyingContract, contracts.ComboExchange.Hex()) {
		t.Fatalf("verifying contract = %s", typed.Domain.VerifyingContract)
	}
}

func TestQuoteBuilderRejectsUnsupportedChain(t *testing.T) {
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	request := QuoteRequest{ConditionID: "0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", YesPositionID: "11", NoPositionID: "22", Direction: DirectionSell, Side: SideYes, RequestedSize: RequestedSize{Unit: RequestedSizeShares, ValueE6: "1000000"}}
	for _, signatureType := range []clob.SignatureType{clob.SignatureTypeEOA, clob.SignatureTypePoly1271} {
		t.Run(string(rune('0'+signatureType)), func(t *testing.T) {
			address := signer.Address().Hex()
			if signatureType == clob.SignatureTypePoly1271 {
				address = "0x2222222222222222222222222222222222222222"
			}
			identity := Identity{SignerAddress: address, MakerAddress: address, SignatureType: signatureType}
			_, err := NewQuoteBuilder(NewClient("", WithSigner(signer), WithIdentity(identity), WithChainID(80002))).BuildAndSign(request, QuoteOptions{Price: "0.45"})
			if err == nil || !strings.Contains(err.Error(), "unsupported chain id") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestSignPoly1271UsesProvidedExchange(t *testing.T) {
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	wallet := common.HexToAddress("0x2222222222222222222222222222222222222222")
	order := SignedOrder{Salt: "1", Maker: wallet.Hex(), Signer: wallet.Hex(), TokenID: "2", MakerAmount: "3", TakerAmount: "4", SignatureType: clob.SignatureTypePoly1271, Timestamp: "5", Metadata: ZeroBytes32, Builder: ZeroBytes32}
	sigA, err := signPoly1271(signer, clob.PolygonChainID, common.HexToAddress(clob.ContractAddressComboExchange), wallet, order)
	if err != nil {
		t.Fatal(err)
	}
	sigB, err := signPoly1271(signer, clob.PolygonChainID, common.HexToAddress("0x3333333333333333333333333333333333333333"), wallet, order)
	if err != nil {
		t.Fatal(err)
	}
	if sigA == sigB {
		t.Fatal("signature did not change with exchange")
	}
}

func TestQuoteBuilderInventoryUsesSellAmounts(t *testing.T) {
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	identity := Identity{SignerAddress: signer.Address().Hex(), MakerAddress: signer.Address().Hex(), SignatureType: clob.SignatureTypeEOA}
	request := QuoteRequest{ConditionID: "0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", YesPositionID: "11", NoPositionID: "22", Direction: DirectionSell, Side: SideYes, RequestedSize: RequestedSize{Unit: RequestedSizeShares, ValueE6: "1000000"}}
	quote, err := NewQuoteBuilder(NewClient("", WithSigner(signer), WithIdentity(identity))).BuildAndSign(request, QuoteOptions{Price: "0.45", Source: QuoteSourceInventory})
	if err != nil {
		t.Fatal(err)
	}
	if quote.SignedOrder.Side != 1 || quote.SignedOrder.TokenID != "22" || quote.SignedOrder.MakerAmount != "1000000" || quote.SignedOrder.TakerAmount != "550000" {
		t.Fatalf("order = %+v", quote.SignedOrder)
	}
}

func TestQuoteBuilderRejectsZeroOrderAmount(t *testing.T) {
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	identity := Identity{SignerAddress: signer.Address().Hex(), MakerAddress: signer.Address().Hex(), SignatureType: clob.SignatureTypeEOA}
	request := QuoteRequest{ConditionID: "0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", YesPositionID: "11", NoPositionID: "22", Direction: DirectionSell, Side: SideYes, RequestedSize: RequestedSize{Unit: RequestedSizeShares, ValueE6: "1"}}
	_, err := NewQuoteBuilder(NewClient("", WithSigner(signer), WithIdentity(identity))).BuildAndSign(request, QuoteOptions{Price: "0.999999", Size: "0.000001", Source: QuoteSourceInventory})
	if err == nil || !strings.Contains(err.Error(), "zero order amount") {
		t.Fatalf("error = %v", err)
	}
}

func TestQuoteBuilderValidatesOrderIdentity(t *testing.T) {
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	wallet := "0x2222222222222222222222222222222222222222"
	request := QuoteRequest{ConditionID: "0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", YesPositionID: "11", NoPositionID: "22", Direction: DirectionSell, Side: SideYes, RequestedSize: RequestedSize{Unit: RequestedSizeShares, ValueE6: "1000000"}}
	tests := []struct {
		name     string
		identity Identity
		want     string
	}{
		{name: "invalid maker", identity: Identity{SignerAddress: signer.Address().Hex(), MakerAddress: "invalid", SignatureType: clob.SignatureTypeEOA}, want: "valid evm addresses"},
		{name: "zero maker", identity: Identity{SignerAddress: signer.Address().Hex(), MakerAddress: "0x0000000000000000000000000000000000000000", SignatureType: clob.SignatureTypeProxy}, want: "must not be zero"},
		{name: "eoa maker mismatch", identity: Identity{SignerAddress: signer.Address().Hex(), MakerAddress: wallet, SignatureType: clob.SignatureTypeEOA}, want: "EOA signer and maker addresses must match"},
		{name: "legacy signer mismatch", identity: Identity{SignerAddress: wallet, MakerAddress: wallet, SignatureType: clob.SignatureTypeProxy}, want: "does not match cryptographic signer"},
		{name: "poly1271 identity mismatch", identity: Identity{SignerAddress: wallet, MakerAddress: signer.Address().Hex(), SignatureType: clob.SignatureTypePoly1271}, want: "must match"},
		{name: "unsupported signature type", identity: Identity{SignerAddress: signer.Address().Hex(), MakerAddress: wallet, SignatureType: clob.SignatureType(99)}, want: "unsupported signature type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewQuoteBuilder(NewClient("", WithSigner(signer), WithIdentity(tt.identity))).BuildAndSign(request, QuoteOptions{Price: "0.45"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestQuoteBuilderPoly1271(t *testing.T) {
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	wallet := "0x2222222222222222222222222222222222222222"
	identity := Identity{SignerAddress: wallet, MakerAddress: wallet, SignatureType: clob.SignatureTypePoly1271}
	request := QuoteRequest{ConditionID: "0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", YesPositionID: "11", NoPositionID: "22", Direction: DirectionSell, Side: SideYes, RequestedSize: RequestedSize{Unit: RequestedSizeShares, ValueE6: "1000000"}}
	quote, err := NewQuoteBuilder(NewClient("", WithSigner(signer), WithIdentity(identity))).BuildAndSign(request, QuoteOptions{Price: "0.45"})
	if err != nil {
		t.Fatal(err)
	}
	if len(quote.SignedOrder.Signature) <= 132 {
		t.Fatalf("wrapped signature too short: %d", len(quote.SignedOrder.Signature))
	}
}

func TestQuoteBuilderLegacySignatureTypes(t *testing.T) {
	signer, _ := clob.ParsePrivateKey("4f3edf983ac636a65a842ce7c78d9aa706d3b113bce036f06f60d7fd4d0d0d01")
	request := QuoteRequest{ConditionID: "0x03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", YesPositionID: "11", NoPositionID: "22", Direction: DirectionSell, Side: SideYes, RequestedSize: RequestedSize{Unit: RequestedSizeShares, ValueE6: "1000000"}}
	for _, signatureType := range []clob.SignatureType{clob.SignatureTypeEOA, clob.SignatureTypeProxy, clob.SignatureTypeGnosisSafe} {
		t.Run(string(rune('0'+signatureType)), func(t *testing.T) {
			maker := "0x2222222222222222222222222222222222222222"
			if signatureType == clob.SignatureTypeEOA {
				maker = signer.Address().Hex()
			}
			identity := Identity{SignerAddress: signer.Address().Hex(), MakerAddress: maker, SignatureType: signatureType}
			quote, err := NewQuoteBuilder(NewClient("", WithSigner(signer), WithIdentity(identity))).BuildAndSign(request, QuoteOptions{Price: "0.45"})
			if err != nil {
				t.Fatal(err)
			}
			if quote.SignedOrder.SignatureType != signatureType || len(quote.SignedOrder.Signature) != 132 {
				t.Fatalf("order = %+v", quote.SignedOrder)
			}
		})
	}
}
