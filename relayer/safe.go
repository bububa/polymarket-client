package relayer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/bububa/polymarket-client/internal/polyauth"
)

const (
	zeroAddress = "0x0000000000000000000000000000000000000000"

	safeInitCodeHash = "0x2bce2127ff07fb632d16c8347c4ebf501f4841168bed00d9e6ef715ddb6fcecf"

	safeFactoryPolygon   = "0xaacFeEa03eb1561C4e67d661e40682Bd20E3541b"
	safeMultisendPolygon = "0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761"
)

var (
	safeTxTypeHash = crypto.Keccak256Hash([]byte("SafeTx(address to,uint256 value,bytes data,uint8 operation,uint256 safeTxGas,uint256 baseGas,uint256 gasPrice,address gasToken,address refundReceiver,uint256 nonce)"))

	// Gnosis Safe domain used by current Safe contracts.
	safeDomainTypeHash = crypto.Keccak256Hash([]byte("EIP712Domain(uint256 chainId,address verifyingContract)"))
)

// SafeSubmitTransactionRequest builds and signs a SAFE relayer submit request.
func (c *Client) SafeSubmitTransactionRequest(
	ctx context.Context,
	signer *polyauth.Signer,
	req *SafeSubmitTransactionArgs,
	out *SubmitTransactionRequest,
) error {
	if out == nil {
		return errors.New("relayer: nil submit transaction output")
	}
	if signer == nil {
		return errors.New("relayer: signer is required")
	}
	if req.ChainID == 0 {
		return errors.New("relayer: chain id is required")
	}
	if len(req.Transactions) == 0 {
		return errors.New("relayer: at least one safe transaction is required")
	}

	from := strings.TrimSpace(req.From)
	if from == "" {
		from = signer.Address().Hex()
	}
	if !common.IsHexAddress(from) {
		return fmt.Errorf("relayer: from must be a valid address")
	}
	from = common.HexToAddress(from).Hex()

	safeAddress := strings.TrimSpace(req.ProxyWallet)
	if safeAddress == "" {
		derived, err := DeriveSafeAddress(common.HexToAddress(from), req.ChainID)
		if err != nil {
			return err
		}
		safeAddress = derived.Hex()
	}
	if !common.IsHexAddress(safeAddress) {
		return fmt.Errorf("relayer: safe proxy wallet must be a valid address")
	}
	safeAddress = common.HexToAddress(safeAddress).Hex()

	relayPayload := NonceResponse{Address: from}
	if err := c.GetRelayPayload(ctx, &relayPayload, NonceTypeSafe); err != nil {
		return err
	}
	if relayPayload.Nonce == "" {
		return errors.New("relayer: empty safe relayer nonce")
	}

	tx, err := aggregateSafeTransactions(req.Transactions, req.ChainID)
	if err != nil {
		return err
	}

	const (
		safeTxGas = "0"
		baseGas   = "0"
		gasPrice  = "0"
	)

	gasToken := zeroAddress
	refundReceiver := zeroAddress

	structHash, err := createSafeStructHash(
		req.ChainID,
		common.HexToAddress(safeAddress),
		tx,
		safeTxGas,
		baseGas,
		gasPrice,
		gasToken,
		refundReceiver,
		relayPayload.Nonce.String(),
	)
	if err != nil {
		return err
	}

	digest, err := createSafeTypedDataDigest(req.ChainID, common.HexToAddress(safeAddress), structHash)
	if err != nil {
		return err
	}

	sig, err := crypto.Sign(digest, signer.PrivateKey())
	if err != nil {
		return fmt.Errorf("relayer: sign safe transaction hash: %w", err)
	}

	packedSig, err := packSafeSignature(sig)
	if err != nil {
		return err
	}

	*out = SubmitTransactionRequest{
		Type:        NonceTypeSafe,
		From:        from,
		To:          common.HexToAddress(tx.To).Hex(),
		ProxyWallet: safeAddress,
		Data:        normalizeHex(tx.Data),
		Nonce:       relayPayload.Nonce.String(),
		Signature:   packedSig,
		SignatureParams: SignatureParams{
			GasPrice:       gasPrice,
			Operation:      fmt.Sprintf("%d", tx.Operation),
			SafeTxGas:      safeTxGas,
			BaseGas:        baseGas,
			GasToken:       gasToken,
			RefundReceiver: refundReceiver,
		},
		Metadata: req.Metadata,
		Value:    tx.Value,
	}
	return nil
}

