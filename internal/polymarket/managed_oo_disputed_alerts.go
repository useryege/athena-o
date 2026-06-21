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
	managedOODisputedAlertTopic  = "[POLY] UMA Disputed"
	managedOODisputedAlertSource = "polymarket.uma-disputed"

	defaultManagedOODisputedAlertMaxPerRefresh      = 100
	defaultManagedOODisputedAlertSendTimeout        = 10 * time.Second
	defaultManagedOODisputedAlertQuestionMaxRunes   = 160
	defaultManagedOODisputedAlertTitleQuestionRunes = 80
)

type ManagedOODisputedAlertsConfig struct {
	Enabled       bool
	MaxPerRefresh int
	SendTimeout   time.Duration
}

func defaultManagedOODisputedAlertsConfig() ManagedOODisputedAlertsConfig {
	return ManagedOODisputedAlertsConfig{
		MaxPerRefresh: defaultManagedOODisputedAlertMaxPerRefresh,
		SendTimeout:   defaultManagedOODisputedAlertSendTimeout,
	}
}

func normalizeManagedOODisputedAlertsConfig(config ManagedOODisputedAlertsConfig) ManagedOODisputedAlertsConfig {
	defaults := defaultManagedOODisputedAlertsConfig()
	if config.MaxPerRefresh <= 0 {
		config.MaxPerRefresh = defaults.MaxPerRefresh
	}
	if config.SendTimeout <= 0 {
		config.SendTimeout = defaults.SendTimeout
	}
	return config
}

func (s *Service) sendManagedOODisputePriceAlerts(ctx context.Context) {
	config := normalizeManagedOODisputedAlertsConfig(s.managedOODisputedAlertsConfig)
	if !config.Enabled || s.store == nil || s.notificationClientset == nil {
		return
	}
	candidates, err := s.store.ListManagedOODisputePriceAlertCandidates(ctx, int32(config.MaxPerRefresh))
	if err != nil {
		if ctx.Err() == nil {
			log.WithError(err).Warn("failed to list polymarket managed oo dispute price alert candidates")
		}
		return
	}
	if len(candidates) == 0 {
		return
	}

	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		if ctx.Err() == nil {
			log.WithError(err).Warn("failed to create polymarket managed oo disputed notification client")
		}
		return
	}
	defer utilio.Close(closer)

	for _, candidate := range candidates {
		request := renderManagedOODisputePriceAlertNotification(candidate)
		sendCtx, cancel := context.WithTimeout(ctx, config.SendTimeout)
		response, err := client.SendNotification(sendCtx, request)
		cancel()
		if err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("tx_hash", candidate.TxHash).
					WithField("log_index", candidate.LogIndex).
					Warn("failed to send polymarket managed oo disputed notification")
			}
			continue
		}

		notificationID := int64(0)
		if response != nil {
			notificationID = response.GetNotificationId()
		}
		if err := s.store.UpsertManagedOODisputePriceAlertState(ctx, candidate.TxHash, candidate.LogIndex, notificationID, s.now().UTC()); err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("tx_hash", candidate.TxHash).
					WithField("log_index", candidate.LogIndex).
					Warn("failed to update polymarket managed oo disputed alert state")
			}
		}
	}
}

func renderManagedOODisputePriceAlertNotification(candidate polymarketstore.ManagedOODisputePriceAlertCandidate) *notificationapiclient.SendNotificationRequest {
	question := firstNonEmpty(candidate.Question, managedOOAncillaryTitle(candidate.AncillaryDataText))
	titleSubject := firstNonEmpty(question, candidate.MarketID, candidate.TxHash)
	bodyLines := []string{
		fmt.Sprintf("Market ID: %s", firstNonEmpty(candidate.MarketID, "-")),
		fmt.Sprintf("Question: %s", firstNonEmpty(truncateRunes(question, defaultManagedOODisputedAlertQuestionMaxRunes), "-")),
		fmt.Sprintf("Matched labels: %s", firstNonEmpty(candidate.MatchedLabels, "-")),
		fmt.Sprintf("Requester: %s", firstNonEmpty(candidate.Requester, "-")),
		fmt.Sprintf("Proposer: %s", firstNonEmpty(candidate.Proposer, "-")),
		fmt.Sprintf("Disputer: %s", firstNonEmpty(candidate.Disputer, "-")),
		fmt.Sprintf("Proposed price: %s", firstNonEmpty(candidate.ProposedPrice, "-")),
		fmt.Sprintf("Request timestamp: %s", formatManagedOOAlertTimestamp(candidate.RequestTimestamp)),
		fmt.Sprintf("Tx hash: %s", firstNonEmpty(candidate.TxHash, "-")),
		fmt.Sprintf("Condition ID: %s", firstNonEmpty(candidate.ConditionID, "-")),
	}

	return &notificationapiclient.SendNotificationRequest{
		Source:     managedOODisputedAlertSource,
		Severity:   notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING,
		Title:      fmt.Sprintf("UMA Disputed: %s", truncateRunes(titleSubject, defaultManagedOODisputedAlertTitleQuestionRunes)),
		Body:       strings.Join(bodyLines, "\n"),
		Link:       managedOODisputePriceAlertLink(candidate),
		TopicLabel: managedOODisputedAlertTopic,
	}
}

func managedOODisputePriceAlertLink(candidate polymarketstore.ManagedOODisputePriceAlertCandidate) string {
	if link := polymarketEventMarketLink(candidate.EventSlug, candidate.MarketSlug); link != "" {
		return link
	}
	return polymarketMarketLink(candidate.MarketSlug)
}

func managedOOAncillaryTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	start := strings.Index(lower, "title:")
	if start < 0 {
		return ""
	}

	title := strings.TrimSpace(value[start+len("title:"):])
	lowerTitle := strings.ToLower(title)
	end := len(title)
	for _, marker := range []string{", description:", "\ndescription:", " description:", " market_id:"} {
		if idx := strings.Index(lowerTitle, marker); idx >= 0 && idx < end {
			end = idx
		}
	}
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(title[:end]), ","))
}
