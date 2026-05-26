# --- HMAC-authenticated request ---

timestamp = str(int(time.time()))
method = "GET"
path = "/orders/?limit=10"
body = ""

payload = f"{timestamp}{method}{path}{body}".encode()
worm_signature = hmac.new(API_SECRET.encode(), payload, hashlib.sha256).hexdigest()

resp = requests.get(
    f"{BASE}{path}",
    headers={
        "WORM_API_KEY": API_KEY,
        "WORM_TIMESTAMP": timestamp,
        "WORM_SIGNATURE": worm_signature,
    },
    timeout=30,
)
resp.raise_for_status()
```

## Security Best Practices

<Warning>
  Never expose API secrets in client-side code. Treat API keys and HMAC secrets like passwords: they belong only in server-side apps, secret managers, or locked-down CI — never in browsers, mobile clients, or public source control.
</Warning>

<Warning>
  **Storage** — Keep `WORM_API_KEY` and `WORM_API_SECRET` in environment variables or a dedicated secret manager. Do not commit `.env` or similar files.

  **Signing** — Perform HMAC signing only on trusted backend infrastructure.

  **Rotation and revocation** — Rotate credentials regularly. Use **[List API keys](/api-reference/auth-keys/list)** (`GET /auth/keys/`) to see active keys, **[Revoke API key](/api-reference/auth-keys/revoke)** (`DELETE /auth/keys/{key_id}/`) to invalidate a key you no longer need, and the **[Create auth challenge](/api-reference/auth-keys/challenge)** / **[Create API key](/api-reference/auth-keys/create)** bootstrap flow to issue a replacement.
</Warning>

## Troubleshooting

| Error                                     | Typical Cause                        | Fix                                                   |
| ----------------------------------------- | ------------------------------------ | ----------------------------------------------------- |
| `Missing required authentication headers` | One or more required headers missing | Send all 3 headers                                    |
| `Timestamp out of range`                  | Client clock skew or delayed send    | Sync clock and sign immediately before request        |
| `Invalid API key`                         | Revoked/unknown key id               | Verify credentials or re-create key                   |
| `Invalid signature`                       | Payload mismatch                     | Verify exact payload format and path/query/body bytes |