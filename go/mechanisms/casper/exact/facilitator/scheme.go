package facilitator

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	eip712 "github.com/casper-ecosystem/casper-eip-712/go"
	"github.com/make-software/casper-go-sdk/v2/casper"
	casperrpc "github.com/make-software/casper-go-sdk/v2/rpc"
	"github.com/make-software/casper-go-sdk/v2/types"
	"github.com/make-software/casper-go-sdk/v2/types/clvalue"
	"github.com/make-software/casper-go-sdk/v2/types/clvalue/cltype"
	"github.com/make-software/casper-go-sdk/v2/types/key"
	"github.com/make-software/casper-go-sdk/v2/types/keypair"

	casperMechanism "github.com/x402-foundation/x402/go/v2/mechanisms/casper"

	x402 "github.com/x402-foundation/x402/go/v2"
	x402types "github.com/x402-foundation/x402/go/v2/types"
)

const defaultPaymentMotes uint64 = 2_500_000_000

var transferWithAuthorizationTypes = eip712.TypeDefinitions{
	"TransferWithAuthorization": {
		{Name: "from", Type: "address"},
		{Name: "to", Type: "address"},
		{Name: "value", Type: "uint256"},
		{Name: "validAfter", Type: "uint256"},
		{Name: "validBefore", Type: "uint256"},
		{Name: "nonce", Type: "bytes32"},
	},
}

type ExactCasperSchemeConfig struct {
	LimitedPaymentMotes         *uint64
	LegacyPackagesWithAmountArg []string
}

type ExactCasperScheme struct {
	signer casperMechanism.FacilitatorCasperSigner
	config *ExactCasperSchemeConfig
}

func NewExactCasperScheme(signer casperMechanism.FacilitatorCasperSigner, config *ExactCasperSchemeConfig) *ExactCasperScheme {
	if config == nil {
		config = &ExactCasperSchemeConfig{}
	}

	return &ExactCasperScheme{signer, config}
}

// Scheme returns the scheme identifier
func (f *ExactCasperScheme) Scheme() string {
	return casperMechanism.SchemeExact
}

// CaipFamily returns the CAIP family pattern this facilitator supports
func (f *ExactCasperScheme) CaipFamily() string {
	return "casper:*"
}

// GetExtra returns mechanism-specific extra data for the supported kinds endpoint.
// For Casper, this includes a fee payer address.
func (f *ExactCasperScheme) GetExtra(network x402.Network) map[string]interface{} {
	feePayer := ""
	addresses := f.signer.GetAddresses(context.Background(), string(network))
	if len(addresses) > 0 {
		feePayer = addresses[0]
	}

	return map[string]interface{}{
		"feePayer": feePayer,
	}
}

// GetSigners returns signer addresses used by this facilitator.
// For Casper, returns all available fee payer addresses for the given network.
func (f *ExactCasperScheme) GetSigners(network x402.Network) []string {
	return f.signer.GetAddresses(context.Background(), string(network))
}

