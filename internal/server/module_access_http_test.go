package server

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/googleoidc"
	"github.com/useryege/athena/internal/moduleaccess"
	"github.com/useryege/athena/internal/phantomauth"
	walletclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/walletsecret"
	"go/ast"
	"go/parser"
	"go/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type admissionWalletClient struct{ walletclient.Clientset }

func TestModuleAccessAllTradingHTTPRegistrationsAndCommands(t *testing.T) {
	s, _ := newCatalogHTTPServer(t)
	s.moduleAccessStore = &moduleAdmissionStore{}
	s.WalletClientset = admissionWalletClient{}
	s.wormCredentialMgr = &walletsecret.Manager{}
	mux := http.NewServeMux()
	registerWormWalletSelectionHandlers(mux, s)
	registerWormConnectionHandlers(mux, s)
	registerWormCombinationHandlers(mux, s)
	registerWormExecutionPlanHandlers(mux, s)
	registerWormExecutionHandlers(mux, s)
	registerWormPositionCashOutHandlers(mux, s)
	registerWormPositionCashOutBatchHandlers(mux, s)
	const id = "22222222-2222-4222-8222-222222222222"
	paths := []struct{ method, path string }{
		{"GET", "/wallet-selection"}, {"PUT", "/wallet-selection"}, {"GET", "/wallet-connections"}, {"POST", "/wallet-connections/1"}, {"POST", "/wallet-connections/1:reconnect"}, {"POST", "/wallet-connections/1:regenerate"}, {"DELETE", "/wallet-connections/1"},
		{"GET", "/events/" + httpCatalogEventID}, {"GET", "/combinations"}, {"POST", "/combinations"}, {"GET", "/combinations/" + id}, {"PUT", "/combinations/" + id}, {"DELETE", "/combinations/" + id},
		{"POST", "/execution-plans"}, {"GET", "/execution-plans/" + id}, {"GET", "/execution-plans/" + id + "/steps"},
		{"GET", "/executions"}, {"POST", "/executions"}, {"GET", "/executions/" + id}, {"GET", "/executions/" + id + "/steps"},
		{"POST", "/executions/" + id + ":start"}, {"POST", "/executions/" + id + ":pause"}, {"POST", "/executions/" + id + ":continue"}, {"POST", "/executions/" + id + ":terminate"}, {"POST", "/executions/" + id + ":heartbeat"}, {"POST", "/executions/" + id + ":execute-next"}, {"POST", "/executions/" + id + "/steps/" + id + ":reconcile"},
		{"POST", "/position-cash-outs"}, {"GET", "/position-cash-outs/" + id}, {"POST", "/position-cash-outs/" + id + ":reconcile"},
		{"POST", "/position-cash-out-batches"}, {"GET", "/position-cash-out-batches/active"}, {"GET", "/position-cash-out-batches/" + id}, {"GET", "/position-cash-out-batches/" + id + "/items"},
		{"POST", "/position-cash-out-batches/" + id + ":cancel"}, {"POST", "/position-cash-out-batches/" + id + ":pause"}, {"POST", "/position-cash-out-batches/" + id + ":continue"}, {"POST", "/position-cash-out-batches/" + id + ":terminate"}, {"POST", "/position-cash-out-batches/" + id + ":check-status"},
	}
	registrations := map[string]bool{}
	for _, entry := range paths {
		t.Run(entry.method+entry.path, func(t *testing.T) {
			r := catalogHTTPRequest(entry.method, "/api/v1/worm-trading"+entry.path, "{}")
			_, pattern := mux.Handler(r)
			require.NotEmpty(t, pattern)
			registrations[pattern] = true
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			require.Equal(t, 503, w.Code, w.Body.String())
			require.Equal(t, moduleaccess.ClosedReason, w.Header().Get("X-Athena-Error-Reason"))
			require.Contains(t, w.Body.String(), "athena.module_access")
		})
	}
	require.Len(t, registrations, 28)
	// Count the registration source too: a newly added route must not escape the
	// behavioral table merely because the old paths still match the mux.
	files, err := filepath.Glob("worm_*.go")
	require.NoError(t, err)
	actual := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(function.Name.Name, "registerWorm") || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Handle" {
					return true
				}
				receiver, ok := selector.X.(*ast.Ident)
				if ok && receiver.Name == "mux" {
					actual++
				}
				return true
			})
		}
	}
	require.Equal(t, len(registrations), actual, "new Trading HTTP registration needs behavioral admission coverage")
}
func TestModuleAccessDevelopmentVerificationEntrypoints(t *testing.T) {
	s, _ := newCatalogHTTPServer(t)
	s.moduleAccessStore = &moduleAdmissionStore{}
	for name, handler := range map[string]http.HandlerFunc{"credentials": s.developmentWormCredentialLease, "executions": s.developmentWormExecutionAuthorization, "cash_out": s.developmentWormPositionCashOutAuthorization, "batch": s.developmentWormPositionCashOutBatchAuthorization} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler(w, catalogHTTPRequest("POST", "/auth/worm-trading/development", "{}"))
			require.Equal(t, 503, w.Code, w.Body.String())
			require.Equal(t, moduleaccess.ClosedReason, w.Header().Get("X-Athena-Error-Reason"))
		})
	}
}

func TestModuleAccessVerificationRegistrationsMatchActualServer(t *testing.T) {
	s, _ := newCatalogHTTPServer(t)
	s.googleOIDC = &googleoidc.Handler{}
	s.phantomAuth = &phantomauth.Handler{}
	s.StaticAssetsDir = t.TempDir()
	conn, err := grpc.NewClient("127.0.0.1:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	switcher := s.newHTTPServer(context.Background(), 0, http.NotFoundHandler(), conn).Handler.(*handlerSwitcher)
	public := []string{"/auth/worm-trading/google", "/auth/worm-trading/executions/google", "/auth/worm-trading/position-cash-outs/google", "/auth/worm-trading/position-cash-out-batches/google", "/auth/worm-trading/solana/challenge", "/auth/worm-trading/solana/verify", "/auth/worm-trading/development"}
	registered := 0
	for path := range switcher.urlToHandler {
		if strings.HasPrefix(path, "/auth/worm-trading/") {
			registered++
		}
	}
	require.Equal(t, len(public), registered)
	for _, path := range public {
		require.Contains(t, switcher.urlToHandler, path)
	}
	mux := switcher.handler.(*http.ServeMux)
	for _, family := range []string{"executions", "position-cash-outs", "position-cash-out-batches"} {
		for _, suffix := range []string{"solana/challenge", "solana/verify", "development"} {
			r := httptest.NewRequest("POST", "http://localhost/auth/worm-trading/"+family+"/22222222-2222-4222-8222-222222222222/"+suffix, nil)
			_, pattern := mux.Handler(r)
			require.Contains(t, pattern, "POST /auth/worm-trading/"+family+"/")
			registered++
		}
	}
	require.Equal(t, 16, registered)
	// Inspect registrations, not just this test's expected routes, so a new raw
	// verification route requires an explicit admission coverage update.
	source, err := os.ReadFile("athena-server.go")
	require.NoError(t, err)
	literals := regexp.MustCompile(`"(?:POST )?/auth/worm-trading[^"]*"`).FindAll(source, -1)
	require.Len(t, literals, 16)
}
