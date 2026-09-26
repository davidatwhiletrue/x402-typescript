package server

import (
	"context"
	"errors"
	"testing"

	"github.com/x402-foundation/x402/go/v2/mechanisms/casper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	x402 "github.com/x402-foundation/x402/go/v2"
	"github.com/x402-foundation/x402/go/v2/types"
)

const (
	// testAsset is a syntactically valid 32-byte contract-package hash that is
	// not registered in the Casper default-asset tables.
	testAsset = "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	// testPayTo is a syntactically valid Casper account hash with the "00"
	// prefix (33-byte address form).
	testPayTo = "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	// testPayToAlt uses the "01" prefix variant for accounts.
	testPayToAlt = "01aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	// testNetwork exercises the Casper testnet CAIP-2 identifier.
	testNetwork = casper.CasperTestnetCAIP2
	// testCSPRUSD is the testnet csprUSD contract-package hash, registered as
	// the default asset for the testnet network.
	testCSPRUSD = casper.CSPRUSDTestnetAsset
)

func validRequirements() types.PaymentRequirements {
	return types.PaymentRequirements{
		Scheme:  casper.SchemeExact,
		Network: testNetwork,
		Asset:   testAsset,
		Amount:  "1000000",
		PayTo:   testPayTo,
		Extra: map[string]interface{}{
			"name":    "TestToken",
			"version": "1",
		},
	}
}

// === Scheme identity ===

func TestScheme_ReturnsExact(t *testing.T) {
	s := NewExactCasperScheme()
	assert.Equal(t, casper.SchemeExact, s.Scheme())
}

func TestDefaultAssetTransferMethod_ReturnsSDKDefault(t *testing.T) {
	s := NewExactCasperScheme()
	assert.Equal(t, x402.SDKDefaultAssetTransferMethod, s.DefaultAssetTransferMethod())
}

func TestPaymentFlowsDeclareAuthorizationOnly(t *testing.T) {
	scheme := NewExactCasperScheme()
	flows := scheme.PaymentFlows()

	require.Contains(t, flows, x402.SDKDefaultAssetTransferMethod)
	assert.Equal(t, []x402.PaymentFlowName{x402.PaymentFlowAuthorization}, flows[x402.SDKDefaultAssetTransferMethod].Supported)
	assert.Equal(t, x402.PaymentFlowAuthorization, flows[x402.SDKDefaultAssetTransferMethod].Default)
	assert.NotContains(t, flows[x402.SDKDefaultAssetTransferMethod].Supported, x402.PaymentFlowUpfront)
}

func TestDynamicExtraFields_ReturnsNameAndVersion(t *testing.T) {
	s := NewExactCasperScheme()
	assert.Equal(t, []string{"name", "version"}, s.DynamicExtraFields())
}

// === GetAssetDecimals ===

func TestGetAssetDecimals_KnownDefaultAsset(t *testing.T) {
	s := NewExactCasperScheme()

	d, ok := s.GetAssetDecimals(testCSPRUSD, testNetwork)
	require.True(t, ok)
	assert.Equal(t, casper.CSPRUSDDecimals, d)
}

func TestGetAssetDecimals_UnknownAsset(t *testing.T) {
	s := NewExactCasperScheme()

	_, ok := s.GetAssetDecimals(testAsset, testNetwork)
	assert.False(t, ok)
}

func TestGetAssetDecimals_UnknownNetwork(t *testing.T) {
	s := NewExactCasperScheme()

	_, ok := s.GetAssetDecimals(testCSPRUSD, "casper:nope")
	assert.False(t, ok)
}

// === ParsePrice — explicit price map ===

