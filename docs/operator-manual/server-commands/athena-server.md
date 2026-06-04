# `athena-server` Command Reference

## athena-server

Run the Athena API server

### Synopsis

The API server is a gRPC/REST server which exposes the API consumed by the Web UI, CLI, and CI/CD systems.  This command runs API server in the foreground.  It can be configured by following options.

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
      --address string                                Listen on given address (default "0.0.0.0")
      --api-content-types string                      Semicolon separated list of allowed content types for non GET api requests. Any content type is allowed if empty. (default "application/json")
      --app-state-cache-expiration duration           Cache expiration for app state (default 1h0m0s)
      --application-server-address string             Athena application server address (default "localhost:8082")
      --basehref string                               Value for base href in index.html. Used if Athena is running behind reverse proxy under subpath different from / (default "/")
      --connection-status-cache-expiration duration   Cache expiration for cluster/repo connection status (default 1h0m0s)
      --content-security-policy value                 Set Content-Security-Policy header in HTTP responses to value. To disable, set to "". (default "frame-ancestors 'self';")
      --default-cache-expiration duration             Cache expiration default (default 24h0m0s)
      --disable-auth                                  Disable client authentication
      --enable-gzip                                   Enable GZIP compression (default true)
      --gloglevel int                                 Set the glog logging level
  -h, --help                                          help for athena-server
      --logformat string                              Set the logging format. One of: json|text (default "json")
      --login-attempts-expiration duration            Cache expiration for failed login attempts. DEPRECATED: this flag is unused and will be removed in a future version. (default 24h0m0s)
      --loglevel string                               Set the logging level. One of: debug|info|warn|error (default "info")
      --notification-server-address string            Athena notification server address (default "localhost:8086")
      --otlp-address string                           OpenTelemetry collector address to send traces to
      --otlp-attrs strings                            List of OpenTelemetry collector extra attrs when send traces, each attribute is separated by a colon(e.g. key:value)
      --otlp-headers stringToString                   List of OpenTelemetry collector extra headers sent with traces, headers are comma-separated key-value pairs(e.g. key1=value1,key2=value2) (default [])
      --otlp-insecure                                 OpenTelemetry collector insecure mode (default true)
      --polymarket-server-address string              Athena polymarket server address (default "localhost:8092")
      --port int                                      Listen on given port (default 8080)
      --redis string                                  Redis server hostname and port (e.g. athena-redis:6379). 
      --redis-compress string                         Enable compression for data sent to Redis with the required compression algorithm. (possible values: gzip, none) (default "gzip")
      --redisdb int                                   Redis database.
      --rootpath string                               Used if Athena is running behind reverse proxy under subpath different from /
      --sentinel stringArray                          Redis sentinel hostname and port (e.g. athena-redis-ha-announce-0:6379). 
      --sentinelmaster string                         Redis sentinel master group name. (default "master")
      --solidity-server-address string                Athena solidity server address (default "localhost:8090")
      --staticassets string                           Directory path that contains additional static assets (default "/shared/app")
      --wallet-server-address string                  Athena wallet server address (default "localhost:8088")
      --worm-server-address string                    Athena worm server address (default "localhost:8084")
      --x-frame-options value                         Set X-Frame-Options header in HTTP responses to value. To disable, set to "". (default "sameorigin")
```

### SEE ALSO

* [athena-server version](athena-server_version.md)	 - Print version information
