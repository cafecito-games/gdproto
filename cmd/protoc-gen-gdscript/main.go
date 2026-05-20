// Package main implements the protoc-gen-gdscript plugin binary.
//
// It speaks the standard protoc plugin protocol: a CodeGeneratorRequest is
// read as binary protobuf from stdin, each requested file is converted to a
// GDScript wrapper, and a CodeGeneratorResponse is written to stdout.
//
// Usage:
//
//	protoc --gdscript_out=<out_dir> --plugin=protoc-gen-gdscript=/path/to/binary <file.proto>
package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/descriptors"
	"github.com/cafecito-games/gdproto/internal/generator"
	"github.com/cafecito-games/gdproto/internal/validator"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run executes one round of the plugin protocol against the supplied IO.
// Errors from request parsing or descriptor conversion are reported through
// the response Error field rather than as a non-zero exit, matching the
// expectation of protoc that the plugin terminate cleanly.
func run(in io.Reader, out io.Writer) error {
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	request := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		response := &pluginpb.CodeGeneratorResponse{}
		message := fmt.Sprintf("unmarshal request: %v", err)
		response.Error = &message
		return writeResponse(out, response)
	}

	response := &pluginpb.CodeGeneratorResponse{}
	features := uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
	response.SupportedFeatures = &features

	files, err := descriptors.FromCodeGeneratorRequest(request)
	if err != nil {
		message := err.Error()
		response.Error = &message
		return writeResponse(out, response)
	}

	fileIndex := make(map[string]int, len(request.GetProtoFile()))
	for i, descriptor := range request.GetProtoFile() {
		fileIndex[descriptor.GetName()] = i
	}

	generationOrder := transitiveGenerationOrder(request.GetFileToGenerate(), files, fileIndex)

	for _, name := range generationOrder {
		index, ok := fileIndex[name]
		if !ok {
			continue
		}
		file := files[index]

		if errs := validator.Validate(file, name); len(errs) != 0 {
			var builder strings.Builder
			for i, validationErr := range errs {
				if i > 0 {
					builder.WriteByte('\n')
				}
				builder.WriteString(validationErr.Error())
			}
			message := builder.String()
			response.Error = &message
			return writeResponse(out, response)
		}

		class, err := generator.Generate(file, name)
		if err != nil {
			message := fmt.Sprintf("generate %s: %v", name, err)
			response.Error = &message
			return writeResponse(out, response)
		}

		outputName := convertToWrapperFilename(name)
		content := class.ToGDScript(0)
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		response.File = append(response.File, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String(outputName),
			Content: proto.String(content),
		})
	}

	if len(response.File) > 0 {
		response.File = append(response.File, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String("proto_core_utils.gd"),
			Content: proto.String(generator.GenerateProtoCoreUtilsRaw()),
		})
	}

	return writeResponse(out, response)
}

// transitiveGenerationOrder returns the explicit file_to_generate list followed
// by every file transitively imported through them that the request also
// supplied a descriptor for. Generated wrappers reference imported messages by
// their wrapper class (e.g. GoogleProtobufTimestampProto.Timestamp), so those
// classes have to exist on disk for Godot to resolve the type. Order is
// deterministic: BFS over the original file_to_generate sequence.
func transitiveGenerationOrder(seeds []string, files []*ast.ProtoFile, fileIndex map[string]int) []string {
	seen := make(map[string]bool, len(seeds))
	order := make([]string, 0, len(seeds))
	queue := make([]string, 0, len(seeds))
	for _, name := range seeds {
		if seen[name] {
			continue
		}
		seen[name] = true
		order = append(order, name)
		queue = append(queue, name)
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		index, ok := fileIndex[name]
		if !ok {
			continue
		}
		for _, imp := range files[index].Imports {
			if seen[imp.Path] {
				continue
			}
			if _, ok := fileIndex[imp.Path]; !ok {
				continue
			}
			seen[imp.Path] = true
			order = append(order, imp.Path)
			queue = append(queue, imp.Path)
		}
	}
	return order
}

func writeResponse(out io.Writer, response *pluginpb.CodeGeneratorResponse) error {
	data, err := proto.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}

// convertToWrapperFilename mirrors the Python plugin's filename rule:
// strip the .proto suffix, snake_case the basename, and append ".pb.gd".
//
//	"google/protobuf/timestamp.proto" -> "google/protobuf/timestamp.pb.gd"
//	"PlayerStats.proto"               -> "player_stats.pb.gd"
func convertToWrapperFilename(path string) string {
	base := strings.TrimSuffix(path, ".proto")
	slash := strings.LastIndex(base, "/")
	var directory, stem string
	if slash >= 0 {
		directory = base[:slash+1]
		stem = base[slash+1:]
	} else {
		stem = base
	}
	return directory + normalizeProtoStem(stem) + ".pb.gd"
}

var (
	snakeBoundary1  = regexp.MustCompile(`(.)([A-Z][a-z]+)`)
	snakeBoundary2  = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	nonAlphanumeric = regexp.MustCompile(`[^A-Za-z0-9]+`)
	multiUnderscore = regexp.MustCompile(`_+`)
)

// toSnakeCase converts PascalCase or camelCase to snake_case, matching the
// Python plugin's _to_snake_case helper.
func toSnakeCase(name string) string {
	intermediate := snakeBoundary1.ReplaceAllString(name, `${1}_${2}`)
	intermediate = snakeBoundary2.ReplaceAllString(intermediate, `${1}_${2}`)
	return strings.ToLower(intermediate)
}

// normalizeProtoStem sanitizes a proto filename stem into a snake_case
// identifier suitable for use in generated wrapper paths.
func normalizeProtoStem(name string) string {
	sanitized := strings.Trim(nonAlphanumeric.ReplaceAllString(name, "_"), "_")
	if sanitized == "" {
		return "proto"
	}
	snake := toSnakeCase(sanitized)
	snake = strings.Trim(multiUnderscore.ReplaceAllString(snake, "_"), "_")
	if snake == "" {
		return "proto"
	}
	if snake[0] >= '0' && snake[0] <= '9' {
		snake = "proto_" + snake
	}
	return snake
}
