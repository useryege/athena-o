# Squid VPS Proxy

This guide deploys a password-protected Squid HTTP proxy on a VPS for
authorized Etherscan live probes.

The default target is:

```bash
root@47.245.183.140
```

The proxy always listens on `6776/tcp` and accepts connections from
`0.0.0.0/0`. Access is controlled by Squid basic authentication, and proxy use
is limited to HTTPS `CONNECT` requests to target port `443`.

## Deploy

Recommended deployment:

```bash
set -a
source .env
set +a

make deploy-squid-vps
```

`SQUID_PASSWORD` must be set in the environment before running the deploy
target. The script does not generate remote passwords.

## Variables

| Name | Default | Description |
| --- | --- | --- |
| `SQUID_HOST` | `root@47.245.183.140` | SSH target for the VPS. |
| `SQUID_USER` | `athena_probe` | Squid basic auth username. |
| `SQUID_PASSWORD` | required | URL-safe Squid basic auth password. |
| `SQUID_PUBLIC_HOST` | host part of `SQUID_HOST` | Host printed in the generated proxy URL. |
| `SQUID_SSH_ARGS` | unset | Extra SSH arguments, such as `-i ~/.ssh/key`. |

The script supports Ubuntu/Debian hosts with `apt-get` and installs `squid` and
`apache2-utils`.

## Security

- `SQUID_PASSWORD` is not printed by the deploy script.
- The remote host stores only the htpasswd hash in `/etc/squid/athena-passwd`.
- Squid only allows authenticated `CONNECT` requests to target port `443`.
- Proxy URLs, passwords, and API keys must stay out of the repository.

In Alibaba Cloud, allow inbound `TCP 6776/6776` from `0.0.0.0/0` in the
security group.

## Use With Etherscan Probe

The proxy URL format is:

```text
http://athena_probe:<SQUID_PASSWORD>@47.245.183.140:6776
```

Use it with the proxy multi-key probe:

```bash
set -a
source .env
set +a

ATHENA_E2E_ETHERSCAN_API_KEYS="$(
  rg -o '"[A-Za-z0-9]{34}"' docs/etherscan-configuration.md \
    | tr -d '"' \
    | paste -sd, -
)" \
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_PROBE=1 \
ATHENA_E2E_ETHERSCAN_PROXY_URLS="http://athena_probe:${SQUID_PASSWORD}@47.245.183.140:6776" \
make e2e-live-etherscan-proxy-multi-key-rate-limit
```

## Operations

Check the service:

```bash
ssh root@47.245.183.140 'systemctl status squid --no-pager'
```

View logs:

```bash
ssh root@47.245.183.140 'tail -n 100 /var/log/squid/access.log'
ssh root@47.245.183.140 'tail -n 100 /var/log/squid/cache.log'
```

Stop and disable:

```bash
ssh root@47.245.183.140 'systemctl disable --now squid'
```
