# Casper Network x402 Mechanism — Design Spec

**Date:** 2026-06-15  
**Status:** Draft (pending implementation plan)  
**Goal:** Add a TypeScript Casper Network mechanism (`@x402/casper`) so that `examples/typescript/clients/advanced` can pay for resources using the Exact payment scheme on Casper.

---

## 1. Goal & scope

Create a new TypeScript mechanism package `typescript/packages/mechanisms/casper` that lets the `examples/typescript/clients/advanced` example pay for resources over the Casper Network.

### In scope (client-only, first iteration)

- `@x402/casper` package implementing the **Exact** scheme client (`SchemeNetworkClient`).
- Client signer factory and interface (`createClientCasperSigner`, `ClientCasperSigner`).
- Validation utilities and network constants matching the Go reference implementation.
- Updates to all four advanced example patterns (`all-networks`, `builder-pattern`, `hooks`, `preferred-network`) so Casper can be registered alongside the other networks.

### Out of scope for this iteration

- Server-side `SchemeNetworkServer` and facilitator `SchemeNetworkFacilitator`.
- V1 x402 compatibility layer.

These are intentionally left as extension points so they can be added later without breaking the client API.

---

## 2. Dependencies

Inside `typescript/packages/mechanisms/casper/package.json`:

```json
{
  "dependencies": {
    "@x402/core": "workspace:~",
    "casper-js-sdk": "5.0.12",
    "@casper-ecosystem/casper-eip-712": "^1.2.1"
  }
}
```

- `casper-js-sdk@5.0.12` — key management, account-hash derivation, signing.
- `@casper-ecosystem/casper-eip-712` — Casper-specific EIP-712 domain and message hashing for `TransferWithAuthorization`.

Peer/dev dependencies mirror the other mechanism packages (`typescript`, `tsup`, `vitest`, `eslint`, `prettier`, etc.).

---

## 3. Package structure

```
typescript/packages/mechanisms/casper/
├── package.json
├── tsconfig.json
├── tsup.config.ts
├── vitest.config.ts
├── src/
│   ├── index.ts
│   ├── signer.ts
│   ├── types.ts
│   ├── constants.ts
│   ├── utils.ts
│   └── exact/
│       └── client/
│           ├── index.ts
│           └── scheme.ts
└── test/
    └── exact/
        └── client/
            └── scheme.test.ts
```

Exports mirror `@x402/aptos`:

- `@x402/casper` — top-level exports.
- `@x402/casper/exact/client` — `ExactCasperScheme` and helpers.

---

## 4. Client signer interface & factory

`src/signer.ts`:

```typescript
import { KeyAlgorithm } from "casper-js-sdk";

export interface ClientCasperSigner {
  /** "00" + account hash, 66 hex chars. */
  accountAddress(): string;
  /** Full public key hex, e.g. "01..." (ed25519) or "02..." (secp256k1). */
  publicKey(): string;
  /** Sign a 32-byte EIP-712 digest and return a 65-byte signature [algo_byte | 64 raw sig bytes]. */
  signEIP712(digest: Uint8Array): Promise<Uint8Array>;
}

export async function createClientCasperSigner(
  pemPath: string,
  algorithm: KeyAlgorithm = KeyAlgorithm.ED25519,
): Promise<ClientCasperSigner>;
```

### Implementation

1. Read the PEM file with Node `fs/promises`.
2. `const privateKey = PrivateKey.fromPem(pemContent, algorithm);`
3. `accountAddress = "00" + privateKey.publicKey.accountHash().toHex()`
4. `publicKey = privateKey.publicKey.toHex()`
5. `signEIP712(digest)` returns `privateKey.signAndAddAlgorithmBytes(digest)` (this already matches the Go `[algo_byte | raw_sig]` format).

A convenience `toClientCasperSigner(privateKey: PrivateKey): ClientCasperSigner` is also exported for callers who already have a `casper-js-sdk` `PrivateKey`.

---

## 5. Payload types

`src/types.ts`:

```typescript
export type ExactCasperAuthorization = {
  from: string;
  to: string;
  value: string;
  validAfter: string;
  validBefore: string;
  nonce: string;
};

export type ExactCasperPayload = {
  signature: string;
  publicKey: string;
  authorization: ExactCasperAuthorization;
};
```

These match the Go `ExactCasperPayload`/`ExactCasperAuthorization` structs exactly so the existing facilitator can deserialize them.

---

## 6. Exact scheme client algorithm

`src/exact/client/scheme.ts`:

```typescript
export class ExactCasperScheme implements SchemeNetworkClient {
  readonly scheme = "exact";

  constructor(private readonly signer: ClientCasperSigner) {}

  async createPaymentPayload(
    x402Version: number,
    paymentRequirements: PaymentRequirements,
  ): Promise<PaymentPayloadResult> { ... }
}
```

### Algorithm

Mirrors `go/mechanisms/casper/exact/client/scheme.go`:

1. Validate `paymentRequirements.asset` is a 64-hex-char contract package hash.
2. Read `paymentRequirements.extra.name` and `paymentRequirements.extra.version`; fail if missing.
3. Build the EIP-712 domain:
   ```typescript
   const domain = buildDomain(
     name,
     version,
     paymentRequirements.network, // "casper:casper" or "casper:casper-test"
     paymentRequirements.asset,
   );
   ```
