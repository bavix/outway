package cmd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

const (
	minPartsForTTL = 6
	minPartsForSet = 2
	percentageBase = 100
)

var errInvalidTTLFormat = errors.New("invalid TTL line format")

func newTTLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ttl",
		Short: "Monitor TTL values in nftables sets",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := zerolog.Ctx(ctx)

			log.Info().Msg("monitoring TTL values in nftables sets")

			// Check if outway table exists
			cmdCheck := exec.CommandContext(ctx, "nft", "list", "table", "inet", "outway")

			out, err := cmdCheck.CombinedOutput()
			if err != nil {
				log.Error().Bytes("out", out).Msg("outway table not found")

				return fmt.Errorf("outway table not found: %w", err)
			}

			// Parse and display TTL information
			return parseAndDisplayTTL(ctx, string(out))
		},
	}

	return cmd
}

func parseAndDisplayTTL(ctx context.Context, nftOutput string) error {
	log := zerolog.Ctx(ctx)

	lines := strings.Split(nftOutput, "\n")
	currentSet := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect set name
		if strings.HasPrefix(line, "set ") {
			parts := strings.Fields(line)
			if len(parts) >= minPartsForSet {
				currentSet = parts[1]
				log.Info().Str("set", currentSet).Msg("found set")
			}

			continue
		}

		// Parse elements with timeout
		if strings.Contains(line, "timeout") && strings.Contains(line, "expires") {
			ip, timeout, expires, err := parseTTLLine(line)
			if err != nil {
				log.Warn().Err(err).Str("line", line).Msg("failed to parse TTL line")

				continue
			}

			// Calculate remaining time
			remaining, err := time.ParseDuration(expires)
			if err != nil {
				log.Warn().Err(err).Str("expires", expires).Msg("failed to parse expires duration")

				continue
			}

			// Calculate total timeout
			total, err := time.ParseDuration(timeout)
			if err != nil {
				log.Warn().Err(err).Str("timeout", timeout).Msg("failed to parse timeout duration")

				continue
			}

			// Calculate elapsed time
			elapsed := total - remaining
			elapsedPercent := float64(elapsed) / float64(total) * percentageBase

			log.Info().
				Str("set", currentSet).
				Str("ip", ip).
				Dur("total_timeout", total).
				Dur("remaining", remaining).
				Dur("elapsed", elapsed).
				Float64("elapsed_percent", elapsedPercent).
				Msg("TTL status")
		}
	}

	return nil
}

func parseTTLLine(line string) (string, string, string, error) {
	// Parse line like: "74.125.11.105 timeout 26m31s expires 14m3s180ms"
	parts := strings.Fields(line)
	if len(parts) < minPartsForTTL {
		return "", "", "", errInvalidTTLFormat
	}

	ip := strings.TrimSuffix(parts[0], ",")

	// Find timeout and expires
	var timeout, expires string

	for i, part := range parts {
		if part == "timeout" && i+1 < len(parts) {
			timeout = parts[i+1]
		}

		if part == "expires" && i+1 < len(parts) {
			expires = parts[i+1]
		}
	}

	return ip, timeout, expires, nil
}
