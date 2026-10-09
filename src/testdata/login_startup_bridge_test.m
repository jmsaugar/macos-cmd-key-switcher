#include "native.h"
#import <ServiceManagement/ServiceManagement.h>
#import <objc/runtime.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Only the fake service receives unregistration requests in this executable.
@interface TestLoginService : NSObject
@property SMAppServiceStatus initialStatus;
@property SMAppServiceStatus finalStatus;
@property NSUInteger statusReads;
@property NSUInteger unregisterCalls;
@property BOOL succeeds;
@property(strong) NSError *failure;
@end

@implementation TestLoginService

/** Simulates an OS status before and after an unregistration attempt. */
- (SMAppServiceStatus)status {
    self.statusReads++;
    return self.statusReads == 1 ? self.initialStatus : self.finalStatus;
}

/** Returns the configured outcome without changing system login registration. */
- (BOOL)unregisterAndReturnError:(NSError **)error {
    self.unregisterCalls++;
    if (error)
        *error = self.failure;
    return self.succeeds;
}
@end

static TestLoginService *testService;

/** Substitutes the fake service for ServiceManagement's main-app getter. */
static id mainAppTestService(id receiver, SEL selector) { return testService; }

/** Reports the failing scenario and terminates the isolated test executable. */
static void require(BOOL condition, const char *scenario, const char *message) {
    if (!condition) {
        fprintf(stderr, "%s: %s\n", scenario, message);
        exit(1);
    }
}

/** Satisfies the bridge's UI dependency and rejects unexpected dialog requests. */
void showAppError(const char *message) {
    fprintf(stderr, "Unexpected error dialog: %s\n", message);
    exit(1);
}

typedef struct {
    const char *name;
    SMAppServiceStatus initialStatus;
    SMAppServiceStatus finalStatus;
    BOOL succeeds;
    NSInteger errorCode;
    BOOL expectsError;
    NSUInteger expectedCalls;
} UnregistrationCase;

/** Checks ambiguous statuses, absent registrations, and failures using a fake OS service. */
int main(void) {
    @autoreleasepool {
        Method getter = class_getClassMethod([SMAppService class], @selector(mainAppService));
        require(getter != NULL, "setup", "Missing main-app service getter");
        class_replaceMethod(object_getClass([SMAppService class]), @selector(mainAppService),
                            (IMP)mainAppTestService, method_getTypeEncoding(getter));

        const UnregistrationCase cases[] = {
            {"already unregistered", SMAppServiceStatusNotRegistered,
             SMAppServiceStatusNotRegistered, NO, 0, NO, 0},
            {"never registered with permission error if called", SMAppServiceStatusNotFound,
             SMAppServiceStatusNotFound, NO, 1, NO, 0},
            {"enabled with successful unregistration", SMAppServiceStatusEnabled,
             SMAppServiceStatusNotRegistered, YES, 0, NO, 1},
            {"enabled with absent job", SMAppServiceStatusEnabled, SMAppServiceStatusNotFound, NO,
             kSMErrorJobNotFound, NO, 1},
            {"failure followed by NotFound", SMAppServiceStatusEnabled, SMAppServiceStatusNotFound,
             NO, 12345, YES, 1},
            {"enabled with permission error followed by NotFound", SMAppServiceStatusEnabled,
             SMAppServiceStatusNotFound, NO, 1, YES, 1},
            {"approval pending with failure", SMAppServiceStatusRequiresApproval,
             SMAppServiceStatusNotFound, NO, 12345, YES, 1},
            {"approval pending with permission error followed by NotFound",
             SMAppServiceStatusRequiresApproval, SMAppServiceStatusNotFound, NO, 1, YES, 1},
            {"failure followed by NotRegistered", SMAppServiceStatusEnabled,
             SMAppServiceStatusNotRegistered, NO, 12345, YES, 1},
            {"failure without NSError", SMAppServiceStatusEnabled, SMAppServiceStatusNotFound, NO,
             0, YES, 1},
        };

        for (NSUInteger i = 0; i < sizeof(cases) / sizeof(cases[0]); i++) {
            UnregistrationCase scenario = cases[i];
            testService = [TestLoginService new];
            testService.initialStatus = scenario.initialStatus;
            testService.finalStatus = scenario.finalStatus;
            testService.succeeds = scenario.succeeds;
            if (scenario.errorCode) {
                NSString *description = scenario.errorCode == 1 ? @"Operation not permitted"
                                                                : @"Blocked by test service";
                testService.failure =
                    [NSError errorWithDomain:@"SMAppServiceErrorDomain"
                                        code:scenario.errorCode
                                    userInfo:@{NSLocalizedDescriptionKey : description}];
            }

            char *message = unregisterLoginStartup();
            require(testService.unregisterCalls == scenario.expectedCalls, scenario.name,
                    "Wrong number of unregistration attempts");
            require((message != NULL) == scenario.expectsError, scenario.name,
                    "Wrong success or failure result");
            if (scenario.expectsError) {
                const char *expected = scenario.errorCode == 1 ? "Operation not permitted"
                                       : scenario.errorCode    ? "Blocked by test service"
                                                            : "Could not unregister login startup.";
                require(strstr(message, expected) != NULL, scenario.name,
                        "Lost the original error or fallback description");
                if (scenario.errorCode) {
                    NSString *diagnostic =
                        [NSString stringWithFormat:@"SMAppServiceErrorDomain (%ld)",
                                                   (long)scenario.errorCode];
                    require(strstr(message, diagnostic.UTF8String) != NULL, scenario.name,
                            "Lost error domain or code diagnostics");
                }
            }
            free(message);
        }
    }
    return 0;
}
