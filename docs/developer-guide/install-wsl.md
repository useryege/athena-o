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
~/work/argo-cd
```

Do not use a Windows-mounted path such as:

```bash
/mnt/d/LEARN/Athena/argo-cd
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
git clone <your-repository-url> argo-cd
cd argo-cd
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
```

---

## 7. Open the Project in Cursor

From the WSL terminal, go to the project directory:

```bash
cd ~/work/argo-cd
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
git clone <your-repository-url> argo-cd
cd argo-cd
cursor .
```

