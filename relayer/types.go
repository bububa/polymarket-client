package relayer

import pmtypes "github.com/bububa/polymarket-client/shared"

// NonceType identifies the relayer nonce domain to query.
type NonceType string

const (
	// NonceTypeProxy selects the proxy-wallet nonce.
	NonceTypeProxy NonceType = "PROXY"
	// NonceTypeSafe selects the Safe-wallet nonce.
	NonceTypeSafe NonceType = "SAFE"
)

type CallType string

const (
	CallTypeInvalid      CallType = "0"
	CallTypeCall         CallType = "1"
	CallTypeDelegateCall CallType = "2"
)

type OperationType uint8

const (
	OperationCall         OperationType = 0
	OperationDelegateCall OperationType = 1
)

// Credentials contains Relayer API-key authentication headers.
type Credentials struct {
	// APIKey is the relayer API key.
	APIKey string
	// Address is the owner address for APIKey.
	Address string
}

// BuilderCredentials contains builder-key authentication material for relayer requests.
type BuilderCredentials struct {
	// APIKey is the builder API key.
	APIKey string
	// Secret is the URL-safe base64 builder API secret.
	Secret string
	// Passphrase is the builder API passphrase.
	Passphrase string
}

// SignatureParams contains Safe transaction signature parameters.
type SignatureParams struct {
	// Existing SAFE fields.
	// GasPrice is the Safe gas price parameter.
	GasPrice string `json:"gasPrice,omitempty"`
	// Operation is the Safe operation parameter.
	Operation string `json:"operation,omitempty"`
	// SafeTxGas is the Safe transaction gas parameter.
	SafeTxGas string `json:"safeTxnGas,omitempty"`
	// BaseGas is the Safe base gas parameter.
	BaseGas string `json:"baseGas,omitempty"`
	// GasToken is the token used for gas refunds.
	GasToken string `json:"gasToken,omitempty"`
	// RefundReceiver is the address receiving gas refunds.
	RefundReceiver string `json:"refundReceiver,omitempty"`

	// PROXY fields.
	GasLimit   string `json:"gasLimit,omitempty"`
	RelayerFee string `json:"relayerFee,omitempty"`
	RelayHub   string `json:"relayHub,omitempty"`
	Relay      string `json:"relay,omitempty"`
}

// SubmitTransactionRequest is the request body for POST /submit.
type SubmitTransactionRequest struct {
	// From is the signer address.
	From string `json:"from"`
	// To is the target contract address.
	To string `json:"to"`
	// ProxyWallet is the user's Polymarket proxy wallet.
	ProxyWallet string `json:"proxyWallet"`
	// Data is 0x-prefixed encoded transaction calldata.
	Data string `json:"data"`
	// Nonce is the relayer transaction nonce.
	Nonce string `json:"nonce"`
	// Signature is the 0x-prefixed transaction signature.
	Signature string `json:"signature"`
	// SignatureParams are Safe transaction parameters.
	SignatureParams SignatureParams `json:"signatureParams"`
	// Type is the transaction type, typically SAFE or PROXY.
	Type NonceType `json:"type"`
	// Metadata is optional caller-provided transaction metadata.
	Metadata string `json:"metadata,omitempty"`
	// Value is the native token value for the call when required.
	Value string `json:"value,omitempty"`
}

// SubmitTransactionResponse is returned immediately after a transaction is accepted.
type SubmitTransactionResponse struct {
	// TransactionID is the relayer transaction identifier.
	TransactionID string `json:"transactionID"`
	// State is the current relayer state.
	State string `json:"state"`
}

// Transaction describes a relayer transaction.
type Transaction struct {
	// TransactionID is the relayer transaction identifier.
	TransactionID string `json:"transactionID"`
	// State is the current relayer state.
	State string `json:"state"`
	// TransactionHash is the on-chain hash after broadcast.
	TransactionHash string `json:"transactionHash"`
	// From is the signer address.
	From string `json:"from"`
	// To is the target contract address.
	To string `json:"to"`
	// ProxyWallet is the user's Polymarket proxy wallet.
	ProxyWallet string `json:"proxyWallet"`
	// Data is the 0x-prefixed calldata submitted to the relayer.
	Data string `json:"data"`
	// Nonce is the transaction nonce.
	Nonce pmtypes.String `json:"nonce"`
	// Value is the transaction value.
	Value pmtypes.String `json:"value"`
	// Signature is the 0x-prefixed transaction signature.
	Signature string `json:"signature"`
	// Type is the transaction type.
	Type NonceType `json:"type"`
	// Owner is the transaction owner.
	Owner string `json:"owner"`
	// Metadata is the transaction metadata.
	Metadata string `json:"metadata"`
	// CreatedAt is the creation timestamp.
	CreatedAt pmtypes.Time `json:"createdAt"`
	// UpdatedAt is the last update timestamp.
	UpdatedAt pmtypes.Time `json:"updatedAt"`
}

// NonceResponse is returned by nonce endpoints.
type NonceResponse struct {
	// Nonce is the current relayer nonce.
	Nonce pmtypes.String `json:"nonce"`
	// Address is the associated relayer or user address when present.
	Address string `json:"address,omitempty"`
}

// SafeDeployedResponse reports whether a Safe wallet is deployed.
type SafeDeployedResponse struct {
	// Address is the Safe wallet address queried.
	Address string `json:"address,omitempty"`
	// Deployed is true when the Safe is deployed.
	Deployed bool `json:"deployed"`
}

// APIKey describes a relayer API key.
type APIKey struct {
	// Key is the API key.
	Key string `json:"apiKey"`
	// Address is the owner address.
	Address string `json:"address"`
	// CreatedAt is the key creation timestamp.
	CreatedAt pmtypes.Time `json:"createdAt"`
	// UpdatedAt is the key update timestamp.
	UpdatedAt pmtypes.Time `json:"updatedAt"`
}

type ProxyTransaction struct {
	To       string   `json:"to"`
	TypeCode CallType `json:"typeCode"`
	Data     string   `json:"data"`
	Value    string   `json:"value"`
}

type ProxySubmitTransactionArgs struct {
	// From is the EOA signer address. If empty, signer.Address() is used.
	From string
	// ProxyWallet is the user's Polymarket proxy wallet / funder address.
	// Required. Do not derive it unless the official CREATE2 formula is fully verified.
	ProxyWallet string
	// Data is encoded proxy transaction batch calldata.
	Data string
	// Metadata is optional relayer metadata.
	Metadata string
	// GasLimit optionally overrides the computed proxy gas limit.
	GasLimit string
}

type SafeTransaction struct {
	To        string        `json:"to"`
	Operation OperationType `json:"operation"`
	Data      string        `json:"data"`
	Value     string        `json:"value"`
}

// SafeSubmitTransactionArgs contains input for building a signed SAFE submit request.
type SafeSubmitTransactionArgs struct {
	// From is the EOA signer address. If empty, signer.Address() is used.
	From string
	// ProxyWallet is the Safe wallet address. If empty, it is derived from From.
	ProxyWallet string
	// ChainID is the EIP-712 chain id.
	ChainID int64
	// Transactions are Safe transactions. Multiple transactions are wrapped in MultiSend.
	Transactions []SafeTransaction
	// Metadata is optional caller-provided relayer metadata.
	Metadata string
}
