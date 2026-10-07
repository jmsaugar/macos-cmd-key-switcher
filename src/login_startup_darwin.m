#include "native.h"
#import <Cocoa/Cocoa.h>
#import <ServiceManagement/ServiceManagement.h>
#include <string.h>

/**
 * @brief Updates the launch-at-login menu item from ServiceManagement
 * status.
 *
 * @param item The menu item whose title, state, and tooltip are updated.
 */
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

/**
 * @brief Removes launch-at-login registration, treating an absent
 * registration as success.
 *
 * @return NULL on success, or an allocated UTF-8 error message that the caller must free.
 */
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

/**
 * @brief Prompts for login-item approval and handles the selected settings or
 * disable action.
 *
 * @note Returns after the modal prompt and selected action finish.
 */
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

/**
 * @brief Toggles launch-at-login registration or prompts when approval is
 * required.
 *
 * @note Registration errors are shown in an alert.
 */
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
