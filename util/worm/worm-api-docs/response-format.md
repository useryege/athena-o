> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Response Format

> Standard JSON envelope used by all Worm API responses.

All responses are returned as `application/json` and follow a consistent three-field envelope.

## Envelope Structure

| Field   | Type                      | Description                                                                                                                                                                                     |
| ------- | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `data`  | `object \| array \| null` | The endpoint payload. An object for detail endpoints, an array for list endpoints, or `null` on error.                                                                                          |
| `meta`  | `object \| null`          | On success, typically `{}` for single-resource endpoints, or — for paginated list/search endpoints — `limit` and `next_cursor` (see [Pagination](/api-reference/pagination)). On error, `null`. |
| `error` | `object \| null`          | Always `null` on success. Contains error details on failure — see [Errors](/api-reference/errors).                                                                                              |

## Success Example

```json theme={null}
{
  "data": {
    "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
    "title": "Will BTC reach $150k by end of 2026?"
  },
  "meta": {},
  "error": null
}
```

## List Example

```json theme={null}
{
  "data": [
    {
      "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
      "title": "Will BTC reach $150k by end of 2026?"
    }
  ],
  "meta": {
    "limit": 20,
    "next_cursor": "eyJiZWZvcmVfaWQiOjEyMzQ1fQ=="
  },
  "error": null
}
```

## Error Example

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
