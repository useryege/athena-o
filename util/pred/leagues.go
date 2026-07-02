package pred

import (
	"context"
	"encoding/json"
	"net/http"
)

func (c *clientImpl) GetLeagueCatalog(ctx context.Context) (json.RawMessage, error) {
	return doRaw(ctx, c, http.MethodGet, "/leagues/catalog", nil, nil, false)
}

func (c *clientImpl) RefreshLeagueCatalog(ctx context.Context) (json.RawMessage, error) {
	return doRaw(ctx, c, http.MethodPost, "/leagues/catalog/refresh", nil, nil, true)
}
