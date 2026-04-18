import { resolve } from 'node:path';

import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';

import type { OpinionServiceClientConstructor } from './types.js';

export const opinionProtoPath = resolve(
  process.cwd(),
  'opinion/v1/opinion.proto',
);

interface LoadedOpinionPackage {
  opinion: {
    v1: {
      OpinionService: OpinionServiceClientConstructor;
    };
  };
}

export function loadOpinionProto() {
  const packageDefinition = protoLoader.loadSync(opinionProtoPath, {
    keepCase: true,
    longs: String,
    defaults: true,
    oneofs: true,
  });

  const grpcObject = grpc.loadPackageDefinition(packageDefinition) as unknown as LoadedOpinionPackage;
  return grpcObject.opinion.v1;
}
