package application

import "context"

type ProjectMetaStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
}
