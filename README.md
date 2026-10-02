# CmdKeySwitcher

A Go macOS menu bar app with a small native Cocoa/IOKit bridge. No external Go dependencies. Requires macOS, Go 1.23 or later, and Xcode Command Line Tools (`xcode-select --install`).

```sh
make test
make build
make install
```

`make install` copies the app to `~/Applications`, registers a user LaunchAgent, and starts it immediately. It starts automatically at **user login**, when macOS provides a menu bar. It does not run before login. Installation requires permission to write outside the project when invoked from a sandboxed coding tool.

The menu bar shows `⌘ mac` or `⊞ win`. Its dropdown contains exactly one action, “Switch to win” or “Switch to mac”. To stop and remove the installed app, run `make uninstall`; this also deletes the saved configuration (including its lock file) and logs from `~/Library/Application Support/CmdKeySwitcher` and `~/Library/Logs/CmdKeySwitcher`. Uninstalling does not reset keyboard mappings previously applied by the app. The LaunchAgent restarts the app if it exits.

## Preview without installation or keyboard changes

Run `make preview` from a terminal. This builds the app inside the project and starts its menu bar UI with `--dry-run`. Click the single switch button to test both states; the terminal prints the HID command that would run. Press **Ctrl+C** in that terminal to exit.

Preview mode never executes `hidutil`, reads or writes your saved configuration, creates a lock file, installs the app, or registers a login agent. State exists only in memory. Building still creates files in `build/` and uses Go's build cache.

By default, preview uses real, read-only keyboard detection, so you can connect/disconnect a keyboard to test automatic selection. You can also simulate a fixed keyboard setup without needing hardware:

```sh
make build
build/CmdKeySwitcher.app/Contents/MacOS/cmd-key-switcher --dry-run --simulate none
build/CmdKeySwitcher.app/Contents/MacOS/cmd-key-switcher --dry-run --simulate mac
build/CmdKeySwitcher.app/Contents/MacOS/cmd-key-switcher --dry-run --simulate win
```

Run one command at a time and exit with Ctrl+C between scenarios. `--simulate` requires `--dry-run`. Preview always starts with the built-in default mappings and ignores `--config`.

## Detection and switching

At startup and when IOKit reports a keyboard connecting or disconnecting, the app chooses:

- No external keyboards, or only Apple/Mac-classified external keyboards: `mac`.
- Any Windows-classified external keyboard: `win`.

Apple vendor ID 1452 defaults to `mac`; other external vendors default to `win`. HID metadata cannot reliably identify a third-party keyboard's key layout or Mac/Windows hardware mode. Add a device override for those keyboards. Internal keyboards are ignored using the Built-In property and SPI/i2c transport. Detection requires the primary HID usage to be Generic Desktop / Keyboard. Secondary keyboard collections on media/audio peripherals are excluded, even if they advertise normal keys. A composite device whose keyboard collection is secondary will also be excluded by this conservative rule.

Manual selection also saves a classification override for every currently connected external keyboard after the command succeeds. Those overrides are used on reconnect and restart. With no external keyboard, manual switching changes only `type`; startup detection still selects `mac`. After wake, the selected mapping is reapplied. If multiple external keyboards have different layouts, `win` takes precedence, and the mapping applies globally to all keyboards, including the built-in keyboard.

The defaults reproduce `hidutils_commands.txt`: `win` swaps **left Command and left Option**, and `mac` maps those two keys to themselves. Right-side modifiers are unchanged. `hidutil property --set` replaces `UserKeyMapping`, so an existing mapping from another tool is overwritten.

## Configuration

The app writes `~/Library/Application Support/CmdKeySwitcher/config.json`. It updates `type` only after the HID command succeeds, and saves atomically. The HID commands run directly as `/usr/bin/hidutil property --set <mapping>` without a shell. Detection reads IORegistry device metadata and subscribes to service arrival/removal notifications, without creating HID input clients or reading keystrokes. Input Monitoring and Accessibility access are not needed for this detection path.

The configuration stores only `type` and `keyboard_types`. HID mappings are constants in `core.go`; changing them requires rebuilding the app. Overrides use `vendor:product_id:serial:<URL-escaped serial>` when a nonblank HID serial number is exposed, otherwise decimal `vendor:product_id`. Detection checks the unit override first, then the model override, then the vendor default. Serial numbers are supplied by the device; the app cannot verify their uniqueness, and shared or changing serials limit unit identification. Registry IDs and USB port locations are never saved as unit identities.

```json
"keyboard_types": {
  "1234:5678": "win",
  "1234:5678:serial:unit-A": "mac"
}
```

Manual switching records overrides automatically, using a fresh device snapshot at click time. If multiple external keyboards are connected, the selection is remembered for all of them; built-in keyboards are excluded. Preview remembers overrides only in memory. Manual switching stops with an error if it cannot enumerate keyboards. Configuration edits require restarting the app.

Find your IDs and exposed serial numbers without applying any mappings:

```sh
build/CmdKeySwitcher.app/Contents/MacOS/cmd-key-switcher --list-keyboards
```

A custom config path is available with `--config /path/to/config.json`. Only one instance per config path can run at a time. Invalid configuration fails startup rather than silently replacing it.

Errors appear in the menu icon's tooltip and, when installed, `~/Library/Logs/CmdKeySwitcher/stderr.log`. Failed automatic HID commands or enumeration attempts receive up to three retries after delays of 2, 4, and 8 seconds. A new device notification or wake starts a fresh retry budget; a manual switch cancels pending retries. Failed manual commands keep the current selection.

## Implementation references

The native bridge uses Apple's [NSStatusBar](https://developer.apple.com/documentation/appkit/nsstatusbar) and [IOService registry notifications](https://developer.apple.com/documentation/iokit/1514362-ioserviceaddmatchingnotification). Go owns classification, configuration, command execution, and switching behavior. The UI and device callbacks stay on the main OS thread. IORegistry service arrival/removal callbacks replace periodic polling; a one-shot 250 ms debounce combines closely spaced notifications. One-shot retry timers run only after failures. Startup detection and wake checks remain in place.

The build produces an unsigned app for local use, for the current machine's architecture. Distribution requires signing/notarization separately.
