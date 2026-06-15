import { KeyAlgorithm, PrivateKey } from "casper-js-sdk";
import { describe, expect, it } from "vitest";
import { ExactCasperScheme } from "../../../src/exact/client/scheme";
import { toClientCasperSigner } from "../../../src/signer";
import { isValidAddress, isValidContractPackageHash } from "../../../src/utils";

const testAsset = "aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788";
const testPayTo = "00aabbccddeeff0011223344556677889900aabbccddeeff001122334455667788";

function createTestSigner() {
  const privateKey = PrivateKey.generate(KeyAlgorithm.ED25519);
  return toClientCasperSigner(privateKey);
}

function buildRequirements(
  overrides?: Partial<{
    asset: string;
    amount: string;
    payTo: string;
    extra: Record<string, unknown>;
  }>,
): Parameters<ExactCasperScheme["createPaymentPayload"]>[1] {
  return {
    scheme: "exact",
    network: "casper:casper-test",
    asset: testAsset,
    amount: "1000000",
    payTo: testPayTo,
    maxTimeoutSeconds: 300,
    extra: {
      name: "TestToken",
      version: "1",
    },
    ...overrides,
  };
}

describe("ExactCasperScheme", () => {
  it("returns a payload with valid shape", async () => {
    const scheme = new ExactCasperScheme(createTestSigner());
    const result = await scheme.createPaymentPayload(2, buildRequirements());

    expect(result.x402Version).toBe(2);
    expect(result.payload).toMatchObject({
      signature: expect.any(String),
      publicKey: expect.any(String),
      authorization: {
        from: expect.any(String),
        to: testPayTo,
        value: "1000000",
        validAfter: expect.any(String),
        validBefore: expect.any(String),
        nonce: expect.any(String),
      },
    });

    const payload = result.payload as {
      signature: string;
      publicKey: string;
      authorization: { from: string };
    };

    expect(payload.signature).toMatch(/^[0-9a-fA-F]{130}$/);
    expect(payload.publicKey).toMatch(/^[0-9a-fA-F]{66}$/);
    expect(isValidAddress(payload.authorization.from)).toBe(true);
    expect(isValidContractPackageHash(testAsset)).toBe(true);
  });

  it("sets validAfter 600 seconds in the past and validBefore relative to maxTimeout", async () => {
    const now = Math.floor(Date.now() / 1000);
    const scheme = new ExactCasperScheme(createTestSigner());
    const result = await scheme.createPaymentPayload(2, buildRequirements());

    const auth = (result.payload as { authorization: { validAfter: string; validBefore: string } })
      .authorization;
    const validAfter = Number(auth.validAfter);
    const validBefore = Number(auth.validBefore);

    expect(validAfter).toBeGreaterThanOrEqual(now - 605);
    expect(validAfter).toBeLessThanOrEqual(now - 595);
    expect(validBefore).toBeGreaterThanOrEqual(now + 295);
    expect(validBefore).toBeLessThanOrEqual(now + 305);
  });

  it("throws for invalid asset", async () => {
    const scheme = new ExactCasperScheme(createTestSigner());
    await expect(
      scheme.createPaymentPayload(2, buildRequirements({ asset: "bad" })),
    ).rejects.toThrow("invalid_exact_casper_client_invalid_asset");
  });

  it("throws for invalid payTo", async () => {
    const scheme = new ExactCasperScheme(createTestSigner());
    await expect(
      scheme.createPaymentPayload(2, buildRequirements({ payTo: "bad" })),
    ).rejects.toThrow("invalid_exact_casper_client_invalid_pay_to");
  });

  it("throws when token name is missing", async () => {
    const scheme = new ExactCasperScheme(createTestSigner());
    await expect(
      scheme.createPaymentPayload(2, buildRequirements({ extra: { version: "1" } })),
    ).rejects.toThrow("invalid_exact_casper_client_missing_token_name");
  });

  it("throws when token version is missing", async () => {
    const scheme = new ExactCasperScheme(createTestSigner());
    await expect(
      scheme.createPaymentPayload(2, buildRequirements({ extra: { name: "TestToken" } })),
    ).rejects.toThrow("invalid_exact_casper_client_missing_token_version");
  });

  it("produces a unique nonce for each payload", async () => {
    const scheme = new ExactCasperScheme(createTestSigner());
    const result1 = await scheme.createPaymentPayload(2, buildRequirements());
    const result2 = await scheme.createPaymentPayload(2, buildRequirements());

    const nonce1 = (result1.payload as { authorization: { nonce: string } }).authorization.nonce;
    const nonce2 = (result2.payload as { authorization: { nonce: string } }).authorization.nonce;

    expect(nonce1).not.toBe(nonce2);
    expect(nonce1).toMatch(/^[0-9a-fA-F]{64}$/);
    expect(nonce2).toMatch(/^[0-9a-fA-F]{64}$/);
  });

  it.each([
    { algorithm: KeyAlgorithm.ED25519, expectedByte: "01" },
    { algorithm: KeyAlgorithm.SECP256K1, expectedByte: "02" },
  ])("signature algorithm byte matches $algorithm", async ({ algorithm, expectedByte }) => {
    const privateKey = PrivateKey.generate(algorithm);
    const scheme = new ExactCasperScheme(toClientCasperSigner(privateKey));
    const result = await scheme.createPaymentPayload(2, buildRequirements());

    const signature = (result.payload as { signature: string }).signature;
    expect(signature).toMatch(/^[0-9a-fA-F]{130}$/);
    expect(signature.slice(0, 2).toLowerCase()).toBe(expectedByte);
  });
});
