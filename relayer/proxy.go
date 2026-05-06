package relayer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/bububa/polymarket-client/internal/polyauth"
)

const (
	proxyFactoryPolygon = "0xaB45c5A4B0c941a2F231C04C3f49182e1A254052"
	relayHubPolygon     = "0xD216153c06E857cD7f72665E0aF1d7D82172F494"
)

const (
	proxyBaseGasPerTx       = 150000
	proxyRelayHubPadding    = 3450000
	proxyOverheadBuffer     = 450000
	proxyIntrinsicCost      = 30000
	proxyMinExecutionBuffer = 500000
)

// CalculateProxyGasLimit mirrors the official proxy relayer example gas limit heuristic.
func CalculateProxyGasLimit(transactionCount int) string {
	if transactionCount <= 0 {
		transactionCount = 1
	}

	txGas := transactionCount * proxyBaseGasPerTx
	relayerWillSend := txGas + proxyRelayHubPadding
	maxSignable := relayerWillSend - proxyIntrinsicCost - proxyOverheadBuffer
	executionNeeds := txGas + proxyMinExecutionBuffer

	gasLimit := min(max(executionNeeds, 3000000), maxSignable)
	return fmt.Sprintf("%d", gasLimit)
}

// EncodeProxyTransactionData encodes a PROXY transaction batch.
// It corresponds to the official TS client path where generic Transaction[]
// is converted to ProxyTransaction[] and passed through encodeProxyTransactionData.
func EncodeProxyTransactionData(txs []ProxyTransaction) (string, error) {
	if len(txs) == 0 {
		return "", errors.New("relayer: no proxy transactions")
	}

	arrayT, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "typeCode", Type: "uint8"},
		{Name: "to", Type: "address"},
		{Name: "value", Type: "uint256"},
		{Name: "data", Type: "bytes"},
	})
	if err != nil {
		return "", err
	}

	type proxyTuple struct {
		To       common.Address
		TypeCode uint8
		Data     []byte
		Value    *big.Int
	}

	values := make([]proxyTuple, 0, len(txs))
	for _, tx := range txs {
		if !common.IsHexAddress(tx.To) {
			return "", fmt.Errorf("relayer: proxy tx to must be a valid address")
		}

		typeCode, err := parseCallType(tx.TypeCode)
		if err != nil {
			return "", err
		}

		data, err := hexutil.Decode(tx.Data)
		if err != nil {
			return "", fmt.Errorf("relayer: decode proxy tx data: %w", err)
		}

		value := strings.TrimSpace(tx.Value)
		if value == "" {
			value = "0"
		}
		valueInt, ok := new(big.Int).SetString(value, 10)
		if !ok || valueInt.Sign() < 0 {
			return "", fmt.Errorf("relayer: proxy tx value must be a non-negative decimal integer")
		}

		values = append(values, proxyTuple{
			To:       common.HexToAddress(tx.To),
			TypeCode: typeCode,
			Data:     data,
			Value:    valueInt,
		})
	}

	args := abi.Arguments{{Type: arrayT}}
	encoded, err := args.Pack(values)
	if err != nil {
		return "", fmt.Errorf("relayer: encode proxy transaction data: %w", err)
	}
	encodedData := hexutil.Encode(encoded)
	encodedData = "0x34ee9791" + encodedData[2:]
	return encodedData, nil
}

