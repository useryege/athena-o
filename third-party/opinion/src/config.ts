import {
  CHAIN_ID_BNB_MAINNET,
  DEFAULT_API_HOST,
  type SupportedChainId,
} from '@opinion-labs/opinion-clob-sdk';

export interface OpinionSdkConfig {
  host: string;
  apiKey: string;
  chainId: SupportedChainId;
  rpcUrl: string;
  privateKey: `0x${string}`;
  multiSigAddress: `0x${string}`;
  /** HTTP(S) proxy for SDK; empty string means no proxy. */
  proxyUrl: string;
}

export interface GrpcServerConfig {
  host: string;
  port: number;
}

export interface OpinionRuntimeConfig {
  grpc: GrpcServerConfig;
  sdk: OpinionSdkConfig;
}

const DEFAULT_GRPC_HOST = '0.0.0.0';
const DEFAULT_GRPC_PORT = '50051';
const DEFAULT_RPC_URL = 'https://bsc-dataseed.binance.org/';
const DEFAULT_PROXY_URL = 'http://172.30.160.1:10809';

function getRequiredEnv(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }

  return value;
}

function getOptionalEnv(name: string, fallback: string): string {
  return process.env[name]?.trim() || fallback;
}

function parsePort(value: string): number {
  const port = Number.parseInt(value, 10);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`Invalid gRPC port: ${value}`);
  }

  return port;
}

function parseChainId(value: string): SupportedChainId {
  const chainId = Number.parseInt(value, 10);
  if (chainId !== CHAIN_ID_BNB_MAINNET) {
    throw new Error(`Unsupported chain id: ${value}. Only ${CHAIN_ID_BNB_MAINNET} is supported`);
  }

  return CHAIN_ID_BNB_MAINNET;
}

function resolveApiHost(): string {
  return process.env.API_HOST?.trim() || process.env.OPINION_HOST?.trim() || DEFAULT_API_HOST;
}

export function loadGrpcServerConfig(): GrpcServerConfig {
  return {
    host: getOptionalEnv('GRPC_HOST', DEFAULT_GRPC_HOST),
    port: parsePort(getOptionalEnv('GRPC_PORT', DEFAULT_GRPC_PORT)),
  };
}

export function loadOpinionSdkConfig(): OpinionSdkConfig {
  return {
    host: resolveApiHost(),
    apiKey: getRequiredEnv('API_KEY'),
    chainId: parseChainId(getOptionalEnv('CHAIN_ID', String(CHAIN_ID_BNB_MAINNET))),
    rpcUrl: getOptionalEnv('RPC_URL', DEFAULT_RPC_URL),
    privateKey: getRequiredEnv('PRIVATE_KEY') as `0x${string}`,
    multiSigAddress: getRequiredEnv('MULTI_SIG_ADDRESS') as `0x${string}`,
    proxyUrl: getOptionalEnv('PROXY_URL', DEFAULT_PROXY_URL),
  };
}

export function loadRuntimeConfig(): OpinionRuntimeConfig {
  return {
    grpc: loadGrpcServerConfig(),
    sdk: loadOpinionSdkConfig(),
  };
}
