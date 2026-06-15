package client_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"casper_x402_facilitator/x402/mechanisms/casper"
	casperClient "casper_x402_facilitator/x402/mechanisms/casper/exact/client"

	eip712 "github.com/casper-ecosystem/casper-eip-712/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/x402-foundation/x402/go/types"
)

type testSigner struct {
	address   string
	publicKey string
}

func (s *testSigner) AccountAddress() string { return s.address }
func (s *testSigner) PublicKey() string      { return s.publicKey }
func (s *testSigner) SignEIP712(digest [32]byte) ([65]byte, error) {
	var sig [65]byte
	copy(sig[:32], digest[:])
	return sig, nil
}

func newTestSigner() *testSigner {
	return &testSigner{
		address:   "01" + "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		publicKey: "01aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788aabb",
	}
}

func validRequirements() types.PaymentRequirements {
	return types.PaymentRequirements{
		Scheme:            "exact",
		Network:           casper.NetworkCasperTestnet,
		Asset:             "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		Amount:            "1000000",
		PayTo:             "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		MaxTimeoutSeconds: 300,
		Extra: map[string]interface{}{
			"name":    "TestToken",
			"version": "1",
		},
	}
}

func TestCreatePaymentPayload_HappyPath(t *testing.T) {
	signer := newTestSigner()
	scheme := casperClient.NewExactCasperScheme(signer)

	payload, err := scheme.CreatePaymentPayload(context.Background(), validRequirements())
	require.NoError(t, err)

	assert.Equal(t, 2, payload.X402Version)
	assert.NotNil(t, payload.Payload)
	assert.NotEmpty(t, payload.Payload["signature"])
	assert.NotEmpty(t, payload.Payload["publicKey"])

	auth, ok := payload.Payload["authorization"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, signer.AccountAddress(), auth["from"])
	assert.Equal(t, "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788", auth["to"])
	assert.Equal(t, "1000000", auth["value"])
	assert.NotEmpty(t, auth["nonce"])

	now := time.Now().Unix()
	validAfterStr, ok := auth["validAfter"].(string)
	require.True(t, ok)
	var validAfter int64
	_, err = fmt.Sscanf(validAfterStr, "%d", &validAfter)
	require.NoError(t, err)
	assert.LessOrEqual(t, validAfter, now)

	validBeforeStr, ok := auth["validBefore"].(string)
	require.True(t, ok)
	var validBefore int64
	_, err = fmt.Sscanf(validBeforeStr, "%d", &validBefore)
	require.NoError(t, err)
	assert.InDelta(t, now+300, validBefore, 5)
}

// TODO: consider adding a check for supported network
//func TestCreatePaymentPayload_UnsupportedNetwork(t *testing.T) {
//	signer := newTestSigner()
//	scheme := casperClient.NewExactCasperScheme(signer)
//
//	req := validRequirements()
//	req.Network = "eip155:1"
//
//	_, err := scheme.CreatePaymentPayload(context.Background(), req)
//	assert.ErrorContains(t, err, casperClient.ErrUnsupportedNetwork)
//}

func TestCreatePaymentPayload_MissingExtraName(t *testing.T) {
	signer := newTestSigner()
	scheme := casperClient.NewExactCasperScheme(signer)

	req := validRequirements()
	delete(req.Extra, "name")

	_, err := scheme.CreatePaymentPayload(context.Background(), req)
	assert.ErrorContains(t, err, casperClient.ErrMissingTokenName)
}

func TestCreatePaymentPayload_MissingExtraVersion(t *testing.T) {
	signer := newTestSigner()
	scheme := casperClient.NewExactCasperScheme(signer)

	req := validRequirements()
	delete(req.Extra, "version")

	_, err := scheme.CreatePaymentPayload(context.Background(), req)
	assert.ErrorContains(t, err, casperClient.ErrMissingTokenVersion)
}

func TestCreatePaymentPayload_InvalidAsset(t *testing.T) {
	signer := newTestSigner()
	scheme := casperClient.NewExactCasperScheme(signer)

	req := validRequirements()
	req.Asset = "not-valid-hex"

	_, err := scheme.CreatePaymentPayload(context.Background(), req)
	assert.ErrorContains(t, err, casperClient.ErrInvalidAsset)
}

func TestCreatePaymentPayload_NonceIsRandom(t *testing.T) {
	signer := newTestSigner()
	scheme := casperClient.NewExactCasperScheme(signer)
	req := validRequirements()

	p1, err := scheme.CreatePaymentPayload(context.Background(), req)
	require.NoError(t, err)
	p2, err := scheme.CreatePaymentPayload(context.Background(), req)
	require.NoError(t, err)

	auth1 := p1.Payload["authorization"].(map[string]interface{})
	auth2 := p2.Payload["authorization"].(map[string]interface{})
	assert.NotEqual(t, auth1["nonce"], auth2["nonce"])
}

func TestHashStruct_ValidTimestampsBigIntAndUint64Match(t *testing.T) {
	fromAddr, err := eip712.NewAddressFromHex("0x01aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788")
	require.NoError(t, err)
	toAddr, err := eip712.NewAddressFromHex("0x00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788")
	require.NoError(t, err)

	value := big.NewInt(1000000)
	validAfter := int64(1700000000)
	validBefore := int64(1700001000)

	nonceRaw, err := hex.DecodeString("aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788")
	require.NoError(t, err)
	var nonce [32]byte
	copy(nonce[:], nonceRaw)

	primaryType := "TransferWithAuthorization"
	types := eip712.TypeDefinitions{
		"TransferWithAuthorization": {
			{Name: "from", Type: "address"},
			{Name: "to", Type: "address"},
			{Name: "value", Type: "uint256"},
			{Name: "validAfter", Type: "uint256"},
			{Name: "validBefore", Type: "uint256"},
			{Name: "nonce", Type: "bytes32"},
		},
	}

	bigIntMessage := map[string]interface{}{
		"from":        fromAddr,
		"to":          toAddr,
		"value":       value,
		"validAfter":  big.NewInt(validAfter),
		"validBefore": big.NewInt(validBefore),
		"nonce":       nonce,
	}
	uint64Message := map[string]interface{}{
		"from":        fromAddr,
		"to":          toAddr,
		"value":       value,
		"validAfter":  uint64(validAfter),
		"validBefore": uint64(validBefore),
		"nonce":       nonce,
	}

	bigIntStructHash, err := eip712.HashStruct(primaryType, types, bigIntMessage)
	require.NoError(t, err)
	uint64StructHash, err := eip712.HashStruct(primaryType, types, uint64Message)
	require.NoError(t, err)

	assert.Equal(t, bigIntStructHash, uint64StructHash)
}
