package firewall

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type pfBackend struct {
	mu     sync.Mutex
	timers map[string]*time.Timer // ip -> timer
}

func newPFBackend() Backend {
	if _, err := exec.LookPath("pfctl"); err != nil {
		return nil
	}

	if _, err := exec.LookPath("route"); err != nil {
		return nil
	}

	return &pfBackend{timers: make(map[string]*time.Timer)}
}

func (p *pfBackend) Name() string { return "pf" }

func (p *pfBackend) EnsurePolicy(ctx context.Context, iface string) error {
	zerolog.Ctx(ctx).Info().Str("iface", iface).Msg("ensure pf policy")

	return nil
}

func (p *pfBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	if ttlSeconds < 30 {
		ttlSeconds = 30
	}

	table := pfTableName(iface)
	zerolog.Ctx(ctx).Debug().Str("iface", iface).Str("ip", ip).Int("ttl", ttlSeconds).Msg("mark ip")

	cmd := exec.Command("pfctl", "-t", table, "-T", "add", ip)
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Str("out", string(out)).Msg("pfctl add failed")
	}

	args := []string{"-n", "add"}
	if strings.Contains(ip, ":") {
		args = append(args, "-inet6")
	}

	args = append(args, "-host", ip, "-interface", iface)

	addRoute := exec.Command("route", args...)
	if out, err := addRoute.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Str("out", string(out)).Str("args", strings.Join(args, " ")).Msg("route add failed")
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
		_ = exec.Command("pfctl", "-t", table, "-T", "delete", ip).Run()

		delArgs := []string{"-n", "delete"}
		if strings.Contains(ip, ":") {
			delArgs = append(delArgs, "-inet6")
		}

		delArgs = append(delArgs, "-host", ip, "-interface", iface)
		_ = exec.Command("route", delArgs...).Run()

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

func pfTableName(iface string) string { return "outway_" + iface }
