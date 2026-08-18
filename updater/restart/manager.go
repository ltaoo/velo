package restart

import (
	"fmt"
	"os"
	"sync"
)

// Request describes the process image that should replace the current one.
type Request struct {
	ExecutablePath string
	Arguments      []string
	Environment    []string
}

// Manager coordinates a two-phase restart. RequestCurrent records the
// replacement and asks the host to shut down. ReplaceIfRequested must be
// called only after the host has released its resources.
type Manager struct {
	mu                 sync.Mutex
	pending_request    *Request
	replacing          bool
	replace_process_fn func(Request) error
}

// NewManager creates an idle restart manager.
func NewManager() *Manager {
	return &Manager{replace_process_fn: replace_process}
}

// CurrentRequest returns a replacement request for the currently running
// executable, preserving its arguments and environment.
func CurrentRequest() (Request, error) {
	executable_path, err := os.Executable()
	if err != nil {
		return Request{}, fmt.Errorf("resolve executable: %w", err)
	}
	return Request{
		ExecutablePath: executable_path,
		Arguments:      append([]string{executable_path}, os.Args[1:]...),
		Environment:    append([]string(nil), os.Environ()...),
	}, nil
}

// RequestCurrent records the current process and asks the host application to
// begin graceful shutdown.
func (m *Manager) RequestCurrent(request_shutdown func()) error {
	request, err := CurrentRequest()
	if err != nil {
		return err
	}
	return m.Request(
		request.ExecutablePath,
		request.Arguments,
		request.Environment,
		request_shutdown,
	)
}

// Request records a replacement process and asks the host application to
// begin graceful shutdown. Repeated requests while one is pending are
// idempotent and do not invoke request_shutdown again.
func (m *Manager) Request(
	executable_path string,
	arguments []string,
	environment []string,
	request_shutdown func(),
) error {
	if request_shutdown == nil {
		return fmt.Errorf("request shutdown callback is nil")
	}
	request := Request{
		ExecutablePath: executable_path,
		Arguments:      append([]string(nil), arguments...),
		Environment:    append([]string(nil), environment...),
	}
	if err := validate_request(request); err != nil {
		return err
	}

	m.mu.Lock()
	if m.replacing {
		m.mu.Unlock()
		return fmt.Errorf("process replacement is already running")
	}
	if m.pending_request != nil {
		m.mu.Unlock()
		return nil
	}
	m.pending_request = &request
	m.mu.Unlock()

	request_shutdown()
	return nil
}

// Pending reports whether a replacement has been requested.
func (m *Manager) Pending() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pending_request != nil
}

// ReplaceIfRequested replaces the current process when a request is pending.
// The request is retained when replacement fails so the caller can retry.
func (m *Manager) ReplaceIfRequested() (bool, error) {
	m.mu.Lock()
	if m.pending_request == nil {
		m.mu.Unlock()
		return false, nil
	}
	if m.replacing {
		m.mu.Unlock()
		return true, fmt.Errorf("process replacement is already running")
	}
	request := clone_request(*m.pending_request)
	m.replacing = true
	m.mu.Unlock()

	err := m.replace_process_fn(request)

	m.mu.Lock()
	m.replacing = false
	if err == nil {
		m.pending_request = nil
	}
	m.mu.Unlock()
	if err != nil {
		return true, fmt.Errorf("replace current process: %w", err)
	}
	return true, nil
}

// Replace immediately replaces the current process with request. Applications
// should normally use Manager so replacement happens after graceful shutdown.
func Replace(request Request) error {
	if err := validate_request(request); err != nil {
		return err
	}
	return replace_process(clone_request(request))
}

// ReplaceCurrent immediately replaces the current process while preserving
// its executable path, arguments, and environment.
func ReplaceCurrent() error {
	request, err := CurrentRequest()
	if err != nil {
		return err
	}
	return Replace(request)
}

func validate_request(request Request) error {
	if request.ExecutablePath == "" {
		return fmt.Errorf("replacement executable path is empty")
	}
	if len(request.Arguments) == 0 {
		return fmt.Errorf("replacement arguments are empty")
	}
	return nil
}

func clone_request(request Request) Request {
	return Request{
		ExecutablePath: request.ExecutablePath,
		Arguments:      append([]string(nil), request.Arguments...),
		Environment:    append([]string(nil), request.Environment...),
	}
}
