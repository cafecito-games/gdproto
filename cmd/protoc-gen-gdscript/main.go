// Package main implements the protoc-gen-gdscript plugin binary.
//
// It speaks the standard protoc plugin protocol: a CodeGeneratorRequest is
// read as binary protobuf from stdin, each requested file is converted to a
// GDScript wrapper, and a CodeGeneratorResponse is written to stdout.
//
// Usage:
//
//	protoc --gdscript_out=<out_dir> --plugin=protoc-gen-gdscript=/path/to/binary <file.proto>
//
// When invoked with --print-options-proto the binary writes the embedded
// gdproto/options.proto bytes to stdout and exits without reading stdin, so
// users who only have the plugin installed can recover the options schema.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/cafecito-games/gdproto/internal/descriptors"
	"github.com/cafecito-games/gdproto/internal/gdprotopb"
	"github.com/cafecito-games/gdproto/internal/generator"
	"github.com/cafecito-games/gdproto/internal/validator"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--print-options-proto" {
			if _, err := printOptionsProto(os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// printOptionsProto writes the embedded gdproto options proto to w.
func printOptionsProto(w io.Writer) (int, error) {
	return w.Write(gdprotopb.Bytes())
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

	emittedFrom := map[string]string{}

	for _, name := range request.GetFileToGenerate() {
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

		imports := make([]generator.FileEntry, 0, len(files)-1)
		for otherIndex, otherFile := range files {
			if otherIndex == index {
				continue
			}
			imports = append(imports, generator.FileEntry{
				File:     otherFile,
				Filename: request.GetProtoFile()[otherIndex].GetName(),
			})
		}

		generated, err := generator.Generate(file, name, imports)
		if err != nil {
			message := fmt.Sprintf("generate %s: %v", name, err)
			response.Error = &message
			return writeResponse(out, response)
		}

		for _, gf := range generated {
			if origin, dup := emittedFrom[gf.Filename]; dup {
				message := fmt.Sprintf(
					"class name collision: %s emitted by both %s and %s; set option (gdproto.class_prefix) to disambiguate",
					gf.Filename, origin, name,
				)
				response.Error = &message
				return writeResponse(out, response)
			}
			emittedFrom[gf.Filename] = name
			response.File = append(response.File, &pluginpb.CodeGeneratorResponse_File{
				Name:    proto.String(gf.Filename),
				Content: proto.String(gf.Source()),
			})
		}
	}

	if len(response.File) > 0 {
		response.File = append(response.File, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String("proto_core_utils.gd"),
			Content: proto.String(generator.GenerateProtoCoreUtilsRaw()),
		})
	}

	return writeResponse(out, response)
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
