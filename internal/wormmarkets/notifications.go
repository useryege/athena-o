package wormmarkets

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	wormmarketsstore "github.com/useryege/athena/internal/wormmarkets/store"
	utilio "github.com/useryege/athena/util/io"
)

const (
	wormNewEventNotificationTopic   = "[WORM] 新比赛"
	wormLiveEventNotificationTopic  = "[WORM] 开赛通知"
	wormPriceAlert80Topic           = "[WORM] 80/20 赔率"
	wormPriceAlert90Topic           = "[WORM] 90/10 赔率"
	wormPriceAlert95Topic           = "[WORM] 95/5 赔率"
	wormNewEventNotificationSource  = "worm-markets.new-event"
	wormLiveEventNotificationSource = "worm-markets.live-event"
	wormPriceAlert80Source          = "worm-markets.price-alert-80-20"
	wormPriceAlert90Source          = "worm-markets.price-alert-90-10"
	wormPriceAlert95Source          = "worm-markets.price-alert-95-5"
	wormNotificationSendTimeout     = 10 * time.Second

	wormPriceAlertBandNone = "none"
	wormPriceAlertBandA    = "a"
	wormPriceAlertBandB    = "b"
	wormPriceAlertBandC    = "c"
)

type wormNotification struct {
	eventConditionID string
	request          *notificationapiclient.SendNotificationRequest
}

type wormNotificationResult struct {
	notification wormNotification
	err          error
}

func newWormEventNotifications(events map[string]wormmarketsstore.WormMarket) []wormNotification {
	eventConditionIDs := make([]string, 0, len(events))
	for eventConditionID := range events {
		eventConditionIDs = append(eventConditionIDs, eventConditionID)
	}
	sort.Strings(eventConditionIDs)

	notifications := make([]wormNotification, 0, len(eventConditionIDs))
	for _, eventConditionID := range eventConditionIDs {
		market := events[eventConditionID]
		eventTitle := firstNonEmptyWormValue(market.EventTitle, market.Title, eventConditionID)
		body := strings.Join([]string{
			fmt.Sprintf("Event: %s", eventTitle),
			fmt.Sprintf("Event condition ID: %s", eventConditionID),
			fmt.Sprintf("Market: %s", firstNonEmptyWormValue(market.Title, market.ConditionID)),
			fmt.Sprintf("Market condition ID: %s", market.ConditionID),
		}, "\n")
		notifications = append(notifications, wormNotification{
			eventConditionID: eventConditionID,
			request: &notificationapiclient.SendNotificationRequest{
				Source:       wormNewEventNotificationSource,
				Severity:     notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
				Title:        fmt.Sprintf("Worm new event: %s", eventTitle),
				Body:         body,
				Link:         wormMarketLink(market.ConditionID),
				TelegramChat: notificationapiclient.TelegramChat_TELEGRAM_CHAT_TEST,
				TopicLabel:   wormNewEventNotificationTopic,
			},
		})
	}
	return notifications
}

func newWormLiveNotification(change wormmarketsstore.WormMarketLivePriceChange) wormNotification {
	eventTitle := firstNonEmptyWormValue(change.EventTitle, change.Title, change.EventConditionID)
	body := strings.Join([]string{
		fmt.Sprintf("Event: %s", eventTitle),
		fmt.Sprintf("Event condition ID: %s", change.EventConditionID),
		fmt.Sprintf("Market: %s", firstNonEmptyWormValue(change.Title, change.ConditionID)),
		fmt.Sprintf("Market condition ID: %s", change.ConditionID),
		fmt.Sprintf("Live price change: %s", firstNonEmptyWormValue(change.PriceChange, "-")),
	}, "\n")
	return wormNotification{
		eventConditionID: change.EventConditionID,
		request: &notificationapiclient.SendNotificationRequest{
			Source:       wormLiveEventNotificationSource,
			Severity:     notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
			Title:        fmt.Sprintf("Worm event is live: %s", eventTitle),
			Body:         body,
			Link:         wormMarketLink(change.ConditionID),
			TelegramChat: notificationapiclient.TelegramChat_TELEGRAM_CHAT_TEST,
			TopicLabel:   wormLiveEventNotificationTopic,
		},
	}
}