// Verify verifies a V2 payment payload against requirements
func (f *ExactCasperScheme) Verify(
	ctx context.Context,
	payload x402types.PaymentPayload,
	requirements x402types.PaymentRequirements,
	_ *x402.FacilitatorContext,
) (*x402.VerifyResponse, error) {
	if payload.Accepted.Scheme != casperMechanism.SchemeExact {
		return nil, x402.NewVerifyError(ErrUnsupportedScheme, "", fmt.Sprintf("invalid scheme: %s", payload.Accepted.Scheme))
	}

	if payload.Accepted.Network != requirements.Network {
		return nil, x402.NewVerifyError(ErrNetworkMismatch, "", fmt.Sprintf("network mismatch: payload=%s requirements=%s", payload.Accepted.Network, requirements.Network))
	}

	p, err := payloadFromMap(payload.Payload)
	if err != nil {
		return nil, x402.NewVerifyError(ErrMalformedPayload, "", err.Error())
	}

	if p.Authorization.To != requirements.PayTo {
		return nil, x402.NewVerifyError(ErrPayToMismatch, "", fmt.Sprintf("authorization.to=%s requirements.payTo=%s", p.Authorization.To, requirements.PayTo))
	}

	if p.Authorization.Value != requirements.Amount {
		return nil, x402.NewVerifyError(ErrAmountMismatch, "", fmt.Sprintf("authorization.value=%s requirements.amount=%s", p.Authorization.Value, requirements.Amount))
	}

	if !casperMechanism.IsValidAddress(requirements.PayTo) || !casperMechanism.IsValidAddress(p.Authorization.To) {
		return nil, x402.NewVerifyError(ErrInvalidPayTo, "", "invalid payTo account-hash")
	}

	if requirements.Amount == "" || requirements.Amount == "0" || p.Authorization.Value == "" || p.Authorization.Value == "0" {
		return nil, x402.NewVerifyError(ErrInvalidAmount, "", "amount must be non-zero")
	}

	if !casperMechanism.IsValidContractPackageHash(requirements.Asset) {
		return nil, x402.NewVerifyError(ErrInvalidAsset, "", requirements.Asset)
	}

	validAfter, err := strconv.ParseInt(p.Authorization.ValidAfter, 10, 64)
	if err != nil {
		return nil, x402.NewVerifyError(ErrMalformedPayload, "", "invalid validAfter")
	}
	validBefore, err := strconv.ParseInt(p.Authorization.ValidBefore, 10, 64)
	if err != nil {
		return nil, x402.NewVerifyError(ErrMalformedPayload, "", "invalid validBefore")
	}

	now := time.Now().Unix()
	if validAfter > now {
		return nil, x402.NewVerifyError(ErrNotYetValid, "", fmt.Sprintf("validAfter=%d > now=%d", validAfter, now))
	}
	if now > validBefore {
		return nil, x402.NewVerifyError(ErrPayloadExpired, "", fmt.Sprintf("validBefore=%d < now=%d", validBefore, now))
	}
	if validBefore-now < 6 {
		return nil, x402.NewVerifyError(ErrInsufficientTime, "", "less than 6s to settle")
	}

	// verify we have a signer for the network in requirements
	cfg, err := f.signer.GetNetworkConfig(ctx, requirements.Network)
	if err != nil {
		return nil, x402.NewVerifyError(ErrNetworkMismatch, "", err.Error())
	}

	name, ok := requirements.Extra["name"].(string)
	if !ok || name == "" {
		return nil, x402.NewVerifyError(ErrMissingTokenName, "", "")
	}
	version, ok := requirements.Extra["version"].(string)
	if !ok || version == "" {
		return nil, x402.NewVerifyError(ErrMissingTokenVersion, "", "")
	}

	contractPackageHash, err := casperMechanism.DecodeContractPackageHash(requirements.Asset)
	if err != nil {
		return nil, x402.NewVerifyError(ErrInvalidAsset, "", err.Error())
	}

	domain := eip712.BuildDomain(name, version, cfg.ChainName, contractPackageHash)

	fromAddr, err := eip712.NewAddressFromHex("0x" + p.Authorization.From)
	if err != nil {
		return nil, x402.NewVerifyError(ErrMalformedPayload, "", fmt.Sprintf("invalid from address: %v", err))
	}
	toAddr, err := eip712.NewAddressFromHex("0x" + p.Authorization.To)
	if err != nil {
		return nil, x402.NewVerifyError(ErrMalformedPayload, "", fmt.Sprintf("invalid to address: %v", err))
	}

	value, ok := new(big.Int).SetString(p.Authorization.Value, 10)
	if !ok {
		return nil, x402.NewVerifyError(ErrMalformedPayload, "", "invalid value")
	}

	nonceRaw, err := hex.DecodeString(p.Authorization.Nonce)
	if err != nil || len(nonceRaw) != 32 {
		return nil, x402.NewVerifyError(ErrMalformedPayload, "", "invalid nonce")
	}
	var nonceBytes [32]byte
	copy(nonceBytes[:], nonceRaw)

	message := map[string]interface{}{
		"from":        fromAddr,
		"to":          toAddr,
		"value":       value,
		"validAfter":  big.NewInt(validAfter),
		"validBefore": big.NewInt(validBefore),
		"nonce":       nonceBytes,
	}

	digest, err := eip712.HashTypedData(domain, transferWithAuthorizationTypes, "TransferWithAuthorization", message, &eip712.TypedDataOptions{DomainTypes: eip712.CasperDomainTypes})
	if err != nil {
		return nil, x402.NewVerifyError(ErrFailedToHash, "", err.Error())
	}

	sigBytes, err := hex.DecodeString(p.Signature)
	if err != nil || len(sigBytes) != 65 {
		return nil, x402.NewVerifyError(ErrInvalidSignature, "", "signature must be 65 bytes hex")
	}
	var sig65 [65]byte
	copy(sig65[:], sigBytes)

	verified, err := f.signer.VerifyEIP712Signature(digest, sig65, p.PublicKey)
	if err != nil {
		return nil, x402.NewVerifyError(ErrInvalidSignature, "", err.Error())
	}
	if !verified {
		return nil, x402.NewVerifyError(ErrInvalidSignature, "", "signature verification failed")
	}

	if preflightErr := f.validatePreflight(ctx, p, requirements, cfg); preflightErr != nil {
		return nil, preflightErr
	}

	return &x402.VerifyResponse{IsValid: true, Payer: p.Authorization.From}, nil
}

