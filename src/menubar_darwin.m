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
- (void)flip:(id)sender {
    appSwitch();
}
@end
void updateMenu(const char *mode, const char *error) {
    NSString *type = [NSString stringWithUTF8String:mode];
    status.button.title = [type isEqualToString:@"mac"] ? @"⌘ mac" : @"⊞ win";
    toggle.title = [type isEqualToString:@"mac"] ? @"Switch to win" : @"Switch to mac";
    NSString *message = [NSString stringWithUTF8String:error];
    status.button.toolTip =
        message.length ? message : [NSString stringWithFormat:@"Keyboard mapping: %@", type];
}
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