func newWormPriceAlertNotification(market wormmarketsstore.WormMarket, band string) wormNotification {
	eventTitle := firstNonEmptyWormValue(market.EventTitle, market.Title, market.EventConditionID)
	marketTitle := firstNonEmptyWormValue(market.Title, market.ConditionID)
	topic := wormPriceAlert80Topic
	source := wormPriceAlert80Source
	alertBand := "80/20 ([0.1, 0.2] or [0.8, 0.9))"
	severity := notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING
	titlePrefix := "Worm 80/20 price alert"
	switch band {
	case wormPriceAlertBandB:
		topic = wormPriceAlert90Topic
		source = wormPriceAlert90Source
		alertBand = "90/10 ((0.05, 0.1) or [0.9, 0.95))"
		titlePrefix = "Worm 90/10 price alert"
	case wormPriceAlertBandC:
		topic = wormPriceAlert95Topic
		source = wormPriceAlert95Source
		alertBand = "95/5 ([0, 0.05] or [0.95, 1])"
		severity = notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL
		titlePrefix = "Worm 95/5 price alert"
	}
	body := strings.Join([]string{
		fmt.Sprintf("Event: %s", eventTitle),
		fmt.Sprintf("Market: %s", marketTitle),
		fmt.Sprintf("Last trade price: %s", firstNonEmptyWormValue(market.LastTradePrice, "-")),
		fmt.Sprintf("Alert band: %s", alertBand),
		fmt.Sprintf("Event condition ID: %s", firstNonEmptyWormValue(market.EventConditionID, "-")),
		fmt.Sprintf("Market condition ID: %s", market.ConditionID),
	}, "\n")
	return wormNotification{
		eventConditionID: market.EventConditionID,
		request: &notificationapiclient.SendNotificationRequest{
			Source:       source,
			Severity:     severity,
			Title:        fmt.Sprintf("%s: %s", titlePrefix, marketTitle),
			Body:         body,
			Link:         wormMarketLink(market.ConditionID),
			TelegramChat: notificationapiclient.TelegramChat_TELEGRAM_CHAT_TEST,
			TopicLabel:   topic,
		},
	}
}

func wormMarketLink(conditionID string) string {
	conditionID = strings.TrimSpace(conditionID)
	if conditionID == "" {
		return ""
	}
	return "https://www.worm.wtf/market/" + url.PathEscape(conditionID)
}

func (s *Service) sendWormNotifications(ctx context.Context, notifications []wormNotification) []wormNotificationResult {
	if len(notifications) == 0 {
		return nil
	}
	results := make([]wormNotificationResult, 0, len(notifications))
	if s.notificationClientset == nil {
		for _, item := range notifications {
			results = append(results, wormNotificationResult{
				notification: item,
				err:          fmt.Errorf("notification clientset is not configured"),
			})
		}
		return results
	}
	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		log.WithError(err).Warn("failed to create worm notification client")
		for _, item := range notifications {
			results = append(results, wormNotificationResult{notification: item, err: err})
		}
		return results
	}
	defer utilio.Close(closer)

	for _, item := range notifications {
		result := wormNotificationResult{notification: item}
		if item.request == nil {
			result.err = fmt.Errorf("notification request is nil")
			results = append(results, result)
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, wormNotificationSendTimeout)
		_, result.err = client.SendNotification(sendCtx, item.request)
		cancel()
		if result.err != nil {
			log.WithError(result.err).
				WithField("event_condition_id", item.eventConditionID).
				Warn("failed to send worm notification")
		}
		results = append(results, result)
	}
	return results
}

func firstNonEmptyWormValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "-"
}
