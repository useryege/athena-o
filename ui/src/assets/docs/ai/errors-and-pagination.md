# Athena Errors and Pagination

[The Swagger 2.0 specification](/swagger.json) is authoritative for each operation's response schema and query parameters. Athena's HTTP surface is generated from gRPC contracts, so errors use the gateway `runtimeError` model rather than a product-specific envelope for every endpoint.

## Error Responses

An error response can contain:

```json
{
  "error": "ACCOUNT_DATA_ACCESS_DENIED",
  "code": 7,
  "message": "ACCOUNT_DATA_ACCESS_DENIED",
  "details": [
    {
      "type_url": "type.googleapis.com/google.rpc.ErrorInfo",
      "value": "<base64-encoded ErrorInfo>"
    }
  ]
}
```

- `error` is the gateway-compatible copy of the service error message.
- `code` is the numeric canonical gRPC status code.
- `message` is the stable reason or human-readable error text supplied by the service.
- `details` is optional structured status detail. Athena's authorization denials use `google.rpc.ErrorInfo`; its decoded metadata can identify the module plus required and effective access levels.

Clients should prefer `code` and `message` over the compatibility `error` field when they are available.

Always use the HTTP status for transport handling and retain the response body for diagnostics. Common mappings include:

| HTTP | Typical gRPC code | Meaning |
| --- | --- | --- |
| `400` | `3` or `9` | Invalid input or failed precondition |
| `401` | `16` | Missing, invalid, or expired Athena credential |
| `403` | `7` | Authenticated account lacks the required role, entitlement, or module level |
| `404` | `5` | Resource is not visible or does not exist |
| `409` | `10` | Aborted operation, including a revision conflict |
| `500` | `13` | Internal failure |
| `503` | `14` | Service unavailable or account maintenance |

Do not retry `400`, `401`, `403`, or `404` blindly. Refresh authorization after a permission change. Retry transient `503` responses with bounded exponential backoff, and retry a revision conflict only after reading the latest state.

## Pagination Patterns

Athena does not impose one pagination shape on every list operation. Follow the exact operation in [the Swagger specification](/swagger.json).

### Page and Page Size

Many durable lists use one-based `page` plus `page_size`, and commonly return `total`, `page`, and `page_size` alongside the items. The account directory instead names its query parameter `pageSize`. Defaults and maximums are operation-specific.

```http
GET /api/v1/tokens/projects?page=1&page_size=20
```

### Limit

Snapshot and bounded-list operations can use `limit` without a continuation token. A smaller result may mean that the current snapshot contains no more matching items. Limits and defaults are operation-specific.

```http
GET /api/v1/market-radar/hot-markets?limit=100
```

### Cursor

Do not parse a cursor, synthesize page numbers for a cursor endpoint, or assume that a `limit` endpoint returns a total count. Preserve the filters and sort option across every page of one traversal.
