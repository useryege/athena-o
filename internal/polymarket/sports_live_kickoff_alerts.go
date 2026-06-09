package polymarket

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/notification"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
)

const (
	sportsKickoffAlertSource        = "polymarket.sports-live"
	defaultSportsKickoffStateTTL    = 72 * time.Hour
	defaultSportsKickoffSendTimeout = 10 * time.Second
	defaultSportsKickoffTitleRunes  = 96
)

type sportsLiveMarketAlertState struct {
	LastSeenAt int64
}

type sportsKickoffAlertCandidate struct {
	conditionID string
	eventSlug   string
	request     *notificationapiclient.SendNotificationRequest
}

func (s *Service) collectSportsKickoffAlertsLocked(items []*v1alpha1.PolymarketSportsLiveMarketItem, nowUnix int64) []sportsKickoffAlertCandidate {
	if items == nil {
		return nil
	}
	if nowUnix <= 0 {
		nowUnix = s.nowUnix()
	}
	if s.sportsLiveMarketAlertStates == nil {
		s.sportsLiveMarketAlertStates = make(map[string]sportsLiveMarketAlertState)
	}

	initialized := s.sportsLiveMarketAlertsInitialized
	alerts := make([]sportsKickoffAlertCandidate, 0)
	for _, item := range items {
		if item == nil {
			continue
		}
		conditionID := strings.TrimSpace(item.ConditionID)
		if conditionID == "" {
			continue
		}

		_, seen := s.sportsLiveMarketAlertStates[conditionID]
		s.sportsLiveMarketAlertStates[conditionID] = sportsLiveMarketAlertState{LastSeenAt: nowUnix}
		if initialized && !seen {
			alerts = append(alerts, sportsKickoffAlertCandidate{
				conditionID: conditionID,
				eventSlug:   strings.TrimSpace(item.EventSlug),
				request:     renderSportsKickoffNotification(item),
			})
		}
	}

	s.sportsLiveMarketAlertsInitialized = true
	s.cleanupSportsLiveMarketAlertStatesLocked(nowUnix)
	return alerts
}

func (s *Service) cleanupSportsLiveMarketAlertStatesLocked(nowUnix int64) {
	cutoff := nowUnix - int64(defaultSportsKickoffStateTTL.Seconds())
	for conditionID, state := range s.sportsLiveMarketAlertStates {
		if state.LastSeenAt > 0 && state.LastSeenAt < cutoff {
			delete(s.sportsLiveMarketAlertStates, conditionID)
		}
	}
}

func renderSportsKickoffNotification(item *v1alpha1.PolymarketSportsLiveMarketItem) *notificationapiclient.SendNotificationRequest {
	eventSlug := strings.TrimSpace(item.EventSlug)
	conditionID := strings.TrimSpace(item.ConditionID)
	titleText := firstNonEmpty(strings.TrimSpace(item.Title), strings.TrimSpace(item.MarketSlug), conditionID, "Live sports market")
	title := fmt.Sprintf("Polymarket live market: %s", truncateRunes(titleText, defaultSportsKickoffTitleRunes))

	body := strings.Join([]string{
		fmt.Sprintf("Market: %s", titleText),
		fmt.Sprintf("Event slug: %s", firstNonEmpty(eventSlug, "-")),
		fmt.Sprintf("Market slug: %s", firstNonEmpty(strings.TrimSpace(item.MarketSlug), "-")),
		fmt.Sprintf("Condition ID: %s", conditionID),
		fmt.Sprintf("Score: %s", firstNonEmpty(strings.TrimSpace(item.Score), "-")),
		fmt.Sprintf("Period: %s", firstNonEmpty(strings.TrimSpace(item.Period), "-")),
		fmt.Sprintf("Elapsed: %s", firstNonEmpty(strings.TrimSpace(item.Elapsed), "-")),
		fmt.Sprintf("Last update: %s", firstNonEmpty(strings.TrimSpace(item.LastUpdate), "-")),
		fmt.Sprintf("Volume: %.2f", item.VolumeNum),
		fmt.Sprintf("Liquidity: %.2f", item.LiquidityNum),
	}, "\n")

	return &notificationapiclient.SendNotificationRequest{
		Source:     sportsKickoffAlertSource,
		Severity:   notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
		Title:      title,
		Body:       body,
		Link:       polymarketEventLink(eventSlug),
		TopicLabel: notification.NotificationTopicLabelPolyKickoff,
	}
}

func (s *Service) sendSportsKickoffAlerts(ctx context.Context, alerts []sportsKickoffAlertCandidate) {
	if len(alerts) == 0 || s.notificationClientset == nil {
		return
	}
	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		log.WithError(err).Warn("failed to create polymarket sports kickoff notification client")
		return
	}
	defer utilio.Close(closer)

	for _, alert := range alerts {
		if alert.request == nil {
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, defaultSportsKickoffSendTimeout)
		_, err := client.SendNotification(sendCtx, alert.request)
		cancel()
		if err != nil {
			log.WithError(err).
				WithField("condition_id", alert.conditionID).
				WithField("event_slug", alert.eventSlug).
				Warn("failed to send polymarket sports live market notification")
		}
	}
}

func polymarketEventLink(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	return polymarketEventBaseURL + url.PathEscape(slug)
}
