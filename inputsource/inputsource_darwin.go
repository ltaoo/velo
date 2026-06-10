//go:build darwin

package inputsource

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Carbon -framework AppKit

#import <AppKit/AppKit.h>
#import <Carbon/Carbon.h>
#import <dispatch/dispatch.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
	char* id;
	char* name;
	char* language;
	int enabled;
	int selectable;
} VeloInputSource;

typedef struct {
	char* id;
	char* name;
	int pid;
} VeloApp;

static char* copy_cfstring(CFStringRef value) {
	if (value == NULL) return NULL;
	CFIndex length = CFStringGetLength(value);
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
	char* buffer = (char*)malloc((size_t)maxSize);
	if (buffer == NULL) return NULL;
	if (!CFStringGetCString(value, buffer, maxSize, kCFStringEncodingUTF8)) {
		free(buffer);
		return NULL;
	}
	return buffer;
}

static int bool_property(TISInputSourceRef source, CFStringRef key) {
	CFBooleanRef value = (CFBooleanRef)TISGetInputSourceProperty(source, key);
	if (value == NULL) return 0;
	return CFBooleanGetValue(value) ? 1 : 0;
}

static char* first_language(TISInputSourceRef source) {
	CFArrayRef languages = (CFArrayRef)TISGetInputSourceProperty(source, kTISPropertyInputSourceLanguages);
	if (languages == NULL || CFArrayGetCount(languages) == 0) return NULL;
	CFStringRef language = (CFStringRef)CFArrayGetValueAtIndex(languages, 0);
	return copy_cfstring(language);
}

static int is_cjkv_language_code(const char* code) {
	if (code == NULL) return 0;
	return strncmp(code, "zh", 2) == 0 ||
		strncmp(code, "ja", 2) == 0 ||
		strncmp(code, "ko", 2) == 0 ||
		strncmp(code, "vi", 2) == 0;
}

static int is_cjkv_source(TISInputSourceRef source) {
	CFArrayRef languages = (CFArrayRef)TISGetInputSourceProperty(source, kTISPropertyInputSourceLanguages);
	if (languages == NULL) return 0;
	CFIndex count = CFArrayGetCount(languages);
	for (CFIndex i = 0; i < count; i++) {
		char* language = copy_cfstring((CFStringRef)CFArrayGetValueAtIndex(languages, i));
		int matched = is_cjkv_language_code(language);
		if (language != NULL) free(language);
		if (matched) return 1;
	}
	return 0;
}

static int is_keyboard_source(TISInputSourceRef source) {
	CFStringRef category = (CFStringRef)TISGetInputSourceProperty(source, kTISPropertyInputSourceCategory);
	if (category == NULL) return 0;
	return CFStringCompare(category, kTISCategoryKeyboardInputSource, 0) == kCFCompareEqualTo;
}

static VeloInputSource make_source(TISInputSourceRef source) {
	VeloInputSource out;
	memset(&out, 0, sizeof(out));
	out.id = copy_cfstring((CFStringRef)TISGetInputSourceProperty(source, kTISPropertyInputSourceID));
	out.name = copy_cfstring((CFStringRef)TISGetInputSourceProperty(source, kTISPropertyLocalizedName));
	out.language = first_language(source);
	out.enabled = bool_property(source, kTISPropertyInputSourceIsEnabled);
	out.selectable = bool_property(source, kTISPropertyInputSourceIsSelectCapable);
	return out;
}

static void free_source(VeloInputSource source) {
	if (source.id != NULL) free(source.id);
	if (source.name != NULL) free(source.name);
	if (source.language != NULL) free(source.language);
}

static void velo_free_source(VeloInputSource source) {
	free_source(source);
}

