package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Execute runs the root command with the given args and IO streams.
// It returns the process exit code. A non-zero return means an error
// was reported on errOut.
func Execute(args []string, out, errOut io.Writer) int {
	cmd := newRootCommand(out, errOut)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func newRootCommand(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gogdproto [flags] INPUT",
		Short:         "Protocol Buffers compiler for GDScript (Godot 4.5)",
		Long:          "gogdproto compiles .proto files to GDScript for use in Godot 4.5.",
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.SetOut(out)
	cmd.SetErr(errOut)

	cmd.SetVersionTemplate(fmt.Sprintf("gogdproto %s\n", Version))

	return cmd
}
