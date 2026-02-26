package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type CheckResult struct {
	Name    string
	Passed  bool
	Message string // Short status message
	Details string // Detailed error or info
}

type DoctorCheck func() CheckResult

func runDoctor() error {
	fmt.Printf("Doctor summary (to see all details, run velo doctor -v):\n")

	checks := []DoctorCheck{
		checkGo,
		checkGit,
	}

	if runtime.GOOS == "darwin" {
		checks = append(checks, checkXcode, checkCocoaPods, checkGomobile, checkIosSdk)
	} else if runtime.GOOS == "linux" {
		checks = append(checks, checkLinuxDeps)
	} else if runtime.GOOS == "windows" {
		checks = append(checks, checkWindowsDeps)
	}

	allPassed := true
	for _, check := range checks {
		res := check()
		mark := "✓"
		if !res.Passed {
			mark = "✗"
			allPassed = false
		}

		fmt.Printf("[%s] %s\n", mark, res.Name)
		if !res.Passed {
			fmt.Printf("    • %s\n", res.Message)
			if res.Details != "" {
				// Indent details
				lines := strings.Split(res.Details, "\n")
				for _, line := range lines {
					fmt.Printf("      %s\n", line)
				}
			}
		} else if res.Message != "" {
			// Show version info even if passed
			fmt.Printf("    • %s\n", res.Message)
		}
	}

	if !allPassed {
		fmt.Println("\nSome checks failed. Please fix the issues above.")
	} else {
		fmt.Println("\nNo issues found! You are ready to develop.")
	}

	return nil
}

func checkGo() CheckResult {
	path, err := exec.LookPath("go")
	if err != nil {
		return CheckResult{
			Name:    "Go Toolchain",
			Passed:  false,
			Message: "Go not found",
			Details: "Install Go from https://go.dev/dl/",
		}
	}

	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return CheckResult{
			Name:    "Go Toolchain",
			Passed:  false,
			Message: "Failed to run 'go version'",
			Details: err.Error(),
		}
	}

	version := strings.TrimSpace(string(out))
	return CheckResult{
		Name:    "Go Toolchain",
		Passed:  true,
		Message: version,
		Details: fmt.Sprintf("Path: %s", path),
	}
}

func checkGit() CheckResult {
	path, err := exec.LookPath("git")
	if err != nil {
		return CheckResult{
			Name:    "Git",
			Passed:  false,
			Message: "Git not found",
			Details: "Install Git from https://git-scm.com/",
		}
	}

	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return CheckResult{
			Name:    "Git",
			Passed:  false,
			Message: "Failed to run 'git --version'",
			Details: err.Error(),
		}
	}

	version := strings.TrimSpace(string(out))
	return CheckResult{
		Name:    "Git",
		Passed:  true,
		Message: version,
		Details: fmt.Sprintf("Path: %s", path),
	}
}

func checkXcode() CheckResult {
	// Check for xcode-select
	_, err := exec.LookPath("xcode-select")
	if err != nil {
		return CheckResult{
			Name:    "Xcode",
			Passed:  false,
			Message: "Xcode tools not found",
			Details: "Install Xcode or Command Line Tools",
		}
	}

	// Check path
	out, err := exec.Command("xcode-select", "-p").Output()
	if err != nil {
		return CheckResult{
			Name:    "Xcode",
			Passed:  false,
			Message: "xcode-select failed",
			Details: err.Error(),
		}
	}

	// Also check for clang/gcc as a proxy for valid toolchain
	_, err = exec.LookPath("clang")
	if err != nil {
		return CheckResult{
			Name:    "Xcode",
			Passed:  false,
			Message: "clang compiler not found",
			Details: "Ensure Command Line Tools are installed via 'xcode-select --install'",
		}
	}

	return CheckResult{
		Name:    "Xcode",
		Passed:  true,
		Message: fmt.Sprintf("Located at %s", strings.TrimSpace(string(out))),
	}
}

func checkCocoaPods() CheckResult {
	path, err := exec.LookPath("pod")
	if err != nil {
		return CheckResult{
			Name:    "CocoaPods",
			Passed:  false, // Optional? Maybe warning? But for iOS it's usually needed.
			Message: "CocoaPods not found (optional for pure macOS, needed for iOS)",
			Details: "Install via 'sudo gem install cocoapods'",
		}
	}

	out, err := exec.Command("pod", "--version").Output()
	if err != nil {
		return CheckResult{
			Name:    "CocoaPods",
			Passed:  false,
			Message: "Failed to run 'pod --version'",
			Details: err.Error(),
		}
	}

	return CheckResult{
		Name:    "CocoaPods",
		Passed:  true,
		Message: strings.TrimSpace(string(out)),
		Details: fmt.Sprintf("Path: %s", path),
	}
}

func checkLinuxDeps() CheckResult {
	// Check for pkg-config
	_, err := exec.LookPath("pkg-config")
	if err != nil {
		return CheckResult{
			Name:    "Linux Toolchain",
			Passed:  false,
			Message: "pkg-config not found",
			Details: "Install pkg-config via your package manager (apt, dnf, etc.)",
		}
	}

	// Check for gtk+-3.0
	cmd := exec.Command("pkg-config", "--exists", "gtk+-3.0")
	if err := cmd.Run(); err != nil {
		return CheckResult{
			Name:    "Linux Dependencies",
			Passed:  false,
			Message: "gtk+-3.0 not found",
			Details: "Install libgtk-3-dev or equivalent",
		}
	}

	return CheckResult{
		Name:    "Linux Dependencies",
		Passed:  true,
		Message: "GTK 3 development headers found",
	}
}

func checkWindowsDeps() CheckResult {
	// Check for gcc (MinGW)
	_, err := exec.LookPath("gcc")
	if err != nil {
		return CheckResult{
			Name:    "Windows Toolchain",
			Passed:  false,
			Message: "gcc not found",
			Details: "Install MinGW or TDM-GCC for CGO support",
		}
	}

	return CheckResult{
		Name:    "Windows Toolchain",
		Passed:  true,
		Message: "GCC found",
	}
}

func checkGomobile() CheckResult {
	path, err := exec.LookPath("gomobile")
	if err != nil {
		return CheckResult{
			Name:    "Gomobile",
			Passed:  false,
			Message: "gomobile not found (required for mobile builds)",
			Details: "Install via 'go install golang.org/x/mobile/cmd/gomobile@latest' and run 'gomobile init'",
		}
	}
	return CheckResult{
		Name:    "Gomobile",
		Passed:  true,
		Message: "Installed",
		Details: fmt.Sprintf("Path: %s", path),
	}
}

func checkIosSdk() CheckResult {
	cmd := exec.Command("xcrun", "--sdk", "iphoneos", "--show-sdk-path")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Name:    "iOS SDK",
			Passed:  false,
			Message: "iOS SDK not found",
			Details: "Ensure Xcode is installed and has iOS SDK components",
		}
	}
	return CheckResult{
		Name:    "iOS SDK",
		Passed:  true,
		Message: "Found",
		Details: fmt.Sprintf("Path: %s", strings.TrimSpace(string(out))),
	}
}
