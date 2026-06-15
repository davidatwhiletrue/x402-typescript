/**
 * Network identifiers for the Casper blockchain.
 */
export const NETWORK_CASPER_MAINNET = "casper:casper";
export const NETWORK_CASPER_TESTNET = "casper:casper-test";

/**
 * Supported Exact payment scheme.
 */
export const SCHEME_EXACT = "exact";

/**
 * Configuration for a Casper network.
 */
export type NetworkConfig = {
  /** Chain name used in transaction payloads, e.g. "casper" or "casper-test". */
  chainName: string;
  /** Default JSON-RPC endpoint for the network. */
  rpcUrl: string;
};

/**
 * Default network configurations keyed by CAIP-2 network identifier.
 */
export const NetworkConfigs: Record<string, NetworkConfig> = {
  [NETWORK_CASPER_MAINNET]: {
    chainName: "casper",
    rpcUrl: "https://node.mainnet.casper.network/rpc",
  },
  [NETWORK_CASPER_TESTNET]: {
    chainName: "casper-test",
    rpcUrl: "https://node.testnet.casper.network/rpc",
  },
};