static int velo_list_sources_impl(VeloInputSource** out) {
	*out = NULL;
	@autoreleasepool {
		CFArrayRef sources = TISCreateInputSourceList(NULL, false);
		if (sources == NULL) return -1;

		CFIndex total = CFArrayGetCount(sources);
		int count = 0;
		for (CFIndex i = 0; i < total; i++) {
			TISInputSourceRef source = (TISInputSourceRef)CFArrayGetValueAtIndex(sources, i);
			if (!is_keyboard_source(source)) continue;
			if (!bool_property(source, kTISPropertyInputSourceIsEnabled)) continue;
			if (!bool_property(source, kTISPropertyInputSourceIsSelectCapable)) continue;
			count++;
		}

		if (count == 0) {
			CFRelease(sources);
			return 0;
		}

		VeloInputSource* result = (VeloInputSource*)calloc((size_t)count, sizeof(VeloInputSource));
		if (result == NULL) {
			CFRelease(sources);
			return -1;
		}

		int index = 0;
		for (CFIndex i = 0; i < total; i++) {
			TISInputSourceRef source = (TISInputSourceRef)CFArrayGetValueAtIndex(sources, i);
			if (!is_keyboard_source(source)) continue;
			if (!bool_property(source, kTISPropertyInputSourceIsEnabled)) continue;
			if (!bool_property(source, kTISPropertyInputSourceIsSelectCapable)) continue;
			result[index++] = make_source(source);
		}

		CFRelease(sources);
		*out = result;
		return count;
	}
}

static int velo_list_sources(VeloInputSource** out) {
	if ([NSThread isMainThread]) {
		return velo_list_sources_impl(out);
	}
	__block int count = -1;
	dispatch_sync(dispatch_get_main_queue(), ^{
		count = velo_list_sources_impl(out);
	});
	return count;
}

static void velo_free_sources(VeloInputSource* sources, int count) {
	if (sources == NULL) return;
	for (int i = 0; i < count; i++) {
		free_source(sources[i]);
	}
	free(sources);
}

static VeloInputSource velo_current_source_impl() {
	VeloInputSource out;
	memset(&out, 0, sizeof(out));
	@autoreleasepool {
		TISInputSourceRef source = TISCopyCurrentKeyboardInputSource();
		if (source == NULL) return out;
		out = make_source(source);
		CFRelease(source);
	}
	return out;
}

static VeloInputSource velo_current_source() {
	if ([NSThread isMainThread]) {
		return velo_current_source_impl();
	}
	__block VeloInputSource out;
	memset(&out, 0, sizeof(out));
	dispatch_sync(dispatch_get_main_queue(), ^{
		out = velo_current_source_impl();
	});
	return out;
}

static TISInputSourceRef copy_source_by_id(const char* sourceID) {
	if (sourceID == NULL || sourceID[0] == '\0') return NULL;
	CFStringRef cfID = CFStringCreateWithCString(NULL, sourceID, kCFStringEncodingUTF8);
	if (cfID == NULL) return NULL;
	const void* keys[] = { kTISPropertyInputSourceID };
	const void* values[] = { cfID };
	CFDictionaryRef filter = CFDictionaryCreate(
		NULL,
		keys,
		values,
		1,
		&kCFTypeDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks
	);
	CFRelease(cfID);
	if (filter == NULL) return NULL;

	CFArrayRef sources = TISCreateInputSourceList(filter, false);
	CFRelease(filter);
	if (sources == NULL) return NULL;
	if (CFArrayGetCount(sources) == 0) {
		CFRelease(sources);
		return NULL;
	}

	TISInputSourceRef source = (TISInputSourceRef)CFRetain(CFArrayGetValueAtIndex(sources, 0));
	CFRelease(sources);
	return source;
}

static int velo_select_source_impl(const char* sourceID) {
	@autoreleasepool {
		TISInputSourceRef source = copy_source_by_id(sourceID);
		if (source == NULL) return 0;
		int ok = TISSelectInputSource(source) == noErr ? 1 : 0;
		int needsNudge = ok && is_cjkv_source(source);
		CFRelease(source);
		if (needsNudge) {
			void (^nudge)(void) = ^{
				NSRunningApplication* previous = [[NSWorkspace sharedWorkspace] frontmostApplication];
				NSWindow* window = [[NSWindow alloc]
					initWithContentRect:NSMakeRect(-10000, -10000, 3, 3)
					styleMask:NSWindowStyleMaskBorderless
					backing:NSBackingStoreBuffered
					defer:NO
				];
				[window setAlphaValue:0];
				[window setIgnoresMouseEvents:YES];
				[window setLevel:NSFloatingWindowLevel];
				[window makeKeyAndOrderFront:nil];
				[window orderOut:nil];
				if (previous != nil && [previous processIdentifier] != [[NSProcessInfo processInfo] processIdentifier]) {
					[previous activateWithOptions:0];
				}
			};
			if ([NSThread isMainThread]) {
				nudge();
			} else {
				dispatch_sync(dispatch_get_main_queue(), nudge);
			}
		}
		return ok;
	}
}

