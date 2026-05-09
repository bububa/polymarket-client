package clob

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/bububa/polymarket-client/internal/polyauth"
)

const (
	depositWalletDomainName    = "DepositWallet"
	depositWalletDomainVersion = "1"

	// Must exactly match py-clob-client-v2 ORDER_TYPE_STRING.
	depositWalletOrderTypeString = "Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"

	// Must exactly match py-clob-client-v2 SOLADY_TYPE_STRING.
	depositWalletTypedDataSignTypeString = "TypedDataSign(Order contents,string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)" + depositWalletOrderTypeString

	// Must exactly match py-clob-client-v2 DOMAIN_TYPE_STRING.
	depositWalletDomainTypeString = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
)

var (
	depositWalletOrderTypeHash         = crypto.Keccak256Hash([]byte(depositWalletOrderTypeString))
	depositWalletTypedDataSignTypeHash = crypto.Keccak256Hash([]byte(depositWalletTypedDataSignTypeString))
	depositWalletDomainTypeHash        = crypto.Keccak256Hash([]byte(depositWalletDomainTypeString))

	depositWalletNameHash    = crypto.Keccak256Hash([]byte(depositWalletDomainName))
	depositWalletVersionHash = crypto.Keccak256Hash([]byte(depositWalletDomainVersion))
)

// SignDepositWalletOrder signs a CLOB v2 order for a deposit wallet.
//
// This is the POLY_1271 / signatureType=3 path. The order maker and signer
// fields must both be the deposit wallet address, while the cryptographic
// signer is the deposit wallet owner/session signer.
//
// Required output shape:
//   - order.SignatureType = SignatureTypePoly1271
//   - order.Maker = depositWallet
//   - order.Signer = depositWallet
//   - order.Signature = ERC-7739 wrapped signature
func (c *Client) SignDepositWalletOrder(order *SignedOrder, depositWallet common.Address, opts ...SignOrderOption) error {
	if c.auth.Signer == nil {
		return errors.New("polymarket: signer is required to sign deposit wallet order")
	}
	return SignDepositWalletOrder(c.auth.Signer, c.auth.ChainID, order, depositWallet, opts...)
}

// SignDepositWalletOrder signs a CLOB v2 POLY_1271 order with an owner/session signer.
func SignDepositWalletOrder(
	signer *polyauth.Signer,
	chainID int64,
	order *SignedOrder,
	depositWallet common.Address,
	opts ...SignOrderOption,
) error {
	if signer == nil {
		return errors.New("polymarket: signer is required to sign deposit wallet order")
	}
	if order == nil {
		return errors.New("polymarket: order is nil")
	}
	if depositWallet == (common.Address{}) {
		return errors.New("polymarket: deposit wallet address is required")
	}

	config := signOrderConfig{now: time.Now}
	for _, opt := range opts {
		opt(&config)
	}

	contracts, err := Contracts(chainID)
	if err != nil {
		return err
	}

	verifyingContract := contracts.Exchange
	if config.negRisk {
		verifyingContract = contracts.NegRiskExchange
	}
	if config.verifyingContract != (common.Address{}) {
		verifyingContract = config.verifyingContract
	}

	if err := prepareDepositWalletOrderForSigning(order, depositWallet, config); err != nil {
		return err
	}

	signature, err := buildDepositWalletOrderSignature(signer, chainID, verifyingContract, depositWallet, *order)
	if err != nil {
		return err
	}
	order.Signature = signature
	return nil
}

func prepareDepositWalletOrderForSigning(order *SignedOrder, depositWallet common.Address, config signOrderConfig) error {
	depositWalletHex := depositWallet.Hex()

	if order.SignatureType != SignatureTypePoly1271 {
		return fmt.Errorf("polymarket: deposit wallet orders require signatureType %d, got %d", SignatureTypePoly1271, order.SignatureType)
	}

	if order.Maker == "" {
		order.Maker = depositWalletHex
	} else if !common.IsHexAddress(order.Maker) {
		return fmt.Errorf("polymarket: invalid maker address %q", order.Maker)
	} else {
		order.Maker = common.HexToAddress(order.Maker).Hex()
	}
	if !equalAddress(order.Maker, depositWalletHex) {
		return fmt.Errorf("polymarket: deposit wallet order maker %s does not match deposit wallet %s", order.Maker, depositWalletHex)
	}

	if order.Signer == "" {
		order.Signer = depositWalletHex
	} else if !common.IsHexAddress(order.Signer) {
		return fmt.Errorf("polymarket: invalid signer address %q", order.Signer)
	} else {
		order.Signer = common.HexToAddress(order.Signer).Hex()
	}
	if !equalAddress(order.Signer, depositWalletHex) {
		return fmt.Errorf("polymarket: deposit wallet order signer %s does not match deposit wallet %s", order.Signer, depositWalletHex)
	}

	if order.Metadata == "" {
		order.Metadata = ZeroBytes32
	} else if err := ValidateBytes32Hex("metadata", order.Metadata); err != nil {
		return err
	}

	if order.Builder == "" {
		order.Builder = ZeroBytes32
	} else if err := ValidateBytes32Hex("builder", order.Builder); err != nil {
		return err
	}

	if err := validateUint256String("tokenId", order.TokenID, true, false); err != nil {
		return err
	}
	if err := validateUint256String("makerAmount", order.MakerAmount, true, true); err != nil {
		return err
	}
	if err := validateUint256String("takerAmount", order.TakerAmount, true, true); err != nil {
		return err
	}

	if order.Side != Buy && order.Side != Sell {
		return fmt.Errorf("polymarket: invalid side %q", order.Side)
	}

	if order.Salt == 0 {
		salt := config.salt
		if salt == nil {
			generated, err := randomOrderSalt()
			if err != nil {
				return err
			}
			salt = generated
		}
		order.Salt = Int64(salt.Int64())
	} else if order.Salt < 0 {
		return fmt.Errorf("polymarket: salt must be non-negative")
	}

	if order.Timestamp == "" {
		order.Timestamp = String(strconv.FormatInt(config.now().UnixMilli(), 10))
	} else if err := validateUint256String("timestamp", order.Timestamp, true, true); err != nil {
		return err
	}

	if order.Expiration == "" {
		order.Expiration = String("0")
	}

	return nil
}

