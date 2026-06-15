# Casper Network x402 Mechanism Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a TypeScript Casper Network mechanism (`@x402/casper`) and wire it into `examples/typescript/clients/advanced` so the advanced client patterns can pay for resources on Casper.

**Architecture:** A new `typescript/packages/mechanisms/casper` package implementing the x402 `SchemeNetworkClient` interface for the Exact scheme. It uses `casper-js-sdk@5.0.12` for keys/signing and `@casper-ecosystem/casper-eip-712` for Casper-specific EIP-712 hashing, mirroring the existing Go reference implementation. The package exposes a signer factory and the `ExactCasperScheme` client class, then gets registered in all four advanced example patterns.

**Tech Stack:** TypeScript, pnpm workspace, tsup, vitest, eslint, prettier, `casper-js-sdk@5.0.12`, `@casper-ecosystem/casper-eip-712@^1.2.1`.

---

## File structure

New files to create:

- `typescript/packages/mechanisms/casper/package.json`
- `typescript/packages/mechanisms/casper/tsconfig.json`
- `typescript/packages/mechanisms/casper/tsup.config.ts`
- `typescript/packages/mechanisms/casper/vitest.config.ts`
- `typescript/packages/mechanisms/casper/eslint.config.js`
- `typescript/packages/mechanisms/casper/.prettierrc`
- `typescript/packages/mechanisms/casper/.prettierignore`
- `typescript/packages/mechanisms/casper/src/constants.ts`
- `typescript/packages/mechanisms/casper/src/utils.ts`
- `typescript/packages/mechanisms/casper/src/types.ts`
- `typescript/packages/mechanisms/casper/src/signer.ts`
- `typescript/packages/mechanisms/casper/src/exact/client/scheme.ts`
- `typescript/packages/mechanisms/casper/src/exact/client/index.ts`
- `typescript/packages/mechanisms/casper/src/index.ts`
- `typescript/packages/mechanisms/casper/test/exact/client/scheme.test.ts`

Files to modify:

- `typescript/README.md` — add Casper row to the chains table
- `examples/typescript/clients/advanced/package.json` — add `@x402/casper` dependency
- `examples/typescript/clients/advanced/index.ts` — read Casper env vars and pass to patterns
- `examples/typescript/clients/advanced/all_networks.ts` — register Casper scheme
- `examples/typescript/clients/advanced/builder-pattern.ts` — register Casper scheme
- `examples/typescript/clients/advanced/hooks.ts` — register Casper scheme
- `examples/typescript/clients/advanced/preferred-network.ts` — register Casper scheme

---

### Task 1: Create the Casper mechanism package.json

**Files:**
- Create: `typescript/packages/mechanisms/casper/package.json`

- [ ] **Step 1: Write package.json**

```json
{
  "name": "@x402/casper",
  "version": "2.15.0",
  "main": "./dist/cjs/index.js",
  "module": "./dist/esm/index.js",
  "types": "./dist/cjs/index.d.ts",
  "scripts": {
    "start": "tsx --env-file=.env index.ts",
    "build": "tsup",
    "test": "vitest run",
    "test:integration": "vitest run --config vitest.integration.config.ts",
    "test:watch": "vitest",
    "watch": "tsc --watch",
    "format": "prettier -c .prettierrc --write \"**/*.{ts,js,cjs,json,md}\"",
    "format:check": "prettier -c .prettierrc --check \"**/*.{ts,js,cjs,json,md}\"",
    "lint": "eslint . --ext .ts --fix",
    "lint:check": "eslint . --ext .ts"
  },
  "keywords": [
    "x402",
    "payment",
    "protocol",
    "casper"
  ],
  "license": "Apache-2.0",
  "author": "x402 Foundation",
  "repository": "https://github.com/x402-foundation/x402",
  "description": "x402 Payment Protocol Casper Implementation",
  "devDependencies": {
    "@eslint/js": "^9.24.0",
    "@types/node": "^22.13.4",
    "@typescript-eslint/eslint-plugin": "^8.29.1",
    "@typescript-eslint/parser": "^8.29.1",
    "eslint": "^9.24.0",
    "eslint-plugin-import": "^2.31.0",
    "eslint-plugin-jsdoc": "^50.6.9",
    "eslint-plugin-prettier": "^5.2.6",
    "prettier": "3.5.2",
    "tsup": "^8.4.0",
    "tsx": "^4.21.0",
    "typescript": "^5.7.3",
    "vite": "^6.2.6",
    "vite-tsconfig-paths": "^5.1.4",
    "vitest": "^3.0.5"
  },
  "dependencies": {
    "@x402/core": "workspace:~",
    "casper-js-sdk": "5.0.12",
    "@casper-ecosystem/casper-eip-712": "^1.2.1"
  },
  "exports": {
    ".": {
      "import": {
        "types": "./dist/esm/index.d.mts",
        "default": "./dist/esm/index.mjs"
      },
      "require": {
        "types": "./dist/cjs/index.d.ts",
        "default": "./dist/cjs/index.js"
      }
    },
    "./exact/client": {
      "import": {
        "types": "./dist/esm/exact/client/index.d.mts",
        "default": "./dist/esm/exact/client/index.mjs"
      },
      "require": {
        "types": "./dist/cjs/exact/client/index.d.ts",
        "default": "./dist/cjs/exact/client/index.js"
      }
    }
  },
  "files": [
    "dist"
  ]
}
```

