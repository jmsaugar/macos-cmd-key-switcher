#!/bin/bash
set -euo pipefail
launchctl bootout "gui/$(id -u)/local.cmdkeyswitcher" 2>/dev/null || true
rm -f "$HOME/Library/LaunchAgents/local.cmdkeyswitcher.plist"
rm -rf "$HOME/Applications/CmdKeySwitcher.app"
rm -rf "$HOME/Library/Application Support/CmdKeySwitcher"
rm -rf "$HOME/Library/Logs/CmdKeySwitcher"
echo "Removed app, login agent, saved configuration, and logs."