func buildDepositWalletOrderSignature(
	signer *polyauth.Signer,
	chainID int64,
	exchange common.Address,
	depositWallet common.Address,
	order SignedOrder,
) (string, error) {
	contentsHash, err := depositWalletOrderContentsHash(order)
	if err != nil {
		return "", err
	}

	appDomainSeparator, err := ctfExchangeV2DomainSeparator(chainID, exchange)
	if err != nil {
		return "", err
	}

	typedDataSignStructHash, err := depositWalletTypedDataSignStructHash(chainID, depositWallet, contentsHash)
	if err != nil {
		return "", err
	}

	digestInput := make([]byte, 0, 66)
	digestInput = append(digestInput, 0x19, 0x01)
	digestInput = append(digestInput, appDomainSeparator.Bytes()...)
	digestInput = append(digestInput, typedDataSignStructHash.Bytes()...)
	digest := crypto.Keccak256(digestInput)

	innerSignatureHex, err := polyauth.SignHash(signer, digest)
	if err != nil {
		return "", fmt.Errorf("polymarket: sign deposit wallet order: %w", err)
	}

	innerSignature, err := hexutil.Decode(innerSignatureHex)
	if err != nil {
		return "", fmt.Errorf("polymarket: decode deposit wallet signature: %w", err)
	}
	if len(innerSignature) != 65 {
		return "", fmt.Errorf("polymarket: deposit wallet signature length = %d, want 65", len(innerSignature))
	}

	typeLen := len(depositWalletOrderTypeString)
	if typeLen > 0xffff {
		return "", fmt.Errorf("polymarket: order type string too long: %d", typeLen)
	}

	wrapped := make([]byte, 0, 65+32+32+typeLen+2)
	wrapped = append(wrapped, innerSignature...)
	wrapped = append(wrapped, appDomainSeparator.Bytes()...)
	wrapped = append(wrapped, contentsHash.Bytes()...)
	wrapped = append(wrapped, []byte(depositWalletOrderTypeString)...)
	wrapped = append(wrapped, byte(typeLen>>8), byte(typeLen))

	return "0x" + hex.EncodeToString(wrapped), nil
}