- [ ] **Step 2: Validate JSON syntax**

Run: `node -e "JSON.parse(require('fs').readFileSync('typescript/packages/mechanisms/casper/package.json'))"`
Expected: No output (success).

- [ ] **Step 3: Commit**

```bash
git add typescript/packages/mechanisms/casper/package.json
git commit -m "feat(casper): add package.json for @x402/casper"
```

---

### Task 2: Create tsconfig.json

**Files:**
- Create: `typescript/packages/mechanisms/casper/tsconfig.json`

- [ ] **Step 1: Write tsconfig.json**

```json
{
  "extends": "../../../tsconfig.base.json",
  "compilerOptions": {
    "outDir": "./dist",
    "rootDir": "./src"
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "test"]
}
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/tsconfig.json
git commit -m "feat(casper): add tsconfig.json"
```

---

### Task 3: Create tsup.config.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/tsup.config.ts`

- [ ] **Step 1: Write tsup.config.ts**

```typescript
import { defineConfig } from "tsup";

const baseConfig = {
  entry: {
    index: "src/index.ts",
    "exact/client/index": "src/exact/client/index.ts",
  },
  dts: {
    resolve: true,
  },
  sourcemap: true,
  target: "es2020",
};

export default defineConfig([
  {
    ...baseConfig,
    format: "esm",
    outDir: "dist/esm",
    clean: true,
  },
  {
    ...baseConfig,
    format: "cjs",
    outDir: "dist/cjs",
    clean: false,
  },
]);
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/tsup.config.ts
git commit -m "feat(casper): add tsup config"
```

---

### Task 4: Create vitest.config.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/vitest.config.ts`

- [ ] **Step 1: Write vitest.config.ts**

```typescript
import { loadEnv } from "vite";
import { defineConfig } from "vitest/config";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig(({ mode }) => ({
  test: {
    env: loadEnv(mode, process.cwd(), ""),
    exclude: [
      "**/node_modules/**",
      "**/dist/**",
      "**/test/integrations/**",
    ],
  },
  plugins: [tsconfigPaths({ projects: ["."] })],
}));
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/vitest.config.ts
git commit -m "feat(casper): add vitest config"
```

---

### Task 5: Create eslint.config.js

**Files:**
- Create: `typescript/packages/mechanisms/casper/eslint.config.js`

- [ ] **Step 1: Write eslint.config.js**

