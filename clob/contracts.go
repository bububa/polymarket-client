package clob

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// ContractAddressCTFExchange is the CTF Exchange V2 contract.
	//
	// Used for normal CLOB order EIP-712 domain separation and settlement of
	// standard, non-negative-risk Polymarket markets.
	ContractAddressCTFExchange = "0xE111180000d2663C0091e4f400237545B87B996B"

	// ContractAddressNegRiskCTFExchange is the negative-risk CTF Exchange V2 contract.
	//
	// Used as the EIP-712 verifying contract when signing orders for negative-risk
	// markets.
	ContractAddressNegRiskCTFExchange = "0xe2222d279d744050d28e00520010520000310F59"

	// ContractAddressNegRiskAdapter is the negative-risk adapter contract.
	//
	// Used by neg-risk conversion/redemption helpers, especially
	// BuildRedeemNegRiskTx and RedeemNegRiskWithDepositWallet.
	ContractAddressNegRiskAdapter = "0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"

	// ContractAddressConditionalTokens is the Gnosis Conditional Tokens Framework
	// contract used by Polymarket.
	//
	// Used for CTF splitPosition, mergePositions, redeemPositions, collection ID,
	// and ERC-1155 conditional-token flows.
	ContractAddressConditionalTokens = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
)

const (
	// ContractAddressPUSD is the pUSD collateral token proxy.
	//
	// Use this address as the collateral token in CTF split/merge/redeem calldata.
	// This is also the token address relevant for collateral balance/allowance
	// checks and deposit-wallet approvals.
	ContractAddressPUSD = "0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB"

	// ContractAddressPUSDImplementation is the pUSD implementation contract.
	//
	// This is mostly useful for audits, debugging, and contract verification. SDK
	// callers should usually interact with ContractAddressPUSD, the proxy address.
	ContractAddressPUSDImplementation = "0x6bBCef9f7ef3B6C592c99e0f206a0DE94Ad0925f"

	// ContractAddressCollateralOnramp is the collateral onramp contract.
	//
	// Used by Polymarket's collateral onboarding/deposit infrastructure. Most CLOB
	// trading flows do not call this contract directly.
	ContractAddressCollateralOnramp = "0x93070a847efEf7F70739046A929D47a521F5B8ee"

	// ContractAddressCollateralOfframp is the collateral offramp contract.
	//
	// Used by Polymarket's collateral withdrawal/offboarding infrastructure. Most
	// CLOB trading flows do not call this contract directly.
	ContractAddressCollateralOfframp = "0x2957922Eb93258b93368531d39fAcCA3B4dC5854"

	// ContractAddressPermissionedRamp is the permissioned ramp contract.
	//
	// Used by Polymarket ramp infrastructure. It is included for completeness and
	// should not be used for normal order signing or CTF split/merge/redeem calls.
	ContractAddressPermissionedRamp = "0xebC2459Ec962869ca4c0bd1E06368272732BCb08"

	// ContractAddressCTFCollateralAdapter is the collateral adapter for standard
	// CTF flows.
	//
	// Included for integrations that need to inspect or interact with collateral
	// adapter flows directly. Normal CTF split/merge/redeem helpers use
	// ConditionalTokens and pUSD instead.
	ContractAddressCTFCollateralAdapter = "0xAdA100Db00Ca00073811820692005400218FcE1f"

	// ContractAddressNegRiskCTFCollateralAdapter is the collateral adapter for
	// negative-risk CTF flows.
	//
	// Included for integrations that need low-level negative-risk collateral
	// adapter access. Normal neg-risk redemption helpers use NegRiskAdapter.
	ContractAddressNegRiskCTFCollateralAdapter = "0xadA2005600Dec949baf300f4C6120000bDB6eAab"
)

const (
	// ContractAddressGnosisSafeFactory is Polymarket's Gnosis Safe factory.
	//
	// Used by legacy SAFE wallet onboarding and SAFE relayer flows. New API users
	// should prefer deposit wallets when applicable.
	ContractAddressGnosisSafeFactory = "0xaacfeea03eb1561c4e67d661e40682bd20e3541b"

	// ContractAddressPolymarketProxyFactory is Polymarket's proxy wallet factory.
	//
	// Used by legacy PROXY wallet onboarding and PROXY relayer flows.
	ContractAddressPolymarketProxyFactory = "0xaB45c5A4B0c941a2F231C04C3f49182e1A254052"

	// ContractAddressDepositWalletFactory is the deposit wallet factory for Polygon
	// mainnet.
	//
	// Used as the "to" address for relayer WALLET-CREATE requests and as the
	// factory address for deterministic deposit wallet derivation. This address is
	// documented in the deposit wallet migration guide, not in the resources/contracts
	// page.
	ContractAddressDepositWalletFactory = "0x00000000000Fb5C9ADea0298D729A0CB3823Cc07"
)

