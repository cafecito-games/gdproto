package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cafecito-games/gogdproto/internal/applog"
	"github.com/cafecito-games/gogdproto/internal/generator"
	"github.com/cafecito-games/gogdproto/internal/importer"
	"github.com/cafecito-games/gogdproto/internal/lexer"
	"github.com/cafecito-games/gogdproto/internal/parser"
	"github.com/cafecito-games/gogdproto/internal/validator"
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
	var outputPath string
	var rootLogger *slog.Logger

	cmd := &cobra.Command{
		Use:           "gogdproto [flags] INPUT",
		Short:         "Protocol Buffers compiler for GDScript (Godot 4.5)",
		Long:          "gogdproto compiles .proto files to GDScript for use in Godot 4.5.",
		Args:          cobra.MaximumNArgs(1),
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if outputPath == "" {
				return fmt.Errorf("required flag(s) \"output\" not set")
			}
			return runCompile(cmd, args[0], outputPath)
		},
	}

	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetVersionTemplate(fmt.Sprintf("gogdproto %s\n", Version))

	cmd.PersistentFlags().StringVar(
		&logLevelFlag, "log-level", "warn",
		"log verbosity (debug|info|warn|error)",
	)

	cmd.Flags().StringVarP(
		&outputPath, "output", "o", "",
		"output .gd file path",
	)

	return cmd
}

func runCompile(cmd *cobra.Command, inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath) //nolint:gosec // user-supplied path; CLI tool reads files by design.
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	tokens, err := lexer.Tokenize(string(data), inputPath)
	if err != nil {
		return err
	}

	file, err := parser.Parse(tokens, inputPath)
	if err != nil {
		return err
	}

	fs := &importer.OSFS{BaseDir: filepath.Dir(inputPath)}
	if err := importer.ResolveExternal(file, inputPath, fs); err != nil {
		return err
	}

	if errs := validator.Validate(file, inputPath); len(errs) != 0 {
		for _, e := range errs {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.Error())
		}
		return fmt.Errorf("validation failed")
	}

	cls, err := generator.Generate(file, filepath.Base(inputPath))
	if err != nil {
		return err
	}
	output := cls.ToGDScript(0)
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	if dir := filepath.Dir(outputPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}
	if err := os.WriteFile(outputPath, []byte(output), 0o644); err != nil { //nolint:gosec // generated source written to user-specified path.
		return fmt.Errorf("write output: %w", err)
	}

	siblingPath := filepath.Join(filepath.Dir(outputPath), "proto_core_utils.gd")
	if err := os.WriteFile(siblingPath, []byte(generator.GenerateProtoCoreUtilsRaw()), 0o644); err != nil { //nolint:gosec // sibling generated source written next to user-specified output.
		return fmt.Errorf("write sibling: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Generated %s\n", outputPath)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Generated %s\n", siblingPath)
	return nil
}
