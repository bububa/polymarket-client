package clob

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBuildSplitPositionTxRejectsInvalidInputs(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))

	tests := []struct {
		name string
		req  *SplitPositionRequest
		out  *CTFTransaction
		want string
	}{
		{
			name: "nil request",
			req:  nil,
			out:  &CTFTransaction{},
			want: "nil split position request",
		},
		{
			name: "nil output",
			req:  validSplitPositionRequest(),
			out:  nil,
			want: "nil CTF transaction output",
		},
		{
			name: "missing condition id",
			req: func() *SplitPositionRequest {
				req := validSplitPositionRequest()
				req.ConditionID = common.Hash{}
				return req
			}(),
			out:  &CTFTransaction{},
			want: "condition id is required",
		},
		{
			name: "missing partition",
			req: func() *SplitPositionRequest {
				req := validSplitPositionRequest()
				req.Partition = nil
				return req
			}(),
			out:  &CTFTransaction{},
			want: "partition is required",
		},
		{
			name: "nil partition item",
			req: func() *SplitPositionRequest {
				req := validSplitPositionRequest()
				req.Partition = []*big.Int{big.NewInt(1), nil}
				return req
			}(),
			out:  &CTFTransaction{},
			want: "partition[1] is required",
		},
		{
			name: "zero partition item",
			req: func() *SplitPositionRequest {
				req := validSplitPositionRequest()
				req.Partition = []*big.Int{big.NewInt(1), big.NewInt(0)}
				return req
			}(),
			out:  &CTFTransaction{},
			want: "partition[1] must be positive",
		},
		{
			name: "missing amount",
			req: func() *SplitPositionRequest {
				req := validSplitPositionRequest()
				req.Amount = nil
				return req
			}(),
			out:  &CTFTransaction{},
			want: "amount is required",
		},
		{
			name: "zero amount",
			req: func() *SplitPositionRequest {
				req := validSplitPositionRequest()
				req.Amount = big.NewInt(0)
				return req
			}(),
			out:  &CTFTransaction{},
			want: "amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.BuildSplitPositionTx(tt.req, tt.out)
			assertErrorContains(t, err, tt.want)
		})
	}
}

func TestBuildSplitPositionTxDefaultsCollateralToken(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	req := validSplitPositionRequest()
	req.CollateralToken = common.Address{}

	var tx CTFTransaction
	if err := client.BuildSplitPositionTx(req, &tx); err != nil {
		t.Fatalf("BuildSplitPositionTx: %v", err)
	}

	assertAddressEqualCTF(t, "to", contracts.CTFCollateralAdapter, tx.To)

	values, err := ctfABI.Methods["splitPosition"].Inputs.Unpack(tx.Data[4:])
	if err != nil {
		t.Fatalf("unpack splitPosition calldata: %v", err)
	}
	assertAddressEqualCTF(t, "default collateral", contracts.Collateral, values[0].(common.Address))
}

func TestBuildMergePositionsTxRejectsInvalidInputs(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))

	tests := []struct {
		name string
		req  *MergePositionsRequest
		out  *CTFTransaction
		want string
	}{
		{
			name: "nil request",
			req:  nil,
			out:  &CTFTransaction{},
			want: "nil merge position request",
		},
		{
			name: "nil output",
			req:  validMergePositionsRequest(),
			out:  nil,
			want: "nil CTF transaction output",
		},
		{
			name: "missing condition id",
			req: func() *MergePositionsRequest {
				req := validMergePositionsRequest()
				req.ConditionID = common.Hash{}
				return req
			}(),
			out:  &CTFTransaction{},
			want: "condition id is required",
		},
		{
			name: "missing partition",
			req: func() *MergePositionsRequest {
				req := validMergePositionsRequest()
				req.Partition = nil
				return req
			}(),
			out:  &CTFTransaction{},
			want: "partition is required",
		},
		{
			name: "missing amount",
			req: func() *MergePositionsRequest {
				req := validMergePositionsRequest()
				req.Amount = nil
				return req
			}(),
			out:  &CTFTransaction{},
			want: "amount is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.BuildMergePositionsTx(tt.req, tt.out)
			assertErrorContains(t, err, tt.want)
		})
	}
}

func TestBuildMergePositionsTxDefaultsCollateralToken(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	req := validMergePositionsRequest()
	req.CollateralToken = common.Address{}

	var tx CTFTransaction
	if err := client.BuildMergePositionsTx(req, &tx); err != nil {
		t.Fatalf("BuildMergePositionsTx: %v", err)
	}

	assertAddressEqualCTF(t, "to", contracts.CTFCollateralAdapter, tx.To)

	values, err := ctfABI.Methods["mergePositions"].Inputs.Unpack(tx.Data[4:])
	if err != nil {
		t.Fatalf("unpack mergePositions calldata: %v", err)
	}
	assertAddressEqualCTF(t, "default collateral", contracts.Collateral, values[0].(common.Address))
}

