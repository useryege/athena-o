package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
)

const projectPolicyEvaluationInterval = time.Minute

type ProjectPolicyFacts struct {
	SourceCodeBlacklistFields []string
	BytecodeBlacklist         map[common.Hash]struct{}
}

type ProjectPolicyRule interface {
	Name() string
	Evaluate(ctx context.Context, project *Project, facts ProjectPolicyFacts) (match bool, evidence map[string]any, err error)
}

type projectPolicyEngineImpl struct {
	projectCache ProjectSnapshotCache

	sourceAnalyzer    sourcecode.Analyzer
	sourceBlacklist   sourceCodeBlacklistLister
	bytecodeBlacklist bytecodeBlacklistLister

	persistencePublisher PersistenceEventPublisher
	rules                []ProjectPolicyRule

	wg sync.WaitGroup
}

type sourceCodeBlacklistLister interface {
	List(ctx context.Context) ([]string, error)
}

type bytecodeBlacklistLister interface {
	List(ctx context.Context) ([]appstore.BytecodeBlacklistContract, error)
}

func NewProjectPolicyEngine(
	projectCache ProjectSnapshotCache,
	sourceAnalyzer sourcecode.Analyzer,
	sourceBlacklist sourceCodeBlacklistLister,
	bytecodeBlacklist bytecodeBlacklistLister,
	persistencePublisher PersistenceEventPublisher,
) ProjectPolicyEngine {
	return &projectPolicyEngineImpl{
		projectCache:         projectCache,
		sourceAnalyzer:       sourceAnalyzer,
		sourceBlacklist:      sourceBlacklist,
		bytecodeBlacklist:    bytecodeBlacklist,
		persistencePublisher: persistencePublisher,
		rules: []ProjectPolicyRule{
			sourceCodeBlacklistRule{},
			bytecodeBlacklistRule{},
		},
	}
}

func (e *projectPolicyEngineImpl) Start(ctx context.Context) error {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.runLoop(ctx)
	}()
	return nil
}

func (e *projectPolicyEngineImpl) Stop() error {
	e.wg.Wait()
	return nil
}

func (e *projectPolicyEngineImpl) runLoop(ctx context.Context) {
	ticker := time.NewTicker(projectPolicyEvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.evaluateAllProjects(ctx); err != nil {
				log.WithFields(log.Fields{
					"component": "project_policy_engine",
					"error":     err.Error(),
				}).Warn("project policy evaluation failed")
			}
		}
	}
}

func (e *projectPolicyEngineImpl) evaluateAllProjects(ctx context.Context) error {
	facts, err := e.buildFacts(ctx)
	if err != nil {
		return err
	}

	projects, err := e.listAllProjects(ctx)
	if err != nil {
		return err
	}

	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil {
			continue
		}
		if err := e.analyzeSourceCodeIfNeeded(ctx, project, facts.SourceCodeBlacklistFields); err != nil {
			continue
		}
		if err := e.evaluateRulesForProject(ctx, project, facts); err != nil {
			continue
		}
	}
	return nil
}

func (e *projectPolicyEngineImpl) buildFacts(ctx context.Context) (ProjectPolicyFacts, error) {
	facts := ProjectPolicyFacts{}
	if e.sourceBlacklist != nil {
		fields, err := e.sourceBlacklist.List(ctx)
		if err != nil {
			return facts, err
		}
		facts.SourceCodeBlacklistFields = fields
	}

	if e.bytecodeBlacklist != nil {
		records, err := e.bytecodeBlacklist.List(ctx)
		if err != nil {
			return facts, err
		}
		facts.BytecodeBlacklist = make(map[common.Hash]struct{}, len(records))
		for _, item := range records {
			facts.BytecodeBlacklist[item.CodeHash] = struct{}{}
		}
	}
	return facts, nil
}

func (e *projectPolicyEngineImpl) listAllProjects(ctx context.Context) ([]*Project, error) {
	activeProjects, err := e.projectCache.ListActiveProjects(ctx)
	if err != nil {
		return nil, err
	}

	archivedProjects := make([]*Project, 0)
	page := int32(1)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		items, total, _, pageSize, err := e.projectCache.ListArchivedProjects(ctx, page, sourceCodeScanPageSize)
		if err != nil {
			return nil, err
		}
		archivedProjects = append(archivedProjects, items...)
		if len(items) == 0 || int64(page)*int64(pageSize) >= total {
			break
		}
		page++
	}

	all := make([]*Project, 0, len(activeProjects)+len(archivedProjects))
	seen := make(map[common.Address]struct{}, len(activeProjects)+len(archivedProjects))
	appendUnique := func(items []*Project) {
		for _, project := range items {
			if project == nil {
				continue
			}
			contract := project.Meta.Contract
			if _, ok := seen[contract]; ok {
				continue
			}
			seen[contract] = struct{}{}
			all = append(all, project)
		}
	}
	appendUnique(activeProjects)
	appendUnique(archivedProjects)
	return all, nil
}

