package file

// ShowFileSelectDialog shows a file selection dialog and returns the selected file path.
// animationType: "default", "none", "document", "utility", "alert", "sheet"
func ShowFileSelectDialog(animationType string) (string, error) {
	// return "", errors.New("unsupported platform: " + runtime.GOOS)
	return showFileSelectDialog(animationType)
}
