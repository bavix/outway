package dnsproxy

import (
	"context"
	"sync"

	"github.com/bavix/outway/internal/config"
)

// Thread-safe implementations of the interfaces

// upstreamsManager is a thread-safe implementation of UpstreamsManager.
type upstreamsManager struct {
	mu        sync.RWMutex
	upstreams []config.UpstreamConfig
	addresses []string
	proxy     *Proxy // Reference to proxy for rebuild operations
}

func newUpstreamsManager(proxy *Proxy, upstreams []config.UpstreamConfig) *upstreamsManager {
	addresses := make([]string, len(upstreams))
	for i, u := range upstreams {
		addresses[i] = u.Type + ":" + u.Address
	}

	return &upstreamsManager{
		upstreams: upstreams,
		addresses: addresses,
		proxy:     proxy,
	}
}

func (um *upstreamsManager) GetUpstreams() []config.UpstreamConfig {
	um.mu.RLock()
	defer um.mu.RUnlock()

	result := make([]config.UpstreamConfig, len(um.upstreams))
	copy(result, um.upstreams)

	return result
}

func (um *upstreamsManager) SetUpstreams(upstreams []config.UpstreamConfig) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	// Update upstreams
	um.upstreams = make([]config.UpstreamConfig, len(upstreams))
	copy(um.upstreams, upstreams)

	// Update addresses
	um.addresses = make([]string, len(upstreams))
	for i, u := range upstreams {
		um.addresses[i] = u.Type + ":" + u.Address
	}

	return nil
}

func (um *upstreamsManager) GetUpstreamAddresses() []string {
	um.mu.RLock()
	defer um.mu.RUnlock()

	result := make([]string, len(um.addresses))
	copy(result, um.addresses)

	return result
}

func (um *upstreamsManager) RebuildResolvers(ctx context.Context, strategies []UpstreamStrategy, deps StrategyDeps) []Resolver {
	um.mu.RLock()
	defer um.mu.RUnlock()

	// Use current upstreams to build resolvers
	ups := um.upstreams
	if len(ups) == 0 {
		// Fallback to legacy string list
		return um.proxy.buildLegacyResolvers(strategies, deps)
	}

	return um.proxy.buildWeightedResolvers(ups, strategies, deps)
}

// hostsManager is a thread-safe implementation of HostsManager.
type hostsManager struct {
	mu    sync.RWMutex
	hosts []config.HostOverride
	cfg   *config.Config
}

func newHostsManager(cfg *config.Config) *hostsManager {
	return &hostsManager{
		hosts: cfg.Hosts,
		cfg:   cfg,
	}
}

func (hm *hostsManager) GetHosts() []config.HostOverride {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make([]config.HostOverride, len(hm.hosts))
	copy(result, hm.hosts)

	return result
}

func (hm *hostsManager) SetHosts(hosts []config.HostOverride) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.hosts = make([]config.HostOverride, len(hosts))
	copy(hm.hosts, hosts)

	// Update config
	hm.cfg.Hosts = hm.hosts

	return nil
}

func (hm *hostsManager) CreateHostsResolver(next Resolver, cfg *config.Config) *HostsResolver {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	return &HostsResolver{
		Next:  next,
		Hosts: hm.hosts,
		Cfg:   cfg,
	}
}

func (hm *hostsManager) UpdateHostsInPlace(hosts []config.HostOverride) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Update hosts atomically
	hm.hosts = make([]config.HostOverride, len(hosts))
	copy(hm.hosts, hosts)

	// Update config
	hm.cfg.Hosts = hm.hosts

	return nil
}

// cacheManager is a thread-safe implementation of CacheManager.
type cacheManager struct {
	mu    sync.RWMutex
	cache *CachedResolver
}

func newCacheManager(cache *CachedResolver) *cacheManager {
	return &cacheManager{
		cache: cache,
	}
}

func (cm *cacheManager) GetCache() *CachedResolver {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.cache
}

func (cm *cacheManager) UpdateCacheNext(next Resolver) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.cache != nil {
		cm.cache.Next = next
	}
}

func (cm *cacheManager) FlushCache() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.cache != nil {
		cm.cache.Flush()
	}
}

func (cm *cacheManager) DeleteCacheEntry(name string, qtype uint16) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.cache != nil {
		cm.cache.Delete(name, qtype)
	}
}

// historyManager is a thread-safe implementation of HistoryManager.
type historyManager struct {
	mu       sync.RWMutex
	events   []QueryEvent
	head     int
	size     int
	capacity int
}

func newHistoryManager(capacity int) *historyManager {
	return &historyManager{
		events:   make([]QueryEvent, capacity),
		capacity: capacity,
	}
}

func (hm *historyManager) AddEvent(event QueryEvent) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.events[hm.head] = event

	hm.head = (hm.head + 1) % hm.capacity
	if hm.size < hm.capacity {
		hm.size++
	}
}

func (hm *historyManager) GetHistory(limit int) []QueryEvent {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if limit <= 0 || limit > hm.size {
		limit = hm.size
	}

	result := make([]QueryEvent, limit)
	for i := range limit {
		idx := (hm.head - limit + i + hm.capacity) % hm.capacity
		result[i] = hm.events[idx]
	}

	return result
}

func (hm *historyManager) ClearHistory() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.events = make([]QueryEvent, hm.capacity)
	hm.head = 0
	hm.size = 0
}

// rulesManager is a thread-safe implementation of RulesManager.
type rulesManager struct {
	mu     sync.RWMutex
	rules  *RuleStore
	groups []config.RuleGroup
}

func newRulesManager(rules *RuleStore, groups []config.RuleGroup) *rulesManager {
	return &rulesManager{
		rules:  rules,
		groups: groups,
	}
}

func (rm *rulesManager) GetRules() *RuleStore {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.rules
}

func (rm *rulesManager) UpdateRules(rules *RuleStore) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.rules = rules
}

func (rm *rulesManager) GetRuleGroups() []config.RuleGroup {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	result := make([]config.RuleGroup, len(rm.groups))
	copy(result, rm.groups)

	return result
}

// configManager is a thread-safe implementation of ConfigManager.
type configManager struct {
	mu  sync.RWMutex
	cfg *config.Config
}

func newConfigManager(cfg *config.Config) *configManager {
	return &configManager{
		cfg: cfg,
	}
}

func (cm *configManager) GetConfig() *config.Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.cfg
}

func (cm *configManager) SaveConfig() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.cfg.Save()
}

func (cm *configManager) UpdateConfig(updater func(*config.Config)) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	updater(cm.cfg)

	return cm.cfg.Save()
}
