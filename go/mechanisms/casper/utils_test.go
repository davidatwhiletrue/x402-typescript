package casper_test

import (
	"math/big"
	"testing"

	"casper_x402_facilitator/x402/mechanisms/casper"
)

func TestGetNetworkConfig_Mainnet(t *testing.T) {
	cfg, err := casper.GetNetworkConfig("casper:casper")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ChainName != "casper" {
		t.Errorf("ChainName = %q, want %q", cfg.ChainName, "casper")
	}
}

func TestGetNetworkConfig_Testnet(t *testing.T) {
	cfg, err := casper.GetNetworkConfig("casper:casper-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ChainName != "casper-test" {
		t.Errorf("ChainName = %q, want %q", cfg.ChainName, "casper-test")
	}
}

func TestGetNetworkConfig_Unsupported(t *testing.T) {
	_, err := casper.GetNetworkConfig("casper:unknown")
	if err == nil {
		t.Error("expected error for unsupported network, got nil")
	}
}

func TestIsValidAccountHash_Valid(t *testing.T) {
	valid := "00" + "aabbccddeeff0011223344556677889900aabbccddeeff0011223344556677889900"[:64]
	if !casper.IsValidAddress(valid) {
		t.Errorf("expected %q to be valid", valid)
	}

	valid2 := "01" + "aabbccddeeff0011223344556677889900aabbccddeeff0011223344556677889900"[:64]
	if !casper.IsValidAddress(valid2) {
		t.Errorf("expected %q to be valid", valid2)
	}
}

func TestIsValidAccountHash_Invalid(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"too short", "00aabb"},
		{"wrong prefix", "02" + "aabbccddeeff0011223344556677889900aabbccddeeff0011223344556677889900"[:64]},
		{"not hex", "0z" + "aabbccddeeff0011223344556677889900aabbccddeeff0011223344556677889900"[:64]},
		{"empty", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if casper.IsValidAddress(tc.input) {
				t.Errorf("expected %q to be invalid", tc.input)
			}
		})
	}
}

func TestIsValidContractPackageHash_Valid(t *testing.T) {
	valid := "aabbccddeeff0011223344556677889900aabbccddeeff0011223344556677889900"[:64]
	if !casper.IsValidContractPackageHash(valid) {
		t.Errorf("expected %q to be valid", valid)
	}
}

func TestIsValidContractPackageHash_Invalid(t *testing.T) {
	cases := []string{
		"tooshort",
		"aabbccddeeff0011223344556677889900aabbccddeeff0011223344556677889900",
		"ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
		"",
	}

	for _, c := range cases {
		if casper.IsValidContractPackageHash(c) {
			t.Errorf("expected %q to be invalid", c)
		}
	}
}

func TestParseAmount_Whole(t *testing.T) {
	result, err := casper.ParseAmount("1000000", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := new(big.Int).SetUint64(1000000)
	if result.Cmp(expected) != 0 {
		t.Errorf("ParseAmount(\"1000000\", 6) = %s, want %s", result, expected)
	}
}

func TestParseAmount_Decimal(t *testing.T) {
	result, err := casper.ParseAmount("1.5", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := new(big.Int).SetUint64(1500000)
	if result.Cmp(expected) != 0 {
		t.Errorf("ParseAmount(\"1.5\", 6) = %s, want %s", result, expected)
	}
}

func TestParseAmount_Invalid(t *testing.T) {
	_, err := casper.ParseAmount("not-a-number", 6)
	if err == nil {
		t.Error("expected error for invalid amount")
	}
}

func TestChainNameFromNetwork(t *testing.T) {
	tests := []struct {
		network   string
		chainName string
	}{
		{"casper:casper", "casper"},
		{"casper:casper-test", "casper-test"},
	}

	for _, tt := range tests {
		got := casper.ChainNameFromNetwork(tt.network)
		if got != tt.chainName {
			t.Errorf("ChainNameFromNetwork(%q) = %q, want %q", tt.network, got, tt.chainName)
		}
	}
}

func TestDecodeContractPackageHash(t *testing.T) {
	hex64 := "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788"
	got, err := casper.DecodeContractPackageHash(hex64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 32 {
		t.Errorf("expected 32-byte array, got %d bytes", len(got))
	}
	if got == ([32]byte{}) {
		t.Error("decoded contract package hash must not be zero")
	}
}
