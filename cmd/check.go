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

	tools := []string{"nft", "ip"}
	if backend == "iptables" {
		tools = append(tools, "iptables", "ipset")
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

	// Check if interfaces exist
	cmd := exec.CommandContext(ctx, "ip", "link", "show")

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
				name := strings.TrimSuffix(parts[1], ":")
				availableInterfaces[name] = true
			}
		}
	}

	// Check configured interfaces
	for iface := range interfaces {
		if iface == "lo" {
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
	switch backend {
	case "nftables":
		return checkNFTablesRules(ctx)
	case "iptables":
		return checkIPTablesRules(ctx)
	}

	return nil
}

func checkNFTablesRules(ctx context.Context) error {
	log := zerolog.Ctx(ctx)

	// Check if outway table exists
	cmd := exec.CommandContext(ctx, "nft", "list", "table", "inet", "outway")

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug().Str("out", string(out)).Msg("outway table not found (normal if not started)")

		return err
	}

	log.Info().Msg("outway nftables table found")
	log.Debug().Str("table", string(out)).Msg("table contents")

	// Check if mangle chain exists
	cmd = exec.CommandContext(ctx, "nft", "list", "chain", "inet", "mangle", "outway_mark")

	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Debug().Str("out", string(out)).Msg("outway_mark chain not found")
	} else {
		log.Info().Msg("outway_mark chain found")
		log.Debug().Str("chain", string(out)).Msg("chain contents")
	}

	return nil
}

func checkIPTablesRules(ctx context.Context) error {
	log := zerolog.Ctx(ctx)

	// Check if ipset rules exist
	cmd := exec.CommandContext(ctx, "ipset", "list")

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug().Err(err).Msg("failed to list ipsets")

		return nil
	}

	if strings.Contains(string(out), "outway_") {
		log.Info().Msg("outway ipsets found")
		log.Debug().Bytes("ipsets", out).Msg("ipset contents")
	} else {
		log.Debug().Msg("no outway ipsets found (normal if not started)")
	}

	return nil
}
