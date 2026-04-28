package clob

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/bububa/polymarket-client/internal/polyauth"
	"github.com/ethereum/go-ethereum/common"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	orderProtocolName    = "Polymarket CTF Exchange"
	orderProtocolVersion = "2"
)

// SignOrderOption customizes how SignOrder builds the EIP-712 domain and defaults.
type SignOrderOption func(*signOrderConfig)

type signOrderConfig struct {
	negRisk           bool
	verifyingContract common.Address
	now               func() time.Time
	salt              *big.Int
}

// WithSignOrderNegRisk signs against the neg-risk exchange contract.
func WithSignOrderNegRisk(enabled bool) SignOrderOption {
	return func(c *signOrderConfig) { c.negRisk = enabled }
}

// WithSignOrderVerifyingContract signs against a custom exchange contract address.
func WithSignOrderVerifyingContract(address common.Address) SignOrderOption {
	return func(c *signOrderConfig) { c.verifyingContract = address }
}

// WithSignOrderTime sets the timestamp used when the order has no timestamp.
func WithSignOrderTime(now time.Time) SignOrderOption {
	return func(c *signOrderConfig) { c.now = func() time.Time { return now } }
}

// WithSignOrderSalt sets the salt used when the order has no salt.
func WithSignOrderSalt(salt *big.Int) SignOrderOption {
	return func(c *signOrderConfig) {
		if salt != nil {
			c.salt = new(big.Int).Set(salt)
		}
	}
}

// SignOrder fills missing v2 order defaults and signs order with the client's signer.
func (c *Client) SignOrder(order *SignedOrder, opts ...SignOrderOption) error {
	if c.auth.Signer == nil {
		return errors.New("polymarket: signer is required to sign order")
	}
	return SignOrder(c.auth.Signer, c.auth.ChainID, order, opts...)
}

// SignOrder fills missing v2 order defaults and writes the EIP-712 signature into order.
func SignOrder(signer *polyauth.Signer, chainID int64, order *SignedOrder, opts ...SignOrderOption) error {
	if signer == nil {
		return errors.New("polymarket: signer is required to sign order")
	}
	if order == nil {
		return errors.New("polymarket: order is nil")
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

	if err := prepareOrderForSigning(signer, order, config); err != nil {
		return err
	}
	signature, err := polyauth.SignTypedData(signer, buildOrderTypedData(chainID, verifyingContract, *order))
	if err != nil {
		return err
	}
	order.Signature = signature
	return nil
}

func prepareOrderForSigning(signer *polyauth.Signer, order *SignedOrder, config signOrderConfig) error {
	signerAddress := signer.Address().Hex()
	if order.Maker == "" {
		order.Maker = signerAddress
	}
	if order.Signer == "" {
		order.Signer = signerAddress
	}
	if !common.IsHexAddress(order.Maker) {
		return fmt.Errorf("polymarket: invalid maker address %q", order.Maker)
	}
	if !common.IsHexAddress(order.Signer) {
		return fmt.Errorf("polymarket: invalid signer address %q", order.Signer)
	}
	if !equalAddress(order.Signer, signerAddress) {
		return fmt.Errorf("polymarket: order signer %s does not match signer %s", order.Signer, signerAddress)
	}
	if order.TokenID == "" {
		return errors.New("polymarket: tokenId is required")
	}
	if order.MakerAmount == "" {
		return errors.New("polymarket: makerAmount is required")
	}
	if order.TakerAmount == "" {
		return errors.New("polymarket: takerAmount is required")
	}
	if order.Side != Buy && order.Side != Sell {
		return fmt.Errorf("polymarket: invalid side %q", order.Side)
	}
	if order.Salt == "" {
		salt := config.salt
		if salt == nil {
			generated, err := randomUint256()
			if err != nil {
				return err
			}
			salt = generated
		}
		order.Salt = String(salt.String())
	}
	if order.Timestamp == "" {
		order.Timestamp = String(strconv.FormatInt(config.now().UnixMilli(), 10))
	}
	if order.Metadata == "" {
		order.Metadata = ZeroBytes32
	}
	if order.Builder == "" {
		order.Builder = ZeroBytes32
	}
	return nil
}

func buildOrderTypedData(chainID int64, verifyingContract common.Address, order SignedOrder) apitypes.TypedData {
	return apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Order": {
				{Name: "salt", Type: "uint256"},
				{Name: "maker", Type: "address"},
				{Name: "signer", Type: "address"},
				{Name: "tokenId", Type: "uint256"},
				{Name: "makerAmount", Type: "uint256"},
				{Name: "takerAmount", Type: "uint256"},
				{Name: "side", Type: "uint8"},
				{Name: "signatureType", Type: "uint8"},
				{Name: "timestamp", Type: "uint256"},
				{Name: "metadata", Type: "bytes32"},
				{Name: "builder", Type: "bytes32"},
			},
		},
		PrimaryType: "Order",
		Domain: apitypes.TypedDataDomain{
			Name:              orderProtocolName,
			Version:           orderProtocolVersion,
			ChainId:           ethmath.NewHexOrDecimal256(chainID),
			VerifyingContract: verifyingContract.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"salt":          order.Salt.String(),
			"maker":         order.Maker,
			"signer":        order.Signer,
			"tokenId":       order.TokenID.String(),
			"makerAmount":   order.MakerAmount.String(),
			"takerAmount":   order.TakerAmount.String(),
			"side":          strconv.Itoa(orderSideValue(order.Side)),
			"signatureType": strconv.Itoa(int(order.SignatureType)),
			"timestamp":     order.Timestamp.String(),
			"metadata":      order.Metadata,
			"builder":       order.Builder,
		},
	}
}

func orderSideValue(side Side) int {
	if side == Sell {
		return 1
	}
	return 0
}

func randomUint256() (*big.Int, error) {
	max := new(big.Int).Lsh(big.NewInt(1), 256)
	salt, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, fmt.Errorf("polymarket: generate order salt: %w", err)
	}
	return salt, nil
}

func equalAddress(a, b string) bool {
	return common.HexToAddress(a) == common.HexToAddress(b)
}
