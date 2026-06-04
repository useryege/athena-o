package settings

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/useryege/athena/common"
	timeutil "github.com/useryege/athena/pkg/time"
	"github.com/useryege/athena/util/env"
)

func getDownloadBinaryUrlsFromEnv() map[string]string {
	binaryUrls := map[string]string{}
	for _, archType := range []string{"darwin-amd64", "darwin-arm64", "windows-amd64", "linux-amd64", "linux-arm64", "linux-ppc64le", "linux-s390x"} {
		envName := "ATHENA_HELP_DOWNLOAD_" + strings.ToUpper(strings.NewReplacer("-", "_").Replace(archType))
		if val := os.Getenv(envName); val != "" {
			binaryUrls[archType] = val
		}
	}

	return binaryUrls
}

func loadHelpFromEnv() Help {
	chatURL := os.Getenv("ATHENA_HELP_CHAT_URL")
	chatText := ""
	if chatURL != "" {
		chatText = env.StringFromEnv("ATHENA_HELP_CHAT_TEXT", "Chat now!")
	}

	return Help{
		ChatURL:    chatURL,
		ChatText:   chatText,
		BinaryURLs: getDownloadBinaryUrlsFromEnv(),
	}
}

func loadSettingsFromEnv(secrets map[string]string) (AthenaSettings, error) {
	serverSignature, err := envOrFile("ATHENA_JWT_SECRET")
	if err != nil {
		return AthenaSettings{}, err
	}

	settings := AthenaSettings{
		StatusBadgeEnabled:   env.ParseBoolFromEnv("ATHENA_STATUS_BADGE_ENABLED", false),
		StatusBadgeRootUrl:   os.Getenv("ATHENA_STATUS_BADGE_ROOT_URL"),
		AnonymousUserEnabled: env.ParseBoolFromEnv("ATHENA_ANONYMOUS_USER_ENABLED", false),
		UiCssURL:             os.Getenv("ATHENA_UI_CSS_URL"),
		UiBannerContent:      os.Getenv("ATHENA_UI_BANNER_CONTENT"),
		UiBannerPermanent:    env.ParseBoolFromEnv("ATHENA_UI_BANNER_PERMANENT", false),
		UiBannerPosition:     os.Getenv("ATHENA_UI_BANNER_POSITION"),
		BinaryUrls:           getDownloadBinaryUrlsFromEnv(),
		UiBannerURL:          os.Getenv("ATHENA_UI_BANNER_URL"),
		UserSessionDuration:  time.Hour * 24,
		PasswordPattern:      env.StringFromEnv("ATHENA_PASSWORD_PATTERN", common.PasswordPatten),
		ServerSignature:      []byte(serverSignature),
		Secrets:              secrets,
	}

	settings.URL = os.Getenv("ATHENA_URL")
	if err := ValidateExternalURL(settings.URL); err != nil {
		log.Warnf("Failed to validate URL in settings: %v", err)
	}
	if err := ValidateExternalURL(settings.UiBannerURL); err != nil {
		log.Warnf("Failed to validate UI banner URL in settings: %v", err)
	}

	additionalURLs := os.Getenv("ATHENA_ADDITIONAL_URLS")
	if additionalURLs != "" {
		if err := yaml.Unmarshal([]byte(additionalURLs), &settings.AdditionalURLs); err != nil {
			settings.AdditionalURLs = splitCommaSeparated(additionalURLs)
		}
	}
	for _, url := range settings.AdditionalURLs {
		if err := ValidateExternalURL(url); err != nil {
			log.Warnf("Failed to validate external URL in settings: %v", err)
		}
	}

	if userSessionDurationStr := os.Getenv("ATHENA_SESSION_DURATION"); userSessionDurationStr != "" {
		if val, err := timeutil.ParseDuration(userSessionDurationStr); err != nil {
			log.Warnf("Failed to parse ATHENA_SESSION_DURATION: %v", err)
		} else {
			settings.UserSessionDuration = *val
		}
	}

	return settings, nil
}

func envOrFile(envName string) (string, error) {
	if value := os.Getenv(envName); value != "" {
		return value, nil
	}

	fileName := envName + "_FILE"
	if filePath := os.Getenv(fileName); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed reading %s: %w", fileName, err)
		}

		return strings.TrimSpace(string(data)), nil
	}

	return "", nil
}

func loadRawSettingsFromEnv() (RawSettings, error) {
	raw := RawSettings{Secrets: map[string]string{}}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok || !strings.HasPrefix(key, "ATHENA_SECRET_") {
			continue
		}

		raw.Secrets[strings.TrimPrefix(key, "ATHENA_SECRET_")] = value
	}

	return raw, nil
}
