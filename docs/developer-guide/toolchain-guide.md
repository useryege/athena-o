# Development Toolchain

Athena uses the local toolchain for day-to-day development.

## Local Toolchain

Install the code generation tools:

```bash
make install-codegen-tools-local
```

For UI work, install dependencies directly in the UI directory:

```bash
cd ui
yarn install
```

For local process orchestration, install `goreman` if it is not already present:

```bash
go install github.com/mattn/goreman@latest
```

Start the local stack:

```bash
make run
```

Run generated-code updates when needed:

```bash
make codegen-local
```
