package gdprotopb

import (
	"bytes"
	"os"
	"testing"
)

func TestBytesMatchesProtoFile(t *testing.T) {
	want, err := os.ReadFile("options.proto")
	if err != nil {
		t.Fatalf("read options.proto: %v", err)
	}
	if got := Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("Bytes() differs from options.proto on disk")
	}
}

func TestExtensionDescriptorAvailable(t *testing.T) {
	if E_ClassPrefix == nil {
		t.Fatal("E_ClassPrefix not generated")
	}
	if got := E_ClassPrefix.TypeDescriptor().Number(); int32(got) != 51000 {
		t.Fatalf("unexpected extension number: got %d want 51000", got)
	}
	if E_Resource == nil {
		t.Fatal("E_Resource not generated")
	}
	if got := E_Resource.TypeDescriptor().Number(); int32(got) != 51001 {
		t.Fatalf("unexpected resource extension number: got %d want 51001", got)
	}
}
