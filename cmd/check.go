package cmd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/firewall"
)

const (
	minPartsForInterface = 2
)

var (
	errToolNotFound      = errors.New("required tool not found")
	errInterfaceNotFound = errors.New("interface not found")
)

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check system status and configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := zerolog.Ctx(ctx)

			path := cfgFile
			if path == "" {
				path = "/etc/outway/config.yaml"
			}

			cfg, err := config.Load(path)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			log.Info().Str("config", path).Msg("checking system status")

			// Check firewall backend
			backend, err := firewall.DetectBackend(ctx)
			if err != nil {
				log.Err(err).Msg("no supported firewall backend detected")

				return err
			}

			log.Info().Str("backend", backend.Name()).Msg("firewall backend detected")

			// Check system tools
			if err := checkSystemTools(ctx, backend.Name()); err != nil {
				log.Err(err).Msg("system tools check failed")

				return err
			}

			// Check network interfaces
			if err := checkNetworkInterfaces(ctx, cfg); err != nil {
				log.Err(err).Msg("network interfaces check failed")

				return err
			}

			// Check firewall rules
			if err := checkFirewallRules(ctx, backend.Name()); err != nil {
				log.Warn().Err(err).Msg("firewall rules check failed")
			}

			log.Info().Msg("system check completed successfully")

			return nil
		},
	}

	return cmd
}

func checkSystemTools(ctx context.Context, backend string) error {
	log := zerolog.Ctx(ctx)

	var tools []string

	switch backend {
	case "simple_route":
		tools = []string{"ip"} // Only need ip command for simple route backend
	case "pf":
		tools = []string{"pfctl", "route"} // pf backend needs pfctl and route
	case "iptables":
		tools = []string{"iptables", "ipset"} // iptables backend
	default:
		tools = []string{"ip"} // Default to ip command
	}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			log.Err(err).Str("tool", tool).Msg("required tool not found")

			return fmt.Errorf("%w: %s", errToolNotFound, tool)
		}

		log.Debug().Str("tool", tool).Msg("tool found")
	}

	return nil
}

func checkNetworkInterfaces(ctx context.Context, cfg *config.Config) error {
	log := zerolog.Ctx(ctx)

	// Get list of configured interfaces
	interfaces := make(map[string]struct{})
	for _, rule := range cfg.GetAllRules() {
		interfaces[rule.Via] = struct{}{}
	}

	// Check if interfaces exist using appropriate command for OS
	var cmd *exec.Cmd
	if _, err := exec.LookPath("ip"); err == nil {
		// Linux: use ip command
		cmd = exec.CommandContext(ctx, "ip", "link", "show")
	} else if _, err := exec.LookPath("ifconfig"); err == nil {
		// macOS/BSD: use ifconfig command
		cmd = exec.CommandContext(ctx, "ifconfig")
	} else {
		log.Warn().Msg("no network interface listing tool found (ip or ifconfig)")
		return nil // Skip interface checking if no tools available
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Err(err).Msg("failed to list network interfaces")
		return err
	}

	availableInterfaces := make(map[string]bool)

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.Fields(line)
			if len(parts) >= minPartsForInterface {
				name := strings.TrimSuffix(parts[0], ":") // ifconfig uses name at start, ip uses at position 1
				availableInterfaces[name] = true
			}
		}
	}

	// Check configured interfaces
	for iface := range interfaces {
		if iface == "lo" || iface == "lo0" {
			log.Debug().Str("iface", iface).Msg("loopback interface (always available)")
			continue
		}

		if availableInterfaces[iface] {
			log.Info().Str("iface", iface).Msg("interface exists")
		} else {
			log.Error().Str("iface", iface).Msg("configured interface not found")
			return fmt.Errorf("%w: %s", errInterfaceNotFound, iface)
		}
	}

	return nil
}

func checkFirewallRules(ctx context.Context, backend string) error {
	// Simplified firewall backends don't need complex rule checking
	log := zerolog.Ctx(ctx)
	log.Debug().Str("backend", backend).Msg("firewall rules check skipped for simplified backend")
	return nil
}
