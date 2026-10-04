package install

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// NetStatus is the state of the connection of the machine.
type NetStatus struct {
	Online bool   // there is a working internet connection
	Kind   string // "ethernet" | "wifi" | "" when disconnected
	Name   string // the connection or network name
}

// AP is a Wi-Fi network in range.
type AP struct {
	SSID     string
	Signal   int // 0-100
	Security string
	InUse    bool
}

// Open reports whether the network needs no password.
func (a AP) Open() bool { return a.Security == "" || a.Security == "--" }

// Network is what the wizard needs from the network: look, scan and connect. The real one is NetworkManager.
type Network interface {
	Status(ctx context.Context) (NetStatus, error)
	Scan(ctx context.Context) ([]AP, error)
	Connect(ctx context.Context, ssid, password string) error
}

// NMCLI is NetworkManager's command line.
type NMCLI struct{ Bin string }

// NewNMCLI returns the network client of a real machine.
func NewNMCLI() *NMCLI { return &NMCLI{Bin: "nmcli"} }

func (n *NMCLI) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, n.Bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return out.String(), nil
}

// splitTerse splits a line of `nmcli -t` on its colons, honouring the \: escape.
func splitTerse(line string) []string {
	var fields []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line):
			cur.WriteByte(line[i+1])
			i++
		case line[i] == ':':
			fields = append(fields, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(line[i])
		}
	}
	return append(fields, cur.String())
}

// ParseStatus reads `nmcli -t -f TYPE,STATE,CONNECTION device` and the connectivity word.
func ParseStatus(devices, connectivity string) NetStatus {
	conn := strings.TrimSpace(connectivity)
	st := NetStatus{Online: conn == "full"}
	for _, l := range strings.Split(devices, "\n") {
		f := splitTerse(l)
		if len(f) < 3 || f[1] != "connected" {
			continue
		}
		if f[0] == "ethernet" || f[0] == "wifi" {
			st.Kind, st.Name = f[0], f[2]
			// without a connectivity check NetworkManager says "unknown": a connected device is the best answer
			st.Online = st.Online || conn == "unknown" || conn == ""
			if f[0] == "ethernet" {
				break // a wire beats Wi-Fi as the answer
			}
		}
	}
	return st
}

// Status looks at the connection.
func (n *NMCLI) Status(ctx context.Context) (NetStatus, error) {
	dev, err := n.run(ctx, "-t", "-f", "TYPE,STATE,CONNECTION", "device")
	if err != nil {
		return NetStatus{}, err
	}
	conn, _ := n.run(ctx, "-t", "-g", "CONNECTIVITY", "general", "status")
	return ParseStatus(dev, conn), nil
}

// ParseAPs reads `nmcli -t -f IN-USE,SSID,SIGNAL,SECURITY device wifi list`: one entry per name (the
// strongest), the one in use first, then by signal.
func ParseAPs(out string) []AP {
	best := map[string]AP{}
	for _, l := range strings.Split(out, "\n") {
		f := splitTerse(l)
		if len(f) < 4 || strings.TrimSpace(f[1]) == "" {
			continue
		}
		sig, _ := strconv.Atoi(f[2])
		ap := AP{SSID: f[1], Signal: sig, Security: f[3], InUse: strings.TrimSpace(f[0]) == "*"}
		if old, ok := best[ap.SSID]; !ok || ap.InUse || (!old.InUse && ap.Signal > old.Signal) {
			best[ap.SSID] = ap
		}
	}
	aps := make([]AP, 0, len(best))
	for _, a := range best {
		aps = append(aps, a)
	}
	sort.Slice(aps, func(i, j int) bool {
		if aps[i].InUse != aps[j].InUse {
			return aps[i].InUse
		}
		if aps[i].Signal != aps[j].Signal {
			return aps[i].Signal > aps[j].Signal
		}
		return aps[i].SSID < aps[j].SSID
	})
	return aps
}

// Scan lists the Wi-Fi networks in range.
func (n *NMCLI) Scan(ctx context.Context) ([]AP, error) {
	out, err := n.run(ctx, "-t", "-f", "IN-USE,SSID,SIGNAL,SECURITY", "device", "wifi", "list", "--rescan", "yes")
	if err != nil {
		return nil, err
	}
	return ParseAPs(out), nil
}

// Connect joins a Wi-Fi network. nmcli only takes the password as an argument; on the live system there
// is a single user, so it is visible to nobody else, and it is never written to a file.
func (n *NMCLI) Connect(ctx context.Context, ssid, password string) error {
	args := []string{"device", "wifi", "connect", ssid}
	if password != "" {
		args = append(args, "password", password)
	}
	if _, err := n.run(ctx, args...); err != nil {
		return fmt.Errorf("could not join %q: %w", ssid, err)
	}
	return nil
}
