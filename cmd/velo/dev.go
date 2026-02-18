package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
)

type crlfWriter struct{ w *os.File }

func (c crlfWriter) Write(p []byte) (int, error) {
	replaced := bytes.ReplaceAll(p, []byte("\n"), []byte("\r\n"))
	_, err := c.w.Write(replaced)
	return len(p), err
}

func runDev(dir string) error {
	mainFile := filepath.Join(dir, "main.go")
	if _, err := os.Stat(mainFile); err != nil {
		return fmt.Errorf("main.go not found in %s", dir)
	}

	width, height := 80, 24
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		width, height = w, h
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	keyCh := make(chan byte, 1)
	go func() {
		buf := make([]byte, 1)
		for {
			if n, err := os.Stdin.Read(buf); n > 0 && err == nil {
				keyCh <- buf[0]
			}
		}
	}()

	for {
		// Clear screen, set scroll region leaving 2 lines for footer
		fmt.Printf("\033[2J\033[H\033[1;%dr\033[H", height-2)
		drawFooter(width, height)

		cmd := exec.Command("go", "run", "main.go")
		cmd.Dir = dir
		crlf := crlfWriter{os.Stdout}
		cmd.Stdout = crlf
		cmd.Stderr = crlf
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		doneCh := make(chan error, 1)
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() { doneCh <- cmd.Wait() }()

		action := ""
	loop:
		for {
			select {
			case <-doneCh:
				// process exited, wait for key
				select {
				case k := <-keyCh:
					switch k {
					case 'r', 'R':
						action = "restart"
					default:
						action = "quit"
					}
				}
				break loop
			case k := <-keyCh:
				switch k {
				case 'r', 'R':
					action = "restart"
					syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
					<-doneCh
					break loop
				case 'q', 'Q', 3: // 3 = Ctrl+C
					action = "quit"
					syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
					<-doneCh
					break loop
				}
			}
		}

		if action != "restart" {
			break
		}
	}

	// Reset terminal
	fmt.Printf("\033[r\033[%d;0H\n", height)
	return nil
}

func drawFooter(width, height int) {
	sep := strings.Repeat("─", width)
	info := fmt.Sprintf(" velo %s   R refresh · Q quit ", version)
	fmt.Printf("\033[s\033[%d;0H\033[90m%s\033[0m\033[%d;0H\033[90m%s\033[0m\033[u", height-1, sep, height, info)
}
