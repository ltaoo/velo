package file

type FileSelectOptions struct {
	AnimationType string
	AllowedTypes  []string
	Directory     string
}

// ShowFileSelectDialog shows a file selection dialog and returns the selected file path.
// animationType: "default", "none", "document", "utility", "alert", "sheet"
func ShowFileSelectDialog(animationType string) (string, error) {
	// return "", errors.New("unsupported platform: " + runtime.GOOS)
	return ShowFileSelectDialogWithOptions(FileSelectOptions{AnimationType: animationType})
}

// ShowFileSelectDialogWithTypes shows a file selection dialog with allowed file type filtering.
// animationType: "default", "none", "document", "utility", "alert", "sheet"
// allowedTypes: file extensions without dot, e.g. []string{"txt", "md"}
func ShowFileSelectDialogWithTypes(animationType string, allowedTypes []string) (string, error) {
	return ShowFileSelectDialogWithOptions(FileSelectOptions{AnimationType: animationType, AllowedTypes: allowedTypes})
}

func ShowFileSelectDialogWithOptions(options FileSelectOptions) (string, error) {
	if options.AnimationType == "" {
		options.AnimationType = "default"
	}
	return showFileSelectDialog(options)
}
