package combo

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/bububa/polymarket-client/clob"
	"github.com/bububa/polymarket-client/internal/polyauth"
	pmtypes "github.com/bububa/polymarket-client/shared"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const comboOrderProtocolVersion = "3"

type QuoteOptions struct {
	Price  string
	Size   string
	Source QuoteSource
}

type BuiltQuote struct {
	PriceE6     pmtypes.String
	SizeE6      pmtypes.String
	SignedOrder SignedOrder
}

type QuoteBuilder struct {
	client *Client
	now    func() time.Time
}

func NewQuoteBuilder(client *Client) *QuoteBuilder {
	return &QuoteBuilder{client: client, now: time.Now}
}

func (b *QuoteBuilder) BuildAndSign(request QuoteRequest, opts QuoteOptions) (*BuiltQuote, error) {
	if b == nil || b.client == nil {
		return nil, errors.New("combo: quote builder client is required")
	}
	if b.client.signer == nil {
		return nil, errors.New("combo: signer is required")
	}
	if err := validateOrderIdentity(b.client.identity, b.client.signer); err != nil {
		return nil, err
	}
	if request.Side != SideYes {
		return nil, errors.New("combo: only YES-side rfq requests are supported")
	}
	if _, err := NormalizeConditionID(string(request.ConditionID)); err != nil {
		return nil, err
	}
	price, err := decimalToE6(opts.Price)
	if err != nil {
		return nil, fmt.Errorf("combo: invalid price: %w", err)
	}
	one := big.NewInt(1_000_000)
	if price.Sign() <= 0 || price.Cmp(one) >= 0 {
		return nil, errors.New("combo: price must be greater than 0 and less than 1")
	}
	requested, err := positiveBigInt(string(request.RequestedSize.ValueE6))
	if err != nil {
		return nil, fmt.Errorf("combo: invalid requested size: %w", err)
	}
	maxSize := new(big.Int).Set(requested)
	if request.RequestedSize.Unit == RequestedSizeNotional {
		maxSize.Mul(maxSize, one).Quo(maxSize, price)
	} else if request.RequestedSize.Unit != RequestedSizeShares {
		return nil, errors.New("combo: invalid requested size unit")
	}
	size := new(big.Int).Set(maxSize)
	if opts.Size != "" {
		size, err = decimalToE6(opts.Size)
		if err != nil {
			return nil, fmt.Errorf("combo: invalid size: %w", err)
		}
	}
	if size.Sign() <= 0 || size.Cmp(maxSize) > 0 {
		return nil, errors.New("combo: quote size must be positive and not exceed the request")
	}
	source := opts.Source
	if source == "" {
		source = QuoteSourceCollateral
	}
	if source != QuoteSourceCollateral && source != QuoteSourceInventory {
		return nil, errors.New("combo: invalid quote source")
	}

	orderPrice := new(big.Int).Set(price)
	useComplement := (request.Direction == DirectionBuy && source == QuoteSourceCollateral) || (request.Direction == DirectionSell && source == QuoteSourceInventory)
	if useComplement {
		orderPrice.Sub(one, orderPrice)
	}
	var tokenID pmtypes.String
	if request.Direction == DirectionBuy {
		if source == QuoteSourceCollateral {
			tokenID = request.NoPositionID
		} else {
			tokenID = request.YesPositionID
		}
	} else if request.Direction == DirectionSell {
		if source == QuoteSourceCollateral {
			tokenID = request.YesPositionID
		} else {
			tokenID = request.NoPositionID
		}
	} else {
		return nil, errors.New("combo: invalid rfq direction")
	}

	orderSide := 0
	makerAmount := ceilMulDiv(orderPrice, size, one)
	takerAmount := new(big.Int).Set(size)
	if source == QuoteSourceInventory {
		orderSide = 1
		makerAmount.Set(size)
		takerAmount.Mul(orderPrice, size).Quo(takerAmount, one)
	}
	if makerAmount.Sign() <= 0 || takerAmount.Sign() <= 0 {
		return nil, errors.New("combo: quote produces a zero order amount")
	}
	salt, err := randomSalt()
	if err != nil {
		return nil, fmt.Errorf("combo: generate salt: %w", err)
	}
	identity := b.client.identity
	contracts, err := clob.Contracts(b.client.chainID)
	if err != nil {
		return nil, fmt.Errorf("combo: resolve contracts: %w", err)
	}
	if contracts.ComboExchange == (common.Address{}) {
		return nil, fmt.Errorf("combo: combo exchange is not configured for chain id %d", b.client.chainID)
	}
	exchange := contracts.ComboExchange
	order := SignedOrder{
		Salt: pmtypes.String(salt.String()), Maker: identity.MakerAddress, Signer: identity.SignerAddress,
		TokenID: tokenID, MakerAmount: pmtypes.String(makerAmount.String()), TakerAmount: pmtypes.String(takerAmount.String()),
		Side: orderSide, SignatureType: identity.SignatureType, Timestamp: pmtypes.String(strconv.FormatInt(b.now().Unix(), 10)),
		Metadata: ZeroBytes32, Builder: ZeroBytes32,
	}
	if identity.SignatureType == clob.SignatureTypePoly1271 {
		order.Signer = identity.MakerAddress
		order.Signature, err = signPoly1271(b.client.signer, b.client.chainID, exchange, common.HexToAddress(identity.MakerAddress), order)
	} else {
		typed := buildTypedData(b.client.chainID, exchange, order)
		order.Signature, err = polyauth.SignTypedData(b.client.signer, typed)
	}
	if err != nil {
		return nil, fmt.Errorf("combo: sign quote order: %w", err)
	}
	return &BuiltQuote{PriceE6: pmtypes.String(price.String()), SizeE6: pmtypes.String(size.String()), SignedOrder: order}, nil
}