func TestParsePrice_ExplicitAssetAmount(t *testing.T) {
	s := NewExactCasperScheme()
	price := map[string]interface{}{
		"amount": "1000000",
		"asset":  testAsset,
		"extra": map[string]interface{}{
			"name":    "MyToken",
			"version": "1",
		},
	}

	result, err := s.ParsePrice(price, testNetwork)
	require.NoError(t, err)
	assert.Equal(t, "1000000", result.Amount)
	assert.Equal(t, testAsset, result.Asset)
	assert.Equal(t, "MyToken", result.Extra["name"])
	assert.Equal(t, "1", result.Extra["version"])
}

func TestParsePrice_ExplicitNoExtra(t *testing.T) {
	s := NewExactCasperScheme()
	price := map[string]interface{}{
		"amount": "500000",
		"asset":  testAsset,
	}

	result, err := s.ParsePrice(price, testNetwork)
	require.NoError(t, err)
	assert.Equal(t, "500000", result.Amount)
	assert.Equal(t, testAsset, result.Asset)
	require.NotNil(t, result.Extra)
	assert.Empty(t, result.Extra)
}

func TestParsePrice_ExplicitDefaultsAsset(t *testing.T) {
	s := NewExactCasperScheme()
	price := map[string]interface{}{
		"amount": "500000",
		// no "asset" key → falls back to default CSPRUSD testnet asset
	}

	result, err := s.ParsePrice(price, testNetwork)
	require.NoError(t, err)
	assert.Equal(t, testCSPRUSD, result.Asset)
}

func TestParsePrice_AmountMustBeString(t *testing.T) {
	s := NewExactCasperScheme()
	price := map[string]interface{}{
		"amount": 1000000, // numeric, not string
		"asset":  testAsset,
	}

	_, err := s.ParsePrice(price, testNetwork)
	assert.ErrorContains(t, err, ErrAmountMustBeString)
}

// === ParsePrice — money fallback to default asset ===

func TestParsePrice_MoneyString_UsesDefaultAsset(t *testing.T) {
	s := NewExactCasperScheme()

	result, err := s.ParsePrice("1.00", testNetwork)
	require.NoError(t, err)
	assert.Equal(t, "1000000", result.Amount) // 1.00 * 10^6
	assert.Equal(t, testCSPRUSD, result.Asset)
}

func TestParsePrice_MoneyFloat_UsesDefaultAsset(t *testing.T) {
	s := NewExactCasperScheme()

	result, err := s.ParsePrice(1.0, testNetwork)
	require.NoError(t, err)
	assert.Equal(t, "1000000", result.Amount)
	assert.Equal(t, testCSPRUSD, result.Asset)
}

func TestParsePrice_MoneyDollarPrefix(t *testing.T) {
	s := NewExactCasperScheme()

	result, err := s.ParsePrice("$5.50", testNetwork)
	require.NoError(t, err)
	assert.Equal(t, "5500000", result.Amount) // 5.50 * 10^6
	assert.Equal(t, testCSPRUSD, result.Asset)
}

func TestParsePrice_MoneySubunitPrecision(t *testing.T) {
	s := NewExactCasperScheme()

	result, err := s.ParsePrice("0.000001", testNetwork)
	require.NoError(t, err)
	assert.Equal(t, "1", result.Amount)
	assert.Equal(t, testCSPRUSD, result.Asset)
}

func TestParsePrice_MoneyUnsupportedNetwork(t *testing.T) {
	s := NewExactCasperScheme()

	_, err := s.ParsePrice("1.00", "casper:nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no default asset configured")
}

// === ParsePrice — custom parser chain ===

func TestParsePrice_WithCustomMoneyParser(t *testing.T) {
	s := NewExactCasperScheme()
	s.RegisterMoneyParser(func(_ string, _ x402.Network) (*x402.AssetAmount, error) {
		return &x402.AssetAmount{
			Amount: "9999",
			Asset:  testAsset,
			Extra:  map[string]interface{}{"name": "Custom", "version": "2"},
		}, nil
	})

	result, err := s.ParsePrice(1.0, testNetwork)
	require.NoError(t, err)
	assert.Equal(t, "9999", result.Amount)
	assert.Equal(t, testAsset, result.Asset)
	assert.Equal(t, "Custom", result.Extra["name"])
}

