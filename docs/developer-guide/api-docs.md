# API Docs

You can find the Swagger docs by setting the path to `/swagger-ui` in your Athena UI. E.g. [http://localhost:8080/swagger-ui](http://localhost:8080/swagger-ui) or [http://localhost:4000/swagger-ui](http://localhost:4000/swagger-ui).

## Authorization

Browser users authenticate with Google at `/auth/google/login`. Athena completes the
OIDC callback server-side and stores its own session token in an HttpOnly cookie; Google
tokens are never used for API requests.

CLI and automation clients must create an Athena API Key from **Account Center →
Security** after signing in. Copy the key when it is issued, then export it locally:

```bash
export ATHENA_TOKEN='<newly-issued-athena-api-key>'
```

Pass it using the HTTP `Authorization` header, prefixing with `Bearer `:

```bash
$ curl $ATHENA_SERVER/api/v1/version -H "Authorization: Bearer $ATHENA_TOKEN"
{"Version":"...","BuildDate":"...","GitCommit":"...","GitTreeState":"...","GoVersion":"...","Compiler":"...","Platform":"..."}
```

Only Athena token version 2 is accepted. Rotating `ATHENA_JWT_SECRET`, deleting the API
Key, disabling its account, or removing its JTI from account metadata invalidates it.
