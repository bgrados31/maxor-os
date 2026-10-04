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
	// DetectZone asks which time zone this connection is in (it contacts a service, so only on request).
	DetectZone func(ctx context.Context, known []string) (string, error)
	// ApplyKeyboard makes the live session use the layout, so the user can test it by typing.
	ApplyKeyboard func(Layout)
	// Reboot restarts the machine (the last step).
	Reboot func() error
	// PowerOff turns the machine off; Shell leaves the installer for a text console.
	PowerOff func() error
	Shell    func() error
}

// RealDeps are the dependencies of a real machine.
func RealDeps() *Deps {
	return &Deps{
		Engine:        NewCLI(),
		Net:           NewNMCLI(),
		Sys:           realSys,
		Zones:         Timezones,
		DetectZone:    DetectZone,
		ApplyKeyboard: applyKeyboard,
		Reboot:        reboot,
		PowerOff:      func() error { return systemctl("poweroff") },
		Shell:         func() error { return asRoot("chvt", "2") },
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
	switch {
	case os.Getenv("SWAYSOCK") != "": // the installation medium: one sway session
		if _, err := exec.LookPath("swaymsg"); err == nil {
			_ = exec.Command("swaymsg", "input", "type:keyboard", "xkb_layout", l.XKB, "xkb_variant", l.Variant).Run()
		}
	case os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "": // a Hyprland session (an installed system)
		_ = exec.Command("hyprctl", "keyword", "input:kb_layout", l.XKB).Run()
		_ = exec.Command("hyprctl", "keyword", "input:kb_variant", l.Variant).Run()
	}
}

func reboot() error { return systemctl("reboot") }

func systemctl(action string) error { return asRoot("systemctl", action) }

// asRoot runs a command as root: directly if it already is, through sudo (without asking) if not.
func asRoot(name string, args ...string) error {
	if os.Geteuid() == 0 {
		return exec.Command(name, args...).Run()
	}
	return exec.Command("sudo", append([]string{"-n", name}, args...)...).Run()
}
