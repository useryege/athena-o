//go:build integration && uiharness

package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraderSyncUIHarnessAssets(t *testing.T) {
	dist := os.Getenv("ATHENA_UI_DIST")
	if dist == "" {
		t.Fatal("ATHENA_UI_DIST is required")
	}
	for _, prefix := range []string{"", "/athena"} {
		adapter, err := NewTraderSyncUIHarnessAdapter(nil, nil, nil, dist, prefix)
		if err != nil {
			t.Fatalf("valid newly built assets must assemble: %v", err)
		}
		for _, path := range []string{"/trader-sync", "/admin/trader-sync/subscriptions"} {
			req := httptest.NewRequest("GET", path, nil)
			req.Header.Set("Accept", "text/html")
			rec := httptest.NewRecorder()
			adapter.Static.ServeHTTP(rec, req)
			base := prefix + "/"
			if strings.HasPrefix(path, "/admin") {
				base += "admin/"
			}
			if rec.Code != 200 || !strings.Contains(rec.Body.String(), `<base href="`+base+`">`) || !strings.Contains(rec.Body.String(), `<meta name="athena-deployment-base-href" content="`+prefix+`/">`) {
				t.Fatalf("wrong real HTML base: %d %s", rec.Code, rec.Body.String())
			}
		}
	}
	bad := t.TempDir()
	if err := os.WriteFile(filepath.Join(bad, "index.html"), []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTraderSyncUIHarnessAdapter(nil, nil, nil, bad, ""); err == nil {
		t.Fatal("stale/incomplete assets accepted")
	}
}
