module github.com/cafecito-games/gdproto

go 1.26

require (
	// Test-only, for the generated-output conformance gate. Both of these are
	// pinned to what the latest *released* `gdkit` binary uses, which is not the
	// same as the newest module: gdkit v0.4.2 exists as a module but requires
	// gdparser v0.1.6, while the distributed binary is still v0.4.1 on gdparser
	// v0.1.5. gdparser v0.1.6 wraps a long concatenation differently, so building
	// against it makes generated output that `gdkit format check` rejects for
	// anyone running the released binary. Bump these only when the released
	// binary moves, together, and verify by regenerating a large real schema --
	// the example corpus has no expression long enough to show the difference.
	github.com/cafecito-games/gdkit v0.4.1
	github.com/cafecito-games/gdparser v0.1.5
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
