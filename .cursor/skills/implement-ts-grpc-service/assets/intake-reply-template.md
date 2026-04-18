# Intake reply (incomplete context)

Copy and send when required inputs for implementing the TS gRPC service are missing or ambiguous. The **`.proto` file** is mandatory. The **TS package directory** is required only when it **cannot** be inferred (walk up from the proto’s directory to the nearest `package.json`) or when multiple roots are plausible—do not guess the contract; do not invent a project root when inference fails.

```markdown
To implement the TypeScript gRPC service, please provide:

1. **Proto file (required)**: Repository-relative path to the `*.proto`, or attach/open the file in the IDE so the full `service` and messages are visible. I cannot implement against a vague API name alone.
2. **TS package directory (optional if inferable)**: Omit if the project root is the nearest ancestor of the proto path that contains `package.json`. Otherwise provide the repository-relative path (e.g. `third-party/foo/`)—needed when inference fails or the layout is ambiguous.
3. **Service / RPCs**: The `service` name(s) to implement and either a finite RPC list or explicit `all` for that service (after the proto is in context).
4. **SDK path** (if calling a TS SDK): Full path under `node_modules/...` (e.g. `third-party/foo/node_modules/@scope/pkg`).
5. **Codegen**: Whether `src/gen` (or equivalent) and `proto:gen` already exist; if not, confirm desired `ts-proto` options.
```

Do not implement handlers or change codegen until the above is answered (or explicitly waived with a written default the user accepts).
