import { KeyAlgorithm, PrivateKey } from "casper-js-sdk";
import { readFile } from "fs/promises";

/**
 * Client-side signer for Casper x402 payments.
 */
export interface ClientCasperSigner {
  /**
   * Get the payer account-hash address as a 66-character hex string
   * prefixed with "00".
   *
   * @returns Account-hash address.
   */
  accountAddress(): string;

  /**
   * Get the payer's full public key hex, including the algorithm prefix byte.
   *
   * @returns Public key hex.
   */
  publicKey(): string;

  /**
   * Sign a 32-byte EIP-712 digest.
   *
   * @param digest - 32-byte digest to sign.
   * @returns 65-byte signature: [1 algorithm byte | 64 raw signature bytes].
   */
  signEIP712(digest: Uint8Array): Promise<Uint8Array>;
}

/**
 * Wrap an existing casper-js-sdk PrivateKey into a ClientCasperSigner.
 *
 * @param privateKey - The Casper private key.
 * @returns A ClientCasperSigner instance.
 */
export function toClientCasperSigner(privateKey: PrivateKey): ClientCasperSigner {
  const accountAddress = "00" + privateKey.publicKey.accountHash().toHex();
  const publicKey = privateKey.publicKey.toHex();

  return {
    accountAddress: () => accountAddress,
    publicKey: () => publicKey,
    signEIP712: async digest => privateKey.signAndAddAlgorithmBytes(digest),
  };
}

/**
 * Create a ClientCasperSigner from a PEM private-key file.
 *
 * @param pemPath - Path to the PEM-encoded private key file.
 * @param algorithm - Key algorithm, defaults to ed25519.
 * @returns A ClientCasperSigner instance.
 */
export async function createClientCasperSigner(
  pemPath: string,
  algorithm: KeyAlgorithm = KeyAlgorithm.ED25519,
): Promise<ClientCasperSigner> {
  const pemContent = await readFile(pemPath, "utf-8");
  const privateKey = PrivateKey.fromPem(pemContent, algorithm);
  return toClientCasperSigner(privateKey);
}
