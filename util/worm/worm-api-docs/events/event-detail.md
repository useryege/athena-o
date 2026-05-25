> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Get event by condition id

> Retrieve detailed information for a single event.

## Path Parameters

<ParamField path="condition_id" type="string" required>
  The unique condition identifier of the event.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Detailed event payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="condition_id" type="string">
    Event condition id.
  </ResponseField>

  <ResponseField name="title" type="string">
    Event title.
  </ResponseField>

  <ResponseField name="description" type="string">
    Event description.
  </ResponseField>

  <ResponseField name="logo" type="string | null">
    Event logo URL when present.
  </ResponseField>

  <ResponseField name="video_url" type="string | null">
    Optional event media URL.
  </ResponseField>

  <ResponseField name="category" type="string">
    Category slug. The platform uses a fixed set (not user-defined): `politics`, `sports`, `crypto`, `wtf`, `tech`, `finance`.
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Created time (unix seconds).
  </ResponseField>

  <ResponseField name="markets" type="array">
    All related market summaries; each item matches [List markets](/api-reference/markets/list-markets) row shape.
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object">
  Empty object on success for this endpoint.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": {
      "condition_id": "9x4H2LdQ7sM1Kp6Wv3Tn8YfR5aC2uJ7mB4eQ9zN6dLp",
      "title": "Boston Celtics vs Philadelphia 76ers (Apr 28, 2026)",
      "description": "NBA event slate for Apr 28, 2026.",
      "logo": "https://cdn.worm.wtf/e/42.png",
      "video_url": null,
      "category": "sports",
      "created": 1714300000,
      "markets": [
        {
          "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
          "description": "",
          "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
          "logo": null,
          "last_trade_price": "0.68",
          "state": "open",
          "category": "sports",
          "created": 1714300100,
          "creator": { "username": "creator1", "image": null, "twitter_username": null },
          "event": { "title": "Boston Celtics vs Philadelphia 76ers (Apr 28, 2026)", "condition_id": "9x4H2LdQ7sM1Kp6Wv3Tn8YfR5aC2uJ7mB4eQ9zN6dLp", "logo": "https://cdn.worm.wtf/e/42.png" },
          "margin_enabled": true
        }
      ]
    },
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

  ```json 404 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -15,
      "slug": "not_found",
      "message": "Not found.",
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
