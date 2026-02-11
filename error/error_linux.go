//go:build linux
// +build linux

package error

import (
	"fmt"
	"os"
	"os/exec"
)

// showErrorDialog shows an error dialog on Linux using zenity or xmessage
func showErrorDialog(message string) {
	// Try zenity first
	cmd := exec.Command("zenity", "--error", "--text="+message, "--title=Application Error")
	if err := cmd.Run(); err == nil {
		return
	}

	// Fallback to xmessage
	cmd = exec.Command("xmessage", "-center", fmt.Sprintf("Application Error\n\n%s", message))
	if err := cmd.Run(); err == nil {
		return
	}

	// If both fail, just print to stderr
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", message)
}
