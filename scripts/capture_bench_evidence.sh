#!/usr/bin/env bash
# Capture benchmark evidence for Arrow perf goal verification.
# Writes exactly four files to SCRATCH (all use the same -count=1 full-suite command):
#   bench_before.log  — pre-optimization hot path on current bench suite
#   bench_after.log   — optimized hot path (HEAD)
#   bench_run1.log    — repeatability snapshot
#   bench_run2.log    — repeatability snapshot
#
# Before/after differ only in hot-path source files (context, pool, pipeline,
# router, writer_wrap) so the benchmark harness and testdata stay identical.
set -euo pipefail

SCRATCH="${SCRATCH:?set SCRATCH to the goal scratch directory}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASELINE_COMMIT="${BASELINE_COMMIT:-95a1c24}"
HOTPATH_FILES=(context.go pool.go pipeline.go router.go writer_wrap.go)
# Tests that reference optimized-only symbols; omitted during baseline capture.
HOTPATH_TEST_FILES=(hotpath_internal_test.go pipeline_internal_test.go)
BENCH_CMD=(go test -bench=. -benchmem -count=1 -run='^$' ./...)

cd "$REPO_ROOT"

STASH="${SCRATCH}/hotpath-stash"
rm -rf "$STASH"
mkdir -p "$STASH"
for f in "${HOTPATH_FILES[@]}" "${HOTPATH_TEST_FILES[@]}"; do
	[[ -f $f ]] && cp "$f" "$STASH/$f"
done

restore_hotpath() {
	local commit=$1
	for f in "${HOTPATH_FILES[@]}"; do
		git show "${commit}:${f}" >"$f"
	done
}

stash_hotpath_tests() {
	for f in "${HOTPATH_TEST_FILES[@]}"; do
		[[ -f $f ]] && rm -f "$f"
	done
}

restore_hotpath_tests() {
	for f in "${HOTPATH_TEST_FILES[@]}"; do
		[[ -f $STASH/$f ]] && cp "$STASH/$f" "$f"
	done
}

echo "==> bench_before.log (hot path at ${BASELINE_COMMIT}, bench suite at HEAD)"
restore_hotpath "$BASELINE_COMMIT"
stash_hotpath_tests
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_before.log"

echo "==> bench_after.log (optimized hot path from working tree stash)"
for f in "${HOTPATH_FILES[@]}" "${HOTPATH_TEST_FILES[@]}"; do
	[[ -f $STASH/$f ]] && cp "$STASH/$f" "$f"
done
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_after.log"

echo "==> bench_run1.log"
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_run1.log"

echo "==> bench_run2.log"
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_run2.log"

echo "Done. Evidence in ${SCRATCH}"