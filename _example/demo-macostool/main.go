package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/ltaoo/velo"
)

//go:embed frontend
var frontendFS embed.FS

//go:embed assets/appicon.png
var appIcon []byte

type powerProfile struct {
	DisplaySleep int  `json:"display_sleep"`
	SystemSleep  int  `json:"system_sleep"`
	DiskSleep    int  `json:"disk_sleep"`
	PowerNap     bool `json:"power_nap"`
}

type applyRequest struct {
	AC            powerProfile `json:"ac"`
	Battery       powerProfile `json:"battery"`
	HibernateMode int          `json:"hibernate_mode"`
	Standby       bool         `json:"standby"`
	AutoPowerOff  bool         `json:"auto_power_off"`
}

type pmsetStatus struct {
	Supported bool                    `json:"supported"`
	OS        string                  `json:"os"`
	AC        map[string]string       `json:"ac"`
	Battery   map[string]string       `json:"battery"`
	Raw       string                  `json:"raw"`
	Presets   map[string]applyRequest `json:"presets"`
}

func main() {
	app := velo.NewApp(&velo.VeloAppOpt{Mode: velo.ModeBridge, IconData: appIcon})

	app.Get("/api/app", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{
			"name": "Velo macOS Tool Demo",
			"os":   runtime.GOOS,
		})
	})

	app.Get("/api/power/status", func(c *velo.BoxContext) interface{} {
		status, err := readPMSet()
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(status)
	})

	app.Post("/api/power/apply", func(c *velo.BoxContext) interface{} {
		var req applyRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		if err := validateApplyRequest(req); err != nil {
			return c.Error(err.Error())
		}
		if err := applyPMSet(req); err != nil {
			return c.Error(err.Error())
		}
		status, err := readPMSet()
		if err != nil {
			return c.Ok(velo.H{"applied": true, "status_error": err.Error()})
		}
		return c.Ok(velo.H{"applied": true, "status": status})
	})

	app.NewWebview(&velo.VeloWebviewOpt{
		Title:      "macOS Power Tool",
		FrontendFS: frontendFS,
		Pathname:   "/",
		Width:      860,
		Height:     720,
	})
	app.Run()
}

func readPMSet() (pmsetStatus, error) {
	status := pmsetStatus{
		Supported: runtime.GOOS == "darwin",
		OS:        runtime.GOOS,
		AC:        map[string]string{},
		Battery:   map[string]string{},
		Presets:   presets(),
	}
	if runtime.GOOS != "darwin" {
		return status, nil
	}

	out, err := exec.Command("/usr/bin/pmset", "-g", "custom").CombinedOutput()
	status.Raw = strings.TrimSpace(string(out))
	if err != nil {
		return status, fmt.Errorf("pmset status failed: %w: %s", err, status.Raw)
	}
	parsePMSetCustom(status.Raw, status.AC, status.Battery)
	return status, nil
}

func parsePMSetCustom(raw string, ac map[string]string, battery map[string]string) {
	var current map[string]string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "AC Power:"):
			current = ac
			continue
		case strings.HasPrefix(line, "Battery Power:"):
			current = battery
			continue
		case line == "" || current == nil:
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			current[fields[0]] = strings.Join(fields[1:], " ")
		}
	}
}

func applyPMSet(req applyRequest) error {
	if runtime.GOOS != "darwin" {
		return errors.New("this demo can only change power settings on macOS")
	}

	acArgs := pmsetArgs("-c", req.AC)
	batteryArgs := pmsetArgs("-b", req.Battery)
	allArgs := []string{"-a", "hibernatemode", strconv.Itoa(req.HibernateMode), "standby", boolInt(req.Standby), "autopoweroff", boolInt(req.AutoPowerOff)}
	script := fmt.Sprintf(
		`do shell script %s with administrator privileges`,
		strconv.Quote(shellCommand("/usr/bin/pmset", acArgs)+"; "+shellCommand("/usr/bin/pmset", batteryArgs)+"; "+shellCommand("/usr/bin/pmset", allArgs)),
	)

	out, err := exec.Command("/usr/bin/osascript", "-e", script).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("pmset apply failed: %s", msg)
	}
	return nil
}

func pmsetArgs(scope string, p powerProfile) []string {
	return []string{
		scope,
		"displaysleep", strconv.Itoa(p.DisplaySleep),
		"sleep", strconv.Itoa(p.SystemSleep),
		"disksleep", strconv.Itoa(p.DiskSleep),
		"powernap", boolInt(p.PowerNap),
	}
}

func shellCommand(name string, args []string) string {
	var buf bytes.Buffer
	buf.WriteString(shellQuote(name))
	for _, arg := range args {
		buf.WriteByte(' ')
		buf.WriteString(shellQuote(arg))
	}
	return buf.String()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func validateApplyRequest(req applyRequest) error {
	if err := validateProfile("AC power", req.AC); err != nil {
		return err
	}
	if err := validateProfile("battery power", req.Battery); err != nil {
		return err
	}
	switch req.HibernateMode {
	case 0, 3, 25:
		return nil
	default:
		return errors.New("hibernate mode must be 0, 3, or 25")
	}
}

func validateProfile(name string, p powerProfile) error {
	for label, value := range map[string]int{
		"display sleep": p.DisplaySleep,
		"system sleep":  p.SystemSleep,
		"disk sleep":    p.DiskSleep,
	} {
		if value < 0 || value > 1440 {
			return fmt.Errorf("%s %s must be between 0 and 1440 minutes", name, label)
		}
	}
	return nil
}

func boolInt(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func presets() map[string]applyRequest {
	return map[string]applyRequest{
		"balanced": {
			AC:            powerProfile{DisplaySleep: 15, SystemSleep: 0, DiskSleep: 10, PowerNap: true},
			Battery:       powerProfile{DisplaySleep: 5, SystemSleep: 15, DiskSleep: 10, PowerNap: false},
			HibernateMode: 3,
			Standby:       true,
			AutoPowerOff:  true,
		},
		"pluggedNeverBatteryTimed": {
			AC:            powerProfile{DisplaySleep: 0, SystemSleep: 0, DiskSleep: 0, PowerNap: true},
			Battery:       powerProfile{DisplaySleep: 5, SystemSleep: 20, DiskSleep: 10, PowerNap: false},
			HibernateMode: 3,
			Standby:       true,
			AutoPowerOff:  true,
		},
		"presentation": {
			AC:            powerProfile{DisplaySleep: 0, SystemSleep: 0, DiskSleep: 0, PowerNap: false},
			Battery:       powerProfile{DisplaySleep: 0, SystemSleep: 0, DiskSleep: 0, PowerNap: false},
			HibernateMode: 3,
			Standby:       false,
			AutoPowerOff:  false,
		},
	}
}
