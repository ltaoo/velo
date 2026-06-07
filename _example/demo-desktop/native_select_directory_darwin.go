//go:build darwin
// +build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>

@interface DirectoryPanelHelper : NSObject
@property (nonatomic, assign) char* result;
@property (nonatomic, assign) int cancelled;
@property (nonatomic, strong) dispatch_semaphore_t sem;
- (void)showPanel;
@end

@implementation DirectoryPanelHelper
- (void)showPanel {
    @autoreleasepool {
        NSOpenPanel* panel = [NSOpenPanel openPanel];
        [panel setCanChooseFiles:NO];
        [panel setCanChooseDirectories:YES];
        [panel setAllowsMultipleSelection:NO];
        [panel setCanCreateDirectories:YES];
        [panel setResolvesAliases:YES];
        [panel setMessage:@"选择或创建一个 Velo vault 目录"];
        [panel setPrompt:@"选择"];
        [NSApp activateIgnoringOtherApps:YES];

        NSModalResponse response = [panel runModal];
        if (response == NSModalResponseOK) {
            NSURL* url = [[panel URLs] firstObject];
            if (url) {
                const char* path = [[url path] UTF8String];
                if (path) {
                    self.result = strdup(path);
                }
            }
        } else {
            self.cancelled = 1;
        }
        dispatch_semaphore_signal(self.sem);
    }
}
@end

static char* BoxSelectDirectory(int* cancelled) {
    DirectoryPanelHelper* helper = [[DirectoryPanelHelper alloc] init];
    helper.sem = dispatch_semaphore_create(0);
    helper.result = NULL;
    helper.cancelled = 0;
    [helper performSelectorOnMainThread:@selector(showPanel) withObject:nil waitUntilDone:NO];
    dispatch_semaphore_wait(helper.sem, DISPATCH_TIME_FOREVER);
    *cancelled = helper.cancelled;
    return helper.result;
}
*/
import "C"
import (
	"errors"
	"runtime"
	"unsafe"
)

func selectVaultDirectory() (string, error) {
	runtime.UnlockOSThread()

	var cancelled C.int
	cPath := C.BoxSelectDirectory(&cancelled)
	if cancelled != 0 {
		return "", errors.New("cancelled")
	}
	if cPath == nil {
		return "", errors.New("failed to get directory path")
	}
	defer C.free(unsafe.Pointer(cPath))
	return C.GoString(cPath), nil
}
