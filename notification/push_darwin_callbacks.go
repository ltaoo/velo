//go:build darwin
// +build darwin

package notification

/*
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

//export veloRemotePushTokenCallback
func veloRemotePushTokenCallback(cToken *C.char) {
	if cToken == nil {
		return
	}
	token := C.GoString(cToken)
	C.free(unsafe.Pointer(cToken))

	remotePushMu.RLock()
	cb := remotePushCallbacks.OnToken
	remotePushMu.RUnlock()
	if cb != nil {
		go cb(token)
	}
}

//export veloRemotePushErrorCallback
func veloRemotePushErrorCallback(cError *C.char) {
	if cError == nil {
		return
	}
	message := C.GoString(cError)
	C.free(unsafe.Pointer(cError))

	remotePushMu.RLock()
	cb := remotePushCallbacks.OnError
	remotePushMu.RUnlock()
	if cb != nil {
		go cb(errors.New(message))
	}
}

//export veloRemotePushPayloadCallback
func veloRemotePushPayloadCallback(cPayload *C.char) {
	if cPayload == nil {
		return
	}
	payload := C.GoString(cPayload)
	C.free(unsafe.Pointer(cPayload))

	remotePushMu.RLock()
	cb := remotePushCallbacks.OnNotification
	remotePushMu.RUnlock()
	if cb != nil {
		go cb(payload)
	}
}
