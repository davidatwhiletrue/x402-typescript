import {
  PaymentPayloadResult,
  PaymentRequirements,
  SchemeNetworkClient,
} from "@x402/core/types";
import {
  buildDomain,
  CASPER_DOMAIN_TYPES,
  hashTypedData,
} from "@casper-ecosystem/casper-eip-712";
import { ClientCasperSigner } from "../../signer";
import { isValidContractPackageHash } from "../../utils";

const transferWithAuthorizationTypes = {
  TransferWithAuthorization: [
    { name: "from", type: "address" },
    { name: "to", type: "address" },
    { name: "value", type: "uint256" },
    { name: "validAfter", type: "uint256" },
    { name: "validBefore", type: "uint256" },
    { name: "nonce", type: "bytes32" },
  ],
};

/**
 * Encode a byte array as a lowercase hex string.
 *
 * @param bytes - The bytes to encode.
 * @returns Hex string without "0x" prefix.
 */
function hexEncode(bytes: Uint8Array): string {
  return Array.from(bytes, b => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Casper client implementation for the Exact payment scheme.
 */
export class ExactCasperScheme implements SchemeNetworkClient {
  readonly scheme = "exact";

  /**
   * Create a new ExactCasperScheme instance.
   *
   * @param signer - The Casper signer for the payer.
   */
  constructor(private readonly signer: ClientCasperSigner) {}

  /**
   * Create a payment payload for the Exact scheme.
   *
   * @param x402Version - The x402 protocol version.
   * @param paymentRequirements - The payment requirements selected by the client.
   * @returns Promise resolving to the payment payload result.
   */
  async createPaymentPayload(
    x402Version: number,
    paymentRequirements: PaymentRequirements,
  ): Promise<PaymentPayloadResult> {
    if (!isValidContractPackageHash(paymentRequirements.asset)) {
      throw new Error(
        `invalid_exact_casper_client_invalid_asset: ${paymentRequirements.asset}`,
      );
    }

    const name = paymentRequirements.extra?.name;
    const version = paymentRequirements.extra?.version;
    if (typeof name !== "string" || name === "") {
      throw new Error("invalid_exact_casper_client_missing_token_name");
    }
    if (typeof version !== "string" || version === "") {
      throw new Error("invalid_exact_casper_client_missing_token_version");
    }

    const domain = buildDomain(
      name,
      version,
      paymentRequirements.network,
      "0x" + paymentRequirements.asset,
    );

    const now = Math.floor(Date.now() / 1000);
    const validAfter = now - 600;
    const validBefore = now + paymentRequirements.maxTimeoutSeconds;

    const nonce = crypto.getRandomValues(new Uint8Array(32));

    const message = {
      from: "0x" + this.signer.accountAddress(),
      to: "0x" + paymentRequirements.payTo,
      value: BigInt(paymentRequirements.amount),
      validAfter: BigInt(validAfter),
      validBefore: BigInt(validBefore),
      nonce: "0x" + hexEncode(nonce),
    };

    let digest: Uint8Array;
    try {
      digest = hashTypedData(
        domain,
        transferWithAuthorizationTypes,
        "TransferWithAuthorization",
        message,
        { domainTypes: CASPER_DOMAIN_TYPES },
      );
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      throw new Error(`invalid_exact_casper_client_failed_to_hash: ${message}`);
    }

    let signature: Uint8Array;
    try {
      signature = await this.signer.signEIP712(digest);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      throw new Error(`invalid_exact_casper_client_failed_to_sign: ${message}`);
    }

    return {
      x402Version,
      payload: {
        signature: hexEncode(signature),
        publicKey: this.signer.publicKey(),
        authorization: {
          from: this.signer.accountAddress(),
          to: paymentRequirements.payTo,
          value: paymentRequirements.amount,
          validAfter: String(validAfter),
          validBefore: String(validBefore),
          nonce: hexEncode(nonce),
        },
      },
    };
  }
}
