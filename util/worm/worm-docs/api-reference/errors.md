> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Errors

> Error envelope, status semantics, and common API error slugs.

All non-success responses use the same [response envelope](/api-reference/response-format): `data` is `null`, `meta` is `null`, and `error` is populated.

## HTTP Status Codes

| Status | Meaning                                                                                         |
| ------ | ----------------------------------------------------------------------------------------------- |
| `400`  | Bad Request — validation failure or malformed input                                             |
| `401`  | Unauthorized — missing or invalid authentication                                                |
| `403`  | Forbidden — valid credentials but insufficient permissions                                      |
| `404`  | Not Found — resource does not exist                                                             |
| `429`  | Too Many Requests — [rate limit](/api-reference/rate-limits) exceeded                           |
| `501`  | Not Implemented — endpoint or backend combination not supported (for example Hyperliquid TP/SL) |
| `500`  | Internal Server Error                                                                           |

## Error Envelope

```json theme={null}
{
  "data": null,
  "meta": null,
  "error": {
    "code": -11,
    "slug": "invalid_request_params",
    "message": "Validation Error",
    "details": [
      { "limit": "Ensure this value is less than or equal to 100." }
    ]
  }
}
```

## Error Object Fields

| Field     | Type            | Description                                    |
| --------- | --------------- | ---------------------------------------------- |
| `code`    | `integer`       | Numeric error code for programmatic handling   |
| `slug`    | `string`        | Stable, machine-readable error key             |
| `message` | `string`        | Human-readable error summary                   |
| `details` | `array \| null` | Optional list of field-level validation errors |

## Common Error Slugs

| Slug                     | HTTP Status | Description                                                                        |
| ------------------------ | ----------- | ---------------------------------------------------------------------------------- |
| `invalid_request_params` | 400         | One or more query/body parameters failed validation                                |
| `not_authenticated`      | 401         | Authenticated endpoint called without valid HMAC headers (no credentials supplied) |
| `authentication_failed`  | 401         | HMAC headers present but invalid (bad key, timestamp, or signature)                |
| `permission_denied`      | 403         | Authenticated but not authorized for this action                                   |
| `not_found`              | 404         | The requested resource was not found                                               |
| `throttled`              | 429         | Rate limit exceeded — retry after the `X-RateLimit-Reset` window                   |
| `internal_error`         | 500         | Unexpected server error                                                            |

Some domain errors use positive numeric **`code`** values and backend-specific slugs (for example **`9201`** / `hyperliquid_margin_tp_sl_not_supported` on [Set TP/SL](/api-reference/margin/set-tp-sl) for Hyperliquid positions).

## Validation Error Shape

Validation failures include field-level entries in `error.details`:

```json theme={null}
{
  "data": null,
  "meta": null,
  "error": {
    "code": -11,
    "slug": "invalid_request_params",
    "message": "Validation Error",
    "details": [
      { "limit": "Ensure this value is less than or equal to 100." }
    ]
  }
}
```
