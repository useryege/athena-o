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
      --address string                  Listen on given address (default "0.0.0.0")
      --api-content-types string        Semicolon separated list of allowed content types for non GET api requests. Any content type is allowed if empty. (default "application/json")
      --as string                       Username to impersonate for the operation
      --as-group stringArray            Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                   UID to impersonate for the operation
      --basehref string                 Value for base href in index.html. Used if Argo CD is running behind reverse proxy under subpath different from / (default "/")
      --certificate-authority string    Path to a cert file for the certificate authority
      --client-certificate string       Path to a client certificate file for TLS
      --client-key string               Path to a client key file for TLS
      --cluster string                  The name of the kubeconfig cluster to use
      --content-security-policy value   Set Content-Security-Policy header in HTTP responses to value. To disable, set to "". (default "frame-ancestors 'self';")
      --context string                  The name of the kubeconfig context to use
      --dex-server string               Dex server address (default "athena-dex-server:5556")
      --disable-auth                    Disable client authentication
      --disable-compression             If true, opt-out of response compression for all requests to the server
      --enable-gzip                     Enable GZIP compression (default true)
      --gloglevel int                   Set the glog logging level
  -h, --help                            help for athena-server
      --insecure                        Run server without TLS
      --insecure-skip-tls-verify        If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string               Path to a kube config. Only required if out-of-cluster
      --logformat string                Set the logging format. One of: json|text (default "json")
      --loglevel string                 Set the logging level. One of: debug|info|warn|error (default "info")
      --metrics-address string          Listen for metrics on given address (default "0.0.0.0")
      --metrics-port int                Start metrics on given port (default 8083)
  -n, --namespace string                If present, the namespace scope for this CLI request
      --password string                 Password for basic authentication to the API server
      --port int                        Listen on given port (default 8080)
      --proxy-url string                If provided, this URL will be used to connect via proxy
      --request-timeout string          The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
      --rootpath string                 Used if Argo CD is running behind reverse proxy under subpath different from /
      --server string                   The address and port of the Kubernetes API server
      --staticassets string             Directory path that contains additional static assets (default "/shared/app")
      --tls-server-name string          If provided, this name will be used to validate server certificate. If this is not provided, hostname used to contact the server is used.
      --token string                    Bearer token for authentication to the API server
      --user string                     The name of the kubeconfig user to use
      --username string                 Username for basic authentication to the API server
      --x-frame-options value           Set X-Frame-Options header in HTTP responses to value. To disable, set to "". (default "sameorigin")
```

### SEE ALSO

* [athena-server version](athena-server_version.md)	 - Print version information

