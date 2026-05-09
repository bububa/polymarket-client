package relayer

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestEncodeProxyTransactionDataIncludesSelectorAndEncodesTupleOrder(t *testing.T) {
	txs := []ProxyTransaction{
		{
			To:       "0x00000000000000000000000000000000000000aa",
			TypeCode: CallTypeCall,
			Value:    "123",
			Data:     "0x01020304",
		},
		{
			To:       "0x00000000000000000000000000000000000000bb",
			TypeCode: CallTypeDelegateCall,
			Value:    "",
			Data:     "0xdeadbeef",
		},
	}

	got, err := EncodeProxyTransactionData(txs)
	if err != nil {
		t.Fatalf("EncodeProxyTransactionData: %v", err)
	}

	if !strings.HasPrefix(got, proxySubmitTransactionSelector) {
		t.Fatalf("encoded data must start with selector %s, got %s", proxySubmitTransactionSelector, got)
	}

	selectorless := strings.TrimPrefix(got, proxySubmitTransactionSelector)
	if strings.HasPrefix(selectorless, proxySubmitTransactionSelector[2:]) {
		t.Fatalf("encoded data appears to contain duplicate selector prefix: %s", got)
	}

	expected, err := encodeExpectedProxyTransactionDataForTest()
	if err != nil {
		t.Fatalf("encode expected proxy transaction data: %v", err)
	}

	if got != expected {
		t.Fatalf("encoded proxy data mismatch:\n got:  %s\n want: %s", got, expected)
	}
}

func encodeExpectedProxyTransactionDataForTest() (string, error) {
	arrayT, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "typeCode", Type: "uint8"},
		{Name: "to", Type: "address"},
		{Name: "value", Type: "uint256"},
		{Name: "data", Type: "bytes"},
	})
	if err != nil {
		return "", err
	}

	type expectedProxyTuple struct {
		TypeCode uint8
		To       common.Address
		Value    *big.Int
		Data     []byte
	}

	args := abi.Arguments{{Type: arrayT}}

	encoded, err := args.Pack([]expectedProxyTuple{
		{
			TypeCode: 1,
			To:       common.HexToAddress("0x00000000000000000000000000000000000000aa"),
			Value:    big.NewInt(123),
			Data:     hexutil.MustDecode("0x01020304"),
		},
		{
			TypeCode: 2,
			To:       common.HexToAddress("0x00000000000000000000000000000000000000bb"),
			Value:    big.NewInt(0),
			Data:     hexutil.MustDecode("0xdeadbeef"),
		},
	})
	if err != nil {
		return "", err
	}

	encodedData := hexutil.Encode(encoded)
	return proxySubmitTransactionSelector + encodedData[2:], nil
}
