package cli

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/cafecito-games/gogdproto/internal/applog"
)

// Execute runs the root command with the given args and IO streams.
// It returns the process exit code.
func Execute(args []string, out, errOut io.Writer) int {
	cmd := newRootCommand(out, errOut)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func newRootCommand(out, errOut io.Writer) *cobra.Command {
	var logLevelFlag string
	var rootLogger *slog.Logger

	cmd := &cobra.Command{
		Use:           "gogdproto [flags] INPUT",
		Short:         "Protocol Buffers compiler for GDScript (Godot 4.5)",
		Long:          "gogdproto compiles .proto files to GDScript for use in Godot 4.5.",
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       Version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			level, err := applog.ParseLevel(logLevelFlag)
			if err != nil {
				return fmt.Errorf("invalid log level: %w", err)
			}
			rootLogger = applog.New(cmd.ErrOrStderr(), level)
			_ = rootLogger
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetVersionTemplate(fmt.Sprintf("gogdproto %s\n", Version))

	cmd.PersistentFlags().StringVar(
		&logLevelFlag, "log-level", "warn",
		"log verbosity (debug|info|warn|error)",
	)

	return cmd
}
