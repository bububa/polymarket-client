package clob

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	depositWalletERC1967Const1 = common.FromHex("0xcc3735a920a3ca505d382bbc545af43d6000803e6038573d6000fd5b3d6000f3")
	depositWalletERC1967Const2 = common.FromHex("0x5155f3363d3d373d3d363d7f360894a13ba1a3210667c828492db98dca3e2076")
	depositWalletERC1967Prefix = new(big.Int).SetBytes(common.FromHex("0x61003d3d8160233d3973"))
)

// DeriveDepositWalletAddress derives Polymarket's deterministic deposit wallet
// address for owner on chainID.
func DeriveDepositWalletAddress(owner common.Address, chainID int64) (common.Address, error) {
	if owner == (common.Address{}) {
		return common.Address{}, errors.New("polymarket: deposit wallet owner is required")
	}

	contracts, err := Contracts(chainID)
	if err != nil {
		return common.Address{}, err
	}
	if contracts.DepositWalletFactory == (common.Address{}) {
		return common.Address{}, fmt.Errorf("polymarket: deposit wallet factory is not configured for chain %d", chainID)
	}
	if contracts.DepositWalletImplementation == (common.Address{}) {
		return common.Address{}, fmt.Errorf("polymarket: deposit wallet implementation is not configured for chain %d", chainID)
	}

	return deriveDepositWalletAddress(owner, contracts.DepositWalletFactory, contracts.DepositWalletImplementation)
}

// DeriveDepositWalletAddress derives the deterministic deposit wallet address
// for the client's signer and chain ID.
func (c *Client) DeriveDepositWalletAddress() (common.Address, error) {
	if c == nil || c.auth.Signer == nil {
		return common.Address{}, errors.New("polymarket: signer is required")
	}
	return DeriveDepositWalletAddress(c.auth.Signer.Address(), c.auth.ChainID)
}

func deriveDepositWalletAddress(owner, factory, implementation common.Address) (common.Address, error) {
	addressT, _ := abi.NewType("address", "", nil)
	bytes32T, _ := abi.NewType("bytes32", "", nil)
	args := abi.Arguments{{Type: addressT}, {Type: bytes32T}}

	var walletID [32]byte
	copy(walletID[12:], owner.Bytes())

	encodedArgs, err := args.Pack(factory, walletID)
	if err != nil {
		return common.Address{}, fmt.Errorf("polymarket: encode deposit wallet args: %w", err)
	}

	salt := crypto.Keccak256Hash(encodedArgs)
	bytecodeHash, err := depositWalletInitCodeHashERC1967(implementation, encodedArgs)
	if err != nil {
		return common.Address{}, err
	}

	return create2Address(factory, salt, bytecodeHash), nil
}

func depositWalletInitCodeHashERC1967(implementation common.Address, args []byte) (common.Hash, error) {
	if len(args) > 0xffff {
		return common.Hash{}, fmt.Errorf("polymarket: deposit wallet args too long: %d", len(args))
	}

	combined := new(big.Int).Set(depositWalletERC1967Prefix)
	combined.Add(combined, new(big.Int).Lsh(big.NewInt(int64(len(args))), 56))

	code := leftPadBytes(combined.Bytes(), 10)
	code = append(code, implementation.Bytes()...)
	code = append(code, 0x60, 0x09)
	code = append(code, depositWalletERC1967Const2...)
	code = append(code, depositWalletERC1967Const1...)
	code = append(code, args...)

	return crypto.Keccak256Hash(code), nil
}

func create2Address(factory common.Address, salt, bytecodeHash common.Hash) common.Address {
	buf := make([]byte, 0, 1+20+32+32)
	buf = append(buf, 0xff)
	buf = append(buf, factory.Bytes()...)
	buf = append(buf, salt.Bytes()...)
	buf = append(buf, bytecodeHash.Bytes()...)

	return common.BytesToAddress(crypto.Keccak256(buf)[12:])
}

func leftPadBytes(in []byte, size int) []byte {
	if len(in) >= size {
		return in
	}
	out := make([]byte, size)
	copy(out[size-len(in):], in)
	return out
}
