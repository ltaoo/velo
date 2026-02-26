//go:build darwin && !ios

package cocoa

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Framework handling
var (
	objc      uintptr
	appkit    uintptr
	webkit    uintptr
	libSystem uintptr
)

func init() {
	var err error
	objc, err = purego.Dlopen("libobjc.A.dylib", purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("failed to load libobjc: %w", err))
	}
	appkit, err = purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("failed to load AppKit: %w", err))
	}
	webkit, err = purego.Dlopen("/System/Library/Frameworks/WebKit.framework/WebKit", purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("failed to load WebKit: %w", err))
	}
	libSystem, err = purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_GLOBAL)
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

	// Specialized msgSend for struct args (needed because purego/SyscallN doesn't handle structs/floats correctly on arm64)
	objc_msgSend_Rect_uint_uint_bool func(id, sel uintptr, r CGRect, style, backing uintptr, defer_ bool) uintptr
	objc_msgSend_Rect_ptr            func(id, sel uintptr, r CGRect, ptr uintptr) uintptr
	objc_msgSend_Size                func(id, sel uintptr, s CGSize) uintptr
	objc_msgSend_Point               func(id, sel uintptr, p CGPoint) uintptr
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

	// Register specialized msgSend variants
	// Note: We register the SAME symbol "objc_msgSend" but with different Go signatures
	purego.RegisterLibFunc(&objc_msgSend_Rect_uint_uint_bool, objc, "objc_msgSend")
	purego.RegisterLibFunc(&objc_msgSend_Rect_ptr, objc, "objc_msgSend")
	purego.RegisterLibFunc(&objc_msgSend_Size, objc, "objc_msgSend")
	purego.RegisterLibFunc(&objc_msgSend_Point, objc, "objc_msgSend")
}

// Dispatch handling
var (
	dispatch_async_f func(queue uintptr, context uintptr, work uintptr)
	dispatch_main_q  uintptr
	executorPtr      uintptr
	tasksMutex       sync.Mutex
	tasks            = make(map[uintptr]func())
	nextTaskID       uintptr
)

func initDispatch() {
	purego.RegisterLibFunc(&dispatch_async_f, libSystem, "dispatch_async_f")

	sym, err := purego.Dlsym(libSystem, "_dispatch_main_q")
	if err != nil {
		sym, err = purego.Dlsym(libSystem, "dispatch_main_q")
	}
	if err != nil {
		panic(fmt.Errorf("failed to find _dispatch_main_q: %w", err))
	}
	dispatch_main_q = sym

	executorPtr = purego.NewCallback(executor)
}

func executor(context uintptr) {
	tasksMutex.Lock()
	task, ok := tasks[context]
	delete(tasks, context)
	tasksMutex.Unlock()

	if ok {
		task()
	}
}

func DispatchMain(f func()) {
	tasksMutex.Lock()
	id := nextTaskID
	nextTaskID++
	tasks[id] = f
	tasksMutex.Unlock()

	dispatch_async_f(dispatch_main_q, id, executorPtr)
}

// ID represents an Objective-C object pointer
type ID uintptr

// Class represents an Objective-C class pointer
type Class uintptr

// Selector represents an Objective-C selector
type Selector uintptr

func GetClass(name string) Class {
	b := append([]byte(name), 0)
	return Class(objc_getClass(&b[0]))
}

func RegisterName(name string) Selector {
	b := append([]byte(name), 0)
	return Selector(sel_registerName(&b[0]))
}

// MsgSend sends a message to an object
func (id ID) Send(sel Selector, args ...interface{}) ID {
	return send(uintptr(id), sel, args...)
}

func (cls Class) Send(sel Selector, args ...interface{}) ID {
	return send(uintptr(cls), sel, args...)
}

func send(id uintptr, sel Selector, args ...interface{}) ID {
	uArgs := make([]uintptr, len(args)+2)
	uArgs[0] = id
	uArgs[1] = uintptr(sel)
	for i, arg := range args {
		switch v := arg.(type) {
		case uintptr:
			uArgs[i+2] = v
		case int:
			uArgs[i+2] = uintptr(v)
		case uint:
			uArgs[i+2] = uintptr(v)
		case int32:
			uArgs[i+2] = uintptr(v)
		case uint32:
			uArgs[i+2] = uintptr(v)
		case int64:
			uArgs[i+2] = uintptr(v)
		case uint64:
			uArgs[i+2] = uintptr(v)
		case bool:
			if v {
				uArgs[i+2] = 1
			} else {
				uArgs[i+2] = 0
			}
		case unsafe.Pointer:
			uArgs[i+2] = uintptr(v)
		case ID:
			uArgs[i+2] = uintptr(v)
		case Class:
			uArgs[i+2] = uintptr(v)
		case Selector:
			uArgs[i+2] = uintptr(v)
		default:
			panic(fmt.Sprintf("unsupported argument type for generic Send: %T. Use specialized Send methods for structs/floats.", v))
		}
	}
	ret, _, _ := purego.SyscallN(objc_msgSend_ptr, uArgs...)
	return ID(ret)
}

