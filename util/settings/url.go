package settings

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/useryege/athena/common"
)

// ValidateExternalURL ensures the external URL that is set on the configmap is valid
func ValidateExternalURL(u string) error {
	if u == "" {
		return nil
	}

	URL, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	if URL.Scheme != "http" && URL.Scheme != "https" {
		return errors.New("URL must include http or https protocol")
	}

	return nil
}

func splitCommaSeparated(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

func appendURLPath(inputURL string, inputPath string) (string, error) {
	u, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}

	u.Path = path.Join(u.Path, inputPath)

	return u.String(), nil
}

func (a *AthenaSettings) RedirectURL() (string, error) {
	return appendURLPath(a.URL, common.CallbackEndpoint)
}

func (a *AthenaSettings) AthenaURLForRequest(r *http.Request) (string, error) {
	for _, candidateURL := range append([]string{a.URL}, a.AdditionalURLs...) {
		u, err := url.Parse(candidateURL)
		if err != nil {
			return "", err
		}
		if u.Host == r.Host && strings.HasPrefix(r.URL.RequestURI(), u.RequestURI()) {
			return candidateURL, nil
		}
	}

	return a.URL, nil
}

func (a *AthenaSettings) RedirectURLForRequest(r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("request is nil")
	}

	base, err := a.AthenaURLForRequest(r)
	if err != nil {
		return "", err
	}

	return appendURLPath(base, common.CallbackEndpoint)
}

func (a *AthenaSettings) RedirectAdditionalURLs() ([]string, error) {
	RedirectAdditionalURLs := []string{}
	for _, url := range a.AdditionalURLs {
		redirectURL, err := appendURLPath(url, common.CallbackEndpoint)
		if err != nil {
			return []string{}, err
		}

		RedirectAdditionalURLs = append(RedirectAdditionalURLs, redirectURL)
	}

	return RedirectAdditionalURLs, nil
}

func (a *AthenaSettings) DexRedirectURL() (string, error) {
	return appendURLPath(a.URL, common.DexCallbackEndpoint)
}
