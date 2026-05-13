//go:build darwin
// +build darwin

package file

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework UniformTypeIdentifiers
#import <Cocoa/Cocoa.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>
#include <string.h>

@interface FilePanelHelper : NSObject
@property (nonatomic, assign) char* result;
@property (nonatomic, assign) int cancelled;
@property (nonatomic, assign) int animationType;
@property (nonatomic, strong) NSArray<NSString*>* allowedTypes;
@property (nonatomic, strong) dispatch_semaphore_t sem;
- (void)showPanel;
@end

@implementation FilePanelHelper
- (void)showPanel {
    NSOpenPanel* panel = [NSOpenPanel openPanel];
    [panel setCanChooseFiles:YES];
    [panel setCanChooseDirectories:NO];
    [panel setAllowsMultipleSelection:NO];
    [panel setResolvesAliases:YES];

    if (self.allowedTypes != nil && self.allowedTypes.count > 0) {
        if (@available(macOS 11.0, *)) {
            NSMutableArray<UTType*>* utTypes = [NSMutableArray array];
            for (NSString* ext in self.allowedTypes) {
                UTType* t = [UTType typeWithFilenameExtension:ext];
                if (t != nil) {
                    [utTypes addObject:t];
                }
            }
            if (utTypes.count > 0) {
                [panel setAllowedContentTypes:utTypes];
            }
        } else {
            [panel setAllowedFileTypes:self.allowedTypes];
        }
    }

    NSWindowAnimationBehavior behavior = NSWindowAnimationBehaviorDefault;
    switch (self.animationType) {
        case 1: behavior = NSWindowAnimationBehaviorNone; break;
        case 2: behavior = NSWindowAnimationBehaviorDocumentWindow; break;
        case 3: behavior = NSWindowAnimationBehaviorUtilityWindow; break;
        case 4: behavior = NSWindowAnimationBehaviorAlertPanel; break;
        default: behavior = NSWindowAnimationBehaviorDefault; break;
    }

    if (self.animationType == 5) {
        NSWindow* window = [NSApp keyWindow];
        if (window) {
            [panel beginSheetModalForWindow:window completionHandler:^(NSModalResponse response) {
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
            }];
            return;
        }
    }

    [panel setAnimationBehavior:behavior];
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
@end

static char* BoxFile_ShowOpenPanel(int* cancelled, int animationType, const char** allowedTypes, int allowedTypesCount) {
    FilePanelHelper* helper = [[FilePanelHelper alloc] init];
    helper.sem = dispatch_semaphore_create(0);
    helper.result = NULL;
    helper.cancelled = 0;
    helper.animationType = animationType;

    if (allowedTypes != NULL && allowedTypesCount > 0) {
        NSMutableArray<NSString*>* types = [NSMutableArray arrayWithCapacity:allowedTypesCount];
        for (int i = 0; i < allowedTypesCount; i++) {
            [types addObject:[NSString stringWithUTF8String:allowedTypes[i]]];
        }
        helper.allowedTypes = types;
    }

    // Always dispatch to main thread.
    // Since ShowFileSelectDialog is now called from a goroutine (via go func in HandleMessage),
    // we are NOT on the main thread, so dispatch_semaphore_wait is safe.
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

// ShowFileSelectDialog shows a file selection dialog and returns the selected file path.
// animationType: "default", "none", "document", "utility", "alert", "sheet"
func showFileSelectDialog(animationType string, allowedTypes []string) (string, error) {
	runtime.UnlockOSThread()

	var animCode int
	switch animationType {
	case "default":
		animCode = 0
	case "none":
		animCode = 1
	case "document":
		animCode = 2
	case "utility":
		animCode = 3
	case "alert":
		animCode = 4
	case "sheet":
		animCode = 5
	default:
		animCode = 0
	}

	var cancelled C.int
	var cStr *C.char

	if len(allowedTypes) > 0 {
		cTypes := make([]*C.char, len(allowedTypes))
		for i, t := range allowedTypes {
			cTypes[i] = C.CString(t)
		}
		defer func() {
			for _, ct := range cTypes {
				C.free(unsafe.Pointer(ct))
			}
		}()
		cStr = C.BoxFile_ShowOpenPanel(&cancelled, C.int(animCode), &cTypes[0], C.int(len(cTypes)))
	} else {
		cStr = C.BoxFile_ShowOpenPanel(&cancelled, C.int(animCode), nil, 0)
	}

	if cancelled != 0 {
		return "", errors.New("cancelled")
	}
	if cStr == nil {
		return "", errors.New("failed to get file path")
	}
	defer C.free(unsafe.Pointer(cStr))

	return C.GoString(cStr), nil
}
