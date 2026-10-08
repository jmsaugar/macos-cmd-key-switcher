#include "native.h"
#import <Cocoa/Cocoa.h>

extern void appClose(void);

extern void appPrepareForUninstall(void);

@interface SettingsController : NSObject <NSMenuDelegate>
@property(strong) NSMenuItem *startupItem;
@end

@implementation SettingsController

/** Refreshes the launch-at-login state before the settings submenu is displayed. */
- (void)menuNeedsUpdate:(NSMenu *)menu {
    refreshLoginStartupItem(self.startupItem);
}

/** Handles the startup action and refreshes the menu item from the resulting OS state. */
- (void)toggleStartup:(id)sender {
    toggleLoginStartup();
    refreshLoginStartupItem(self.startupItem);
}

/** Confirms the cleanup action before asking Go to prepare the app for removal. */
- (void)prepareForUninstall:(id)sender {
    NSAlert *alert = [NSAlert new];
    alert.messageText = @"Prepare for uninstall?";
    alert.informativeText =
        @"This disables launch at login, deletes saved keyboard preferences and "
        @"logs, and closes the app. The current keyboard mapping stays applied. "
        @"Afterward, move Cmd Key Switcher from Applications to the Trash.";
    alert.alertStyle = NSAlertStyleWarning;
    [alert addButtonWithTitle:@"Cancel"];
    [alert addButtonWithTitle:@"Prepare for uninstall"];
    [NSApp activateIgnoringOtherApps:YES];
    if ([alert runModal] == NSAlertSecondButtonReturn)
        appPrepareForUninstall();
}

/** Asks Go to close the app while preserving its preferences and startup registration. */
- (void)closeApp:(id)sender {
    appClose();
}
@end

/** Builds the settings submenu and retains its controller for menu callbacks. */
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

/** Presents a modal warning with the supplied application error. */
void showAppError(const char *message) {
    NSAlert *alert = [NSAlert new];
    alert.messageText = @"Cmd Key Switcher";
    alert.informativeText = [NSString stringWithUTF8String:message];
    alert.alertStyle = NSAlertStyleWarning;
    [alert addButtonWithTitle:@"OK"];
    [NSApp activateIgnoringOtherApps:YES];
    [alert runModal];
}
