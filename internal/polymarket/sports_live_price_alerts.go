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

	defaultSportsLivePriceAlertSendTimeout = 10 * time.Second
	defaultSportsLivePriceAlertCooldown    = 15 * time.Minute
)

type SportsLivePriceAlertsConfig struct {
	Enabled     bool
	Cooldown    time.Duration
	SendTimeout time.Duration
}

func defaultSportsLivePriceAlertsConfig() SportsLivePriceAlertsConfig {
	return SportsLivePriceAlertsConfig{
		Cooldown:    defaultSportsLivePriceAlertCooldown,
		SendTimeout: defaultSportsLivePriceAlertSendTimeout,
	}
}

func normalizeSportsLivePriceAlertsConfig(config SportsLivePriceAlertsConfig) SportsLivePriceAlertsConfig {
	defaults := defaultSportsLivePriceAlertsConfig()
	if config.Cooldown <= 0 {
		config.Cooldown = defaults.Cooldown
	}
	if config.SendTimeout <= 0 {
		config.SendTimeout = defaults.SendTimeout
	}
	return config
}

func (s *Service) updateSportsLivePriceAlerts(ctx context.Context) {
	config := normalizeSportsLivePriceAlertsConfig(s.sportsLivePriceAlertsConfig)
	if !config.Enabled || s.store == nil {
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

	alertedAt := s.now().UTC()
	candidates := make([]struct {
		token polymarketstore.SportsLivePriceAlertToken
		band  string
	}, 0)
	for _, token := range tokens {
		if ctx.Err() != nil {
			return
		}
		band := classifySportsLivePriceAlertBand(token.Price)
		if band == sportsLivePriceAlertBandNone {
			s.clearSportsLivePriceAlertState(ctx, token)
			continue
		}
		if !shouldSendSportsLivePriceAlert(token, band, config, alertedAt) {
			continue
		}
		candidates = append(candidates, struct {
			token polymarketstore.SportsLivePriceAlertToken
			band  string
		}{token: token, band: band})
	}
	if len(candidates) == 0 || s.notificationClientset == nil {
		return
	}

	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		log.WithError(err).Warn("failed to create polymarket sports live price alert notification client")
		return
	}
	defer utilio.Close(closer)

	for _, candidate := range candidates {
		token := candidate.token
		band := candidate.band
		if ctx.Err() != nil {
			return
		}
		if err := s.sendSportsLivePriceAlert(ctx, client, config, token, band); err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("token_id", token.TokenID).
					WithField("condition_id", token.ConditionID).
					Warn("failed to send polymarket sports live price alert")
			}
			continue
		}
		if err := s.store.UpsertSportsLivePriceAlertState(ctx, polymarketstore.SportsLivePriceAlertState{
			TokenID:       token.TokenID,
			MarketKey:     token.MarketKey,
			EventKey:      token.EventKey,
			ConditionID:   token.ConditionID,
			Outcome:       token.Outcome,
			AlertBand:     band,
			LastAlertedAt: alertedAt,
			LastPriceTs:   token.PriceTs,
			LastPrice:     token.Price,
		}); err != nil {
			if ctx.Err() == nil {
				log.WithError(err).
					WithField("token_id", token.TokenID).
					WithField("condition_id", token.ConditionID).
					Warn("failed to update polymarket sports live price alert state")
			}
		}
	}
}

func (s *Service) clearSportsLivePriceAlertState(ctx context.Context, token polymarketstore.SportsLivePriceAlertToken) {
	if token.LastAlertedAt.IsZero() && strings.TrimSpace(token.LastAlertBand) == "" {
		return
	}
	if err := s.store.DeleteSportsLivePriceAlertState(ctx, token.TokenID); err != nil && ctx.Err() == nil {
		log.WithError(err).
			WithField("token_id", token.TokenID).
			WithField("condition_id", token.ConditionID).
			Warn("failed to clear polymarket sports live price alert state")
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

func shouldSendSportsLivePriceAlert(token polymarketstore.SportsLivePriceAlertToken, band string, config SportsLivePriceAlertsConfig, now time.Time) bool {
	lastBand := strings.TrimSpace(token.LastAlertBand)
	if token.LastAlertedAt.IsZero() || lastBand == "" {
		return true
	}
	if sportsLivePriceAlertBandRank(band) > sportsLivePriceAlertBandRank(lastBand) {
		return true
	}
	if band == lastBand && !now.Before(token.LastAlertedAt.UTC().Add(config.Cooldown)) {
		return true
	}
	return false
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

func sportsLivePriceAlertBandRank(band string) int {
	switch band {
	case sportsLivePriceAlertBandD:
		return 3
	case sportsLivePriceAlertBandC:
		return 2
	case sportsLivePriceAlertBandB:
		return 1
	default:
		return 0
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