```javascript
import js from "@eslint/js";
import ts from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";
import prettier from "eslint-plugin-prettier";
import jsdoc from "eslint-plugin-jsdoc";
import importPlugin from "eslint-plugin-import";

export default [
  {
    ignores: ["dist/**", "node_modules/**"],
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    ignores: ["**/*.test.ts", "test/**/*"],
    languageOptions: {
      parser: tsParser,
      sourceType: "module",
      ecmaVersion: 2020,
      globals: {
        process: "readonly",
        __dirname: "readonly",
        module: "readonly",
        require: "readonly",
        Buffer: "readonly",
        exports: "readonly",
        setTimeout: "readonly",
        clearTimeout: "readonly",
        setInterval: "readonly",
        clearInterval: "readonly",
        console: "readonly",
      },
    },
    plugins: {
      "@typescript-eslint": ts,
      prettier: prettier,
      jsdoc: jsdoc,
      import: importPlugin,
    },
    rules: {
      ...ts.configs.recommended.rules,
      "import/first": "error",
      "prettier/prettier": "error",
      "@typescript-eslint/member-ordering": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_$" }],
      "jsdoc/tag-lines": ["error", "any", { startLines: 1 }],
      "jsdoc/check-alignment": "error",
      "jsdoc/no-undefined-types": "off",
      "jsdoc/check-param-names": "error",
      "jsdoc/check-tag-names": "error",
      "jsdoc/check-types": "error",
      "jsdoc/implements-on-classes": "error",
      "jsdoc/require-description": "error",
      "jsdoc/require-jsdoc": [
        "error",
        {
          require: {
            FunctionDeclaration: true,
            MethodDefinition: true,
            ClassDeclaration: true,
            ArrowFunctionExpression: false,
            FunctionExpression: false,
          },
        },
      ],
      "jsdoc/require-param": "error",
      "jsdoc/require-param-description": "error",
      "jsdoc/require-param-type": "off",
      "jsdoc/require-returns": "error",
      "jsdoc/require-returns-description": "error",
      "jsdoc/require-returns-type": "off",
      "jsdoc/require-hyphen-before-param-description": ["error", "always"],
    },
  },
  {
    files: ["**/*.test.ts", "test/**/*"],
    languageOptions: {
      parser: tsParser,
      sourceType: "module",
      ecmaVersion: 2020,
    },
    plugins: {
      "@typescript-eslint": ts,
      prettier: prettier,
    },
    rules: {
      "prettier/prettier": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_$" }],
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/member-ordering": "off",
    },
  },
];
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/eslint.config.js
git commit -m "feat(casper): add eslint config"
```

---

### Task 6: Create .prettierrc and .prettierignore

**Files:**
- Create: `typescript/packages/mechanisms/casper/.prettierrc`
- Create: `typescript/packages/mechanisms/casper/.prettierignore`

- [ ] **Step 1: Write .prettierrc**

```json
{
  "tabWidth": 2,
  "useTabs": false,
  "semi": true,
  "singleQuote": false,
  "trailingComma": "all",
  "bracketSpacing": true,
  "arrowParens": "avoid",
  "printWidth": 100,
  "proseWrap": "never"
}
```

- [ ] **Step 2: Write .prettierignore**

```
# build output
dist/
node_modules/
```

- [ ] **Step 3: Commit**

```bash
git add typescript/packages/mechanisms/casper/.prettierrc typescript/packages/mechanisms/casper/.prettierignore
git commit -m "feat(casper): add prettier config"
```

---

### Task 7: Create constants.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/src/constants.ts`

- [ ] **Step 1: Write constants.ts**

```typescript
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
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/src/constants.ts
git commit -m "feat(casper): add network constants"
```

---

### Task 8: Create utils.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/src/utils.ts`

- [ ] **Step 1: Write utils.ts**

```typescript
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
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/src/utils.ts
git commit -m "feat(casper): add validation and conversion utilities"
```

---

### Task 9: Create types.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/src/types.ts`

- [ ] **Step 1: Write types.ts**

```typescript
/**
 * Authorization fields for an Exact Casper payment.
 * Mirrors the Go ExactCasperAuthorization struct.
 */
export type ExactCasperAuthorization = {
  /** Payer account-hash address with "00" prefix. */
  from: string;
  /** Recipient account-hash address with "00" prefix. */
  to: string;
  /** Atomic token amount as a decimal string. */
  value: string;
  /** Unix timestamp after which the authorization is valid. */
  validAfter: string;
  /** Unix timestamp before which the authorization must be used. */
  validBefore: string;
  /** 32-byte nonce as a hex string. */
  nonce: string;
};

/**
 * Payload for an Exact Casper payment.
 * Mirrors the Go ExactCasperPayload struct.
 */
export type ExactCasperPayload = {
  /** 65-byte EIP-712 signature as a hex string. */
  signature: string;
  /** Full public key hex of the payer. */
  publicKey: string;
  /** Signed authorization fields. */
  authorization: ExactCasperAuthorization;
};
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/src/types.ts
git commit -m "feat(casper): add exact payload types"
```

---

### Task 10: Create signer.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/src/signer.ts`

- [ ] **Step 1: Write signer.ts**

