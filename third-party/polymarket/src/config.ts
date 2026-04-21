import { type ApiKeyCreds, Chain } from '@polymarket/clob-client-v2';

export interface GrpcServerConfig {
  host: string;
  port: number;
}

export interface PolymarketSdkConfig {
  host: string;
  chain: Chain;
  privateKey?: `0x${string}`;
  creds?: ApiKeyCreds;
  proxyUrl?: string;
}

export interface PolymarketRuntimeConfig {
  grpc: GrpcServerConfig;
  sdk: PolymarketSdkConfig;
}

const DEFAULT_GRPC_HOST = '0.0.0.0';
const DEFAULT_GRPC_PORT = '50051';
const DEFAULT_CLOB_API_HOST = 'https://clob.polymarket.com';
const DEFAULT_CHAIN_ID = String(Chain.POLYGON);

function getOptionalEnv(name: string, fallback: string): string {
  return process.env[name]?.trim() || fallback;
}

function getOptionalEnvNoFallback(name: string): string | undefined {
  const value = process.env[name]?.trim();
  return value ? value : undefined;
}

function parsePort(value: string): number {
  const port = Number.parseInt(value, 10);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`Invalid gRPC port: ${value}`);
  }

  return port;
}

function parseChain(value: string): Chain {
  const chain = Number.parseInt(value, 10);
  if (chain === Chain.POLYGON || chain === Chain.AMOY) {
    return chain;
  }

  throw new Error(
    `Unsupported chain id: ${value}. Supported chain ids are ${Chain.POLYGON} (POLYGON) and ${Chain.AMOY} (AMOY)`,
  );
}

function resolveClobApiHost(): string {
  return (
    process.env.CLOB_API_URL?.trim() ||
    process.env.POLYMARKET_HOST?.trim() ||
    DEFAULT_CLOB_API_HOST
  );
}

function resolvePrivateKey(): `0x${string}` | undefined {
  const value = getOptionalEnvNoFallback('PK') ?? getOptionalEnvNoFallback('PRIVATE_KEY');
  if (!value) {
    return undefined;
  }

  if (!value.startsWith('0x')) {
    throw new Error('PK/PRIVATE_KEY must be a 0x-prefixed hex private key');
  }

  return value as `0x${string}`;
}

function resolveApiCreds(): ApiKeyCreds | undefined {
  const key = getOptionalEnvNoFallback('CLOB_API_KEY');
  const secret = getOptionalEnvNoFallback('CLOB_SECRET') ?? getOptionalEnvNoFallback('CLOB_API_SECRET');
  const passphrase =
    getOptionalEnvNoFallback('CLOB_PASS_PHRASE') ??
    getOptionalEnvNoFallback('CLOB_API_PASSPHRASE');

  const values = [key, secret, passphrase];
  const anyProvided = values.some((value) => value !== undefined);
  const allProvided = values.every((value) => value !== undefined);

  if (anyProvided && !allProvided) {
    throw new Error(
      'Incomplete CLOB API credentials: CLOB_API_KEY + CLOB_SECRET (or CLOB_API_SECRET) + CLOB_PASS_PHRASE (or CLOB_API_PASSPHRASE) are required together',
    );
  }

  if (!allProvided) {
    return undefined;
  }

  return {
    key: key!,
    secret: secret!,
    passphrase: passphrase!,
  };
}

export function loadGrpcServerConfig(): GrpcServerConfig {
  return {
    host: getOptionalEnv('GRPC_HOST', DEFAULT_GRPC_HOST),
    port: parsePort(getOptionalEnv('GRPC_PORT', DEFAULT_GRPC_PORT)),
  };
}

export function loadPolymarketSdkConfig(): PolymarketSdkConfig {
  const privateKey = resolvePrivateKey();
  const creds = resolveApiCreds();
  const proxyUrl = getOptionalEnvNoFallback('PROXY_URL');

  if ((privateKey && !creds) || (!privateKey && creds)) {
    throw new Error('Authenticated SDK mode requires both private key (PK/PRIVATE_KEY) and CLOB API creds');
  }

  return {
    host: resolveClobApiHost(),
    chain: parseChain(getOptionalEnv('CHAIN_ID', DEFAULT_CHAIN_ID)),
    ...(privateKey ? { privateKey } : {}),
    ...(creds ? { creds } : {}),
    ...(proxyUrl ? { proxyUrl } : {}),
  };
}

export function loadRuntimeConfig(): PolymarketRuntimeConfig {
  return {
    grpc: loadGrpcServerConfig(),
    sdk: loadPolymarketSdkConfig(),
  };
}
