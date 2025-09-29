package cmd

import (
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/outway/internal/adminhttp"
	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
	"github.com/bavix/outway/internal/firewall"
	"github.com/bavix/outway/internal/metrics"
	"github.com/bavix/outway/internal/version"
)

var dryRun bool //nolint:gochecknoglobals // cobra command flag

func newRunCmd() *cobra.Command { //nolint:cyclop,funlen
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Outway DNS proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := zerolog.Ctx(ctx)

			// Log version information at startup
			log.Info().
				Str("version", version.GetVersion()).
				Str("build_time", version.GetBuildTime()).
				Msg("outway starting")

			path := cfgFile
			if path == "" {
				path = "/etc/outway/config.yaml"
			}

			cfg, err := config.Load(path)
			if err != nil {
				return err
			}

			metrics.RegisterCollectors()
			metrics.SetService(cfg.AppName)
			metrics.BindService()
			log.Info().Str("config", path).Msg("starting")

			backend, err := firewall.DetectBackend(ctx)
			if err != nil {
				return err
			}

			// Extract tunnel interfaces from config
			tunnels := map[string]struct{}{}
			for _, r := range cfg.GetAllRules() {
				tunnels[r.Via] = struct{}{}
			}

			tunnelList := make([]string, 0, len(tunnels))
			for tunnel := range tunnels {
				tunnelList = append(tunnelList, tunnel)
			}

			if dryRun {
				for _, tunnel := range tunnelList {
					log.Info().Str("tunnel", tunnel).Str("backend", backend.Name()).Msg("dry-run tunnel validation")
				}

				log.Info().Msg("dry-run complete")

				return nil
			}

			defer func() { _ = backend.CleanupAll(ctx) }()

			// Log configured tunnels (no initialization needed for simple backend)
			if len(tunnelList) > 0 {
				log.Info().
					Int("tunnels", len(tunnelList)).
					Strs("tunnel_list", tunnelList).
					Str("backend", backend.Name()).
					Msg("tunnels configured")
			}

			proxy := dnsproxy.New(cfg, backend)

			if cfg.HTTP.Enabled {
				admin := adminhttp.NewServerWithConfig(&cfg.HTTP, proxy)
				if err := admin.Start(ctx); err != nil {
					return err
				}
			}

			if err := proxy.Start(ctx); err != nil {
				return err
			}

			<-ctx.Done()

			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate config and backend, then exit")

	return cmd
}
