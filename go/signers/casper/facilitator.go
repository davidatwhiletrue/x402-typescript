package casper

import (
	"context"
	"fmt"
	"net/http"
	"time"

	mechanismcasper "casper_x402_facilitator/x402/mechanisms/casper"

	casperSDK "github.com/make-software/casper-go-sdk/v2/casper"
	"github.com/make-software/casper-go-sdk/v2/types"
	"github.com/make-software/casper-go-sdk/v2/types/keypair"
)

type FacilitatorSigner struct {
	keys    map[string]keypair.PrivateKey
	rpcURLs map[string]string
}

func NewFacilitatorSigner(
	keys map[string]keypair.PrivateKey,
	rpcURLs map[string]string,
) *FacilitatorSigner {
	return &FacilitatorSigner{keys: keys, rpcURLs: rpcURLs}
}

func (s *FacilitatorSigner) GetNetworkConfig(_ context.Context, network string) (mechanismcasper.NetworkConfig, error) {
	rpcURL := s.rpcURL(network)

	if len(rpcURL) == 0 {
		return mechanismcasper.NetworkConfig{
			"",
			"",
		}, fmt.Errorf("network %s not configured in this signer", network)
	}

	return mechanismcasper.NetworkConfig{
		network,
		rpcURL,
	}, nil
}

func (s *FacilitatorSigner) GetAddresses(_ context.Context, network string) []string {
	key, ok := s.keys[network]
	if !ok {
		return nil
	}

	pubKey := key.PublicKey()
	return []string{pubKey.AccountHash().ToHex()}
}

func (s *FacilitatorSigner) GetPublicKeyHex(_ context.Context, network string) (string, error) {
	key, ok := s.keys[network]
	if !ok {
		return "", fmt.Errorf("no key registered for network %s", network)
	}

	return key.PublicKey().ToHex(), nil
}

func (s *FacilitatorSigner) VerifyEIP712Signature(digest [32]byte, sig [65]byte, publicKey string) (bool, error) {
	pk, err := keypair.NewPublicKey(publicKey)
	if err != nil {
		return false, err
	}
	err = pk.VerifySignature(digest[:], sig[:])
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *FacilitatorSigner) SignTransaction(transaction *types.TransactionV1, network string) error {
	key, ok := s.keys[network]
	if !ok {
		return fmt.Errorf("no key registered for network %s", network)
	}

	return transaction.Sign(key)
}

func (s *FacilitatorSigner) PutTransaction(ctx context.Context, network string, transaction types.TransactionV1) (string, error) {
	handler := casperSDK.NewRPCHandler(s.rpcURL(network), http.DefaultClient)
	client := casperSDK.NewRPCClient(handler)
	result, err := client.PutTransactionV1(ctx, transaction)
	if err != nil {
		return "", fmt.Errorf("PutTransaction RPC failed: %w", err)
	}

	return result.TransactionHash.String(), nil
}

func (s *FacilitatorSigner) WaitForTransaction(ctx context.Context, network string, transactionHash string) error {
	handler := casperSDK.NewRPCHandler(s.rpcURL(network), http.DefaultClient)
	client := casperSDK.NewRPCClient(handler)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			res, err := client.GetTransactionByTransactionHash(ctx, transactionHash)
			if err != nil {
				continue
			}
			if res.ExecutionInfo == nil || res.ExecutionInfo.BlockHeight == 0 ||
				res.ExecutionInfo.ExecutionResult == nil {
				continue
			}

			execResult := res.ExecutionInfo.ExecutionResult
			if execResult.ErrorMessage != nil {
				return fmt.Errorf("transaction execution failed: %s", *execResult.ErrorMessage)
			}

			return nil
		}
	}
}

func (s *FacilitatorSigner) rpcURL(network string) string {
	if url, ok := s.rpcURLs[network]; ok && url != "" {
		return url
	}

	if cfg, err := mechanismcasper.GetNetworkConfig(network); err == nil {
		return cfg.RPCURL
	}

	return ""
}
