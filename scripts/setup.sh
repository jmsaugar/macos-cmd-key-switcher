#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."
dry_run=false
case "${1:-}" in
  "") ;;
  --dry-run) dry_run=true ;;
  *) echo "Usage: bash scripts/setup.sh [--dry-run]" >&2; exit 2 ;;
esac
if [ "$#" -gt 1 ]; then
  echo "Usage: bash scripts/setup.sh [--dry-run]" >&2
  exit 2
fi

# Capture separately so a discovery failure cannot become a bare install.
# Disable colors to keep the first column plain, even in an interactive terminal.
listing=$(MISE_COLOR=0 NO_COLOR=1 mise ls --local --no-header)
tools=()
while read -r tool remainder; do
  [ -n "$tool" ] || continue
  # Multiple configured versions can produce repeated rows for the same tool.
  duplicate=false
  for existing in "${tools[@]-}"; do
    if [ "$existing" = "$tool" ]; then duplicate=true; break; fi
  done
  if [ "$duplicate" = false ]; then tools+=("$tool"); fi
done <<< "$listing"

if [ "${#tools[@]}" -eq 0 ]; then
  echo "No project tools found; skipping installation." >&2
  exit 1
fi
printf 'Project tools:'
printf ' %s' "${tools[@]}"
printf '\n'
if [ "$dry_run" = true ]; then
  exec mise install --locked --dry-run "${tools[@]}"
fi
exec mise install --locked "${tools[@]}"
