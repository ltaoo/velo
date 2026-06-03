//go:build darwin
// +build darwin

package notification

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#import <objc/runtime.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>
#include <string.h>

extern void veloRemotePushTokenCallback(char* token);
extern void veloRemotePushErrorCallback(char* error);
extern void veloRemotePushPayloadCallback(char* payload);

@interface VeloRemotePushDelegate : NSObject <NSApplicationDelegate>
@end
@implementation VeloRemotePushDelegate
@end

static char* velo_push_copy_error(NSError* error) {
    if (error == nil || error.localizedDescription == nil) {
        return strdup("remote push registration failed");
    }
    return strdup([error.localizedDescription UTF8String]);
}

static NSString* velo_push_token_hex(NSData* deviceToken) {
    const unsigned char* bytes = (const unsigned char*)deviceToken.bytes;
    NSMutableString* token = [NSMutableString stringWithCapacity:deviceToken.length * 2];
    for (NSUInteger i = 0; i < deviceToken.length; i++) {
        [token appendFormat:@"%02x", bytes[i]];
    }
    return token;
}

static NSString* velo_push_payload_json(NSDictionary* userInfo) {
    if (userInfo == nil) {
        return @"{}";
    }
    if (![NSJSONSerialization isValidJSONObject:userInfo]) {
        return @"{}";
    }
    NSError* error = nil;
    NSData* data = [NSJSONSerialization dataWithJSONObject:userInfo options:0 error:&error];
    if (error != nil || data == nil) {
        return @"{}";
    }
    return [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding] ?: @"{}";
}

static void velo_push_did_register(id self, SEL _cmd, NSApplication* application, NSData* deviceToken) {
    NSString* token = velo_push_token_hex(deviceToken);
    veloRemotePushTokenCallback(strdup([token UTF8String]));
}

static void velo_push_did_fail(id self, SEL _cmd, NSApplication* application, NSError* error) {
    veloRemotePushErrorCallback(velo_push_copy_error(error));
}

static void velo_push_did_receive(id self, SEL _cmd, NSApplication* application, NSDictionary* userInfo) {
    NSString* payload = velo_push_payload_json(userInfo);
    veloRemotePushPayloadCallback(strdup([payload UTF8String]));
}

static void velo_push_install_delegate_methods(id delegate) {
    Class cls = object_getClass(delegate);
    class_addMethod(cls,
                   @selector(application:didRegisterForRemoteNotificationsWithDeviceToken:),
                   (IMP)velo_push_did_register,
                   "v@:@@");
    class_addMethod(cls,
                   @selector(application:didFailToRegisterForRemoteNotificationsWithError:),
                   (IMP)velo_push_did_fail,
                   "v@:@@");
    class_addMethod(cls,
                   @selector(application:didReceiveRemoteNotification:),
                   (IMP)velo_push_did_receive,
                   "v@:@@");
}

static char* velo_register_remote_push(void) {
    @autoreleasepool {
        void (^registerBlock)(void) = ^{
            [NSApplication sharedApplication];
            id delegate = [NSApp delegate];
            if (delegate == nil) {
                delegate = [[VeloRemotePushDelegate alloc] init];
                [NSApp setDelegate:delegate];
            }
            velo_push_install_delegate_methods(delegate);
            [NSApp registerForRemoteNotifications];
        };
        if ([NSThread isMainThread]) {
            registerBlock();
        } else {
            dispatch_sync(dispatch_get_main_queue(), registerBlock);
        }
        return NULL;
    }
}
*/
import "C"
import (
	"errors"
	"sync"
	"unsafe"
)

var (
	remotePushMu        sync.RWMutex
	remotePushCallbacks RemotePushCallbacks
)

func registerRemotePushNative(callbacks RemotePushCallbacks) error {
	remotePushMu.Lock()
	remotePushCallbacks = callbacks
	remotePushMu.Unlock()

	cErr := C.velo_register_remote_push()
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}