func TestBuildRedeemPositionsTxRejectsInvalidInputs(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))

	tests := []struct {
		name string
		req  *RedeemPositionsRequest
		out  *CTFTransaction
		want string
	}{
		{
			name: "nil request",
			req:  nil,
			out:  &CTFTransaction{},
			want: "nil redeem position request",
		},
		{
			name: "nil output",
			req:  validRedeemPositionsRequest(),
			out:  nil,
			want: "nil CTF transaction output",
		},
		{
			name: "missing condition id",
			req: func() *RedeemPositionsRequest {
				req := validRedeemPositionsRequest()
				req.ConditionID = common.Hash{}
				return req
			}(),
			out:  &CTFTransaction{},
			want: "condition id is required",
		},
		{
			name: "missing index sets",
			req: func() *RedeemPositionsRequest {
				req := validRedeemPositionsRequest()
				req.IndexSets = nil
				return req
			}(),
			out:  &CTFTransaction{},
			want: "index sets is required",
		},
		{
			name: "nil index set",
			req: func() *RedeemPositionsRequest {
				req := validRedeemPositionsRequest()
				req.IndexSets = []*big.Int{big.NewInt(1), nil}
				return req
			}(),
			out:  &CTFTransaction{},
			want: "index sets[1] is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.BuildRedeemPositionsTx(tt.req, tt.out)
			assertErrorContains(t, err, tt.want)
		})
	}
}

func TestBuildRedeemPositionsTxDefaultsCollateralToken(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	req := validRedeemPositionsRequest()
	req.CollateralToken = common.Address{}

	var tx CTFTransaction
	if err := client.BuildRedeemPositionsTx(req, &tx); err != nil {
		t.Fatalf("BuildRedeemPositionsTx: %v", err)
	}

	assertAddressEqualCTF(t, "to", contracts.CTFCollateralAdapter, tx.To)

	values, err := ctfABI.Methods["redeemPositions"].Inputs.Unpack(tx.Data[4:])
	if err != nil {
		t.Fatalf("unpack redeemPositions calldata: %v", err)
	}
	assertAddressEqualCTF(t, "default collateral", contracts.Collateral, values[0].(common.Address))
}

func TestBuildRedeemNegRiskTxRejectsInvalidInputs(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))

	tests := []struct {
		name string
		req  *RedeemNegRiskRequest
		out  *CTFTransaction
		want string
	}{
		{
			name: "nil request",
			req:  nil,
			out:  &CTFTransaction{},
			want: "nil neg-risk redeem request",
		},
		{
			name: "nil output",
			req:  validRedeemNegRiskRequest(),
			out:  nil,
			want: "nil CTF transaction output",
		},
		{
			name: "missing condition id",
			req: func() *RedeemNegRiskRequest {
				req := validRedeemNegRiskRequest()
				req.ConditionID = common.Hash{}
				return req
			}(),
			out:  &CTFTransaction{},
			want: "condition id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.BuildRedeemNegRiskTx(tt.req, tt.out)
			assertErrorContains(t, err, tt.want)
		})
	}
}

func TestBuildRedeemNegRiskTxUsesNegRiskCTFCollateralAdapter(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatalf("Contracts: %v", err)
	}

	var tx CTFTransaction
	if err := client.BuildRedeemNegRiskTx(validRedeemNegRiskRequest(), &tx); err != nil {
		t.Fatalf("BuildRedeemNegRiskTx: %v", err)
	}

	assertAddressEqualCTF(t, "to", contracts.NegRiskCTFCollateralAdapter, tx.To)
	assertMethodSelectorCTF(t, "redeemPositions", ctfABI.Methods["redeemPositions"].ID, tx.Data)

	values, err := ctfABI.Methods["redeemPositions"].Inputs.Unpack(tx.Data[4:])
	if err != nil {
		t.Fatalf("unpack neg-risk adapter redeemPositions calldata: %v", err)
	}

	assertAddressEqualCTF(t, "ignored collateral token", common.Address{}, values[0].(common.Address))
	assertHashEqualCTF(t, "ignored parent collection", common.Hash{}, values[1].([32]byte))
	assertHashEqualCTF(t, "condition id", ctfSafetyConditionID, values[2].([32]byte))
	assertBigIntSliceEqualCTF(t, "index sets placeholder", BinaryPartition(), values[3].([]*big.Int))
}

func validSplitPositionRequest() *SplitPositionRequest {
	return &SplitPositionRequest{
		CollateralToken:    ctfSafetyCollateral,
		ParentCollectionID: common.Hash{},
		ConditionID:        ctfSafetyConditionID,
		Partition:          BinaryPartition(),
		Amount:             big.NewInt(1_000_000),
	}
}

func validMergePositionsRequest() *MergePositionsRequest {
	return &MergePositionsRequest{
		CollateralToken:    ctfSafetyCollateral,
		ParentCollectionID: common.Hash{},
		ConditionID:        ctfSafetyConditionID,
		Partition:          BinaryPartition(),
		Amount:             big.NewInt(1_000_000),
	}
}

func validRedeemPositionsRequest() *RedeemPositionsRequest {
	return &RedeemPositionsRequest{
		CollateralToken:    ctfSafetyCollateral,
		ParentCollectionID: common.Hash{},
		ConditionID:        ctfSafetyConditionID,
		IndexSets:          BinaryPartition(),
	}
}

func validRedeemNegRiskRequest() *RedeemNegRiskRequest {
	return &RedeemNegRiskRequest{
		ConditionID: ctfSafetyConditionID,
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}
