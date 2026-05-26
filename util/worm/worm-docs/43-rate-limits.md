# Rate Limits

Source: https://docs.worm.wtf/api-reference/rate-limits

Throttle scopes, limits, response headers, and retry guidance.

Worm applies per-scope throttling to protect API reliability.

## Scopes

| Scope                | Limit     | Applies To                                      |
| -------------------- | --------- | ----------------------------------------------- |
| `api_public`         | `120/min` | Public endpoints                                |
| `api_authenticated`  | `300/min` | Authenticated endpoints                         |
| `api_auth_bootstrap` | `10/min`  | Key bootstrap endpoints (`challenge`, `create`) |

## Rate Limit Headers

Responses include:

| Header                  | Description                                    |
| ----------------------- | ---------------------------------------------- |
| `X-RateLimit-Limit`     | Maximum requests allowed in the current window |
| `X-RateLimit-Remaining` | Requests remaining in the current window       |
| `X-RateLimit-Reset`     | Unix timestamp when the window resets          |

## 429 behavior

When a request is throttled, the API returns `429` with the standard error envelope:

```json 429 theme={null}
{
  "data": null,
  "meta": null,
  "error": {
    "code": -17,
    "slug": "throttled",
    "message": "Request was throttled. Expected available in 12 seconds.",
    "details": []
  }
}
```

## Retry Strategy

When you receive `429`:

1. Prefer the `Retry-After` response header (seconds until retry) when present.
2. Otherwise use `X-RateLimit-Reset` to determine when the window resets.
3. Add jittered exponential backoff for bursts.
4. Cache repeated reads and avoid polling tight loops.