const (
	// ContractAddressUMAAdapter is Polymarket's UMA adapter.
	//
	// Used as the oracle address when deriving CTF condition IDs from question IDs.
	ContractAddressUMAAdapter = "0x6A9D222616C90FcA5754cd1333cFD9b7fb6a4F74"

	// ContractAddressUMAOptimisticOracle is UMA's Optimistic Oracle contract used
	// in Polymarket resolution infrastructure.
	//
	// Included for integrations that need resolution/audit context. Most SDK callers
	// should use UMAAdapter for condition ID derivation.
	ContractAddressUMAOptimisticOracle = "0xCB1822859cEF82Cd2Eb4E6276C7916e692995130"
)

// ContractConfig contains the on-chain contract addresses used by Polymarket on a chain.
type ContractConfig struct {
	// Exchange is the CTF Exchange V2 contract used for standard market order
	// signing and settlement.
	Exchange common.Address

	// NegRiskExchange is the CTF Exchange V2 contract used for negative-risk
	// market order signing and settlement.
	NegRiskExchange common.Address

	// NegRiskAdapter is the adapter used for negative-risk conversion and
	// redemption flows.
	NegRiskAdapter common.Address

	// ConditionalTokens is the Gnosis Conditional Tokens Framework contract used
	// by Polymarket outcome tokens.
	ConditionalTokens common.Address

	// Collateral is the pUSD collateral token proxy used by CTF split, merge,
	// redeem, balance, allowance, and approval flows.
	Collateral common.Address

	// CollateralImplementation is the pUSD implementation contract behind the
	// proxy. Most callers should use Collateral instead.
	CollateralImplementation common.Address

	// CollateralOnramp is Polymarket's collateral onramp contract.
	CollateralOnramp common.Address

	// CollateralOfframp is Polymarket's collateral offramp contract.
	CollateralOfframp common.Address

	// PermissionedRamp is Polymarket's permissioned ramp contract.
	PermissionedRamp common.Address

	// CTFCollateralAdapter is the collateral adapter for standard CTF flows.
	CTFCollateralAdapter common.Address

	// NegRiskCTFCollateralAdapter is the collateral adapter for negative-risk CTF
	// flows.
	NegRiskCTFCollateralAdapter common.Address

	// GnosisSafeFactory is the factory used for legacy SAFE wallet deployments.
	GnosisSafeFactory common.Address

	// ProxyFactory is Polymarket's legacy proxy wallet factory.
	ProxyFactory common.Address

	// DepositWalletFactory is the factory used to deploy deterministic deposit
	// wallets through relayer WALLET-CREATE requests.
	DepositWalletFactory common.Address

	// UMAAdapter is the oracle adapter used when deriving condition IDs from
	// question IDs.
	UMAAdapter common.Address

	// UMAOptimisticOracle is the UMA Optimistic Oracle used by resolution
	// infrastructure.
	UMAOptimisticOracle common.Address
}

var contractConfigs = map[int64]ContractConfig{
	PolygonChainID: {
		Exchange:                    common.HexToAddress(ContractAddressCTFExchange),
		NegRiskExchange:             common.HexToAddress(ContractAddressNegRiskCTFExchange),
		NegRiskAdapter:              common.HexToAddress(ContractAddressNegRiskAdapter),
		ConditionalTokens:           common.HexToAddress(ContractAddressConditionalTokens),
		Collateral:                  common.HexToAddress(ContractAddressPUSD),
		CollateralImplementation:    common.HexToAddress(ContractAddressPUSDImplementation),
		CollateralOnramp:            common.HexToAddress(ContractAddressCollateralOnramp),
		CollateralOfframp:           common.HexToAddress(ContractAddressCollateralOfframp),
		PermissionedRamp:            common.HexToAddress(ContractAddressPermissionedRamp),
		CTFCollateralAdapter:        common.HexToAddress(ContractAddressCTFCollateralAdapter),
		NegRiskCTFCollateralAdapter: common.HexToAddress(ContractAddressNegRiskCTFCollateralAdapter),
		GnosisSafeFactory:           common.HexToAddress(ContractAddressGnosisSafeFactory),
		ProxyFactory:                common.HexToAddress(ContractAddressPolymarketProxyFactory),
		DepositWalletFactory:        common.HexToAddress(ContractAddressDepositWalletFactory),
		UMAAdapter:                  common.HexToAddress(ContractAddressUMAAdapter),
		UMAOptimisticOracle:         common.HexToAddress(ContractAddressUMAOptimisticOracle),
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
