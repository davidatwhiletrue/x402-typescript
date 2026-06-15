/**
 * @module @x402/casper - x402 Payment Protocol Casper Implementation
 *
 * This module provides the Casper-specific client implementation of the x402 payment protocol.
 */

// Exact scheme client
export { ExactCasperScheme } from "./exact/client";

// Signers
export { createClientCasperSigner, toClientCasperSigner } from "./signer";
export type { ClientCasperSigner } from "./signer";

// Types
export type { ExactCasperAuthorization, ExactCasperPayload } from "./types";

// Constants
export {
  NETWORK_CASPER_MAINNET,
  NETWORK_CASPER_TESTNET,
  NetworkConfigs,
  SCHEME_EXACT,
} from "./constants";
export type { NetworkConfig } from "./constants";

// Utils
export {
  chainNameFromNetwork,
  decodeContractPackageHash,
  formatAmount,
  getNetworkConfig,
  isValidAddress,
  isValidContractPackageHash,
  parseAmount,
} from "./utils";
