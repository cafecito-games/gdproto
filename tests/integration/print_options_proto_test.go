//go:build integration

package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGdprotoPrintOptionsProto(t *testing.T) {
	gdproto, _ := buildBinaries(t)
	out, err := exec.Command(gdproto, "--print-options-proto").Output()
	if err != nil {
		t.Fatalf("gdproto --print-options-proto: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(repoRoot(t), "proto/gdproto/options.proto"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, want) {
		t.Fatalf("gdproto --print-options-proto output differs from proto/gdproto/options.proto on disk (%d vs %d bytes)", len(out), len(want))
	}
}

func TestPluginPrintOptionsProto(t *testing.T) {
	_, plugin := buildBinaries(t)
	out, err := exec.Command(plugin, "--print-options-proto").Output()
	if err != nil {
		t.Fatalf("protoc-gen-gdscript --print-options-proto: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(repoRoot(t), "proto/gdproto/options.proto"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, want) {
		t.Fatalf("plugin --print-options-proto output differs from proto/gdproto/options.proto on disk (%d vs %d bytes)", len(out), len(want))
	}
}
