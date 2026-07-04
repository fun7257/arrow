#!/usr/bin/env bash
# Medium-pressure HTTP load test against examples/server.
#
# Defaults: 30s sustained per endpoint, concurrency 100 (/health) and 50 (API).
# Requires examples/server on BASE (auto-starts when START_SERVER=1).
#
# Usage:
#   ./scripts/stress_test.sh
#   BASE=http://127.0.0.1:8080 OUT=stress.log ./scripts/stress_test.sh
#   START_SERVER=0 ./scripts/stress_test.sh   # server already running
#
# Documented in testdata/bench/README.md (medium stress section).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASE="${BASE:-http://127.0.0.1:8080}"
OUT="${OUT:-}"
START_SERVER="${START_SERVER:-1}"
PORT="${PORT:-8080}"
DURATION="${DURATION:-30s}"
HEALTH_C="${HEALTH_C:-100}"
API_C="${API_C:-50}"
AUTH_HEADER="Authorization: Bearer demo-token"

if command -v hey >/dev/null 2>&1; then
	LOADER=hey
elif [[ -x "$(go env GOPATH)/bin/hey" ]]; then
	LOADER="$(go env GOPATH)/bin/hey"
else
	LOADER=ab
fi

server_pid=""
cleanup() {
	if [[ -n $server_pid ]]; then
		kill "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
	if [[ "$START_SERVER" == "1" ]]; then
		lsof -ti:"${PORT}" | xargs kill -9 2>/dev/null || true
	fi
}
trap cleanup EXIT

start_server() {
	if curl -sf "${BASE}/health" >/dev/null 2>&1; then
		echo "Server already up at ${BASE}"
		return
	fi
	if [[ "$START_SERVER" != "1" ]]; then
		echo "error: server not reachable at ${BASE} (START_SERVER=0)" >&2
		exit 1
	fi
	lsof -ti:"${PORT}" | xargs kill -9 2>/dev/null || true
	sleep 0.5
	cd "$REPO_ROOT"
	PORT="$PORT" go run ./examples/server &
	server_pid=$!
	for _ in $(seq 1 30); do
		if curl -sf "${BASE}/health" >/dev/null 2>&1; then
			echo "Started examples/server on ${BASE} (pid ${server_pid})"
			return
		fi
		sleep 0.2
	done
	echo "error: server failed to start on ${BASE}" >&2
	exit 1
}

run_health() {
	echo "--- GET /health (${DURATION}, c=${HEALTH_C}) ---"
	if [[ $LOADER == hey ]]; then
		"$LOADER" -z "$DURATION" -c "$HEALTH_C" -m GET "${BASE}/health"
	else
		# ab: ~50k requests at c=100 ≈ medium sustained load
		ab -n 50000 -c "$HEALTH_C" -k "${BASE}/health"
	fi
}

run_api() {
	echo "--- GET /api/v1/posts (${DURATION}, c=${API_C}, auth) ---"
	if [[ $LOADER == hey ]]; then
		"$LOADER" -z "$DURATION" -c "$API_C" -m GET \
			-H "$AUTH_HEADER" "${BASE}/api/v1/posts"
	else
		ab -n 20000 -c "$API_C" -k -H "$AUTH_HEADER" "${BASE}/api/v1/posts"
	fi
}

run_parametric() {
	echo "--- GET /api/v1/posts/1 (${DURATION}, c=${API_C}, auth) ---"
	if [[ $LOADER == hey ]]; then
		"$LOADER" -z "$DURATION" -c "$API_C" -m GET \
			-H "$AUTH_HEADER" "${BASE}/api/v1/posts/1"
	else
		ab -n 20000 -c "$API_C" -k -H "$AUTH_HEADER" "${BASE}/api/v1/posts/1"
	fi
}

main() {
	{
		echo "=== Arrow medium stress test ==="
		echo "Time: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
		echo "Loader: ${LOADER}"
		echo "Target: ${BASE}"
		echo ""
		start_server
		echo ""
		run_health
		echo ""
		run_api
		echo ""
		run_parametric
		echo ""
		echo "=== DONE ==="
	} | if [[ -n $OUT ]]; then tee "$OUT"; else cat; fi
}

main "$@"