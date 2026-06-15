import { NetworkConfig, NetworkConfigs } from "./constants";

const addressHashRegex = /^(00|01)[0-9a-fA-F]{64}$/;
const contractPackageHashRegex = /^[0-9a-fA-F]{64}$/;

/**
 * Decode a hex string into a Uint8Array. Works in Node and browsers.
 *
 * @param hex - Hex string, with or without "0x" prefix.
 * @returns Decoded bytes.
 */
function hexToBytes(hex: string): Uint8Array {
  const cleaned = hex.startsWith("0x") || hex.startsWith("0X") ? hex.slice(2) : hex;
  if (cleaned.length % 2 !== 0) {
    throw new Error("hex string must have an even number of characters");
  }
  const bytes = new Uint8Array(cleaned.length / 2);
  for (let i = 0; i < cleaned.length; i += 2) {
    bytes[i / 2] = parseInt(cleaned.substring(i, i + 2), 16);
  }
  return bytes;
}

/**
 * Check whether a string is a valid Casper account-hash address.
 * Valid addresses are 66 hex characters prefixed with "00" (account hash).
 *
 * @param value - The string to validate.
 * @returns True when the value is a valid address.
 */
export function isValidAddress(value: string): boolean {
  return addressHashRegex.test(value);
}

/**
 * Check whether a string is a valid 64-character contract package hash.
 *
 * @param value - The string to validate.
 * @returns True when the value is a valid contract package hash.
 */
export function isValidContractPackageHash(value: string): boolean {
  return contractPackageHashRegex.test(value);
}

/**
 * Decode a 64-character contract package hash into 32 raw bytes.
 *
 * @param value - The hex string to decode.
 * @returns The 32-byte Uint8Array.
 * @throws Error if the input is not 64 valid hex characters.
 */
export function decodeContractPackageHash(value: string): Uint8Array {
  if (!isValidContractPackageHash(value)) {
    throw new Error(`contract_package_hash must be 64 hex chars, got ${value.length}`);
  }
  const bytes = hexToBytes(value);
  if (bytes.length !== 32) {
    throw new Error("invalid hex in contract_package_hash");
  }
  return bytes;
}

/**
 * Convert a decimal amount string to atomic units.
 *
 * @param amount - Decimal amount, e.g. "1.5".
 * @param decimals - Token decimals.
 * @returns Atomic amount as bigint.
 */
export function parseAmount(amount: string, decimals: number): bigint {
  const trimmed = amount.trim();
  if (!trimmed.includes(".")) {
    return BigInt(trimmed);
  }

  const [intPart, fracPartRaw] = trimmed.split(".");
  const multiplier = 10n ** BigInt(decimals);
  let result = BigInt(intPart || "0") * multiplier;

  if (fracPartRaw) {
    const fracPart = fracPartRaw.slice(0, decimals).padEnd(decimals, "0");
    result += BigInt(fracPart);
  }

  return result;
}

/**
 * Convert an atomic amount back to a decimal string.
 *
 * @param amount - Atomic amount.
 * @param decimals - Token decimals.
 * @returns Decimal string.
 */
export function formatAmount(amount: bigint, decimals: number): string {
  if (amount === 0n) {
    return "0";
  }

  const divisor = 10n ** BigInt(decimals);
  const quotient = amount / divisor;
  const remainder = amount % divisor;

  if (remainder === 0n) {
    return quotient.toString();
  }

  const decStr = remainder.toString().padStart(decimals, "0").replace(/0+$/, "");
  return `${quotient}.${decStr}`;
}

/**
 * Extract the chain name portion from a CAIP-2 Casper network identifier.
 * E.g. "casper:casper-test" returns "casper-test".
 *
 * @param network - CAIP-2 network identifier.
 * @returns Chain name.
 */
export function chainNameFromNetwork(network: string): string {
  const parts = network.split(":");
  return parts.length === 2 ? parts[1] : network;
}

/**
 * Look up the default configuration for a Casper network.
 *
 * @param network - CAIP-2 network identifier.
 * @returns The network configuration.
 * @throws Error if the network is not supported.
 */
export function getNetworkConfig(network: string): NetworkConfig {
  const config = NetworkConfigs[network];
  if (!config) {
    throw new Error(`unsupported Casper network: ${network}`);
  }
  return config;
}
