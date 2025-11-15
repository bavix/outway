package firewall

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type pfBackend struct {
	mu         sync.Mutex
	timers     map[string]*time.Timer // ip -> timer
	routeCache map[string]time.Time   // Cache of existing routes: "ip:iface" -> expiry time
	cacheMu    sync.RWMutex
}

func NewPFBackend() *pfBackend {
	if _, err := exec.LookPath("pfctl"); err != nil {
		return nil
	}

	if _, err := exec.LookPath("route"); err != nil {
		return nil
	}

	return &pfBackend{
		timers:     make(map[string]*time.Timer),
		routeCache: make(map[string]time.Time),
	}
}

func (p *pfBackend) Name() string { return "pf" }

//nolint:cyclop,nestif,funlen // complex route management logic with nested cache checks
func (p *pfBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	ttlSeconds = max(ttlSeconds, minTTLSeconds)

	table := PFTableName(iface)
	zerolog.Ctx(ctx).Debug().Str("iface", iface).IPAddr("ip", net.ParseIP(ip)).Int("ttl", ttlSeconds).Msg("mark ip")

	cmd := exec.CommandContext(ctx, "pfctl", "-t", table, "-T", "add", ip) //nolint:gosec // pfctl is a system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).Msg("pfctl add failed")

		return fmt.Errorf("failed to add IP %s to pfctl table %s: %w", ip, table, err)
	}

	// Check cache first to avoid unnecessary route add attempts
	cacheKey := ip + ":" + iface

	p.cacheMu.RLock()

	if expiry, exists := p.routeCache[cacheKey]; exists && time.Now().Before(expiry) {
		p.cacheMu.RUnlock()
		zerolog.Ctx(ctx).Debug().Str("ip", ip).Str("iface", iface).Msg("route exists in cache, skipping")
		// Route exists in cache, continue with timer setup
	} else {
		p.cacheMu.RUnlock()

		args := []string{"-n", "add"}
		if strings.Contains(ip, ":") {
			args = append(args, "-inet6")
		}

		args = append(args, "-host", ip, "-interface", iface)

		addRoute := exec.CommandContext(ctx, "route", args...)
		if out, err := addRoute.CombinedOutput(); err != nil {
			output := string(out)
			// Check if route already exists - this is not an error
			if strings.Contains(output, "File exists") || strings.Contains(output, "already exists") {
				zerolog.Ctx(ctx).Debug().Str("ip", ip).Str("iface", iface).Msg("route already exists, skipping")
				// Route already exists, add to cache
				p.cacheMu.Lock()
				p.routeCache[cacheKey] = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
				p.cacheMu.Unlock()
				// Continue with timer setup
			} else {
				zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("args", strings.Join(args, " ")).Msg("route add failed")

				return fmt.Errorf("failed to add route for IP %s via interface %s: %w", ip, iface, err)
			}
		} else {
			// Route added successfully, add to cache
			p.cacheMu.Lock()
			p.routeCache[cacheKey] = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
			p.cacheMu.Unlock()
		}
	}
	// schedule/delete via cancellable timer (no blocking sleeps)
	d := time.Duration(ttlSeconds) * time.Second

	p.mu.Lock()

	if t, ok := p.timers[ip]; ok {
		if t.Stop() {
			delete(p.timers, ip)
		}
	}

	t := time.AfterFunc(d, func() {
		// best-effort deletion on expiry
		_ = exec.CommandContext(ctx, "pfctl", "-t", table, "-T", "delete", ip).Run() //nolint:gosec // pfctl is a system utility

		delArgs := []string{"-n", "delete"}
		if strings.Contains(ip, ":") {
			delArgs = append(delArgs, "-inet6")
		}

		delArgs = append(delArgs, "-host", ip, "-interface", iface)
		_ = exec.CommandContext(ctx, "route", delArgs...).Run()

		// Remove from cache
		p.cacheMu.Lock()
		delete(p.routeCache, cacheKey)
		p.cacheMu.Unlock()

		p.mu.Lock()
		delete(p.timers, ip)
		p.mu.Unlock()
	})
	p.timers[ip] = t
	p.mu.Unlock()

	return nil
}

func (p *pfBackend) CleanupAll(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Msg("cleanup pf tables")
	p.mu.Lock()

	for ip, t := range p.timers {
		if t.Stop() {
			delete(p.timers, ip)
		}
	}

	p.mu.Unlock()

	return nil
}
