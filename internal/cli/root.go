package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cafecito-games/gdproto/internal/applog"
	"github.com/cafecito-games/gdproto/internal/gdprotopb"
	"github.com/cafecito-games/gdproto/internal/generator"
	"github.com/cafecito-games/gdproto/internal/importer"
	"github.com/cafecito-games/gdproto/internal/lexer"
	"github.com/cafecito-games/gdproto/internal/parser"
	"github.com/cafecito-games/gdproto/internal/validator"
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
	var printOptionsProto bool
	var rootLogger *slog.Logger

	cmd := &cobra.Command{
		Use:           "gdproto [flags] INPUT",
		Short:         "Protocol Buffers compiler for GDScript (Godot 4.5)",
		Long:          "gdproto compiles .proto files to GDScript for use in Godot 4.5.",
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
			if printOptionsProto {
				_, err := cmd.OutOrStdout().Write(gdprotopb.Bytes())
				return err
			}
			if len(args) == 0 {
				return cmd.Help()
			}
			return runCompile(cmd, args[0], outputPath)
		},
	}

	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetVersionTemplate(fmt.Sprintf("gdproto %s\n", Version))

	cmd.PersistentFlags().StringVar(
		&logLevelFlag, "log-level", "warn",
		"log verbosity (debug|info|warn|error)",
	)

	cmd.Flags().StringVarP(
		&outputPath, "output", "o", "",
		"output directory for generated .pb.gd files (default cwd)",
	)
	cmd.Flags().BoolVar(
		&printOptionsProto, "print-options-proto", false,
		"print the embedded gdproto/options.proto to stdout and exit",
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

	files, err := generator.Generate(file, sourceNameForCLI(inputPath))
	if err != nil {
		return err
	}

	outDir := outputPath
	if outDir == "" {
		outDir = "."
	}
	if err := validateOutputDir(outDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	written := 0
	for _, gf := range files {
		p := filepath.Join(outDir, gf.Filename)
		if err := os.WriteFile(p, []byte(gf.Source()), 0o644); err != nil { //nolint:gosec
			return fmt.Errorf("write %s: %w", p, err)
		}
		written++
	}
	siblingPath := filepath.Join(outDir, "proto_core_utils.gd")
	if err := os.WriteFile(siblingPath, []byte(generator.GenerateProtoCoreUtilsRaw()), 0o644); err != nil { //nolint:gosec
		return fmt.Errorf("write %s: %w", siblingPath, err)
	}
	written++

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d files to %s/\n", written, outDir)
	return nil
}

func validateOutputDir(p string) error {
	if strings.HasSuffix(p, ".gd") {
		return fmt.Errorf("-o must be a directory; per-message files are written inside it. Got: %s", p)
	}
	info, err := os.Stat(p)
	if err == nil && !info.IsDir() {
		return fmt.Errorf("-o must be a directory; per-message files are written inside it. Got: %s", p)
	}
	return nil
}

func sourceNameForCLI(inputPath string) string {
	cleaned := filepath.Clean(inputPath)
	if !filepath.IsAbs(cleaned) {
		parts := strings.Split(filepath.ToSlash(cleaned), "/")
		filtered := parts[:0]
		for _, part := range parts {
			if part == "." || part == ".." || part == "" {
				continue
			}
			filtered = append(filtered, part)
		}
		if len(filtered) == 0 {
			return filepath.ToSlash(filepath.Base(cleaned))
		}
		return strings.Join(filtered, "/")
	}
	dir := filepath.Base(filepath.Dir(cleaned))
	base := filepath.Base(cleaned)
	if dir == "." || dir == string(filepath.Separator) || dir == "" {
		return filepath.ToSlash(base)
	}
	return filepath.ToSlash(filepath.Join(dir, base))
}
