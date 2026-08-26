# AI Discovery Documentation

## Scope

AI Discovery Documentation owns Athena's public, build-time documentation
surface for LLMs and API clients: the curated `/llms.txt` entry point, focused
Markdown guidance under `/docs/ai/`, and the generated Swagger 2.0 description
served at `/swagger.json` and rendered at `/swagger-ui`.

This capability describes and exposes the current HTTP API; it does not define
business RPC contracts, authenticate callers, authorize modules, or provide an
agent execution protocol. Athena does not publish its internal `docs/design/`
tree through this surface, and this capability does not implement MCP, OAuth,
scoped API Keys, `llms-full.txt`, or AI-specific write operations.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Curated public discovery entry | [ui/src/assets/llms.txt](../../../ui/src/assets/llms.txt) | `Start Here`, `API Reference`, `Optional` link groups |
| Public AI guidance | [ui/src/assets/docs/ai/](../../../ui/src/assets/docs/ai/) | overview, authentication, modules, errors and pagination, and safety documents |
| Vite public-file pipeline | [ui/vite.config.ts](../../../ui/vite.config.ts), [ui/package.json](../../../ui/package.json) | `publicDir`, `build.outDir`, `build` |
| Embedded static-file serving | [ui/embed.go](../../../ui/embed.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `ui.Embedded`, `NewServer`, `uiAssetExists`, `newStaticAssetsHandler`, `withRootPath` |
| Swagger source and generation | [internal/server/](../../../internal/server/), [hack/generate-proto.sh](../../../hack/generate-proto.sh) | protobuf HTTP annotations, `collect_swagger`, `clean_swagger` |
| Generated Swagger embedding | [assets/swagger.json](../../../assets/swagger.json), [assets/embed.go](../../../assets/embed.go), [util/assets/assets.go](../../../util/assets/assets.go) | `assets.Embedded`, `SwaggerJSON` |
| Swagger HTTP delivery | [util/swagger/swagger.go](../../../util/swagger/swagger.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `ServeSwaggerUI`, `newHTTPServer` |
| Authentication and module enforcement | [internal/server/authz.go](../../../internal/server/authz.go), [internal/server/version/version.go](../../../internal/server/version/version.go), [internal/server/session/session.go](../../../internal/server/session/session.go), [internal/server/appbootstrap/appbootstrap.go](../../../internal/server/appbootstrap/appbootstrap.go) | `publicGRPCMethods`, `moduleGRPCRules`, `AuthFuncOverride` |

## Architecture

```mermaid
flowchart LR
    D["ui/src/assets public documents"] --> V["Vite publicDir copy"]
    V --> U["ui/dist/app"]
    U --> UE["ui.Embedded"]
    UE --> S["Athena static-file handler"]
    S --> L["/llms.txt and /docs/ai/*.md"]

    P["Server proto HTTP annotations"] --> G["generate-proto.sh"]
    G --> J["assets/swagger.json"]
    J --> AE["assets.Embedded / SwaggerJSON"]
    AE --> W["ServeSwaggerUI"]
    W --> O["/swagger.json and /swagger-ui"]
```

The curated documents and the Swagger contract use separate build and embed
paths. Vite treats `ui/src/assets` as its public directory and copies its files
without bundling them into `ui/dist/app`. `ui/embed.go` embeds that complete
output tree. At startup, `AthenaServer` creates a static filesystem from the
embedded tree and optionally composes it with the configured additional static
asset directory. The embedded filesystem is checked first, so the additional
directory supplies missing files rather than overriding embedded ones.

Swagger is not part of the Vite public directory. Protobuf HTTP annotations
under `internal/server/` define paths and wire schemas. `collect_swagger` mixes
the generated per-service specifications with Athena's top-level metadata and
security declaration, normalizes the result, and writes `assets/swagger.json`.
The root `assets` package embeds that generated file; `util/assets` loads it as
`SwaggerJSON`; and `ServeSwaggerUI` registers both the JSON response and ReDoc
handler on the HTTP mux.

All discovery routes are readable without authentication. The Swagger global
Bearer declaration documents the normal API requirement, while these optional-
authentication operations explicitly set an empty operation-level security
list:

- `GET /api/version`
- `GET /api/v1/session/userinfo`
- `GET /api/v1/app/bootstrap`

The gRPC authentication overrides remain authoritative for those operations.
For every protected operation, the server's authentication and authorization
interceptors remain authoritative regardless of how Swagger or the Markdown
guidance describes the request.

## Runtime Flow

1. Maintainers update the curated sources in `ui/src/assets` when public API
   capabilities, access guidance, or safety boundaries change.
2. The UI build copies those sources to the same relative locations in
   `ui/dist/app`; the Go build embeds that output into `ui.Embedded`.
3. Protobuf generation creates per-service Swagger inputs. `collect_swagger`
   combines them, applies Athena metadata and security semantics, and replaces
   the generated `assets/swagger.json`; the Go build embeds it separately.
4. `NewServer` opens the embedded UI subtree and adds the optional external
   static filesystem. `newHTTPServer` registers API, Swagger, health, download,
   and static handlers on one HTTP mux.
5. A request for an existing `/llms.txt` or `/docs/ai/*.md` path is served as a
   file by `newStaticAssetsHandler`. A request for `/swagger.json` or
   `/swagger-ui` is served by `ServeSwaggerUI` before the catch-all static
   handler.
6. When `RootPath` is non-empty, `withRootPath` mounts the complete handler below
   that prefix and strips the prefix before dispatch. No separate origin-root
   discovery handler is registered.

There is no background work, shutdown action, transaction, or mutable runtime
state associated with discovery documents.

## State / Data

The Markdown files, `llms.txt`, and Swagger JSON are immutable build inputs or
outputs. They do not contain per-account data, credentials, runtime service
state, or a complete copy of internal design documentation.

`llms.txt` is a concise navigation document rather than a duplicated endpoint
catalog. The focused Markdown pages provide stable conceptual guidance, while
`/swagger.json` is the machine-readable source for concrete paths, methods,
parameters, responses, and schemas. Executable proto and server code remain the
ultimate source of truth when public prose or a generated artifact is stale.

## Configuration

| Setting | Behavior |
| --- | --- |
| Vite `publicDir: '../assets'` | Copies `ui/src/assets` into the UI output root. |
| Vite `build.outDir: '../../dist/app'` | Produces the tree embedded by `ui/embed.go`. |
| `ATHENA_SERVER_STATIC_ASSETS` / `--staticassets` | Adds a filesystem after the embedded UI assets; the default is `/shared/app`, and embedded files take precedence. |
| `ATHENA_SERVER_ROOTPATH` / `--rootpath` | Mounts the HTTP handler below a non-empty proxy prefix; the default is empty. |
| `ATHENA_SERVER_BASEHREF` / `--basehref` | Rewrites the SPA's `index.html` base element only; it does not rewrite links inside `llms.txt` or Markdown files. |

Public AI documents deliberately use origin-root-relative links. The supported
discovery layout therefore assumes Athena is deployed at the domain root. With
a non-empty `ATHENA_SERVER_ROOTPATH`, a caller may reach the files below the
configured prefix, but Athena does not guarantee an origin-root `/llms.txt`, and
the root-relative links inside the documents are not rewritten for that prefix.

## Invariants

- `ui/src/assets` is the source for public LLM-oriented prose; internal
  `docs/design/` content is never copied into the public tree.
- `assets/swagger.json` is generated from proto annotations and
  `hack/generate-proto.sh`; it is never hand-edited.
- Protected operations inherit the global `athenaBearer` Swagger requirement;
  only the three optional-authentication operations listed above declare
  `security: []`.
- Swagger security metadata documents server behavior but cannot grant or
  weaken access. Authentication, API Key entitlement, administrator checks,
  Profit Sharing entitlement, and module requirements remain server decisions.
- Public documents contain no bearer values, wallet secrets, external-provider
  credentials, internal implementation details, or claims that unavailable
  MCP, OAuth, scoped-key, or AI-write capabilities exist.
- Existing API Keys are described only for trusted CLI and self-hosted
  automation. They inherit the account's current access and are not presented
  as credentials to hand to third-party AI services.
- Origin-root-relative discovery links assume an empty server root path.

## Failure Recovery

A missing source file, omitted Vite build, or stale Go binary leaves the
corresponding public document unavailable or stale until the artifacts and
binary are rebuilt. The static handler returns the normal HTTP file-server
response; it has no fallback discovery document. Because embedded files take
precedence, placing a same-named file in the additional static directory does
not repair or override an embedded document.

Swagger generation replaces the generated artifact from its sources. A failed
generation must be corrected in proto annotations or the generator rather than
patched in `assets/swagger.json`. Documentation drift never changes runtime
authorization: server interceptors continue to allow or deny the request from
the current credential and access state.

## Observability

Discovery delivery adds no dedicated metrics, health state, or tracing. Normal
HTTP status and access logging diagnose availability. Vite build output confirms
the static copy, and protobuf generation output plus the generated Swagger diff
diagnose contract production. Operators can inspect `/llms.txt`, each linked
Markdown path, `/swagger.json`, and `/swagger-ui` on the deployed origin.

## Change Checklist

- [ ] Curated public documents and their links match the current API and safety boundaries.
- [ ] The Vite-to-UI-embed and Swagger-to-assets-embed delivery paths remain current.
- [ ] Swagger metadata, global Bearer security, and public operation exceptions match server behavior.
- [ ] Authentication and authorization guidance matches the credential and module implementation.
- [ ] Root-path constraints and static-asset precedence remain current.
- [ ] Source links resolve, and the [design index](../README.md) contains the current summary.
