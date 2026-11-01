package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// FirewallRuleRepository is an in-memory ports.FirewallRuleRepository.
type FirewallRuleRepository struct {
	mu    sync.RWMutex
	rules map[string]entities.NetworkPolicyRule
}

// NewFirewallRuleRepository creates an empty in-memory firewall rule store.
func NewFirewallRuleRepository() *FirewallRuleRepository {
	return &FirewallRuleRepository{rules: map[string]entities.NetworkPolicyRule{}}
}

func sortRules(rs []entities.NetworkPolicyRule) {
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].Priority != rs[j].Priority {
			return rs[i].Priority < rs[j].Priority
		}
		return rs[i].CreatedAt.Before(rs[j].CreatedAt)
	})
}

func (r *FirewallRuleRepository) ListByApp(_ context.Context, appID string) ([]entities.NetworkPolicyRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []entities.NetworkPolicyRule{}
	for _, rule := range r.rules {
		if rule.AppID == appID {
			out = append(out, rule)
		}
	}
	sortRules(out)
	return out, nil
}

func (r *FirewallRuleRepository) ListAll(_ context.Context) ([]entities.NetworkPolicyRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]entities.NetworkPolicyRule, 0, len(r.rules))
	for _, rule := range r.rules {
		out = append(out, rule)
	}
	sortRules(out)
	return out, nil
}

func (r *FirewallRuleRepository) Get(_ context.Context, id string) (*entities.NetworkPolicyRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.rules[id]
	if !ok {
		return nil, fmt.Errorf("firewall rule not found")
	}
	return &rule, nil
}

func (r *FirewallRuleRepository) Create(_ context.Context, rule *entities.NetworkPolicyRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID] = *rule
	return nil
}

func (r *FirewallRuleRepository) Update(_ context.Context, rule *entities.NetworkPolicyRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rules[rule.ID]; !ok {
		return fmt.Errorf("firewall rule not found")
	}
	r.rules[rule.ID] = *rule
	return nil
}

func (r *FirewallRuleRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rules, id)
	return nil
}

var _ ports.FirewallRuleRepository = (*FirewallRuleRepository)(nil)
