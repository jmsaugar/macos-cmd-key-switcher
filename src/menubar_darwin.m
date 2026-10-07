#include "native.h"
#import <Cocoa/Cocoa.h>

/**
 * @brief Forwards a native device notification to the active application on the main
 * thread.
 */
extern void appTick(void);

/**
 * @brief Forwards a native manual switch action to the active application on the main
 * thread.
 */
extern void appSwitch(void);

/**
 * @brief Forwards a native wake notification to the active application on the main thread.
 */
extern void appWake(void);
static NSStatusItem *status;
static NSMenuItem *toggle;

@interface Switcher : NSObject

/**
 * @brief Forwards the menu switch action to Go.
 *
 * @param sender The unused menu action source.
 */
- (void)flip:(id)sender;
@end
@implementation Switcher

/**
 * @brief Forwards the menu switch action to Go.
 *
 * @param sender The unused menu action source.
 */
- (void)flip:(id)sender {
    appSwitch();
}
@end

/**
 * @brief Updates the status title, switch action, and tooltip on the main thread.
 *
 * @param mode A UTF-8 mapping name.
 * @param error UTF-8 error text or an empty string.
 */
void updateMenu(const char *mode, const char *error) {
    NSString *type = [NSString stringWithUTF8String:mode];
    status.button.title = [type isEqualToString:@"mac"] ? @"⌘ mac" : @"⊞ win";
    toggle.title = [type isEqualToString:@"mac"] ? @"Switch to win" : @"Switch to mac";
    NSString *message = [NSString stringWithUTF8String:error];
    status.button.toolTip =
        message.length ? message : [NSString stringWithFormat:@"Keyboard mapping: %@", type];
}

/**
 * @brief Creates the menu bar UI and runs the Cocoa event loop on the main thread.
 *
 * @note Returns after the application stops.
 */
void runApp(void) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        // The menu target is weak; retain the controller for the app lifetime.
        static Switcher *controller;
        controller = [Switcher new];
        status = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        NSMenu *menu = [NSMenu new];
        toggle = [[NSMenuItem alloc] initWithTitle:@"Switch to win"
                                            action:@selector(flip:)
                                     keyEquivalent:@""];
        toggle.target = controller;
        [menu addItem:toggle];
        NSMenuItem *settings = [[NSMenuItem alloc] initWithTitle:@"Settings"
                                                          action:nil
                                                   keyEquivalent:@""];
        settings.submenu = settingsMenu();
        [menu addItem:settings];
        menu.autoenablesItems = NO;
        status.menu = menu;
        beginKeyboardWatching();
        appTick();
        [[[NSWorkspace sharedWorkspace] notificationCenter]
            addObserverForName:NSWorkspaceDidWakeNotification
                        object:nil
                         queue:[NSOperationQueue mainQueue]
                    // Callback forwards the workspace wake notification to Go. Note is the unused
                    // workspace notification.
                    usingBlock:^(NSNotification *note) {
                        appWake();
                    }];
        [NSApp run];
        endKeyboardWatching();
        cancelRetry();
    }
}

/**
 * @brief Enables or disables top-level menu actions on the main thread.
 *
 * @param enabled Zero to disable actions or nonzero to enable them.
 */
void setMenuEnabled(int enabled) {
    for (NSMenuItem *item in status.menu.itemArray)
        item.enabled = enabled;
}

/**
 * @brief Stops Cocoa and posts an event to wake the main event loop.
 */
void stopApp(void) {
    [NSApp stop:nil];
    // Wake the event loop when shutdown finishes in a dispatched worker callback.
    [NSApp postEvent:[NSEvent otherEventWithType:NSEventTypeApplicationDefined
                                        location:NSZeroPoint
                                   modifierFlags:0
                                       timestamp:0
                                    windowNumber:0
                                         context:nil
                                         subtype:0
                                           data1:0
                                           data2:0]
             atStart:NO];
}
