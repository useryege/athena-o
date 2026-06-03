# Development Toolchain

Athena supports a local toolchain and a containerized toolchain.

## Local vs Containerized

- Local commands end with `-local` and run directly on your machine.
- Containerized commands run through the `athena-test-tools` Docker image.

Examples:

```bash
make test-local
make test
```

The local toolchain is faster for everyday work. The containerized toolchain is useful when you want a repeatable environment.

## Containerized Toolchain

Build the test tools image with:

```bash
make test-tools-image
```

The image is defined by `test/container/Dockerfile` and includes Go, Node/Yarn, Redis, protobuf/codegen tools, lint tools, and unit-test helpers.

The container mounts:

- the repository source tree
- your Go module cache
- your Go build cache

If Docker requires elevated privileges on your machine, set:

```bash
SUDO=sudo make test
```

## Local Toolchain

Install the local tools:

```bash
make install-tools-local
```

For UI work, install Yarn dependencies:

```bash
make dep-ui-local
```

For local process orchestration, install `goreman` if it is not already present:

```bash
go install github.com/mattn/goreman@latest
```

Start the local stack:

```bash
make start-local
```
