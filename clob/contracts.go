package clob

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// ContractConfig contains the on-chain contract addresses used by Polymarket on a chain.
type ContractConfig struct {
	// Exchange is the CTF Exchange V2 contract used for standard market settlement.
	Exchange common.Address
	// NegRiskExchange is the CTF Exchange V2 contract used for neg-risk market settlement.
	NegRiskExchange common.Address
	// NegRiskAdapter is the adapter used for neg-risk conversion and redemption flows.
	NegRiskAdapter common.Address
	// Collateral is the pUSD collateral token used by CTF split, merge, and redeem calls.
	Collateral common.Address
	// ConditionalTokens is the Gnosis Conditional Tokens contract used by Polymarket.
	ConditionalTokens common.Address
	// UMAAdapter is the oracle address used when deriving condition IDs from question IDs.
	UMAAdapter common.Address
	// CtfCollateralAdapter is the adapter used for CTF collateral operations.
	CtfCollateralAdapter common.Address
}

var contractConfigs = map[int64]ContractConfig{
	PolygonChainID: {
		Exchange:             common.HexToAddress("0xE111180000d2663C0091e4f400237545B87B996B"),
		NegRiskExchange:      common.HexToAddress("0xe2222d279d744050d28e00520010520000310F59"),
		NegRiskAdapter:       common.HexToAddress("0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"),
		Collateral:           common.HexToAddress("0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB"),
		ConditionalTokens:    common.HexToAddress("0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"),
		UMAAdapter:           common.HexToAddress("0x6A9D222616C90FcA5754cd1333cFD9b7fb6a4F74"),
		CtfCollateralAdapter: common.HexToAddress("0xAdA100Db00Ca00073811820692005400218FcE1f"),
	},
}

// Contracts returns the known Polymarket contract addresses for chainID.
func Contracts(chainID int64) (ContractConfig, error) {
	config, ok := contractConfigs[chainID]
	if !ok {
		return ContractConfig{}, fmt.Errorf("polymarket: unsupported chain id %d", chainID)
	}
	return config, nil
}
