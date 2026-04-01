# WSL Git and GitHub SSH Guide

This guide covers one task:

Generate an SSH key and configure GitHub to use SSH access


### Optional: configure Git user information

```bash
git config --global user.name "Your Name"
git config --global user.email "your-email@example.com"
```

### Optional: Recommended line ending settings for WSL

```bash
git config --global core.autocrlf input
git config --global core.eol lf
```


## Generate an SSH Key for GitHub

### Create the `.ssh` directory

```bash
mkdir -p ~/.ssh
chmod 700 ~/.ssh
```

### Generate an Ed25519 SSH key

Replace the email below with your GitHub email address.

```bash
ssh-keygen -t ed25519 -C "your-email@example.com"
```

When you see this prompt:

```text
Enter file in which to save the key (/home/your-user/.ssh/id_ed25519):
```

Press `Enter` to use the default path.

### Verify the key files

```bash
ls -la ~/.ssh
```

You should see files similar to:

- `id_ed25519`
- `id_ed25519.pub`

### Set permissions

```bash
chmod 600 ~/.ssh/id_ed25519
chmod 644 ~/.ssh/id_ed25519.pub
```

## 3. Add the Public Key to GitHub

### Print the public key

```bash
cat ~/.ssh/id_ed25519.pub
```

Copy the full output.

### Add it in GitHub

1. Open GitHub
2. Go to `Settings`
3. Go to `SSH and GPG keys`
4. Click `New SSH key`
5. Enter a title such as `WSL Ubuntu`
6. Paste the public key
7. Click `Add SSH key`

## 5. Test the GitHub SSH Connection

Run:

```bash
ssh -T git@github.com
```

The first time, you may see a message like:

```text
The authenticity of host 'github.com (...)' can't be established.
```

Type:

```text
yes
```

Then you should see a success message similar to:

```text
Hi <your-github-name>! You've successfully authenticated, but GitHub does not provide shell access.
```
