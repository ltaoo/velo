//go:build !darwin && !windows

package webview

func sendCallback(id, result string) {}
func sendMessage(payload string) bool { return false }
func notifyReady()                    {}
