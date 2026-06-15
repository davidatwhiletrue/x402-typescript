package server_test

import (
	"context"
	"testing"

	"casper_x402_facilitator/x402/mechanisms/casper"
	casperServer "casper_x402_facilitator/x402/mechanisms/casper/exact/server"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	x402 "github.com/x402-foundation/x402/go"
	"github.com/x402-foundation/x402/go/types"
)

const (
	testAsset   = "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	testNetwork = casper.NetworkCasperTestnet
)

func TestParsePrice_ExplicitAssetAmount(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	price := map[string]interface{}{
		"amount": "1000000",
		"asset":  testAsset,
		"extra": map[string]interface{}{
			"name":    "MyToken",
			"version": "1",
		},
	}

	result, err := s.ParsePrice(price, x402.Network(testNetwork))
	require.NoError(t, err)
	assert.Equal(t, "1000000", result.Amount)
	assert.Equal(t, testAsset, result.Asset)
	assert.Equal(t, "MyToken", result.Extra["name"])
}

func TestParsePrice_ExplicitNoExtra(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	price := map[string]interface{}{
		"amount": "500000",
		"asset":  testAsset,
	}

	result, err := s.ParsePrice(price, x402.Network(testNetwork))
	require.NoError(t, err)
	assert.Equal(t, "500000", result.Amount)
	assert.Equal(t, testAsset, result.Asset)
}

func TestParsePrice_MoneyString_NoDefaultAsset(t *testing.T) {
	s := casperServer.NewExactCasperScheme()

	_, err := s.ParsePrice("1.00", x402.Network(testNetwork))
	assert.ErrorContains(t, err, casperServer.ErrNoDefaultAsset)
}

func TestParsePrice_MoneyFloat_NoDefaultAsset(t *testing.T) {
	s := casperServer.NewExactCasperScheme()

	_, err := s.ParsePrice(1.0, x402.Network(testNetwork))
	assert.ErrorContains(t, err, casperServer.ErrNoDefaultAsset)
}

func TestParsePrice_WithCustomMoneyParser(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	s.RegisterMoneyParser(func(_ float64, _ x402.Network) (*x402.AssetAmount, error) {
		return &x402.AssetAmount{
			Amount: "9999",
			Asset:  testAsset,
			Extra:  map[string]interface{}{"name": "Custom", "version": "2"},
		}, nil
	})

	result, err := s.ParsePrice(1.0, x402.Network(testNetwork))
	require.NoError(t, err)
	assert.Equal(t, "9999", result.Amount)
}

func TestParsePrice_InvalidAssetInMap(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	price := map[string]interface{}{
		"amount": "1000000",
		"asset":  "not-valid",
	}

	_, err := s.ParsePrice(price, x402.Network(testNetwork))
	assert.ErrorContains(t, err, casperServer.ErrInvalidAsset)
}

func TestEnhancePaymentRequirements_HappyPath(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	req := types.PaymentRequirements{
		Scheme:  "exact",
		Network: testNetwork,
		Asset:   testAsset,
		Amount:  "1000000",
		PayTo:   "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		Extra: map[string]interface{}{
			"name":    "TestToken",
			"version": "1",
		},
	}

	result, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "1000000", result.Amount)
}

func TestEnhancePaymentRequirements_DecimalAmount(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	s.RegisterAsset(testNetwork, testAsset, 6)
	req := types.PaymentRequirements{
		Scheme:  "exact",
		Network: testNetwork,
		Asset:   testAsset,
		Amount:  "1.5",
		PayTo:   "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		Extra: map[string]interface{}{
			"name":    "TestToken",
			"version": "1",
		},
	}

	result, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "1500000", result.Amount)
}

func TestEnhancePaymentRequirements_MissingTokenName(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	req := types.PaymentRequirements{
		Scheme:  "exact",
		Network: testNetwork,
		Asset:   testAsset,
		Amount:  "1000000",
		PayTo:   "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		Extra:   map[string]interface{}{"version": "1"},
	}

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, casperServer.ErrMissingTokenName)
}

func TestEnhancePaymentRequirements_InvalidAsset(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	req := types.PaymentRequirements{
		Scheme:  "exact",
		Network: testNetwork,
		Asset:   "too-short",
		Amount:  "1000000",
		PayTo:   "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		Extra:   map[string]interface{}{"name": "T", "version": "1"},
	}

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, casperServer.ErrInvalidAsset)
}

func TestEnhancePaymentRequirements_InvalidPayTo(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	req := types.PaymentRequirements{
		Scheme:  "exact",
		Network: testNetwork,
		Asset:   testAsset,
		Amount:  "1000000",
		PayTo:   "invalid",
		Extra:   map[string]interface{}{"name": "T", "version": "1"},
	}

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, casperServer.ErrInvalidPayTo)
}

func TestEnhancePaymentRequirements_ExtensionKeysForwarded(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	req := types.PaymentRequirements{
		Scheme:  "exact",
		Network: testNetwork,
		Asset:   testAsset,
		Amount:  "1000000",
		PayTo:   "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788",
		Extra:   map[string]interface{}{"name": "T", "version": "1"},
	}
	sk := types.SupportedKind{Extra: map[string]interface{}{"customKey": "customValue"}}

	result, err := s.EnhancePaymentRequirements(context.Background(), req, sk, []string{"customKey"})
	require.NoError(t, err)
	assert.Equal(t, "customValue", result.Extra["customKey"])
}

func TestGetAssetDecimals_Registered(t *testing.T) {
	s := casperServer.NewExactCasperScheme()
	s.RegisterAsset(testNetwork, testAsset, 9)

	d := s.GetAssetDecimals(testAsset, x402.Network(testNetwork))
	assert.Equal(t, 9, d)
}

func TestGetAssetDecimals_DefaultFallback(t *testing.T) {
	s := casperServer.NewExactCasperScheme()

	d := s.GetAssetDecimals("unknown", x402.Network(testNetwork))
	assert.Equal(t, 9, d)
}
