package install

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// SysInfo is what the welcome step shows about this machine.
type SysInfo struct {
	UEFI     bool
	RAMBytes int64
}

// Deps is everything the wizard needs from the outside world, as one value, so the tests can swap each
// piece for a double.
type Deps struct {
	Engine Engine
	Net    Network
	Sys    func() SysInfo
	// Zones lists the time zones.
	Zones func(ctx context.Context) []string
	// ApplyKeyboard makes the live session use the layout, so the user can test it by typing.
	ApplyKeyboard func(Layout)
	// Reboot restarts the machine (the last step).
	Reboot func() error
}

// RealDeps are the dependencies of a real machine.
func RealDeps() *Deps {
	return &Deps{
		Engine:        NewCLI(),
		Net:           NewNMCLI(),
		Sys:           realSys,
		Zones:         Timezones,
		ApplyKeyboard: applyKeyboard,
		Reboot:        reboot,
	}
}

func realSys() SysInfo {
	info := SysInfo{}
	if _, err := os.Stat("/sys/firmware/efi"); err == nil {
		info.UEFI = true
	}
	if f, err := os.Open("/proc/meminfo"); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if rest, ok := strings.CutPrefix(sc.Text(), "MemTotal:"); ok {
				fields := strings.Fields(rest)
				if len(fields) > 0 {
					kb, _ := strconv.ParseInt(fields[0], 10, 64)
					info.RAMBytes = kb * 1024
				}
				break
			}
		}
	}
	return info
}

// applyKeyboard tells the compositor about the layout. It is a convenience: if there is no Hyprland, nothing happens.
func applyKeyboard(l Layout) {
	if _, err := exec.LookPath("hyprctl"); err != nil {
		return
	}
	_ = exec.Command("hyprctl", "keyword", "input:kb_layout", l.XKB).Run()
	_ = exec.Command("hyprctl", "keyword", "input:kb_variant", l.Variant).Run()
}

func reboot() error {
	if os.Geteuid() == 0 {
		return exec.Command("systemctl", "reboot").Run()
	}
	return exec.Command("sudo", "-n", "systemctl", "reboot").Run()
}
