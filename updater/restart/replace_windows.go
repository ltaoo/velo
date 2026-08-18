//go:build windows

package restart

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

func replace_process(request Request) error {
	command := exec.Command(request.ExecutablePath, request.Arguments[1:]...)
	command.Env = request.Environment
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	if err := command.Start(); err != nil {
		return fmt.Errorf("start replacement process: %w", err)
	}
	if err := command.Wait(); err != nil {
		var exit_error *exec.ExitError
		if !errors.As(err, &exit_error) {
			return fmt.Errorf("wait for replacement process: %w", err)
		}
	}
	return nil
}
