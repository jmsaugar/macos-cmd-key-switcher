#include "native.h"
#import <Cocoa/Cocoa.h>

/**
 * @brief Forwards a native retry timer callback to the active application on the main
 * thread.
 */
extern void appRetry(void);
static NSTimer *retryTimer;

/**
 * @brief Invalidates and releases the retry timer.
 */
void cancelRetry(void) {
    [retryTimer invalidate];
    retryTimer = nil;
}

/**
 * @brief Replaces the retry timer with a one-shot main-run-loop callback.
 *
 * @param seconds The delay before calling appRetry.
 */
void scheduleRetry(double seconds) {
    cancelRetry();
    retryTimer = [NSTimer timerWithTimeInterval:seconds
                                        repeats:NO
                                          // Callback clears the fired retry timer and invokes the
                                          // Go retry callback. Timer is the unused firing timer.
                                          block:^(NSTimer *timer) {
                                              retryTimer = nil;
                                              appRetry();
                                          }];
    [[NSRunLoop mainRunLoop] addTimer:retryTimer forMode:NSRunLoopCommonModes];
}
