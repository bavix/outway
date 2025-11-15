package dnsproxy

import (
	"context"

	"github.com/bavix/outway/internal/config"
)

// UpstreamsManager defines the interface for managing upstream resolvers.
type UpstreamsManager interface {
	// GetUpstreams returns current upstream configurations
	GetUpstreams() []config.UpstreamConfig
	// SetUpstreams updates upstream configurations
	SetUpstreams(upstreams []config.UpstreamConfig) error
	// GetUpstreamAddresses returns upstream addresses for legacy resolvers
	GetUpstreamAddresses() []string
	// RebuildResolvers rebuilds the resolver chain with current upstreams
	RebuildResolvers(ctx context.Context, strategies []UpstreamStrategy, deps StrategyDeps) []Resolver
}

// HostsManager defines the interface for managing host overrides.
type HostsManager interface {
	// GetHosts returns current host overrides
	GetHosts() []config.HostOverride
	// SetHosts updates host overrides
	SetHosts(hosts []config.HostOverride) error
	// CreateHostsResolver creates a new HostsResolver with current hosts
	CreateHostsResolver(next Resolver, cfg *config.Config) *HostsResolver
	// UpdateHostsInPlace updates hosts without rebuilding resolver pipeline
	UpdateHostsInPlace(hosts []config.HostOverride) error
}

// CacheManager defines the interface for managing DNS cache.
type CacheManager interface {
	// GetCache returns the current cache resolver
	GetCache() *CachedResolver
	// UpdateCacheNext updates the Next resolver in the cache
	UpdateCacheNext(next Resolver)
	// FlushCache clears all cache entries
	FlushCache()
	// DeleteCacheEntry removes specific cache entries
	DeleteCacheEntry(name string, qtype uint16)
}

// HistoryManager defines the interface for managing query history.
type HistoryManager interface {
	// AddEvent adds a query event to history
	AddEvent(event QueryEvent)
	// GetHistory returns recent query events
	GetHistory(limit int) []QueryEvent
	// GetHistoryPaginated returns paginated history with offset and limit
	GetHistoryPaginated(offset, limit int) []QueryEvent
	// GetHistorySize returns the total number of history events
	GetHistorySize() int
	// ClearHistory clears all history
	ClearHistory()
}

// RulesManager defines the interface for managing routing rules.
type RulesManager interface {
	// GetRules returns the current rule store
	GetRules() *RuleStore
	// UpdateRules updates the rule store
	UpdateRules(rules *RuleStore)
	// GetRuleGroups returns current rule groups
	GetRuleGroups() []config.RuleGroup
}

// ConfigManager defines the interface for managing configuration.
type ConfigManager interface {
	// GetConfig returns current configuration
	GetConfig() *config.Config
	// SaveConfig saves configuration to disk
	SaveConfig() error
	// UpdateConfig updates configuration atomically
	UpdateConfig(updater func(*config.Config)) error
}
