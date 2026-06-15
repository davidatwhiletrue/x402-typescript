package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"casper_x402_facilitator/x402/mechanisms/casper"

	x402 "github.com/x402-foundation/x402/go"
	"github.com/x402-foundation/x402/go/types"
)

type assetKey struct {
	network string
	asset   string
}

type ExactCasperScheme struct {
	moneyParsers  []x402.MoneyParser
	assetDecimals map[assetKey]int
}

func NewExactCasperScheme() *ExactCasperScheme {
	return &ExactCasperScheme{assetDecimals: make(map[assetKey]int)}
}

func (s *ExactCasperScheme) Scheme() string {
	return casper.SchemeExact
}

func (s *ExactCasperScheme) RegisterMoneyParser(parser x402.MoneyParser) *ExactCasperScheme {
	s.moneyParsers = append(s.moneyParsers, parser)
	return s
}

func (s *ExactCasperScheme) RegisterAsset(network, asset string, decimals int) *ExactCasperScheme {
	s.assetDecimals[assetKey{network: network, asset: asset}] = decimals
	return s
}

func (s *ExactCasperScheme) GetAssetDecimals(asset string, network x402.Network) int {
	if d, ok := s.assetDecimals[assetKey{network: string(network), asset: asset}]; ok {
		return d
	}

	return 9
}

func (s *ExactCasperScheme) ParsePrice(price x402.Price, network x402.Network) (x402.AssetAmount, error) {
	if priceMap, ok := price.(map[string]interface{}); ok {
		amountValue, hasAmount := priceMap["amount"]
		if hasAmount {
			amount, ok := amountValue.(string)
			if !ok {
				return x402.AssetAmount{}, errors.New(ErrAmountMustBeString)
			}

			assetValue, ok := priceMap["asset"]
			if !ok {
				return x402.AssetAmount{}, fmt.Errorf("%s: asset field required", ErrInvalidAsset)
			}
			asset, ok := assetValue.(string)
			if !ok || !casper.IsValidContractPackageHash(asset) {
				return x402.AssetAmount{}, fmt.Errorf("%s: %v", ErrInvalidAsset, assetValue)
			}

			extra := make(map[string]interface{})
			if extraValue, hasExtra := priceMap["extra"]; hasExtra {
				if extraMap, ok := extraValue.(map[string]interface{}); ok {
					extra = extraMap
				}
			}

			return x402.AssetAmount{Amount: amount, Asset: asset, Extra: extra}, nil
		}
	}

	decimalAmount, err := parseMoneyToDecimal(price)
	if err != nil {
		return x402.AssetAmount{}, err
	}

	for _, parser := range s.moneyParsers {
		assetAmount, err := parser(decimalAmount, network)
		if err != nil {
			continue
		}
		if assetAmount != nil {
			return *assetAmount, nil
		}
	}

	return x402.AssetAmount{}, fmt.Errorf(
		"%s: no default asset configured for network %s; provide an explicit AssetAmount with 'amount', 'asset', and 'extra' fields, or register a MoneyParser",
		ErrNoDefaultAsset,
		network,
	)
}

func parseMoneyToDecimal(price x402.Price) (float64, error) {
	switch v := price.(type) {
	case string:
		clean := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), "$"))
		f, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return 0, fmt.Errorf("%s: %q", ErrInvalidPriceFormat, v)
		}
		return f, nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("%s: unsupported type %T", ErrInvalidPriceFormat, price)
	}
}

func (s *ExactCasperScheme) EnhancePaymentRequirements(
	_ context.Context,
	requirements types.PaymentRequirements,
	supportedKind types.SupportedKind,
	extensionKeys []string,
) (types.PaymentRequirements, error) {
	if !casper.IsValidContractPackageHash(requirements.Asset) {
		return requirements, fmt.Errorf("%s: %s", ErrInvalidAsset, requirements.Asset)
	}

	if !casper.IsValidAddress(requirements.PayTo) {
		return requirements, fmt.Errorf("%s: %s", ErrInvalidPayTo, requirements.PayTo)
	}

	if strings.Contains(requirements.Amount, ".") {
		decimals := s.GetAssetDecimals(requirements.Asset, x402.Network(requirements.Network))
		parsedAmount, err := casper.ParseAmount(requirements.Amount, decimals)
		if err != nil {
			return requirements, fmt.Errorf("%s: %w", ErrFailedToParseAmount, err)
		}
		requirements.Amount = parsedAmount.String()
	}

	if requirements.Extra == nil {
		requirements.Extra = map[string]interface{}{}
	}

	if name, ok := requirements.Extra["name"].(string); !ok || name == "" {
		return requirements, errors.New(ErrMissingTokenName)
	}
	if version, ok := requirements.Extra["version"].(string); !ok || version == "" {
		return requirements, errors.New(ErrMissingTokenVersion)
	}

	if supportedKind.Extra != nil {
		for _, key := range extensionKeys {
			if value, ok := supportedKind.Extra[key]; ok {
				requirements.Extra[key] = value
			}
		}
	}

	return requirements, nil
}
