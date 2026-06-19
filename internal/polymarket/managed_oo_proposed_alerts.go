package polymarket

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	utilio "github.com/useryege/athena/util/io"
)

const (
	managedOOProposedAlertTopic  = "[POLY] UMA Proposed"
	managedOOProposedAlertSource = "polymarket.uma-proposed"

	defaultManagedOOProposedAlertMaxPerRefresh      = 100
	defaultManagedOOProposedAlertSendTimeout        = 10 * time.Second
	defaultManagedOOProposedAlertQuestionMaxRunes   = 160
	defaultManagedOOProposedAlertTitleQuestionRunes = 80
)

type ManagedOOProposedAlertsConfig struct {
	Enabled       bool
	MaxPerRefresh int
	SendTimeout   time.Duration
}

func defaultManagedOOProposedAlertsConfig() ManagedOOProposedAlertsConfig {
	return ManagedOOProposedAlertsConfig{
		MaxPerRefresh: defaultManagedOOProposedAlertMaxPerRefresh,
		SendTimeout:   defaultManagedOOProposedAlertSendTimeout,
	}
}

func normalizeManagedOOProposedAlertsConfig(config ManagedOOProposedAlertsConfig) ManagedOOProposedAlertsConfig {
	defaults := defaultManagedOOProposedAlertsConfig()
	if config.MaxPerRefresh <= 0 {
		config.MaxPerRefresh = defaults.MaxPerRefresh
	}
	if config.SendTimeout <= 0 {
		config.SendTimeout = defaults.SendTimeout
	}
	return config
}

func (s *Service) sendManagedOOProposePriceAlerts(ctx context.Context) {
	config := normalizeManagedOOProposedAlertsConfig(s.managedOOProposedAlertsConfig)
	if !config.Enabled || s.store == nil || s.notificationClientset == nil {
		return
	}
	candidates, err := s.store.ListManagedOOProposePriceAlertCandidates(ctx, int32(config.MaxPerRefresh))
	if err != nil {
		if ctx.Err() == nil {
			log.WithError(err).Warn("failed to list polymarket managed oo propose price alert candidates")
		}
		return
	}
	if len(candidates) == 0 {
		return
	}

	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		if ctx.Err() == nil {
			log.WithError(err).Warn("failed to create polymarket managed oo proposed notification client")
		}
		return
	}
	defer utilio.Close(closer)

	for _, candidate := range candidates {
		request := renderManagedOOProposePriceAlertNotification(candidate)
		sendCtx, cancel := context.WithTimeout(ctx, config.SendTimeout)
		response, err := client.SendNotification(sendCtx, request)
		cancel()
		if err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("tx_hash", candidate.TxHash).
					WithField("log_index", candidate.LogIndex).
					Warn("failed to send polymarket managed oo proposed notification")
			}
			continue
		}

		notificationID := int64(0)
		if response != nil {
			notificationID = response.GetNotificationId()
		}
		if err := s.store.UpsertManagedOOProposePriceAlertState(ctx, candidate.TxHash, candidate.LogIndex, notificationID, s.now().UTC()); err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("tx_hash", candidate.TxHash).
					WithField("log_index", candidate.LogIndex).
					Warn("failed to update polymarket managed oo proposed alert state")
			}
		}
	}
}

func renderManagedOOProposePriceAlertNotification(candidate polymarketstore.ManagedOOProposePriceAlertCandidate) *notificationapiclient.SendNotificationRequest {
	titleSubject := firstNonEmpty(candidate.Question, candidate.MarketID)
	bodyLines := []string{
		fmt.Sprintf("Market ID: %s", firstNonEmpty(candidate.MarketID, "-")),
		fmt.Sprintf("Question: %s", firstNonEmpty(truncateRunes(candidate.Question, defaultManagedOOProposedAlertQuestionMaxRunes), "-")),
		fmt.Sprintf("Matched labels: %s", firstNonEmpty(candidate.MatchedLabels, "-")),
		fmt.Sprintf("Proposer: %s", firstNonEmpty(candidate.Proposer, "-")),
		fmt.Sprintf("Proposed price: %s", firstNonEmpty(candidate.ProposedPrice, "-")),
		fmt.Sprintf("Request timestamp: %s", formatManagedOOAlertTimestamp(candidate.RequestTimestamp)),
		fmt.Sprintf("Expiration timestamp: %s", formatManagedOOAlertTimestamp(candidate.ExpirationTimestamp)),
		fmt.Sprintf("Tx hash: %s", firstNonEmpty(candidate.TxHash, "-")),
		fmt.Sprintf("Condition ID: %s", firstNonEmpty(candidate.ConditionID, "-")),
	}

	return &notificationapiclient.SendNotificationRequest{
		Source:     managedOOProposedAlertSource,
		Severity:   notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING,
		Title:      fmt.Sprintf("UMA Proposed: %s", truncateRunes(titleSubject, defaultManagedOOProposedAlertTitleQuestionRunes)),
		Body:       strings.Join(bodyLines, "\n"),
		Link:       polymarketEventLink(candidate.Slug),
		TopicLabel: managedOOProposedAlertTopic,
	}
}

func formatManagedOOAlertTimestamp(value int64) string {
	if value <= 0 {
		return "-"
	}
	return fmt.Sprintf("%s (%d)", time.Unix(value, 0).UTC().Format(time.RFC3339), value)
}
