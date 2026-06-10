//go:build !darwin && !windows

package inputsource

func list() ([]Source, error) {
	return nil, ErrUnsupported
}

func current() (Source, error) {
	return Source{}, ErrUnsupported
}

func selectSource(sourceID string) error {
	return ErrUnsupported
}

func frontmostApp() (App, error) {
	return App{}, ErrUnsupported
}
