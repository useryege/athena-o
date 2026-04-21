export interface GrpcServerConfig {
  host: string;
  port: number;
}

/** Runtime config placeholder; extend with CLOB host, chain, signer, etc. */
export interface PolymarketRuntimeConfig {
  grpc: GrpcServerConfig;
}

const DEFAULT_GRPC_HOST = '0.0.0.0';
const DEFAULT_GRPC_PORT = '50051';

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

export function loadGrpcServerConfig(): GrpcServerConfig {
  return {
    host: getOptionalEnv('GRPC_HOST', DEFAULT_GRPC_HOST),
    port: parsePort(getOptionalEnv('GRPC_PORT', DEFAULT_GRPC_PORT)),
  };
}

export function loadRuntimeConfig(): PolymarketRuntimeConfig {
  return {
    grpc: loadGrpcServerConfig(),
  };
}
