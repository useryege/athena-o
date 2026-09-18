package devruntime

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestOperationLogReadinessUsesConfiguredTLS(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, IsCA: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	caPath := filepath.Join(t.TempDir(), "operation-log-ca.pem")
	if err := os.WriteFile(caPath, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}})))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go server.Serve(listener)
	defer func() { server.Stop(); listener.Close() }()
	env := map[string]string{"ATHENA_OPERATION_LOG_GRPC_TRANSPORT": "tls", "ATHENA_OPERATION_LOG_TLS_CA_FILE": caPath, "ATHENA_OPERATION_LOG_TLS_SERVER_NAME": "localhost"}
	if err := probeService(context.Background(), "operation-log", listener.Addr().String(), env); err != nil {
		t.Fatal(err)
	}
}

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
