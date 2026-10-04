# CmdKeySwitcher

A Go macOS menu bar app with a small native Cocoa/IOKit bridge. No external Go dependencies. Requires macOS, mise, and Xcode Command Line Tools (`xcode-select --install`). `go.mod` declares Go 1.23 as the minimum language/toolchain version; development uses the exact Go version in `.config/mise.toml`.

Application source and tests live in `src/`, including the Go code, Objective-C bridge, and app bundle metadata in `src/Info.plist`. `go.mod` stays at the repository root; build and tooling commands run from that root.

Development tools are pinned in `.config/mise.toml` and `.config/mise.lock` for Apple Silicon and Intel Macs. The formatter uses mise's Conda backend, which installs its supporting packages in mise-managed directories; this is development tooling, not an application dependency. Apple Clang and the macOS SDK come separately from Command Line Tools or Xcode and are not installed or pinned by mise. Builds use Apple Clang from the selected developer tools, not the formatter's LLVM libraries.

```sh
mise trust .config/mise.toml
make setup
make format
make check
make test
make build
```

`make format` updates Go and native source formatting. `make check` verifies formatting, runs Go vet, checks shell/plist syntax, and runs Clang's static analyzer. `make test` runs tests; `make build` produces the local app bundle. These commands do not launch or install the app or modify keyboard mappings. Analyzer output is saved under `build/`. `make setup` uses Bash to read the tool-name column from `mise ls --local --no-header` and pass the project tool names to `mise install --locked`. Tool names are not duplicated in the Makefile, and unrelated globally configured tools are excluded. Setup stops if discovery fails or finds no tools, rather than running a bare install. Make invokes Go and clang-format through `mise exec`, so shell activation is optional. Run the locked installation first; mise commands may otherwise install missing tools.

To change tool versions, edit `.config/mise.toml`, run `mise lock --platform macos-arm64,macos-x64`, then `make setup`. Commit both files together. Formatting rules are committed in `src/.clang-format`; Go uses standard gofmt rules. Verify changes on macOS because the bridge depends on Cocoa and IOKit. To record the Apple toolchain used locally, run `xcrun clang --version` and `xcrun --show-sdk-version`.

Install the app separately when ready:

```sh
make test
make build
make install
```

`make install` copies the app to `~/Applications`, registers a user LaunchAgent, and starts it immediately. It starts automatically at **user login**, when macOS provides a menu bar. It does not run before login. Installation requires permission to write outside the project when invoked from a sandboxed coding tool.

The menu bar shows `⌘ mac` or `⊞ win`. Its dropdown contains exactly one action, “Switch to win” or “Switch to mac”. To stop and remove the installed app, run `make uninstall`; this also deletes the saved configuration (including its lock file) and logs from `~/Library/Application Support/CmdKeySwitcher` and `~/Library/Logs/CmdKeySwitcher`. Uninstalling does not reset keyboard mappings previously applied by the app. The LaunchAgent restarts the app if it exits.

## Detection and switching

At startup and when IOKit reports a keyboard connecting or disconnecting, the app chooses:

- No external keyboards, or only Apple/Mac-classified external keyboards: `mac`.
- Any Windows-classified external keyboard: `win`.

Apple vendor ID 1452 defaults to `mac`; other external vendors default to `win`. HID metadata cannot reliably identify a third-party keyboard's key layout or Mac/Windows hardware mode. Add a device override for those keyboards. Internal keyboards are ignored using the Built-In property and SPI/i2c transport. Detection requires the primary HID usage to be Generic Desktop / Keyboard. Secondary keyboard collections on media/audio peripherals are excluded, even if they advertise normal keys. A composite device whose keyboard collection is secondary will also be excluded by this conservative rule.

Manual selection also saves a classification override for every currently connected external keyboard after the command succeeds. Those overrides are used on reconnect and restart. With no external keyboard, manual switching changes only `type`; startup detection still selects `mac`. After wake, the selected mapping is reapplied. If multiple external keyboards have different layouts, `win` takes precedence, and the mapping applies globally to all keyboards, including the built-in keyboard.

The mappings are defined in `src/mapping.go`: `win` swaps **left Command and left Option**, and `mac` maps those two keys to themselves. Right-side modifiers are unchanged. `hidutil property --set` replaces `UserKeyMapping`, so an existing mapping from another tool is overwritten.


