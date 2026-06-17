#!/bin/bash
set -uo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

ITERATIONS="${1:-50}"

if ! [[ "${ITERATIONS}" =~ ^[0-9]+$ ]] || [ "${ITERATIONS}" -le 0 ]; then
    echo "usage: $0 [positive-iteration-count]" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

echo -e "${BLUE}==${NC} Building xdrun for fuzzing"
mkdir -p bin
go build -o bin/xdrun ./cmd/xdrun

CORPUS=()
while IFS= read -r file; do
    CORPUS+=("${file}")
done < <(find examples -maxdepth 1 -type f -name '*.drun' | sort)
if [ "${#CORPUS[@]}" -eq 0 ]; then
    echo -e "${RED}No example corpus found under examples/${NC}" >&2
    exit 1
fi

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/drun-fuzz.XXXXXX")"
trap 'rm -rf "${TMP_ROOT}"' EXIT

TOTAL=0
PARSE_OK=0
PARSE_FAIL=0
DRY_RUN_OK=0
DRY_RUN_SKIP=0
CRASHES=0

extract_first_task() {
    local list_file="$1"
    grep -E "^  " "${list_file}" | head -1 | sed 's/^  //' | sed -E 's/  +.*//' | xargs
}

try_dry_run() {
    local spec_file="$1"
    local task_name="$2"
    local output_file="$3"

    local param_attempts=(
        ""
        "name=test"
        "environment=dev"
        "items=test1,test2"
        "source_path=/tmp/test"
        "name=test environment=dev"
        "name=test title=friend"
        "app_name=myapp namespace=default"
    )

    local params
    for params in "${param_attempts[@]}"; do
        if [ -n "${params}" ]; then
            # shellcheck disable=SC2086
            if ./bin/xdrun -f "${spec_file}" "${task_name}" ${params} --dry-run >"${output_file}" 2>&1; then
                return 0
            fi
        else
            if ./bin/xdrun -f "${spec_file}" "${task_name}" --dry-run >"${output_file}" 2>&1; then
                return 0
            fi
        fi
    done

    return 1
}

append_generated_task() {
    local target="$1"
    local iteration="$2"
    cat >> "${target}" <<EOF

task "fuzz generated ${iteration}" means "Generated local fuzz case":
    info "Generated from the example corpus"
    step "iteration ${iteration}"
    run "echo fuzz-${iteration}"
EOF
}

mutate_case() {
    local source_file="$1"
    local target_file="$2"
    local iteration="$3"

    cp "${source_file}" "${target_file}"

    local passes=$((1 + RANDOM % 3))
    local pass
    for ((pass = 0; pass < passes; pass++)); do
        case $((RANDOM % 8)) in
            0)
                perl -0pi -e 's/\binfo\b/step/' "${target_file}"
                ;;
            1)
                perl -0pi -e 's/\bstep\b/info/' "${target_file}"
                ;;
            2)
                perl -0pi -e 's/task "([^"]+)":/task "$1 fuzzed":/' "${target_file}"
                ;;
            3)
                perl -0pi -e 's/version: 2\.0/version: 2.0\n\n# fuzzed input/' "${target_file}"
                ;;
            4)
                perl -0pi -e 's/"([^"]+)"/"$1 fuzz"/' "${target_file}"
                ;;
            5)
                append_generated_task "${target_file}" "${iteration}"
                ;;
            6)
                printf '\n# fuzz-note-%s\n' "${iteration}" >> "${target_file}"
                ;;
            7)
                perl -0pi -e 's/\brun\b/info/' "${target_file}"
                ;;
        esac
    done

    if [ $((RANDOM % 5)) -eq 0 ]; then
        case $((RANDOM % 3)) in
            0)
                perl -0pi -e 's/:\n/\n/' "${target_file}"
                ;;
            1)
                perl -0pi -e 's/task "([^"]+)":/task $1:/' "${target_file}"
                ;;
            2)
                perl -0pi -e 's/"([^"]+)"/"$1/' "${target_file}"
                ;;
        esac
    fi
}

echo -e "${BLUE}==${NC} Running ${ITERATIONS} semantic fuzz iterations"

for ((i = 1; i <= ITERATIONS; i++)); do
    TOTAL=$((TOTAL + 1))
    SOURCE_FILE="${CORPUS[RANDOM % ${#CORPUS[@]}]}"
    CASE_FILE="${TMP_ROOT}/case-${i}.drun"
    LIST_OUTPUT="${TMP_ROOT}/case-${i}.list.log"
    DRY_OUTPUT="${TMP_ROOT}/case-${i}.dry.log"

    mutate_case "${SOURCE_FILE}" "${CASE_FILE}" "${i}"

    ./bin/xdrun -f "${CASE_FILE}" -l >"${LIST_OUTPUT}" 2>&1
    status=$?

    if [ "${status}" -eq 0 ]; then
        PARSE_OK=$((PARSE_OK + 1))
        FIRST_TASK="$(extract_first_task "${LIST_OUTPUT}")"

        if [ -n "${FIRST_TASK}" ] && try_dry_run "${CASE_FILE}" "${FIRST_TASK}" "${DRY_OUTPUT}"; then
            DRY_RUN_OK=$((DRY_RUN_OK + 1))
            echo -e "${GREEN}PASS${NC} [${i}/${ITERATIONS}] $(basename "${SOURCE_FILE}") -> dry-run task '${FIRST_TASK}'"
        else
            DRY_RUN_SKIP=$((DRY_RUN_SKIP + 1))
            echo -e "${YELLOW}SOFT${NC} [${i}/${ITERATIONS}] $(basename "${SOURCE_FILE}") parsed, but no runnable first task"
        fi
        continue
    fi

    if [ "${status}" -ge 128 ] || grep -Eq 'panic:|fatal error:' "${LIST_OUTPUT}"; then
        CRASHES=$((CRASHES + 1))
        echo -e "${RED}CRASH${NC} [${i}/${ITERATIONS}] $(basename "${SOURCE_FILE}")"
        sed -n '1,80p' "${LIST_OUTPUT}"
    else
        PARSE_FAIL=$((PARSE_FAIL + 1))
        echo -e "${BLUE}MISS${NC} [${i}/${ITERATIONS}] $(basename "${SOURCE_FILE}") rejected by parser"
    fi
done

echo
echo -e "${BLUE}==${NC} Fuzz summary"
echo "iterations: ${TOTAL}"
echo "parsed: ${PARSE_OK}"
echo "rejected: ${PARSE_FAIL}"
echo "dry-run validated: ${DRY_RUN_OK}"
echo "parsed but not runnable: ${DRY_RUN_SKIP}"
echo "crashes: ${CRASHES}"

if [ "${PARSE_OK}" -eq 0 ]; then
    echo -e "${RED}No mutated inputs parsed successfully. The generator is too destructive.${NC}" >&2
    exit 1
fi

if [ "${CRASHES}" -ne 0 ]; then
    echo -e "${RED}Fuzzing found ${CRASHES} crash(es).${NC}" >&2
    exit 1
fi

echo -e "${GREEN}Semantic fuzzing completed without crashes.${NC}"
