#include "native.h"
#import <Foundation/Foundation.h>
#include <string.h>

static NSString *const configurationKey = @"configuration";

/** Reads only the app domain and serializes its configuration for the Go bridge. */
char *readPreferences(void) {
    @autoreleasepool {
        NSString *domain = NSBundle.mainBundle.bundleIdentifier;
        if (!domain.length)
            return NULL;
        id configuration =
            [NSUserDefaults.standardUserDefaults persistentDomainForName:domain][configurationKey];
        if (!configuration)
            return strdup("{}");
        if (![configuration isKindOfClass:[NSDictionary class]] ||
            ![NSJSONSerialization isValidJSONObject:configuration])
            return NULL;
        NSData *data = [NSJSONSerialization dataWithJSONObject:configuration options:0 error:nil];
        if (!data)
            return NULL;
        return strdup(
            [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String);
    }
}

/** Stores the complete configuration as one dictionary; macOS handles disk persistence. */
char *writePreferences(const char *json) {
    @autoreleasepool {
        if (!NSBundle.mainBundle.bundleIdentifier.length)
            return strdup("Preferences require running the app bundle.");
        NSString *text = [NSString stringWithUTF8String:json];
        NSData *data = [text dataUsingEncoding:NSUTF8StringEncoding];
        id configuration =
            data ? [NSJSONSerialization JSONObjectWithData:data options:0 error:nil] : nil;
        if (![configuration isKindOfClass:[NSDictionary class]] ||
            ![NSPropertyListSerialization propertyList:configuration
                                      isValidForFormat:NSPropertyListBinaryFormat_v1_0])
            return strdup("Invalid configuration dictionary.");
        [NSUserDefaults.standardUserDefaults setObject:configuration forKey:configurationKey];
        return NULL;
    }
}

/** Clears the app domain after Go has stopped all further preference updates. */
char *clearPreferences(void) {
    @autoreleasepool {
        NSString *domain = NSBundle.mainBundle.bundleIdentifier;
        if (!domain.length)
            return strdup("Preferences require running the app bundle.");
        [NSUserDefaults.standardUserDefaults removePersistentDomainForName:domain];
        return NULL;
    }
}
