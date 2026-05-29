package reconcile

import (
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/model"
	"github.com/useryege/athena/internal/application/persistence"
	"github.com/useryege/athena/internal/application/pipeline"
	"github.com/useryege/athena/internal/application/policy"
	"github.com/useryege/athena/internal/application/simulate"
)

type Project = model.Project
type ProjectMeta = model.ProjectMeta
type ProjectReport = model.ProjectReport
type SimulateResult = model.SimulateResult
type GenesisWalletMeta = model.GenesisWalletMeta
type ProjectAveDetail = model.ProjectAveDetail
type ProjectAveTokenDetail = model.ProjectAveTokenDetail
type ProjectAvePair = model.ProjectAvePair
type DiscoveredProjectCandidate = model.DiscoveredProjectCandidate
type ProjectDiscoverySource = model.ProjectDiscoverySource
type ProjectSnapshotCache = appcache.ProjectSnapshotCache
type ProjectComponentCache = appcache.ProjectComponentCache
type ProjectUpdater = appcache.ProjectUpdater
type ProjectSimulator = simulate.ProjectSimulator
type PersistenceEventPublisher = persistence.PersistenceEventPublisher
type PersistenceEvent = persistence.PersistenceEvent
type ProjectPolicyEngine = policy.Engine
type ProjectStateReconciler = pipeline.ProjectStateReconciler

const (
	ProjectDiscoverySourceCatchUp     = model.ProjectDiscoverySourceCatchUp
	ProjectDiscoverySourceFollowHeads = model.ProjectDiscoverySourceFollowHeads
	ProjectDiscoverySourcePairSwap    = model.ProjectDiscoverySourcePairSwap
)