func TestParsePrice_CustomParserReturnsNilFallsToDefault(t *testing.T) {
	s := NewExactCasperScheme()
	s.RegisterMoneyParser(func(_ string, _ x402.Network) (*x402.AssetAmount, error) {
		return nil, nil
	})

	result, err := s.ParsePrice(1.0, testNetwork)
	require.NoError(t, err)
	assert.Equal(t, testCSPRUSD, result.Asset)
	assert.Equal(t, "1000000", result.Amount)
}

func TestParsePrice_CustomParserErrorFallsToDefault(t *testing.T) {
	s := NewExactCasperScheme()
	s.RegisterMoneyParser(func(_ string, _ x402.Network) (*x402.AssetAmount, error) {
		return nil, errors.New("parser exploded")
	})

	result, err := s.ParsePrice(1.0, testNetwork)
	require.NoError(t, err)
	assert.Equal(t, testCSPRUSD, result.Asset)
	assert.Equal(t, "1000000", result.Amount)
}

func TestParsePrice_FirstMatchingParserWins(t *testing.T) {
	s := NewExactCasperScheme()
	s.RegisterMoneyParser(func(_ string, _ x402.Network) (*x402.AssetAmount, error) {
		return &x402.AssetAmount{Amount: "111", Asset: testAsset, Extra: map[string]interface{}{"name": "First", "version": "1"}}, nil
	})
	s.RegisterMoneyParser(func(_ string, _ x402.Network) (*x402.AssetAmount, error) {
		return &x402.AssetAmount{Amount: "222", Asset: testAsset, Extra: map[string]interface{}{"name": "Second", "version": "1"}}, nil
	})

	result, err := s.ParsePrice(1.0, testNetwork)
	require.NoError(t, err)
	assert.Equal(t, "111", result.Amount)
	assert.Equal(t, "First", result.Extra["name"])
}

func TestRegisterMoneyParser_Chainability(t *testing.T) {
	s := NewExactCasperScheme()

	result := s.
		RegisterMoneyParser(func(_ string, _ x402.Network) (*x402.AssetAmount, error) { return nil, nil }).
		RegisterMoneyParser(func(_ string, _ x402.Network) (*x402.AssetAmount, error) { return nil, nil })

	assert.Same(t, s, result)
}

// === EnhancePaymentRequirements — happy path ===

func TestEnhancePaymentRequirements_HappyPath(t *testing.T) {
	s := NewExactCasperScheme()

	result, err := s.EnhancePaymentRequirements(context.Background(), validRequirements(), types.SupportedKind{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "1000000", result.Amount)
	assert.Equal(t, testAsset, result.Asset)
	assert.Equal(t, testPayTo, result.PayTo)
	assert.Equal(t, "TestToken", result.Extra["name"])
	assert.Equal(t, "1", result.Extra["version"])
}

func TestEnhancePaymentRequirements_DefaultsAssetWhenEmpty(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Asset = ""

	result, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	require.NoError(t, err)
	assert.Equal(t, testCSPRUSD, result.Asset)
}

func TestEnhancePaymentRequirements_PayToWith01Prefix(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.PayTo = testPayToAlt

	result, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	require.NoError(t, err)
	assert.Equal(t, testPayToAlt, result.PayTo)
}

// === EnhancePaymentRequirements — amount conversion ===

func TestEnhancePaymentRequirements_DecimalAmountConvertedToSmallestUnit(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Asset = testCSPRUSD
	req.Amount = "1.5"

	result, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "1500000", result.Amount)
}

func TestEnhancePaymentRequirements_SmallestUnitPreserved(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Asset = testCSPRUSD
	req.Amount = "1234567"

	result, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "1234567", result.Amount)
}

func TestEnhancePaymentRequirements_EmptyAmountUnchanged(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Asset = testCSPRUSD
	req.Amount = ""

	result, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "", result.Amount)
}

