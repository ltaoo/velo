//go:build darwin
// +build darwin

package notification

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UserNotifications
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>
#include <string.h>

@interface VeloNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation VeloNotificationDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter*)center
        willPresentNotification:(UNNotification*)notification
          withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler API_AVAILABLE(macos(10.14)) {
    if (@available(macOS 11.0, *)) {
        completionHandler(UNNotificationPresentationOptionBanner |
                          UNNotificationPresentationOptionList |
                          UNNotificationPresentationOptionSound);
    } else {
        completionHandler(UNNotificationPresentationOptionAlert |
                          UNNotificationPresentationOptionSound);
    }
}
@end

static VeloNotificationDelegate* velo_notification_delegate = nil;

static char* velo_copy_error(NSError* error) {
    if (error == nil || error.localizedDescription == nil) {
        return strdup("notification failed");
    }
    if (@available(macOS 10.14, *)) {
        if ([error.domain isEqualToString:UNErrorDomain] && error.code == UNErrorCodeNotificationsNotAllowed) {
            return strdup("notification permission is not allowed by macOS");
        }
    }
    return strdup([error.localizedDescription UTF8String]);
}

static char* velo_show_notification(const char* title, const char* body, int sound) {
    @autoreleasepool {
        NSString* nsTitle = title ? [NSString stringWithUTF8String:title] : @"";
        NSString* nsBody = body ? [NSString stringWithUTF8String:body] : @"";

        if (@available(macOS 10.14, *)) {
            UNUserNotificationCenter* center = [UNUserNotificationCenter currentNotificationCenter];
            if (velo_notification_delegate == nil) {
                velo_notification_delegate = [[VeloNotificationDelegate alloc] init];
                center.delegate = velo_notification_delegate;
            }

            __block char* authError = NULL;
            dispatch_semaphore_t authSem = dispatch_semaphore_create(0);
            UNAuthorizationOptions options = UNAuthorizationOptionAlert | UNAuthorizationOptionBadge;
            if (sound) {
                options |= UNAuthorizationOptionSound;
            }

            [center requestAuthorizationWithOptions:options completionHandler:^(BOOL granted, NSError* error) {
                if (error != nil) {
                    authError = velo_copy_error(error);
                } else if (!granted) {
                    authError = strdup("notification permission denied");
                }
                dispatch_semaphore_signal(authSem);
            }];
            dispatch_semaphore_wait(authSem, DISPATCH_TIME_FOREVER);
            if (authError != NULL) {
                return authError;
            }

            UNMutableNotificationContent* content = [[UNMutableNotificationContent alloc] init];
            content.title = nsTitle;
            content.body = nsBody;
            if (sound) {
                content.sound = [UNNotificationSound defaultSound];
            }

            NSString* identifier = [[NSUUID UUID] UUIDString];
            UNNotificationRequest* request = [UNNotificationRequest requestWithIdentifier:identifier content:content trigger:nil];

            __block char* addError = NULL;
            dispatch_semaphore_t addSem = dispatch_semaphore_create(0);
            [center addNotificationRequest:request withCompletionHandler:^(NSError* error) {
                if (error != nil) {
                    addError = velo_copy_error(error);
                }
                dispatch_semaphore_signal(addSem);
            }];
            dispatch_semaphore_wait(addSem, DISPATCH_TIME_FOREVER);
            return addError;
        }

        return strdup("notification requires macOS 10.14 or later");
    }
}

static char* velo_notification_status(void) {
    @autoreleasepool {
        NSString* bundleID = [[NSBundle mainBundle] bundleIdentifier] ?: @"";
        NSString* bundlePath = [[NSBundle mainBundle] bundlePath] ?: @"";

        if (@available(macOS 10.14, *)) {
            UNUserNotificationCenter* center = [UNUserNotificationCenter currentNotificationCenter];
            __block NSString* status = @"unknown";
            dispatch_semaphore_t sem = dispatch_semaphore_create(0);
            [center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings* settings) {
                switch (settings.authorizationStatus) {
                    case UNAuthorizationStatusNotDetermined:
                        status = @"not_determined";
                        break;
                    case UNAuthorizationStatusDenied:
                        status = @"denied";
                        break;
                    case UNAuthorizationStatusAuthorized:
                        status = @"authorized";
                        break;
                    case UNAuthorizationStatusProvisional:
                        status = @"provisional";
                        break;
                    default:
                        status = @"unknown";
                        break;
                }
                dispatch_semaphore_signal(sem);
            }];
            dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);

            NSString* result = [NSString stringWithFormat:@"1\n%@\n%@\n%@",
                                status, bundleID, bundlePath];
            return strdup([result UTF8String]);
        }

        NSString* result = [NSString stringWithFormat:@"0\nunsupported\n%@\n%@",
                            bundleID, bundlePath];
        return strdup([result UTF8String]);
    }
}

static char* velo_cleanup_notifications(void) {
    @autoreleasepool {
        if (@available(macOS 10.14, *)) {
            UNUserNotificationCenter* center = [UNUserNotificationCenter currentNotificationCenter];
            [center removeAllPendingNotificationRequests];
            [center removeAllDeliveredNotifications];
            return NULL;
        }
        return strdup("notification cleanup requires macOS 10.14 or later");
    }
}
*/
import "C"
import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"unsafe"
)

func showNative(opts Options) error {
	cTitle := C.CString(opts.Title)
	cBody := C.CString(opts.Body)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cBody))

	sound := 0
	if opts.Sound {
		sound = 1
	}
	cErr := C.velo_show_notification(cTitle, cBody, C.int(sound))
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		errText := C.GoString(cErr)
		if isPermissionError(errText) {
			return showWithAppleScript(opts)
		}
		return errors.New(errText)
	}
	return nil
}

func isPermissionError(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "not allowed") ||
		strings.Contains(message, "permission denied") ||
		strings.Contains(message, "unerrordomain error 1")
}

func showWithAppleScript(opts Options) error {
	body := opts.Body
	if body == "" {
		body = opts.Title
	}
	title := opts.Title
	if title == "" {
		title = opts.AppName
	}

	script := fmt.Sprintf("display notification %s with title %s subtitle %s",
		appleScriptString(body),
		appleScriptString(title),
		appleScriptString(opts.Type),
	)
	if opts.Sound {
		script += " sound name \"default\""
	}
	if out, err := exec.Command("osascript", "-e", script).CombinedOutput(); err != nil {
		return fmt.Errorf("notification: macOS notification not allowed and AppleScript fallback failed: %w: %s", err, string(out))
	}
	return nil
}

func appleScriptString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + value + "\""
}

func permissionStatusNative() Status {
	cStatus := C.velo_notification_status()
	if cStatus == nil {
		return Status{Supported: false, Status: "unknown"}
	}
	defer C.free(unsafe.Pointer(cStatus))

	parts := strings.SplitN(C.GoString(cStatus), "\n", 4)
	status := Status{Supported: false, Status: "unknown"}
	if len(parts) > 0 && parts[0] == "1" {
		status.Supported = true
	}
	if len(parts) > 1 {
		status.Status = parts[1]
	}
	if len(parts) > 2 {
		status.BundleID = parts[2]
	}
	if len(parts) > 3 {
		status.BundlePath = parts[3]
	}
	return status
}

func cleanupNative(opts CleanupOptions) error {
	cErr := C.velo_cleanup_notifications()
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}
