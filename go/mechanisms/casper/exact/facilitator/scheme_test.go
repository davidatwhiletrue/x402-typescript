package facilitator_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/x402-foundation/x402/go/v2/mechanisms/casper"
	casperFacilitator "github.com/x402-foundation/x402/go/v2/mechanisms/casper/exact/facilitator"

	caspertypes "github.com/make-software/casper-go-sdk/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	x402 "github.com/x402-foundation/x402/go/v2"
	"github.com/x402-foundation/x402/go/v2/types"
)

const (
	testNetwork     = casper.CasperTestnetCAIP2
	testAsset       = "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	testPayTo       = "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	testFrom        = "01aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	facilitatorAddr = testFrom
)

type mockFacilitatorSigner struct {
	addresses           []string
	publicKeyHex        string
	verifyResult        bool
	verifyErr           error
	putDeployResult     string
	putDeployErr        error
	waitErr             error
	speculativeRpcURL   string
	capturedTransaction *caspertypes.TransactionV1
	capturedDeploy      *caspertypes.Deploy
}

func (m *mockFacilitatorSigner) GetNetworkConfig(_ context.Context, _ string) (casper.NetworkConfig, error) {
	return casper.NetworkConfig{
		testNetwork,
		"http://127.0.0.1:11101/rpc",
	}, nil
}

func (m *mockFacilitatorSigner) GetAddresses(_ context.Context, _ string) []string {
	return m.addresses
}

func (m *mockFacilitatorSigner) GetPublicKeyHex(_ context.Context, _ string) (string, error) {
	return m.publicKeyHex, nil
}

func (m *mockFacilitatorSigner) VerifyEIP712Signature(_ [32]byte, _ [65]byte, _ string) (bool, error) {
	return m.verifyResult, m.verifyErr
}

func (m *mockFacilitatorSigner) SignTransaction(tx *caspertypes.TransactionV1, _ string) error {
	m.capturedTransaction = tx
	return nil
}
func (m *mockFacilitatorSigner) SignDeploy(deploy *caspertypes.Deploy, _ string) error {
	m.capturedDeploy = deploy
	return nil
}
func (m *mockFacilitatorSigner) GetSpeculativeRpcUrl(_ string) string {
	return m.speculativeRpcURL
}
func (m *mockFacilitatorSigner) PutTransaction(_ context.Context, _ string, _ caspertypes.TransactionV1) (string, error) {
	return m.putDeployResult, m.putDeployErr
}
func (m *mockFacilitatorSigner) WaitForTransaction(_ context.Context, _ string, _ string) error {
	return m.waitErr
}

func defaultSigner() *mockFacilitatorSigner {
	return &mockFacilitatorSigner{
		addresses:       []string{facilitatorAddr},
		publicKeyHex:    "01aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		verifyResult:    true,
		putDeployResult: strings.Repeat("ab", 32),
	}
}

func buildValidPayload(t *testing.T) (types.PaymentPayload, types.PaymentRequirements) {
	t.Helper()

	now := time.Now().Unix()
	payloadObj := &casper.ExactCasperPayload{
		Signature: strings.Repeat("ab", 65),
		PublicKey: "01aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		Authorization: casper.ExactCasperAuthorization{
			From:        testFrom,
			To:          testPayTo,
			Value:       "1000000",
			ValidAfter:  fmt.Sprintf("%d", now-60),
			ValidBefore: fmt.Sprintf("%d", now+300),
			Nonce:       "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		},
	}

	payload := types.PaymentPayload{
		X402Version: 2,
		Payload:     payloadObj.ToMap(),
		Accepted: types.PaymentRequirements{
			Scheme:  "exact",
			Network: testNetwork,
		},
	}

	req := types.PaymentRequirements{
		Scheme:  "exact",
		Network: testNetwork,
		Asset:   testAsset,
		Amount:  "1000000",
		PayTo:   testPayTo,
		Extra: map[string]interface{}{
			"name":    "TestToken",
			"version": "1",
		},
	}

	return payload, req
}