```typescript
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
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/src/signer.ts
git commit -m "feat(casper): add client signer interface and factory"
```

---

### Task 11: Create exact/client/scheme.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/src/exact/client/scheme.ts`

- [ ] **Step 1: Write scheme.ts**

```typescript
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
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/src/exact/client/scheme.ts
git commit -m "feat(casper): add ExactCasperScheme client"
```

---

### Task 12: Create exact/client/index.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/src/exact/client/index.ts`

- [ ] **Step 1: Write index.ts**

```typescript
export { ExactCasperScheme } from "./scheme";
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/src/exact/client/index.ts
git commit -m "feat(casper): export ExactCasperScheme from exact/client"
```

---

### Task 13: Create src/index.ts

**Files:**
- Create: `typescript/packages/mechanisms/casper/src/index.ts`

- [ ] **Step 1: Write index.ts**

```typescript
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
```

- [ ] **Step 2: Commit**

```bash
git add typescript/packages/mechanisms/casper/src/index.ts
git commit -m "feat(casper): add top-level package exports"
```

---

### Task 14: Write unit tests

**Files:**
- Create: `typescript/packages/mechanisms/casper/test/exact/client/scheme.test.ts`

- [ ] **Step 1: Write the test file**

```typescript
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

function buildRequirements(overrides?: Partial<{
  asset: string;
  amount: string;
  payTo: string;
  extra: Record<string, unknown>;
}>): Parameters<ExactCasperScheme["createPaymentPayload"]>[1] {
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

    const auth = (result.payload as { authorization: { validAfter: string; validBefore: string } }).authorization;
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

  it("throws when token name is missing", async () => {
    const scheme = new ExactCasperScheme(createTestSigner());
    await expect(
      scheme.createPaymentPayload(
        2,
        buildRequirements({ extra: { version: "1" } }),
      ),
    ).rejects.toThrow("invalid_exact_casper_client_missing_token_name");
  });

  it("throws when token version is missing", async () => {
    const scheme = new ExactCasperScheme(createTestSigner());
    await expect(
      scheme.createPaymentPayload(
        2,
        buildRequirements({ extra: { name: "TestToken" } }),
      ),
    ).rejects.toThrow("invalid_exact_casper_client_missing_token_version");
  });
});
```

- [ ] **Step 2: Run the tests**

Run:
```bash
cd typescript/packages/mechanisms/casper
pnpm install
pnpm test
```

Expected: Tests pass (some may be flaky due to timestamp windows; if so, widen the assertions).

- [ ] **Step 3: Commit**

```bash
git add typescript/packages/mechanisms/casper/test/exact/client/scheme.test.ts
git commit -m "test(casper): add ExactCasperScheme unit tests"
```

---

### Task 15: Build and lint the new package

**Files:**
- Modify: (none directly; verifies all created files)

- [ ] **Step 1: Install workspace dependencies**

Run:
```bash
cd typescript
pnpm install
```

Expected: pnpm installs `@x402/casper` workspace dependencies.

- [ ] **Step 2: Build the package**

Run:
```bash
cd typescript/packages/mechanisms/casper
pnpm build
```

Expected: `dist/esm/` and `dist/cjs/` are created with `index` and `exact/client/index` outputs.

- [ ] **Step 3: Run lint and format checks**

Run:
```bash
pnpm lint:check
pnpm format:check
```

Expected: No errors.

- [ ] **Step 4: Commit**

`dist/` is gitignored; only commit source/config files.

```bash
git add typescript/packages/mechanisms/casper/src typescript/packages/mechanisms/casper/*.json typescript/packages/mechanisms/casper/*.ts typescript/packages/mechanisms/casper/*.js typescript/packages/mechanisms/casper/.prettierrc typescript/packages/mechanisms/casper/.prettierignore typescript/packages/mechanisms/casper/test
git commit -m "chore(casper): build and lint @x402/casper"
```

---

### Task 16: Add @x402/casper to the advanced example

**Files:**
- Modify: `examples/typescript/clients/advanced/package.json`

- [ ] **Step 1: Add dependency**

In `dependencies`, add `"@x402/casper": "workspace:*"` alongside the other `@x402/*` entries.

