import 'dotenv/config';

function required(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

function optional(name: string, fallback: string): string {
  return process.env[name] ?? fallback;
}

function optionalInt(name: string, fallback: number): number {
  const raw = process.env[name];
  if (!raw) return fallback;
  const parsed = parseInt(raw, 10);
  if (isNaN(parsed)) {
    throw new Error(`Environment variable ${name} must be an integer, got: ${raw}`);
  }
  return parsed;
}

export const env = {
  // Server
  PORT: optionalInt('PORT', 18080),
  NODE_ENV: optional('NODE_ENV', 'production'),

  // Opinion SDK — required at startup
  API_KEY: required('API_KEY'),
  RPC_URL: optional('RPC_URL', 'https://bsc-dataseed.binance.org/'),
  PRIVATE_KEY: required('PRIVATE_KEY'),
  MULTI_SIG_ADDRESS: required('MULTI_SIG_ADDRESS'),
  OPINION_HOST: optional('OPINION_HOST', ''),

  // Opinion SDK — optional tuning
  MARKET_CACHE_TTL: optionalInt('MARKET_CACHE_TTL', 300),
  QUOTE_TOKENS_CACHE_TTL: optionalInt('QUOTE_TOKENS_CACHE_TTL', 3600),
  ENABLE_TRADING_CHECK_INTERVAL: optionalInt('ENABLE_TRADING_CHECK_INTERVAL', 3600),
} as const;

export type Env = typeof env;
