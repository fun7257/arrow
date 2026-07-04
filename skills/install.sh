#!/usr/bin/env bash
# Install Arrow skills from skills/ into Grok skill directories.
#
# Usage:
#   ./skills/install.sh          # project scope → .grok/skills/arrow
#   ./skills/install.sh project
#   ./skills/install.sh user     # global → ~/.grok/skills/arrow
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="${REPO_ROOT}/skills/arrow"
SCOPE="${1:-project}"

if [[ ! -f "${SRC}/SKILL.md" ]]; then
	echo "error: missing ${SRC}/SKILL.md" >&2
	exit 1
fi

case "$SCOPE" in
project)
	DST="${REPO_ROOT}/.grok/skills/arrow"
	;;
user)
	DST="${HOME}/.grok/skills/arrow"
	;;
*)
	echo "usage: $0 [project|user]" >&2
	exit 1
	;;
esac

mkdir -p "$(dirname "$DST")"
rm -rf "$DST"
cp -R "$SRC" "$DST"
echo "Installed arrow skill:"
echo "  from: ${SRC}"
echo "  to:   ${DST}"
echo ""
echo "Grok: slash command /arrow (auto-reloads when files change)"