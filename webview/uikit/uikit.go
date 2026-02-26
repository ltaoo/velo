//go:build ios

package uikit

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Foundation -framework UIKit

#import <Foundation/Foundation.h>
#import <UIKit/UIKit.h>
#import <objc/message.h>

void* msgSend_Rect(void* id, void* sel, double x, double y, double w, double h) {
    CGRect r = CGRectMake(x, y, w, h);
    typedef void* (*send_type)(void*, void*, CGRect);
    send_type func = (send_type)objc_msgSend;
    return func(id, sel, r);
}

void* msgSend_Rect_ID(void* id, void* sel, double x, double y, double w, double h, void* obj) {
    CGRect r = CGRectMake(x, y, w, h);
    typedef void* (*send_type)(void*, void*, CGRect, void*);
    send_type func = (send_type)objc_msgSend;
    return func(id, sel, r, obj);
}

CGRect msgSend_GetRect(void* id, void* sel) {
    typedef CGRect (*send_type)(void*, void*);
    send_type func = (send_type)objc_msgSend;
    return func(id, sel);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ID represents an Objective-C object pointer
type ID uintptr

// Class represents an Objective-C class pointer
type Class uintptr

// Selector represents an Objective-C selector
type Selector uintptr

// CGRect and geometry types
type CGPoint struct {
	X float64
	Y float64
}

type CGSize struct {
	Width  float64
	Height float64
}

type CGRect struct {
	Origin CGPoint
	Size   CGSize
}

// Frameworks
var (
	objc      uintptr
	uikit     uintptr
	webkit    uintptr
	libSystem uintptr
)

func init() {
	var err error
	// Try loading libobjc
	objc, err = purego.Dlopen("libobjc.dylib", purego.RTLD_GLOBAL)
	if err != nil {
		objc, err = purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_GLOBAL)
	}
	if err != nil {
		panic(fmt.Errorf("failed to load libobjc: %w", err))
	}
	
	// UIKit
	uikit, err = purego.Dlopen("/System/Library/Frameworks/UIKit.framework/UIKit", purego.RTLD_GLOBAL)
	if err != nil {
		uikit, err = purego.Dlopen("UIKit.framework/UIKit", purego.RTLD_GLOBAL)
	}
	if err != nil {
		panic(fmt.Errorf("failed to load UIKit: %w", err))
	}

	// WebKit
	webkit, err = purego.Dlopen("/System/Library/Frameworks/WebKit.framework/WebKit", purego.RTLD_GLOBAL)
	if err != nil {
		webkit, err = purego.Dlopen("WebKit.framework/WebKit", purego.RTLD_GLOBAL)
	}
	if err != nil {
		panic(fmt.Errorf("failed to load WebKit: %w", err))
	}

	// libSystem
	libSystem, err = purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_GLOBAL)
	if err != nil {
		libSystem, err = purego.Dlopen("libSystem.B.dylib", purego.RTLD_GLOBAL)
	}
	if err != nil {
		panic(fmt.Errorf("failed to load libSystem: %w", err))
	}

	initObjcRuntime()
	initDispatch()
}

// ObjC Runtime wrappers
var (
	objc_getClass          func(name *byte) uintptr
	sel_registerName       func(name *byte) uintptr
	class_addMethod        func(cls uintptr, name uintptr, imp uintptr, types *byte) bool
	objc_allocateClassPair func(superclass uintptr, name *byte, extraBytes int) uintptr
	objc_registerClassPair func(cls uintptr)
	object_getClass        func(obj uintptr) uintptr

	// msgSend variants for specific ABIs
	objc_msgSend_ptr uintptr // Generic for pointer/int args using SyscallN

	// UIApplicationMain
	uiApplicationMain func(argc int32, argv **byte, principalClassName uintptr, delegateClassName uintptr) int32
)

func initObjcRuntime() {
	purego.RegisterLibFunc(&objc_getClass, objc, "objc_getClass")
	purego.RegisterLibFunc(&sel_registerName, objc, "sel_registerName")
	purego.RegisterLibFunc(&class_addMethod, objc, "class_addMethod")
	purego.RegisterLibFunc(&objc_allocateClassPair, objc, "objc_allocateClassPair")
	purego.RegisterLibFunc(&objc_registerClassPair, objc, "objc_registerClassPair")
	purego.RegisterLibFunc(&object_getClass, objc, "object_getClass")

	// Get msgSend address for generic SyscallN
	var err error
	objc_msgSend_ptr, err = purego.Dlsym(objc, "objc_msgSend")
	if err != nil {
		panic(fmt.Errorf("failed to find objc_msgSend: %w", err))
	}

	// UIApplicationMain
	purego.RegisterLibFunc(&uiApplicationMain, uikit, "UIApplicationMain")
}

func SharedApplication() ID {
	cls := GetClass("UIApplication")
	return cls.Send(RegisterName("sharedApplication"))
}

// Dispatch handling
var (
	dispatch_async_f func(queue uintptr, context uintptr, work uintptr)
	dispatch_main_q  uintptr
)

func initDispatch() {
	purego.RegisterLibFunc(&dispatch_async_f, libSystem, "dispatch_async_f")

	sym, err := purego.Dlsym(libSystem, "_dispatch_main_q")
	if err != nil {
		fmt.Println("failed to find _dispatch_main_q:", err)
		return
	}
	dispatch_main_q = sym
}

