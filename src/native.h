#ifndef CMD_KEY_SWITCHER_NATIVE_H
#define CMD_KEY_SWITCHER_NATIVE_H

// Go-facing UI, registry snapshot, and retry timer functions.
void runApp(void);
char *keyboardJSON(void);
void updateMenu(const char *mode, const char *error);
void scheduleRetry(double seconds);
void cancelRetry(void);
void notifyMappingComplete(void);
void stopApp(void);
void setMenuEnabled(int enabled);
void showAppError(const char *message);
// Returns an allocated error message, or NULL on success. Caller frees it.
char *unregisterLoginStartup(void);

#ifdef __OBJC__
@class NSMenu;
@class NSMenuItem;
NSMenu *settingsMenu(void);
void refreshLoginStartupItem(NSMenuItem *item);
void toggleLoginStartup(void);
#endif

// Native device notification lifecycle, shared by registry and menu implementations.
void beginKeyboardWatching(void);
int ensureKeyboardWatching(void);
void endKeyboardWatching(void);

#endif
