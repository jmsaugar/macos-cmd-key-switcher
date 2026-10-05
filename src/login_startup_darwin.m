#include "native.h"
#import <Cocoa/Cocoa.h>
#import <ServiceManagement/ServiceManagement.h>
#include <string.h>

void refreshLoginStartupItem(NSMenuItem *item) {
    SMAppServiceStatus state = SMAppService.mainAppService.status;
    item.title = @"Launch at login";
    item.state = NSControlStateValueOff;
    item.toolTip = nil;
    if (state == SMAppServiceStatusEnabled)
        item.state = NSControlStateValueOn;
    else if (state == SMAppServiceStatusRequiresApproval) {
        item.state = NSControlStateValueMixed;
        item.title = @"Launch at login (approval required)";
        item.toolTip = @"Allow CmdKeySwitcher in System Settings → General → Login Items.";
    }
}

// Idempotent cleanup: absence of a registration is already the desired state.
char *unregisterLoginStartup(void) {
    SMAppService *service = SMAppService.mainAppService;
    if (service.status == SMAppServiceStatusNotRegistered)
        return NULL;
    NSError *error = nil;
    if ([service unregisterAndReturnError:&error] || error.code == kSMErrorJobNotFound)
        return NULL;
    return strdup(
        (error.localizedDescription ?: @"Could not unregister login startup.").UTF8String);
}

static void requestApproval(void) {
    NSAlert *alert = [NSAlert new];
    alert.messageText = @"Launch at login needs approval";
    alert.informativeText = @"Allow CmdKeySwitcher in System Settings → General → Login Items. "
                            @"You can also disable its startup registration here.";
    [alert addButtonWithTitle:@"Open System Settings"];
    [alert addButtonWithTitle:@"Turn off launch at login"];
    [alert addButtonWithTitle:@"Cancel"];
    [NSApp activateIgnoringOtherApps:YES];
    NSModalResponse response = [alert runModal];
    if (response == NSAlertFirstButtonReturn)
        [SMAppService openSystemSettingsLoginItems];
    else if (response == NSAlertSecondButtonReturn) {
        char *message = unregisterLoginStartup();
        if (message) {
            showAppError(message);
            free(message);
        }
    }
}

void toggleLoginStartup(void) {
    SMAppService *service = SMAppService.mainAppService;
    if (service.status == SMAppServiceStatusRequiresApproval) {
        requestApproval();
        return;
    }
    if (service.status == SMAppServiceStatusEnabled) {
        char *message = unregisterLoginStartup();
        if (message) {
            showAppError(message);
            free(message);
        }
        return;
    }
    NSError *error = nil;
    BOOL registered = [service registerAndReturnError:&error];
    if (service.status == SMAppServiceStatusRequiresApproval)
        requestApproval();
    else if (!registered)
        showAppError(
            (error.localizedDescription ?: @"Could not register launch at login.").UTF8String);
}