// validatePreflight runs a live preflight check against the network when a speculative
// execution endpoint is configured for the payment network; otherwise it falls back to
// a (TODO) targeted preflight check over the regular RPC URL.
func (f *ExactCasperScheme) validatePreflight(
	ctx context.Context,
	p *casperMechanism.ExactCasperPayload,
	requirements x402types.PaymentRequirements,
	networkCfg casperMechanism.NetworkConfig,
) error {
	speculativeRpcURL := f.signer.GetSpeculativeRpcUrl(requirements.Network)
	if speculativeRpcURL != "" {
		return f.validateSpeculativeExecution(ctx, p, requirements, networkCfg, speculativeRpcURL)
	}
	return f.validateTargetedPreflight(ctx, p, requirements, networkCfg)
}

// validateSpeculativeExecution builds a Casper 1.5 deploy equivalent to the settlement
// transaction, signs it with the facilitator key, and runs speculative_exec against the
// configured speculative RPC endpoint. The deploy is never broadcast; it is only used
// to confirm the authorization would succeed end-to-end before settlement.
func (f *ExactCasperScheme) validateSpeculativeExecution(
	ctx context.Context,
	p *casperMechanism.ExactCasperPayload,
	requirements x402types.PaymentRequirements,
	networkCfg casperMechanism.NetworkConfig,
	speculativeRpcURL string,
) error {
	deploy, err := f.buildTransferWithAuthorizationDeploy(ctx, p, requirements, networkCfg)
	if err != nil {
		return x402.NewVerifyError(ErrSpeculativeExecutionFailed, p.Authorization.From, err.Error())
	}

	if err := f.signer.SignDeploy(deploy, requirements.Network); err != nil {
		return x402.NewVerifyError(ErrSpeculativeExecutionFailed, p.Authorization.From, err.Error())
	}

	client := casperrpc.NewSpeculativeClient(casper.NewRPCHandler(speculativeRpcURL, http.DefaultClient))
	result, err := client.SpeculativeExec(ctx, *deploy, nil)
	if err != nil {
		return x402.NewVerifyError(ErrSpeculativeExecutionFailed, p.Authorization.From, err.Error())
	}

	if result.ExecutionResult.ErrorMessage != nil {
		return x402.NewVerifyError(ErrSpeculativeExecutionFailed, p.Authorization.From,
			fmt.Sprintf("speculative execution failed: %s", *result.ExecutionResult.ErrorMessage))
	}

	return nil
}

// validateTargetedPreflight will run a set of targeted RPC reads (active contract entry
// points, payer balance, used-nonce state) against the regular RPC URL when no
// speculative execution endpoint is configured. Implementation is tracked in a
// follow-up task.
func (f *ExactCasperScheme) validateTargetedPreflight(
	_ context.Context,
	p *casperMechanism.ExactCasperPayload,
	_ x402types.PaymentRequirements,
	_ casperMechanism.NetworkConfig,
) error {
	// TODO: implement targeted preflight checks (active contract, balance, used nonces)
	// using the signer-supplied RPC URL when no speculative execution endpoint is set.
	_ = p
	return nil
}

func (f *ExactCasperScheme) paymentMotes() uint64 {
	if f.config.LimitedPaymentMotes != nil {
		return *f.config.LimitedPaymentMotes
	}

	return defaultPaymentMotes
}

func (f *ExactCasperScheme) amountArgumentName(packageHash string) string {
	for _, legacyPackageHash := range f.config.LegacyPackagesWithAmountArg {
		if strings.EqualFold(strings.TrimSpace(legacyPackageHash), packageHash) {
			return "amount"
		}
	}

	return "value"
}

