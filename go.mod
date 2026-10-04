module github.com/cafecito-games/gdproto

go 1.26

require (
	// Test-only, for the generated-output conformance gate. Pinned rather than
	// floating because a newer gdkit pulls in a newer gdparser; bump the two
	// together. The released `gdkit` binary may lag this module version: what
	// matters is that the formatter produces the same bytes, which is checked
	// against the golden output rather than assumed.
	github.com/cafecito-games/gdkit v0.4.2
	// Held at the gdparser version gdkit pins, so Canonicalize and a user's
	// installed `gdkit format` agree byte for byte. Bump deliberately, with
	// gdkit, and regenerate the goldens to prove the output did not move.
	github.com/cafecito-games/gdparser v0.1.6
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