4. Compute timestamps:
   - `now = Math.floor(Date.now() / 1000)`
   - `validAfter = now - 600`
   - `validBefore = now + paymentRequirements.maxTimeoutSeconds`
5. Generate a random 32-byte nonce.
6. Build the message:
   ```typescript
   const message = {
     from: "0x" + this.signer.accountAddress(),
     to: "0x" + paymentRequirements.payTo,
     value: BigInt(paymentRequirements.amount),
     validAfter: BigInt(validAfter),
     validBefore: BigInt(validBefore),
     nonce: nonceBytes,
   };
   ```
7. Hash with:
   ```typescript
   const digest = hashTypedData(
     domain,
     transferWithAuthorizationTypes,
     "TransferWithAuthorization",
     message,
     { domainTypes: CASPER_DOMAIN_TYPES },
   );
   ```
8. Sign: `const signature = await this.signer.signEIP712(digest);`
9. Return:
   ```typescript
   {
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
         nonce: hexEncode(nonceBytes),
       },
     },
   }
   ```

### Validation errors

Use the same error codes as Go where applicable:

- `invalid_exact_casper_client_unsupported_network`
- `invalid_exact_casper_client_invalid_asset`
- `invalid_exact_casper_client_missing_token_name`
- `invalid_exact_casper_client_missing_token_version`
- `invalid_exact_casper_client_failed_to_sign`
- `invalid_exact_casper_client_failed_to_hash`

---

## 7. Constants & utilities

`src/constants.ts`:

```typescript
export const SCHEME_EXACT = "exact";

export const NETWORK_CASPER_MAINNET = "casper:casper";
export const NETWORK_CASPER_TESTNET = "casper:casper-test";

export type NetworkConfig = {
  chainName: string;
  rpcUrl: string;
};

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
```

`src/utils.ts`:

```typescript
export function isValidAddress(value: string): boolean;
export function isValidContractPackageHash(value: string): boolean;
export function decodeContractPackageHash(value: string): Uint8Array;
export function parseAmount(amount: string, decimals: number): bigint;
export function formatAmount(amount: bigint, decimals: number): string;
export function chainNameFromNetwork(network: string): string;
export function getNetworkConfig(network: string): NetworkConfig;
```

These mirror the Go helpers in `go/mechanisms/casper/utils.go` so future server/facilitator code can reuse them.

---

## 8. Example integration

### Dependency

`examples/typescript/clients/advanced/package.json` adds:

```json
"@x402/casper": "workspace:*"
```

### Environment variables

```bash
CASPER_PRIVATE_KEY_PATH=/path/to/casper_key.pem
CASPER_KEY_ALGORITHM=ed25519   # or secp256k1
```

### Pattern updates

- `all_networks.ts`: register `casper:*` with `ExactCasperScheme` when `CASPER_PRIVATE_KEY_PATH` is set.
- `builder-pattern.ts`: register `casper:*` and optionally a mainnet override.
- `hooks.ts`: register `casper:*` with `ExactCasperScheme`.
- `preferred-network.ts`: add `"casper:"` to the preference list and register `casper:*`.

### Example snippet

```typescript
import { KeyAlgorithm } from "casper-js-sdk";
import { createClientCasperSigner } from "@x402/casper";
import { ExactCasperScheme } from "@x402/casper/exact/client";

const casperSigner = await createClientCasperSigner(
  process.env.CASPER_PRIVATE_KEY_PATH,
  process.env.CASPER_KEY_ALGORITHM === "secp256k1"
    ? KeyAlgorithm.SECP256K1
    : KeyAlgorithm.ED25519,
);

client.register("casper:*", new ExactCasperScheme(casperSigner));
```

---

## 9. Testing approach

Unit tests in `test/exact/client/scheme.test.ts`:

- Validate payload shape matches `ExactCasperPayload`.
- Verify invalid asset / missing token name / missing token version throw expected errors.
- Verify `validAfter`/`validBefore` are within expected bounds.
- Verify the returned signature is 130 hex chars (65 bytes) and the public key matches the signer.
- Use a test vector approach where possible: create a payload and assert it can be re-hashed to the same digest (without needing a live facilitator).

No integration tests against a live Casper node in this iteration.

---

## 10. Risks & open questions

1. **Signature format parity.** `casper-js-sdk`'s `signAndAddAlgorithmBytes` must produce the exact same 65-byte layout the Go facilitator verifies. The implementation will include a test that compares the first byte to the algorithm identifier.
2. **`@casper-ecosystem/casper-eip-712` message shape.** Need to confirm the package accepts `value`/`validAfter`/`validBefore` as `bigint` and `nonce` as `Uint8Array`, or whether strings/hex are required. The spec assumes compatibility with the Go crate's TypeScript companion; if not, a thin adapter will be added.
3. **Browser usage.** The PEM-reading factory is Node-specific. Browser clients can use `toClientCasperSigner(PrivateKey.fromPem(pemString, ...))` instead.
4. **Server/facilitator future work.** This design leaves clear extension points so server and facilitator can be added later without breaking the client API.
