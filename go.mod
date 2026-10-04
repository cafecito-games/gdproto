module github.com/cafecito-games/gdproto

go 1.26

require (
	// Pinned to the gdparser version gdkit pins, so Canonicalize and a user's
	// installed `gdkit format` agree byte for byte. Bump deliberately, with gdkit.
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
