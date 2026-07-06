# 3proxy VPS Proxy

This guide deploys a password-protected 3proxy HTTP proxy on a VPS for
authorized Etherscan live probes.

The default target is:

```bash
root@47.245.183.140
```

## Deploy

Recommended deployment:

```bash
THREEPROXY_PASSWORD='strong-url-safe-password' \
make deploy-3proxy-vps
```

If `THREEPROXY_PASSWORD` is omitted, the remote host generates one and stores it
in `/root/.athena-3proxy.env` with root-only permissions. Re-running the deploy
script reuses that password unless you explicitly pass a new one.

The proxy always listens on `6776/tcp` and allows source CIDR `0.0.0.0/0`.
Access is controlled by 3proxy username/password authentication.

## Variables

| Name | Default | Description |
| --- | --- | --- |
| `THREEPROXY_HOST` | `root@47.245.183.140` | SSH target for the VPS. |
| `THREEPROXY_USER` | `athena_probe` | 3proxy username. |
| `THREEPROXY_PASSWORD` | generated remotely | URL-safe 3proxy password. |
| `THREEPROXY_PUBLIC_HOST` | host part of `THREEPROXY_HOST` | Host printed in the generated proxy URL. |
| `THREEPROXY_RELEASE_VERSION` | `0.9.4` | 3proxy release used for the fallback Debian package install. |
| `THREEPROXY_SSH_ARGS` | unset | Extra SSH arguments, such as `-i ~/.ssh/key`. |

The script supports Ubuntu/Debian hosts with `apt-get`. It first tries
`apt-get install 3proxy`; if the package is unavailable, it downloads the
official `3proxy-0.9.4.x86_64.deb` release from GitHub.

## Security

The deployed proxy accepts connections from any IPv4 source, but it is not
anonymous:

- `auth strong` requires username/password authentication.
- ACLs allow source CIDR `0.0.0.0/0`.
- ACLs only allow HTTPS target port `443`.
- Proxy URLs, passwords, and API keys must stay out of the repository.
- If UFW is active, the script adds an allow rule for `6776/tcp`.

In Alibaba Cloud, allow inbound `TCP 6776/6776` from `0.0.0.0/0` in the
security group.

## Use With Etherscan Probe

The deploy script prints a proxy URL like:

```text
http://athena_probe:<password>@47.245.183.140:6776
```

Use it with the proxy multi-key probe:

```bash
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_PROBE=1 \
ATHENA_E2E_ETHERSCAN_API_KEYS='key1,key2,key3' \
ATHENA_E2E_ETHERSCAN_PROXY_URLS='http://athena_probe:<password>@47.245.183.140:6776' \
make e2e-live-etherscan-proxy-multi-key-rate-limit
```

## Operations

Check the service:

```bash
ssh root@47.245.183.140 'systemctl status 3proxy --no-pager'
```

View logs:

```bash
ssh root@47.245.183.140 'tail -n 100 /var/log/3proxy/3proxy.log'
```

Stop and disable:

```bash
ssh root@47.245.183.140 'systemctl disable --now 3proxy'
```

Inspect generated credentials:

```bash
ssh root@47.245.183.140 'cat /root/.athena-3proxy.env'
```

## References

- [3proxy GitHub](https://github.com/3proxy/3proxy)
- [3proxy releases](https://github.com/3proxy/3proxy/releases)
- [3proxy official site](https://3proxy.org/)
