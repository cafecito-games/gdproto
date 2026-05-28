#!/usr/bin/env bash
# Integration smoke test: builds protoc-gen-gdscript, generates the
# fixture .pb.gd wrappers under tests/godot/fixtures/godot/generated,
# imports the project headlessly, and runs the Vest test suite under
# res://tests to exercise round-tripping over the cross-file paths
# that have regressed before. Vest runs in a real SceneTree, so
# class_name resolution behaves the same way it does in a downstream
# game (running tests via --check-only / --script bypasses that and
# gives false negatives on cross-file enum references).
#
# Required on PATH: go, protoc, godot (4.6.x).

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HERE/../.." && pwd)"
PROTO_ROOT="$HERE/fixtures/proto"
GODOT_PROJECT="$HERE/fixtures/godot"
GENERATED_DIR="$GODOT_PROJECT/generated"
BIN_DIR="$HERE/.bin"
PLUGIN="$BIN_DIR/protoc-gen-gdscript"

GODOT="${GODOT:-godot}"

for tool in go protoc "$GODOT"; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "error: required tool '$tool' not found on PATH" >&2
        exit 127
    fi
done

mkdir -p "$BIN_DIR"
(cd "$REPO_ROOT" && go build -o "$PLUGIN" ./cmd/protoc-gen-gdscript)

rm -rf "$GENERATED_DIR" "$GODOT_PROJECT/.godot"
mkdir -p "$GENERATED_DIR"

# Drive every fixture .proto through the plugin in one protoc invocation
# so transitive imports (well-known timestamp.proto, sibling shared.proto)
# are exercised the same way buf would invoke us.
PROTO_FILES=()
while IFS= read -r line; do
    PROTO_FILES+=("$line")
done < <(cd "$PROTO_ROOT" && find . -type f -name '*.proto' -not -path './google/*' | sed 's|^\./||')
protoc \
    --plugin=protoc-gen-gdscript="$PLUGIN" \
    --gdscript_out="$GENERATED_DIR" \
    -I "$PROTO_ROOT" \
    "${PROTO_FILES[@]}"

if ! grep -q '^gdscript/warnings/treat_warnings_as_errors=true$' "$GODOT_PROJECT/project.godot"; then
    echo "error: test Godot project must keep strict GDScript warnings enabled" >&2
    exit 1
fi

project_backup="$(mktemp)"
cp "$GODOT_PROJECT/project.godot" "$project_backup"
vest_log=""
cleanup() {
    if [[ -f "$project_backup" ]]; then
        cp "$project_backup" "$GODOT_PROJECT/project.godot"
        rm -f "$project_backup"
    fi
    if [[ -n "${vest_log:-}" ]]; then
        rm -f "$vest_log"
    fi
}
trap cleanup EXIT

# The checked-in project.godot keeps strict warnings enabled so generated
# scripts fail during import if they are not strict-clean.
"$GODOT" --headless --path "$GODOT_PROJECT" --import

# Vest and the handwritten fixture tests are not strict-clean. Keep the strict
# import above as the generated-code gate, then temporarily relax the project
# for the existing runtime round-trip suite.
awk '
    /^\[debug\]$/ { skip = 1; next }
    /^\[/ { skip = 0 }
    skip != 1 { print }
' "$project_backup" > "$GODOT_PROJECT/project.godot"
rm -rf "$GODOT_PROJECT/.godot"
"$GODOT" --headless --path "$GODOT_PROJECT" --import

vest_log="$(mktemp)"
"$GODOT" --headless --path "$GODOT_PROJECT" \
    -s addons/vest/cli/vest-cli.gd \
    --vest-glob 'res://tests/**/test_*.gd' \
    --vest-report-format tap 2>&1 | tee "$vest_log"

# Vest's CLI exits 0 even when assertions fail; parse the TAP output.
if grep -E '^not ok ' "$vest_log" >/dev/null; then
    echo "error: at least one Vest assertion failed" >&2
    exit 1
fi
if ! grep -E '^ok ' "$vest_log" >/dev/null; then
    echo "error: no Vest tests ran" >&2
    exit 1
fi

cleanup
trap - EXIT
