#!/usr/bin/env bash
# Install Arrow skills from skills/ into Grok skill directories.
#
# Usage:
#   ./skills/install.sh          # project scope → .grok/skills/arrow
#   ./skills/install.sh project
#   ./skills/install.sh user     # global → ~/.grok/skills/arrow
#
# Remote (no clone):
#   curl -fsSL https://raw.githubusercontent.com/fun7257/arrow/refs/heads/main/skills/install.sh | bash
#   curl -fsSL .../skills/install.sh | bash -s user
#   curl -fsSL .../skills/install.sh | bash -s project
set -euo pipefail

RAW_BASE="${ARROW_SKILL_RAW_BASE:-https://raw.githubusercontent.com/fun7257/arrow/refs/heads/main/skills/arrow}"
SCOPE="${1:-project}"

script_path="${BASH_SOURCE[0]:-$0}"
if [[ -n "$script_path" && "$script_path" != bash && "$script_path" != -bash && -f "$script_path" ]]; then
	REPO_ROOT="$(cd "$(dirname "$script_path")/.." && pwd)"
	SRC="${REPO_ROOT}/skills/arrow"
else
	REPO_ROOT=""
	SRC=""
fi

case "$SCOPE" in
project)
	if [[ -n "$REPO_ROOT" ]]; then
		DST="${REPO_ROOT}/.grok/skills/arrow"
	else
		DST="$(pwd)/.grok/skills/arrow"
	fi
	;;
user)
	DST="${HOME}/.grok/skills/arrow"
	;;
*)
	echo "usage: $0 [project|user]" >&2
	exit 1
	;;
esac

install_remote() {
	local dst="$1"
	local files=(
		SKILL.md
		references/api.md
		references/semantics.md
		references/recipes.md
	)

	mkdir -p "$dst"
	for rel in "${files[@]}"; do
		local url="${RAW_BASE}/${rel}"
		local out="${dst}/${rel}"
		mkdir -p "$(dirname "$out")"
		if ! curl -fsSL "$url" -o "$out"; then
			echo "error: failed to download ${url}" >&2
			exit 1
		fi
	done
}

if [[ -f "${SRC}/SKILL.md" ]]; then
	mkdir -p "$(dirname "$DST")"
	rm -rf "$DST"
	cp -R "$SRC" "$DST"
	echo "Installed arrow skill:"
	echo "  from: ${SRC}"
	echo "  to:   ${DST}"
else
	rm -rf "$DST"
	install_remote "$DST"
	echo "Installed arrow skill:"
	echo "  from: ${RAW_BASE}"
	echo "  to:   ${DST}"
fi

echo ""
echo "Grok: slash command /arrow (auto-reloads when files change)"