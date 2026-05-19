package clob

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type depositWalletAddressGoldenFile struct {
	Kind    string                             `json:"kind"`
	Source  string                             `json:"source"`
	Vectors []depositWalletAddressGoldenVector `json:"vectors"`
}

type depositWalletAddressGoldenVector struct {
	Name           string `json:"name"`
	ChainID        int64  `json:"chainId"`
	Owner          string `json:"owner"`
	Factory        string `json:"factory"`
	Implementation string `json:"implementation"`
	DepositWallet  string `json:"depositWallet"`
}

func TestDeriveDepositWalletAddress_GoldenBuilderRelayerClient(t *testing.T) {
	golden := loadDepositWalletAddressGolden(t)
	if golden.Kind != "builder_relayer_client_deposit_wallet_address" {
		t.Fatalf("golden kind = %q", golden.Kind)
	}
	if len(golden.Vectors) == 0 {
		t.Fatal("golden vectors are empty")
	}

	for _, vector := range golden.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			owner := common.HexToAddress(vector.Owner)
			factory := common.HexToAddress(vector.Factory)
			implementation := common.HexToAddress(vector.Implementation)

			got, err := deriveDepositWalletAddress(owner, factory, implementation)
			if err != nil {
				t.Fatalf("deriveDepositWalletAddress: %v", err)
			}
			if !strings.EqualFold(got.Hex(), vector.DepositWallet) {
				t.Fatalf("explicit deposit wallet = %s, want %s", got.Hex(), vector.DepositWallet)
			}

			got, err = DeriveDepositWalletAddress(owner, vector.ChainID)
			if err != nil {
				t.Fatalf("DeriveDepositWalletAddress: %v", err)
			}
			if !strings.EqualFold(got.Hex(), vector.DepositWallet) {
				t.Fatalf("chain config deposit wallet = %s, want %s", got.Hex(), vector.DepositWallet)
			}
		})
	}
}

func loadDepositWalletAddressGolden(t *testing.T) depositWalletAddressGoldenFile {
	t.Helper()

	path := filepath.Join("..", "testdata", "golden", "builder-relayer-client", "deposit_wallet_address.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read deposit wallet address golden file %s: %v", path, err)
	}
	var out depositWalletAddressGoldenFile
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode deposit wallet address golden file %s: %v", path, err)
	}
	return out
}

func TestClientDeriveDepositWalletAddress(t *testing.T) {
	signer, err := ParsePrivateKey(depositWalletTestPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}
	client := NewClient("", WithSigner(signer), WithChainID(PolygonChainID))

	got, err := client.DeriveDepositWalletAddress()
	if err != nil {
		t.Fatalf("DeriveDepositWalletAddress: %v", err)
	}
	want, err := DeriveDepositWalletAddress(signer.Address(), PolygonChainID)
	if err != nil {
		t.Fatalf("package DeriveDepositWalletAddress: %v", err)
	}
	if got != want {
		t.Fatalf("client deposit wallet = %s, want %s", got.Hex(), want.Hex())
	}
}

func TestClientDeriveDepositWalletAddressRejectsMissingSigner(t *testing.T) {
	client := NewClient("", WithChainID(PolygonChainID))
	_, err := client.DeriveDepositWalletAddress()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "signer is required") {
		t.Fatalf("error = %q, want signer error", err.Error())
	}
}
