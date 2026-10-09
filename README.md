# Cmd Key Switcher

A Go macOS menu bar app with a small native Cocoa/IOKit/ServiceManagement bridge. No external Go dependencies. The app requires macOS 13 or later. Building requires mise and Xcode Command Line Tools (`xcode-select --install`). `go.mod` declares Go 1.23 as the minimum language/toolchain version; development uses the exact Go version in `.config/mise.toml`.

Application source and tests live in `src/`, including the Go code, Objective-C bridge, app bundle metadata in `src/Info.plist`, and the UserDefaults API declaration in `src/PrivacyInfo.xcprivacy`. `go.mod` is at the repository root; build and tooling commands run from that root.

The app icon source is `src/assets/AppIcon.png`. `make build` uses macOS's `sips` and `iconutil` through `scripts/build-icon.sh` to generate standard and Retina icon sizes and package them as `Contents/Resources/AppIcon.icns` before signing the bundle. `src/Info.plist` references it through Apple's [CFBundleIconFile](https://developer.apple.com/documentation/BundleResources/Information-Property-List/CFBundleIconFile) key. Replace the source with a square PNG of at least 1024 × 1024 pixels and rebuild to change the Finder/Application icon. The menu bar shows the current `mac`/`win` mode as text.

Development tools are pinned in `.config/mise.toml` and `.config/mise.lock` for Apple Silicon and Intel Macs. The formatter uses mise's Conda backend, which installs its supporting packages in mise-managed directories; this is development tooling, not an application dependency. Apple Clang and the macOS SDK come separately from Command Line Tools or Xcode and are not installed or pinned by mise. Builds use Apple Clang from the selected developer tools, not the formatter's LLVM libraries.

```sh
mise trust .config/mise.toml
make setup
make format
make check
make test
make build
```

`make format` updates Go and native source formatting. `make check` verifies formatting, runs Go vet, checks shell/plist syntax, and runs Clang's static analyzer with documentation warnings enabled. `make test` runs tests; `make build` produces the local app bundle. These commands do not launch or install the app or modify keyboard mappings. Analyzer output is saved under `build/`. `make setup` uses Bash to read the tool-name column from `mise ls --local --no-header` and pass the project tool names to `mise install --locked`. Tool names are not duplicated in the Makefile, and unrelated globally configured tools are excluded. Setup stops if discovery fails or finds no tools, rather than running a bare install. Make invokes Go and clang-format through `mise exec`, so shell activation is optional. Run the locked installation first; mise commands may otherwise install missing tools.

To change tool versions, edit `.config/mise.toml`, run `mise lock --platform macos-arm64,macos-x64`, then `make setup`. Commit both files together. Formatting rules are committed in `src/.clang-format`; Go uses standard gofmt rules. Verify changes on macOS because the bridge depends on Cocoa and IOKit. To record the Apple toolchain used locally, run `xcrun clang --version` and `xcrun --show-sdk-version`.

## Install and use

After `make build`, copy `build/Cmd Key Switcher.app` into `/Applications` or `~/Applications` using Finder, then double-click it. No installer or terminal command is required to run the built app. Opening it applies real keyboard mappings.

Close an already-running app before replacing its bundle. Enable login startup from the app's Settings submenu if desired.

The menu bar shows `[⌘] Mac` or `[⊞] Win`. The dropdown has two entries:

```text
Switch to Win keyboard  (or Switch to Mac keyboard)
Settings ▸
    ✓ Launch at login
    ────────────────
      Prepare for uninstall…
      Close app
```

- **Launch at login** controls `SMAppService.mainAppService` registration. A fresh installation starts with login startup disabled. The submenu reads the macOS status each time it opens; this preference is managed by macOS separately from keyboard preferences, and normal launches never register automatically. A mixed checkmark and “approval required” label indicate that registration needs approval. Clicking it offers to open Login Items in System Settings or turn registration off. Disabling startup leaves the app running. Startup runs at user login, not before login, and does not supervise or automatically restart the app after it exits.
- **Close app** exits without disabling login startup or deleting preferences/logs. It waits for an active HID command to finish and preserves its result; pending commands are discarded. Applied mappings stay in effect.
- **Prepare for uninstall…** asks for confirmation, unregisters any known login-startup registration, waits for an active HID command, deletes the app's logs and lock directory, clears its UserDefaults domain, then quits. It does not delete the app bundle or reset applied keyboard mappings. If an attempted startup unregistration fails, data is kept. Preferences are cleared only after file removal succeeds. If cleanup fails, the app reports the error and permits another attempt after restoring its instance lock and logging; some files may have already been removed. If those resources cannot be restored, it reports that failure and closes instead. UserDefaults clearing is submitted to macOS without waiting for disk persistence.

To uninstall, choose **Settings → Prepare for uninstall…**, then move the app from Applications to the Trash. Launching the app after cleanup starts with default configuration and an empty `keyboard_types` map, detects connected keyboards, creates preferences and log/lock files, and leaves login startup disabled until you explicitly enable it again.

Deleting the bundle directly does not run the app's cleanup: preferences, logs, and the lock file remain, and an already-running process is not necessarily stopped. Once the bundle is actually deleted, macOS cannot launch that copy at login, but its login entry may remain visible temporarily while macOS performs maintenance. Apple describes this delayed removal in its [ServiceManagement sample documentation](https://developer.apple.com/documentation/ServiceManagement/updating-your-app-package-installer-to-use-the-new-service-management-api). Neither direct deletion nor preparing for uninstall resets the applied keyboard mapping.

## Detection and switching

At startup and when IOKit reports a keyboard connecting or disconnecting, the app chooses:

- No external keyboards, or only Apple/Mac-classified external keyboards: `mac`.
- Any Windows-classified external keyboard: `win`.

Apple vendor ID 1452 defaults to `mac`; other external vendors default to `win`. HID metadata cannot reliably identify a third-party keyboard's key layout or Mac/Windows hardware mode. Add a device override for those keyboards. Internal keyboards are ignored using the Built-In property and SPI/i2c transport. Detection requires the primary HID usage to be Generic Desktop / Keyboard. Secondary keyboard collections on media/audio peripherals are excluded, even if they advertise normal keys. A composite device whose keyboard collection is secondary will also be excluded by this conservative rule.

When a manual command succeeds and no newer request is pending, its selection is saved as a classification override for every external keyboard in the snapshot taken at click time. Those overrides are used on reconnect and restart. With no external keyboard, manual switching changes only `type`; startup detection selects `mac`. After wake, unchanged devices cause the latest requested mapping to be reapplied; changed devices cause automatic classification to choose the mapping. If multiple external keyboards have different layouts, `win` takes precedence, and the mapping applies globally to all keyboards, including the built-in keyboard.

The mappings are defined in `src/mapping.go`: `win` swaps **left Command and left Option**, and `mac` maps those two keys to themselves. Right-side modifiers are unchanged. `hidutil property --set` replaces `UserKeyMapping`, so an existing mapping from another tool is overwritten.

The exact commands are shown below for reference. Running them directly changes the active keyboard mapping. Usage IDs are decimal: `30064771298` (`0x7000000E2`) is left Option, and `30064771299` (`0x7000000E3`) is left Command.

```sh
# win mode
/usr/bin/hidutil property --set '{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771299}]}'

# mac mode
/usr/bin/hidutil property --set '{"UserKeyMapping":[{"HIDKeyboardModifierMappingSrc":30064771298,"HIDKeyboardModifierMappingDst":30064771298},{"HIDKeyboardModifierMappingSrc":30064771299,"HIDKeyboardModifierMappingDst":30064771299}]}'
```

## Configuration

The app stores a `configuration` dictionary in [NSUserDefaults](https://developer.apple.com/documentation/foundation/userdefaults), in its bundle identifier's domain, `local.cmdkeyswitcher`. That dictionary contains `type` and `keyboard_types`. Go validates each configuration and submits both values together after a successful HID command. macOS updates its in-memory preferences immediately and persists them asynchronously; a successful submission does not acknowledge a completed disk write. The app does not call `synchronize()` or manage preference files directly.

To inspect the stored configuration without changing it:

```sh
defaults read local.cmdkeyswitcher configuration
```

The single-instance lock is `~/Library/Application Support/CmdKeySwitcher/instance.lock`. Logs are stored at `~/Library/Logs/CmdKeySwitcher/app.log`.

The HID commands run directly as `/usr/bin/hidutil property --set <mapping>` without a shell. Detection reads IORegistry device metadata and subscribes to service arrival/removal notifications, without creating HID input clients or reading keystrokes. Input Monitoring and Accessibility access are not needed for this detection path.

HID mappings are constants in `src/mapping.go`; changing them requires rebuilding the app. Overrides use `vendor:product_id:serial:<URL-escaped serial>` when a nonblank HID serial number is exposed, otherwise decimal `vendor:product_id`. Detection checks the unit override first, then the model override, then the vendor default. Serial numbers are supplied by the device; the app cannot verify their uniqueness, and shared or changing serials limit unit identification. Registry IDs and USB port locations are never saved as unit identities.

Example configuration dictionary, shown as JSON:

```json
{
  "type": "win",
  "keyboard_types": {
    "1234:5678": "win",
    "1234:5678:serial:unit-A": "mac"
  }
}
```

Manual switching records overrides automatically, using a fresh device snapshot at click time. If multiple external keyboards are connected, the selection is remembered for all of them; built-in keyboards are excluded. Manual switching stops with an error if it cannot enumerate keyboards. The app loads preferences at startup; external preference changes require restarting it.

The app reads only its own UserDefaults domain and allows only one instance. Invalid configuration stops startup with an error. Run the built app bundle so its bundle identifier is available to the preferences bridge. Running the app applies real keyboard mappings.

Runtime mapping, detection, and preference-submission errors appear in the menu icon's tooltip. Mapping failures, preference-submission failures, and manual detection failures are also written to `~/Library/Logs/CmdKeySwitcher/app.log`, including when launched from Finder or at login. Automatic detection failures appear in the tooltip; exhausting automatic retries is logged. UserDefaults does not report a per-update disk-write result. Login-registration and cleanup errors appear in native dialogs. Failed automatic HID commands or enumeration attempts receive up to three retries after delays of 2, 4, and 8 seconds. A new device notification or wake starts a fresh retry budget; a manual switch cancels pending retries. Failed manual commands leave the displayed mode unchanged. If a mapping succeeds but preference submission fails, the applied mode remains active and the error is reported.

## Implementation references

The native bridge uses Apple's [NSStatusBar](https://developer.apple.com/documentation/appkit/nsstatusbar) and [IOService registry notifications](https://developer.apple.com/documentation/iokit/1514362-ioserviceaddmatchingnotification). Go owns classification, configuration, command execution, and switching behavior. The UI and device callbacks stay on the main OS thread. HID commands run in a Go worker with a five-second timeout; completion returns to the main thread before changing state, configuration, or the menu. Only one HID command runs at a time; newer requests replace pending work, and rapid manual clicks toggle the requested mode. Superseded operations do not record manual overrides or schedule retries. IORegistry service arrival/removal callbacks trigger detection; a one-shot 250 ms debounce combines closely spaced notifications. One-shot retry timers run only after failures. The app also detects keyboards at startup and checks them after wake.

The native login-startup code uses Apple's [SMAppService](https://developer.apple.com/documentation/servicemanagement/smappservice?language=objc). Builds target macOS 13 and produce a bundle for the current machine's architecture. `make build` ad-hoc signs the complete bundle for local use because ServiceManagement requires code signing. You can supply a signing identity with `make build SIGN_IDENTITY="<identity>"`. Local ad-hoc signing is not a distribution signature and does not provide stable identity across rebuilt versions; verify login registration on your Mac, and use a stable development identity if local registration fails. For distribution outside the Mac App Store, follow Apple's [Developer ID signing and notarization guidance](https://developer.apple.com/developer-id/). The build does not perform notarization.

`src/PrivacyInfo.xcprivacy` declares UserDefaults access for this app's own preferences (reason `CA92.1`) and is copied into `Contents/Resources` before signing. See Apple's [required reason API guidance](https://developer.apple.com/documentation/technotes/tn3183-adding-required-reason-api-entries-to-your-privacy-manifest).

Cleanup skips startup unregistration when macOS initially reports `NotRegistered` or `NotFound`. For other initial states, successful unregistration or `kSMErrorJobNotFound` allows cleanup; other errors stop cleanup even if a later status is `NotFound`.

## Source organization

Go doc comments use complete sentences beginning with the documented symbol, following the [Go doc-comment conventions](https://go.dev/doc/comment). They describe behavior and contracts rather than repeating parameter names or obvious test stubs. Detailed native API documentation lives in `src/native.h`, with ownership, thread requirements, and meaningful parameter/return descriptions. Objective-C implementations have concise documentation above functions and methods, including private helpers; inline comments explain non-obvious decisions. `make check` enables Clang's `-Wdocumentation` and `-Wdocumentation-pedantic` diagnostics for native documentation.

The app uses one Go package with a small native bridge. Responsibilities are separated by file:

| Files in `src/` | Responsibility |
|---|---|
| `main_darwin.go`, `app_darwin.go`, `app_darwin_test.go` | Startup, instance lock, application state ownership, dependencies, and state isolation tests |
| `instance_darwin.go`, `instance_darwin_test.go` | Single-instance file lock and recovery after partial cleanup |
| `paths.go`, `logging.go` | Fixed log/lock paths and persistent logging for UI/login launches |
| `lifecycle_darwin.go`, `lifecycle_darwin_test.go` | Close/cleanup orchestration, draining active work, and failure handling |
| `config.go`, `config_test.go` | Configuration defaults, validation, and storage contract tests |
| `preferences_darwin.go`, `preferences_darwin.m` | cgo transport and UserDefaults dictionary read/write/clear |
| `preferences_darwin_test.go`, `testdata/preferences_bridge_test.m` | Isolated native bridge tests using an in-memory defaults replacement |
| `keyboard.go`, `keyboard_test.go` | Device identity, remembered classifications, vendor rules, and fingerprints |
| `detection_darwin.go` | Respond to device changes and wake; choose automatic selections |
| `mapping.go`, `mapping_worker_darwin.go`, `mapping_worker_darwin_test.go` | Fixed mappings, background hidutil execution, serialized requests, and worker coordination tests |
| `mapping_completion_darwin.m` | Dispatch worker completion to the main thread |
| `switching_darwin.go`, `events_darwin_test.go` | Apply selections, save manual preferences, and coordinate UI state |
| `retry.go`, `retry_darwin.go`, `retry_test.go` | Retry budget and automatic retry orchestration |
| `bridge_darwin.go`, `native.h` | cgo conversions and native callbacks |
| `settings_bridge_darwin.go` | cgo callbacks and lifecycle/settings conversions |
| `menubar_darwin.m` | Cocoa menu, click handling, wake listener, and app event loop |
| `settings_darwin.m` | Settings submenu, cleanup confirmation, and error dialogs |
| `login_startup_darwin.m` | Main-app login registration, macOS status, and approval handling |
| `login_startup_darwin_test.go`, `testdata/login_startup_bridge_test.m` | Isolated native unregistration tests using a fake login service |
| `keyboard_registry_darwin.m` | Read-only registry enumeration and primary keyboard filtering |
| `keyboard_notifications_darwin.m` | Registry arrival/removal subscriptions and notification debounce |
| `retry_darwin.m` | Main-thread one-shot retry timers |
| `Info.plist`, `.clang-format` | Bundle metadata and native formatting rules |
| `PrivacyInfo.xcprivacy` | Declaration of the app's UserDefaults API usage |
| `assets/AppIcon.png` | Source image for the app bundle's Finder icon |

Application state and native callbacks run on the main OS thread; HID command execution runs in a Go worker. Production native sources compile into the app executable; native test harnesses compile into separate test executables.

Automated tests use in-memory preference stores and mock HID commands and native lifecycle operations; they do not change macOS preferences, keyboard mappings, or login registration. Native preference and login-startup tests compile isolated harnesses that replace UserDefaults and the main-app login service with in-memory fakes. Finder launch, the Settings submenu, real preference persistence, and actual login registration/approval need manual verification using the installed bundle. Passing tests and signature verification do not verify those UI/system interactions.

`App` owns the applied mode and keyboard preferences, device observations, active/pending mapping requests, retry budget, shutdown state, and per-instance dependencies. All state transitions run on the main OS thread. The native bridge holds the one running AppKit instance and forwards callbacks to it. Workers capture only a copied request, executor, result channel, and completion notifier; they never access mutable App state. Configuration maps are copied on construction to avoid sharing ownership with callers. Tests construct independent instances with mocked native/command dependencies. Shutdown stops device watching and retries, discards pending commands, and drains an active command before ending the native event loop. During cleanup, command completion is not saved to preferences; a successful command's mode is retained in memory if cleanup fails and the app resumes. The instance lock and log file remain open until shutdown returns to `main`, then close. Go and Objective-C exchange configuration as temporary JSON; UserDefaults stores a native dictionary.
