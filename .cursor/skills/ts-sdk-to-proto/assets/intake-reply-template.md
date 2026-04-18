# Intake reply (incomplete context)

Copy and send when the user has not provided required context:

```markdown
To proceed with ts-sdk → proto, please provide:

1. **SDK path**: The full repository-relative path under `node_modules/...` (e.g. `node_modules/@scope/package-name`).
2. **Scope**: A finite list of methods/functions to model, with enough detail to locate each symbol (e.g. `Client#methodName`, `exportedFn` in `src/api.ts`).
3. **Output proto**: The repository-relative path to the target `*.proto` file where results should be written (e.g. `third-party/foo/bar/service.proto`).
```

Do not start designing `.proto` files until all of the above are answered.
