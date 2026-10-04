#include "native.h"
#import <Cocoa/Cocoa.h>
#import <IOKit/IOKitLib.h>
extern void appTick(void);
static IONotificationPortRef notificationPort;
static io_iterator_t addedDevices, removedDevices;
static NSTimer *deviceUpdateTimer;
static BOOL watchingDevices;

void beginKeyboardWatching(void) { watchingDevices = YES; }

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

void endKeyboardWatching(void) {
    watchingDevices = NO;
    [deviceUpdateTimer invalidate];
    deviceUpdateTimer = nil;
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
int ensureKeyboardWatching(void) {
    if (!watchingDevices)
        return YES;
    if (notificationPort)
        return YES;
    notificationPort = IONotificationPortCreate(kIOMainPortDefault);
    if (!notificationPort)
        return NO;
    CFRunLoopSourceRef source = IONotificationPortGetRunLoopSource(notificationPort);
    if (!source) {
        endKeyboardWatching();
        watchingDevices = YES;
        return NO;
    }
    CFRunLoopAddSource(CFRunLoopGetMain(), source, kCFRunLoopCommonModes);
    kern_return_t result = IOServiceAddMatchingNotification(
        notificationPort, kIOFirstMatchNotification, IOServiceMatching("IOHIDDevice"),
        deviceChanged, NULL, &addedDevices);
    if (result != KERN_SUCCESS) {
        endKeyboardWatching();
        watchingDevices = YES;
        return NO;
    }
    // Empty both initial iterators before relying on future notifications.
    deviceChanged(NULL, addedDevices);
    result = IOServiceAddMatchingNotification(notificationPort, kIOTerminatedNotification,
                                              IOServiceMatching("IOHIDDevice"), deviceChanged, NULL,
                                              &removedDevices);
    if (result != KERN_SUCCESS) {
        endKeyboardWatching();
        watchingDevices = YES;
        return NO;
    }
    deviceChanged(NULL, removedDevices);
    return YES;
}