static int velo_select_source(const char* sourceID) {
	if ([NSThread isMainThread]) {
		return velo_select_source_impl(sourceID);
	}
	__block int ok = 0;
	dispatch_sync(dispatch_get_main_queue(), ^{
		ok = velo_select_source_impl(sourceID);
	});
	return ok;
}

static VeloApp velo_frontmost_app_impl() {
	VeloApp out;
	memset(&out, 0, sizeof(out));
	@autoreleasepool {
		NSRunningApplication* app = [[NSWorkspace sharedWorkspace] frontmostApplication];
		if (app == nil) return out;
		NSString* bundleID = [app bundleIdentifier];
		NSString* name = [app localizedName];
		if (bundleID == nil || [bundleID length] == 0) bundleID = name;
		if (bundleID != nil) out.id = strdup([bundleID UTF8String]);
		if (name != nil) out.name = strdup([name UTF8String]);
		out.pid = [app processIdentifier];
	}
	return out;
}

static VeloApp velo_frontmost_app() {
	if ([NSThread isMainThread]) {
		return velo_frontmost_app_impl();
	}
	__block VeloApp out;
	memset(&out, 0, sizeof(out));
	dispatch_sync(dispatch_get_main_queue(), ^{
		out = velo_frontmost_app_impl();
	});
	return out;
}

static void velo_free_app(VeloApp app) {
	if (app.id != NULL) free(app.id);
	if (app.name != NULL) free(app.name);
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func list() ([]Source, error) {
	var cSources *C.VeloInputSource
	count := int(C.velo_list_sources(&cSources))
	if count < 0 {
		return nil, errors.New("inputsource: failed to list input sources")
	}
	if count == 0 {
		return nil, nil
	}
	defer C.velo_free_sources(cSources, C.int(count))

	raw := unsafe.Slice(cSources, count)
	sources := make([]Source, 0, count)
	for _, source := range raw {
		sources = append(sources, sourceFromC(source))
	}
	return sources, nil
}

func current() (Source, error) {
	cSource := C.velo_current_source()
	defer C.velo_free_source(cSource)
	source := sourceFromC(cSource)
	if source.ID == "" {
		return Source{}, errors.New("inputsource: failed to read current input source")
	}
	return source, nil
}

func selectSource(sourceID string) error {
	if sourceID == "" {
		return errors.New("inputsource: source ID is empty")
	}
	cID := C.CString(sourceID)
	defer C.free(unsafe.Pointer(cID))
	if C.velo_select_source(cID) == 0 {
		return errors.New("inputsource: failed to select input source " + sourceID)
	}
	return nil
}

func frontmostApp() (App, error) {
	cApp := C.velo_frontmost_app()
	defer C.velo_free_app(cApp)
	app := App{
		ID:   cString(cApp.id),
		Name: cString(cApp.name),
		PID:  int(cApp.pid),
	}
	if app.ID == "" && app.PID == 0 {
		return App{}, errors.New("inputsource: failed to read frontmost app")
	}
	return app, nil
}

func sourceFromC(source C.VeloInputSource) Source {
	id := cString(source.id)
	name := cString(source.name)
	if name == "" {
		name = id
	}
	return Source{
		ID:         id,
		Name:       name,
		Language:   cString(source.language),
		Enabled:    source.enabled != 0,
		Selectable: source.selectable != 0,
	}
}

func cString(value *C.char) string {
	if value == nil {
		return ""
	}
	return C.GoString(value)
}
