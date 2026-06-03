package notification

import "errors"

// RemotePushCallbacks receives macOS APNs registration and remote notification events.
type RemotePushCallbacks struct {
	OnToken        func(token string)
	OnError        func(err error)
	OnNotification func(payload string)
}

// RegisterRemotePush registers the current app for platform remote push
// notifications where supported.
//
// On macOS this registers for APNs remote notifications. The app must be a
// signed .app bundle with a valid bundle identifier and the aps-environment
// entitlement supplied by an Apple provisioning profile.
func RegisterRemotePush(callbacks RemotePushCallbacks) error {
	if callbacks.OnToken == nil && callbacks.OnError == nil && callbacks.OnNotification == nil {
		return errors.New("notification: at least one remote push callback is required")
	}
	return registerRemotePushNative(callbacks)
}