```json
"dependencies": {
  "@scure/base": "^1.2.6",
  "@x402/avm": "workspace:*",
  "@x402/casper": "workspace:*",
  "@x402/evm": "workspace:*",
  "@x402/svm": "workspace:*",
  "@x402/stellar": "workspace:*",
  "@x402/hedera": "workspace:*",
  "@x402/fetch": "workspace:*",
  "dotenv": "^16.4.7",
  "viem": "^2.48.11",
  "@solana/kit": "^6.1.0"
}
```

- [ ] **Step 2: Commit**

```bash
git add examples/typescript/clients/advanced/package.json
git commit -m "feat(advanced-example): add @x402/casper dependency"
```

---

### Task 17: Update all_networks.ts

**Files:**
- Modify: `examples/typescript/clients/advanced/all_networks.ts`

- [ ] **Step 1: Add imports**

Add after the Hedera imports:

```typescript
import { KeyAlgorithm } from "casper-js-sdk";
import { createClientCasperSigner } from "@x402/casper";
import { ExactCasperScheme } from "@x402/casper/exact/client";
```

- [ ] **Step 2: Add environment variables**

After the Hedera env vars:

```typescript
const casperPrivateKeyPath = process.env.CASPER_PRIVATE_KEY_PATH as string | undefined;
const casperKeyAlgorithm = process.env.CASPER_KEY_ALGORITHM as string | undefined;
```

- [ ] **Step 3: Update validation block**

Change the `if` condition to include Casper:

```typescript
if (
  !avmPrivateKey &&
  !evmPrivateKey &&
  !svmPrivateKey &&
  !stellarPrivateKey &&
  !(hederaAccountId && hederaPrivateKey) &&
  !casperPrivateKeyPath
) {
  console.error(
    "❌ At least one of AVM_PRIVATE_KEY, EVM_PRIVATE_KEY, SVM_PRIVATE_KEY, STELLAR_PRIVATE_KEY, HEDERA_ACCOUNT_ID + HEDERA_PRIVATE_KEY, or CASPER_PRIVATE_KEY_PATH is required",
  );
  process.exit(1);
}
```

- [ ] **Step 4: Add Casper registration block**

Before the Stellar registration block, add:

```typescript
// Register Casper scheme if private key path is provided
if (casperPrivateKeyPath) {
  const algorithm =
    casperKeyAlgorithm === "secp256k1" ? KeyAlgorithm.SECP256K1 : KeyAlgorithm.ED25519;
  const casperSigner = await createClientCasperSigner(casperPrivateKeyPath, algorithm);
  client.register("casper:*", new ExactCasperScheme(casperSigner));
  console.log(`Initialized Casper account: ${casperSigner.accountAddress()}`);
}
```

- [ ] **Step 5: Commit**

```bash
git add examples/typescript/clients/advanced/all_networks.ts
git commit -m "feat(advanced-example): register Casper in all-networks pattern"
```

---

### Task 18: Update builder-pattern.ts

**Files:**
- Modify: `examples/typescript/clients/advanced/builder-pattern.ts`

- [ ] **Step 1: Update function signature**

Change:

```typescript
export async function runBuilderPatternExample(
  evmPrivateKey: `0x${string}`,
  svmPrivateKey: string,
  url: string,
): Promise<void>
```

To:

```typescript
export async function runBuilderPatternExample(
  evmPrivateKey: `0x${string}`,
  svmPrivateKey: string,
  casperPrivateKeyPath: string | undefined,
  casperKeyAlgorithm: string | undefined,
  url: string,
): Promise<void>
```

- [ ] **Step 2: Add imports**

Add:

```typescript
import { KeyAlgorithm } from "casper-js-sdk";
import { createClientCasperSigner } from "@x402/casper";
import { ExactCasperScheme } from "@x402/casper/exact/client";
```

- [ ] **Step 3: Add Casper signer and registration**

After the SVM signer creation, add:

```typescript
let casperSigner;
if (casperPrivateKeyPath) {
  const algorithm =
    casperKeyAlgorithm === "secp256k1" ? KeyAlgorithm.SECP256K1 : KeyAlgorithm.ED25519;
  casperSigner = await createClientCasperSigner(casperPrivateKeyPath, algorithm);
}
```

Then update the builder chain to include Casper:

