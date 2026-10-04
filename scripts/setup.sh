#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."
if [ "$#" -ne 0 ]; then
  echo "Usage: bash scripts/setup.sh" >&2
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
exec mise install --locked "${tools[@]}"
