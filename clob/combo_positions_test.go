package clob

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestComboPositionBalances(t *testing.T) {
	legs := []string{comboTestLeg(1, 0), comboTestLeg(2, 1)}
	combo, err := DeriveComboPositionContext(legs)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
			ID     json.RawMessage   `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var call struct {
			To    string `json:"to"`
			Data  string `json:"data"`
			Input string `json:"input"`
		}
		if request.Method != "eth_call" || len(request.Params) == 0 || json.Unmarshal(request.Params[0], &call) != nil {
			http.Error(w, "unexpected rpc request", http.StatusBadRequest)
			return
		}
		if !strings.EqualFold(call.To, ContractAddressComboPositionManager) {
			http.Error(w, "unexpected position manager", http.StatusBadRequest)
			return
		}
		if call.Data == "" {
			call.Data = call.Input
		}
		data := common.FromHex(call.Data)
		if len(data) < 4 {
			http.Error(w, "missing calldata", http.StatusBadRequest)
			return
		}
		var result []byte
		switch string(data[:4]) {
		case string(comboPositionManagerABI.Methods["balanceOf"].ID):
			result, _ = comboPositionManagerABI.Methods["balanceOf"].Outputs.Pack(big.NewInt(7))
		case string(comboPositionManagerABI.Methods["balanceOfBatch"].ID):
			result, _ = comboPositionManagerABI.Methods["balanceOfBatch"].Outputs.Pack([]*big.Int{big.NewInt(9), big.NewInt(4)})
		default:
			http.Error(w, "unexpected selector", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(request.ID), "result": "0x" + common.Bytes2Hex(result)})
	}))
	defer server.Close()

	client := NewClient("", WithRPCURL(server.URL))
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	balance, err := client.ComboPositionBalance(context.Background(), owner, combo.PositionIDs[0])
	if err != nil || balance.Cmp(big.NewInt(7)) != 0 {
		t.Fatalf("ComboPositionBalance = %v, %v", balance, err)
	}
	amount, err := client.ComboMaxMergeAmount(context.Background(), owner, legs)
	if err != nil || amount.Cmp(big.NewInt(4)) != 0 {
		t.Fatalf("ComboMaxMergeAmount = %v, %v", amount, err)
	}
	resolved, err := client.resolveComboMergeAmount(context.Background(), &ComboMergePositionsRequest{LegPositionIDs: legs, Amount: big.NewInt(4)}, owner)
	if err != nil || resolved.Amount.Cmp(big.NewInt(4)) != 0 {
		t.Fatalf("resolve explicit merge amount = %v, %v", resolved, err)
	}
	_, err = client.resolveComboMergeAmount(context.Background(), &ComboMergePositionsRequest{LegPositionIDs: legs, Amount: big.NewInt(5)}, owner)
	if err == nil || !strings.Contains(err.Error(), "exceeds available amount") {
		t.Fatalf("oversized merge error = %v", err)
	}
}

func TestDeriveComboPositionContextOfficialVector(t *testing.T) {
	legs := []string{comboTestLeg(2, 1), comboTestLeg(1, 0)}
	context, err := DeriveComboPositionContext(legs)
	if err != nil {
		t.Fatalf("DeriveComboPositionContext: %v", err)
	}
	want := "032def24bfb0c5c57fb236fac08b94236a0000000000000000000000000000"
	if got := common.Bytes2Hex(context.ConditionID[:]); got != want {
		t.Fatalf("condition id = %s, want %s", got, want)
	}
	for outcome, positionID := range context.PositionIDs {
		condition, gotOutcome, err := DecodeComboPositionID(positionID)
		if err != nil {
			t.Fatalf("DecodeComboPositionID(%d): %v", outcome, err)
		}
		if condition != context.ConditionID || int(gotOutcome) != outcome {
			t.Fatalf("decoded position %d = (%x,%d)", outcome, condition, gotOutcome)
		}
	}
}

func TestCanonicalizeComboLegsValidation(t *testing.T) {
	tests := []struct {
		name string
		legs []string
		want string
	}{
		{name: "empty", want: "1 to 50"},
		{name: "invalid uint256", legs: []string{"nope"}, want: "uint256"},
		{name: "combo module leg", legs: []string{comboTestPosition(3, 1, 0)}, want: "binary or neg-risk"},
		{name: "duplicate", legs: []string{comboTestLeg(1, 0), comboTestLeg(1, 0)}, want: "duplicate"},
		{name: "both outcomes", legs: []string{comboTestLeg(1, 0), comboTestLeg(1, 1)}, want: "both outcomes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CanonicalizeComboLegs(tt.legs)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestBuildComboPositionTransactions(t *testing.T) {
	client := NewClient("")
	var collateralApproval CTFTransaction
	if err := client.BuildComboCollateralApprovalTx(big.NewInt(10), &collateralApproval); err != nil {
		t.Fatalf("BuildComboCollateralApprovalTx: %v", err)
	}
	if collateralApproval.To != common.HexToAddress(ContractAddressPUSD) {
		t.Fatalf("collateral approval target = %s", collateralApproval.To)
	}
	assertComboSelector(t, collateralApproval.Data, comboApprovalABI.Methods["approve"].ID)
	if err := client.BuildComboCollateralApprovalTx(big.NewInt(0), &collateralApproval); err != nil {
		t.Fatalf("BuildComboCollateralApprovalTx zero revoke: %v", err)
	}
	approvalValues, err := comboApprovalABI.Methods["approve"].Inputs.Unpack(collateralApproval.Data[4:])
	if err != nil {
		t.Fatalf("unpack zero approval: %v", err)
	}
	if approvalValues[1].(*big.Int).Sign() != 0 {
		t.Fatalf("zero approval amount = %v", approvalValues[1])
	}
	var positionApproval CTFTransaction
	if err := client.BuildComboPositionApprovalTx(true, &positionApproval); err != nil {
		t.Fatalf("BuildComboPositionApprovalTx: %v", err)
	}
	if positionApproval.To != common.HexToAddress(ContractAddressComboPositionManager) {
		t.Fatalf("position approval target = %s", positionApproval.To)
	}
	assertComboSelector(t, positionApproval.Data, comboApprovalABI.Methods["setApprovalForAll"].ID)
	legs := []string{comboTestLeg(2, 1), comboTestLeg(1, 0)}
	context, err := DeriveComboPositionContext(legs)
	if err != nil {
		t.Fatal(err)
	}

	var split []CTFTransaction
	if err := client.BuildComboSplitPositionTxs(&ComboSplitPositionRequest{LegPositionIDs: legs, Amount: big.NewInt(5)}, &split); err != nil {
		t.Fatalf("BuildComboSplitPositionTxs: %v", err)
	}
	if len(split) != 2 || split[0].To.Hex() != common.HexToAddress(ContractAddressCombinatorialModule).Hex() || split[1].To.Hex() != common.HexToAddress(ContractAddressComboRouter).Hex() {
		t.Fatalf("unexpected split targets: %+v", split)
	}
	assertComboSelector(t, split[0].Data, comboABI.Methods["prepareCondition"].ID)
	assertComboSelector(t, split[1].Data, comboABI.Methods["split"].ID)

	var merge []CTFTransaction
	if err := client.BuildComboMergePositionsTxs(&ComboMergePositionsRequest{LegPositionIDs: legs, Amount: big.NewInt(4)}, &merge); err != nil {
		t.Fatalf("BuildComboMergePositionsTxs: %v", err)
	}
	assertComboSelector(t, merge[1].Data, comboABI.Methods["merge"].ID)

	var redeem CTFTransaction
	if err := client.BuildComboRedeemPositionTx(&ComboRedeemPositionRequest{PositionID: context.PositionIDs[1], Amount: big.NewInt(3)}, &redeem); err != nil {
		t.Fatalf("BuildComboRedeemPositionTx: %v", err)
	}
	assertComboSelector(t, redeem.Data, comboABI.Methods["redeem"].ID)
	values, err := comboABI.Methods["redeem"].Inputs.Unpack(redeem.Data[4:])
	if err != nil {
		t.Fatalf("unpack redeem: %v", err)
	}
	condition := ComboConditionID(values[0].([31]byte))
	if condition != context.ConditionID || values[1].(*big.Int).Uint64() != 1 || values[2].(*big.Int).Cmp(big.NewInt(3)) != 0 {
		t.Fatalf("unexpected redeem values: %#v", values)
	}
}

func TestBuildComboTransactionsRejectUnsupportedChain(t *testing.T) {
	client := NewClient("", WithChainID(80002))
	var txs []CTFTransaction
	err := client.BuildComboSplitPositionTxs(&ComboSplitPositionRequest{LegPositionIDs: []string{comboTestLeg(1, 0)}, Amount: big.NewInt(1)}, &txs)
	if err == nil || !strings.Contains(err.Error(), "unsupported chain") {
		t.Fatalf("error = %v", err)
	}
}

func comboTestLeg(marker, outcome byte) string { return comboTestPosition(1, marker, outcome) }

func comboTestPosition(module, marker, outcome byte) string {
	value := make([]byte, 32)
	value[0], value[30], value[31] = module, marker, outcome
	return new(big.Int).SetBytes(value).String()
}

func assertComboSelector(t *testing.T, data, selector []byte) {
	t.Helper()
	if len(data) < 4 || string(data[:4]) != string(selector) {
		t.Fatalf("selector = %x, want %x", data, selector)
	}
}
