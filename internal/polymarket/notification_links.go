package polymarket

import (
	"net/url"
	"strings"
)

func (s *Service) polymarketNotificationLink(link string) string {
	if s == nil {
		return link
	}
	return polymarketLinkWithInviteCode(link, s.notificationInviteCode)
}

func polymarketLinkWithInviteCode(link string, inviteCode string) string {
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
