package polymarket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/net/html"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ProfileAdapterVersion = "polymarket-profile-rsc-2026-09-10-v1"
const profileBodyLimit = 2 * 1024 * 1024

var profileHandle = regexp.MustCompile(`^/@[A-Za-z0-9_-]+$`)

// ProfileAdapter keeps production origins fixed; only the HTTP transport is injectable.
type ProfileAdapter struct{ http *http.Client }
type ProfileIdentity struct {
	ResolutionInput string
	PublicSource    string
	Wallet          common.Address
	CanonicalURL    string
	Public          PublicProfile
	Stats           ProfileStats
	PositionValue   Decimal
	QueriedAt       time.Time
}

func NewProfileAdapter(transport http.RoundTripper) *ProfileAdapter {
	return &ProfileAdapter{http: &http.Client{Transport: boundedProfileTransport{next: transport}, Timeout: 5 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return fmt.Errorf("too many redirects")
		}
		if len(via) == 0 {
			return fmt.Errorf("missing redirect origin")
		}
		if via[0].URL.Host == "polymarket.com" || via[0].URL.Host == "www.polymarket.com" {
			return validateProfileURL(req.URL)
		}
		return fmt.Errorf("API redirects unsupported")
	}}}
}

func validateProfileURL(u *url.URL) error {
	if u.Scheme != "https" || (u.Host != "polymarket.com" && u.Host != "www.polymarket.com") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || u.RawPath != "" || !profileHandle.MatchString(u.Path) {
		return fmt.Errorf("expected an exact Polymarket HTTPS /@handle URL")
	}
	return nil
}

func (a *ProfileAdapter) get(ctx context.Context, source string) ([]byte, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if e != nil {
		return nil, e
	}
	resp, e := a.http.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeHTTPErrorWithService(resp, "profile")
	}
	raw, e := io.ReadAll(io.LimitReader(resp.Body, profileBodyLimit+1))
	if e != nil {
		return nil, e
	}
	if len(raw) > profileBodyLimit {
		return nil, fmt.Errorf("response exceeds 2 MiB")
	}
	return raw, nil
}

type boundedProfileTransport struct{ next http.RoundTripper }

func (t boundedProfileTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}
	resp, e := next.RoundTrip(req)
	if e != nil {
		return nil, e
	}
	raw, e := io.ReadAll(io.LimitReader(resp.Body, profileBodyLimit+1))
	resp.Body.Close()
	if e != nil {
		return nil, e
	}
	if len(raw) > profileBodyLimit {
		return nil, fmt.Errorf("response exceeds 2 MiB")
	}
	resp.Body = io.NopCloser(bytes.NewReader(raw))
	return resp, nil
}

func (a *ProfileAdapter) publicProfile(ctx context.Context, address string) (PublicProfile, error) {
	gamma, e := NewGammaClient(GammaConfig{HTTPClient: a.http})
	if e != nil {
		return PublicProfile{}, e
	}
	p, e := gamma.GetPublicProfile(ctx, strings.ToLower(address))
	if e != nil {
		return PublicProfile{}, e
	}
	if p == nil {
		return PublicProfile{}, fmt.Errorf("missing public profile")
	}
	return *p, nil
}

func (a *ProfileAdapter) ResolveIdentity(ctx context.Context, input string) (ProfileIdentity, error) {
	var out ProfileIdentity
	input = strings.TrimSpace(input)
	if common.IsHexAddress(input) && len(input) == 42 && strings.HasPrefix(input, "0x") {
		p, e := a.publicProfile(ctx, input)
		if e != nil {
			return out, status.Errorf(codes.FailedPrecondition, "public identity: %v", e)
		}
		if p.ProxyWallet == nil || !common.IsHexAddress(*p.ProxyWallet) || common.HexToAddress(*p.ProxyWallet) == (common.Address{}) {
			return out, status.Error(codes.FailedPrecondition, "public profile has no unique wallet")
		}
		out.ResolutionInput = strings.ToLower(input)
		out.Wallet = common.HexToAddress(*p.ProxyWallet)
		out.Public = p
		out.PublicSource = DefaultGammaBaseURL + "/public-profile?" + url.Values{"address": {strings.ToLower(input)}}.Encode()
		out.QueriedAt = time.Now().UTC()
		return out, nil
	}
	u, e := url.Parse(input)
	if e != nil || validateProfileURL(u) != nil {
		return out, status.Error(codes.InvalidArgument, "invalid profile URL or address")
	}
	raw, e := a.get(ctx, u.String())
	if e != nil {
		return out, status.Errorf(codes.FailedPrecondition, "profile request: %v", e)
	}
	out, e = parseProfileSSR(raw, u)
	if e != nil {
		return out, status.Errorf(codes.FailedPrecondition, "profile structure: %v", e)
	}
	p, e := a.publicProfile(ctx, out.Wallet.Hex())
	if e != nil {
		return out, status.Errorf(codes.FailedPrecondition, "public identity: %v", e)
	}
	if p.ProxyWallet == nil || !common.IsHexAddress(*p.ProxyWallet) || common.HexToAddress(*p.ProxyWallet) != out.Wallet {
		return out, status.Error(codes.FailedPrecondition, "SSR and public profile wallet conflict")
	}
	out.ResolutionInput = u.String()
	out.Public = p
	out.PublicSource = DefaultGammaBaseURL + "/public-profile?" + url.Values{"address": {strings.ToLower(out.Wallet.Hex())}}.Encode()
	out.QueriedAt = time.Now().UTC()
	return out, nil
}

