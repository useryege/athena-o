package application

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
)

const projectEventSubscriberBuffer = 128

var (
	projectEventsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "athena",
		Name:      "project_events_total",
		Help:      "Total number of project events published.",
	})
	projectSSEClients = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "athena",
		Name:      "project_sse_clients",
		Help:      "Current number of project SSE subscribers.",
	})
	projectSSEDroppedEventsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "athena",
		Name:      "project_sse_dropped_events_total",
		Help:      "Total number of project events dropped for slow SSE subscribers.",
	})
	projectSnapshotLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "athena",
		Name:      "project_snapshot_latency_ms",
		Help:      "Latency of project snapshot reads in milliseconds.",
		Buckets:   []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000},
	})
)

func init() {
	prometheus.MustRegister(
		projectEventsTotal,
		projectSSEClients,
		projectSSEDroppedEventsTotal,
		projectSnapshotLatency,
	)
}

type ProjectEventHub struct {
	mu          sync.RWMutex
	nextID      uint64
	nextSeq     atomic.Uint64
	subscribers map[uint64]chan *applicationpkg.ProjectEvent
}

func NewProjectEventHub() *ProjectEventHub {
	return &ProjectEventHub{
		subscribers: make(map[uint64]chan *applicationpkg.ProjectEvent),
	}
}

func (h *ProjectEventHub) Subscribe() (<-chan *applicationpkg.ProjectEvent, func()) {
	ch := make(chan *applicationpkg.ProjectEvent, projectEventSubscriberBuffer)

	h.mu.Lock()
	h.nextID++
	id := h.nextID
	h.subscribers[id] = ch
	h.mu.Unlock()

	projectSSEClients.Inc()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers, id)
			close(ch)
			h.mu.Unlock()
			projectSSEClients.Dec()
		})
	}

	return ch, unsubscribe
}

func (h *ProjectEventHub) PublishProjectCreated(project *Project) {
	event := &applicationpkg.ProjectEvent{
		Type:      applicationpkg.ProjectEventType_PROJECT_EVENT_TYPE_CREATED,
		EventSeq:  h.nextSeq.Add(1),
		EventTime: time.Now().UnixMilli(),
		Payload: &applicationpkg.ProjectEvent_Created{
			Created: &applicationpkg.ProjectCreatedEvent{
				Project: projectToView(project),
			},
		},
	}
	h.publish(event)
}

func (h *ProjectEventHub) publish(event *applicationpkg.ProjectEvent) {
	projectEventsTotal.Inc()

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			projectSSEDroppedEventsTotal.Inc()
		}
	}
}
