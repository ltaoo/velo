//go:build darwin
// +build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>

@interface ExternalLinkConfirmHelper : NSObject
@property (nonatomic, assign) const char* message;
@property (nonatomic, assign) int result;
@property (nonatomic, strong) dispatch_semaphore_t sem;
- (void)showConfirm;
@end

@implementation ExternalLinkConfirmHelper
- (void)showConfirm {
    @autoreleasepool {
        NSAlert *alert = [[NSAlert alloc] init];
        [alert setMessageText:@"打开外部链接"];
        [alert setInformativeText:[NSString stringWithUTF8String:self.message]];
        [alert setAlertStyle:NSAlertStyleWarning];
        [alert addButtonWithTitle:@"使用默认浏览器打开"];
        [alert addButtonWithTitle:@"取消"];
        [NSApp activateIgnoringOtherApps:YES];
        NSModalResponse response = [alert runModal];
        self.result = response == NSAlertFirstButtonReturn ? 1 : 0;
        dispatch_semaphore_signal(self.sem);
    }
}
@end

static int BoxConfirmExternalLinkOpen(const char* message) {
    ExternalLinkConfirmHelper* helper = [[ExternalLinkConfirmHelper alloc] init];
    helper.sem = dispatch_semaphore_create(0);
    helper.message = message;
    helper.result = 0;
    [helper performSelectorOnMainThread:@selector(showConfirm) withObject:nil waitUntilDone:NO];
    dispatch_semaphore_wait(helper.sem, DISPATCH_TIME_FOREVER);
    return helper.result;
}
*/
import "C"
import "unsafe"

func confirmExternalBrowserOpen(target string) (bool, error) {
	message := externalBrowserConfirmMessage(target)
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))
	return C.BoxConfirmExternalLinkOpen(cMessage) == 1, nil
}
