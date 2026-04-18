import 'dotenv/config';

import { Server, ServerCredentials } from '@grpc/grpc-js';

import { OpinionServiceService } from '../../src/gen/opinion/opinion.js';
import { loadRuntimeConfig } from '../../src/config.js';
import { createOpinionService } from '../../src/service.js';

async function bindServer(server: Server, address: string): Promise<number> {
  return await new Promise((resolve, reject) => {
    server.bindAsync(address, ServerCredentials.createInsecure(), (error, port) => {
      if (error) {
        reject(error);
        return;
      }

      resolve(port);
    });
  });
}

async function main() {
  const config = loadRuntimeConfig();
  const server = new Server();

  server.addService(OpinionServiceService, createOpinionService());

  const address = `${config.grpc.host}:${config.grpc.port}`;
  const boundPort = await bindServer(server, address);

  console.log(`Opinion gRPC server listening on ${config.grpc.host}:${boundPort}`);

  let shuttingDown = false;
  const shutdown = () => {
    if (shuttingDown) {
      return;
    }

    shuttingDown = true;
    server.tryShutdown((error) => {
      if (error) {
        console.error('Failed to shutdown Opinion gRPC server gracefully:', error);
        server.forceShutdown();
      }

      process.exit(error ? 1 : 0);
    });
  };

  process.once('SIGINT', shutdown);
  process.once('SIGTERM', shutdown);
}

void main().catch((error) => {
  console.error('Failed to start Opinion gRPC server:', error);
  process.exit(1);
});
