package discovery

import (
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/model"
	"github.com/useryege/athena/internal/application/pipeline"
)

type Project = model.Project
type ProjectMeta = model.ProjectMeta
type DiscoveredProjectCandidate = model.DiscoveredProjectCandidate
type ProjectDiscoverySource = model.ProjectDiscoverySource
type ProjectSnapshotCache = appcache.ProjectSnapshotCache
type ProjectComponentCache = appcache.ProjectComponentCache
type DiscoveryIntake = pipeline.DiscoveryIntake
type ProjectStateReconciler = pipeline.ProjectStateReconciler
type ProjectDiscoveryIndexer = pipeline.ProjectDiscoveryIndexer

const (
	ProjectDiscoverySourceCatchUp     = model.ProjectDiscoverySourceCatchUp
	ProjectDiscoverySourceFollowHeads = model.ProjectDiscoverySourceFollowHeads
	ProjectDiscoverySourcePairSwap    = model.ProjectDiscoverySourcePairSwap
)
