package polymarket

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	utilio "github.com/useryege/athena/util/io"
)

const (
	sportsLivePriceAlert90Topic = "[POLY] Sports Live 90/10"
	sportsLivePriceAlert95Topic = "[POLY] Sports Live 95/5"
	sportsLivePriceAlert97Topic = "[POLY] Sports Live 97/3"

	sportsLivePriceAlert90Source = "polymarket.sports-live-price-alert-90-10"
	sportsLivePriceAlert95Source = "polymarket.sports-live-price-alert-95-5"
	sportsLivePriceAlert97Source = "polymarket.sports-live-price-alert-97-3"

	sportsLivePriceAlertBandNone = "none"
	sportsLivePriceAlertBandB    = "b"
	sportsLivePriceAlertBandC    = "c"
	sportsLivePriceAlertBandD    = "d"

	defaultSportsLivePriceAlertSendTimeout       = 10 * time.Second
	defaultSportsLivePriceAlertDowngradeCooldown = 3 * time.Minute
)

type SportsLivePriceAlertsConfig struct {
	Enabled           bool
	SendTimeout       time.Duration
	DowngradeCooldown time.Duration
}

func defaultSportsLivePriceAlertsConfig() SportsLivePriceAlertsConfig {
	return SportsLivePriceAlertsConfig{
		SendTimeout:       defaultSportsLivePriceAlertSendTimeout,
		DowngradeCooldown: defaultSportsLivePriceAlertDowngradeCooldown,
	}
}

func normalizeSportsLivePriceAlertsConfig(config SportsLivePriceAlertsConfig) SportsLivePriceAlertsConfig {
	defaults := defaultSportsLivePriceAlertsConfig()
	if config.SendTimeout <= 0 {
		config.SendTimeout = defaults.SendTimeout
	}
	if config.DowngradeCooldown <= 0 {
		config.DowngradeCooldown = defaults.DowngradeCooldown
	}
	return config
}

func (s *Service) updateSportsLivePriceAlerts(ctx context.Context) {
	config := normalizeSportsLivePriceAlertsConfig(s.sportsLivePriceAlertsConfig)
	if !config.Enabled || s.notificationClientset == nil || s.store == nil {
		return
	}
	tokens, err := s.store.ListSportsLiveLatestPriceAlertTokens(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.WithError(err).Warn("failed to list polymarket sports live price alert tokens")
		}
		return
	}
	if len(tokens) == 0 {
		return
	}

	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		log.WithError(err).Warn("failed to create polymarket sports live price alert notification client")
		return
	}
	defer utilio.Close(closer)

	for _, token := range tokens {
		if ctx.Err() != nil {
			return
		}
		currentBand := strings.TrimSpace(token.AlertBand)
		if currentBand == "" {
			currentBand = sportsLivePriceAlertBandNone
		}
		nextBand := classifySportsLivePriceAlertBand(token.Price)
		if nextBand == currentBand {
			continue
		}

		state := sportsLivePriceAlertState(token, nextBand, time.Time{})
		now := s.now().UTC()
		if shouldSuppressSportsLivePriceAlert97Repeat(nextBand, token.LastNotifiedAt, now, config.DowngradeCooldown) {
			state.LastNotifiedAt = token.LastNotifiedAt
			if err := s.store.UpsertSportsLivePriceAlertState(ctx, state); err != nil {
				if ctx.Err() == nil {
					log.WithError(err).
						WithField("token_id", token.TokenID).
						WithField("condition_id", token.ConditionID).
						Warn("failed to update polymarket sports live price alert state")
				}
			}
			continue
		}
		if isSportsLivePriceAlertDowngrade(currentBand, nextBand) && !token.LastNotifiedAt.IsZero() && now.Sub(token.LastNotifiedAt) < config.DowngradeCooldown {
			state.AlertBand = currentBand
			state.LastNotifiedAt = token.LastNotifiedAt
			if err := s.store.UpsertSportsLivePriceAlertState(ctx, state); err != nil {
				if ctx.Err() == nil {
					log.WithError(err).
						WithField("token_id", token.TokenID).
						WithField("condition_id", token.ConditionID).
						Warn("failed to update polymarket sports live price alert state")
				}
			}
			continue
		}
		if nextBand != sportsLivePriceAlertBandNone {
			if err := s.sendSportsLivePriceAlert(ctx, client, config, token, nextBand); err != nil {
				if ctx.Err() == nil {
					log.WithError(err).
						WithField("token_id", token.TokenID).
						WithField("condition_id", token.ConditionID).
						Warn("failed to send polymarket sports live price alert")
				}
				continue
			}
			state.LastNotifiedAt = now
		} else {
			state.LastNotifiedAt = token.LastNotifiedAt
		}
		if err := s.store.UpsertSportsLivePriceAlertState(ctx, state); err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("token_id", token.TokenID).
					WithField("condition_id", token.ConditionID).
					Warn("failed to update polymarket sports live price alert state")
			}
		}
	}
}

func shouldSuppressSportsLivePriceAlert97Repeat(nextBand string, lastNotifiedAt, now time.Time, cooldown time.Duration) bool {
	return nextBand == sportsLivePriceAlertBandD &&
		!lastNotifiedAt.IsZero() &&
		now.Sub(lastNotifiedAt) < cooldown
}

