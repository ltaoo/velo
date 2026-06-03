# Velo Notification Demo

This example shows a form that sends system-level desktop notifications through `github.com/ltaoo/velo/notification`.

## Run

```bash
go run .
```

## Message Types

The form supports `info`, `success`, `warning`, and `error` notifications.

- macOS uses `UNUserNotificationCenter`.
- Windows uses the system Toast notification API through PowerShell.
- Linux uses `notify-send`, with a D-Bus fallback. `error` maps to critical urgency.

## Cleanup

The Cleanup button removes pending and delivered notifications where the platform supports it. Notification permission entries themselves are managed by the OS and generally cannot be revoked by the app through a public API.

## Remote Push

The Register APNs button calls the macOS remote notification registration API and displays the APNs device token or registration error.

Real APNs delivery requires:

- A signed `.app` bundle with a stable bundle identifier.
- An Apple Developer provisioning profile.
- The `com.apple.developer.aps-environment` entitlement (`development` or `production`).
- A server that sends APNs payloads to the returned device token.
