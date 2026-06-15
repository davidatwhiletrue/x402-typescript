package casper

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
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

func ParseAmount(amount string, decimals int) (*big.Int, error) {
	amount = strings.TrimSpace(amount)
	if !strings.Contains(amount, ".") {
		whole, ok := new(big.Int).SetString(amount, 10)
		if !ok {
			return nil, fmt.Errorf("invalid integer part in amount: %q", amount)
		}

		return whole, nil
	}

	parts := strings.SplitN(amount, ".", 2)

	intPart, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, fmt.Errorf("invalid integer part in amount: %q", parts[0])
	}

	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	result := new(big.Int).Mul(intPart, multiplier)

	if len(parts) == 2 && parts[1] != "" {
		decPart := parts[1]
		if len(decPart) > decimals {
			decPart = decPart[:decimals]
		} else {
			decPart += strings.Repeat("0", decimals-len(decPart))
		}

		parsedDecPart, ok := new(big.Int).SetString(decPart, 10)
		if !ok {
			return nil, fmt.Errorf("invalid decimal part in amount: %q", parts[1])
		}

		result.Add(result, parsedDecPart)
	}

	return result, nil
}

func FormatAmount(amount *big.Int, decimals int) string {
	if amount.Sign() == 0 {
		return "0"
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.DivMod(amount, divisor, remainder)

	if remainder.Sign() == 0 {
		return quotient.String()
	}

	decStr := fmt.Sprintf("%0*s", decimals, remainder.String())
	decStr = strings.TrimRight(decStr, "0")
	return fmt.Sprintf("%s.%s", quotient.String(), decStr)
}
