package cmd

import (
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/outway/internal/adminhttp"
	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
	"github.com/bavix/outway/internal/firewall"
	"github.com/bavix/outway/internal/metrics"
)

var dryRun bool //nolint:gochecknoglobals // cobra command flag

func newRunCmd() *cobra.Command { //nolint:cyclop,funlen
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Outway DNS proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := zerolog.Ctx(ctx)

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

			if dryRun {
				ifaces := map[string]struct{}{}
				for _, r := range cfg.GetAllRules() {
					ifaces[r.Via] = struct{}{}
				}
				for iface := range ifaces {
					log.Info().Str("iface", iface).Str("backend", backend.Name()).Msg("dry-run ensure policy")
				}
				log.Info().Msg("dry-run complete")

				return nil
			}

			defer func() { _ = backend.CleanupAll(ctx) }()
			_ = backend.CleanupAll(ctx)

			ifaces := map[string]struct{}{}
			for _, r := range cfg.GetAllRules() {
				ifaces[r.Via] = struct{}{}
			}
			for iface := range ifaces {
				if err := backend.EnsurePolicy(ctx, iface); err != nil {
					return err
				}
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
