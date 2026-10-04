#include "native.h"
#import <Cocoa/Cocoa.h>
extern void appRetry(void);
static NSTimer *retryTimer;

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
