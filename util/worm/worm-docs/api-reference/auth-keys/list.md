> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# List API keys

> Retrieve active (non-revoked) API credentials for the authenticated user.

## Response

<ResponseField name="data" type="array">
  Active API credential rows (`revoked_at` is always `null` on this endpoint).
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="key_id" type="string">
    Credential key id (opaque url-safe token from create).
  </ResponseField>

  <ResponseField name="created" type="integer">
    Created timestamp (unix seconds).
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object">
  Empty object on success (`{}`). This listing is not paginated.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
        "key_id": "xK9mP2nQ4rT6vW8y",
        "created": 1714300000,
        "revoked_at": null
      }
    ],
    "meta": {},
    "error": null
  }
  ```

  ```json 400 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -11,
      "slug": "invalid_request_params",
      "message": "Validation Error",
      "details": []
    }
  }
  ```

  ```json 401 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -12,
      "slug": "authentication_failed",
      "message": "Invalid signature",
      "details": []
    }
  }
  ```

  ```json 403 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -14,
      "slug": "permission_denied",
      "message": "You do not have permission to perform this action.",
      "details": []
    }
  }
  ```

  ```json 429 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -17,
      "slug": "throttled",
      "message": "Request was throttled.",
      "details": []
    }
  }
  ```

  ```json 500 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -3,
      "slug": "internal_error",
      "message": "An internal error occurred.",
      "details": []
    }
  }
  ```
</ResponseExample>
