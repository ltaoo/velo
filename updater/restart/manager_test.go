package restart

import (
	"errors"
	"reflect"
	"testing"
)

func TestManagerRequestsShutdownAndReplacesOnce(t *testing.T) {
	manager := NewManager()
	shutdown_calls := 0
	replace_calls := 0
	var received_request Request
	manager.replace_process_fn = func(request Request) error {
		replace_calls++
		received_request = clone_request(request)
		return nil
	}

	err := manager.Request(
		"/tmp/example",
		[]string{"/tmp/example", "server"},
		[]string{"EXAMPLE=value"},
		func() { shutdown_calls++ },
	)
	if err != nil {
		t.Fatalf("request restart: %v", err)
	}
	if shutdown_calls != 1 || !manager.Pending() {
		t.Fatalf("unexpected pending state: shutdown=%d pending=%t", shutdown_calls, manager.Pending())
	}

	if err := manager.Request(
		"/tmp/ignored",
		[]string{"/tmp/ignored"},
		nil,
		func() { shutdown_calls++ },
	); err != nil {
		t.Fatalf("repeat restart request: %v", err)
	}
	if shutdown_calls != 1 {
		t.Fatalf("repeat request invoked shutdown %d times", shutdown_calls)
	}

	replaced, err := manager.ReplaceIfRequested()
	if err != nil || !replaced {
		t.Fatalf("replace pending process: replaced=%t err=%v", replaced, err)
	}
	if replace_calls != 1 {
		t.Fatalf("replace calls = %d, want 1", replace_calls)
	}
	want_request := Request{
		ExecutablePath: "/tmp/example",
		Arguments:      []string{"/tmp/example", "server"},
		Environment:    []string{"EXAMPLE=value"},
	}
	if !reflect.DeepEqual(received_request, want_request) {
		t.Fatalf("replacement request = %#v, want %#v", received_request, want_request)
	}
	if manager.Pending() {
		t.Fatal("successful replacement remained pending")
	}

	replaced, err = manager.ReplaceIfRequested()
	if err != nil || replaced {
		t.Fatalf("second replacement: replaced=%t err=%v", replaced, err)
	}
}

func TestManagerRetainsFailedReplacement(t *testing.T) {
	manager := NewManager()
	want_error := errors.New("replacement failed")
	manager.replace_process_fn = func(request Request) error { return want_error }

	if err := manager.Request(
		"/tmp/example",
		[]string{"/tmp/example"},
		nil,
		func() {},
	); err != nil {
		t.Fatalf("request restart: %v", err)
	}
	replaced, err := manager.ReplaceIfRequested()
	if !replaced || !errors.Is(err, want_error) {
		t.Fatalf("failed replacement: replaced=%t err=%v", replaced, err)
	}
	if !manager.Pending() {
		t.Fatal("failed replacement request was discarded")
	}
}

func TestManagerRejectsInvalidRequests(t *testing.T) {
	manager := NewManager()
	if err := manager.Request("", []string{"example"}, nil, func() {}); err == nil {
		t.Fatal("empty executable path was accepted")
	}
	if err := manager.Request("/tmp/example", nil, nil, func() {}); err == nil {
		t.Fatal("empty arguments were accepted")
	}
	if err := manager.Request("/tmp/example", []string{"/tmp/example"}, nil, nil); err == nil {
		t.Fatal("nil shutdown callback was accepted")
	}
}
