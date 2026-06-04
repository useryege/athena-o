package settings

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
