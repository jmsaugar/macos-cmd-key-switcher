#include "native.h"
#import <Foundation/Foundation.h>

extern void appMappingComplete(void);

/**
 * Dispatches a worker's completion signal to Go on the main queue.
 * No Go pointers cross the C boundary; the result stays in Go's result channel.
 */
void notifyMappingComplete(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        appMappingComplete();
    });
}
