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
	var includePaths []string
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
			return runCompile(cmd, args[0], outputPath, includePaths)
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
	cmd.Flags().StringSliceVarP(
		&includePaths, "proto_path", "I", nil,
		"directories to search for imported .proto files (repeatable, like protoc)",
	)

	return cmd
}

func runCompile(cmd *cobra.Command, inputPath, outputPath string, includePaths []string) error {
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

	fs := &importer.OSFS{
		BaseDir:      filepath.Dir(inputPath),
		IncludePaths: includePaths,
	}
	importedFiles, err := importer.ResolveExternalWithFiles(file, inputPath, fs)
	if err != nil {
		return err
	}

	if errs := validator.Validate(file, inputPath); len(errs) != 0 {
		for _, e := range errs {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.Error())
		}
		return fmt.Errorf("validation failed")
	}

	imports := make([]generator.FileEntry, 0, len(importedFiles))
	for _, imp := range importedFiles {
		imports = append(imports, generator.FileEntry{File: imp.File, Filename: imp.Filename})
	}

	files, err := generator.Generate(file, sourceNameForCLI(inputPath), imports)
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
		source, err := gf.Source()
		if err != nil {
			return err
		}
		count, err := writeWithSidecar(outDir, gf.Filename, source)
		if err != nil {
			return err
		}
		written += count
	}
	count, err := writeWithSidecar(outDir, "proto_core_utils.gd", generator.GenerateProtoCoreUtilsRaw())
	if err != nil {
		return err
	}
	written += count

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d files to %s/\n", written, outDir)
	return nil
}

// writeWithSidecar writes a generated GDScript file into outDir together with
// its .uid sidecar, and reports how many files it created.
//
// Godot writes the sidecar itself on import, but a project that has never
// opened the generated files in the editor has none, and gdkit reports a
// script without one as having no stable identity. Emitting it here also fixes
// the identifier, instead of letting whichever machine imports first decide it.
func writeWithSidecar(outDir, filename, source string) (int, error) {
	path := filepath.Join(outDir, filename)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil { //nolint:gosec // generated source intended to be world-readable
		return 0, fmt.Errorf("write %s: %w", path, err)
	}
	sidecarPath := filepath.Join(outDir, generator.SidecarFilename(filename))
	if err := os.WriteFile(sidecarPath, []byte(generator.SidecarSource(filename)), 0o644); err != nil { //nolint:gosec // generated source intended to be world-readable
		return 0, fmt.Errorf("write %s: %w", sidecarPath, err)
	}
	return 2, nil
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
	// CLI users pass filesystem paths whose intermediate directory
	// structure rarely carries proto-package meaning (especially absolute
	// paths under temp dirs, build trees, etc.). Use the basename so the
	// derived class_name prefix is stable and free of incidental segments
	// like build-artifact digits.
	return filepath.ToSlash(filepath.Base(filepath.Clean(inputPath)))
}
