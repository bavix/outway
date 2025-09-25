package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bavix/outway/internal/firewall"
)

func newCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup all rules created by Outway",
		RunE: func(cmd *cobra.Command, args []string) error {
			backend, err := firewall.DetectBackend(cmd.Context())
			if err != nil {
				return err
			}

			return backend.CleanupAll(cmd.Context())
		},
	}

	return cmd
}
