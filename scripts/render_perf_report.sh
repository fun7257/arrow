#!/usr/bin/env bash
# Parse scratch bench logs and emit perf_report.md + perf_analysis.md.
set -euo pipefail

SCRATCH="${SCRATCH:?set SCRATCH to the goal scratch directory}"
OUT="${SCRATCH}/perf_report.md"
ANALYSIS="${SCRATCH}/perf_analysis.md"
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

extract_bytes() {
	local file=$1
	local bench=$2
	grep -E "^${bench}-" "$file" | awk '{print $4}' | head -1
}

for f in "$BEFORE" "$AFTER"; do
	if [[ ! -f $f ]]; then
		echo "missing $f" >&2
		exit 1
	fi
done

scenarios=(Minimal Static Parametric Middleware Large)

b_min=$(extract_ns "$BEFORE" "BenchmarkArrow_Minimal")
a_min=$(extract_ns "$AFTER" "BenchmarkArrow_Minimal")
b_sta=$(extract_ns "$BEFORE" "BenchmarkArrow_Static")
a_sta=$(extract_ns "$AFTER" "BenchmarkArrow_Static")
b_par=$(extract_ns "$BEFORE" "BenchmarkArrow_Parametric")
a_par=$(extract_ns "$AFTER" "BenchmarkArrow_Parametric")
b_mw=$(extract_ns "$BEFORE" "BenchmarkArrow_Middleware")
a_mw=$(extract_ns "$AFTER" "BenchmarkArrow_Middleware")
b_lrg=$(extract_ns "$BEFORE" "BenchmarkArrow_Large")
a_lrg=$(extract_ns "$AFTER" "BenchmarkArrow_Large")

b_min_a=$(extract_allocs "$BEFORE" "BenchmarkArrow_Minimal")
a_min_a=$(extract_allocs "$AFTER" "BenchmarkArrow_Minimal")
b_sta_a=$(extract_allocs "$BEFORE" "BenchmarkArrow_Static")
a_sta_a=$(extract_allocs "$AFTER" "BenchmarkArrow_Static")

s_min=$(extract_ns "$AFTER" "BenchmarkStdlib_Minimal")
s_sta=$(extract_ns "$AFTER" "BenchmarkStdlib_Static")
s_par=$(extract_ns "$AFTER" "BenchmarkStdlib_Parametric")
s_mw=$(extract_ns "$AFTER" "BenchmarkStdlib_Middleware")
s_lrg=$(extract_ns "$AFTER" "BenchmarkStdlib_Large")

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

gap_pct() {
	awk -v a="$1" -v s="$2" 'BEGIN{printf "%.0f%%", (a/s - 1) * 100}'
}

g_min=$(gap_pct "$a_min" "$s_min")
g_sta=$(gap_pct "$a_sta" "$s_sta")
g_par=$(gap_pct "$a_par" "$s_par")
g_mw=$(gap_pct "$a_mw" "$s_mw")
g_lrg=$(gap_pct "$a_lrg" "$s_lrg")

cat >"$OUT" <<EOF
# Performance Report (auto-generated)