func (id ID) SendRect(sel Selector, r CGRect, ptr uintptr) ID {
	return ID(objc_msgSend_Rect_ptr(uintptr(id), uintptr(sel), r, ptr))
}

func (cls Class) SendRect(sel Selector, r CGRect, ptr uintptr) ID {
	return ID(objc_msgSend_Rect_ptr(uintptr(cls), uintptr(sel), r, ptr))
}

func (id ID) SendRectStyle(sel Selector, r CGRect, style, backing uintptr, defer_ bool) ID {
	return ID(objc_msgSend_Rect_uint_uint_bool(uintptr(id), uintptr(sel), r, style, backing, defer_))
}

func (cls Class) SendRectStyle(sel Selector, r CGRect, style, backing uintptr, defer_ bool) ID {
	return ID(objc_msgSend_Rect_uint_uint_bool(uintptr(cls), uintptr(sel), r, style, backing, defer_))
}

func (id ID) SendSize(sel Selector, s CGSize) ID {
	return ID(objc_msgSend_Size(uintptr(id), uintptr(sel), s))
}

func (id ID) SendPoint(sel Selector, p CGPoint) ID {
	return ID(objc_msgSend_Point(uintptr(id), uintptr(sel), p))
}

// Helper functions for class creation
func AllocateClassPair(superclass Class, name string, extraBytes int) Class {
	b := append([]byte(name), 0)
	return Class(objc_allocateClassPair(uintptr(superclass), &b[0], extraBytes))
}

func RegisterClassPair(cls Class) {
	objc_registerClassPair(uintptr(cls))
}

func AddMethod(cls Class, name Selector, imp interface{}, types string) bool {
	b := append([]byte(types), 0)
	return class_addMethod(uintptr(cls), uintptr(name), purego.NewCallback(imp), &b[0])
}

// StringToNSString converts a Go string to NSString
func StringToNSString(s string) ID {
	nsString := GetClass("NSString")
	alloc := RegisterName("alloc")
	initWithUTF8String := RegisterName("initWithUTF8String:")

	// Convert string to C string (byte slice)
	// We need to ensure it's null-terminated and passed as *byte
	b := []byte(s)
	b = append(b, 0)

	obj := nsString.Send(alloc)
	// We pass pointer to first byte using unsafe.Pointer
	return obj.Send(initWithUTF8String, unsafe.Pointer(&b[0]))
}

// NSStringToString converts NSString to Go string
func NSStringToString(nsStr ID) string {
	if nsStr == 0 {
		return ""
	}
	utf8String := RegisterName("UTF8String")
	cstr := nsStr.Send(utf8String)
	if cstr == 0 {
		return ""
	}

	ptr := unsafe.Pointer(uintptr(cstr))
	var bytes []byte
	// Read until null terminator
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

// BytesToNSData converts a byte slice to NSData
func BytesToNSData(b []byte) ID {
	if len(b) == 0 {
		return GetClass("NSData").Send(RegisterName("data"))
	}
	return GetClass("NSData").Send(RegisterName("dataWithBytes:length:"), unsafe.Pointer(&b[0]), len(b))
}

// Core Graphics types
type CGFloat float64

type CGPoint struct {
	X CGFloat
	Y CGFloat
}

type CGSize struct {
	Width  CGFloat
	Height CGFloat
}

type CGRect struct {
	X      CGFloat
	Y      CGFloat
	Width  CGFloat
	Height CGFloat
}

// Constants
const (
	NSWindowStyleMaskTitled         = 1 << 0
	NSWindowStyleMaskClosable       = 1 << 1
	NSWindowStyleMaskMiniaturizable = 1 << 2
	NSWindowStyleMaskResizable      = 1 << 3
	NSWindowStyleMaskFullScreen     = 1 << 14

	NSBackingStoreBuffered = 2

	NSApplicationActivationPolicyRegular = 0

	NSFloatingWindowLevel = 3
	NSNormalWindowLevel   = 0
)
