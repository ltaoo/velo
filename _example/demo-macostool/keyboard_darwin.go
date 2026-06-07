//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices -framework Foundation

#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>
#include <unistd.h>

#ifndef kCGEventSystemDefined
#define kCGEventSystemDefined 14
#endif

static volatile int g_keyboard_disabled = 0;
static volatile int g_event_tap_initialized = 0;
static volatile int g_request_init = 0;
static volatile int g_init_result = 0;
static CFMachPortRef g_event_tap = NULL;

int keyboard_get_disabled() {
    return g_keyboard_disabled;
}

void keyboard_set_disabled(int value) {
    g_keyboard_disabled = value;
}

int keyboard_is_event_tap_initialized() {
    return g_event_tap_initialized;
}

int keyboard_is_accessibility_trusted() {
    return AXIsProcessTrusted() ? 1 : 0;
}

void keyboard_request_event_tap_init() {
    g_request_init = 1;
}

int keyboard_get_init_result() {
    return g_init_result;
}

void keyboard_reset_init_state() {
    if (g_init_result == -1) {
        g_init_result = 0;
    }
}

void keyboard_prompt_accessibility_permission() {
    NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
    AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}

CGEventRef keyboard_event_callback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon) {
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (g_event_tap) {
            CGEventTapEnable(g_event_tap, true);
        }
        return event;
    }

    if (g_keyboard_disabled &&
        (type == kCGEventKeyDown ||
         type == kCGEventKeyUp ||
         type == kCGEventFlagsChanged ||
         type == kCGEventSystemDefined)) {
        return NULL;
    }

    return event;
}

int keyboard_try_init_event_tap() {
    if (g_event_tap_initialized) {
        return 1;
    }

    CGEventMask event_mask =
        CGEventMaskBit(kCGEventKeyDown) |
        CGEventMaskBit(kCGEventKeyUp) |
        CGEventMaskBit(kCGEventFlagsChanged) |
        CGEventMaskBit(kCGEventSystemDefined);

    g_event_tap = CGEventTapCreate(
        kCGSessionEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionDefault,
        event_mask,
        keyboard_event_callback,
        NULL
    );

    if (!g_event_tap) {
        keyboard_prompt_accessibility_permission();
        g_init_result = -1;
        return 0;
    }

    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, g_event_tap, 0);
    CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopCommonModes);
    CFRelease(source);
    CGEventTapEnable(g_event_tap, true);

    g_event_tap_initialized = 1;
    g_init_result = 1;
    return 1;
}

void keyboard_run_main_loop() {
    while (1) {
        if (g_request_init && !g_event_tap_initialized && g_init_result == 0) {
            if (keyboard_try_init_event_tap()) {
                CFRunLoopRun();
            }
            g_request_init = 0;
        }
        usleep(50000);
    }
}
*/
import "C"

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

type keyboardState struct {
	Supported         bool   `json:"supported"`
	OS                string `json:"os"`
	Disabled          bool   `json:"disabled"`
	PermissionGranted bool   `json:"permission_granted"`
	EventTapReady     bool   `json:"event_tap_ready"`
}

var keyboardLoopOnce sync.Once

func startKeyboardLoop() {
	keyboardLoopOnce.Do(func() {
		go func() {
			runtime.LockOSThread()
			C.keyboard_run_main_loop()
		}()
	})
}

func readKeyboardState() keyboardState {
	return keyboardState{
		Supported:         true,
		OS:                runtime.GOOS,
		Disabled:          C.keyboard_get_disabled() == 1,
		PermissionGranted: C.keyboard_is_accessibility_trusted() == 1,
		EventTapReady:     C.keyboard_is_event_tap_initialized() == 1,
	}
}

func disableKeyboard() (keyboardState, error) {
	startKeyboardLoop()

	if C.keyboard_is_event_tap_initialized() == 0 {
		C.keyboard_reset_init_state()
		C.keyboard_request_event_tap_init()

		for i := 0; i < 50; i++ {
			if C.keyboard_get_init_result() != 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if C.keyboard_get_init_result() == -1 {
		return readKeyboardState(), errors.New("需要在系统设置 -> 隐私与安全性 -> 辅助功能中允许本应用后再禁用键盘")
	}
	if C.keyboard_is_event_tap_initialized() == 0 {
		return readKeyboardState(), errors.New("键盘事件拦截初始化超时")
	}

	C.keyboard_set_disabled(1)
	return readKeyboardState(), nil
}

func enableKeyboard() (keyboardState, error) {
	C.keyboard_set_disabled(0)
	return readKeyboardState(), nil
}
