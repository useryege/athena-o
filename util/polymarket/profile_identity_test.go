package polymarket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type routeTransport struct {
	base *url.URL
	next http.RoundTripper
}

func (r routeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c := req.Clone(req.Context())
	u := *req.URL
	u.Scheme, u.Host = r.base.Scheme, r.base.Host
	c.URL = &u
	return r.next.RoundTrip(c)
}

func localProfileAdapter(t *testing.T, h http.HandlerFunc) *ProfileAdapter {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	u, _ := url.Parse(s.URL)
	return NewProfileAdapter(routeTransport{u, s.Client().Transport})
}

func profileFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, e := os.ReadFile("testdata/profile/" + name)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestProfileIdentityExactSSRAndGamma(t *testing.T) {
	html, public := profileFixture(t, "profile.html"), profileFixture(t, "public-profile.json")
	a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/@GCR", "/@gcr":
			w.Write(html)
		case "/public-profile":
			if r.URL.Query().Get("address") != "0x5c52d767c32cc18100c72d7471209d9618216058" {
				t.Error(r.URL)
			}
			w.Write(public)
		default:
			t.Error(r.URL)
			w.WriteHeader(500)
		}
	})
	p, e := a.ResolveIdentity(context.Background(), "https://polymarket.com/@GCR")
	if e != nil || strings.ToLower(p.Wallet.Hex()) != "0x5c52d767c32cc18100c72d7471209d9618216058" || p.CanonicalURL != "https://polymarket.com/@gcr" {
		t.Fatal(p, e)
	}
	if p.PositionValue.Value == nil || *p.PositionValue.Value != "0" || p.Stats.JoinDate == nil || *p.Stats.JoinDate != "2024-06-19T21:35:45.083000Z" {
		t.Fatal(p)
	}
}

func TestProfileIdentityRejectsUnsafeAndAmbiguousInputs(t *testing.T) {
	for _, input := range []string{"GCR", "http://polymarket.com/@gcr", "https://evil.polymarket.com/@gcr", "https://polymarket.com.evil/@gcr", "https://x@polymarket.com/@gcr", "https://polymarket.com:443/@gcr", "https://polymarket.com/@gcr/extra", "https://polymarket.com/@gcr?user=x", "https://polymarket.com/@gcr#x", "https://polymarket.com/%40gcr", "0x1234"} {
		a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unsafe request escaped validation", r.URL) })
		if _, e := a.ResolveIdentity(context.Background(), input); status.Code(e) != codes.InvalidArgument {
			t.Fatal(input, e)
		}
	}
	html := string(profileFixture(t, "profile.html"))
	public := string(profileFixture(t, "public-profile.json"))
	for _, tt := range []struct{ name, h, p string }{{"canonical", strings.Replace(html, `rel="canonical" href="https://polymarket.com/@gcr"`, `rel="canonical" href="https://polymarket.com/@other"`, 1), public}, {"structure", "<html></html>", public}, {"wallet", html, strings.ReplaceAll(public, "0x5c52d767c32cc18100c72d7471209d9618216058", "0x1111111111111111111111111111111111111111")}, {"multiple", strings.Replace(html, `\"/api/profile/userData\",\"0x5c52d767c32cc18100c72d7471209d9618216058\"`, `\"/api/profile/userData\",\"0x1111111111111111111111111111111111111111\"`, 1), public}} {
		t.Run(tt.name, func(t *testing.T) {
			a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/public-profile" {
					w.Write([]byte(tt.p))
				} else {
					w.Write([]byte(tt.h))
				}
			})
			if _, e := a.ResolveIdentity(context.Background(), "https://polymarket.com/@gcr"); status.Code(e) != codes.FailedPrecondition {
				t.Fatal(e)
			}
		})
	}
}

func TestProfileRedirectSizeAndTimeout(t *testing.T) {
	for _, tt := range []struct {
		name string
		h    http.HandlerFunc
	}{{"cross-domain", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://example.org/@gcr", 302) }}, {"too-many", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://polymarket.com/@gcr", 302) }}, {"oversize", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(strings.Repeat("x", 2*1024*1024+1))) }}} {
		t.Run(tt.name, func(t *testing.T) {
			a := localProfileAdapter(t, tt.h)
			if _, e := a.ResolveIdentity(context.Background(), "https://polymarket.com/@gcr"); e == nil {
				t.Fatal("accepted")
			}
		})
	}
	a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, e := a.ResolveIdentity(ctx, "https://polymarket.com/@gcr"); e == nil {
		t.Fatal("timeout accepted")
	}
}

func TestProfileAddressMappingRecordsActualPublicQuery(t *testing.T) {
	input := "0x1111111111111111111111111111111111111111"
	a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("address") != input {
			t.Error(r.URL)
		}
		w.Write([]byte(`{"proxyWallet":"0x2222222222222222222222222222222222222222","name":"Mapped"}`))
	})
	p, e := a.ResolveIdentity(context.Background(), input)
	if e != nil || !strings.HasSuffix(p.PublicSource, "address="+input) {
		t.Fatal(p, e)
	}
}

func TestProfileThreeRedirectsAndBuiltInDeadline(t *testing.T) {
	html, public := profileFixture(t, "profile.html"), profileFixture(t, "public-profile.json")
	n := 0
	a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/public-profile" {
			w.Write(public)
			return
		}
		n++
		if n <= 3 {
			http.Redirect(w, r, "https://polymarket.com/@gcr", 302)
			return
		}
		w.Write(html)
	})
	if _, e := a.ResolveIdentity(context.Background(), "https://polymarket.com/@GCR"); e != nil {
		t.Fatal(e)
	}
	a = localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	start := time.Now()
	if _, e := a.ResolveIdentity(context.Background(), "https://polymarket.com/@gcr"); e == nil {
		t.Fatal("deadline absent")
	}
	if d := time.Since(start); d < 4*time.Second || d > 7*time.Second {
		t.Fatal(d)
	}
}

func TestProfileMalformedAuxiliaryFieldsDoNotEraseIdentity(t *testing.T) {
	a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"proxyWallet":"0x1111111111111111111111111111111111111111","name":123,"createdAt":"invalid-date","profileImage":45,"verifiedBadge":"wrong"}`))
	})
	p, e := a.ResolveIdentity(context.Background(), "0x1111111111111111111111111111111111111111")
	if e != nil || p.Public.Name != nil || p.Public.ProfileImage != nil || p.Public.VerifiedBadge != nil || p.Public.CreatedAt != nil {
		t.Fatal(p, e)
	}
}
