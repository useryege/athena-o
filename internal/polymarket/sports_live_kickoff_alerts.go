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
	utilio "github.com/useryege/athena/util/io"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
)

const (
	sportsKickoffAlertSource        = "polymarket.sports-live"
	defaultSportsKickoffStateTTL    = 72 * time.Hour
	defaultSportsKickoffSendTimeout = 10 * time.Second
	defaultSportsKickoffTitleRunes  = 96
	defaultSportsKickoffMarketRunes = 80
	defaultSportsKickoffMarketLimit = 5
)

type sportsKickoffState struct {
	WasLive    bool
	Alerted    bool
	LastSeenAt int64
}

type sportsKickoffAlertCandidate struct {
	eventSlug string
	request   *notificationapiclient.SendNotificationRequest
}

func (s *Service) collectSportsKickoffAlertsLocked(snapshot *utilpolymarket.SportsLiveSnapshot, nowUnix int64) []sportsKickoffAlertCandidate {
	if snapshot == nil {
		return nil
	}
	if nowUnix <= 0 {
		nowUnix = s.nowUnix()
	}
	if s.sportsKickoffStates == nil {
		s.sportsKickoffStates = make(map[string]sportsKickoffState)
	}

	for i := range snapshot.Soon {
		eventSlug := strings.TrimSpace(snapshot.Soon[i].Slug)
		if eventSlug == "" {
			continue
		}
		state := s.sportsKickoffStates[eventSlug]
		state.WasLive = false
		state.LastSeenAt = nowUnix
		s.sportsKickoffStates[eventSlug] = state
	}

	alerts := make([]sportsKickoffAlertCandidate, 0)
	for i := range snapshot.Live {
		event := snapshot.Live[i]
		eventSlug := strings.TrimSpace(event.Slug)
		if eventSlug == "" || !isSportsLiveEventStarted(event) {
			continue
		}

		state, seen := s.sportsKickoffStates[eventSlug]
		if seen && !state.WasLive && !state.Alerted {
			alerts = append(alerts, sportsKickoffAlertCandidate{
				eventSlug: eventSlug,
				request:   renderSportsKickoffNotification(event),
			})
			state.Alerted = true
		}
		if !seen {
			state.Alerted = true
		}
		state.WasLive = true
		state.LastSeenAt = nowUnix
		s.sportsKickoffStates[eventSlug] = state
	}

	s.cleanupSportsKickoffStatesLocked(nowUnix)
	return alerts
}

func (s *Service) cleanupSportsKickoffStatesLocked(nowUnix int64) {
	cutoff := nowUnix - int64(defaultSportsKickoffStateTTL.Seconds())
	for eventSlug, state := range s.sportsKickoffStates {
		if state.LastSeenAt > 0 && state.LastSeenAt < cutoff {
			delete(s.sportsKickoffStates, eventSlug)
		}
	}
}

func isSportsLiveEventStarted(event utilpolymarket.SportsEventSnapshot) bool {
	return boolValue(event.Live) && !boolValue(event.Ended)
}

func renderSportsKickoffNotification(event utilpolymarket.SportsEventSnapshot) *notificationapiclient.SendNotificationRequest {
	eventSlug := strings.TrimSpace(event.Slug)
	titleText := firstNonEmpty(strings.TrimSpace(stringValue(event.Title)), eventSlug, "Sports event")
	title := fmt.Sprintf("Polymarket kickoff: %s", truncateRunes(titleText, defaultSportsKickoffTitleRunes))

	body := strings.Join([]string{
		fmt.Sprintf("Event: %s", titleText),
		fmt.Sprintf("Score: %s", firstNonEmpty(strings.TrimSpace(stringValue(event.Score)), "-")),
		fmt.Sprintf("Period: %s", firstNonEmpty(strings.TrimSpace(stringValue(event.Period)), "-")),
		fmt.Sprintf("Elapsed: %s", firstNonEmpty(strings.TrimSpace(stringValue(event.Elapsed)), "-")),
		fmt.Sprintf("Game status: %s", firstNonEmpty(strings.TrimSpace(stringValue(event.GameStatus)), "-")),
		fmt.Sprintf("Start time: %s", firstNonEmpty(formatTimeRFC3339(event.StartTime), "-")),
		fmt.Sprintf("Event slug: %s", eventSlug),
		"Moneyline markets:",
		formatSportsKickoffMarketSummaries(event.Markets),
	}, "\n")

	return &notificationapiclient.SendNotificationRequest{
		Source:   sportsKickoffAlertSource,
		Severity: notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
		Title:    title,
		Body:     body,
		Link:     polymarketEventLink(eventSlug),
		Topic:    notification.NotificationTopicPolyKickoff,
	}
}

func formatSportsKickoffMarketSummaries(groups []utilpolymarket.SportsMarketGroup) string {
	lines := make([]string, 0, defaultSportsKickoffMarketLimit)
	for i := range groups {
		if strings.TrimSpace(strings.ToLower(groups[i].Type)) != sportsLiveMoneylineMarketType {
			continue
		}
		for j := range groups[i].Markets {
			if len(lines) >= defaultSportsKickoffMarketLimit {
				return strings.Join(lines, "\n")
			}
			market := groups[i].Markets[j]
			question := firstNonEmpty(strings.TrimSpace(stringValue(market.Question)), strings.TrimSpace(groups[i].Title), strings.TrimSpace(stringValue(market.Slug)), "Market")
			outcomes := parseJSONStringList(market.Outcomes)
			prices := parseJSONStringList(market.OutcomePrices)
			lines = append(lines, fmt.Sprintf("- %s (%s): %s",
				truncateRunes(question, defaultSportsKickoffMarketRunes),
				firstNonEmpty(strings.TrimSpace(stringValue(market.ConditionID)), strings.TrimSpace(stringValue(market.Slug)), "-"),
				formatSportsKickoffOutcomes(outcomes, prices),
			))
		}
	}
	if len(lines) == 0 {
		return "-"
	}
	return strings.Join(lines, "\n")
}

func formatSportsKickoffOutcomes(outcomes, prices []string) string {
	if len(outcomes) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(outcomes))
	for i := range outcomes {
		outcome := strings.TrimSpace(outcomes[i])
		if outcome == "" {
			continue
		}
		price := ""
		if i < len(prices) {
			price = strings.TrimSpace(prices[i])
		}
		if price == "" {
			parts = append(parts, outcome)
			continue
		}
		parts = append(parts, outcome+" "+price)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " | ")
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
			log.WithError(err).WithField("event_slug", alert.eventSlug).Warn("failed to send polymarket sports kickoff notification")
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