type profileQuery struct {
	Key   []string `json:"queryKey"`
	State struct {
		Data   json.RawMessage `json:"data"`
		Status string          `json:"status"`
	} `json:"state"`
}

func parseProfileSSR(raw []byte, requested *url.URL) (ProfileIdentity, error) {
	var out ProfileIdentity
	z := html.NewTokenizer(bytes.NewReader(raw))
	canonicals := []string{}
	var flight strings.Builder
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() != io.EOF {
				return out, z.Err()
			}
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		t := z.Token()
		if t.Data == "link" {
			var rel, href string
			for _, a := range t.Attr {
				if a.Key == "rel" {
					rel = a.Val
				}
				if a.Key == "href" {
					href = a.Val
				}
			}
			if rel == "canonical" {
				canonicals = append(canonicals, href)
			}
		}
		if t.Data == "script" && z.Next() == html.TextToken {
			s := string(z.Text())
			const prefix = "self.__next_f.push("
			if strings.HasPrefix(s, prefix) && strings.HasSuffix(s, ")") {
				var push []json.RawMessage
				if json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimPrefix(s, prefix), ")")), &push) == nil && len(push) == 2 && string(push[0]) == "1" {
					var chunk string
					if json.Unmarshal(push[1], &chunk) == nil {
						flight.WriteString(chunk)
					}
				}
			}
		}
	}
	if len(canonicals) != 1 {
		return out, fmt.Errorf("expected one canonical URL")
	}
	canonical, e := url.Parse(canonicals[0])
	if e != nil || validateProfileURL(canonical) != nil || !strings.EqualFold(canonical.Path, requested.Path) {
		return out, fmt.Errorf("canonical identity conflict")
	}
	canonical.Host = "polymarket.com"
	canonical.Path = strings.ToLower(canonical.Path)
	out.CanonicalURL = canonical.String()
	var queries []profileQuery
	for _, line := range strings.Split(flight.String(), "\n") {
		_, payload, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		var node []json.RawMessage
		if json.Unmarshal([]byte(payload), &node) != nil || len(node) != 4 {
			continue
		}
		var props struct {
			State struct {
				Queries []profileQuery `json:"queries"`
			} `json:"state"`
		}
		if json.Unmarshal(node[3], &props) == nil {
			queries = append(queries, props.State.Queries...)
		}
	}
	wallets := map[common.Address]bool{}
	identities := 0
	for _, q := range queries {
		if len(q.Key) < 2 || (q.Key[0] != "/api/profile/userData" && q.Key[0] != "/api/profile/volume") {
			continue
		}
		if q.State.Status != "success" {
			return out, fmt.Errorf("identity query unavailable")
		}
		var p PublicProfile
		if decodeObject(q.State.Data, &p) != nil || p.ProxyWallet == nil || !common.IsHexAddress(*p.ProxyWallet) || !common.IsHexAddress(q.Key[1]) {
			return out, fmt.Errorf("identity schema changed")
		}
		wallets[common.HexToAddress(*p.ProxyWallet)] = true
		wallets[common.HexToAddress(q.Key[1])] = true
		for _, key := range q.Key[2:] {
			if common.IsHexAddress(key) {
				wallets[common.HexToAddress(key)] = true
			}
		}
		if q.Key[0] == "/api/profile/userData" {
			identities++
		}
	}
	if len(wallets) != 1 || identities != 1 {
		return out, fmt.Errorf("expected one SSR identity")
	}
	for wallet := range wallets {
		out.Wallet = wallet
	}
	if out.Wallet == (common.Address{}) {
		return out, fmt.Errorf("empty wallet")
	}
	for _, q := range queries {
		if q.State.Status != "success" {
			continue
		}
		if len(q.Key) == 2 && q.Key[0] == "user-stats" && strings.EqualFold(q.Key[1], out.Wallet.Hex()) {
			_ = json.Unmarshal(q.State.Data, &out.Stats)
		}
		if len(q.Key) == 3 && q.Key[0] == "positions" && q.Key[1] == "value" && strings.EqualFold(q.Key[2], out.Wallet.Hex()) {
			_ = json.Unmarshal(q.State.Data, &out.PositionValue)
		}
	}
	return out, nil
}
