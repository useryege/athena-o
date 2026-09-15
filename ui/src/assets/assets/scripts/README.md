# Local API documentation client

`redoc.standalone.js` is the unmodified ReDoc 2.4.0 browser bundle, distributed under the accompanying MIT license. Source: https://registry.yarnpkg.com/redoc/-/redoc-2.4.0.tgz (SHA-512 integrity is pinned in `ui/scripts/vendor-redoc.py`).

Run `python3 ui/scripts/vendor-redoc.py` from the repository root to reproduce the two vendored files. It verifies the package and each extracted file before writing. Normal offline UI builds copy this directory through Vite's public directory into `ui/dist/app/assets/scripts`, which the Go static server embeds/serves. The client script itself needs no CDN, Argo package or additional runtime package. The unmodified client still requests Google Fonts and the Redocly logo from external origins; this vendoring does not make the documentation page fully offline. `/swagger-ui` references the deployment-prefixed local bundle; the documentation client is separate from the application's approved theme.
