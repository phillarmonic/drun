#!/usr/bin/env bash
# Sync canonical skill packages from this repo into a phillarmonic/ai-skills
# checkout while preserving catalog-specific metadata and README entries.

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: vendor-skills-to-ai-skills.sh --dest <ai-skills-checkout> [--version <tag>] [skill...]

Syncs skill directories from ./skills into an ai-skills repository checkout,
updates repertoire.yaml entries, and refreshes the README available-skills table.

Options:
  --dest PATH       Path to a phillarmonic/ai-skills checkout (required)
  --version TAG     Source tag or version label reported in command output
  --source PATH     Source repository root (default: repo containing this script)
  -h, --help        Show this help

If no skill names are given, reads skills/VENDOR (one skill name per line).
EOF
}

SOURCE_ROOT=""
DEST_ROOT=""
VERSION=""
SKILLS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dest)
      DEST_ROOT="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --source)
      SOURCE_ROOT="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      SKILLS+=("$@")
      break
      ;;
    -*)
      echo "error: unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      SKILLS+=("$1")
      shift
      ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -z "$SOURCE_ROOT" ]]; then
  SOURCE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
fi

if [[ -z "$DEST_ROOT" ]]; then
  echo "error: --dest is required" >&2
  usage >&2
  exit 2
fi

if [[ ! -d "$DEST_ROOT" ]]; then
  echo "error: destination is not a directory: $DEST_ROOT" >&2
  exit 1
fi

if [[ ! -f "$DEST_ROOT/repertoire.yaml" ]]; then
  echo "error: destination does not look like an ai-skills checkout (missing repertoire.yaml): $DEST_ROOT" >&2
  exit 1
fi

if ! python3 -c "import yaml" >/dev/null 2>&1; then
  echo "error: PyYAML is required (python3 -m pip install 'pyyaml>=6')" >&2
  exit 1
fi

VENDOR_LIST="$SOURCE_ROOT/skills/VENDOR"
if [[ ${#SKILLS[@]} -eq 0 ]]; then
  if [[ ! -f "$VENDOR_LIST" ]]; then
    echo "error: no skills specified and $VENDOR_LIST is missing" >&2
    exit 1
  fi
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%%#*}"
    line="$(printf '%s' "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    [[ -z "$line" ]] && continue
    SKILLS+=("$line")
  done <"$VENDOR_LIST"
fi

if [[ ${#SKILLS[@]} -eq 0 ]]; then
  echo "error: no skills to vendor" >&2
  exit 1
fi

validate_skill() {
  local skill="$1"
  local skill_dir="$SOURCE_ROOT/skills/$skill"
  local skill_md="$skill_dir/SKILL.md"

  if [[ ! "$skill" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
    echo "error: invalid skill name '$skill' (use lowercase letters, digits, single hyphens)" >&2
    return 1
  fi
  if [[ ! -f "$skill_md" ]]; then
    echo "error: missing SKILL.md for skill '$skill' at $skill_md" >&2
    return 1
  fi

  python3 - "$skill_md" "$skill" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
expected = sys.argv[2]
text = path.read_text(encoding="utf-8")
if not text.startswith("---\n"):
    raise SystemExit(f"error: {path} is missing YAML frontmatter")
try:
    _, frontmatter, _ = text.split("---", 2)
except ValueError as exc:
    raise SystemExit(f"error: {path} has unterminated YAML frontmatter") from exc
header = yaml.safe_load(frontmatter) or {}
if header.get("name") != expected:
    raise SystemExit(
        f"error: {path} frontmatter name {header.get('name')!r} "
        f"must match directory/catalog key {expected!r}"
    )
description = str(header.get("description", "")).strip()
if not description:
    raise SystemExit(f"error: {path} is missing a frontmatter description")
print(" ".join(description.split()))
PY
}

DESCRIPTIONS=()
for skill in "${SKILLS[@]}"; do
  description="$(validate_skill "$skill")"
  src="$SOURCE_ROOT/skills/$skill"
  dest="$DEST_ROOT/skills/$skill"
  mkdir -p "$DEST_ROOT/skills"
  rm -rf "$dest"
  mkdir -p "$dest"
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete \
      --exclude '.DS_Store' \
      --exclude '.git' \
      "$src/" "$dest/"
  else
    cp -R "$src/." "$dest/"
  fi

  if [[ ${#description} -gt 120 ]]; then
    description="${description:0:117}..."
  fi
  DESCRIPTIONS+=("$description")
  echo "vendored skills/$skill -> $dest"
done

python3 - "$DEST_ROOT/repertoire.yaml" "${SKILLS[@]}" <<'PY'
from __future__ import annotations

import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
skills = sys.argv[2:]
data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
catalog = data.setdefault("catalog", {})
if not isinstance(catalog, dict):
    raise SystemExit("error: repertoire.yaml catalog must be a mapping")
skill_map = catalog.setdefault("skills", {})
if not isinstance(skill_map, dict):
    raise SystemExit("error: repertoire.yaml catalog.skills must be a mapping")

for skill in skills:
    entry = skill_map.setdefault(skill, {})
    if not isinstance(entry, dict):
        raise SystemExit(f"error: repertoire.yaml entry for {skill!r} must be a mapping")
    entry["path"] = f"skills/{skill}"

catalog["skills"] = dict(sorted(skill_map.items()))
path.write_text(
    yaml.safe_dump(data, default_flow_style=False, sort_keys=False, allow_unicode=True, width=100),
    encoding="utf-8",
)
print(f"updated {path}")
PY

README="$DEST_ROOT/README.md"
if [[ -f "$README" ]]; then
  python3 - "$README" "${SKILLS[@]}" "${DESCRIPTIONS[@]}" <<'PY'
from __future__ import annotations

import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
args = sys.argv[2:]
if len(args) % 2:
    raise SystemExit("error: skill/description argument mismatch")
mid = len(args) // 2
updates = dict(zip(args[:mid], args[mid:]))

text = path.read_text(encoding="utf-8")
heading = "## Available skills"
start = text.find(heading)
if start < 0:
    raise SystemExit(f"error: {path} is missing the {heading!r} section")
body_start = start + len(heading)
next_heading = text.find("\n## ", body_start)
body_end = len(text) if next_heading < 0 else next_heading
body = text[body_start:body_end]

table_row = re.compile(r"^\|\s*`([^`]+)`\s*\|\s*(.*?)\s*\|\s*$")
lines = body.splitlines()
rows: list[tuple[str, str]] = []
row_indexes: list[int] = []
for index, line in enumerate(lines):
    match = table_row.match(line)
    if match:
        rows.append((match.group(1), match.group(2)))
        row_indexes.append(index)
if not rows:
    raise SystemExit(f"error: {path} Available skills section does not contain a skill table")

row_map = dict(rows)
order = [name for name, _ in rows]
for name, description in updates.items():
    row_map[name] = description.replace("|", r"\|")
    if name not in order:
        order.append(name)

first_row = row_indexes[0]
last_row = row_indexes[-1]
replacement = [f"| `{name}` | {row_map[name]} |" for name in order]
lines[first_row:last_row + 1] = replacement
new_body = "\n".join(lines)
if body.endswith("\n"):
    new_body += "\n"
path.write_text(text[:body_start] + new_body + text[body_end:], encoding="utf-8")
print(f"updated {path}")
PY
fi

if [[ -n "$VERSION" ]]; then
  echo "source_version=$VERSION"
fi
echo "vendored_skills=${SKILLS[*]}"
echo "Vendor sync complete."
