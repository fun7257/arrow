#!/usr/bin/env bash
# Run the standard Arrow performance suite:
#   1) go test micro-benchmarks (Arrow vs stdlib, 5 scenarios)
#   2) medium HTTP stress test (examples/server)
#
# Usage:
#   ./scripts/run_perf.sh
#   BENCH_COUNT=3 OUT_DIR=./perf-out ./scripts/run_perf.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${OUT_DIR:-}"
BENCH_COUNT="${BENCH_COUNT:-1}"
PORT="${PORT:-8080}"

cd "$REPO_ROOT"

bench_out=""
stress_out=""
if [[ -n $OUT_DIR ]]; then
	mkdir -p "$OUT_DIR"
	bench_out="$OUT_DIR/bench.log"
	stress_out="$OUT_DIR/stress.log"
fi

echo "==> Micro-benchmarks (go test -bench, count=${BENCH_COUNT})"
if [[ -n $bench_out ]]; then
	go test -bench=. -benchmem -count="$BENCH_COUNT" -run='^$' ./... 2>&1 | tee "$bench_out"
else
	go test -bench=. -benchmem -count="$BENCH_COUNT" -run='^$' ./...
fi

echo ""
echo "==> Medium stress test (examples/server)"
PORT="$PORT" OUT="$stress_out" "$REPO_ROOT/scripts/stress_test.sh"

echo ""
echo "Perf suite complete."