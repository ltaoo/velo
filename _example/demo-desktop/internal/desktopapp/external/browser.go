package external

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func NormalizeBrowserURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("url is required")
	}
	for _, r := range value {
		if r <= 0x20 || r == 0x7f {
			return "", fmt.Errorf("invalid url")
		}
	}

	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() {
		return "", fmt.Errorf("invalid url")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("only http and https URLs can be opened externally")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("url host is required")
	}
	parsed.Scheme = scheme
	return parsed.String(), nil
}

func OpenBrowser(target string) error {
	cmd, err := browserCommand(target)
	if err != nil {
		return err
	}
	cmd.Env = os.Environ()
	return cmd.Start()
}

func browserCommand(target string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target), nil
	default:
		return exec.Command("xdg-open", target), nil
	}
}

func BrowserConfirmMessage(target string) string {
	return "即将使用默认浏览器打开以下链接：\n\n" + target + "\n\n是否继续？"
}
