#include "native.h"
#import <Cocoa/Cocoa.h>

extern void appRetry(void);
static NSTimer *retryTimer;

/** Invalidates and releases any pending mapping retry timer. */
void cancelRetry(void) {
    [retryTimer invalidate];
    retryTimer = nil;
}

/** Replaces any pending retry with a one-shot Go callback on the main run loop. */
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