func TestGetExtra_ReturnsCasperMetadata(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	extra := scheme.GetExtra(x402.Network(testNetwork))
	require.NotNil(t, extra)
	assert.Equal(t, facilitatorAddr, extra["feePayer"])
}

func TestGetExtra_EmptySignerAddresses(t *testing.T) {
	signer := defaultSigner()
	signer.addresses = nil
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	extra := scheme.GetExtra(x402.Network(testNetwork))
	require.NotNil(t, extra)
	assert.Equal(t, "", extra["feePayer"])
}

func TestVerify_HappyPath(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	resp, err := scheme.Verify(context.Background(), payload, req, nil)
	require.NoError(t, err)
	assert.True(t, resp.IsValid)
	assert.Equal(t, testFrom, resp.Payer)
}

func TestVerify_WrongScheme(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	payload.Accepted.Scheme = "upto"

	_, err := scheme.Verify(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrUnsupportedScheme)
}

func TestVerify_NetworkMismatch(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	payload.Accepted.Network = casper.CasperMainnetCAIP2

	_, err := scheme.Verify(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrNetworkMismatch)
}

func TestVerify_PayToMismatch(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	req.PayTo = "00" + strings.Repeat("bb", 32)

	_, err := scheme.Verify(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrPayToMismatch)
}

func TestVerify_AmountMismatch(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	req.Amount = "9999999"

	_, err := scheme.Verify(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrAmountMismatch)
}

func TestVerify_Expired(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	auth := payload.Payload["authorization"].(map[string]interface{})
	auth["validBefore"] = fmt.Sprintf("%d", time.Now().Unix()-1)

	_, err := scheme.Verify(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrPayloadExpired)
}

func TestVerify_NotYetValid(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	now := time.Now().Unix()
	auth := payload.Payload["authorization"].(map[string]interface{})
	auth["validAfter"] = fmt.Sprintf("%d", now+9999)
	auth["validBefore"] = fmt.Sprintf("%d", now+10000)

	_, err := scheme.Verify(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrNotYetValid)
}

func TestSettle_HappyPath(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	resp, err := scheme.Settle(context.Background(), payload, req, nil)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Regexp(t, `^[0-9a-fA-F]{64}$`, resp.Transaction)
	assert.Equal(t, testFrom, resp.Payer)
	assert.Equal(t, x402.Network(testNetwork), resp.Network)
}

func TestSettle_UsesValueArgumentByDefault(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	_, err := scheme.Settle(context.Background(), payload, req, nil)
	require.NoError(t, err)
	require.NotNil(t, signer.capturedTransaction)

	args := signer.capturedTransaction.Payload.Fields.NamedArgs.Args
	require.NotNil(t, args)
	_, err = args.Find("value")
	assert.NoError(t, err)
	_, err = args.Find("amount")
	assert.Error(t, err)
}

func TestSettle_UsesAmountArgumentForLegacyPackage(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, &casperFacilitator.ExactCasperSchemeConfig{
		LegacyPackagesWithAmountArg: []string{testAsset},
	})

	payload, req := buildValidPayload(t)
	_, err := scheme.Settle(context.Background(), payload, req, nil)
	require.NoError(t, err)
	require.NotNil(t, signer.capturedTransaction)

	args := signer.capturedTransaction.Payload.Fields.NamedArgs.Args
	require.NotNil(t, args)
	_, err = args.Find("amount")
	assert.NoError(t, err)
	_, err = args.Find("value")
	assert.Error(t, err)
}

func TestSettle_VerifyFails(t *testing.T) {
	signer := defaultSigner()
	signer.verifyResult = false

	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)
	payload, req := buildValidPayload(t)

	_, err := scheme.Settle(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrVerificationFailed)
}

