# Authentication

Source: https://docs.worm.wtf/api-reference/authentication

How HMAC authentication works for protected Worm API endpoints.

## Public vs Authenticated

Most read-only endpoints are public. Auth is required for trading, account, margin, redeem, and key-management operations.

| Endpoint Type                                                                              | Auth         |
| ------------------------------------------------------------------------------------------ | ------------ |
| Markets, events, search, orderbook, prices, candles, market trades, market margin activity | None         |
| Orders, account, margin, redeems, auth-keys list/revoke                                    | HMAC headers |

## Required Headers

All authenticated requests require these headers:

<ParamField type="string">
  API key id.
</ParamField>

<ParamField type="string">
  Current unix timestamp in seconds.
</ParamField>

<ParamField type="string">
  Lowercase hex `HMAC-SHA256` digest of the signing payload.
</ParamField>

## Signature Payload

The signature payload is:

```text theme={null}
{timestamp}{HTTP_METHOD}{full_path_with_query}{raw_body}
```

Example payload for `GET /orders/?limit=10` with no body:

```text theme={null}
1714300000GET/orders/?limit=10
```

The signed `path` must match **`{path}?{query}`** exactly as on the request URL you send to **`https://api.worm.wtf`** (including leading slash and query string). Edge routing may map these URLs to internal handlers; always sign what you actually request.

## Bootstrap Flow

Generate API credentials once, then use them for HMAC:

<Steps>
  <Step title="Request challenge">
    `POST https://api.worm.wtf/auth/keys/challenge/` with `wallet_address` to receive `nonce`, `message`, and `expires_in_seconds`.
  </Step>

  <Step title="Sign with wallet">
    Sign the **`message`** bytes (UTF-8) with the same Solana keypair; send the signature as hex to the create step.
  </Step>

  <Step title="Create API key">
    `POST https://api.worm.wtf/auth/keys/create/` with `wallet_address`, `message`, `signature`, and `nonce` to receive `api_key` and `secret`.
  </Step>

  <Step title="Store credentials">
    Persist the key material securely. The secret is shown once.
  </Step>
</Steps>

<Note>
  The **`signature` here** is only for proving you control the wallet: sign the exact UTF-8 **`message`** from the challenge step and send it as **hex**. Submit endpoints for orders, margin requests, and redeems use a different **`message`** (the draft program-call payload from the API), but the same idea: sign that **`message`** and send the hex **`signature`** in the submit body.
</Note>

## Signing Python Example

The sample below bootstraps an API key (challenge → sign → create), then calls a signed endpoint with HMAC. Install HTTP and signing dependencies:

```bash theme={null}
pip install requests solders base58
```

The [`solana`](https://pypi.org/project/solana/) package on PyPI is optional; it depends on `solders` and fits stacks that already use `solana-py`.

```python theme={null}
import hashlib
import hmac
import os
import time

import base58
import requests
from solders.keypair import Keypair

BASE = "https://api.worm.wtf"


def load_keypair(secret: str) -> Keypair:
    """Load a Solana keypair from base58 secret bytes or 128-char hex (full keypair)."""
    s = secret.strip()
    if len(s) == 128 and all(c in "0123456789abcdefABCDEF" for c in s):
        return Keypair.from_bytes(bytes.fromhex(s))
    return Keypair.from_bytes(base58.b58decode(s))


def unwrap(response: requests.Response) -> dict:
    """Return ``data`` from the standard API envelope or raise on error."""
    response.raise_for_status()
    body = response.json()
    err = body.get("error")
    if err is not None:
        raise RuntimeError(err)
    return body["data"]