# Setting Up the Development Environment

## Required Tools

Install:

- Git
- Go, using the version required by `go.mod`
- Docker
- GNU Make
- Node.js and Yarn for UI work
- MkDocs dependencies when editing documentation

Verify the core tools:

```bash
go version
docker version
make --version
```

## Fork and Clone

Fork the Athena repository, then clone your fork:

```bash
git clone https://github.com/YOUR-USERNAME/athena.git
cd athena
git remote add upstream https://github.com/useryege/athena.git
```

## Install Development Tools

Install Go/codegen/lint tools:

```bash
make install-go-tools-local
make install-codegen-tools-local
```

Install UI dependencies:

```bash
make dep-ui-local
```

You can also install the broader local toolchain:

```bash
make install-tools-local
```

## Local Services

Athena local development uses local processes plus Docker-backed dependencies. Start the local stack with:

```bash
make start-local
```

For production-like local validation, use Docker Compose:

```bash
make prod-start-local
```
