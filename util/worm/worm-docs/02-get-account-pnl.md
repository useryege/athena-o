# Get account PnL

Source: https://docs.worm.wtf/api-reference/account/account-pnl

GET /account/pnl/
Retrieve your account profit and loss breakdown with optional market or event scope.

## Query Parameters

<ParamField type="string">
  Optional market condition id scope.
</ParamField>

<ParamField type="string">
  Optional event condition id scope.
</ParamField>

## Response

<ResponseField name="data" type="object">
  PnL breakdown payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="margin_position_settlements_pnl" type="string">
    Settlements PnL.
  </ResponseField>

  <ResponseField name="margin_positions_unrealized_pnl" type="string">
    Unrealized margin PnL.
  </ResponseField>

  <ResponseField name="redeems_funds" type="string">
    Redeem funds component.
  </ResponseField>

  <ResponseField name="active_assets_value" type="string">
    Active assets value component.
  </ResponseField>

  <ResponseField name="creator_fees" type="string">
    Creator fees component.
  </ResponseField>

  <ResponseField name="total_pnl" type="string">
    Total account PnL.
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
      "margin_position_settlements_pnl": "123.45",
      "margin_positions_unrealized_pnl": "42.10",
      "redeems_funds": "18.00",
      "active_assets_value": "950.00",
      "creator_fees": "6.75",
      "total_pnl": "190.30"
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