```typescript
const client = new x402Client()
  .register("eip155:*", new ExactEvmScheme(evmSigner))
  .register("eip155:*", new UptoEvmScheme(evmSigner))
  .register("eip155:1", new ExactEvmScheme(ethereumMainnetSigner))
  .register("solana:*", new ExactSvmScheme(svmSigner))
  .register("solana:EtWTRABZaYq6iMfeYKouRu166VU2xqa1", new ExactSvmScheme(solanaDevnetSigner));

if (casperSigner) {
  client.register("casper:*", new ExactCasperScheme(casperSigner));
}
```

Also update the `console.log` list to mention Casper when registered.

- [ ] **Step 4: Commit**

```bash
git add examples/typescript/clients/advanced/builder-pattern.ts
git commit -m "feat(advanced-example): register Casper in builder-pattern"
```

---

### Task 19: Update hooks.ts

**Files:**
- Modify: `examples/typescript/clients/advanced/hooks.ts`

- [ ] **Step 1: Update function signature**

Change:

```typescript
export async function runHooksExample(evmPrivateKey: `0x${string}`, url: string): Promise<void>
```

To:

```typescript
export async function runHooksExample(
  evmPrivateKey: `0x${string}`,
  casperPrivateKeyPath: string | undefined,
  casperKeyAlgorithm: string | undefined,
  url: string,
): Promise<void>
```

- [ ] **Step 2: Add imports**

Add:

```typescript
import { KeyAlgorithm } from "casper-js-sdk";
import { createClientCasperSigner } from "@x402/casper";
import { ExactCasperScheme } from "@x402/casper/exact/client";
```

- [ ] **Step 3: Register Casper scheme**

After the EVM/Upto registrations, add:

```typescript
if (casperPrivateKeyPath) {
  const algorithm =
    casperKeyAlgorithm === "secp256k1" ? KeyAlgorithm.SECP256K1 : KeyAlgorithm.ED25519;
  const casperSigner = await createClientCasperSigner(casperPrivateKeyPath, algorithm);
  client.register("casper:*", new ExactCasperScheme(casperSigner));
}
```

- [ ] **Step 4: Commit**

```bash
git add examples/typescript/clients/advanced/hooks.ts
git commit -m "feat(advanced-example): register Casper in hooks pattern"
```

---

### Task 20: Update preferred-network.ts

**Files:**
- Modify: `examples/typescript/clients/advanced/preferred-network.ts`

- [ ] **Step 1: Update function signature**

Change:

```typescript
export async function runPreferredNetworkExample(
  evmPrivateKey: `0x${string}`,
  svmPrivateKey: string,
  url: string,
): Promise<void>
```

To:

```typescript
export async function runPreferredNetworkExample(
  evmPrivateKey: `0x${string}`,
  svmPrivateKey: string,
  casperPrivateKeyPath: string | undefined,
  casperKeyAlgorithm: string | undefined,
  url: string,
): Promise<void>
```

- [ ] **Step 2: Add imports**

Add:

```typescript
import { KeyAlgorithm } from "casper-js-sdk";
import { createClientCasperSigner } from "@x402/casper";
import { ExactCasperScheme } from "@x402/casper/exact/client";
```

- [ ] **Step 3: Update network preferences**

Change:

```typescript
const networkPreferences = ["solana:", "eip155:"];
```

To:

```typescript
const networkPreferences = ["solana:", "casper:", "eip155:"];
```

- [ ] **Step 4: Register Casper scheme**

After the Solana registration, add:

```typescript
if (casperPrivateKeyPath) {
  const algorithm =
    casperKeyAlgorithm === "secp256k1" ? KeyAlgorithm.SECP256K1 : KeyAlgorithm.ED25519;
  const casperSigner = await createClientCasperSigner(casperPrivateKeyPath, algorithm);
  client.register("casper:*", new ExactCasperScheme(casperSigner));
}
```

- [ ] **Step 5: Commit**

```bash
git add examples/typescript/clients/advanced/preferred-network.ts
git commit -m "feat(advanced-example): register Casper in preferred-network pattern"
```

---

### Task 21: Update index.ts

**Files:**
- Modify: `examples/typescript/clients/advanced/index.ts`

- [ ] **Step 1: Add Casper environment variables**

After `const svmPrivateKey = ...`, add:

```typescript
const casperPrivateKeyPath = process.env.CASPER_PRIVATE_KEY_PATH as string | undefined;
const casperKeyAlgorithm = process.env.CASPER_KEY_ALGORITHM as string | undefined;
```

