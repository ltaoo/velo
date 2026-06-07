//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func confirmExternalBrowserOpen(target string) (bool, error) {
	message := externalBrowserConfirmMessage(target)
	commands := [][]string{
		{"zenity", "--question", "--modal", "--title=打开外部链接", "--ok-label=使用默认浏览器打开", "--cancel-label=取消", "--text=" + message},
		{"kdialog", "--warningyesno", message, "--title", "打开外部链接"},
		{"xmessage", "-center", "-buttons", "使用默认浏览器打开:0,取消:1", message},
	}

	for _, candidate := range commands {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			continue
		}
		cmd := exec.Command(candidate[0], candidate[1:]...)
		if err := cmd.Run(); err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}

	fmt.Fprintf(os.Stderr, "Confirm external link open:\n%s\n", message)
	return false, nil
}
