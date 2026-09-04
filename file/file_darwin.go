//go:build darwin

package file

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework UniformTypeIdentifiers
#import <Cocoa/Cocoa.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>
#include <string.h>

@interface VeloFilePanelHelper : NSObject
@property (nonatomic, assign) BOOL save;
@property (nonatomic, assign) BOOL chooseFiles;
@property (nonatomic, assign) BOOL chooseDirectories;
@property (nonatomic, assign) BOOL multiple;
@property (nonatomic, assign) int cancelled;
@property (nonatomic, assign) char* result;
@property (nonatomic, strong) NSString* title;
@property (nonatomic, strong) NSString* directoryPath;
@property (nonatomic, strong) NSString* defaultFilename;
@property (nonatomic, strong) NSArray<NSString*>* allowedTypes;
@property (nonatomic, strong) dispatch_semaphore_t semaphore;
- (void)showPanel;
@end

@implementation VeloFilePanelHelper
- (void)configurePanel:(NSSavePanel*)panel {
    if (self.title.length > 0) [panel setTitle:self.title];
    if (self.directoryPath.length > 0) {
        [panel setDirectoryURL:[NSURL fileURLWithPath:self.directoryPath isDirectory:YES]];
    }
    if (self.defaultFilename.length > 0) [panel setNameFieldStringValue:self.defaultFilename];
    [panel setCanCreateDirectories:YES];
    if (self.allowedTypes.count > 0) {
        if (@available(macOS 11.0, *)) {
            NSMutableArray<UTType*>* types = [NSMutableArray array];
            for (NSString* extension in self.allowedTypes) {
                UTType* type = [UTType typeWithFilenameExtension:extension];
                if (type != nil) [types addObject:type];
            }
            if (types.count > 0) [panel setAllowedContentTypes:types];
        } else {
            [panel setAllowedFileTypes:self.allowedTypes];
        }
    }
}

- (void)finishWithURLs:(NSArray<NSURL*>*)urls {
    NSMutableArray<NSString*>* paths = [NSMutableArray arrayWithCapacity:urls.count];
    for (NSURL* url in urls) {
        if (url.path != nil) [paths addObject:url.path];
    }
    NSData* data = [NSJSONSerialization dataWithJSONObject:paths options:0 error:nil];
    if (data != nil) self.result = strndup(data.bytes, data.length);
    dispatch_semaphore_signal(self.semaphore);
}

- (void)showPanel {
    [NSApp activateIgnoringOtherApps:YES];
    if (self.save) {
        NSSavePanel* panel = [NSSavePanel savePanel];
        [self configurePanel:panel];
        if ([panel runModal] == NSModalResponseOK && panel.URL != nil) {
            [self finishWithURLs:@[panel.URL]];
        } else {
            self.cancelled = 1;
            dispatch_semaphore_signal(self.semaphore);
        }
        return;
    }

    NSOpenPanel* panel = [NSOpenPanel openPanel];
    [self configurePanel:panel];
    [panel setCanChooseFiles:self.chooseFiles];
    [panel setCanChooseDirectories:self.chooseDirectories];
    [panel setAllowsMultipleSelection:self.multiple];
    [panel setResolvesAliases:YES];
    if ([panel runModal] == NSModalResponseOK) {
        [self finishWithURLs:panel.URLs];
    } else {
        self.cancelled = 1;
        dispatch_semaphore_signal(self.semaphore);
    }
}
@end

static char* VeloFile_ShowPanel(
    int save, int chooseFiles, int chooseDirectories, int multiple, int* cancelled,
    const char* title, const char* directoryPath, const char* defaultFilename,
    const char** allowedTypes, int allowedTypesCount
) {
    VeloFilePanelHelper* helper = [[VeloFilePanelHelper alloc] init];
    helper.save = save != 0;
    helper.chooseFiles = chooseFiles != 0;
    helper.chooseDirectories = chooseDirectories != 0;
    helper.multiple = multiple != 0;
    helper.cancelled = 0;
    helper.result = NULL;
    helper.semaphore = dispatch_semaphore_create(0);
    if (title != NULL) helper.title = [NSString stringWithUTF8String:title];
    if (directoryPath != NULL) helper.directoryPath = [NSString stringWithUTF8String:directoryPath];
    if (defaultFilename != NULL) helper.defaultFilename = [NSString stringWithUTF8String:defaultFilename];
    if (allowedTypes != NULL && allowedTypesCount > 0) {
        NSMutableArray<NSString*>* types = [NSMutableArray arrayWithCapacity:allowedTypesCount];
        for (int i = 0; i < allowedTypesCount; i++) {
            [types addObject:[NSString stringWithUTF8String:allowedTypes[i]]];
        }
        helper.allowedTypes = types;
    }
    [helper performSelectorOnMainThread:@selector(showPanel) withObject:nil waitUntilDone:NO];
    dispatch_semaphore_wait(helper.semaphore, DISPATCH_TIME_FOREVER);
    *cancelled = helper.cancelled;
    return helper.result;
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

func show_open_dialog(options OpenDialogOptions) ([]string, error) {
	return show_dialog(false, options.CanChooseFiles, options.CanChooseDirectories, options.AllowsMultipleSelection, options.Title, options.Directory, "", options.AllowedTypes)
}

func show_save_dialog(options SaveDialogOptions) (string, error) {
	paths, err := show_dialog(true, false, false, false, options.Title, options.Directory, options.DefaultFilename, options.AllowedTypes)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", ErrCancelled
	}
	return paths[0], nil
}

func show_dialog(save, choose_files, choose_directories, multiple bool, title, directory, default_filename string, allowed_types []string) ([]string, error) {
	c_title := optional_c_string(title)
	c_directory := optional_c_string(directory)
	c_filename := optional_c_string(default_filename)
	defer free_c_string(c_title)
	defer free_c_string(c_directory)
	defer free_c_string(c_filename)

	c_types := make([]*C.char, len(allowed_types))
	for index, allowed_type := range allowed_types {
		c_types[index] = C.CString(allowed_type)
	}
	defer func() {
		for _, c_type := range c_types {
			C.free(unsafe.Pointer(c_type))
		}
	}()

	var c_types_pointer **C.char
	if len(c_types) > 0 {
		c_types_pointer = &c_types[0]
	}
	var cancelled C.int
	result := C.VeloFile_ShowPanel(bool_int(save), bool_int(choose_files), bool_int(choose_directories), bool_int(multiple), &cancelled, c_title, c_directory, c_filename, c_types_pointer, C.int(len(c_types)))
	if cancelled != 0 {
		return nil, ErrCancelled
	}
	if result == nil {
		return nil, fmt.Errorf("file dialog returned no path")
	}
	defer C.free(unsafe.Pointer(result))
	var paths []string
	if err := json.Unmarshal([]byte(C.GoString(result)), &paths); err != nil {
		return nil, err
	}
	return paths, nil
}

func bool_int(value bool) C.int {
	if value {
		return 1
	}
	return 0
}

func optional_c_string(value string) *C.char {
	if value == "" {
		return nil
	}
	return C.CString(value)
}

func free_c_string(value *C.char) {
	if value != nil {
		C.free(unsafe.Pointer(value))
	}
}
