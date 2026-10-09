#include "native.h"
#import <Foundation/Foundation.h>
#import <objc/runtime.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// This test executable replaces the shared defaults getter before calling the bridge.
// No UserDefaults read, write, or deletion reaches the operating system.
@interface MemoryDefaults : NSObject
@property(strong) NSMutableDictionary *values;
@property(copy) NSString *clearedDomain;
@property NSUInteger writes;
@end

@implementation MemoryDefaults

/** Returns the simulated app domain and verifies the bridge uses bundle metadata. */
- (NSDictionary *)persistentDomainForName:(NSString *)domain {
    NSCAssert([domain isEqualToString:NSBundle.mainBundle.bundleIdentifier], @"Wrong domain");
    return self.values;
}

/** Records one preference update without communicating with the preferences daemon. */
- (void)setObject:(id)value forKey:(NSString *)key {
    self.values[key] = value;
    self.writes++;
}

/** Records domain removal and clears only the in-memory fixture. */
- (void)removePersistentDomainForName:(NSString *)domain {
    self.clearedDomain = domain;
    [self.values removeAllObjects];
}
@end

static MemoryDefaults *memoryDefaults;

/** Substitutes the test store for the shared UserDefaults getter. */
static id testDefaults(id receiver, SEL selector) { return memoryDefaults; }

/** Fails the executable with a useful message if an assertion does not hold. */
static void require(BOOL condition, const char *message) {
    if (!condition) {
        fprintf(stderr, "%s\n", message);
        exit(1);
    }
}

/** Exercises snapshots, malformed values, and complete domain clearing without persistent I/O. */
int main(void) {
    @autoreleasepool {
        NSString *domain = NSBundle.mainBundle.bundleIdentifier;
        require([domain isEqualToString:@"local.cmdkeyswitcher.preferences-test"],
                "Missing isolated bundle metadata");
        memoryDefaults = [MemoryDefaults new];
        memoryDefaults.values = [NSMutableDictionary new];
        Method getter =
            class_getClassMethod([NSUserDefaults class], @selector(standardUserDefaults));
        class_replaceMethod(object_getClass([NSUserDefaults class]),
                            @selector(standardUserDefaults), (IMP)testDefaults,
                            method_getTypeEncoding(getter));

        char *raw = readPreferences();
        require(raw && strcmp(raw, "{}") == 0,
                "Missing preferences did not return an empty object");
        free(raw);

        const char *configuration = "{\"type\":\"win\",\"keyboard_types\":{\"1:2\":\"mac\","
                                    "\"1:2:serial:unit+A\":\"win\"}}";
        char *error = writePreferences(configuration);
        require(!error, "Valid configuration was rejected");
        NSDictionary *stored = memoryDefaults.values[@"configuration"];
        require([stored isKindOfClass:[NSDictionary class]] && memoryDefaults.writes == 1,
                "Configuration was not stored as one dictionary update");
        raw = readPreferences();
        require(raw != NULL, "Could not reload stored configuration");
        NSData *data = [[NSString stringWithUTF8String:raw] dataUsingEncoding:NSUTF8StringEncoding];
        NSDictionary *reloaded = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
        require([reloaded isEqualToDictionary:stored], "Mode and overrides did not round trip");
        free(raw);

        for (NSString *invalid in @[ @"not JSON", @"[]", @"{\"type\":null}" ]) {
            error = writePreferences(invalid.UTF8String);
            require(error != NULL && memoryDefaults.writes == 1,
                    "Malformed bridge input mutated stored preferences");
            free(error);
        }
        memoryDefaults.values[@"configuration"] = @YES;
        raw = readPreferences();
        require(!raw, "Accepted a non-dictionary stored configuration");
        memoryDefaults.values[@"configuration"] = @{@"type" : [NSData data]};
        raw = readPreferences();
        require(!raw, "Accepted a configuration that cannot be represented as JSON");

        memoryDefaults.values[@"configuration"] = stored;
        memoryDefaults.values[@"window-setting"] = @"value";
        error = clearPreferences();
        require(!error && memoryDefaults.values.count == 0 &&
                    [memoryDefaults.clearedDomain isEqualToString:domain],
                "Cleanup did not remove the app's complete preferences domain");
        raw = readPreferences();
        require(raw && strcmp(raw, "{}") == 0, "Cleared preferences did not reload as empty");
        free(raw);
    }
    return 0;
}