func aggregateSafeTransactions(txs []SafeTransaction, chainID int64) (SafeTransaction, error) {
	if len(txs) == 0 {
		return SafeTransaction{}, errors.New("relayer: no safe transactions")
	}
	if len(txs) == 1 {
		tx := txs[0]
		if tx.Value == "" {
			tx.Value = "0"
		}
		if tx.Data == "" {
			tx.Data = "0x"
		}
		if !common.IsHexAddress(tx.To) {
			return SafeTransaction{}, fmt.Errorf("relayer: safe tx to must be a valid address")
		}
		tx.To = common.HexToAddress(tx.To).Hex()
		tx.Data = normalizeHex(tx.Data)
		return tx, nil
	}

	config, err := relayerContractConfig(chainID)
	if err != nil {
		return SafeTransaction{}, err
	}
	return CreateSafeMultiSendTransaction(txs, config.SafeMultisend)
}

// CreateSafeMultiSendTransaction creates a Safe DelegateCall transaction to MultiSend.
func CreateSafeMultiSendTransaction(txs []SafeTransaction, safeMultisend common.Address) (SafeTransaction, error) {
	if len(txs) == 0 {
		return SafeTransaction{}, errors.New("relayer: no safe transactions")
	}

	encodedTxs := make([]byte, 0)
	for _, tx := range txs {
		if !common.IsHexAddress(tx.To) {
			return SafeTransaction{}, fmt.Errorf("relayer: safe tx to must be a valid address")
		}

		data, err := hexutil.Decode(normalizeHex(tx.Data))
		if err != nil {
			return SafeTransaction{}, fmt.Errorf("relayer: decode safe tx data: %w", err)
		}

		value := tx.Value
		if strings.TrimSpace(value) == "" {
			value = "0"
		}
		valueInt, err := parseSafeUint256("value", value)
		if err != nil {
			return SafeTransaction{}, err
		}

		dataLen := big.NewInt(int64(len(data)))

		encodedTxs = append(encodedTxs, byte(tx.Operation))
		encodedTxs = append(encodedTxs, common.HexToAddress(tx.To).Bytes()...)
		encodedTxs = append(encodedTxs, safeLeftPad32(valueInt)...)
		encodedTxs = append(encodedTxs, safeLeftPad32(dataLen)...)
		encodedTxs = append(encodedTxs, data...)
	}

	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return SafeTransaction{}, err
	}
	args := abi.Arguments{{Type: bytesType}}
	payload, err := args.Pack(encodedTxs)
	if err != nil {
		return SafeTransaction{}, fmt.Errorf("relayer: encode multisend bytes: %w", err)
	}

	// multiSend(bytes) selector.
	fullData := append([]byte{0x8d, 0x80, 0xff, 0x0a}, payload...)

	return SafeTransaction{
		To:        safeMultisend.Hex(),
		Operation: OperationDelegateCall,
		Data:      hexutil.Encode(fullData),
		Value:     "0",
	}, nil
}

func createSafeStructHash(
	chainID int64,
	safe common.Address,
	tx SafeTransaction,
	safeTxGas string,
	baseGas string,
	gasPrice string,
	gasToken string,
	refundReceiver string,
	nonce string,
) (common.Hash, error) {
	_ = chainID
	_ = safe

	if !common.IsHexAddress(tx.To) {
		return common.Hash{}, fmt.Errorf("relayer: safe tx to must be a valid address")
	}

	value, err := parseSafeUint256("value", defaultZero(tx.Value))
	if err != nil {
		return common.Hash{}, err
	}
	data, err := hexutil.Decode(normalizeHex(tx.Data))
	if err != nil {
		return common.Hash{}, fmt.Errorf("relayer: decode safe tx data: %w", err)
	}
	safeTxGasInt, err := parseSafeUint256("safeTxGas", safeTxGas)
	if err != nil {
		return common.Hash{}, err
	}
	baseGasInt, err := parseSafeUint256("baseGas", baseGas)
	if err != nil {
		return common.Hash{}, err
	}
	gasPriceInt, err := parseSafeUint256("gasPrice", gasPrice)
	if err != nil {
		return common.Hash{}, err
	}
	if !common.IsHexAddress(gasToken) {
		return common.Hash{}, fmt.Errorf("relayer: gasToken must be a valid address")
	}
	if !common.IsHexAddress(refundReceiver) {
		return common.Hash{}, fmt.Errorf("relayer: refundReceiver must be a valid address")
	}
	nonceInt, err := parseSafeUint256("nonce", nonce)
	if err != nil {
		return common.Hash{}, err
	}

	uint8T, _ := abi.NewType("uint8", "", nil)
	uint256T, _ := abi.NewType("uint256", "", nil)
	addressT, _ := abi.NewType("address", "", nil)
	bytes32T, _ := abi.NewType("bytes32", "", nil)

	args := abi.Arguments{
		{Type: bytes32T},
		{Type: addressT},
		{Type: uint256T},
		{Type: bytes32T},
		{Type: uint8T},
		{Type: uint256T},
		{Type: uint256T},
		{Type: uint256T},
		{Type: addressT},
		{Type: addressT},
		{Type: uint256T},
	}

	encoded, err := args.Pack(
		safeTxTypeHash,
		common.HexToAddress(tx.To),
		value,
		crypto.Keccak256Hash(data),
		uint8(tx.Operation),
		safeTxGasInt,
		baseGasInt,
		gasPriceInt,
		common.HexToAddress(gasToken),
		common.HexToAddress(refundReceiver),
		nonceInt,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("relayer: encode safe struct hash: %w", err)
	}

	return crypto.Keccak256Hash(encoded), nil
}

