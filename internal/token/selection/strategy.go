package selection

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type Strategy interface {
	Key() string
	Version() string
	Evaluate(context.Context, StrategyInput) (SelectionDecision, error)
}

type StrategyRegistry struct {
	mu         sync.RWMutex
	strategies map[strategyIdentity]Strategy
	active     strategyIdentity
}

type strategyIdentity struct {
	key     string
	version string
}

func NewStrategyRegistry(activeKey, activeVersion string, strategies ...Strategy) (*StrategyRegistry, error) {
	registry := &StrategyRegistry{strategies: make(map[strategyIdentity]Strategy)}
	if err := registry.Register(NotConfiguredStrategy{}); err != nil {
		return nil, err
	}
	for _, strategy := range strategies {
		if err := registry.Register(strategy); err != nil {
			return nil, err
		}
	}
	active := strategyIdentity{key: strings.TrimSpace(activeKey), version: strings.TrimSpace(activeVersion)}
	if active.key == "" || active.version == "" {
		return nil, fmt.Errorf("token selection active strategy key and version are required")
	}
	if _, exists := registry.strategies[active]; !exists {
		return nil, fmt.Errorf("token selection strategy %s/%s is not registered", active.key, active.version)
	}
	registry.active = active
	return registry, nil
}

func (registry *StrategyRegistry) Register(strategy Strategy) error {
	if strategy == nil {
		return fmt.Errorf("token selection strategy is required")
	}
	identity := strategyIdentity{key: strings.TrimSpace(strategy.Key()), version: strings.TrimSpace(strategy.Version())}
	if identity.key == "" || identity.version == "" {
		return fmt.Errorf("token selection strategy key and version are required")
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.strategies[identity]; exists {
		return fmt.Errorf("token selection strategy %s/%s is already registered", identity.key, identity.version)
	}
	registry.strategies[identity] = strategy
	return nil
}

func (registry *StrategyRegistry) Active() (Strategy, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	strategy := registry.strategies[registry.active]
	if strategy == nil {
		return nil, fmt.Errorf("token selection strategy %s/%s is not registered", registry.active.key, registry.active.version)
	}
	return strategy, nil
}

type NotConfiguredStrategy struct{}

func (NotConfiguredStrategy) Key() string     { return "default" }
func (NotConfiguredStrategy) Version() string { return "1" }
func (NotConfiguredStrategy) Evaluate(context.Context, StrategyInput) (SelectionDecision, error) {
	return SelectionDecision{Outcome: SelectionOutcomeDeferred, ReasonCodes: []string{"strategy_not_configured"}}, nil
}

var reasonCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func NormalizeDecision(decision SelectionDecision) (SelectionDecision, error) {
	switch decision.Outcome {
	case SelectionOutcomeSelected, SelectionOutcomeRejected, SelectionOutcomeDeferred:
	default:
		return SelectionDecision{}, fmt.Errorf("token selection strategy returned invalid outcome %q", decision.Outcome)
	}
	seen := make(map[string]struct{}, len(decision.ReasonCodes))
	reasonCodes := make([]string, 0, len(decision.ReasonCodes))
	for _, reasonCode := range decision.ReasonCodes {
		reasonCode = strings.TrimSpace(reasonCode)
		if !reasonCodePattern.MatchString(reasonCode) {
			return SelectionDecision{}, fmt.Errorf("token selection strategy returned invalid reason code %q", reasonCode)
		}
		if _, exists := seen[reasonCode]; exists {
			continue
		}
		seen[reasonCode] = struct{}{}
		reasonCodes = append(reasonCodes, reasonCode)
	}
	if len(reasonCodes) == 0 {
		return SelectionDecision{}, fmt.Errorf("token selection strategy must return at least one reason code")
	}
	sort.Strings(reasonCodes)
	decision.ReasonCodes = reasonCodes
	decision.ReasonDetail = strings.TrimSpace(decision.ReasonDetail)
	return decision, nil
}
