package sportslive

import (
	"net/url"
	"strings"
)

const polymarketEventBaseURL = "https://polymarket.com/event/"

func ptrBool(value bool) *bool {
	return &value
}

func (s *Service) polymarketNotificationLink(link string) string {
	if s == nil {
		return link
	}
	return polymarketLinkWithInviteCode(link, s.notificationInviteCode)
}

func polymarketLinkWithInviteCode(link, inviteCode string) string {
	trimmedLink := strings.TrimSpace(link)
	inviteCode = strings.TrimSpace(inviteCode)
	if trimmedLink == "" || inviteCode == "" {
		return link
	}
	parsed, err := url.Parse(trimmedLink)
	if err != nil {
		return link
	}
	query := parsed.Query()
	query.Set("r", inviteCode)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func polymarketEventLink(slug string) string {
	slug = strings.Trim(strings.TrimSpace(slug), "/")
	if slug == "" {
		return ""
	}
	return polymarketEventBaseURL + slug
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