// Helper functions
func DispatchAsyncMain(work func()) {
	cb := purego.NewCallback(func(ctx uintptr) uintptr {
		work()
		return 0
	})
	dispatch_async_f(dispatch_main_q, 0, cb)
}

func GetClass(name string) Class {
	b := append([]byte(name), 0)
	c := Class(objc_getClass(&b[0]))
	if c == 0 {
		fmt.Printf("GetClass failed for: %s\n", name)
	}
	return c
}

func RegisterName(name string) Selector {
	b := append([]byte(name), 0)
	s := Selector(sel_registerName(&b[0]))
	if s == 0 {
		fmt.Printf("RegisterName failed for: %s\n", name)
	}
	return s
}

func (id ID) Send(sel Selector, args ...interface{}) ID {
	callArgs := make([]uintptr, len(args)+2)
	callArgs[0] = uintptr(id)
	callArgs[1] = uintptr(sel)
	for i, arg := range args {
		switch v := arg.(type) {
		case uintptr:
			callArgs[i+2] = v
		case int:
			callArgs[i+2] = uintptr(v)
		case uint:
			callArgs[i+2] = uintptr(v)
		case bool:
			if v {
				callArgs[i+2] = 1
			} else {
				callArgs[i+2] = 0
			}
		case ID:
			callArgs[i+2] = uintptr(v)
		case Class:
			callArgs[i+2] = uintptr(v)
		case Selector:
			callArgs[i+2] = uintptr(v)
		case unsafe.Pointer:
			callArgs[i+2] = uintptr(v)
		default:
			// Fallback or panic?
			callArgs[i+2] = 0
		}
	}

	ret, _, _ := purego.SyscallN(objc_msgSend_ptr, callArgs...)
	return ID(ret)
}

func (c Class) Send(sel Selector, args ...interface{}) ID {
	return ID(c).Send(sel, args...)
}

// SendRect sends a message with CGRect argument
func (id ID) SendRect(sel Selector, r CGRect) ID {
	ret := C.msgSend_Rect(unsafe.Pointer(id), unsafe.Pointer(sel), 
		C.double(r.Origin.X), C.double(r.Origin.Y), 
		C.double(r.Size.Width), C.double(r.Size.Height))
	return ID(uintptr(ret))
}

// SendRectAndID sends a message with CGRect and ID arguments
func (id ID) SendRectAndID(sel Selector, r CGRect, obj ID) ID {
	ret := C.msgSend_Rect_ID(unsafe.Pointer(id), unsafe.Pointer(sel), 
		C.double(r.Origin.X), C.double(r.Origin.Y), 
		C.double(r.Size.Width), C.double(r.Size.Height),
		unsafe.Pointer(obj))
	return ID(uintptr(ret))
}

// SendGetRect sends a message that returns a CGRect
func (id ID) SendGetRect(sel Selector) CGRect {
	ret := C.msgSend_GetRect(unsafe.Pointer(id), unsafe.Pointer(sel))
	return CGRect{
		Origin: CGPoint{X: float64(ret.origin.x), Y: float64(ret.origin.y)},
		Size:   CGSize{Width: float64(ret.size.width), Height: float64(ret.size.height)},
	}
}

// Helper to create NSString
func NSString(str string) uintptr {
	cls := GetClass("NSString")
	sel := RegisterName("stringWithUTF8String:")
	b := append([]byte(str), 0)
	ptr := unsafe.Pointer(&b[0])
	ret, _, _ := purego.SyscallN(objc_msgSend_ptr, uintptr(cls), uintptr(sel), uintptr(ptr))
	runtime.KeepAlive(b)
	return ret
}

func NSStringToString(nsStr ID) string {
	if nsStr == 0 {
		return ""
	}
	sel := RegisterName("UTF8String")
	ret, _, _ := purego.SyscallN(objc_msgSend_ptr, uintptr(nsStr), uintptr(sel))
	if ret == 0 {
		return ""
	}

	ptr := unsafe.Pointer(ret)
	var bytes []byte
	for {
		b := *(*byte)(ptr)
		if b == 0 {
			break
		}
		bytes = append(bytes, b)
		ptr = unsafe.Pointer(uintptr(ptr) + 1)
	}
	return string(bytes)
}

func AllocateClassPair(superclass Class, name string, extraBytes int) Class {
	b := append([]byte(name), 0)
	return Class(objc_allocateClassPair(uintptr(superclass), &b[0], extraBytes))
}

func RegisterClassPair(cls Class) {
	objc_registerClassPair(uintptr(cls))
}

func AddMethod(cls Class, sel Selector, imp interface{}, types string) bool {
	cb := purego.NewCallback(imp)
	t := append([]byte(types), 0)
	return class_addMethod(uintptr(cls), uintptr(sel), cb, &t[0])
}

func UIApplicationMain(argc int32, argv **byte, principalClassName string, delegateClassName string) int32 {
	var principalPtr, delegatePtr uintptr
	if principalClassName != "" {
		principalPtr = NSString(principalClassName)
	}
	if delegateClassName != "" {
		delegatePtr = NSString(delegateClassName)
	}
	return uiApplicationMain(argc, argv, principalPtr, delegatePtr)
}
