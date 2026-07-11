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
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestNegRiskAdapterGetters(t *testing.T) {
	marketID := common.HexToHash("0x1112131415161718192021222324252627282930313233343536373839404142")
	contracts, err := Contracts(PolygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		method string
		value  *big.Int
		call   func(context.Context, *Client, common.Hash) (*big.Int, error)
	}{
		{name: "fee bps", method: "getFeeBips", value: big.NewInt(125), call: func(ctx context.Context, client *Client, id common.Hash) (*big.Int, error) {
			return client.GetFeeBps(ctx, id)
		}},
		{name: "question count", method: "getQuestionCount", value: big.NewInt(17), call: func(ctx context.Context, client *Client, id common.Hash) (*big.Int, error) {
			return client.GetQuestionCount(ctx, id)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					JSONRPC string            `json:"jsonrpc"`
					ID      json.RawMessage   `json:"id"`
					Method  string            `json:"method"`
					Params  []json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				if request.Method != "eth_call" || len(request.Params) == 0 {
					t.Fatalf("rpc request = %+v", request)
				}
				var call struct {
					To    string `json:"to"`
					Input string `json:"input"`
				}
				if err := json.Unmarshal(request.Params[0], &call); err != nil {
					t.Fatal(err)
				}
				if !strings.EqualFold(call.To, contracts.NegRiskAdapter.Hex()) {
					t.Fatalf("to = %q", call.To)
				}
				input, err := hexutil.Decode(call.Input)
				if err != nil {
					t.Fatal(err)
				}
				method := negRiskABI.Methods[tt.method]
				if len(input) != 4+32 || string(input[:4]) != string(method.ID) || common.BytesToHash(input[4:]) != marketID {
					t.Fatalf("input = %s", call.Input)
				}
				encoded, err := method.Outputs.Pack(tt.value)
				if err != nil {
					t.Fatal(err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(request.ID) + `,"result":"` + hexutil.Encode(encoded) + `"}`))
			}))
			defer server.Close()

			client := NewClient("", WithChainID(PolygonChainID), WithRPCURL(server.URL))
			got, err := tt.call(context.Background(), client, marketID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Cmp(tt.value) != 0 {
				t.Fatalf("value = %s, want %s", got, tt.value)
			}
		})
	}
}

func TestNegRiskAdapterGettersValidateConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		client *Client
		market common.Hash
		want   string
	}{
		{name: "missing market", client: NewClient(""), want: "market id is required"},
		{name: "missing rpc", client: NewClient("", WithRPCURL("")), market: common.HexToHash("0x01"), want: "rpc url is required"},
		{name: "unsupported chain", client: NewClient("", WithChainID(80002)), market: common.HexToHash("0x01"), want: "unsupported chain id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.client.GetFeeBps(context.Background(), tt.market)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
