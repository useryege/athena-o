# Intake reply (incomplete context)

Copy and send when required inputs for implementing the TS gRPC service are missing. **Both** the **`.proto` file** and the **TS package directory** are mandatory—without them, do not guess the contract or invent a project root.

```markdown
To implement the TypeScript gRPC service, please provide:

1. **Proto file (required)**: Repository-relative path to the `*.proto`, or attach/open the file in the IDE so the full `service` and messages are visible. I cannot implement against a vague API name alone.
2. **TS package directory (required)**: Repository-relative path to the TypeScript project root that owns this service (folder with `package.json` / `tsconfig`, e.g. `third-party/foo/`). I need this as explicit context for codegen paths and layout.
3. **Service / RPCs**: The `service` name(s) to implement and either a finite RPC list or explicit `all` for that service (after the proto is in context).
4. **SDK path** (if calling a TS SDK): Full path under `node_modules/...` (e.g. `third-party/foo/node_modules/@scope/pkg`).
5. **Codegen**: Whether `src/gen` (or equivalent) and `proto:gen` already exist; if not, confirm desired `ts-proto` options.
```

Do not implement handlers or change codegen until the above is answered (or explicitly waived with a written default the user accepts).