func TestSettle_PutDeployFails(t *testing.T) {
	signer := defaultSigner()
	signer.putDeployErr = fmt.Errorf("rpc timeout")

	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)
	payload, req := buildValidPayload(t)

	_, err := scheme.Settle(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrPutDeployFailed)
}

func TestSettle_WaitDeployFails(t *testing.T) {
	signer := defaultSigner()
	signer.waitErr = fmt.Errorf("context deadline exceeded")

	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)
	payload, req := buildValidPayload(t)

	_, err := scheme.Settle(context.Background(), payload, req, nil)
	assert.ErrorContains(t, err, casperFacilitator.ErrWaitDeployFailed)
}

func TestVerify_NoSpeculativeRpcURL_AcceptsPayload(t *testing.T) {
	signer := defaultSigner()
	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)

	payload, req := buildValidPayload(t)
	resp, err := scheme.Verify(context.Background(), payload, req, nil)
	require.NoError(t, err)
	assert.True(t, resp.IsValid)
	assert.Nil(t, signer.capturedDeploy, "no deploy should be signed when speculative RPC URL is unset")
}

func TestVerify_SpeculativeRpcURL_SignsDeployAndReportsEndpointError(t *testing.T) {
	signer := defaultSigner()
	signer.speculativeRpcURL = "http://127.0.0.1:1/rpc"

	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)
	payload, req := buildValidPayload(t)

	_, err := scheme.Verify(context.Background(), payload, req, nil)
	assert.Error(t, err)
	assert.ErrorContains(t, err, casperFacilitator.ErrSpeculativeExecutionFailed)
	require.NotNil(t, signer.capturedDeploy, "deploy must be built and passed to SignDeploy before speculative exec")
	assert.Equal(t, "transfer_with_authorization", signer.capturedDeploy.Session.StoredVersionedContractByHash.EntryPoint)
	assert.NotNil(t, signer.capturedDeploy.Session.StoredVersionedContractByHash.Args)
}

func TestVerify_SpeculativeRpcURL_AcceptsSuccessfulExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"api_version": "1.0.0",
				"block_hash": "0000000000000000000000000000000000000000000000000000000000000000",
				"execution_result": {
					"Version2": {
						"initiator": {"PublicKey": "010000000000000000000000000000000000000000000000000000000000000000"},
						"error_message": null,
						"limit": "0",
						"consumed": "0",
						"cost": "0",
						"refund": "0",
						"payment": null,
						"transfers": [],
						"size_estimate": 0,
						"effects": []
					}
				}
			}
		}`))
	}))
	defer server.Close()

	signer := defaultSigner()
	signer.speculativeRpcURL = server.URL

	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)
	payload, req := buildValidPayload(t)

	resp, err := scheme.Verify(context.Background(), payload, req, nil)
	require.NoError(t, err)
	assert.True(t, resp.IsValid)
	assert.Equal(t, testFrom, resp.Payer)
	require.NotNil(t, signer.capturedDeploy)
}

func TestVerify_SpeculativeRpcURL_RejectsExecutionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"api_version": "1.0.0",
				"block_hash": "0000000000000000000000000000000000000000000000000000000000000000",
				"execution_result": {
					"Version2": {
						"initiator": {"PublicKey": "010000000000000000000000000000000000000000000000000000000000000000"},
						"error_message": "InsufficientFunds",
						"limit": "0",
						"consumed": "0",
						"cost": "0",
						"refund": "0",
						"payment": null,
						"transfers": [],
						"size_estimate": 0,
						"effects": []
					}
				}
			}
		}`))
	}))
	defer server.Close()

	signer := defaultSigner()
	signer.speculativeRpcURL = server.URL

	scheme := casperFacilitator.NewExactCasperScheme(signer, nil)
	payload, req := buildValidPayload(t)

	_, err := scheme.Verify(context.Background(), payload, req, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, casperFacilitator.ErrSpeculativeExecutionFailed)
	assert.ErrorContains(t, err, "InsufficientFunds")
}