func validateOrderIdentity(identity Identity, signer *polyauth.Signer) error {
	if !common.IsHexAddress(identity.MakerAddress) || !common.IsHexAddress(identity.SignerAddress) {
		return errors.New("combo: signer and maker addresses must be valid evm addresses")
	}
	if common.HexToAddress(identity.MakerAddress) == (common.Address{}) || common.HexToAddress(identity.SignerAddress) == (common.Address{}) {
		return errors.New("combo: signer and maker addresses must not be zero")
	}
	switch identity.SignatureType {
	case clob.SignatureTypeEOA:
		if !strings.EqualFold(identity.MakerAddress, identity.SignerAddress) {
			return errors.New("combo: EOA signer and maker addresses must match")
		}
		if !strings.EqualFold(identity.SignerAddress, signer.Address().Hex()) {
			return errors.New("combo: order signer address does not match cryptographic signer")
		}
	case clob.SignatureTypeProxy, clob.SignatureTypeGnosisSafe:
		if !strings.EqualFold(identity.SignerAddress, signer.Address().Hex()) {
			return errors.New("combo: order signer address does not match cryptographic signer")
		}
	case clob.SignatureTypePoly1271:
		if !strings.EqualFold(identity.SignerAddress, identity.MakerAddress) {
			return errors.New("combo: POLY_1271 signer and maker addresses must match")
		}
	default:
		return fmt.Errorf("combo: unsupported signature type %d", identity.SignatureType)
	}
	return nil
}

func decimalToE6(value string) (*big.Int, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || strings.HasPrefix(value, "-") {
		return nil, errors.New("value must be a positive decimal")
	}
	if len(parts) == 1 {
		parts = append(parts, "")
	}
	if len(parts[1]) > 6 {
		return nil, errors.New("value must have at most 6 decimal places")
	}
	whole, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, errors.New("invalid decimal")
	}
	fraction := parts[1] + strings.Repeat("0", 6-len(parts[1]))
	frac, ok := new(big.Int).SetString(fraction, 10)
	if !ok {
		frac = new(big.Int)
	}
	return whole.Mul(whole, big.NewInt(1_000_000)).Add(whole, frac), nil
}

func positiveBigInt(value string) (*big.Int, error) {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok || n.Sign() <= 0 {
		return nil, errors.New("value must be a positive integer")
	}
	return n, nil
}
func ceilMulDiv(a, b, d *big.Int) *big.Int {
	n := new(big.Int).Mul(a, b)
	n.Add(n, new(big.Int).Sub(d, big.NewInt(1)))
	return n.Quo(n, d)
}
func randomSalt() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).SetUint64(^uint64(0)>>1))
}

