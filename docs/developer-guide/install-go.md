# Install Go on Linux（WSL）

This guide installs Go `1.25.5` from the official Linux `amd64` tarball on WSL.

## 1. Download the Archive

```bash
wget https://dl.google.com/go/go1.25.5.linux-amd64.tar.gz
```

## 2. Remove the Previous Installation

```bash
sudo rm -rf /usr/local/go
```

## 3. Extract Go

```bash
sudo tar -C /usr/local -xzf go1.25.5.linux-amd64.tar.gz
```

## 4. Add Go to `PATH`

Add `/usr/local/go/bin` to the `PATH` environment variable.

Add it to `$HOME/.profile` with a single command:

```bash
echo 'export PATH=$PATH:/usr/local/go/bin' >> "$HOME/.profile"
```

Note: changes made to a profile file may not apply until the next time you log in. To apply the changes immediately, reload the profile:

```bash
source "$HOME/.profile"
```

## 5. Verify the Installation

Verify that Go is installed correctly:

```bash
go version
```

## 6. Clean Up

```bash
rm go1.25.5.linux-amd64.tar.gz
```