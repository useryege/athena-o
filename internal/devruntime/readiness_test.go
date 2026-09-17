package devruntime

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

func TestHTTPReadinessRejectsInvalidBootstrapAndUI(t *testing.T) {
	for _, base := range []string{"/", "/athena/"} {
		for _, name := range []string{"api-server", "ui"} {
			for _, broken := range []string{"", "realm", "maintenance", "identity", "bootstrap", "html", "resource", "base"} {
				if name == "api-server" && (broken == "html" || broken == "resource" || broken == "base") {
					continue
				}
				t.Run(name+base+broken, func(t *testing.T) {
					realms := map[string]bool{}
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case base + "healthz":
							fmt.Fprint(w, "ok")
						case base + "api/v1/app/bootstrap":
							realm := r.Header.Get("X-Athena-Application-Realm")
							realms[realm] = true
							if broken == "bootstrap" {
								fmt.Fprint(w, "<html>fallback</html>")
								return
							}
							status := "AUTHENTICATED"
							if broken == "maintenance" {
								status = "ACCOUNT_MAINTENANCE"
							}
							id := "fixture-" + realm
							if broken == "identity" {
								id = ""
							}
							admin := realm == "admin"
							if broken == "realm" {
								admin = false
							}
							fmt.Fprintf(w, `{"settings":{},"session":{"status":"APP_BOOTSTRAP_SESSION_STATUS_%s","user_info":{"loggedIn":true,"accountId":%q,"administrator":%t}}}`, status, id, admin)
						case base, base + "admin/":
							if broken == "html" {
								fmt.Fprint(w, "<html>loading</html>")
								return
							}
							pageBase := r.URL.Path
							if broken == "base" {
								pageBase = "/wrong/"
							}
							fmt.Fprintf(w, `<html><head><base href="%s"><meta name="athena-deployment-base-href" content="%s"></head><body><script type="module" src="%sentry/app.js"></script></body></html>`, pageBase, base, base)
						case base + "entry/app.js":
							if broken == "resource" {
								w.Header().Set("Content-Type", "text/html")
								fmt.Fprint(w, "<html>fallback</html>")
								return
							}
							w.Header().Set("Content-Type", "application/javascript")
							fmt.Fprint(w, "export const app = true;")
						default:
							http.NotFound(w, r)
						}
					}))
					defer server.Close()
					err := probeService(context.Background(), name, strings.TrimPrefix(server.URL, "http://"), map[string]string{"ATHENA_SERVER_BASEHREF": base, "ATHENA_SERVER_ROOTPATH": base})
					if broken == "" {
						if err != nil {
							t.Fatal(err)
						}
						if !realms["member"] || !realms["admin"] {
							t.Fatal("did not verify both realms", realms)
						}
					} else if err == nil {
						t.Fatal("accepted invalid readiness", broken)
					}
				})
			}
		}
	}
}

func TestBootstrapAnonymousRequiresAuthenticationEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			fmt.Fprint(w, "ok")
			return
		}
		fmt.Fprint(w, `{"settings":{},"session":{"status":"APP_BOOTSTRAP_SESSION_STATUS_ANONYMOUS"}}`)
	}))
	defer server.Close()
	for _, disabled := range []string{"false", "true"} {
		err := probeService(context.Background(), "api-server", strings.TrimPrefix(server.URL, "http://"), map[string]string{"ATHENA_SERVER_DISABLE_AUTH": disabled})
		if (err == nil) != (disabled == "false") {
			t.Fatalf("disable_auth=%s err=%v", disabled, err)
		}
	}
}

func TestDeploymentUIProxyUsesBaseWhileDirectAPIUsesRoot(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			fmt.Fprint(w, "ok")
			return
		}
		if r.URL.Path != "/api/v1/app/bootstrap" {
			http.NotFound(w, r)
			return
		}
		realm := r.Header.Get("X-Athena-Application-Realm")
		fmt.Fprintf(w, `{"session":{"status":"APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED","user_info":{"loggedIn":true,"accountId":%q,"administrator":%t}}}`, realm+"-account", realm == "admin")
	}))
	defer api.Close()
	target, _ := url.Parse(api.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	ui := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/athena/api/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/athena")
			proxy.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/athena/app.js" {
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprint(w, "export const app=true;")
			return
		}
		fmt.Fprintf(w, `<base href="%s"><meta name="athena-deployment-base-href" content="/athena/"><script src="/athena/app.js"></script>`, r.URL.Path)
	}))
	defer ui.Close()
	env := map[string]string{"ATHENA_SERVER_BASEHREF": "/athena/"}
	for name, address := range map[string]string{"ui": ui.URL, "api-server": api.URL} {
		if err := probeService(context.Background(), name, strings.TrimPrefix(address, "http://"), env); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
