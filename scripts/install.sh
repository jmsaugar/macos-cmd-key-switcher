#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
app="$HOME/Applications/CmdKeySwitcher.app"
agent="$HOME/Library/LaunchAgents/local.cmdkeyswitcher.plist"
launchctl bootout "gui/$(id -u)/local.cmdkeyswitcher" 2>/dev/null || true
mkdir -p "$HOME/Applications" "$HOME/Library/LaunchAgents" "$HOME/Library/Logs/CmdKeySwitcher"
ditto build/CmdKeySwitcher.app "$app"
# plistlib escapes paths correctly, including spaces and XML characters.
python3 - "$app" "$agent" "$HOME/Library/Logs/CmdKeySwitcher" <<'PY'
import plistlib, sys
app, agent, logs = sys.argv[1:]
with open(agent, 'wb') as f:
    plistlib.dump({'Label': 'local.cmdkeyswitcher',
                  'ProgramArguments': [app + '/Contents/MacOS/cmd-key-switcher'],
                  'RunAtLoad': True, 'KeepAlive': True,
                  'LimitLoadToSessionType': 'Aqua',
                  'StandardOutPath': logs + '/stdout.log',
                  'StandardErrorPath': logs + '/stderr.log'}, f)
PY
launchctl bootstrap "gui/$(id -u)" "$agent"
echo "Installed and started CmdKeySwitcher. It will start at login."
