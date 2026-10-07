#include "native.h"
#import <Cocoa/Cocoa.h>

/**
 * @brief Forwards the native close action to the active application on the main thread.
 */
extern void appClose(void);

/**
 * @brief Forwards confirmed native cleanup to the active application on the
 * main thread.
 */
extern void appPrepareForUninstall(void);

@interface SettingsController : NSObject <NSMenuDelegate>
@property(strong) NSMenuItem *startupItem;
@end

@implementation SettingsController

/**
 * @brief Refreshes launch-at-login status before the settings menu opens.
 *
 * @param menu The unused menu being updated.
 */
- (void)menuNeedsUpdate:(NSMenu *)menu {
    refreshLoginStartupItem(self.startupItem);
}

/**
 * @brief Toggles login startup and refreshes the associated menu item.
 *
 * @param sender The unused menu action source.
 */
- (void)toggleStartup:(id)sender {
    toggleLoginStartup();
    refreshLoginStartupItem(self.startupItem);
}

/**
 * @brief Asks for confirmation before invoking Go cleanup.
 *
 * @param sender The unused menu action source.
 */
- (void)prepareForUninstall:(id)sender {
    NSAlert *alert = [NSAlert new];
    alert.messageText = @"Prepare for uninstall?";
    alert.informativeText =
        @"This disables launch at login, deletes saved keyboard preferences and "
        @"logs, and closes the app. The current keyboard mapping stays applied. "
        @"Afterward, move CmdKeySwitcher from Applications to the Trash.";
    alert.alertStyle = NSAlertStyleWarning;
    [alert addButtonWithTitle:@"Cancel"];
    [alert addButtonWithTitle:@"Prepare for uninstall"];
    [NSApp activateIgnoringOtherApps:YES];
    if ([alert runModal] == NSAlertSecondButtonReturn)
        appPrepareForUninstall();
}

/**
 * @brief Forwards the close menu action to Go.
 *
 * @param sender The unused menu action source.
 */
- (void)closeApp:(id)sender {
    appClose();
}
@end

/**
 * @brief Creates the settings menu and retains its controller for callbacks.
 *
 * @return The settings submenu managed by ARC.
 */
NSMenu *settingsMenu(void) {
    // NSMenu delegates and item targets are weak references.
    static SettingsController *controller;
    controller = [SettingsController new];
    NSMenu *menu = [[NSMenu alloc] initWithTitle:@"Settings"];
    menu.delegate = controller;
    menu.autoenablesItems = NO;
    controller.startupItem = [[NSMenuItem alloc] initWithTitle:@"Launch at login"
                                                        action:@selector(toggleStartup:)
                                                 keyEquivalent:@""];
    controller.startupItem.target = controller;
    [menu addItem:controller.startupItem];
    [menu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *cleanup = [[NSMenuItem alloc] initWithTitle:@"Prepare for uninstall…"
                                                     action:@selector(prepareForUninstall:)
                                              keyEquivalent:@""];
    cleanup.target = controller;
    [menu addItem:cleanup];
    NSMenuItem *close = [[NSMenuItem alloc] initWithTitle:@"Close app"
                                                   action:@selector(closeApp:)
                                            keyEquivalent:@""];
    close.target = controller;
    [menu addItem:close];
    return menu;
}

/**
 * @brief Shows a modal warning alert on the main thread.
 *
 * @param message The UTF-8 error text.
 *
 * @note Returns after the alert is dismissed.
 */
void showAppError(const char *message) {
    NSAlert *alert = [NSAlert new];
    alert.messageText = @"CmdKeySwitcher";
    alert.informativeText = [NSString stringWithUTF8String:message];
    alert.alertStyle = NSAlertStyleWarning;
    [alert addButtonWithTitle:@"OK"];
    [NSApp activateIgnoringOtherApps:YES];
    [alert runModal];
}
