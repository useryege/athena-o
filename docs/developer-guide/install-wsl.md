## WSL Development Setup

This guide shows the shortest recommended setup for local development with WSL2.

## 1. Install WSL2 and Ubuntu

Check your current WSL status in PowerShell:

```powershell
wsl -l -v
```

If you only see `docker-desktop`, you do not have a Linux distro ready for development yet.

Install Ubuntu 24.04:

```powershell
wsl --install -d Ubuntu-24.04
```

If you are not sure about the distro name, list the available options first:

```powershell
wsl --list --online
```

After the installation finishes, verify it again:

```powershell
wsl -l -v
```

Expected output looks like this:

```text
  NAME              STATE           VERSION
* Ubuntu-24.04      Running         2
  docker-desktop    Running         2
```

Set the default connect:

```powershell
wsl --set-default Ubuntu-24.04
```

---

## 2. Check the Linux Version

From Windows:

```powershell
wsl -l -v
```

Inside WSL:

```bash
grep PRETTY_NAME /etc/os-release
```

Expected output:

```text
PRETTY_NAME="Ubuntu 24.04.2 LTS"
```

---

## 3. Enter WSL

Start Ubuntu from PowerShell:

```powershell
wsl -d Ubuntu-24.04
```

The first time you enter the system, WSL may ask you to create a Linux username and password. Complete that step manually.

This document uses `yege` as the example username.

Your prompt should look similar to this:

```bash
yege@DESKTOP-RO80FPM:~$
```

That means you are now inside the Linux environment.

---

## 4. Store the Project in `~/work`

Keep the project in the WSL Linux filesystem and use this path format consistently:

```bash
mkdir ~/work/athena
cd ~/work/athena
```

Do not use a Windows-mounted path such as:

```bash
/mnt/d/LEARN/Athena/athena
```

Why:

- Better file I/O performance
- More reliable file watching
- Linux permissions behave as expected
- Better experience for Node, Go, and Docker tools

---

## 5. Move the Repository into WSL

The recommended way is to clone the repository again inside WSL:

```bash
mkdir -p ~/work
cd ~/work
git clone <your-repository-url> athena
cd athena
```

---

## 6. Install Required Tools

Install the basic tools needed for local development inside WSL.

For example, install `make` with:

```bash
sudo apt update
sudo apt install -y make unzip
sudo apt install -y net-tools
sudo apt install -y jq
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
```

---

## 7. Install the Athena toolchain in WSL

### 7.1 Install Node.js

```bash
# Install nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/master/install.sh | bash

# Reload shell
source ~/.bashrc

# From the Athena repository, install and select the UI version from ui/.nvmrc
cd ~/work/athena/ui
nvm install
nvm use

# Check that the project version is selected
node -v
npm -v
```

`ui/.nvmrc` currently selects Node.js `24.14.1`. The UI package accepts Node.js
`>=24.14.1 <25`. Do not change your global NVM default for Athena; run `nvm use`
from `ui` in each shell that will run UI commands or `make run`.

If NVM is already installed, skip its installation command, enter `ui`, and run
`nvm install` followed by `nvm use`.

### 7.2 Install the yarn

```bash
# Install yarn globally
npm install --global yarn

# Check if installation is successful
yarn -v
```

### 7.3 Install Python

Install Python with `apt` inside Ubuntu:

```bash
sudo apt update
sudo apt install -y python3 python3-pip python3-venv
```

Check the installed versions:

```bash
python3 --version
pip3 --version
```

If you want `python` to call Python 3, install:

```bash
sudo apt install -y python-is-python3
```

Then check:

```bash
python --version
```

It is recommended to test with a virtual environment:

```bash
mkdir -p ~/python-test
cd ~/python-test
python3 -m venv .venv
source .venv/bin/activate
python --version
```

Exit the virtual environment:

```bash
deactivate
```

Install packages with:

```bash
pip install requests
```

If installing packages outside a virtual environment is restricted, prefer using a virtual environment instead of installing many packages into the system Python.

### 7.4 Local process runtime

Athena uses its Go instance runtime with Linux/WSL, Docker and Bash5.1+. Goreman is no longer a prerequisite. Verify Bash with `bash --version`; run `make run-service SERVICE=trader-sync` for only Trader Sync and PostgreSQL, or `make run` for the explicit full stack. Read [Running Locally](running-locally.md) for configuration, status and instance-owned shutdown.

### 7.5 Install the abigen

```bash
# Install abigen
go install github.com/ethereum/go-ethereum/cmd/abigen@v1.17.2

# Check if installation is successful
abigen -version
```

### 7.6 Install the buf

```bash
# Download and install buf into /usr/local/bin
sudo curl -sSL \
  "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" \
  -o /usr/local/bin/buf

sudo chmod +x /usr/local/bin/buf

# Check if installation is successful
buf --version
```

### 7.7 Install the solc 

```bash
npm install -g solc

# Check if installation is successful
solc --version
```

## 8. Open the Project in Cursor

Your must install cursor first. From the WSL terminal, go to the project directory:

```bash
cd ~/work/athena
```

Then run:

```bash
cursor .
```

If the command is available, Cursor will open the current folder in WSL mode.

---

## Quick Start

If you already installed Ubuntu, the shortest path is:

```bash
wsl -d Ubuntu-24.04
sudo apt update
sudo apt install -y make
mkdir -p ~/work
cd ~/work
git clone <your-repository-url> athena
cd athena
cursor .
```


## AI 开发检查工具

完成 Node、Go、Docker 和项目依赖配置后，在同一 WSL 终端运行：

```bash
cd ~/work/athena/ui
nvm use
cd ..
make install-ai-dev-tools
make ai-dev-tools-check
```

安装器将三个 CLI 工具放到项目 `dist/`，安装 PostgreSQL 16 系统客户端和锁定的
UI 无障碍依赖。系统包安装可能要求 sudo 密码；不安装 PostgreSQL 服务端，也不
改变 NVM 全局默认版本。日常检查入口及连接示例见 [开发工具链](toolchain-guide.md#ai-开发检查工具)。
