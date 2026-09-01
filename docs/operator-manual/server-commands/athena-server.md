# `athena-server` Command Reference

## athena-server

Run the Athena API server

### Synopsis

The API server is a gRPC/REST server which exposes the API consumed by the Web UI, CLI, and CI/CD systems. This command runs API server in the foreground. It can be configured by following options.

ATHENA_WALLET_INTERNAL_AUTH_TOKEN must contain at least 32 bytes without whitespace and must match the Wallet service value. It is an internal service credential, not an Athena user API Key, and the server refuses startup when it is absent or invalid.

```
athena-server [flags]
```

### Examples

```
  # Start the Athena API server with default settings
  $ athena-server
  
  # Start the Athena API server on a custom port and enable tracing
  $ athena-server --port 8888 --otlp-address localhost:4317
```

### Options

```
      --address string                                 Listen on given address (default "0.0.0.0")
      --api-content-types string                       Semicolon separated list of allowed content types for non GET api requests. Any content type is allowed if empty. (default "application/json")
      --app-state-cache-expiration duration            Cache expiration for app state (default 1h0m0s)
      --basehref string                                Value for base href in index.html. Used if Athena is running behind reverse proxy under subpath different from / (default "/")
      --connection-status-cache-expiration duration    Cache expiration for cluster/repo connection status (default 1h0m0s)
      --content-security-policy value                  Set Content-Security-Policy header in HTTP responses to value. To disable, set to "". (default "frame-ancestors 'self';")
      --default-cache-expiration duration              Cache expiration default (default 24h0m0s)
      --disable-auth                                   Disable client authentication
      --enable-gzip                                    Enable GZIP compression (default true)
      --etherscan-api-keys string                      Comma, space, or newline-separated Etherscan API keys used by Etherscan Gateway probe runs
      --etherscan-gateway-auth-token string            Bearer token for Etherscan Gateway gRPC status calls
      --etherscan-gateway-ips string                   Comma, space, or newline-separated Etherscan Gateway IP addresses
      --etherscan-gateway-probe-query-address string   Ethereum address used by Etherscan Gateway probe runs
      --gloglevel int                                  Set the glog logging level
  -h, --help                                           help for athena-server
      --logformat string                               Set the logging format. One of: json|text (default "json")
      --loglevel string                                Set the logging level. One of: debug|info|warn|error (default "info")
      --managed-oo-server-address string               Athena Managed OO server address (default "127.0.0.1:8106")
      --market-radar-server-address string             Athena Market Radar server address (default "127.0.0.1:8092")
      --notification-server-address string             Athena notification server address (default "127.0.0.1:8086")
      --otlp-address string                            OpenTelemetry collector address to send traces to
      --otlp-attrs strings                             List of OpenTelemetry collector extra attrs when send traces, each attribute is separated by a colon(e.g. key:value)
      --otlp-headers stringToString                    List of OpenTelemetry collector extra headers sent with traces, headers are comma-separated key-value pairs(e.g. key1=value1,key2=value2) (default [])
      --otlp-insecure                                  OpenTelemetry collector insecure mode (default true)
      --port int                                       Listen on given port (default 8080)
      --profit-sharing-server-address string           Athena Profit Sharing server address (default "127.0.0.1:8108")
      --redis string                                   Redis server hostname and port (e.g. athena-redis:6379).
      --redis-compress string                          Enable compression for data sent to Redis with the required compression algorithm. (possible values: gzip, none) (default "gzip")
      --redisdb int                                    Redis database.
      --rootpath string                                Used if Athena is running behind reverse proxy under subpath different from /
      --sentinel stringArray                           Redis sentinel hostname and port (e.g. athena-redis-ha-announce-0:6379).
      --sentinelmaster string                          Redis sentinel master group name. (default "master")
      --sports-history-server-address string           Athena Sports History server address (default "127.0.0.1:8104")
      --sports-live-server-address string              Athena Sports Live server address (default "127.0.0.1:8094")
      --staticassets string                            Directory path that contains additional static assets (default "/shared/app")
      --token-api-server-address string                Athena token API server address (default "127.0.0.1:8096")
      --wallet-server-address string                   Athena wallet server address (default "127.0.0.1:8088")
      --worm-markets-server-address string             Athena Worm Markets server address (default "127.0.0.1:8084")
      --x-frame-options value                          Set X-Frame-Options header in HTTP responses to value. To disable, set to "". (default "sameorigin")
```

### Disabled-auth development mode

`--disable-auth` is restricted to non-Compose development processes listening on a
loopback address. At startup, the server creates or reuses both isolated development
accounts: `local-user` for the member application and `local-admin` for the
administrator application. There is no role-selection flag or environment variable.

Every disabled-auth request must select its application realm explicitly with
`X-Athena-Application-Realm: member` or
`X-Athena-Application-Realm: admin`. Browser transports that cannot set headers use
the `athenaRealm` query parameter for private avatar GETs and EventSource requests.
When both forms are present they must agree. Missing, invalid, repeated, or conflicting
realm values do not fall back to either identity.

With disabled auth enabled, `make run` serves the member application at `/` and the
administrator application at `/admin/` at the same time; each frontend supplies its
own realm. Before returning to normal authentication, run `make run-reset` so the
persisted development identities cannot conflict with the authenticated account
directory. Production Compose keeps authentication enabled, and deployment tooling
rejects attempts to enable disabled-auth mode.

### SEE ALSO

* [athena-server version](athena-server_version.md)	 - Print version information