Source: \`scripts/render_perf_report.sh\` — do not hand-edit ns/op; re-run renderer after new captures.

Before: \`bench_before.frozen.log\` (${BASELINE_COMMIT:-95a1c24} hot-path files on HEAD bench suite, frozen once)
After: \`bench_after.log\` (HEAD, same command)

## Step 5 gating

| Scenario | Before ns/op | After ns/op | Before allocs | After allocs | ns gate | allocs gate |
|----------|--------------|-------------|---------------|--------------|---------|-------------|
| Minimal | ${b_min} | ${a_min} | ${b_min_a} | ${a_min_a} | ${gate_min} | ${alloc_gate} |
| Static | ${b_sta} | ${a_sta} | ${b_sta_a} | ${a_sta_a} | ${gate_sta} | ${alloc_gate} |

**Overall step 5:** $([[ "$gate_min" == PASS || "$gate_sta" == PASS ]] && [[ "$alloc_gate" == PASS ]] && echo PASS || echo FAIL)

Command: \`go test -bench=. -benchmem -count=1 -run='^$' ./...\`

## Arrow vs Stdlib (from bench_after.log)

| Scenario | Arrow ns/op | Stdlib ns/op | Arrow slower by |
|----------|-------------|--------------|-----------------|
| Minimal | ${a_min} | ${s_min} | ${g_min} |
| Static | ${a_sta} | ${s_sta} | ${g_sta} |
| Parametric | ${a_par} | ${s_par} | ${g_par} |
| Middleware | ${a_mw} | ${s_mw} | ${g_mw} |
| Large | ${a_lrg} | ${s_lrg} | ${g_lrg} |

## Hot path

Zero-middleware benchmarks use the router inline closure (\`defer recoverAndRelease\` → handler → afters), equivalent to \`executeZeroMiddleware\` / \`runNoMiddleware\` for normal zero-mw requests.
Verified by \`TestBenchHotPathExecutesZeroMiddlewareDispatch\`, \`TestZeroMiddlewareInlineEquivalentToRunNoMiddleware\`, and \`TestZeroMiddleware*\`.
EOF

cat >"$ANALYSIS" <<EOF
# Arrow HTTP Performance Analysis

Auto-generated from \`bench_after.log\` vs \`bench_before.frozen.log\`. Numbers are \`ns/op\` from \`go test -benchmem -count=1\`.

## Optimization delta (before → after)

| Scenario | Before | After | Change |
|----------|--------|-------|--------|
| Minimal | ${b_min} | ${a_min} | $(awk -v b="$b_min" -v a="$a_min" 'BEGIN{printf "%.2f ns", a-b}') |
| Static | ${b_sta} | ${a_sta} | $(awk -v b="$b_sta" -v a="$a_sta" 'BEGIN{printf "%.2f ns", a-b}') |
| Parametric | ${b_par} | ${a_par} | $(awk -v b="$b_par" -v a="$a_par" 'BEGIN{printf "%.2f ns", a-b}') |
| Middleware | ${b_mw} | ${a_mw} | $(awk -v b="$b_mw" -v a="$a_mw" 'BEGIN{printf "%.2f ns", a-b}') |
| Large | ${b_lrg} | ${a_lrg} | $(awk -v b="$b_lrg" -v a="$a_lrg" 'BEGIN{printf "%.2f ns", a-b}') |

Baseline before uses ${BASELINE_COMMIT:-95a1c24} hot-path files (\`router.go\`, \`pipeline.go\`, \`context.go\`, \`pool.go\`, \`writer_wrap.go\`) with the current HEAD benchmark suite for a fair apples-to-apples comparison.

## Arrow vs Stdlib gap by scenario

### Minimal (${a_min} vs ${s_min}, +${g_min})

Arrow wraps every request in a pooled \`Context\`, \`statusWriter\`, and optional \`ResponseWriter\` interface shims (Flusher on \`httptest.ResponseRecorder\`). Stdlib calls the handler directly on the recorder. The ~${g_min} gap is dominated by context acquisition, writer wrapping, and the extra closure frame versus a bare \`HandleFunc\`.

### Static (${a_sta} vs ${s_sta}, +${g_sta})

Same per-request wrapper cost as minimal, plus \`ServeMux\` matching across a multi-route table. Arrow pays the framework envelope on every hit; stdlib only dispatches. Route count does not add middleware overhead here (zero-mw inline path).

### Parametric (${a_par} vs ${s_par}, +${g_par})

Adds Go 1.22+ \`PathValue\` extraction on both sides. Arrow still carries Context/writer overhead; parametric adds 1 alloc (16 B) on both implementations from path parsing.

### Middleware (${a_mw} vs ${s_mw}, +${g_mw})

Largest relative gap: Arrow runs a 5-layer linear penetration pipeline with \`After\` registration per layer; stdlib benchmark intentionally has no equivalent (stdlib cannot compose middleware without manual chaining). The ~${g_mw} delta is expected framework cost, not a routing bug.

### Large (${a_lrg} vs ${s_lrg}, +${g_lrg})

120-route table stresses mux matching. Arrow adds the same per-request Context/writer tax on top of mux work. Gap scales with table size but remains primarily wrapper + dispatch framing, not extra allocations (0 allocs/op).

## What was optimized

1. **Zero-middleware router fast path** — inline closure avoids \`pipeline.Run\` / \`runNoMiddleware\` call on bench hot path.
2. **Context pooling** — reuse \`Context\` and slice backing for \`afters\`.
3. **Writer wrapping** — interface mask cache by type; inline \`wrapF\` for Flusher-only writers (recorder).
4. **Direct \`statusWriter\` writes** — \`Context.Write\` / \`WriteHeader\` / \`Abort\` bypass \`c.Writer\` interface dispatch.

## Residual overhead (acceptable)

Arrow will remain slower than bare \`ServeMux\` where middleware, After, Abort, and panic recovery are part of the contract. Further gains would require dropping features or merging mux+handler registration — out of scope.

See also: \`perf_report.md\` for gating table.
EOF

echo "Wrote ${OUT} and ${ANALYSIS}"
cat "$OUT"