func (e *projectPolicyEngineImpl) analyzeSourceCodeIfNeeded(ctx context.Context, project *Project, fields []string) error {
	if project == nil || project.Meta.SourceCode == "" || e.sourceAnalyzer == nil || !project.Runtime.SourceCodeBlacklist.ResolvedAt.IsZero() {
		return nil
	}
	report := e.sourceAnalyzer.AnalyzeSourceCode(project.Meta.SourceCode, fields)
	_, err := e.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil || current.Meta.SourceCode == "" || !current.Runtime.SourceCodeBlacklist.ResolvedAt.IsZero() {
			return nil, false, nil
		}
		current.Runtime.SourceCodeBlacklist = report
		return current, true, nil
	})
	if err != nil {
		return err
	}
	project.Runtime.SourceCodeBlacklist = report
	return nil
}

func (e *projectPolicyEngineImpl) evaluateRulesForProject(ctx context.Context, project *Project, facts ProjectPolicyFacts) error {
	if project == nil {
		return nil
	}

	for _, rule := range e.rules {
		if rule == nil {
			continue
		}
		match, evidence, err := rule.Evaluate(ctx, project, facts)
		if err != nil {
			continue
		}
		if !match {
			continue
		}

		now := time.Now().UTC()
		if !project.Meta.IsArchived {
			if err := e.archiveProjectByPolicy(ctx, project, rule.Name(), evidence, now); err != nil {
				continue
			}
			project.Meta.IsArchived = true
			project.Meta.ArchivedAt = now
			continue
		}
		if err := e.persistPolicyAuditEvent(ctx, project.Meta.Contract, rule.Name(), evidence, now); err != nil {
			continue
		}
	}
	return nil
}

func (e *projectPolicyEngineImpl) archiveProjectByPolicy(ctx context.Context, project *Project, ruleName string, evidence map[string]any, now time.Time) error {
	if project == nil || e.persistencePublisher == nil {
		return nil
	}
	if err := e.persistencePublisher.PublishProjectArchive(ctx, project.Meta.Contract); err != nil {
		return err
	}
	_, err := e.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil || current.Meta.IsArchived {
			return nil, false, nil
		}
		current.Meta.IsArchived = true
		current.Meta.ArchivedAt = now
		return current, true, nil
	})
	if err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]any{
		"rule":     ruleName,
		"evidence": evidence,
		"source":   "policy_engine",
	})

	return e.persistencePublisher.PublishProjectEventLog(ctx, appstore.ProjectEventLog{
		Contract:       project.Meta.Contract,
		EventType:      projectEventTypeAutoArchivePolicy,
		OccurredAt:     now,
		Message:        fmt.Sprintf("Project auto archived by policy rule %s", ruleName),
		Payload:        string(payload),
		IdempotencyKey: projectEventIdempotencyAutoArchivePolicy(ruleName),
	})
}

func (e *projectPolicyEngineImpl) persistPolicyAuditEvent(ctx context.Context, contract common.Address, ruleName string, evidence map[string]any, now time.Time) error {
	if e.persistencePublisher == nil {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"rule":     ruleName,
		"evidence": evidence,
		"source":   "policy_engine",
	})
	return e.persistencePublisher.PublishProjectEventLog(ctx, appstore.ProjectEventLog{
		Contract:       contract,
		EventType:      projectEventTypePolicyMatchAudit,
		OccurredAt:     now,
		Message:        fmt.Sprintf("Policy rule %s matched archived project", ruleName),
		Payload:        string(payload),
		IdempotencyKey: projectEventIdempotencyPolicyMatch(ruleName),
	})
}

type sourceCodeBlacklistRule struct{}

func (r sourceCodeBlacklistRule) Name() string { return "sourcecode_blacklist" }

func (r sourceCodeBlacklistRule) Evaluate(_ context.Context, project *Project, _ ProjectPolicyFacts) (bool, map[string]any, error) {
	if project == nil || project.Meta.SourceCode == "" {
		return false, nil, nil
	}
	report := project.Runtime.SourceCodeBlacklist
	if !report.HasBlacklistFields {
		return false, nil, nil
	}
	return true, map[string]any{
		"blacklist_fields": report.BlacklistFields,
	}, nil
}

type bytecodeBlacklistRule struct{}

func (r bytecodeBlacklistRule) Name() string { return "bytecode_blacklist" }

func (r bytecodeBlacklistRule) Evaluate(_ context.Context, project *Project, facts ProjectPolicyFacts) (bool, map[string]any, error) {
	if project == nil || project.Runtime.RuntimeCodeHash == (common.Hash{}) || len(facts.BytecodeBlacklist) == 0 {
		return false, nil, nil
	}
	if _, ok := facts.BytecodeBlacklist[project.Runtime.RuntimeCodeHash]; !ok {
		return false, nil, nil
	}
	return true, map[string]any{
		"runtime_code_hash": strings.ToLower(project.Runtime.RuntimeCodeHash.Hex()),
	}, nil
}
