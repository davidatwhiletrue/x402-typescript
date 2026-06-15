package casper

const SchemeExact = "exact"

const (
	NetworkCasperMainnet = "casper:casper"
	NetworkCasperTestnet = "casper:casper-test"
)

var NetworkConfigs = map[string]NetworkConfig{
	NetworkCasperMainnet: {
		ChainName: "casper",
		RPCURL:    "https://node.mainnet.casper.network/rpc",
	},
	NetworkCasperTestnet: {
		ChainName: "casper-test",
		RPCURL:    "https://node.testnet.casper.network/rpc",
	},
}
