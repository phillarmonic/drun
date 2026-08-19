#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TEMP_ROOT"' EXIT

DEST_ROOT="$TEMP_ROOT/ai-skills"
mkdir -p "$DEST_ROOT/skills"

cat >"$DEST_ROOT/repertoire.yaml" <<'EOF'
schema: 1
catalog:
  name: phillarmonic
  skills:
    drun:
      path: old/drun
      instructions:
        codex:
          - id: drun-guidance
            source: skills/drun/pointers/agents.md
            destination: AGENTS.md
            mode: markdown-section
    existing:
      path: skills/existing
EOF

cat >"$DEST_ROOT/README.md" <<'EOF'
# Fixture

## Available skills

| Skill | Description |
| --- | --- |
| `existing` | Existing description. |

## Next section

Preserve this content.
EOF

"$SCRIPT_DIR/vendor-skills-to-ai-skills.sh" \
  --source "$SOURCE_ROOT" \
  --dest "$DEST_ROOT" \
  --version v0.0.0-test

python3 - "$DEST_ROOT" <<'PY'
import pathlib
import sys
import yaml

root = pathlib.Path(sys.argv[1])
manifest = yaml.safe_load((root / "repertoire.yaml").read_text(encoding="utf-8"))
skills = manifest["catalog"]["skills"]
assert skills["drun"]["path"] == "skills/drun"
assert skills["drun"]["instructions"]["codex"][0]["destination"] == "AGENTS.md"
assert skills["existing"]["path"] == "skills/existing"
readme = (root / "README.md").read_text(encoding="utf-8")
assert "| `existing` | Existing description. |" in readme
assert "| `drun` |" in readme
assert "Preserve this content." in readme
assert (root / "skills/drun/SKILL.md").is_file()
assert (root / "skills/drun/pointers/agents.md").is_file()
PY

FIRST_DIGEST="$(
  find "$DEST_ROOT" -type f -print0 |
    sort -z |
    xargs -0 shasum |
    shasum |
    awk '{print $1}'
)"

"$SCRIPT_DIR/vendor-skills-to-ai-skills.sh" \
  --source "$SOURCE_ROOT" \
  --dest "$DEST_ROOT" \
  --version v0.0.0-test >/dev/null

SECOND_DIGEST="$(
  find "$DEST_ROOT" -type f -print0 |
    sort -z |
    xargs -0 shasum |
    shasum |
    awk '{print $1}'
)"

if [[ "$FIRST_DIGEST" != "$SECOND_DIGEST" ]]; then
  echo "error: vendor sync is not idempotent" >&2
  exit 1
fi

if "$SCRIPT_DIR/vendor-skills-to-ai-skills.sh" \
  --source "$SOURCE_ROOT" \
  --dest "$DEST_ROOT" \
  "Invalid Name" >/dev/null 2>&1; then
  echo "error: invalid skill name unexpectedly succeeded" >&2
  exit 1
fi

if "$SCRIPT_DIR/vendor-skills-to-ai-skills.sh" \
  --source "$SOURCE_ROOT" \
  --dest "$DEST_ROOT" \
  missing-skill >/dev/null 2>&1; then
  echo "error: missing skill unexpectedly succeeded" >&2
  exit 1
fi

echo "Vendor skill tests passed."