The exact commands are shown below for reference. Running them directly changes the active keyboard mapping. Usage IDs are decimal: `30064771298` (`0x7000000E2`) is left Option, and `30064771299` (`0x7000000E3`) is left Command.

```sh
# win mode
/usr/bin/hidutil property --set '{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771299}]}'

# mac mode
/usr/bin/hidutil property --set '{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771299}]}'
```

## Configuration

The app writes `~/Library/Application Support/CmdKeySwitcher/config.json`. It updates `type` only after the HID command succeeds, and saves atomically. The HID commands run directly as `/usr/bin/hidutil property --set <mapping>` without a shell. Detection reads IORegistry device metadata and subscribes to service arrival/removal notifications, without creating HID input clients or reading keystrokes. Input Monitoring and Accessibility access are not needed for this detection path.

The configuration stores only `type` and `keyboard_types`. HID mappings are constants in `src/mapping.go`; changing them requires rebuilding the app. Overrides use `vendor:product_id:serial:<URL-escaped serial>` when a nonblank HID serial number is exposed, otherwise decimal `vendor:product_id`. Detection checks the unit override first, then the model override, then the vendor default. Serial numbers are supplied by the device; the app cannot verify their uniqueness, and shared or changing serials limit unit identification. Registry IDs and USB port locations are never saved as unit identities.

```json
"keyboard_types": {
  "1234:5678": "win",
  "1234:5678:serial:unit-A": "mac"
}
```

Manual switching records overrides automatically, using a fresh device snapshot at click time. If multiple external keyboards are connected, the selection is remembered for all of them; built-in keyboards are excluded. Manual switching stops with an error if it cannot enumerate keyboards. Configuration edits require restarting the app.

The app uses the fixed configuration location above and allows only one instance. Invalid configuration fails startup rather than silently replacing it. There are no preview, simulation, diagnostic-listing, or custom-config runtime modes; running the app applies real keyboard mappings.

Errors appear in the menu icon's tooltip and, when installed, `~/Library/Logs/CmdKeySwitcher/stderr.log`. Failed automatic HID commands or enumeration attempts receive up to three retries after delays of 2, 4, and 8 seconds. A new device notification or wake starts a fresh retry budget; a manual switch cancels pending retries. Failed manual commands keep the current selection.

## Implementation references

The native bridge uses Apple's [NSStatusBar](https://developer.apple.com/documentation/appkit/nsstatusbar) and [IOService registry notifications](https://developer.apple.com/documentation/iokit/1514362-ioserviceaddmatchingnotification). Go owns classification, configuration, command execution, and switching behavior. The UI and device callbacks stay on the main OS thread. IORegistry service arrival/removal callbacks replace periodic polling; a one-shot 250 ms debounce combines closely spaced notifications. One-shot retry timers run only after failures. Startup detection and wake checks remain in place.

The build produces an unsigned app for local use, for the current machine's architecture. Distribution requires signing/notarization separately.

## Source organization

The app remains one Go package with a small native bridge. Responsibilities are separated by file:

| Files in `src/` | Responsibility |
|---|---|
| `main_darwin.go` | Startup, config path, instance lock, and main-thread ownership |
| `config.go`, `config_test.go` | Configuration defaults, validation, and atomic file persistence |
| `keyboard.go`, `keyboard_test.go` | Device identity, remembered classifications, vendor rules, and fingerprints |
| `detection_darwin.go` | Respond to device changes and wake; choose automatic selections |
| `mapping.go` | Fixed modifier mappings and hidutil execution |
| `switching_darwin.go`, `events_darwin_test.go` | Apply selections, save manual preferences, and coordinate UI state |
| `retry.go`, `retry_darwin.go`, `retry_test.go` | Retry budget and automatic retry orchestration |
| `bridge_darwin.go`, `native.h` | cgo conversions and native callbacks |
| `menubar_darwin.m` | Cocoa menu, click handling, wake listener, and app event loop |
| `keyboard_registry_darwin.m` | Read-only registry enumeration and primary keyboard filtering |
| `keyboard_notifications_darwin.m` | Registry arrival/removal subscriptions and notification debounce |
| `retry_darwin.m` | Main-thread one-shot retry timers |
| `Info.plist`, `.clang-format` | Bundle metadata and native formatting rules |

Go and native callbacks run on the main OS thread. Each native source is compiled into the same executable; no separate runtime services or libraries were added.
