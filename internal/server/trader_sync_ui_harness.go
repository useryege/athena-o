//go:build integration && uiharness

package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/moduleaccess"
	"github.com/useryege/athena/ui"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc"
)

// TraderSyncUIHarnessAdapter exposes only existing authentication and transport
// boundaries. It exists exclusively in the explicitly selected UI test build.
type TraderSyncUIHarnessAdapter struct {
	Authenticator interface {
		Authenticate(context.Context) (context.Context, error)
	}
	Unary      grpc.UnaryServerInterceptor
	Gateway    *runtime.ServeMux
	API        http.Handler
	Static     http.Handler
	HTMLHashes map[string]string
}

func NewTraderSyncUIHarnessAdapter(credentials *accountcredentials.CredentialManager, sessions *session.SessionManager, access *accountaccess.Controller, moduleStore moduleaccess.Store, distDir, deploymentBase string) (*TraderSyncUIHarnessAdapter, error) {
	hashes := map[string]string{}
	for _, name := range []string{"index.html", "admin/index.html"} {
		input, err := os.ReadFile(filepath.Join(distDir, name))
		if err != nil {
			return nil, fmt.Errorf("read UI input %s: %w", name, err)
		}
		embedded, err := ui.Embedded.ReadFile("dist/app/" + name)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(input, embedded) {
			return nil, fmt.Errorf("UI input %s differs from compiled assets; build before compiling harness", name)
		}
		hashes["input/"+name] = fmt.Sprintf("%x", sha256.Sum256(input))
		hashes["embedded/"+name] = fmt.Sprintf("%x", sha256.Sum256(embedded))
	}
	server := &AthenaServer{AthenaServerOpts: AthenaServerOpts{BaseHRef: deploymentBase}, credentialMgr: credentials, sessionMgr: sessions, accessController: access, moduleAccessStore: moduleStore, staticAssets: http.FS(os.DirFS(distDir))}
	gateway := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, new(moduleAccessJSONMarshaler)), runtime.WithForwardResponseOption(server.translateGRPCResponseHeaders), runtime.WithIncomingHeaderMatcher(applicationRealmHeaderMatcher), runtime.WithOutgoingHeaderMatcher(moduleAccessOutgoingHeader))
	return &TraderSyncUIHarnessAdapter{Authenticator: server, Unary: server.unaryAuthInterceptor, Gateway: gateway, API: adaptApplicationRealmQuery(gateway), Static: http.HandlerFunc(server.newStaticAssetsHandler()), HTMLHashes: hashes}, nil
}
