package generator

import (
	"testing"

	"github.com/cafecito-games/gdproto/internal/ast"
)

func TestResolvePrefixFromFilename(t *testing.T) {
	cases := []struct{ in, want string }{
		{"example.proto", "Example"},
		{"game_state.proto", "GameState"},
		{"weird-name.proto", "WeirdName"},
		{"nested/foo_bar.proto", "FooBar"},
		{"v1/api.proto", "Api"},
	}
	for _, c := range cases {
		f := &ast.ProtoFile{}
		got, err := ResolvePrefix(f, c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("%s: got %q want %q", c.in, got, c.want)
		}
	}
}

func TestResolvePrefixFromOption(t *testing.T) {
	f := &ast.ProtoFile{
		Options: map[string]any{"(gdproto.class_prefix)": "Game"},
	}
	got, err := ResolvePrefix(f, "example.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Game" {
		t.Fatalf("got %q want Game", got)
	}
}

func TestResolvePrefixOptionValidation(t *testing.T) {
	bad := []string{"game", "1Game", "Game-X", ""}
	for _, v := range bad {
		f := &ast.ProtoFile{
			Options: map[string]any{"(gdproto.class_prefix)": v},
		}
		if _, err := ResolvePrefix(f, "example.proto"); err == nil {
			t.Fatalf("%q: expected error", v)
		}
	}
}

func TestNameResolverResolvesNested(t *testing.T) {
	file := &ast.ProtoFile{
		Package: "",
		Messages: []*ast.Message{
			{Name: "Player", NestedMessages: []*ast.Message{{Name: "Position"}}},
			{Name: "GameState"},
		},
		Enums: []*ast.Enum{{Name: "Foo"}},
	}
	r, err := NewNameResolver([]FileEntry{{File: file, Filename: "example.proto"}})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"Player":          "ExamplePlayer",
		"Player.Position": "ExamplePlayerPosition",
		"GameState":       "ExampleGameState",
		"Foo":             "ExampleFoo",
	}
	for fqn, want := range cases {
		got, ok := r.Lookup(fqn)
		if !ok {
			t.Fatalf("missing %s", fqn)
		}
		if got != want {
			t.Fatalf("%s: got %s want %s", fqn, got, want)
		}
	}
}

func TestNameResolverEnumLookup(t *testing.T) {
	file := &ast.ProtoFile{
		Messages: []*ast.Message{{Name: "Player"}},
		Enums: []*ast.Enum{{
			Name:   "PlayerStatus",
			Values: []*ast.EnumValue{{Name: "OFFLINE", Number: 0}},
		}},
	}
	r, err := NewNameResolver([]FileEntry{{File: file, Filename: "example.proto"}})
	if err != nil {
		t.Fatal(err)
	}
	if !r.IsEnum("PlayerStatus") {
		t.Fatalf("expected PlayerStatus to be reported as an enum")
	}
	if r.IsEnum("Player") {
		t.Fatalf("Player message should not be reported as an enum")
	}
	wrapper, inner, ok := r.LookupEnum("PlayerStatus")
	if !ok {
		t.Fatalf("LookupEnum(PlayerStatus) returned ok=false")
	}
	if wrapper != "ExamplePlayerStatus" {
		t.Fatalf("wrapper: got %q want ExamplePlayerStatus", wrapper)
	}
	if inner != "PlayerStatus" {
		t.Fatalf("inner: got %q want PlayerStatus", inner)
	}
	if _, _, ok := r.LookupEnum("Player"); ok {
		t.Fatalf("LookupEnum(Player) should return ok=false for a message")
	}
	// Lookup still returns the wrapper class for enums for backwards
	// compatibility.
	if got, ok := r.Lookup("PlayerStatus"); !ok || got != "ExamplePlayerStatus" {
		t.Fatalf("Lookup(PlayerStatus): got %q ok=%v", got, ok)
	}
}

func TestNameResolverWithPackage(t *testing.T) {
	file := &ast.ProtoFile{
		Package:  "game.v1",
		Messages: []*ast.Message{{Name: "Player"}},
	}
	r, _ := NewNameResolver([]FileEntry{{File: file, Filename: "example.proto"}})
	got, ok := r.Lookup("game.v1.Player")
	if !ok || got != "ExamplePlayer" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	// Leading-dot tolerance:
	if got, ok := r.Lookup(".game.v1.Player"); !ok || got != "ExamplePlayer" {
		t.Fatalf("leading-dot lookup: got %q ok=%v", got, ok)
	}
}
