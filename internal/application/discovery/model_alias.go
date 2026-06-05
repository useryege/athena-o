package discovery

import (
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/model"
)

type Project = model.Project
type ProjectMeta = model.ProjectMeta
type DiscoveredProjectCandidate = model.DiscoveredProjectCandidate
type ProjectDiscoverySource = model.ProjectDiscoverySource
type ProjectComponentCache = appcache.ProjectComponentCache

const (
	ProjectDiscoverySourceCatchUp     = model.ProjectDiscoverySourceCatchUp
	ProjectDiscoverySourceFollowHeads = model.ProjectDiscoverySourceFollowHeads
	ProjectDiscoverySourcePairSwap    = model.ProjectDiscoverySourcePairSwap
)
