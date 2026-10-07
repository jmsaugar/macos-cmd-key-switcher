#ifndef CMD_KEY_SWITCHER_NATIVE_H
#define CMD_KEY_SWITCHER_NATIVE_H

// Runtime calls use the main OS thread unless a function documents otherwise.
// These declarations are the native API contracts; implementation comments
// explain local behavior without duplicating the contracts.

/**
 * @brief Creates the menu bar UI and runs the Cocoa event loop on the main thread.
 *
 * @note Blocks until the Cocoa event loop stops, then releases device observers and timers.
 */
void runApp(void);

/**
 * @brief Enumerates keyboard registry metadata without opening input devices.
 *
 * @note Installs HID service notifications if watching is active and setup is still pending.
 * @return An allocated UTF-8 JSON array, including an empty array when no keyboards match.
 *         Returns NULL if notification setup, enumeration, or serialization fails.
 *         The caller must free a non-NULL string with free().
 */
char *keyboardJSON(void);

/**
 * @brief Updates the status title, switch action, and tooltip on the main thread.
 *
 * @param mode A non-NULL UTF-8 mapping name: "mac" or "win".
 * @param error Non-NULL UTF-8 error text, or an empty string for the normal tooltip.
 */
void updateMenu(const char *mode, const char *error);

/**
 * @brief Replaces the retry timer with a one-shot main-run-loop callback.
 *
 * @param seconds The delay, in seconds, before calling the Go appRetry callback.
 */
void scheduleRetry(double seconds);

/**
 * @brief Invalidates and releases the retry timer.
 */
void cancelRetry(void);

/**
 * @brief Dispatches the Go mapping completion callback to the main queue.
 *
 * @note May be called from any thread. The worker must enqueue its Go result before
 *       calling this function; the Go callback receives that result on the main thread.
 */
void notifyMappingComplete(void);

/**
 * @brief Requests that the Cocoa event loop return and posts an event to wake it.
 * @note Does not terminate the process directly; Go performs final resource cleanup.
 */
void stopApp(void);

/**
 * @brief Enables or disables top-level menu actions on the main thread.
 *
 * @param enabled Zero to disable actions or nonzero to enable them.
 */
void setMenuEnabled(int enabled);

/**
 * @brief Shows a modal warning alert on the main thread.
 *
 * @param message Non-NULL UTF-8 error text.
 *
 * @note Returns after the alert is dismissed.
 */
void showAppError(const char *message);

/**
 * @brief Removes launch-at-login registration, treating an absent
 * registration as success.
 *
 * @note Does not quit the main app or remove its configuration and logs.
 * @return NULL on success, or an allocated UTF-8 error message to release with free().
 */
char *unregisterLoginStartup(void);

#ifdef __OBJC__
@class NSMenu;
@class NSMenuItem;

/**
 * @brief Creates the settings menu and retains its controller for callbacks.
 *
 * @return A new settings submenu. The caller must retain it, for example by assigning
 *         it to a parent menu item's submenu property.
 */
NSMenu *settingsMenu(void);

/**
 * @brief Updates the launch-at-login menu item from ServiceManagement
 * status.
 *
 * @param item The menu item whose title, state, and tooltip are updated.
 */
void refreshLoginStartupItem(NSMenuItem *item);

/**
 * @brief Toggles launch-at-login registration or prompts when approval is
 * required.
 *
 * @note Approval prompts offer System Settings, disabling registration, or canceling.
 *       Registration and unregistration errors are shown in an alert.
 */
void toggleLoginStartup(void);
#endif

// Native device notification lifecycle, shared by registry and menu implementations.

/**
 * @brief Marks device watching active; a later ensureKeyboardWatching call installs notifications.
 */
void beginKeyboardWatching(void);

/**
 * @brief Installs device notifications when watching is active and not yet
 * registered.
 *
 * @return 1 if watching is inactive or notifications are ready; 0 if setup fails.
 */
int ensureKeyboardWatching(void);

/**
 * @brief Stops watching and releases device timers, iterators, and
 * notification ports.
 */
void endKeyboardWatching(void);

#endif
