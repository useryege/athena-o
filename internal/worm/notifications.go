package worm

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/notification"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	wormstore "github.com/useryege/athena/internal/worm/store"
	utilio "github.com/useryege/athena/util/io"
)

const (
	wormNewEventNotificationSource  = "worm.new-event"
	wormLiveEventNotificationSource = "worm.live-event"
	wormNotificationSendTimeout     = 10 * time.Second
)

type wormNotification struct {
	eventConditionID string
	request          *notificationapiclient.SendNotificationRequest
}

func newWormEventNotifications(events map[string]wormstore.WormMarket) []wormNotification {
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
				Source:   wormNewEventNotificationSource,
				Severity: notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
				Title:    fmt.Sprintf("Worm new event: %s", eventTitle),
				Body:     body,
				Topic:    notification.NotificationTopicWorm,
			},
		})
	}
	return notifications
}

func newWormLiveNotification(change wormstore.WormMarketLivePriceChange) wormNotification {
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
			Source:   wormLiveEventNotificationSource,
			Severity: notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
			Title:    fmt.Sprintf("Worm event is live: %s", eventTitle),
			Body:     body,
			Topic:    notification.NotificationTopicWorm,
		},
	}
}

func (s *Service) sendWormNotifications(ctx context.Context, notifications []wormNotification) {
	if len(notifications) == 0 || s.notificationClientset == nil {
		return
	}
	closer, client, err := s.notificationClientset.NewNotificationServiceClient()
	if err != nil {
		log.WithError(err).Warn("failed to create worm notification client")
		return
	}
	defer utilio.Close(closer)

	for _, item := range notifications {
		if item.request == nil {
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, wormNotificationSendTimeout)
		_, err := client.SendNotification(sendCtx, item.request)
		cancel()
		if err != nil {
			log.WithError(err).
				WithField("event_condition_id", item.eventConditionID).
				Warn("failed to send worm notification")
		}
	}
}

func firstNonEmptyWormValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "-"
}
