package polymarket

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
)

const (
	moverAlertSource           = "polymarket.movers"
	moverAlertSeverityWarning  = "warning"
	moverAlertSeverityCritical = "critical"
	moverAlertBaseURL          = "https://polymarket.com/event/"

	defaultMoverAlertWarningScore       = 6
	defaultMoverAlertCriticalScore      = 12
	defaultMoverAlertWarning1mPP        = 3
	defaultMoverAlertWarning5mPP        = 5
	defaultMoverAlertWarning15mPP       = 8
	defaultMoverAlertCritical1mPP       = 8
	defaultMoverAlertCritical5mPP       = 12
	defaultMoverAlertMinVolume24hr      = 10000
	defaultMoverAlertCooldown           = 15 * time.Minute
	defaultMoverAlertMaxPerRefresh      = 3
	defaultMoverAlertSendTimeout        = 10 * time.Second
	defaultMoverAlertQuestionMaxRunes   = 120
	defaultMoverAlertTitleQuestionRunes = 80
)

type MoverAlertsConfig struct {
	Enabled       bool
	WarningScore  float64
	CriticalScore float64
	Warning1mPP   float64
	Warning5mPP   float64
	Warning15mPP  float64
	Critical1mPP  float64
	Critical5mPP  float64
	MinVolume24hr float64
	Cooldown      time.Duration
	MaxPerRefresh int
	SendTimeout   time.Duration
}

type moverAlertState struct {
	LastSentAt int64
	Severity   string
	Score      float64
}

type moverAlertCandidate struct {
	key       string
	severity  string
	score     float64
	change1m  float64
	request   *notificationapiclient.SendNotificationRequest
	condition string
}

func defaultMoverAlertsConfig() MoverAlertsConfig {
	return MoverAlertsConfig{
		WarningScore:  defaultMoverAlertWarningScore,
		CriticalScore: defaultMoverAlertCriticalScore,
		Warning1mPP:   defaultMoverAlertWarning1mPP,
		Warning5mPP:   defaultMoverAlertWarning5mPP,
		Warning15mPP:  defaultMoverAlertWarning15mPP,
		Critical1mPP:  defaultMoverAlertCritical1mPP,
		Critical5mPP:  defaultMoverAlertCritical5mPP,
		MinVolume24hr: defaultMoverAlertMinVolume24hr,
		Cooldown:      defaultMoverAlertCooldown,
		MaxPerRefresh: defaultMoverAlertMaxPerRefresh,
		SendTimeout:   defaultMoverAlertSendTimeout,
	}
}

func normalizeMoverAlertsConfig(config MoverAlertsConfig) MoverAlertsConfig {
	defaults := defaultMoverAlertsConfig()
	if config.WarningScore <= 0 {
		config.WarningScore = defaults.WarningScore
	}
	if config.CriticalScore <= 0 {
		config.CriticalScore = defaults.CriticalScore
	}
	if config.Warning1mPP <= 0 {
		config.Warning1mPP = defaults.Warning1mPP
	}
	if config.Warning5mPP <= 0 {
		config.Warning5mPP = defaults.Warning5mPP
	}
	if config.Warning15mPP <= 0 {
		config.Warning15mPP = defaults.Warning15mPP
	}
	if config.Critical1mPP <= 0 {
		config.Critical1mPP = defaults.Critical1mPP
	}
	if config.Critical5mPP <= 0 {
		config.Critical5mPP = defaults.Critical5mPP
	}
	if config.MinVolume24hr <= 0 {
		config.MinVolume24hr = defaults.MinVolume24hr
	}
	if config.Cooldown <= 0 {
		config.Cooldown = defaults.Cooldown
	}
	if config.MaxPerRefresh <= 0 {
		config.MaxPerRefresh = defaults.MaxPerRefresh
	}
	if config.SendTimeout <= 0 {
		config.SendTimeout = defaults.SendTimeout
	}
	return config
}

