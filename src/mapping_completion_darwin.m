#include "native.h"
#import <Foundation/Foundation.h>

/**
 * @brief Receives a worker result and finishes its mapping on the main thread.
 *
 * @note Waits for the signaled result when an application is active.
 */
extern void appMappingComplete(void);

// Workers signal completion without touching AppKit or passing Go pointers to C.

/**
 * @brief Dispatches the Go mapping completion callback to the main queue.
 *
 * @note The callback runs asynchronously.
 */
void notifyMappingComplete(void) {
    // Callback forwards worker completion to Go on the main queue.
    dispatch_async(dispatch_get_main_queue(), ^{
        appMappingComplete();
    });
}
