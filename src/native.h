#ifndef CMD_KEY_SWITCHER_NATIVE_H
#define CMD_KEY_SWITCHER_NATIVE_H

// Go-facing UI, registry snapshot, and retry timer functions.

/**
 * @brief Creates the menu bar UI and runs the Cocoa event loop on the main thread.
 *
 * @note Returns after the application stops.
 */
void runApp(void);

/**
 * @brief Enumerates keyboard registry metadata without opening input devices.
 *
 * @return An allocated UTF-8 JSON snapshot, or NULL on failure; the caller must free the string.
 */
char *keyboardJSON(void);

/**
 * @brief Updates the status title, switch action, and tooltip on the main thread.
 *
 * @param mode A UTF-8 mapping name.
 * @param error UTF-8 error text or an empty string.
 */
void updateMenu(const char *mode, const char *error);

/**
 * @brief Replaces the retry timer with a one-shot main-run-loop callback.
 *
 * @param seconds The delay before calling appRetry.
 */
void scheduleRetry(double seconds);

/**
 * @brief Invalidates and releases the retry timer.
 */
void cancelRetry(void);

/**
 * @brief Dispatches the Go mapping completion callback to the main queue.
 *
 * @note The callback runs asynchronously.
 */
void notifyMappingComplete(void);

/**
 * @brief Stops Cocoa and posts an event to wake the main event loop.
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
 * @param message The UTF-8 error text.
 *
 * @note Returns after the alert is dismissed.
 */
void showAppError(const char *message);

/**
 * @brief Removes launch-at-login registration, treating an absent
 * registration as success.
 *
 * @return NULL on success, or an allocated UTF-8 error message that the caller must free.
 */
char *unregisterLoginStartup(void);

#ifdef __OBJC__
@class NSMenu;
@class NSMenuItem;

/**
 * @brief Creates the settings menu and retains its controller for callbacks.
 *
 * @return The settings submenu managed by ARC.
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
 * @note Registration errors are shown in an alert.
 */
void toggleLoginStartup(void);
#endif

// Native device notification lifecycle, shared by registry and menu implementations.

/**
 * @brief Marks device watching active so notifications can be installed.
 */
void beginKeyboardWatching(void);

/**
 * @brief Installs device notifications when watching is active and not yet
 * registered.
 *
 * @return YES if watching is inactive or notifications are ready; NO if setup fails.
 */
int ensureKeyboardWatching(void);

/**
 * @brief Stops watching and releases device timers, iterators, and
 * notification ports.
 */
void endKeyboardWatching(void);

#endif