func (s *Service) collectMoverAlertsLocked(nowUnix int64) []moverAlertCandidate {
	config := normalizeMoverAlertsConfig(s.moverAlertsConfig)
	if !config.Enabled || s.notificationClientset == nil {
		return nil
	}
	if nowUnix <= 0 {
		nowUnix = s.nowUnix()
	}

	markets := realtimeTopHotMarketsLocked(s.hotMarketItems)
	if len(markets) == 0 {
		return nil
	}
	s.ensureRealtimeStatesForMarketsLocked(markets)

	candidates := make([]moverAlertCandidate, 0, config.MaxPerRefresh)
	for _, market := range markets {
		item := s.moverMarketItemLocked(market, nowUnix)
		candidate, ok := s.moverAlertCandidateLocked(item, config, nowUnix)
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if moverAlertSeverityRank(left.severity) != moverAlertSeverityRank(right.severity) {
			return moverAlertSeverityRank(left.severity) > moverAlertSeverityRank(right.severity)
		}
		if left.score != right.score {
			return left.score > right.score
		}
		if left.change1m != right.change1m {
			return left.change1m > right.change1m
		}
		return left.condition < right.condition
	})
	if len(candidates) > config.MaxPerRefresh {
		candidates = candidates[:config.MaxPerRefresh]
	}

	for _, candidate := range candidates {
		s.moverAlertStates[candidate.key] = moverAlertState{
			LastSentAt: nowUnix,
			Severity:   candidate.severity,
			Score:      candidate.score,
		}
	}
	return candidates
}

func (s *Service) moverAlertCandidateLocked(item *v1alpha1.PolymarketMoverMarketItem, config MoverAlertsConfig, nowUnix int64) (moverAlertCandidate, bool) {
	if item == nil || item.Leader == nil {
		return moverAlertCandidate{}, false
	}
	leader := item.Leader
	if leader.Warmup || item.Volume24hr < config.MinVolume24hr {
		return moverAlertCandidate{}, false
	}
	severity := moverAlertSeverity(item, config)
	if severity == "" {
		return moverAlertCandidate{}, false
	}

	key := moverAlertKey(item, leader)
	if key == "" || !s.shouldSendMoverAlertLocked(key, severity, config, nowUnix) {
		return moverAlertCandidate{}, false
	}

	return moverAlertCandidate{
		key:       key,
		severity:  severity,
		score:     item.Score,
		change1m:  moverWindowAbsChange(leader, "1m"),
		request:   renderMoverAlertNotification(item, severity),
		condition: item.ConditionID,
	}, true
}

func (s *Service) shouldSendMoverAlertLocked(key, severity string, config MoverAlertsConfig, nowUnix int64) bool {
	state, exists := s.moverAlertStates[key]
	if !exists || state.LastSentAt <= 0 {
		return true
	}
	if severity == moverAlertSeverityCritical && state.Severity == moverAlertSeverityWarning {
		return true
	}
	return nowUnix-state.LastSentAt >= int64(config.Cooldown.Seconds())
}

func moverAlertSeverity(item *v1alpha1.PolymarketMoverMarketItem, config MoverAlertsConfig) string {
	leader := item.Leader
	if leader == nil {
		return ""
	}
	change1m := moverWindowAbsChange(leader, "1m")
	change5m := moverWindowAbsChange(leader, "5m")
	change15m := moverWindowAbsChange(leader, "15m")
	if item.Score >= config.CriticalScore || change1m >= config.Critical1mPP || change5m >= config.Critical5mPP {
		return moverAlertSeverityCritical
	}
	if item.Score >= config.WarningScore && (change1m >= config.Warning1mPP || change5m >= config.Warning5mPP || change15m >= config.Warning15mPP) {
		return moverAlertSeverityWarning
	}
	return ""
}

func moverAlertKey(item *v1alpha1.PolymarketMoverMarketItem, leader *v1alpha1.PolymarketMoverTokenItem) string {
	conditionID := strings.TrimSpace(item.ConditionID)
	tokenID := strings.TrimSpace(leader.TokenID)
	direction := strings.TrimSpace(leader.Direction)
	if conditionID == "" || tokenID == "" || direction == "" {
		return ""
	}
	return conditionID + ":" + tokenID + ":" + direction
}