// buildTransferWithAuthorizationArgs builds the runtime args for the
// transfer_with_authorization contract entry point from an exact Casper payload.
func (f *ExactCasperScheme) buildTransferWithAuthorizationArgs(p *casperMechanism.ExactCasperPayload, requirements x402types.PaymentRequirements) (types.Args, error) {
	fromKey, err := key.NewKey("account-hash-" + p.Authorization.From[2:])
	if err != nil {
		return nil, fmt.Errorf("invalid from account-hash: %w", err)
	}
	toKey, err := key.NewKey("account-hash-" + p.Authorization.To[2:])
	if err != nil {
		return nil, fmt.Errorf("invalid to account-hash: %w", err)
	}
	amountInt, ok := new(big.Int).SetString(p.Authorization.Value, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}
	validAfterInt, err := strconv.ParseUint(p.Authorization.ValidAfter, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid validAfter")
	}
	validBeforeInt, err := strconv.ParseUint(p.Authorization.ValidBefore, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid validBefore")
	}
	nonceBytes, err := hex.DecodeString(p.Authorization.Nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}
	sigBytes, err := hex.DecodeString(p.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}
	pubKey, err := keypair.NewPublicKey(p.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	args := types.Args{}
	args.AddArgument("from", clvalue.NewCLKey(fromKey)).
		AddArgument("to", clvalue.NewCLKey(toKey)).
		AddArgument(f.amountArgumentName(requirements.Asset), *clvalue.NewCLUInt256(amountInt)).
		AddArgument("valid_after", *clvalue.NewCLUInt64(validAfterInt)).
		AddArgument("valid_before", *clvalue.NewCLUInt64(validBeforeInt)).
		AddArgument("nonce", clListU8WithBytes(nonceBytes)).
		AddArgument("public_key", clvalue.NewCLPublicKey(pubKey)).
		AddArgument("signature", clListU8WithBytes(sigBytes))

	return args, nil
}

// buildTransferWithAuthorizationDeploy builds a Casper 1.5 deploy (not a
// TransactionV1) that targets the transfer_with_authorization entry point with the
// runtime args derived from the payload. The deploy is unsigned; the caller signs it
// before passing it to a speculative execution endpoint.
func (f *ExactCasperScheme) buildTransferWithAuthorizationDeploy(
	ctx context.Context,
	p *casperMechanism.ExactCasperPayload,
	requirements x402types.PaymentRequirements,
	networkCfg casperMechanism.NetworkConfig,
) (*types.Deploy, error) {
	facilitatorPubKeyHex, err := f.signer.GetPublicKeyHex(ctx, requirements.Network)
	if err != nil || facilitatorPubKeyHex == "" {
		return nil, fmt.Errorf("no facilitator public key: %w", err)
	}
	facilitatorPubKey, err := keypair.NewPublicKey(facilitatorPubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid facilitator public key: %w", err)
	}

	args, err := f.buildTransferWithAuthorizationArgs(p, requirements)
	if err != nil {
		return nil, err
	}

	contractHashRaw, err := casper.NewHash(requirements.Asset)
	if err != nil {
		return nil, fmt.Errorf("invalid contract hash: %w", err)
	}

	chainName := strings.TrimPrefix(networkCfg.ChainName, "casper:")

	header := types.DefaultDeployHeader()
	header.Account = facilitatorPubKey
	header.ChainName = chainName
	header.GasPrice = 1

	payment := types.StandardPayment(big.NewInt(int64(f.paymentMotes())))
	session := types.ExecutableDeployItem{
		StoredVersionedContractByHash: &types.StoredVersionedContractByHash{
			Hash:       key.ContractHash{Hash: contractHashRaw},
			EntryPoint: "transfer_with_authorization",
			Args:       &args,
		},
	}

	return types.MakeDeploy(header, payment, session)
}

func (f *ExactCasperScheme) Settle(
	ctx context.Context,
	payload x402types.PaymentPayload,
	requirements x402types.PaymentRequirements,
	fctx *x402.FacilitatorContext,
) (*x402.SettleResponse, error) {
	network := x402.Network(requirements.Network)

	verifyResp, err := f.Verify(ctx, payload, requirements, fctx)
	if err != nil {
		ve := &x402.VerifyError{}
		if errors.As(err, &ve) {
			return nil, x402.NewSettleError(ErrVerificationFailed, ve.Payer, network, "", ve.InvalidMessage)
		}
		return nil, x402.NewSettleError(ErrVerificationFailed, "", network, "", err.Error())
	}

	p, err := payloadFromMap(payload.Payload)
	if err != nil {
		return nil, x402.NewSettleError(ErrMalformedPayload, verifyResp.Payer, network, "", err.Error())
	}

	args, err := f.buildTransferWithAuthorizationArgs(p, requirements)
	if err != nil {
		return nil, x402.NewSettleError(ErrBuildDeployFailed, verifyResp.Payer, network, "", err.Error())
	}

	packageHash, err := casper.NewHash(requirements.Asset)
	if err != nil {
		return nil, x402.NewSettleError(ErrBuildDeployFailed, verifyResp.Payer, network, "", err.Error())
	}
	entryPoint := "transfer_with_authorization"

	facilitatorPubKeyHex, err := f.signer.GetPublicKeyHex(ctx, requirements.Network)
	if err != nil || facilitatorPubKeyHex == "" {
		return nil, x402.NewSettleError(ErrBuildDeployFailed, verifyResp.Payer, network, "", "no facilitator public key")
	}
	facilitatorPubKey, err := keypair.NewPublicKey(facilitatorPubKeyHex)
	if err != nil {
		return nil, x402.NewSettleError(ErrBuildDeployFailed, verifyResp.Payer, network, "", err.Error())
	}

	// verify we have a signer for this network before proceeding
	networkCfg, err := f.signer.GetNetworkConfig(ctx, requirements.Network)
	if err != nil {
		return nil, x402.NewSettleError(ErrNetworkMismatch, verifyResp.Payer, network, "", err.Error())
	}

	chainName := strings.TrimPrefix(networkCfg.ChainName, "casper:")

	v1Payload, err := types.NewTransactionV1Payload(
		types.InitiatorAddr{
			PublicKey: &facilitatorPubKey,
		},
		types.Timestamp(time.Now().UTC()),
		900000000000,
		chainName,
		types.PricingMode{
			Limited: &types.LimitedMode{
				GasPriceTolerance: 1,
				StandardPayment:   true,
				PaymentAmount:     f.paymentMotes(),
			},
		},
		types.NewNamedArgs(&args),
		types.TransactionTarget{
			Stored: &types.StoredTarget{
				ID: types.TransactionInvocationTarget{
					ByPackageHash: &types.ByPackageHashInvocationTarget{
						Addr:    packageHash,
						Version: nil,
					},
				},
				Runtime: types.NewVmCasperV1TransactionRuntime(),
			},
		},
		types.TransactionEntryPoint{
			Custom: &entryPoint,
		},
		types.TransactionScheduling{
			Standard: &struct{}{},
		},
	)

	if err != nil {
		return nil, x402.NewSettleError(ErrSignDeployFailed, verifyResp.Payer, network, "", err.Error())
	}

	transaction, err := types.MakeTransactionV1(v1Payload)
	if err != nil {
		return nil, x402.NewSettleError(ErrSignDeployFailed, verifyResp.Payer, network, "", err.Error())
	}

	if err := f.signer.SignTransaction(transaction, requirements.Network); err != nil {
		return nil, x402.NewSettleError(ErrSignDeployFailed, verifyResp.Payer, network, "", err.Error())
	}

	_, err = f.signer.PutTransaction(ctx, requirements.Network, *transaction)
	if err != nil {
		return nil, x402.NewSettleError(ErrPutDeployFailed, verifyResp.Payer, network, transaction.Hash.String(), err.Error())
	}

	if err := f.signer.WaitForTransaction(ctx, requirements.Network, transaction.Hash.String()); err != nil {
		return nil, x402.NewSettleError(ErrWaitDeployFailed, verifyResp.Payer, network, transaction.Hash.String(), err.Error())
	}

	return &x402.SettleResponse{
		Success:     true,
		Transaction: transaction.Hash.String(),
		Network:     network,
		Payer:       verifyResp.Payer,
	}, nil
}

func payloadFromMap(data map[string]interface{}) (*casperMechanism.ExactCasperPayload, error) {
	if data == nil {
		return nil, fmt.Errorf("payload is nil")
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload data: %w", err)
	}

	var payload casperMechanism.ExactCasperPayload
	if err := json.Unmarshal(jsonBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if payload.Signature == "" || payload.PublicKey == "" {
		return nil, fmt.Errorf("missing signature or publicKey field")
	}

	if payload.Authorization.From == "" ||
		payload.Authorization.To == "" ||
		payload.Authorization.Value == "" ||
		payload.Authorization.ValidAfter == "" ||
		payload.Authorization.ValidBefore == "" ||
		payload.Authorization.Nonce == "" {
		return nil, fmt.Errorf("missing authorization fields")
	}

	return &payload, nil
}

func clListU8WithBytes(bytes []byte) clvalue.CLValue {
	elements := make([]clvalue.CLValue, 0)
	for _, v := range bytes {
		ui8 := clvalue.UInt8(v)
		elements = append(elements, clvalue.CLValue{
			Type: cltype.UInt8,
			UI8:  &ui8,
		})
	}
	clType := cltype.NewList(cltype.UInt8)
	value := clvalue.CLValue{
		Type: clType,
		List: &clvalue.List{
			Type:     clType,
			Elements: elements,
		},
	}

	return value
}
