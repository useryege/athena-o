package api

import (
	appstore "github.com/useryege/athena/internal/application/store"
)

func projectMetaToStore(meta ProjectMeta) appstore.ProjectMeta {
	return meta
}

func projectMetaFromStore(meta appstore.ProjectMeta) ProjectMeta {
	return meta
}
