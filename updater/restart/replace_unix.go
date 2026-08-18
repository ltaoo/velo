//go:build !windows

package restart

import "syscall"

func replace_process(request Request) error {
	return syscall.Exec(
		request.ExecutablePath,
		request.Arguments,
		request.Environment,
	)
}
