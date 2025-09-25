package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bavix/outway/internal/logging"
	verpkg "github.com/bavix/outway/internal/version"
)

var (
	cfgFile   string //nolint:gochecknoglobals // cobra command flag
	logLevel  string //nolint:gochecknoglobals // cobra command flag
	logFormat string //nolint:gochecknoglobals // cobra command flag
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "outway",
		Short:         "DNS proxy that marks destination IPs and routes by interface",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			base := logging.Base("outway", logLevel, logFormat)
			ctx := base.WithContext(cmd.Context())
			cmd.SetContext(ctx)

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Path to config file (default: /etc/outway/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "json", "Log format: json, console")

	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newCleanupCmd())
	rootCmd.AddCommand(newUpdateCmd())

	// Add version command using built-in cobra version
	rootCmd.Version = verpkg.GetVersion()
	rootCmd.SetVersionTemplate("outway " + verpkg.GetVersion())

	return rootCmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func ExecuteContext(ctx context.Context) {
	if err := NewRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
