package casper

import (
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var addressHashRegex = regexp.MustCompile(`^(00|01)[0-9a-fA-F]{64}$`)
var contractPackageHashRegex = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

func GetNetworkConfig(network string) (*NetworkConfig, error) {
	cfg, ok := NetworkConfigs[network]
	if !ok {
		return nil, fmt.Errorf("unsupported Casper network: %s", network)
	}

	return &cfg, nil
}

func ChainNameFromNetwork(network string) string {
	parts := strings.SplitN(network, ":", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return network
}

func IsValidAddress(s string) bool {
	return addressHashRegex.MatchString(s)
}

func IsValidContractPackageHash(s string) bool {
	return contractPackageHashRegex.MatchString(s)
}

func DecodeContractPackageHash(hexStr string) ([32]byte, error) {
	var result [32]byte
	if len(hexStr) != 64 {
		return result, fmt.Errorf("contract_package_hash must be 64 hex chars, got %d", len(hexStr))
	}

	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return result, fmt.Errorf("invalid hex in contract_package_hash: %w", err)
	}

	copy(result[:], b)
	return result, nil
}

// GetAssetInfo returns information about an asset on a network
func GetAssetInfo(network string, assetSymbol string) (*AssetInfo, error) {
	if found := FindDefaultAsset(assetSymbol, network); found != nil {
		return defaultAssetToAssetInfo(found), nil
	}

	info, err := GetDefaultAsset(network, "")
	if err != nil {
		return nil, err
	}
	return defaultAssetToAssetInfo(info), nil
}

// ParseAmount converts a decimal string amount to token smallest units
func ParseAmount(amount string, decimals int) (uint64, error) {
	// Remove any whitespace
	amount = strings.TrimSpace(amount)

	// Parse the decimal amount
	parts := strings.Split(amount, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount format: %s", amount)
	}

	// Parse integer part
	intPart, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer part: %s", parts[0])
	}

	// Handle decimal part
	decPart := uint64(0)
	if len(parts) == 2 && parts[1] != "" {
		// Pad or truncate decimal part to match token decimals
		decStr := parts[1]
		if len(decStr) > decimals {
			decStr = decStr[:decimals]
		} else {
			decStr += strings.Repeat("0", decimals-len(decStr))
		}

		decPart, err = strconv.ParseUint(decStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid decimal part: %s", parts[1])
		}
	}

	// Calculate total in smallest unit
	multiplier := uint64(math.Pow10(decimals))
	result := intPart*multiplier + decPart

	return result, nil
}

// FormatAmount converts an amount in smallest units to a decimal string
func FormatAmount(amount uint64, decimals int) string {
	if amount == 0 {
		return "0"
	}

	divisor := uint64(math.Pow10(decimals))
	quotient := amount / divisor
	remainder := amount % divisor

	// Format the decimal part with leading zeros
	decStr := fmt.Sprintf("%0*d", decimals, remainder)

	// Remove trailing zeros
	decStr = strings.TrimRight(decStr, "0")

	if decStr == "" {
		return fmt.Sprintf("%d", quotient)
	}

	return fmt.Sprintf("%d.%s", quotient, decStr)
}
