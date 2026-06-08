> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Sports catalog

> List sports and leagues for sport/league filters on markets, events, and search.

Use this endpoint to discover valid **`sport`** and **`league`** slugs before calling [List markets](/api-reference/markets/list-markets), [List events](/api-reference/events/list-events), or [Search](/api-reference/search/search).

## Query Parameters

This endpoint does not accept query parameters today.

<Info>
  Use slugs from this catalog on [List markets](/api-reference/markets/list-markets), [List events](/api-reference/events/list-events), and search endpoints:

  * **`sport`** — comma-separated sport slugs (for example `basketball` or `basketball,football`).
  * **`league`** — comma-separated league slugs from each sport's `leagues` array. You must also send a matching **`sport`** on the same request.
</Info>

## Response

<ResponseField name="data" type="array">
  Sports catalog rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="name" type="string">
    Display name of the sport.
  </ResponseField>

  <ResponseField name="slug" type="string">
    Sport slug. Pass this value in comma-separated `sport` query parameters on list and search endpoints.
  </ResponseField>

  <ResponseField name="leagues" type="array">
    Leagues under this sport. Each item:
  </ResponseField>

  <Expandable title="leagues[] item">
    <ResponseField name="name" type="string">
      Display name of the league.
    </ResponseField>

    <ResponseField name="slug" type="string">
      League slug. Pass this value in comma-separated `league` query parameters. **`league` requires `sport`** on the same request.
    </ResponseField>
  </Expandable>
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
    "data": [
      {
        "name": "Basketball",
        "slug": "basketball",
        "leagues": [
          { "name": "NBA", "slug": "nba" }
        ]
      }
    ],
    "meta": {},
    "error": null
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
