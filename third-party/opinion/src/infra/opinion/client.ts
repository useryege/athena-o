import { Client } from '@opinion-labs/opinion-clob-sdk';
import { buildOpinionConfig } from '../../config/opinion.js';

type ClientStatus =
  | { initialized: false; error: null }
  | { initialized: true; error: null }
  | { initialized: false; error: string };

let instance: Client | null = null;
let status: ClientStatus = { initialized: false, error: null };

export function initOpinionClient(): void {
  const config = buildOpinionConfig();
  instance = new Client(config);
  status = { initialized: true, error: null };
}

export function getClient(): Client {
  if (!instance) {
    throw new Error('Opinion client is not initialized');
  }
  return instance;
}

export function getClientStatus(): ClientStatus {
  return status;
}

export function setClientError(err: unknown): void {
  const message = err instanceof Error ? err.message : String(err);
  status = { initialized: false, error: message };
}
