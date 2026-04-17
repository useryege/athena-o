import { initOpinionClient, setClientError } from '../infra/opinion/client.js';
import { logger } from '../infra/logger/index.js';

/**
 * Bootstrap all singleton dependencies.
 * Called once before the HTTP server starts accepting connections.
 */
export function bootstrap(): void {
  try {
    initOpinionClient();
    logger.info('Opinion SDK client initialized');
  } catch (err) {
    setClientError(err);
    logger.error('Opinion SDK client initialization failed', err);
    // Do not throw — the server should start and expose /readyz for inspection.
  }
}
