#!/usr/bin/env bash
# Parse scratch bench logs and emit perf_report.md with gating verdict.
set -euo pipefail

SCRATCH="${SCRATCH:?set SCRATCH to the goal scratch directory}"
OUT="${SCRATCH}/perf_report.md"
BEFORE="${SCRATCH}/bench_before.frozen.log"
AFTER="${SCRATCH}/bench_after.log"

extract_ns() {
	local file=$1
	local bench=$2
	grep -E "^${bench}-" "$file" | awk '{print $3}' | head -1
}

extract_allocs() {
	local file=$1
	local bench=$2
	grep -E "^${bench}-" "$file" | awk '{print $5}' | head -1
}

for f in "$BEFORE" "$AFTER"; do
	if [[ ! -f $f ]]; then
		echo "missing $f" >&2
		exit 1
	fi
done

b_min=$(extract_ns "$BEFORE" "BenchmarkArrow_Minimal")
a_min=$(extract_ns "$AFTER" "BenchmarkArrow_Minimal")
b_sta=$(extract_ns "$BEFORE" "BenchmarkArrow_Static")
a_sta=$(extract_ns "$AFTER" "BenchmarkArrow_Static")
b_min_a=$(extract_allocs "$BEFORE" "BenchmarkArrow_Minimal")
a_min_a=$(extract_allocs "$AFTER" "BenchmarkArrow_Minimal")
b_sta_a=$(extract_allocs "$BEFORE" "BenchmarkArrow_Static")
a_sta_a=$(extract_allocs "$AFTER" "BenchmarkArrow_Static")

gate_min="FAIL"
gate_sta="FAIL"
if awk -v b="$b_min" -v a="$a_min" 'BEGIN{exit !(a<b)}'; then
	gate_min="PASS"
fi
if awk -v b="$b_sta" -v a="$a_sta" 'BEGIN{exit !(a<b)}'; then
	gate_sta="PASS"
fi
if [[ "$a_min_a" -le "$b_min_a" && "$a_sta_a" -le "$b_sta_a" ]]; then
	alloc_gate="PASS"
else
	alloc_gate="FAIL"
fi

extract_std() {
	local file=$1
	local bench=$2
	grep -E "^${bench}-" "$file" | awk '{print $3}' | head -1
}

s_min=$(extract_std "$AFTER" "BenchmarkStdlib_Minimal")
s_sta=$(extract_std "$AFTER" "BenchmarkStdlib_Static")

cat >"$OUT" <<EOF
# Performance Report (auto-generated)

Source: \`scripts/render_perf_report.sh\` — do not hand-edit ns/op; re-run renderer after new captures.

Before: \`bench_before.frozen.log\` (commit 95a1c24, full tree, frozen once)
After: \`bench_after.log\` (HEAD, same command)

## Step 5 gating

| Scenario | Before ns/op | After ns/op | Before allocs | After allocs | ns gate | allocs gate |
|----------|--------------|-------------|---------------|--------------|---------|-------------|
| Minimal | ${b_min} | ${a_min} | ${b_min_a} | ${a_min_a} | ${gate_min} | ${alloc_gate} |
| Static | ${b_sta} | ${a_sta} | ${b_sta_a} | ${a_sta_a} | ${gate_sta} | ${alloc_gate} |

**Overall step 5:** $([[ "$gate_min" == PASS || "$gate_sta" == PASS ]] && [[ "$alloc_gate" == PASS ]] && echo PASS || echo FAIL)

Command: \`go test -bench=. -benchmem -count=1 -run='^$' ./...\`

## Stdlib gap (from bench_after.log)

| Scenario | Arrow ns/op | Stdlib ns/op |
|----------|-------------|--------------|
| Minimal | ${a_min} | ${s_min} |
| Static | ${a_sta} | ${s_sta} |

## Hot path

Zero-middleware benchmarks use the router inline closure (\`defer recoverAndRelease\` → handler → afters).
Verified by \`TestRouterZeroMiddlewareUsesInlineClosure\` and \`TestZeroMiddleware*\`.
\`runNoMiddleware\` / \`finishRequest\` are used only by \`pipeline.Run\`.
EOF

echo "Wrote ${OUT}"
cat "$OUT"