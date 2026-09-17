package devruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func deploymentBase(value string) string {
	return "/" + strings.Trim(strings.TrimSpace(value), "/") + "/"
}
func normalizedBase(value string) string {
	if strings.Trim(strings.TrimSpace(value), "/") == "" {
		return "/"
	}
	return deploymentBase(value)
}

func probeHTTP(ctx context.Context, name, address string, env map[string]string) error {
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	origin := "http://" + address
	base := normalizedBase(env["ATHENA_SERVER_BASEHREF"])
	if name == "api-server" {
		base = normalizedBase(env["ATHENA_SERVER_ROOTPATH"])
		if _, _, err := readHTTP(ctx, client, origin+base+"healthz", "", false); err != nil {
			return err
		}
	} else {
		for _, suffix := range []string{"", "admin/"} {
			if err := probePage(ctx, client, origin, base, base+suffix); err != nil {
				return err
			}
		}
	}
	for _, realm := range []string{"member", "admin"} {
		body, _, err := readHTTP(ctx, client, origin+base+"api/v1/app/bootstrap", realm, false)
		if err != nil {
			return err
		}
		var response struct {
			Session struct {
				Status   string
				UserInfo *struct {
					LoggedIn      bool
					AccountID     string `json:"accountId"`
					Administrator bool
				} `json:"user_info"`
			}
		}
		if err = json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("%s bootstrap: %w", realm, err)
		}
		user := response.Session.UserInfo
		switch response.Session.Status {
		case "APP_BOOTSTRAP_SESSION_STATUS_ANONYMOUS":
			if disabled, _ := strconv.ParseBool(env["ATHENA_SERVER_DISABLE_AUTH"]); disabled {
				return errors.New("development authentication bootstrap is unexpectedly anonymous")
			}
			if user != nil && (user.LoggedIn || user.AccountID != "" || user.Administrator) {
				return errors.New("anonymous bootstrap includes authenticated identity")
			}
		case "APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED":
			if user == nil || !user.LoggedIn || user.AccountID == "" || user.Administrator != (realm == "admin") {
				return fmt.Errorf("%s bootstrap has invalid realm identity", realm)
			}
		default:
			return fmt.Errorf("%s bootstrap session is not usable: %s", realm, response.Session.Status)
		}
	}
	return nil
}
func readHTTP(ctx context.Context, client *http.Client, target, realm string, navigation bool) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	if realm != "" {
		req.Header.Set("X-Athena-Application-Realm", realm)
	}
	if navigation {
		req.Header.Set("Accept", "text/html")
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP readiness %s: %s", req.URL.Path, response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	return body, response.Header.Get("Content-Type"), nil
}
func probePage(ctx context.Context, client *http.Client, origin, base, page string) error {
	body, _, err := readHTTP(ctx, client, origin+page, "", true)
	if err != nil {
		return err
	}
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return err
	}
	var pageBase, deployment string
	var resources []string
	scripts := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		attrs := map[string]string{}
		for _, attr := range node.Attr {
			attrs[attr.Key] = attr.Val
		}
		if node.Type == html.ElementNode {
			switch node.Data {
			case "base":
				pageBase = attrs["href"]
			case "meta":
				if attrs["name"] == "athena-deployment-base-href" {
					deployment = attrs["content"]
				}
			case "script":
				if attrs["src"] != "" {
					resources = append(resources, attrs["src"])
					scripts++
				}
			case "link":
				if attrs["rel"] == "stylesheet" || attrs["rel"] == "modulepreload" {
					resources = append(resources, attrs["href"])
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	if pageBase != page || deployment != base || scripts == 0 {
		return fmt.Errorf("UI %s has invalid application/deployment base or missing entry", page)
	}
	originURL, _ := url.Parse(origin + page)
	for _, resource := range resources {
		reference, e := url.Parse(resource)
		if e != nil {
			return e
		}
		target := originURL.ResolveReference(reference)
		if target.Host != originURL.Host || target.Scheme != originURL.Scheme {
			return errors.New("UI readiness resource is outside this instance")
		}
		content, kind, e := readHTTP(ctx, client, target.String(), "", false)
		if e != nil {
			return e
		}
		if len(bytes.TrimSpace(content)) == 0 || strings.Contains(strings.ToLower(kind), "text/html") || bytes.HasPrefix(bytes.ToLower(bytes.TrimSpace(content)), []byte("<!doctype html")) || bytes.HasPrefix(bytes.ToLower(bytes.TrimSpace(content)), []byte("<html")) {
			return fmt.Errorf("UI resource %s returned empty content or HTML fallback", target.Path)
		}
	}
	return nil
}
