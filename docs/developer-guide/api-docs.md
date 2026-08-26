# API Docs

You can find the Swagger docs by setting the path to `/swagger-ui` in your Athena UI. E.g. [http://localhost:8080/swagger-ui](http://localhost:8080/swagger-ui) or [http://localhost:4000/swagger-ui](http://localhost:4000/swagger-ui).

## Authorization

Browser users authenticate either through Google at `/auth/google/login` or a desktop
Phantom Solana provider through `/auth/phantom/challenge` and
`/auth/phantom/verify`. Athena verifies OIDC or the server-generated SIWS message, then
stores its own session token in an HttpOnly cookie. Google tokens and wallet signatures
are never used for business API requests.

For an unknown Google subject or Solana address, provider verification creates a short-lived
shared registration ticket and directs the browser to `/register`; it does not create an
account or issue the session cookie. The registration API is
`GET|POST|DELETE /auth/registration` plus
`GET /auth/registration/username-availability`. The user must first choose a permanent
username. Successful registration creates the UUID `account_id` used by JWT subjects and
authorization, while API and UI display the username separately. Google and Phantom
identities always create separate accounts.

Phantom login accepts only the injected desktop extension in the Web UI. Challenge creation
accepts `{address, returnTo}` and returns `{message, expiresAt}`; verification accepts only a
raw-base64url `{signature}` and returns `{redirectTo}`. The server performs Ed25519
verification without Solana RPC and never asks for a transaction, private key, or fee.

CLI and automation clients must create an Athena API Key from **Account Center →
Security** after an administrator enables the account's independent API Key access.
Copy the key when it is issued, then export it locally:

```bash
export ATHENA_TOKEN='<newly-issued-athena-api-key>'
```

Pass it using the HTTP `Authorization` header, prefixing with `Bearer `:

```bash
$ curl $ATHENA_SERVER/api/v1/version -H "Authorization: Bearer $ATHENA_TOKEN"
{"Version":"...","BuildDate":"...","GitCommit":"...","GitTreeState":"...","GoVersion":"...","Compiler":"...","Platform":"..."}
```

Only Athena token version 3 is accepted. Rotating `ATHENA_JWT_SECRET`, deleting the API
Key, or disabling its account invalidates it. Turning off API Key access pauses existing
keys without deleting them; re-enabling access restores undeleted, unexpired keys.
