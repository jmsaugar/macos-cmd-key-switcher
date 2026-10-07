#include "native.h"
#import <Cocoa/Cocoa.h>

extern void appTick(void);

extern void appSwitch(void);

extern void appWake(void);
static NSStatusItem *status;
static NSMenuItem *toggle;

@interface Switcher : NSObject

- (void)flip:(id)sender;
@end
@implementation Switcher

/** Forwards the switch menu action to the Go application. */
- (void)flip:(id)sender {
    appSwitch();
}
@end

/** Reflects the applied mapping and any error in the menu bar UI. */
void updateMenu(const char *mode, const char *error) {
    NSString *type = [NSString stringWithUTF8String:mode];
    status.button.title = [type isEqualToString:@"mac"] ? @"⌘ Mac" : @"⊞ Win";
    toggle.title =
        [type isEqualToString:@"mac"] ? @"Switch to Win keyboard" : @"Switch to Mac keyboard";
    NSString *message = [NSString stringWithUTF8String:error];
    status.button.toolTip =
        message.length ? message : [NSString stringWithFormat:@"Keyboard mapping: %@", type];
}

/** Builds the menu bar UI, starts device watching, and runs the Cocoa event loop. */
void runApp(void) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        // The menu target is weak; retain the controller for the app lifetime.
        static Switcher *controller;
        controller = [Switcher new];
        status = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        NSMenu *menu = [NSMenu new];
        toggle = [[NSMenuItem alloc] initWithTitle:@"Switch to Win keyboard"
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
                    usingBlock:^(NSNotification *note) {
                        appWake();
                    }];
        [NSApp run];
        endKeyboardWatching();
        cancelRetry();
    }
}

/** Enables or disables top-level actions while Go coordinates shutdown. */
void setMenuEnabled(int enabled) {
    for (NSMenuItem *item in status.menu.itemArray)
        item.enabled = enabled;
}

/** Stops and wakes the Cocoa event loop so Go can finish resource cleanup. */
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
