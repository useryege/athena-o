import type { FastifyInstance, FastifyError } from 'fastify';
import { OpinionError, OpinionErrorCode } from '../opinion/errors.js';
import { err } from './response.js';

export function registerErrorHandler(app: FastifyInstance): void {
  app.setErrorHandler((error: FastifyError | Error, _req, reply) => {
    if (error instanceof OpinionError) {
      const status =
        error.code === OpinionErrorCode.INVALID_PARAMS
          ? 400
          : error.code === OpinionErrorCode.NOT_INITIALIZED
            ? 503
            : 502;
      return reply.status(status).send(err(error.code, error.message));
    }

    const fastifyError = error as FastifyError;
    if (fastifyError.validation) {
      return reply.status(400).send(err('VALIDATION_ERROR', error.message));
    }

    const statusCode = fastifyError.statusCode ?? 500;
    return reply.status(statusCode).send(err('INTERNAL_ERROR', error.message));
  });
}
