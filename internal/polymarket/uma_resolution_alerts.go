package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	utilio "github.com/useryege/athena/util/io"
)

const (
	umaResolutionStatusProposed = "proposed"
	umaResolutionStatusDisputed = "disputed"

	umaResolutionProposedTopic = "[POLY] UMA Proposed"
	umaResolutionDisputedTopic = "[POLY] UMA Disputed"

	umaResolutionProposedSource = "polymarket.uma-resolution.proposed"
	umaResolutionDisputedSource = "polymarket.uma-resolution.disputed"

	defaultUMAResolutionAlertSendTimeout   = 10 * time.Second
	defaultUMAResolutionAlertQuestionRunes = 120
	defaultUMAResolutionAlertTitleRunes    = 80
)

type UMAResolutionAlertsConfig struct {
	Enabled     bool
	SendTimeout time.Duration
}

func defaultUMAResolutionAlertsConfig() UMAResolutionAlertsConfig {
	return UMAResolutionAlertsConfig{
		SendTimeout: defaultUMAResolutionAlertSendTimeout,
	}
}

func normalizeUMAResolutionAlertsConfig(config UMAResolutionAlertsConfig) UMAResolutionAlertsConfig {
	defaults := defaultUMAResolutionAlertsConfig()
	if config.SendTimeout <= 0 {
		config.SendTimeout = defaults.SendTimeout
	}
	return config
}

func (s *Service) sendUMAResolutionAlerts(ctx context.Context, candidates []polymarketstore.UMAResolutionNotificationCandidate) {
	config := normalizeUMAResolutionAlertsConfig(s.umaResolutionAlertsConfig)
	if len(candidates) == 0 || !config.Enabled || s.notificationClientset == nil {
		return
	}
	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		log.WithError(err).Warn("failed to create polymarket uma resolution notification client")
		return
	}
	defer utilio.Close(closer)

	for _, candidate := range candidates {
		if ctx.Err() != nil {
			return
		}
		notificationID, err := s.sendUMAResolutionAlert(ctx, client, config, candidate)
		if err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("market_key", candidate.MarketKey).
					WithField("uma_resolution_status", candidate.UMAResolutionStatus).
					Warn("failed to send polymarket uma resolution notification")
			}
			continue
		}
		if err := s.store.MarkUMAResolutionNotificationSent(ctx, candidate, notificationID, s.now().UTC()); err != nil && ctx.Err() == nil {
			log.WithError(err).
				WithField("market_key", candidate.MarketKey).
				WithField("uma_resolution_status", candidate.UMAResolutionStatus).
				Warn("failed to update polymarket uma resolution notification state")
		}
	}
}

func (s *Service) sendUMAResolutionAlert(ctx context.Context, client notificationapiclient.NotificationServiceClient, config UMAResolutionAlertsConfig, candidate polymarketstore.UMAResolutionNotificationCandidate) (int64, error) {
	request := renderUMAResolutionAlertNotification(candidate)
	if request == nil {
		return 0, fmt.Errorf("notification request is nil")
	}
	sendCtx, cancel := context.WithTimeout(ctx, config.SendTimeout)
	defer cancel()
	resp, err := client.SendNotification(sendCtx, request)
	if err != nil {
		return 0, err
	}
	return resp.GetNotificationId(), nil
}

func renderUMAResolutionAlertNotification(candidate polymarketstore.UMAResolutionNotificationCandidate) *notificationapiclient.SendNotificationRequest {
	status := normalizeUMAResolutionStatus(candidate.UMAResolutionStatus)
	topic := umaResolutionProposedTopic
	source := umaResolutionProposedSource
	severity := notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING
	if status == umaResolutionStatusDisputed {
		topic = umaResolutionDisputedTopic
		source = umaResolutionDisputedSource
		severity = notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL
	}

	question := firstNonEmpty(candidate.Question, candidate.MarketKey)
	trail := formatUMAResolutionStatusTrail(candidate.UMAResolutionStatuses)
	if trail == "" {
		trail = status
	}
	bodyLines := []string{
		fmt.Sprintf("Question: %s", truncateRunes(question, defaultUMAResolutionAlertQuestionRunes)),
		fmt.Sprintf("Status: %s", status),
		fmt.Sprintf("Status trail: %s", trail),
		fmt.Sprintf("24h volume: %.2f", candidate.Volume24hr),
		fmt.Sprintf("Liquidity: %.2f", candidate.LiquidityNum),
		fmt.Sprintf("Condition ID: %s", firstNonEmpty(candidate.ConditionID, "-")),
		fmt.Sprintf("Market key: %s", candidate.MarketKey),
	}
	if !candidate.LastSeenAt.IsZero() {
		bodyLines = append(bodyLines, fmt.Sprintf("Seen at: %s", candidate.LastSeenAt.UTC().Format(time.RFC3339)))
	}

	return &notificationapiclient.SendNotificationRequest{
		Source:     source,
		Severity:   severity,
		Title:      fmt.Sprintf("Polymarket UMA %s: %s", status, truncateRunes(question, defaultUMAResolutionAlertTitleRunes)),
		Body:       strings.Join(bodyLines, "\n"),
		Link:       polymarketUMAResolutionLink(candidate),
		TopicLabel: topic,
	}
}

func formatUMAResolutionStatusTrail(value string) string {
	parts := parseUMAResolutionStatusTrail(value)
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " > ")
}

func parseUMAResolutionStatusTrail(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var parsed []string
	if err := json.Unmarshal([]byte(value), &parsed); err == nil {
		return compactUMAResolutionStatusTrail(parsed)
	}
	value = strings.Trim(value, `[]`)
	rawParts := strings.Split(value, ",")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return compactUMAResolutionStatusTrail(parts)
}

func compactUMAResolutionStatusTrail(values []string) []string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return parts
}

func polymarketUMAResolutionLink(candidate polymarketstore.UMAResolutionNotificationCandidate) string {
	return polymarketEventLink(firstNonEmpty(candidate.EventSlug, candidate.Slug))
}
