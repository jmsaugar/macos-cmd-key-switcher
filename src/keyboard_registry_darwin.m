#include "native.h"
#import <Cocoa/Cocoa.h>
#import <IOKit/IOKitLib.h>
#import <IOKit/hid/IOHIDKeys.h>
#import <IOKit/hid/IOHIDUsageTables.h>
#include <stdlib.h>

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
        if (!ensureKeyboardWatching())
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