func TestEnhancePaymentRequirements_InvalidDecimalAmount(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Amount = "1.5.5"

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, ErrFailedToParseAmount)
}

// === EnhancePaymentRequirements — extra map ===

func TestEnhancePaymentRequirements_NilExtraStillFailsTokenMetadataCheck(t *testing.T) {
	// The scheme materialises an empty extra map when callers pass nil, but the
	// subsequent token-metadata check still runs against the now-empty map and
	// rejects the requirements. This documents the deliberate "fail loud" path.
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Extra = nil

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, ErrMissingTokenName)
}

// === EnhancePaymentRequirements — token metadata validation ===

func TestEnhancePaymentRequirements_MissingTokenName(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Extra = map[string]interface{}{"version": "1"}

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, ErrMissingTokenName)
}

func TestEnhancePaymentRequirements_EmptyTokenName(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Extra = map[string]interface{}{"name": "", "version": "1"}

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, ErrMissingTokenName)
}

func TestEnhancePaymentRequirements_NonStringTokenName(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Extra = map[string]interface{}{"name": 123, "version": "1"}

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, ErrMissingTokenName)
}

func TestEnhancePaymentRequirements_MissingTokenVersion(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Extra = map[string]interface{}{"name": "T"}

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, ErrMissingTokenVersion)
}

func TestEnhancePaymentRequirements_EmptyTokenVersion(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Extra = map[string]interface{}{"name": "T", "version": ""}

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, ErrMissingTokenVersion)
}

// === EnhancePaymentRequirements — address validation ===

func TestEnhancePaymentRequirements_InvalidAsset(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.Asset = "too-short"

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, ErrInvalidAsset)
}

func TestEnhancePaymentRequirements_InvalidPayTo(t *testing.T) {
	s := NewExactCasperScheme()
	req := validRequirements()
	req.PayTo = "invalid"

	_, err := s.EnhancePaymentRequirements(context.Background(), req, types.SupportedKind{}, nil)
	assert.ErrorContains(t, err, ErrInvalidPayTo)
}

// === EnhancePaymentRequirements — supportedKind extensions ===

func TestEnhancePaymentRequirements_ExtensionKeysForwarded(t *testing.T) {
	s := NewExactCasperScheme()
	sk := types.SupportedKind{Extra: map[string]interface{}{"customKey": "customValue"}}

	result, err := s.EnhancePaymentRequirements(context.Background(), validRequirements(), sk, []string{"customKey"})
	require.NoError(t, err)
	assert.Equal(t, "customValue", result.Extra["customKey"])
}

func TestEnhancePaymentRequirements_UnknownExtensionKeyIgnored(t *testing.T) {
	s := NewExactCasperScheme()
	sk := types.SupportedKind{Extra: map[string]interface{}{"customKey": "customValue"}}

	result, err := s.EnhancePaymentRequirements(context.Background(), validRequirements(), sk, []string{"otherKey"})
	require.NoError(t, err)
	_, has := result.Extra["otherKey"]
	assert.False(t, has)
}

func TestEnhancePaymentRequirements_NilExtensionKeysDoNotForward(t *testing.T) {
	s := NewExactCasperScheme()
	sk := types.SupportedKind{Extra: map[string]interface{}{"customKey": "customValue"}}

	result, err := s.EnhancePaymentRequirements(context.Background(), validRequirements(), sk, nil)
	require.NoError(t, err)
	_, has := result.Extra["customKey"]
	assert.False(t, has)
}

func TestEnhancePaymentRequirements_NilSupportedKindExtraSafe(t *testing.T) {
	s := NewExactCasperScheme()
	sk := types.SupportedKind{}

	result, err := s.EnhancePaymentRequirements(context.Background(), validRequirements(), sk, []string{"customKey"})
	require.NoError(t, err)
	_, has := result.Extra["customKey"]
	assert.False(t, has)
}
