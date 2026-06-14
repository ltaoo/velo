package autostart

import (
	"strings"
	"unicode"
)

func safeIdentifier(appName string) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "app"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(unicode.ToLower(r))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	identifier := strings.Trim(b.String(), "-._")
	if identifier == "" {
		return "app"
	}
	return identifier
}

func displayName(appName string) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		return "App"
	}
	return name
}