func isSportsLivePriceAlertDowngrade(currentBand, nextBand string) bool {
	currentRank := sportsLivePriceAlertBandRank(currentBand)
	nextRank := sportsLivePriceAlertBandRank(nextBand)
	return currentRank > 0 && nextRank > 0 && nextRank < currentRank
}

func sportsLivePriceAlertBandRank(band string) int {
	switch band {
	case sportsLivePriceAlertBandB:
		return 1
	case sportsLivePriceAlertBandC:
		return 2
	case sportsLivePriceAlertBandD:
		return 3
	default:
		return 0
	}
}

func (s *Service) sendSportsLivePriceAlert(ctx context.Context, client notificationapiclient.NotificationServiceClient, config SportsLivePriceAlertsConfig, token polymarketstore.SportsLivePriceAlertToken, band string) error {
	request := renderSportsLivePriceAlertNotification(token, band)
	if request == nil {
		return fmt.Errorf("notification request is nil")
	}
	sendCtx, cancel := context.WithTimeout(ctx, config.SendTimeout)
	defer cancel()
	_, err := client.SendNotification(sendCtx, request)
	return err
}

func sportsLivePriceAlertState(token polymarketstore.SportsLivePriceAlertToken, band string, notifiedAt time.Time) polymarketstore.SportsLivePriceAlertState {
	return polymarketstore.SportsLivePriceAlertState{
		TokenID:        strings.TrimSpace(token.TokenID),
		MarketKey:      strings.TrimSpace(token.MarketKey),
		EventKey:       strings.TrimSpace(token.EventKey),
		ConditionID:    strings.TrimSpace(token.ConditionID),
		Outcome:        strings.TrimSpace(token.Outcome),
		AlertBand:      band,
		LastPrice:      token.Price,
		LastPriceTs:    token.PriceTs,
		LastNotifiedAt: notifiedAt,
	}
}

func classifySportsLivePriceAlertBand(price float64) string {
	if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 || price > 1 {
		return sportsLivePriceAlertBandNone
	}
	switch {
	case price < 0.03:
		return sportsLivePriceAlertBandD
	case price < 0.05:
		return sportsLivePriceAlertBandC
	case price < 0.1:
		return sportsLivePriceAlertBandB
	default:
		return sportsLivePriceAlertBandNone
	}
}

func renderSportsLivePriceAlertNotification(token polymarketstore.SportsLivePriceAlertToken, band string) *notificationapiclient.SendNotificationRequest {
	topic := sportsLivePriceAlert90Topic
	source := sportsLivePriceAlert90Source
	severity := notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING
	alertBand := "< 0.1"
	titlePrefix := "Polymarket sports live 90/10 price alert"
	switch band {
	case sportsLivePriceAlertBandD:
		topic = sportsLivePriceAlert97Topic
		source = sportsLivePriceAlert97Source
		severity = notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL
		alertBand = "< 0.03"
		titlePrefix = "Polymarket sports live 97/3 price alert"
	case sportsLivePriceAlertBandC:
		topic = sportsLivePriceAlert95Topic
		source = sportsLivePriceAlert95Source
		severity = notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL
		alertBand = "< 0.05"
		titlePrefix = "Polymarket sports live 95/5 price alert"
	case sportsLivePriceAlertBandB:
		topic = sportsLivePriceAlert90Topic
		source = sportsLivePriceAlert90Source
		alertBand = "< 0.1"
		titlePrefix = "Polymarket sports live 90/10 price alert"
	}

	eventTitle := firstNonEmpty(token.EventTitle, token.EventKey)
	marketTitle := firstNonEmpty(token.MarketTitle, token.MarketKey)
	outcome := firstNonEmpty(token.Outcome, "Outcome")
	bodyLines := []string{
		fmt.Sprintf("Event: %s", eventTitle),
		fmt.Sprintf("Market: %s", marketTitle),
		fmt.Sprintf("Outcome: %s", outcome),
		fmt.Sprintf("Latest sampled price: %.2f%%", token.Price*100),
		fmt.Sprintf("Alert band: %s", alertBand),
		fmt.Sprintf("Sampled at: %s", token.PriceTs.UTC().Format(time.RFC3339)),
	}
	if score := strings.TrimSpace(token.Score); score != "" {
		bodyLines = append(bodyLines, fmt.Sprintf("Score: %s", score))
	}
	if gameStatus := sportsLivePriceAlertGameStatus(token); gameStatus != "" {
		bodyLines = append(bodyLines, fmt.Sprintf("Game status: %s", gameStatus))
	}
	bodyLines = append(bodyLines,
		fmt.Sprintf("Volume: %.2f", token.Volume),
		fmt.Sprintf("Liquidity: %.2f", token.Liquidity),
		fmt.Sprintf("Condition ID: %s", firstNonEmpty(token.ConditionID, "-")),
		fmt.Sprintf("Token ID: %s", token.TokenID),
	)

	return &notificationapiclient.SendNotificationRequest{
		Source:     source,
		Severity:   severity,
		Title:      fmt.Sprintf("%s: %s %.2f%%", titlePrefix, outcome, token.Price*100),
		Body:       strings.Join(bodyLines, "\n"),
		Link:       polymarketEventLink(token.EventSlug),
		TopicLabel: topic,
	}
}

func sportsLivePriceAlertGameStatus(token polymarketstore.SportsLivePriceAlertToken) string {
	parts := []string{
		strings.TrimSpace(token.Period),
		strings.TrimSpace(token.Elapsed),
		strings.TrimSpace(token.GameStatus),
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " · ")
}
