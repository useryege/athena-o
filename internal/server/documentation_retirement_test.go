package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetiredDocumentationPathsReturnNotFoundBeforeStaticFilesAndHistoryFallback(t *testing.T) {
	staticDir := t.TempDir()
	files := map[string]string{
		"swagger-ui/index.html":              "legacy swagger ui",
		"swagger.json":                       `{"swagger":"2.0"}`,
		"llms.txt":                           "legacy discovery",
		"docs/ai/overview.md":                "legacy ai guide",
		"assets/scripts/redoc.standalone.js": "legacy redoc",
		"assets/scripts/redoc-LICENSE.txt":   "legacy license",
		"assets/scripts/README.md":           "legacy readme",
		"assets/kept.js":                     "kept asset",
	}
	for name, contents := range files {
		path := filepath.Join(staticDir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	}

	retiredPaths := []string{
		"/swagger-ui",
		"/swagger-ui/index.html",
		"/swagger.json",
		"/llms.txt",
		"/docs/ai",
		"/docs/ai/overview.md",
		"/assets/scripts/redoc.standalone.js",
		"/assets/scripts/redoc-LICENSE.txt",
		"/assets/scripts/README.md",
	}
	accepts := []string{"application/json", "text/html"}

	for _, prefix := range []string{"", "/athena"} {
		t.Run("prefix="+prefix, func(t *testing.T) {
			server := &AthenaServer{
				AthenaServerOpts: AthenaServerOpts{RootPath: prefix, BaseHRef: prefix},
				staticAssets:     http.FS(os.DirFS(staticDir)),
			}
			handler := http.Handler(http.HandlerFunc(server.newStaticAssetsHandler()))
			if prefix != "" {
				handler = withRootPath(handler, server)
			}

			for _, retiredPath := range retiredPaths {
				for _, method := range []string{http.MethodGet, http.MethodHead} {
					for _, accept := range accepts {
						name := method + " " + retiredPath + " " + accept
						t.Run(name, func(t *testing.T) {
							req := httptest.NewRequest(method, prefix+retiredPath, nil)
							req.Header.Set("Accept", accept)
							response := httptest.NewRecorder()
							handler.ServeHTTP(response, req)
							require.Equal(t, http.StatusNotFound, response.Code)
							require.Empty(t, response.Header().Get("Location"))
							if method == http.MethodGet {
								require.NotContains(t, response.Body.String(), "legacy")
							}
						})
					}
				}
			}

			for _, path := range []string{"/assets/kept.js", "/swagger-ui-guide", "/docs/aide"} {
				req := httptest.NewRequest(http.MethodGet, prefix+path, nil)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, req)
				if path == "/assets/kept.js" {
					require.Equal(t, http.StatusOK, response.Code)
					require.Equal(t, "kept asset", response.Body.String())
				} else {
					require.Equal(t, http.StatusNotFound, response.Code)
				}
			}
		})
	}
}
