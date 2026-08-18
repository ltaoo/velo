//go:build windows

package main

import "os/exec"

func configure_process_group(cmd *exec.Cmd) {}

func terminate_process(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}