func renderMoverAlertNotification(item *v1alpha1.PolymarketMoverMarketItem, severity string) *notificationapiclient.SendNotificationRequest {
	leader := item.Leader
	title := fmt.Sprintf("Polymarket mover %s %s %.1f%%: %s",
		strings.ToUpper(leader.Direction),
		firstNonEmpty(leader.Outcome, "Outcome"),
		leader.Price*100,
		truncateRunes(item.Question, defaultMoverAlertTitleQuestionRunes),
	)
	body := strings.Join([]string{
		fmt.Sprintf("Question: %s", truncateRunes(item.Question, defaultMoverAlertQuestionMaxRunes)),
		fmt.Sprintf("Outcome: %s", firstNonEmpty(leader.Outcome, "Outcome")),
		fmt.Sprintf("Direction: %s", strings.ToUpper(leader.Direction)),
		fmt.Sprintf("Price: %.1f%%", leader.Price*100),
		fmt.Sprintf("Score: %.2f", item.Score),
		fmt.Sprintf("Move: 1m %s | 5m %s | 15m %s",
			formatMoverSignedPP(moverWindowChange(leader, "1m")),
			formatMoverSignedPP(moverWindowChange(leader, "5m")),
			formatMoverSignedPP(moverWindowChange(leader, "15m")),
		),
		fmt.Sprintf("24h volume: %.2f", item.Volume24hr),
		fmt.Sprintf("Liquidity: %.2f", item.LiquidityNum),
		fmt.Sprintf("Condition ID: %s", item.ConditionID),
		fmt.Sprintf("Token ID: %s", leader.TokenID),
	}, "\n")

	return &notificationapiclient.SendNotificationRequest{
		Source:   moverAlertSource,
		Severity: notificationSeverityForMoverAlert(severity),
		Title:    title,
		Body:     body,
		Link:     polymarketMoverLink(item),
		Topic:    "poly",
	}
}

func (s *Service) sendMoverAlerts(ctx context.Context, alerts []moverAlertCandidate) {
	if len(alerts) == 0 || s.notificationClientset == nil {
		return
	}
	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		log.WithError(err).Warn("failed to create polymarket mover notification client")
		return
	}
	defer utilio.Close(closer)

	config := normalizeMoverAlertsConfig(s.moverAlertsConfig)
	for _, alert := range alerts {
		if alert.request == nil {
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, config.SendTimeout)
		_, err := client.SendNotification(sendCtx, alert.request)
		cancel()
		if err != nil {
			log.WithError(err).WithField("condition_id", alert.condition).Warn("failed to send polymarket mover notification")
		}
	}
}

func notificationSeverityForMoverAlert(severity string) notificationapiclient.NotificationSeverity {
	if severity == moverAlertSeverityCritical {
		return notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL
	}
	return notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING
}

func moverAlertSeverityRank(severity string) int {
	switch severity {
	case moverAlertSeverityCritical:
		return 2
	case moverAlertSeverityWarning:
		return 1
	default:
		return 0
	}
}

func moverWindowChange(token *v1alpha1.PolymarketMoverTokenItem, label string) float64 {
	if token == nil {
		return 0
	}
	for _, window := range token.Windows {
		if window != nil && window.Window == label && !window.Warmup {
			return window.PriceChangePp
		}
	}
	return 0
}

func formatMoverSignedPP(value float64) string {
	if value == 0 || math.Abs(value) < 0.005 {
		return "0.00pp"
	}
	return fmt.Sprintf("%+.2fpp", value)
}

func polymarketMoverLink(item *v1alpha1.PolymarketMoverMarketItem) string {
	slug := firstNonEmpty(item.EventSlug, item.MarketSlug)
	if slug == "" {
		return ""
	}
	return moverAlertBaseURL + url.PathEscape(slug)
}

func truncateRunes(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}
