#include "native.h"
#import <Foundation/Foundation.h>
extern void appMappingComplete(void);

// Workers signal completion without touching AppKit or passing Go pointers to C.
void notifyMappingComplete(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        appMappingComplete();
    });
}
