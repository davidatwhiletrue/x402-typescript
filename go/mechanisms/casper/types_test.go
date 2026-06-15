package casper_test

import (
	"context"
	"testing"

	"casper_x402_facilitator/x402/mechanisms/casper"

	caspertypes "github.com/make-software/casper-go-sdk/v2/types"
)

func TestConstants(t *testing.T) {
	if casper.SchemeExact != "exact" {
		t.Errorf("SchemeExact = %q, want %q", casper.SchemeExact, "exact")
	}

	if casper.NetworkCasperMainnet != "casper:casper" {
		t.Errorf("NetworkCasperMainnet = %q, want %q", casper.NetworkCasperMainnet, "casper:casper")
	}

	if casper.NetworkCasperTestnet != "casper:casper-test" {
		t.Errorf("NetworkCasperTestnet = %q, want %q", casper.NetworkCasperTestnet, "casper:casper-test")
	}
}

func TestNetworkConfigs(t *testing.T) {
	mainnet, ok := casper.NetworkConfigs[casper.NetworkCasperMainnet]
	if !ok {
		t.Fatal("mainnet config not found")
	}
	if mainnet.ChainName != "casper" {
		t.Errorf("ChainName = %q, want %q", mainnet.ChainName, "casper")
	}
	if mainnet.RPCURL == "" {
		t.Error("RPCURL must not be empty")
	}

	testnet, ok := casper.NetworkConfigs[casper.NetworkCasperTestnet]
	if !ok {
		t.Fatal("testnet config not found")
	}
	if testnet.ChainName != "casper-test" {
		t.Errorf("ChainName = %q, want %q", testnet.ChainName, "casper-test")
	}
}

func TestClientCasperSignerInterface(t *testing.T) {
	var _ casper.ClientCasperSigner = (*mockClientSigner)(nil)
}

type mockClientSigner struct{}

func (m *mockClientSigner) AccountAddress() string { return "" }
func (m *mockClientSigner) PublicKey() string      { return "" }
func (m *mockClientSigner) SignEIP712(_ [32]byte) ([65]byte, error) {
	return [65]byte{}, nil
}

func TestFacilitatorCasperSignerInterface(t *testing.T) {
	var _ casper.FacilitatorCasperSigner = (*mockFacilitatorSigner)(nil)
}

type mockFacilitatorSigner struct{}

func (m *mockFacilitatorSigner) GetNetworkConfig(_ context.Context, _ string) (casper.NetworkConfig, error) {
	return casper.NetworkConfig{}, nil
}
func (m *mockFacilitatorSigner) GetAddresses(_ context.Context, _ string) []string { return nil }
func (m *mockFacilitatorSigner) GetPublicKeyHex(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (m *mockFacilitatorSigner) VerifyEIP712Signature(_ [32]byte, _ [65]byte, _ string) (bool, error) {
	return true, nil
}
func (m *mockFacilitatorSigner) SignTransaction(_ *caspertypes.TransactionV1, _ string) error {
	return nil
}
func (m *mockFacilitatorSigner) PutTransaction(_ context.Context, _ string, _ caspertypes.TransactionV1) (string, error) {
	return "", nil
}
func (m *mockFacilitatorSigner) WaitForTransaction(_ context.Context, _ string, _ string) error {
	return nil
}
