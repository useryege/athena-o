# Setting Up the Development Environment

## Required Tools

Install:

- Git
- Go, using the version required by `go.mod`
- Docker
- GNU Make
- Node.js selected by `ui/.nvmrc` and Yarn for UI work; see the
  [WSL installation guide](install-wsl.md) to install NVM and select the project version

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

Install codegen tools:

```bash
make install-codegen-tools-local
```

Install UI dependencies:

```bash
cd ui
nvm install
nvm use
yarn install
cd ..
```

The UI version is defined by `ui/.nvmrc` (currently Node.js `24.14.1`), and
`ui/package.json` accepts `>=24.14.1 <25`. If NVM is already installed, the
commands above only install and select the project version. Do not change the
global NVM default.

## AI 开发检查工具

完成上述 Node 激活后，从仓库根目录执行：

```bash
make install-ai-dev-tools
make ai-dev-tools-check
```

该入口安装 ShellCheck、grpcurl、PostgreSQL 16 客户端、govulncheck 和 UI 的
axe 检查依赖。系统客户端安装可能需要 sudo，不安装数据库服务端。固定版本、
检查命令和使用场景见 [开发工具链](toolchain-guide.md#ai-开发检查工具)。

## Local Services

Athena local development uses local processes plus Docker-backed dependencies. Start the local stack with:

```bash
cd ui
nvm use
cd ..
make run
```

Run these commands in the same shell from the repository root so `make run` inherits
the Node.js version selected by `ui/.nvmrc`.

For production-like local validation, use Docker Compose:

```bash
make prod-start-local
```
