package accountavatarhttp

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	defaultGarbageCollectionInterval = 24 * time.Hour
	defaultOrphanGracePeriod         = 24 * time.Hour
)

// RunGarbageCollector removes old object-store entries that are not referenced
// by any durable account profile. A newly uploaded candidate is protected by
// the grace period while its database CAS is in flight.
func (h *Handler) RunGarbageCollector(ctx context.Context, interval, gracePeriod time.Duration) {
	if interval <= 0 {
		interval = defaultGarbageCollectionInterval
	}
	if gracePeriod <= 0 {
		gracePeriod = defaultOrphanGracePeriod
	}

	h.collectGarbage(ctx, gracePeriod)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.collectGarbage(ctx, gracePeriod)
		}
	}
}

func (h *Handler) collectGarbage(ctx context.Context, gracePeriod time.Duration) {
	referencedKeys, err := h.profiles.ListAvatarObjectKeys(ctx)
	if err != nil {
		h.log.WithError(err).Warn("account avatar garbage collection could not load profile references")
		return
	}
	referenced := make(map[string]struct{}, len(referencedKeys))
	for _, key := range referencedKeys {
		referenced[key] = struct{}{}
	}

	objects, err := h.objects.List(ctx, objectPrefix)
	if err != nil {
		h.log.WithError(err).Warn("account avatar garbage collection could not list objects")
		return
	}
	deleteBefore := time.Now().Add(-gracePeriod)
	deleted := 0
	for _, object := range objects {
		if _, ok := referenced[object.Key]; ok || object.LastModified.IsZero() || object.LastModified.After(deleteBefore) {
			continue
		}
		if err := h.objects.Delete(ctx, object.Key); err != nil {
			h.log.WithError(err).WithField("object_key", object.Key).Warn("account avatar garbage collection failed to delete orphan")
			continue
		}
		deleted++
	}
	if deleted > 0 {
		h.log.WithFields(log.Fields{"deleted": deleted, "objects_scanned": len(objects)}).Info("account avatar garbage collection completed")
	}
}
