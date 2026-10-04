// Package install holds everything the guided installer needs that is not drawing: the answers collected by
// the wizard, their validation, the disks, and the client of the engine (`maxor-install`). The screens only
// show it and change it, so it can be tested without a terminal.
package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Region is a free region of a disk, in 512-byte sectors.
type Region struct{ Start, End int64 }

// State is what the wizard has collected so far. It maps to installer/schema/answers.v1.json. The two
// secrets (Password, Passphrase) never go into the JSON: the password becomes a hash made by the engine,
// and the passphrase travels on standard input.
type State struct {
	Locale     string
	Keymap     string // console keyboard
	XKBLayout  string // desktop keyboard
	XKBVariant string
	Timezone   string

	Disk       string
	Strategy   string // "whole" | "alongside"
	Region     Region
	Filesystem string // "ext4" | "btrfs"
	Encrypt    bool
	Passphrase string
	SwapKind   string // "zram" | "file" | "none"
	SwapGiB    int

	Hostname     string
	Username     string
	Fullname     string
	Password     string
	PasswordHash string
	Autologin    bool

	Theme    string
	Profiles []string
	Offline  bool
	GPU      string // "auto" | "integrated" | "nvidia" | "hybrid"

	Confirmed string // what the user typed to confirm (ERASE or INSTALL)
	PlanHash  string
}

// Default is the state the wizard starts from.
func Default() State {
	return State{
		Locale: "en_US.UTF-8", Keymap: "us", XKBLayout: "us", Timezone: "UTC",
		Strategy: "whole", Filesystem: "btrfs", SwapKind: "zram",
		Hostname: "maxor", Theme: "maxor-dark", GPU: "auto",
	}
}

// Answers renders the state as the JSON document the engine reads. It needs PasswordHash: the plain
// password is never written anywhere.
func (s State) Answers() ([]byte, error) {
	if s.PasswordHash == "" {
		return nil, errors.New("the password has not been hashed yet")
	}
	disk := map[string]any{
		"device":     s.Disk,
		"strategy":   s.Strategy,
		"filesystem": s.Filesystem,
		"encrypt":    map[string]any{"enabled": s.Encrypt},
		"swap":       map[string]any{"kind": s.SwapKind, "gib": s.SwapGiB},
	}
	if s.Strategy == "alongside" {
		disk["region"] = map[string]any{"start": s.Region.Start, "end": s.Region.End}
	}
	if s.PlanHash != "" {
		disk["plan_hash"] = s.PlanHash
	}
	if s.Confirmed != "" {
		disk["confirmed"] = s.Confirmed
	}
	profiles := s.Profiles
	if profiles == nil {
		profiles = []string{}
	}
	doc := map[string]any{
		"schema":   1,
		"locale":   s.Locale,
		"keymap":   s.Keymap,
		"xkb":      map[string]any{"layout": s.XKBLayout, "variant": s.XKBVariant},
		"timezone": s.Timezone,
		"disk":     disk,
		"machine":  map[string]any{"hostname": s.Hostname},
		"user": map[string]any{
			"name": s.Username, "fullname": s.Fullname, "password_hash": s.PasswordHash, "autologin": s.Autologin,
		},
		"look":    map[string]any{"theme": s.Theme, "profiles": profiles},
		"network": map[string]any{"offline": s.Offline},
		"hardware": map[string]any{"gpu": s.GPU},
	}
	return json.MarshalIndent(doc, "", "  ")
}

// ── Validation ───────────────────────────────────────────────────────

var (
	hostnameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	usernameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	reserved   = map[string]bool{"root": true, "daemon": true, "bin": true, "sys": true, "nobody": true, "nixbld": true,
		"maxor": true, "admin": true, "guest": true, "shutdown": true, "halt": true}
)

// ValidateHostname checks a machine name the way the engine will.
func ValidateHostname(h string) error {
	switch {
	case h == "":
		return errors.New("the machine needs a name")
	case h == "localhost":
		return errors.New("localhost cannot be a machine name")
	case !hostnameRe.MatchString(h):
		return errors.New("use lowercase letters, digits and dashes, not starting or ending with a dash (up to 63 characters)")
	}
	return nil
}

// ValidateUsername checks an account name the way the engine will.
func ValidateUsername(u string) error {
	switch {
	case u == "":
		return errors.New("the account needs a name")
	case reserved[u]:
		return fmt.Errorf("%q is a reserved name", u)
	case !usernameRe.MatchString(u):
		return errors.New("start with a lowercase letter and use lowercase letters, digits, - and _ (up to 32 characters)")
	}
	return nil
}

// ValidateFullname rejects what would break /etc/passwd.
func ValidateFullname(n string) error {
	if len([]rune(n)) > 64 {
		return errors.New("up to 64 characters")
	}
	if strings.ContainsAny(n, ":\n\r") {
		return errors.New("a name cannot contain a colon or a line break")
	}
	return nil
}

// ValidatePassword checks the account password and its confirmation.
func ValidatePassword(pw, confirm string) error {
	switch {
	case len(pw) < 8:
		return errors.New("use at least 8 characters")
	case pw != confirm:
		return errors.New("the two passwords differ")
	case strings.ContainsAny(pw, "\n\r"):
		return errors.New("a password cannot contain a line break")
	}
	return nil
}

// ValidatePassphrase checks the disk encryption passphrase and its confirmation. It is stricter than the
// account password: it is the only protection of a stolen disk.
func ValidatePassphrase(pw, confirm string) error {
	switch {
	case len(pw) < 8:
		return errors.New("use at least 8 characters")
	case pw != confirm:
		return errors.New("the two passphrases differ")
	case Strength(pw) < 2:
		return errors.New("too easy to guess: mix words, digits or symbols, or make it longer")
	}
	return nil
}

// Strength rates a password from 0 (very weak) to 4 (strong). It is a hint for the screen, not a policy.
func Strength(pw string) int {
	if len(pw) < 6 {
		return 0
	}
	var lower, upper, digit, other bool
	for _, r := range pw {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		default:
			other = true
		}
	}
	classes := 0
	for _, b := range []bool{lower, upper, digit, other} {
		if b {
			classes++
		}
	}
	score := 0
	switch {
	case len(pw) >= 16:
		score += 2
	case len(pw) >= 12:
		score += 1
	}
	if classes >= 3 {
		score++
	}
	if classes == 4 || len(pw) >= 20 {
		score++
	}
	if distinct(pw) < 5 {
		score = 0 // «aaaaaaaa» is not strong however long it is
	}
	if score > 4 {
		score = 4
	}
	return score
}

func distinct(s string) int {
	seen := map[rune]bool{}
	for _, r := range s {
		seen[r] = true
	}
	return len(seen)
}

// ValidateRegion checks that a region can hold the installation (at least min GiB).
func ValidateRegion(r Region, minGiB int64) error {
	if r.End <= r.Start {
		return errors.New("the region is empty")
	}
	if (r.End-r.Start+1)*512 < minGiB<<30 {
		return fmt.Errorf("the free region is smaller than %d GiB", minGiB)
	}
	return nil
}
