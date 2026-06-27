# Wrapper Contract

## Client shape

Use this structure per API family:
- `Default<Base>URL`
- `DefaultTimeout`
- `<Xxx>Config`
- `<Xxx>Client` interface
- `New<Xxx>Client(config)`
- private `<xxx>ClientImpl`
- private `do(ctx, method, path, query, body, out)`

## Method rules

- Accept `context.Context` in every method.
- Use typed option structs for query parameters.
- Use pointer fields for optional booleans/numbers when `false/0` is meaningful.
- Escape path params with `url.PathEscape`.
- Encode repeated values with repeated query keys.

## Model rules

- Keep top-level payload strongly typed.
- Represent nullable API fields with pointers.
- Use `json.RawMessage` only for unstable deep nested fragments.
- Avoid `any` unless payload can be structurally variant by design.

## Error rules

Implement typed `APIError`:
- `StatusCode int`
- `Type string` (optional)
- `Message string`
- `RawBody string` (size-limited)

Support both JSON and plain-text error bodies.

## HTTP rules

- Build URLs with `net/url`.
- Marshal request body exactly once.
- Set `Content-Type: application/json` only with JSON body.
- Decode envelope/non-envelope explicitly per endpoint group.

## Drift adaptation rules

- If online payload diverges from docs, treat live payload as source for runtime behavior.
- Prefer widening nested fields (`json.RawMessage`) over broad `any` at top-level.
- Keep method signatures stable when possible; change only when evidence requires it.