func buildTypedData(chainID int64, exchange common.Address, order SignedOrder) apitypes.TypedData {
	return apitypes.TypedData{Types: apitypes.Types{
		"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}, {Name: "chainId", Type: "uint256"}, {Name: "verifyingContract", Type: "address"}},
		"Order":        {{Name: "salt", Type: "uint256"}, {Name: "maker", Type: "address"}, {Name: "signer", Type: "address"}, {Name: "tokenId", Type: "uint256"}, {Name: "makerAmount", Type: "uint256"}, {Name: "takerAmount", Type: "uint256"}, {Name: "side", Type: "uint8"}, {Name: "signatureType", Type: "uint8"}, {Name: "timestamp", Type: "uint256"}, {Name: "metadata", Type: "bytes32"}, {Name: "builder", Type: "bytes32"}},
	}, PrimaryType: "Order", Domain: apitypes.TypedDataDomain{Name: "Polymarket CTF Exchange", Version: comboOrderProtocolVersion, ChainId: ethmath.NewHexOrDecimal256(chainID), VerifyingContract: exchange.Hex()}, Message: apitypes.TypedDataMessage{
		"salt": string(order.Salt), "maker": order.Maker, "signer": order.Signer, "tokenId": string(order.TokenID), "makerAmount": string(order.MakerAmount), "takerAmount": string(order.TakerAmount), "side": strconv.Itoa(order.Side), "signatureType": strconv.Itoa(int(order.SignatureType)), "timestamp": string(order.Timestamp), "metadata": order.Metadata, "builder": order.Builder,
	}}
}

const orderType = "Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"
const typedSignType = "TypedDataSign(Order contents,string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)" + orderType

func signPoly1271(signer *polyauth.Signer, chainID int64, exchange, wallet common.Address, order SignedOrder) (string, error) {
	contents, err := orderContentsHash(order)
	if err != nil {
		return "", err
	}
	domain := buildTypedData(chainID, exchange, order)
	domainSeparator, err := domain.HashStruct("EIP712Domain", domain.Domain.Map())
	if err != nil {
		return "", err
	}
	typeHash := crypto.Keccak256Hash([]byte(typedSignType))
	nameHash := crypto.Keccak256Hash([]byte("DepositWallet"))
	versionHash := crypto.Keccak256Hash([]byte("1"))
	packed, err := abiPack([]string{"bytes32", "bytes32", "bytes32", "bytes32", "uint256", "address", "bytes32"}, []any{typeHash, contents, nameHash, versionHash, big.NewInt(chainID), wallet, common.Hash{}})
	if err != nil {
		return "", err
	}
	structHash := crypto.Keccak256Hash(packed)
	digest := crypto.Keccak256(append(append([]byte{0x19, 0x01}, domainSeparator...), structHash.Bytes()...))
	innerHex, err := polyauth.SignHash(signer, digest)
	if err != nil {
		return "", err
	}
	inner, err := hexutil.Decode(innerHex)
	if err != nil {
		return "", err
	}
	wrapped := append(append(append(append([]byte{}, inner...), domainSeparator...), contents.Bytes()...), []byte(orderType)...)
	wrapped = append(wrapped, byte(len(orderType)>>8), byte(len(orderType)))
	return "0x" + hex.EncodeToString(wrapped), nil
}

func orderContentsHash(order SignedOrder) (common.Hash, error) {
	vals := []string{string(order.Salt), string(order.TokenID), string(order.MakerAmount), string(order.TakerAmount), string(order.Timestamp)}
	ints := make([]*big.Int, len(vals))
	for i, v := range vals {
		n, ok := new(big.Int).SetString(v, 10)
		if !ok || n.Sign() < 0 {
			return common.Hash{}, fmt.Errorf("invalid uint256 %q", v)
		}
		ints[i] = n
	}
	metadata := common.HexToHash(order.Metadata)
	builder := common.HexToHash(order.Builder)
	packed, err := abiPack([]string{"bytes32", "uint256", "address", "address", "uint256", "uint256", "uint256", "uint8", "uint8", "uint256", "bytes32", "bytes32"}, []any{crypto.Keccak256Hash([]byte(orderType)), ints[0], common.HexToAddress(order.Maker), common.HexToAddress(order.Signer), ints[1], ints[2], ints[3], uint8(order.Side), uint8(order.SignatureType), ints[4], metadata, builder})
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(packed), nil
}

func abiPack(types []string, values []any) ([]byte, error) {
	args := make(abi.Arguments, len(types))
	for i, name := range types {
		typ, err := abi.NewType(name, "", nil)
		if err != nil {
			return nil, err
		}
		args[i] = abi.Argument{Type: typ}
	}
	return args.Pack(values...)
}
