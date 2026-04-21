#!/usr/bin/env node

"use strict";

const fs = require("fs");
const path = require("path");

function printUsage() {
  console.error(
    "Usage: node scripts/verify-proto-rules.js <proto-file> [more-proto-files...]"
  );
}

function stripComments(input) {
  const withoutBlock = input.replace(/\/\*[\s\S]*?\*\//g, "");
  return withoutBlock
    .split("\n")
    .map((line) => {
      const idx = line.indexOf("//");
      return idx >= 0 ? line.slice(0, idx) : line;
    })
    .join("\n");
}

function isLowerSnakeCase(name) {
  return /^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$/.test(name);
}

function collectFieldViolations(content) {
  const violations = [];
  const lines = content.split("\n");
  const fieldRegex =
    /^\s*(?:repeated\s+)?(?:map\s*<[^>]+>\s+)?[A-Za-z_][\w.<>]*\s+([A-Za-z_]\w*)\s*=\s*\d+\b/;

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i];
    const match = line.match(fieldRegex);
    if (!match) {
      continue;
    }
    const fieldName = match[1];
    if (!isLowerSnakeCase(fieldName)) {
      violations.push({
        line: i + 1,
        fieldName,
        source: line.trim(),
      });
    }
  }

  return violations;
}

function collectRpcDefinitions(content) {
  const defs = [];
  const lines = content.split("\n");
  const rpcRegex =
    /^\s*rpc\s+([A-Za-z_]\w*)\s*\(\s*([A-Za-z_][\w.]*)\s*\)\s*returns\s*\(\s*([A-Za-z_][\w.]*)\s*\)\s*;/;

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i];
    const match = line.match(rpcRegex);
    if (!match) {
      continue;
    }
    defs.push({
      line: i + 1,
      rpcName: match[1],
      requestType: match[2].split(".").pop(),
      responseType: match[3].split(".").pop(),
      source: line.trim(),
    });
  }

  return defs;
}

function collectRpcFormatViolations(rpcDefs) {
  const violations = [];

  for (const def of rpcDefs) {
    const expectedRequest = `${def.rpcName}Request`;
    const expectedResponse = `${def.rpcName}Response`;
    if (
      def.requestType === expectedRequest &&
      def.responseType === expectedResponse
    ) {
      continue;
    }

    violations.push({
      line: def.line,
      rpcName: def.rpcName,
      requestType: def.requestType,
      responseType: def.responseType,
      expectedRequest,
      expectedResponse,
      source: def.source,
    });
  }

  return violations;
}

function collectRpcTypeReuseViolations(rpcDefs) {
  const requestUsage = new Map();
  const responseUsage = new Map();

  for (const def of rpcDefs) {
    if (!requestUsage.has(def.requestType)) {
      requestUsage.set(def.requestType, []);
    }
    requestUsage.get(def.requestType).push(def);

    if (!responseUsage.has(def.responseType)) {
      responseUsage.set(def.responseType, []);
    }
    responseUsage.get(def.responseType).push(def);
  }

  const violations = [];

  for (const [requestType, defs] of requestUsage.entries()) {
    if (defs.length <= 1) {
      continue;
    }
    violations.push({
      kind: "request",
      typeName: requestType,
      rpcRefs: defs.map((d) => ({
        line: d.line,
        rpcName: d.rpcName,
      })),
    });
  }

  for (const [responseType, defs] of responseUsage.entries()) {
    if (defs.length <= 1) {
      continue;
    }
    violations.push({
      kind: "response",
      typeName: responseType,
      rpcRefs: defs.map((d) => ({
        line: d.line,
        rpcName: d.rpcName,
      })),
    });
  }

  return violations;
}

function verifyProtoFile(protoPath) {
  const absolute = path.resolve(protoPath);
  if (!fs.existsSync(absolute)) {
    return {
      file: protoPath,
      readError: `File not found: ${protoPath}`,
    };
  }

  const raw = fs.readFileSync(absolute, "utf8");
  const content = stripComments(raw);
  const rpcDefs = collectRpcDefinitions(content);

  return {
    file: protoPath,
    fieldViolations: collectFieldViolations(content),
    rpcFormatViolations: collectRpcFormatViolations(rpcDefs),
    rpcReuseViolations: collectRpcTypeReuseViolations(rpcDefs),
  };
}

function main() {
  const targets = process.argv.slice(2);
  if (targets.length === 0) {
    printUsage();
    process.exit(2);
  }

  let hasFailures = false;

  for (const target of targets) {
    const result = verifyProtoFile(target);
    console.log(`\n# ${result.file}`);

    if (result.readError) {
      hasFailures = true;
      console.log(`ERROR: ${result.readError}`);
      continue;
    }

    const { fieldViolations, rpcFormatViolations, rpcReuseViolations } = result;

    if (
      fieldViolations.length === 0 &&
      rpcFormatViolations.length === 0 &&
      rpcReuseViolations.length === 0
    ) {
      console.log("PASS: no violations found");
      continue;
    }

    hasFailures = true;

    if (fieldViolations.length > 0) {
      console.log("\n[FIELD_LOWER_SNAKE_CASE] violations:");
      for (const v of fieldViolations) {
        console.log(`  - line ${v.line}: "${v.fieldName}" -> ${v.source}`);
      }
    }

    if (rpcFormatViolations.length > 0) {
      console.log("\n[RPC_REQUEST_RESPONSE_FORMAT] violations:");
      for (const v of rpcFormatViolations) {
        console.log(
          `  - line ${v.line}: rpc ${v.rpcName}(${v.requestType}) returns (${v.responseType}); expected rpc ${v.rpcName}(${v.expectedRequest}) returns (${v.expectedResponse});`
        );
      }
    }

    if (rpcReuseViolations.length > 0) {
      console.log("\n[RPC_REQUEST_RESPONSE_UNIQUE] violations:");
      for (const v of rpcReuseViolations) {
        const refs = v.rpcRefs
          .map((ref) => `line ${ref.line} (${ref.rpcName})`)
          .join(", ");
        console.log(
          `  - ${v.kind} type "${v.typeName}" is reused by multiple RPCs: ${refs}`
        );
      }
    }
  }

  process.exit(hasFailures ? 1 : 0);
}

main();