// ProxySubmitTransactionRequest builds and signs a PROXY relayer submit request.
func (c *Client) ProxySubmitTransactionRequest(
	ctx context.Context,
	signer *polyauth.Signer,
	req *ProxySubmitTransactionArgs,
	out *SubmitTransactionRequest,
) error {
	if out == nil {
		return errors.New("relayer: nil submit transaction output")
	}
	if signer == nil {
		return errors.New("relayer: signer is required")
	}

	from := strings.TrimSpace(req.From)
	if from == "" {
		from = signer.Address().Hex()
	}
	if !common.IsHexAddress(from) {
		return fmt.Errorf("relayer: from must be a valid address")
	}
	from = common.HexToAddress(from).Hex()

	proxyWallet := strings.TrimSpace(req.ProxyWallet)
	if proxyWallet == "" {
		return errors.New("relayer: proxy wallet is required")
	}
	if !common.IsHexAddress(proxyWallet) {
		return fmt.Errorf("relayer: proxy wallet must be a valid address")
	}
	proxyWallet = common.HexToAddress(proxyWallet).Hex()

	data := strings.TrimSpace(req.Data)
	if data == "" {
		return errors.New("relayer: data is required")
	}
	if _, err := hexutil.Decode(data); err != nil {
		return fmt.Errorf("relayer: data must be 0x-prefixed hex: %w", err)
	}

	relayPayload := NonceResponse{Address: from}
	if err := c.GetRelayPayload(ctx, &relayPayload, NonceTypeProxy); err != nil {
		return err
	}
	if relayPayload.Address == "" {
		return errors.New("relayer: empty relay address")
	}
	if !common.IsHexAddress(relayPayload.Address) {
		return fmt.Errorf("relayer: relay address must be valid")
	}
	if relayPayload.Nonce == "" {
		return errors.New("relayer: empty relay nonce")
	}

	gasLimit := strings.TrimSpace(req.GasLimit)
	if gasLimit == "" {
		gasLimit = CalculateProxyGasLimit(1)
	}

	const (
		relayerFee = "0"
		gasPrice   = "0"
	)

	to := common.HexToAddress(proxyFactoryPolygon)
	relayHub := common.HexToAddress(relayHubPolygon)
	relay := common.HexToAddress(relayPayload.Address)

	hash, err := createProxyRelayHash(
		common.HexToAddress(from),
		to,
		data,
		relayerFee,
		gasPrice,
		gasLimit,
		relayPayload.Nonce.String(),
		relayHub,
		relay,
	)
	if err != nil {
		return err
	}

	signature, err := signPersonalHash(signer, hash)
	if err != nil {
		return err
	}

	*out = SubmitTransactionRequest{
		From:        from,
		To:          to.Hex(),
		ProxyWallet: proxyWallet,
		Data:        data,
		Nonce:       relayPayload.Nonce.String(),
		Signature:   signature,
		SignatureParams: &SignatureParams{
			GasPrice:   gasPrice,
			GasLimit:   gasLimit,
			RelayerFee: relayerFee,
			RelayHub:   relayHub.Hex(),
			Relay:      relay.Hex(),
		},
		Type:     NonceTypeProxy,
		Metadata: req.Metadata,
		Value:    "0",
	}

	return nil
}

func createProxyRelayHash(
	from common.Address,
	to common.Address,
	dataHex string,
	relayerFee string,
	gasPrice string,
	gasLimit string,
	nonce string,
	relayHub common.Address,
	relay common.Address,
) ([]byte, error) {
	data, err := hexutil.Decode(dataHex)
	if err != nil {
		return nil, fmt.Errorf("relayer: decode data: %w", err)
	}

	relayerFeeInt, err := parseDecimalUint256("relayerFee", relayerFee)
	if err != nil {
		return nil, err
	}
	gasPriceInt, err := parseDecimalUint256("gasPrice", gasPrice)
	if err != nil {
		return nil, err
	}
	gasLimitInt, err := parseDecimalUint256("gasLimit", gasLimit)
	if err != nil {
		return nil, err
	}
	nonceInt, err := parseDecimalUint256("nonce", nonce)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 0, 3+20+20+len(data)+32*4+20+20)
	buf = append(buf, []byte("rlx:")...)
	buf = append(buf, from.Bytes()...)
	buf = append(buf, to.Bytes()...)
	buf = append(buf, data...)
	buf = append(buf, leftPad32(relayerFeeInt)...)
	buf = append(buf, leftPad32(gasPriceInt)...)
	buf = append(buf, leftPad32(gasLimitInt)...)
	buf = append(buf, leftPad32(nonceInt)...)
	buf = append(buf, relayHub.Bytes()...)
	buf = append(buf, relay.Bytes()...)

	return crypto.Keccak256(buf), nil
}

func signPersonalHash(signer *polyauth.Signer, hash []byte) (string, error) {
	personalHash := accounts.TextHash(hash)

	sig, err := crypto.Sign(personalHash, signer.PrivateKey())
	if err != nil {
		return "", fmt.Errorf("relayer: sign proxy relay hash: %w", err)
	}

	// ethers/viem signMessage returns 27/28. go-ethereum crypto.Sign returns 0/1.
	if sig[64] < 27 {
		sig[64] += 27
	}

	return hexutil.Encode(sig), nil
}

func parseDecimalUint256(name, value string) (*big.Int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("relayer: %s is required", name)
	}
	n, ok := new(big.Int).SetString(value, 10)
	if !ok || n.Sign() < 0 {
		return nil, fmt.Errorf("relayer: %s must be a non-negative decimal integer", name)
	}
	if n.BitLen() > 256 {
		return nil, fmt.Errorf("relayer: %s overflows uint256", name)
	}
	return n, nil
}

func leftPad32(n *big.Int) []byte {
	out := make([]byte, 32)
	n.FillBytes(out)
	return out
}

func parseCallType(v CallType) (uint8, error) {
	switch v {
	case "", CallTypeCall:
		return 1, nil
	case CallTypeInvalid:
		return 0, nil
	case CallTypeDelegateCall:
		return 2, nil
	default:
		return 0, fmt.Errorf("relayer: unsupported proxy call type %q", v)
	}
}
