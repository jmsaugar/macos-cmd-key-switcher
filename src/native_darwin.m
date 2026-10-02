#include "native.h"
#import <Cocoa/Cocoa.h>
#import <IOKit/IOKitLib.h>
#import <IOKit/hid/IOHIDKeys.h>
#import <IOKit/hid/IOHIDUsageTables.h>
#include <stdlib.h>
extern void appTick(void);
extern void appSwitch(void);
extern void appWake(void);
extern void appRetry(void);
static NSStatusItem *status;
static NSMenuItem *toggle;
static IONotificationPortRef notificationPort;
static io_iterator_t addedDevices, removedDevices;
static NSTimer *deviceUpdateTimer;
static NSTimer *retryTimer;
static BOOL watchingDevices;
void cancelRetry(void);

// One-shot debounce: no periodic keyboard polling.
static void deviceChanged(void *context, io_iterator_t iterator) {
    // Drain the iterator to release services and rearm notifications.
    io_service_t service;
    BOOL changed = NO;
    while ((service = IOIteratorNext(iterator))) {
        changed = YES;
        IOObjectRelease(service);
    }
    if (!changed)
        return;
    if (!watchingDevices)
        return;
    cancelRetry();
    [deviceUpdateTimer invalidate];
    deviceUpdateTimer = [NSTimer timerWithTimeInterval:0.25
                                               repeats:NO
                                                 block:^(NSTimer *timer) {
                                                     deviceUpdateTimer = nil;
                                                     appTick();
                                                 }];
    [[NSRunLoop mainRunLoop] addTimer:deviceUpdateTimer forMode:NSRunLoopCommonModes];
}

void cancelRetry(void) {
    [retryTimer invalidate];
    retryTimer = nil;
}
void scheduleRetry(double seconds) {
    cancelRetry();
    retryTimer = [NSTimer timerWithTimeInterval:seconds
                                        repeats:NO
                                          block:^(NSTimer *timer) {
                                              retryTimer = nil;
                                              appRetry();
                                          }];
    [[NSRunLoop mainRunLoop] addTimer:retryTimer forMode:NSRunLoopCommonModes];
}
@interface Switcher : NSObject
- (void)flip:(id)sender;
@end
@implementation Switcher
- (void)flip:(id)sender {
    appSwitch();
}
@end
static void stopWatching(void) {
    if (addedDevices) {
        IOObjectRelease(addedDevices);
        addedDevices = 0;
    }
    if (removedDevices) {
        IOObjectRelease(removedDevices);
        removedDevices = 0;
    }
    if (notificationPort) {
        CFRunLoopSourceRef source = IONotificationPortGetRunLoopSource(notificationPort);
        if (source)
            CFRunLoopRemoveSource(CFRunLoopGetMain(), source, kCFRunLoopCommonModes);
        IONotificationPortDestroy(notificationPort);
        notificationPort = NULL;
    }
}
static BOOL startWatching(void) {
    if (notificationPort)
        return YES;
    notificationPort = IONotificationPortCreate(kIOMainPortDefault);
    if (!notificationPort)
        return NO;
    CFRunLoopSourceRef source = IONotificationPortGetRunLoopSource(notificationPort);
    if (!source) {
        stopWatching();
        return NO;
    }
    CFRunLoopAddSource(CFRunLoopGetMain(), source, kCFRunLoopCommonModes);
    kern_return_t result = IOServiceAddMatchingNotification(
        notificationPort, kIOFirstMatchNotification, IOServiceMatching("IOHIDDevice"),
        deviceChanged, NULL, &addedDevices);
    if (result != KERN_SUCCESS) {
        stopWatching();
        return NO;
    }
    // Empty both initial iterators before relying on future notifications.
    deviceChanged(NULL, addedDevices);
    result = IOServiceAddMatchingNotification(notificationPort, kIOTerminatedNotification,
                                              IOServiceMatching("IOHIDDevice"), deviceChanged, NULL,
                                              &removedDevices);
    if (result != KERN_SUCCESS) {
        stopWatching();
        return NO;
    }
    deviceChanged(NULL, removedDevices);
    return YES;
}
static BOOL isKeyboard(NSDictionary *properties) {
    // Auxiliary keyboard collections can exist on media peripherals (including
    // devices that advertise a full key range). Require the primary function
    // to be a keyboard instead of accepting any DeviceUsagePairs entry.
    return [properties[@(kIOHIDPrimaryUsagePageKey)] intValue] == kHIDPage_GenericDesktop &&
           [properties[@(kIOHIDPrimaryUsageKey)] intValue] == kHIDUsage_GD_Keyboard;
}
char *keyboardJSON(void) {
    @autoreleasepool {
        // Read registry metadata only: no HID clients, device opens, or input queues.
        if (watchingDevices && !startWatching())
            return NULL;
        io_iterator_t iterator = 0;
        if (IOServiceGetMatchingServices(kIOMainPortDefault, IOServiceMatching("IOHIDDevice"),
                                         &iterator) != KERN_SUCCESS)
            return NULL;
        NSMutableArray *result = [NSMutableArray array];
        io_service_t device;
        BOOL failed = NO;
        while ((device = IOIteratorNext(iterator))) {
            CFMutableDictionaryRef rawProperties = NULL;
            if (IORegistryEntryCreateCFProperties(device, &rawProperties, kCFAllocatorDefault, 0) !=
                KERN_SUCCESS) {
                IOObjectRelease(device);
                failed = YES;
                continue;
            }
            NSDictionary *properties = CFBridgingRelease(rawProperties);
            if (!isKeyboard(properties)) {
                IOObjectRelease(device);
                continue;
            }
            id product = properties[@(kIOHIDProductKey)];
            id vendor = properties[@(kIOHIDVendorIDKey)];
            id pid = properties[@(kIOHIDProductIDKey)];
            id serial = properties[@(kIOHIDSerialNumberKey)];
            if (![serial isKindOfClass:[NSString class]])
                serial = @"";
            id builtIn = properties[@"Built-In"];
            id transport = properties[@(kIOHIDTransportKey)];
            uint64_t registryID = 0;
            if (IORegistryEntryGetRegistryEntryID(device, &registryID) != KERN_SUCCESS)
                failed = YES;
            BOOL internal =
                [builtIn respondsToSelector:@selector(boolValue)] && [builtIn boolValue];
            internal = internal || [transport isEqual:@"SPI"] || [transport isEqual:@"i2c"];
            [result addObject:@{
                @"id" : [NSString stringWithFormat:@"%llu", registryID],
                @"product" : [product isKindOfClass:[NSString class]] ? product
                                                                      : @"Unknown keyboard",
                @"vendor" : [vendor isKindOfClass:[NSNumber class]] ? vendor : @0,
                @"product_id" : [pid isKindOfClass:[NSNumber class]] ? pid : @0,
                @"built_in" : @(internal),
                @"serial" : serial
            }];
            IOObjectRelease(device);
        }
        IOObjectRelease(iterator);
        if (failed)
            return NULL;
        NSData *data = [NSJSONSerialization dataWithJSONObject:result options:0 error:nil];
        if (!data)
            return NULL;
        return strdup(
            [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String);
    }
}
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
        watchingDevices = YES;
        appTick();
        [[[NSWorkspace sharedWorkspace] notificationCenter]
            addObserverForName:NSWorkspaceDidWakeNotification
                        object:nil
                         queue:[NSOperationQueue mainQueue]
                    usingBlock:^(NSNotification *note) {
                        appWake();
                    }];
        [NSApp run];
        watchingDevices = NO;
        [deviceUpdateTimer invalidate];
        cancelRetry();
        stopWatching();
    }
}
