package autostart

type AutoStart interface {
	Enable() error
	Disable() error
	IsEnabled() bool
}

func New(appName string) AutoStart {
	return newPlatformAutoStart(appName)
}