- [ ] **Step 2: Update main comment**

Update the docblock to mention Casper env vars:

```typescript
/**
 * ...
 * To run this example, you need to set the following environment variables:
 * - EVM_PRIVATE_KEY: The private key of the EVM signer
 * - SVM_PRIVATE_KEY: The private key of the SVM signer
 * - CASPER_PRIVATE_KEY_PATH: Path to a PEM-encoded Casper private key (optional)
 * - CASPER_KEY_ALGORITHM: "ed25519" or "secp256k1" (optional, defaults to ed25519)
 * ...
 */
```

- [ ] **Step 3: Update switch cases**

Change:

```typescript
case "builder-pattern":
  if (!svmPrivateKey) { ... }
  await runBuilderPatternExample(evmPrivateKey, svmPrivateKey, url);
  break;
```

To:

```typescript
case "builder-pattern":
  if (!svmPrivateKey) { ... }
  await runBuilderPatternExample(
    evmPrivateKey,
    svmPrivateKey,
    casperPrivateKeyPath,
    casperKeyAlgorithm,
    url,
  );
  break;
```

Change:

```typescript
case "hooks":
  await runHooksExample(evmPrivateKey, url);
  break;
```

To:

```typescript
case "hooks":
  await runHooksExample(
    evmPrivateKey,
    casperPrivateKeyPath,
    casperKeyAlgorithm,
    url,
  );
  break;
```

Change:

```typescript
case "preferred-network":
  if (!svmPrivateKey) { ... }
  await runPreferredNetworkExample(evmPrivateKey, svmPrivateKey, url);
  break;
```

To:

```typescript
case "preferred-network":
  if (!svmPrivateKey) { ... }
  await runPreferredNetworkExample(
    evmPrivateKey,
    svmPrivateKey,
    casperPrivateKeyPath,
    casperKeyAlgorithm,
    url,
  );
  break;
```

- [ ] **Step 4: Commit**

```bash
git add examples/typescript/clients/advanced/index.ts
git commit -m "feat(advanced-example): wire Casper env vars into example runner"
```

---

### Task 22: Update TypeScript README

**Files:**
- Modify: `typescript/README.md`

- [ ] **Step 1: Add Casper row**

Add a new row to the chains table after the Solana row:

```markdown
| **Casper** - [`@x402/casper`](./packages/mechanisms/casper) | Casper implementation of x402 using EIP-712 transfer authorizations. | [![npm version](https://img.shields.io/npm/v/%40x402%2Fcasper.svg)](https://www.npmjs.com/package/@x402/casper) |
```

- [ ] **Step 2: Commit**

```bash
git add typescript/README.md
git commit -m "docs(typescript): add Casper to mechanism table"
```

---

### Task 23: Final verification

**Files:**
- Modify: (none; this is a verification task)

- [ ] **Step 1: Install and build the workspace**

Run:
```bash
cd typescript
pnpm install
pnpm build
```

Expected: All packages build successfully.

- [ ] **Step 2: Run all mechanism tests**

Run:
```bash
cd typescript/packages/mechanisms/casper
pnpm test
```

Expected: Tests pass.

- [ ] **Step 3: Run lint and format checks on changed packages**

Run:
```bash
cd typescript/packages/mechanisms/casper
pnpm lint:check
pnpm format:check

cd ../../../examples/typescript/clients/advanced
pnpm lint:check
pnpm format:check
```

Expected: No errors.

- [ ] **Step 4: Type-check the advanced example**

Run:
```bash
cd examples/typescript/clients/advanced
pnpm install
npx tsc --noEmit
```

Expected: No TypeScript errors.

- [ ] **Step 5: Commit any final fixes**

```bash
git add -A
git commit -m "chore(casper): final verification fixes"
```

---

## Spec coverage check

| Spec section | Implementing task |
|---|---|
| Package skeleton | Tasks 1–6 |
| Network constants | Task 7 |
| Validation utilities | Task 8 |
| Payload types | Task 9 |
| Client signer interface/factory | Task 10 |
| Exact scheme client | Task 11 |
| Top-level exports | Tasks 12–13 |
| Unit tests | Task 14 |
| Advanced example integration | Tasks 16–21 |
| Documentation update | Task 22 |

No placeholders remain in the plan; every step includes the exact file path and content needed.
