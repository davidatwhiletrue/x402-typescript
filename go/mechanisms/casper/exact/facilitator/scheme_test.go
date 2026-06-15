package facilitator_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"casper_x402_facilitator/x402/mechanisms/casper"
	casperFacilitator "casper_x402_facilitator/x402/mechanisms/casper/exact/facilitator"

	caspertypes "github.com/make-software/casper-go-sdk/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	x402 "github.com/x402-foundation/x402/go"
	"github.com/x402-foundation/x402/go/types"
)

const (
	testNetwork     = casper.NetworkCasperTestnet
	testAsset       = "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	testPayTo       = "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	testFrom        = "01aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	facilitatorAddr = testFrom
)

type mockFacilitatorSigner struct {
	addresses       []string
	publicKeyHex    string
	verifyResult    bool
	verifyErr       error
	putDeployResult string
	putDeployErr    error
	waitErr         error
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

func (m *mockFacilitatorSigner) SignTransaction(_ *caspertypes.TransactionV1, _ string) error {
	return nil
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
	payload.Accepted.Network = casper.NetworkCasperMainnet

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
