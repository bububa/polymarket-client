package clob

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/bububa/polymarket-client/relayer"
	"github.com/ethereum/go-ethereum/common"
)

func TestBinaryPartition(t *testing.T) {
	partition := BinaryPartition()
	if len(partition) != 2 || partition[0].Cmp(big.NewInt(1)) != 0 || partition[1].Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("partition = %v", partition)
	}
}

func TestBuildSplitPositionTx(t *testing.T) {
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient("", WithChainID(PolygonChainID))
	var tx CTFTransaction

	req := SplitBinary(
		contracts.Collateral,
		common.HexToHash("0x1234"),
		big.NewInt(1_000_000),
	)
	err = client.BuildSplitPositionTx(&req, &tx)
	if err != nil {
		t.Fatal(err)
	}
	if tx.To != contracts.CTFCollateralAdapter {
		t.Fatalf("to = %s, want %s", tx.To, contracts.CTFCollateralAdapter)
	}
	if len(tx.Data) < 4 {
		t.Fatalf("calldata too short: %d", len(tx.Data))
	}
	selector := hex.EncodeToString(tx.Data[:4])
	if selector != "72ce4275" {
		t.Fatalf("selector = %s", selector)
	}
}

type captureRelayer struct {
	req *relayer.SubmitTransactionRequest
}

func (r *captureRelayer) SubmitTransaction(_ context.Context, req *relayer.SubmitTransactionRequest, out *relayer.SubmitTransactionResponse) error {
	r.req = req
	*out = relayer.SubmitTransactionResponse{TransactionID: "tx-1", State: "STATE_NEW"}
	return nil
}

func TestSubmitCTFRelayerTransaction(t *testing.T) {
	capture := &captureRelayer{}
	client := NewClient("", WithRelayerSubmitter(capture))
	var resp relayer.SubmitTransactionResponse
	err := client.SubmitCTFRelayerTransaction(context.Background(), &CTFTransaction{
		To:   common.HexToAddress("0x0000000000000000000000000000000000000001"),
		Data: []byte{0x12, 0x34},
	}, &RelayerCTFRequest{
		From:        "0x0000000000000000000000000000000000000001",
		ProxyWallet: "0x0000000000000000000000000000000000000002",
		Nonce:       "7",
		Signature:   "0xsig",
		Type:        relayer.NonceTypeSafe,
	}, &resp)
	if err != nil {
		t.Fatal(err)
	}
	if resp.TransactionID != "tx-1" || capture.req.To != "0x0000000000000000000000000000000000000001" || capture.req.Data != "0x1234" {
		t.Fatalf("response=%+v request=%+v", resp, capture.req)
	}
}
