package relayer

import (
	"encoding/json"
	"testing"
)

func TestSubmitTransactionRequestWalletCreateJSON(t *testing.T) {
	req := SubmitTransactionRequest{
		Type: NonceTypeWalletCreate,
		From: "0x0000000000000000000000000000000000000001",
		To:   "0x00000000000Fb5C9ADea0298D729A0CB3823Cc07",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if raw["type"] != string(NonceTypeWalletCreate) {
		t.Fatalf("type = %v, want %s", raw["type"], NonceTypeWalletCreate)
	}
	if raw["from"] != req.From {
		t.Fatalf("from = %v, want %s", raw["from"], req.From)
	}
	if raw["to"] != req.To {
		t.Fatalf("to = %v, want %s", raw["to"], req.To)
	}

	mustNotHaveJSONField(t, raw, "proxyWallet")
	mustNotHaveJSONField(t, raw, "data")
	mustNotHaveJSONField(t, raw, "nonce")
	mustNotHaveJSONField(t, raw, "signature")
	mustNotHaveJSONField(t, raw, "signatureParams")
	mustNotHaveJSONField(t, raw, "depositWalletParams")
}

func TestSubmitTransactionRequestWalletBatchJSON(t *testing.T) {
	req := SubmitTransactionRequest{
		Type:      NonceTypeWallet,
		From:      "0x0000000000000000000000000000000000000001",
		To:        "0x00000000000Fb5C9ADea0298D729A0CB3823Cc07",
		Nonce:     "7",
		Signature: "0xabcdef",
		DepositWalletParams: &DepositWalletParams{
			DepositWallet: "0x0000000000000000000000000000000000000002",
			Deadline:      "1760000000",
			Calls: []DepositWalletCall{
				{
					Target: "0x0000000000000000000000000000000000000003",
					Value:  "0",
					Data:   "0x01020304",
				},
			},
		},
		Metadata: "deposit-wallet-batch",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if raw["type"] != string(NonceTypeWallet) {
		t.Fatalf("type = %v, want %s", raw["type"], NonceTypeWallet)
	}
	if raw["nonce"] != "7" {
		t.Fatalf("nonce = %v, want 7", raw["nonce"])
	}
	if raw["signature"] != "0xabcdef" {
		t.Fatalf("signature = %v, want 0xabcdef", raw["signature"])
	}

	params, ok := raw["depositWalletParams"].(map[string]any)
	if !ok {
		t.Fatalf("depositWalletParams = %T(%v), want object", raw["depositWalletParams"], raw["depositWalletParams"])
	}
	if params["depositWallet"] != req.DepositWalletParams.DepositWallet {
		t.Fatalf("depositWallet = %v, want %s", params["depositWallet"], req.DepositWalletParams.DepositWallet)
	}
	if params["deadline"] != req.DepositWalletParams.Deadline {
		t.Fatalf("deadline = %v, want %s", params["deadline"], req.DepositWalletParams.Deadline)
	}

	calls, ok := params["calls"].([]any)
	if !ok || len(calls) != 1 {
		t.Fatalf("calls = %T(%v), want one call", params["calls"], params["calls"])
	}

	call, ok := calls[0].(map[string]any)
	if !ok {
		t.Fatalf("call = %T(%v), want object", calls[0], calls[0])
	}
	if call["target"] != "0x0000000000000000000000000000000000000003" {
		t.Fatalf("target = %v", call["target"])
	}
	if call["value"] != "0" {
		t.Fatalf("value = %v", call["value"])
	}
	if call["data"] != "0x01020304" {
		t.Fatalf("data = %v", call["data"])
	}

	mustNotHaveJSONField(t, raw, "proxyWallet")
	mustNotHaveJSONField(t, raw, "data")
	mustNotHaveJSONField(t, raw, "signatureParams")
}

func mustNotHaveJSONField(t *testing.T, raw map[string]any, key string) {
	t.Helper()

	if v, ok := raw[key]; ok {
		t.Fatalf("%s should be omitted, got %v", key, v)
	}
}
