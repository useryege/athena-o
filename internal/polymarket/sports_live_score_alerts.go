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
	sportsLiveScoreAlertTopic  = "[POLY] FIFWC 比分"
	sportsLiveScoreAlertSource = "polymarket.sports-live-score"

	defaultSportsLiveScoreAlertSendTimeout   = 10 * time.Second
	defaultSportsLiveScoreAlertTitleMaxRunes = 120
)

type SportsLiveScoreAlertsConfig struct {
	Enabled     bool
	SendTimeout time.Duration
}

func defaultSportsLiveScoreAlertsConfig() SportsLiveScoreAlertsConfig {
	return SportsLiveScoreAlertsConfig{
		SendTimeout: defaultSportsLiveScoreAlertSendTimeout,
	}
}

func normalizeSportsLiveScoreAlertsConfig(config SportsLiveScoreAlertsConfig) SportsLiveScoreAlertsConfig {
	defaults := defaultSportsLiveScoreAlertsConfig()
	if config.SendTimeout <= 0 {
		config.SendTimeout = defaults.SendTimeout
	}
	return config
}

func (s *Service) updateSportsLiveScoreAlerts(ctx context.Context) {
	config := normalizeSportsLiveScoreAlertsConfig(s.sportsLiveScoreAlertsConfig)
	if !config.Enabled || s.store == nil || s.notificationClientset == nil {
		return
	}
	candidates, err := s.store.ListSportsLiveScoreAlertCandidates(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.WithError(err).Warn("failed to list polymarket sports live score alert candidates")
		}
		return
	}
	if len(candidates) == 0 {
		return
	}

	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		if ctx.Err() == nil {
			log.WithError(err).Warn("failed to create polymarket sports live score alert notification client")
		}
		return
	}
	defer utilio.Close(closer)

	for _, candidate := range candidates {
		if ctx.Err() != nil {
			return
		}
		sendCtx, cancel := context.WithTimeout(ctx, config.SendTimeout)
		response, err := client.SendNotification(sendCtx, s.renderSportsLiveScoreAlertNotification(candidate))
		cancel()
		if err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("event_key", candidate.EventKey).
					WithField("previous_score", candidate.PreviousScore).
					WithField("score", candidate.Score).
					Warn("failed to send polymarket sports live score alert")
			}
			continue
		}

		notificationID := int64(0)
		if response != nil {
			notificationID = response.GetNotificationId()
		}
		if err := s.store.UpdateSportsLiveScoreAlertState(ctx, polymarketstore.SportsLiveScoreAlertState{
			EventKey:       candidate.EventKey,
			Score:          candidate.Score,
			NotificationID: notificationID,
			LastNotifiedAt: s.now().UTC(),
		}); err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("event_key", candidate.EventKey).
					WithField("notification_id", notificationID).
					Warn("failed to update polymarket sports live score alert state")
			}
		}
	}
}

func (s *Service) renderSportsLiveScoreAlertNotification(candidate polymarketstore.SportsLiveScoreAlertCandidate) *notificationapiclient.SendNotificationRequest {
	eventTitle := firstNonEmpty(candidate.Title, candidate.Slug, candidate.EventKey)
	bodyLines := []string{
		fmt.Sprintf("Event: %s", eventTitle),
		fmt.Sprintf("Score: %s → %s", candidate.PreviousScore, candidate.Score),
		fmt.Sprintf("Period: %s", firstNonEmpty(candidate.Period, "-")),
		fmt.Sprintf("Elapsed: %s", firstNonEmpty(candidate.Elapsed, "-")),
		fmt.Sprintf("Game status: %s", firstNonEmpty(candidate.GameStatus, "-")),
	}
	if !candidate.FetchedAt.IsZero() {
		bodyLines = append(bodyLines, fmt.Sprintf("Updated at: %s", candidate.FetchedAt.UTC().Format(time.RFC3339)))
	}

	return &notificationapiclient.SendNotificationRequest{
		Source:       sportsLiveScoreAlertSource,
		Severity:     notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
		Title:        truncateRunes(fmt.Sprintf("FIFWC score update: %s %s", eventTitle, candidate.Score), defaultSportsLiveScoreAlertTitleMaxRunes),
		Body:         strings.Join(bodyLines, "\n"),
		Link:         s.polymarketNotificationLink(polymarketEventLink(candidate.Slug)),
		TelegramChat: notificationapiclient.TelegramChat_TELEGRAM_CHAT_TEST,
		TopicLabel:   sportsLiveScoreAlertTopic,
	}
}