func depositWalletOrderContentsHash(order SignedOrder) (common.Hash, error) {
	salt := big.NewInt(int64(order.Salt))

	tokenID, err := parseUint256Big("tokenId", order.TokenID.String())
	if err != nil {
		return common.Hash{}, err
	}
	makerAmount, err := parseUint256Big("makerAmount", order.MakerAmount.String())
	if err != nil {
		return common.Hash{}, err
	}
	takerAmount, err := parseUint256Big("takerAmount", order.TakerAmount.String())
	if err != nil {
		return common.Hash{}, err
	}
	timestamp, err := parseUint256Big("timestamp", order.Timestamp.String())
	if err != nil {
		return common.Hash{}, err
	}

	metadata, err := bytes32FromHex("metadata", order.Metadata)
	if err != nil {
		return common.Hash{}, err
	}
	builder, err := bytes32FromHex("builder", order.Builder)
	if err != nil {
		return common.Hash{}, err
	}

	encoded, err := abiEncodeDepositWalletOrderContents(
		depositWalletOrderTypeHash,
		salt,
		common.HexToAddress(order.Maker),
		common.HexToAddress(order.Signer),
		tokenID,
		makerAmount,
		takerAmount,
		uint8(orderSideValue(order.Side)),
		uint8(order.SignatureType),
		timestamp,
		metadata,
		builder,
	)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

func ctfExchangeV2DomainSeparator(chainID int64, exchange common.Address) (common.Hash, error) {
	encoded, err := abiEncodeDepositWalletDomain(
		depositWalletDomainTypeHash,
		crypto.Keccak256Hash([]byte(orderProtocolName)),
		crypto.Keccak256Hash([]byte(orderProtocolVersion)),
		big.NewInt(chainID),
		exchange,
	)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

func depositWalletTypedDataSignStructHash(chainID int64, depositWallet common.Address, contentsHash common.Hash) (common.Hash, error) {
	encoded, err := abiEncodeDepositWalletTypedDataSign(
		depositWalletTypedDataSignTypeHash,
		contentsHash,
		depositWalletNameHash,
		depositWalletVersionHash,
		big.NewInt(chainID),
		depositWallet,
		common.Hash{},
	)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

// BuildDepositWalletOrderTypedData returns the normal CLOB v2 Order typed data.
//
// This is useful for debugging and golden-vector comparison. Deposit wallet
// orders are not signed directly with this typed data; they are signed through
// the ERC-7739 wrapper implemented by SignDepositWalletOrder.
func BuildDepositWalletOrderTypedData(chainID int64, verifyingContract common.Address, order SignedOrder) apitypes.TypedData {
	return buildOrderTypedData(chainID, verifyingContract, order)
}

func abiEncodeDepositWalletOrderContents(
	typeHash common.Hash,
	salt *big.Int,
	maker common.Address,
	signer common.Address,
	tokenID *big.Int,
	makerAmount *big.Int,
	takerAmount *big.Int,
	side uint8,
	signatureType uint8,
	timestamp *big.Int,
	metadata common.Hash,
	builder common.Hash,
) ([]byte, error) {
	return abiEncodeByTypes(
		[]string{
			"bytes32",
			"uint256",
			"address",
			"address",
			"uint256",
			"uint256",
			"uint256",
			"uint8",
			"uint8",
			"uint256",
			"bytes32",
			"bytes32",
		},
		[]any{
			hashBytes32(typeHash),
			salt,
			maker,
			signer,
			tokenID,
			makerAmount,
			takerAmount,
			side,
			signatureType,
			timestamp,
			hashBytes32(metadata),
			hashBytes32(builder),
		},
	)
}

func abiEncodeDepositWalletDomain(
	typeHash common.Hash,
	nameHash common.Hash,
	versionHash common.Hash,
	chainID *big.Int,
	verifyingContract common.Address,
) ([]byte, error) {
	return abiEncodeByTypes(
		[]string{
			"bytes32",
			"bytes32",
			"bytes32",
			"uint256",
			"address",
		},
		[]any{
			hashBytes32(typeHash),
			hashBytes32(nameHash),
			hashBytes32(versionHash),
			chainID,
			verifyingContract,
		},
	)
}

func abiEncodeDepositWalletTypedDataSign(
	typeHash common.Hash,
	contentsHash common.Hash,
	nameHash common.Hash,
	versionHash common.Hash,
	chainID *big.Int,
	verifyingContract common.Address,
	salt common.Hash,
) ([]byte, error) {
	return abiEncodeByTypes(
		[]string{
			"bytes32",
			"bytes32",
			"bytes32",
			"bytes32",
			"uint256",
			"address",
			"bytes32",
		},
		[]any{
			hashBytes32(typeHash),
			hashBytes32(contentsHash),
			hashBytes32(nameHash),
			hashBytes32(versionHash),
			chainID,
			verifyingContract,
			hashBytes32(salt),
		},
	)
}

func abiEncodeByTypes(typeNames []string, values []any) ([]byte, error) {
	if len(typeNames) != len(values) {
		return nil, fmt.Errorf("polymarket: abi type/value length mismatch: %d types, %d values", len(typeNames), len(values))
	}

	args := make(abi.Arguments, 0, len(typeNames))
	for _, typeName := range typeNames {
		t, err := abi.NewType(typeName, "", nil)
		if err != nil {
			return nil, fmt.Errorf("polymarket: create abi type %q: %w", typeName, err)
		}
		args = append(args, abi.Argument{Type: t})
	}

	encoded, err := args.Pack(values...)
	if err != nil {
		return nil, fmt.Errorf("polymarket: abi encode: %w", err)
	}
	return encoded, nil
}

func parseUint256Big(name, value string) (*big.Int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("polymarket: %s is required", name)
	}

	n, ok := new(big.Int).SetString(value, 10)
	if !ok || n.Sign() < 0 {
		return nil, fmt.Errorf("polymarket: %s must be a non-negative decimal integer", name)
	}
	if n.BitLen() > 256 {
		return nil, fmt.Errorf("polymarket: %s overflows uint256", name)
	}

	return n, nil
}

func bytes32FromHex(name, value string) (common.Hash, error) {
	value = strings.TrimSpace(value)
	if err := ValidateBytes32Hex(name, value); err != nil {
		return common.Hash{}, err
	}

	raw, err := hexutil.Decode(value)
	if err != nil {
		return common.Hash{}, fmt.Errorf("polymarket: decode %s: %w", name, err)
	}
	if len(raw) != 32 {
		return common.Hash{}, fmt.Errorf("polymarket: %s must be 32 bytes, got %d", name, len(raw))
	}

	return common.BytesToHash(raw), nil
}

func hashBytes32(h common.Hash) [32]byte {
	var out [32]byte
	copy(out[:], h.Bytes())
	return out
}