func createSafeTypedDataDigest(chainID int64, safe common.Address, structHash common.Hash) ([]byte, error) {
	uint256T, _ := abi.NewType("uint256", "", nil)
	addressT, _ := abi.NewType("address", "", nil)
	bytes32T, _ := abi.NewType("bytes32", "", nil)

	args := abi.Arguments{
		{Type: bytes32T},
		{Type: uint256T},
		{Type: addressT},
	}

	domainEncoded, err := args.Pack(
		safeDomainTypeHash,
		big.NewInt(chainID),
		safe,
	)
	if err != nil {
		return nil, fmt.Errorf("relayer: encode safe domain separator: %w", err)
	}

	domainSeparator := crypto.Keccak256Hash(domainEncoded)

	data := make([]byte, 0, 2+32+32)
	data = append(data, 0x19, 0x01)
	data = append(data, domainSeparator.Bytes()...)
	data = append(data, structHash.Bytes()...)

	return crypto.Keccak256(data), nil
}

func packSafeSignature(sig []byte) (string, error) {
	if len(sig) != 65 {
		return "", fmt.Errorf("relayer: invalid signature length %d", len(sig))
	}

	r := new(big.Int).SetBytes(sig[0:32])
	s := new(big.Int).SetBytes(sig[32:64])
	vRaw := sig[64]

	var v byte
	switch vRaw {
	case 0, 1:
		v = vRaw + 31
	case 27, 28:
		v = vRaw + 4
	default:
		return "", fmt.Errorf("relayer: invalid signature v %d", vRaw)
	}

	out := make([]byte, 0, 65)
	out = append(out, safeLeftPad32(r)...)
	out = append(out, safeLeftPad32(s)...)
	out = append(out, v)

	return hexutil.Encode(out), nil
}

func DeriveSafeAddress(owner common.Address, chainID int64) (common.Address, error) {
	config, err := relayerContractConfig(chainID)
	if err != nil {
		return common.Address{}, err
	}

	addressT, _ := abi.NewType("address", "", nil)
	args := abi.Arguments{{Type: addressT}}
	encoded, err := args.Pack(owner)
	if err != nil {
		return common.Address{}, fmt.Errorf("relayer: encode safe owner salt: %w", err)
	}

	salt := crypto.Keccak256Hash(encoded)
	return getCreate2Address(common.HexToHash(safeInitCodeHash), config.SafeFactory, salt), nil
}

func getCreate2Address(bytecodeHash common.Hash, factory common.Address, salt common.Hash) common.Address {
	buf := make([]byte, 0, 1+20+32+32)
	buf = append(buf, 0xff)
	buf = append(buf, factory.Bytes()...)
	buf = append(buf, salt.Bytes()...)
	buf = append(buf, bytecodeHash.Bytes()...)

	return common.BytesToAddress(crypto.Keccak256(buf)[12:])
}

type relayerContracts struct {
	SafeFactory   common.Address
	SafeMultisend common.Address
}

func relayerContractConfig(chainID int64) (relayerContracts, error) {
	switch chainID {
	case 137:
		return relayerContracts{
			SafeFactory:   common.HexToAddress(safeFactoryPolygon),
			SafeMultisend: common.HexToAddress(safeMultisendPolygon),
		}, nil
	default:
		return relayerContracts{}, fmt.Errorf("relayer: unsupported chain id %d", chainID)
	}
}

func normalizeHex(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "0x"
	}
	if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
		return v
	}
	return "0x" + v
}

func defaultZero(v string) string {
	if strings.TrimSpace(v) == "" {
		return "0"
	}
	return strings.TrimSpace(v)
}

func parseSafeUint256(name, value string) (*big.Int, error) {
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

func safeLeftPad32(n *big.Int) []byte {
	out := make([]byte, 32)
	n.FillBytes(out)
	return out
}
