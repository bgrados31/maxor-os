package screens

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // the zone database inside the program: the review shows the time of any zone

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/i18n"
	"github.com/bgrados31/maxor-os/tui/internal/install"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

const pickRows = 8

// stepBase gives every step the default behaviour, so a step only writes what is its own.
type stepBase struct{}

func (stepBase) Enter(*Installer, *core.Env) tea.Cmd                 { return nil }
func (stepBase) Done(*Installer, *core.Env, task.DoneMsg) tea.Cmd    { return nil }
func (stepBase) Gate(*Installer) string                              { return "" }
func (stepBase) Captures() bool                                      { return false }
// Key: a step that asks nothing moves on with Enter.
func (stepBase) Key(_ *Installer, _ *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	return k.String() == "enter", nil
}

// listRows draws the visible window of a picker: the filter, the rows and what is hidden above or below.
func listRows[T any](env *core.Env, pk *picker, items []T, text func(T) string, extra func(T) string, width int) []ui.Line {
	p := env.P
	lines := []ui.Line{ui.Of(ui.S(p.Mu, ui.G.Find+"  "), ui.Seg{T: ""}), gap()}
	lines[0] = ui.Line{L: append([]ui.Seg{ui.S(p.Mu, ui.G.Find+"  ")}, pk.in.Segs(p, !pk.blurred, width-6)...)}
	if len(items) == 0 {
		return append(lines, muted(env, tr("Nothing matches. Keep typing, or delete a letter.")))
	}
	from, to := pk.list.window(len(items), pickRows)
	for i := from; i < to; i++ {
		segs := []ui.Seg{ui.S(p.Ac, pad(i == pk.list.sel)), ui.S(p.Text, text(items[i]))}
		var right []ui.Seg
		if extra != nil {
			right = []ui.Seg{ui.S(p.Mu, extra(items[i])+" ")}
		}
		lines = append(lines, ui.Line{L: segs, R: right, Sel: i == pk.list.sel})
	}
	if hidden := len(items) - (to - from); hidden > 0 {
		lines = append(lines, muted(env, fmt.Sprintf(tr("   %d more: type to narrow the list"), hidden)))
	}
	return lines
}

// ── 1 · Welcome ──────────────────────────────────────────────────────

type welcomeStep struct {
	stepBase
	pk  picker
	sys install.SysInfo
	net install.NetStatus
	got bool
}

func (*welcomeStep) ID() string    { return "welcome" }
func (*welcomeStep) Title() string { return tr("Language") }
func (*welcomeStep) Intro() string {
	return tr("A few questions, and nothing changes until the end.")
}

func (s *welcomeStep) filtered() []install.Locale {
	return install.Filter(install.Locales, s.pk.in.Text(), func(l install.Locale) string { return l.Name + " " + l.Code })
}

func (s *welcomeStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pk = newPicker("language")
	s.sys = env.Install.Sys()
	for i, l := range install.Locales {
		if l.Code == w.st.Locale {
			s.pk.list.sel = i
		}
	}
	return runTask(env, "welcome.net", tr("Looking at the network"), true, func(ctx context.Context) (any, error) {
		return env.Install.Net.Status(ctx)
	})
}

func (s *welcomeStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID == "install.welcome.net" {
		s.net, _ = d.Value.(install.NetStatus)
		s.got = true
	}
	return nil
}

func (s *welcomeStep) Captures() bool { return true }

func (s *welcomeStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	items := s.filtered()
	if k.String() == "enter" {
		if s.pk.list.sel >= 0 && s.pk.list.sel < len(items) {
			w.st.Locale = items[s.pk.list.sel].Code
			i18n.Set(w.st.Locale) // from the next screen on the installer speaks the language that was chosen
			w.suggest()
		}
		return true, nil
	}
	s.pk.handle(k, len(items), pickRows)
	return false, nil
}

func (s *welcomeStep) Gate(w *Installer) string {
	if !s.sys.UEFI {
		return tr("This machine did not start in UEFI mode, and legacy BIOS is not supported yet. Restart it in UEFI mode.")
	}
	if s.sys.RAMBytes > 0 && s.sys.RAMBytes < 1500*1000*1000 {
		return tr("Maxor OS needs at least 2 GB of memory to install.")
	}
	return ""
}

func (s *welcomeStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	tick := func(ok bool, text string) ui.Line {
		if ok {
			return ui.Of(ui.S(p.Ok, ui.G.Tick+"  "), ui.S(p.Text, text))
		}
		return ui.Of(ui.S(p.Bad, ui.G.Bad+"  "), ui.S(p.Text, text))
	}
	// the network is not a requirement (Maxor OS installs without one): not having it is said plainly, not as a fault
	netLine := ui.Of(ui.S(p.Mu, ui.G.Info+"  "), ui.S(p.Mu, tr("Looking for a network…")))
	if s.got {
		if s.net.Online {
			netLine = tick(true, tr("Connected to the internet (%s)", s.net.Name))
		} else {
			netLine = ui.Of(ui.S(p.Mu, ui.G.Info+"  "), ui.S(p.Text, tr("No network yet: connect in a moment, or install without one")))
		}
	}
	lines := []ui.Line{
		tick(s.sys.UEFI, tr("UEFI firmware")),
		tick(s.sys.RAMBytes == 0 || s.sys.RAMBytes >= 1500*1000*1000, fmt.Sprintf(tr("%.1f GB of memory"), float64(s.sys.RAMBytes)/1e9)),
		netLine,
		gap(), heading(env, tr("System language")),
	}
	return append(lines, listRows(env, &s.pk, s.filtered(), func(l install.Locale) string { return l.Name }, func(l install.Locale) string { return l.Code }, width)...)
}

// ── 2 · Keyboard ─────────────────────────────────────────────────────

type keyboardStep struct {
	stepBase
	pk      picker
	test    ui.Input
	onTest  bool
	applied string
}

func (*keyboardStep) ID() string    { return "keyboard" }
func (*keyboardStep) Title() string { return tr("Keyboard") }
func (*keyboardStep) Intro() string { return tr("Pick your layout, then type in the box to check that the keys are where you expect.") }

func (s *keyboardStep) filtered() []install.Layout {
	return install.Filter(install.Layouts, s.pk.in.Text(), func(l install.Layout) string { return l.Name + " " + l.XKB })
}

func (s *keyboardStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pk = newPicker("layout")
	s.test.Placeholder = tr("try your keys here, accents included")
	s.onTest, s.pk.blurred = false, false
	cur := install.FindLayout(w.st.XKBLayout, w.st.XKBVariant)
	for i, l := range install.Layouts {
		if l == cur {
			s.pk.list.sel = i
			s.pk.list.top = max(0, i-pickRows/2)
		}
	}
	// the layout shown is also the one the keys type with, from the start (it may be the one the language proposed)
	s.apply(w, env)
	return nil
}

func (s *keyboardStep) Captures() bool { return true }

func (s *keyboardStep) apply(w *Installer, env *core.Env) {
	items := s.filtered()
	if s.pk.list.sel < 0 || s.pk.list.sel >= len(items) {
		return
	}
	l := items[s.pk.list.sel]
	w.st.Keymap, w.st.XKBLayout, w.st.XKBVariant = l.Console, l.XKB, l.Variant
	if key := l.XKB + "/" + l.Variant; key != s.applied {
		s.applied = key
		if env.Install.ApplyKeyboard != nil {
			env.Install.ApplyKeyboard(l)
		}
	}
}

func (s *keyboardStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "tab":
		s.onTest = !s.onTest
		s.pk.blurred = s.onTest
		return false, nil
	case "enter":
		s.apply(w, env)
		return true, nil
	}
	if s.onTest {
		s.test.Key(k)
		return false, nil
	}
	if s.pk.handle(k, len(s.filtered()), pickRows) {
		s.apply(w, env)
	}
	return false, nil
}

func (s *keyboardStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	lines := listRows(env, &s.pk, s.filtered(), func(l install.Layout) string { return l.Name }, func(l install.Layout) string { return l.XKB }, width)
	return append(lines, gap(), field(env, tr("Try it"), &s.test, s.onTest, width), muted(env, tr("tab switches between the list and the test box")))
}

// ── 3 · Network ──────────────────────────────────────────────────────

type networkStep struct {
	stepBase
	status  install.NetStatus
	loaded  bool
	aps     []install.AP
	apsDone bool
	list    listState
	pw      ui.Input
	askPW   bool
	joining bool
	msg     string
}

func (*networkStep) ID() string    { return "network" }
func (*networkStep) Title() string { return tr("Network") }
func (*networkStep) Intro() string {
	return tr("The installer can download updates. It also works without a network.")
}

func (s *networkStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pw = ui.Input{Mask: true, Placeholder: tr("Wi-Fi password")}
	s.askPW, s.joining, s.msg = false, false, ""
	return tea.Batch(s.refresh(env), s.scan(env))
}

func (s *networkStep) refresh(env *core.Env) tea.Cmd {
	return runTask(env, "network.status", tr("Looking at the network"), true, func(ctx context.Context) (any, error) {
		return env.Install.Net.Status(ctx)
	})
}

func (s *networkStep) scan(env *core.Env) tea.Cmd {
	s.apsDone = false
	return runTask(env, "network.scan", tr("Looking for Wi-Fi networks"), false, func(ctx context.Context) (any, error) {
		return env.Install.Net.Scan(ctx)
	})
}

func (s *networkStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	switch d.ID {
	case "install.network.status":
		if st, ok := d.Value.(install.NetStatus); ok {
			s.status, s.loaded = st, true
			if st.Online {
				w.st.Offline = false
			}
		}
	case "install.network.scan":
		s.apsDone = true
		if aps, ok := d.Value.([]install.AP); ok {
			s.aps = aps
		}
	case "install.network.join":
		s.joining = false
		if d.Err != nil {
			s.msg = d.Err.Error()
			return nil
		}
		s.askPW, s.msg = false, ""
		s.pw.Set("")
		return s.refresh(env)
	}
	return nil
}

func (s *networkStep) Captures() bool { return true }

func (s *networkStep) join(env *core.Env, ap install.AP, password string) tea.Cmd {
	s.joining, s.msg = true, ""
	return runTask(env, "network.join", tr("Joining %s", ap.SSID), false, func(ctx context.Context) (any, error) {
		return nil, env.Install.Net.Connect(ctx, ap.SSID, password)
	})
}

func (s *networkStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	if s.joining {
		return false, nil
	}
	if s.askPW {
		switch k.String() {
		case "esc":
			s.askPW = false
			return false, nil
		case "enter":
			if s.list.sel < len(s.aps) {
				return false, s.join(env, s.aps[s.list.sel], s.pw.Text())
			}
		}
		s.pw.Key(k)
		return false, nil
	}
	n := len(s.aps) + 1 // the last row is «without a network»
	switch k.String() {
	case "up", "down":
		d := 1
		if k.String() == "up" {
			d = -1
		}
		s.list.move(d, n, pickRows)
	case "r":
		return false, tea.Batch(s.refresh(env), s.scan(env))
	case "enter":
		if s.status.Online && s.list.sel >= len(s.aps) {
			return true, nil // connected and nothing else chosen: carry on
		}
		if s.list.sel >= len(s.aps) {
			w.st.Offline = true
			return true, nil
		}
		ap := s.aps[s.list.sel]
		if ap.InUse {
			return true, nil
		}
		if ap.Open() {
			return false, s.join(env, ap, "")
		}
		s.askPW = true
	}
	return false, nil
}

func (s *networkStep) Gate(w *Installer) string {
	if s.loaded && !s.status.Online && !w.st.Offline {
		return tr("Join a network, or choose «Install without a network».")
	}
	return ""
}

func (s *networkStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	var lines []ui.Line
	switch {
	case !s.loaded:
		lines = append(lines, muted(env, tr("Looking at the network…")))
	case s.status.Online:
		lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+"  "), ui.S(p.Text, tr("Connected through %s", s.status.Name))))
	default:
		lines = append(lines, ui.Of(ui.S(p.Warn, ui.G.Warn+"  "), ui.S(p.Text, tr("No internet connection"))))
	}
	lines = append(lines, gap(), heading(env, "Wi-Fi"))
	if l, ok := working(env, "install.network.scan", tr("Looking for Wi-Fi networks")); ok {
		lines = append(lines, l)
	} else if s.apsDone && len(s.aps) == 0 {
		for _, l := range ui.Wrap(tr("No Wi-Fi networks found (a wired connection works without any setup)."), width) {
			lines = append(lines, muted(env, l))
		}
	}
	from, to := s.list.window(len(s.aps)+1, pickRows)
	for i := from; i < to; i++ {
		sel := i == s.list.sel
		if i == len(s.aps) {
			lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Ac, pad(sel)), ui.S(p.Text, tr("Install without a network"))}, R: []ui.Seg{ui.S(p.Mu, tr("offline "))}, Sel: sel})
			continue
		}
		ap := s.aps[i]
		lock := ui.G.Info
		if !ap.Open() {
			lock = "🔒"
			if ui.G.Info == "-" {
				lock = "*"
			}
		}
		right := fmt.Sprintf("%s  %d%% ", lock, ap.Signal)
		name := ap.SSID
		if ap.InUse {
			name += tr("  (connected)")
		}
		lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Ac, pad(sel)), ui.S(p.Text, name)}, R: []ui.Seg{ui.S(p.Mu, right)}, Sel: sel})
	}
	if s.askPW && s.list.sel < len(s.aps) {
		lines = append(lines, gap(), field(env, tr("Password for %s", s.aps[s.list.sel].SSID), &s.pw, true, width))
	}
	if l, ok := working(env, "install.network.join", tr("Joining the network")); ok {
		lines = append(lines, gap(), l)
	}
	if s.msg != "" {
		lines = append(lines, gap())
		for i, l := range ui.Wrap(oneLine(s.msg), width-3) {
			mark := "   "
			if i == 0 {
				mark = ui.G.Bad + "  "
			}
			lines = append(lines, ui.Of(ui.S(p.Bad, mark), ui.S(p.Text, l)))
		}
	}
	return lines
}

// ── 4 · Region ───────────────────────────────────────────────────────

type regionStep struct {
	stepBase
	pk        picker
	zones     []string
	detecting bool
	msg       string
}

func (*regionStep) ID() string    { return "region" }
func (*regionStep) Title() string { return tr("Time zone") }
func (*regionStep) Intro() string { return tr("Where is this computer? It sets the clock. Type a city or a region.") }

// detectRow stands for «find it from the internet connection»; it is the first row while nothing is typed.
const detectRow = "\x00detect"

func (s *regionStep) canDetect(w *Installer, env *core.Env) bool {
	return env.Install.DetectZone != nil && !w.st.Offline && s.pk.in.Text() == ""
}

func (s *regionStep) filtered(w *Installer, env *core.Env) []string {
	items := install.Filter(s.zones, strings.ReplaceAll(s.pk.in.Text(), " ", "_"), func(z string) string { return z })
	if s.canDetect(w, env) {
		return append([]string{detectRow}, items...)
	}
	return items
}

func (s *regionStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pk = newPicker(tr("city, e.g. Lima or Madrid"))
	s.detecting, s.msg = false, ""
	if len(s.zones) > 0 {
		s.select_(w, env)
		return nil
	}
	return runTask(env, "region.zones", tr("Reading the time zones"), true, func(ctx context.Context) (any, error) {
		return env.Install.Zones(ctx), nil
	})
}

func (s *regionStep) select_(w *Installer, env *core.Env) {
	for i, z := range s.filtered(w, env) {
		if z == w.st.Timezone {
			s.pk.list.sel = i
			s.pk.list.top = max(0, i-pickRows/2)
		}
	}
}

func (s *regionStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	switch d.ID {
	case "install.region.zones":
		s.zones, _ = d.Value.([]string)
		s.select_(w, env)
	case "install.region.detect":
		s.detecting = false
		if d.Err != nil {
			s.msg = oneLine(d.Err.Error())
			return nil
		}
		if z, ok := d.Value.(string); ok {
			w.st.Timezone = z
			return w.advance(env)
		}
	}
	return nil
}

func (s *regionStep) Captures() bool { return true }

func (s *regionStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	if s.detecting {
		return false, nil
	}
	items := s.filtered(w, env)
	if k.String() == "enter" {
		if s.pk.list.sel < 0 || s.pk.list.sel >= len(items) {
			w.notice = tr("No time zone matches: keep typing or delete a letter.")
			return false, nil
		}
		if items[s.pk.list.sel] == detectRow {
			s.detecting, s.msg = true, ""
			zones := s.zones
			return false, runTask(env, "region.detect", tr("Asking the internet for the time zone"), false, func(ctx context.Context) (any, error) {
				return env.Install.DetectZone(ctx, zones)
			})
		}
		w.st.Timezone = items[s.pk.list.sel]
		return true, nil
	}
	s.pk.handle(k, len(items), pickRows)
	return false, nil
}

func (s *regionStep) Gate(w *Installer) string {
	if len(s.zones) > 0 && w.st.Timezone == "" {
		return tr("Choose a time zone from the list.")
	}
	return ""
}

func (s *regionStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	if len(s.zones) == 0 {
		return []ui.Line{muted(env, tr("Reading the time zones…"))}
	}
	text := func(z string) string {
		if z == detectRow {
			return tr("Detect automatically")
		}
		return z
	}
	extra := func(z string) string {
		if z == detectRow {
			return tr("from your internet connection")
		}
		return ""
	}
	lines := listRows(env, &s.pk, s.filtered(w, env), text, extra, width)
	if l, ok := working(env, "install.region.detect", tr("Asking the internet for the time zone")); ok {
		lines = append(lines, gap(), l)
	}
	if s.msg != "" {
		lines = append(lines, gap())
		for i, l := range ui.Wrap(s.msg, width-3) {
			mark := "   "
			if i == 0 {
				mark = ui.G.Bad + "  "
			}
			lines = append(lines, ui.Of(ui.S(env.P.Bad, mark), ui.S(env.P.Text, l)))
		}
	}
	if s.canDetect(w, env) {
		lines = append(lines, gap(), muted(env, tr("Detecting tells ipapi.co your address. Choosing a city sends nothing.")))
	}
	return lines
}

// ── 5 · Disk ─────────────────────────────────────────────────────────

type diskStep struct {
	stepBase
	disks  []install.Disk
	loaded bool
	err    error
	list   listState
}

func (*diskStep) ID() string    { return "disk" }
func (*diskStep) Title() string { return tr("Disk") }
func (*diskStep) Intro() string { return tr("Where should Maxor OS go? Nothing is written yet.") }

func (s *diskStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.loaded, s.err = false, nil
	return runTask(env, "disk.probe", tr("Looking at the disks"), false, func(ctx context.Context) (any, error) {
		return env.Install.Engine.Probe(ctx)
	})
}

func (s *diskStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID != "install.disk.probe" {
		return nil
	}
	s.loaded, s.err = true, d.Err
	if disks, ok := d.Value.([]install.Disk); ok {
		s.disks = disks
		for i, dk := range disks {
			if dk.Path == w.st.Disk {
				s.list.sel = i
			}
		}
	}
	return nil
}

// usable: a disk can take the installation by erasing it or by sharing it.
func usable(d install.Disk) bool {
	return d.Problem("whole") == "" || d.Problem("alongside") == "" && len(d.Partitions) > 0
}

func (s *diskStep) Captures() bool { return false }

func (s *diskStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "up", "k":
		s.list.move(-1, len(s.disks), diskRows)
	case "down", "j":
		s.list.move(1, len(s.disks), diskRows)
	case "r":
		return false, s.Enter(w, env)
	case "enter":
		if s.list.sel < 0 || s.list.sel >= len(s.disks) {
			return false, nil
		}
		d := s.disks[s.list.sel]
		if !usable(d) {
			w.notice = tr("This disk cannot be used: %s", firstProblem(d))
			return false, nil
		}
		if d.Path != w.st.Disk {
			w.st.Disk, w.st.Strategy, w.st.PlanHash, w.st.Confirmed = d.Path, "whole", "", ""
		}
		return true, nil
	}
	return false, nil
}

func firstProblem(d install.Disk) string {
	if p := d.Problem("whole"); p != "" && d.Problem("alongside") != "" {
		if len(d.Partitions) > 0 {
			return d.Problem("alongside")
		}
		return p
	}
	return d.Problem("alongside")
}

func (s *diskStep) Gate(w *Installer) string {
	if !s.loaded {
		return tr("Looking at the disks…")
	}
	if s.list.sel < 0 || s.list.sel >= len(s.disks) {
		return tr("No disk to install on. Plug one in and press r.")
	}
	return ""
}

func (s *diskStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	if l, ok := working(env, "install.disk.probe", tr("Looking at the disks")); ok {
		return []ui.Line{l}
	}
	if s.err != nil {
		return append(failed(env, tr("Could not list the disks"), s.err), gap(), muted(env, tr("r tries again")))
	}
	if len(s.disks) == 0 {
		return []ui.Line{muted(env, tr("No disks found. Plug one in and press r."))}
	}
	// Each disk is a small block: its name and size, what it is and what is on it, and the disk drawn to scale.
	// The one with the focus is tinted as a whole and also names its parts.
	var lines []ui.Line
	from, to := s.list.window(len(s.disks), diskRows)
	for i := from; i < to; i++ {
		d := s.disks[i]
		sel := i == s.list.sel
		ok := usable(d)
		block := diskBlock(env, d, sel, ok, width)
		for k := range block {
			block[k].Sel = sel
		}
		lines = append(lines, block...)
		if i < to-1 {
			lines = append(lines, gap())
		}
	}
	if from > 0 || to < len(s.disks) {
		lines = append(lines, muted(env, fmt.Sprintf(tr("   %d of %d disks · ↑ ↓ for the others"), to-from, len(s.disks))))
	}
	lines = append(lines, gap())
	for _, l := range ui.Wrap(tr("The disk this installer started from is never offered."), width) {
		lines = append(lines, muted(env, l))
	}
	return lines
}

// diskRows is how many disks the list shows at once: each takes four or five rows.
const diskRows = 3

// diskBlock draws one disk of the list.
func diskBlock(env *core.Env, d install.Disk, sel, ok bool, width int) []ui.Line {
	p := env.P
	name := diskName(d)
	nameSt := p.Text.Bold(true)
	if !ok {
		nameSt = p.Mu
	}
	mark := ui.S(p.Ac, "  ")
	if sel {
		mark = ui.S(p.Ac, ui.G.Sel+" ")
	}
	title := ui.Line{L: []ui.Seg{mark, ui.S(nameSt, name)}, R: []ui.Seg{ui.S(p.Text, install.HumanSize(d.Size))}}

	meta := []string{d.Path}
	if t := transportName(d); t != "" {
		meta = append(meta, t)
	}
	meta = append(meta, d.Contents())
	lines := []ui.Line{title, ui.Of(ui.S(p.Mu, "  "+strings.Join(meta, " · ")))}
	if !ok {
		return append(lines, ui.Of(ui.S(p.Warn, "  "+ui.G.Warn+" "), ui.S(p.Mu, tr("cannot be used: %s", firstProblem(d)))))
	}
	spans := d.Layout()
	if !sel {
		return append(lines, ui.Of(append([]ui.Seg{ui.S(p.Mu, "  ")}, diskBar(env, spans, width-4)...)...))
	}
	return append(lines, diskMap(env, spans, width-2, 2)...)
}

// diskName is what a disk is called in the list: its model, or what kind of disk it is when it does not say (a
// virtual machine's disks have no model). The device path goes on the line below, never twice.
func diskName(d install.Disk) string {
	if m := strings.TrimSpace(d.Model); m != "" {
		return m
	}
	dev := strings.TrimPrefix(d.Path, "/dev/")
	switch {
	case strings.HasPrefix(dev, "vd") || strings.HasPrefix(dev, "xvd"):
		return tr("Virtual disk")
	case strings.HasPrefix(dev, "nvme"):
		return tr("NVMe disk")
	case strings.HasPrefix(dev, "mmcblk"):
		return tr("Memory card")
	case d.Removable || d.Transport == "usb":
		return tr("USB drive")
	}
	return tr("Disk")
}

// transportName is how a disk is connected, in the words on the box.
func transportName(d install.Disk) string {
	switch {
	case d.Removable || d.Transport == "usb":
		return tr("USB, removable")
	case d.Transport == "nvme":
		return "NVMe"
	case d.Transport == "sata" || d.Transport == "ata":
		return "SATA"
	case d.Transport == "mmc":
		return tr("SD / eMMC")
	}
	return ""
}

// ── 6 · Strategy ─────────────────────────────────────────────────────

type strategyStep struct {
	stepBase
	opts []string
	sel  int
}

func (*strategyStep) ID() string    { return "strategy" }
func (*strategyStep) Title() string { return tr("How to install") }
func (*strategyStep) Intro() string { return tr("Use the whole disk, or share it with what is already there.") }

func (s *strategyStep) disk(w *Installer) *install.Disk {
	if d, ok := w.currentDisk(); ok {
		return &d
	}
	return nil
}

func (s *strategyStep) options(w *Installer) []string {
	d := s.disk(w)
	if d == nil {
		return nil
	}
	var o []string
	if d.Problem("whole") == "" {
		o = append(o, "whole")
	}
	if d.Problem("alongside") == "" && len(d.Partitions) > 0 {
		o = append(o, "alongside")
	}
	return o
}

// Skip: with a single way to install there is nothing to ask.
func (s *strategyStep) Skip(w *Installer) bool {
	o := s.options(w)
	if len(o) == 1 {
		w.st.Strategy = o[0]
		if o[0] == "alongside" {
			s.setRegion(w)
		}
		return true
	}
	return false
}

func (s *strategyStep) setRegion(w *Installer) {
	if d := s.disk(w); d != nil && d.LargestFree() != nil {
		w.st.Region = install.RegionOf(*d.LargestFree())
	}
}

func (s *strategyStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.opts = s.options(w)
	s.sel = 0
	for i, o := range s.opts {
		if o == w.st.Strategy {
			s.sel = i
		}
	}
	return nil
}

func (s *strategyStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "up", "k":
		s.sel = max(0, s.sel-1)
	case "down", "j":
		s.sel = min(len(s.opts)-1, s.sel+1)
	case "enter":
		if s.sel >= 0 && s.sel < len(s.opts) {
			w.st.Strategy = s.opts[s.sel]
			if w.st.Strategy == "alongside" {
				s.setRegion(w)
			}
			return true, nil
		}
	}
	return false, nil
}

func (s *strategyStep) Gate(w *Installer) string {
	if len(s.opts) == 0 {
		return tr("This disk cannot take Maxor OS. Go back and choose another.")
	}
	return ""
}

func (s *strategyStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	d := s.disk(w)
	if d == nil || len(s.opts) == 0 {
		return []ui.Line{muted(env, tr("This disk cannot take Maxor OS. Go back and choose another."))}
	}
	var lines []ui.Line
	for i, o := range s.opts {
		sel := i == s.sel
		var opt []ui.Line
		switch o {
		case "whole":
			opt = []ui.Line{radio(env, sel, sel, tr("Erase the disk and install Maxor OS"), ""),
				ui.T(p.Warn, "      Everything on "+d.Path+" will be deleted.")}
		case "alongside":
			f := d.LargestFree()
			size := ""
			if f != nil {
				size = install.HumanSize(f.Sectors*512) + " free"
			}
			other := tr("what is already there")
			if d.Windows {
				other = "Windows"
			}
			opt = []ui.Line{radio(env, sel, sel, tr("Install alongside %s", other), size),
				ui.T(p.Mu, tr("      Only the free space is used. Nothing else is touched."))}
		}
		for k := range opt {
			opt[k].Sel = sel
		}
		lines = append(lines, opt...)
		lines = append(lines, gap())
	}
	// the disk now, and as it will be with the option that has the focus
	label := func(t string) ui.Seg { return ui.S(p.Mu, fmt.Sprintf("%-8s", t)) }
	lines = append(lines, ui.Of(append([]ui.Seg{label("now")}, diskBar(env, d.Layout(), width-8)...)...))
	after := d.Planned(s.opts[min(s.sel, len(s.opts)-1)])
	lines = append(lines, ui.Of(append([]ui.Seg{label(tr("after"))}, diskBar(env, after, width-8)...)...))
	for _, l := range diskLegend(env, after, width-8) {
		lines = append(lines, ui.Of(append([]ui.Seg{label("")}, l.L...)...))
	}
	if d.Windows {
		lines = append(lines, gap())
		for _, l := range ui.Wrap(tr("Need more room for Maxor OS? Shrink the Windows partition from Windows first (Disk Management), then come back."), width) {
			lines = append(lines, muted(env, l))
		}
	}
	return lines
}

// ── 7 · Storage ──────────────────────────────────────────────────────

type storageStep struct {
	stepBase
	focus   int
	pass    ui.Input
	confirm ui.Input
	gib     ui.Input
}

const (
	rFS = iota
	rEnc
	rPass
	rConf
	rSwap
	rGiB
)

func (*storageStep) ID() string    { return "storage" }
func (*storageStep) Title() string { return tr("Storage") }
func (*storageStep) Intro() string { return tr("How Maxor OS sits on the disk. ↑ ↓ moves, ← → changes.") }

func (s *storageStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pass = ui.Input{Mask: true, Placeholder: tr("passphrase")}
	s.confirm = ui.Input{Mask: true, Placeholder: tr("again")}
	s.gib = ui.Input{Placeholder: "GiB", Filter: install.DigitRune}
	s.gib.Set(fmt.Sprint(max(w.st.SwapGiB, 4)))
	s.pass.Set(w.st.Passphrase)
	s.confirm.Set(w.st.Passphrase)
	s.focus = 0
	return nil
}

func (s *storageStep) rows(w *Installer) []int {
	r := []int{rFS, rEnc}
	if w.st.Encrypt {
		r = append(r, rPass, rConf)
	}
	r = append(r, rSwap)
	if w.st.SwapKind == "file" {
		r = append(r, rGiB)
	}
	return r
}

func (s *storageStep) row(w *Installer) int {
	r := s.rows(w)
	s.focus = min(max(s.focus, 0), len(r)-1)
	return r[s.focus]
}

func (s *storageStep) Captures() bool { return true }

func cycle(cur string, opts []string, d int) string {
	for i, o := range opts {
		if o == cur {
			return opts[(i+d+len(opts))%len(opts)]
		}
	}
	return opts[0]
}

func (s *storageStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	row := s.row(w)
	rows := s.rows(w)
	switch k.String() {
	case "up":
		s.focus = max(0, s.focus-1)
		return false, nil
	case "down", "tab":
		s.focus = min(len(rows)-1, s.focus+1)
		return false, nil
	case "enter":
		if s.focus < len(rows)-1 {
			s.focus++
			return false, nil
		}
		return true, nil
	}
	switch row {
	case rFS:
		if k.String() == "left" || k.String() == "right" || k.String() == " " {
			w.st.Filesystem = cycle(w.st.Filesystem, []string{"btrfs", "ext4"}, 1)
		}
	case rEnc:
		if k.String() == "left" || k.String() == "right" || k.String() == " " {
			w.st.Encrypt = !w.st.Encrypt
		}
	case rPass:
		s.pass.Key(k)
	case rConf:
		s.confirm.Key(k)
	case rSwap:
		d := 1
		if k.String() == "left" {
			d = -1
		}
		if k.String() == "left" || k.String() == "right" || k.String() == " " {
			w.st.SwapKind = cycle(w.st.SwapKind, []string{"zram", "file", "none"}, d)
			if w.st.SwapKind == "file" && w.st.SwapGiB < 1 {
				w.st.SwapGiB = 4
			}
		}
	case rGiB:
		s.gib.Key(k)
		fmt.Sscanf(s.gib.Text(), "%d", &w.st.SwapGiB)
	}
	return false, nil
}

func (s *storageStep) Gate(w *Installer) string {
	if w.st.Encrypt {
		if err := install.ValidatePassphrase(s.pass.Text(), s.confirm.Text()); err != nil {
			return tr("Encryption passphrase: %s", err.Error())
		}
		w.st.Passphrase = s.pass.Text()
	} else {
		w.st.Passphrase = ""
	}
	if w.st.SwapKind == "file" && (w.st.SwapGiB < 1 || w.st.SwapGiB > 128) {
		return tr("A swap file needs between 1 and 128 GiB.")
	}
	if w.st.SwapKind != "file" {
		w.st.SwapGiB = 0
	}
	return ""
}

func (s *storageStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	row := s.row(w)
	var lines []ui.Line

	// what the disk will hold, with the choices of this step written on the Maxor OS part
	if d, ok := w.currentDisk(); ok {
		spans := d.Planned(w.st.Strategy)
		for i := range spans {
			if spans[i].Kind == "maxor" && spans[i].New {
				what := []string{w.st.Filesystem}
				if w.st.Encrypt {
					what = append(what, tr("encrypted"))
				}
				spans[i].Label = tr("Maxor OS (%s)", strings.Join(what, ", "))
			}
		}
		lines = append(lines, diskMap(env, spans, width, 0)...)
		lines = append(lines, gap())
	}

	lines = append(lines, choice(env, tr("Filesystem"), []string{"btrfs", "ext4"}, []string{"btrfs", "ext4"}, w.st.Filesystem, row == rFS))
	enc := "off"
	if w.st.Encrypt {
		enc = "on"
	}
	lines = append(lines, gap(), choice(env, tr("Encryption"), []string{"off", "on"}, []string{tr("Off"), "On"}, enc, row == rEnc))
	if w.st.Encrypt {
		lines = append(lines, field(env, tr("    Passphrase"), &s.pass, row == rPass, width))
		if s.pass.Text() != "" {
			n := install.Strength(s.pass.Text())
			lines = append(lines, ui.Of(ui.S(p.Mu, strings.Repeat(" ", 18)), meter(env, n), ui.S(p.Mu, "  "+strengthWord(n))))
		}
		again := field(env, tr("    Again"), &s.confirm, row == rConf, width)
		if s.confirm.Text() != "" && s.confirm.Text() == s.pass.Text() {
			again.L = append(again.L, ui.S(p.Ok, "  "+ui.G.Tick+" match"))
		}
		lines = append(lines, again)
	}
	lines = append(lines, gap(), choice(env, tr("Low memory"), []string{"zram", "file", "none"}, []string{"zram", tr("swap file"), tr("none")}, w.st.SwapKind, row == rSwap))
	if w.st.SwapKind == "file" {
		lines = append(lines, field(env, tr("    Size (GiB)"), &s.gib, row == rGiB, width))
	}

	// One place, always the same, explains the row with the focus: the screen stays short and nothing jumps
	// when the focus moves.
	text, st := s.help(w), p.Mu
	if (row == rPass || row == rConf) && w.st.Encrypt {
		st = p.Warn
	}
	lines = append(lines, gap())
	for _, l := range ui.Wrap(text, width) {
		lines = append(lines, ui.T(st, l))
	}
	return lines
}

// help explains the setting that has the focus, in its current value.
func (s *storageStep) help(w *Installer) string {
	switch s.row(w) {
	case rFS:
		if w.st.Filesystem == "ext4" {
			return tr("ext4: the classic Linux filesystem, plain and proven, without snapshots.")
		}
		return tr("btrfs: snapshots to roll back a bad change, and compression. Recommended.")
	case rEnc:
		if w.st.Encrypt {
			return tr("Encrypted with LUKS2: the passphrase is asked every time the computer starts.")
		}
		return tr("Encryption protects your files if the computer is lost or stolen. ← → turns it on.")
	case rPass, rConf:
		return tr("There is no way to recover the passphrase if you forget it. Write it down somewhere safe.")
	case rSwap:
		switch w.st.SwapKind {
		case "file":
			return tr("A swap file on the disk: slower than zram, but it lets the computer hibernate.")
		case "none":
			return tr("Nothing: when memory runs out, programs are closed.")
		}
		return tr("zram: compressed memory, fast and easy on the disk. Recommended.")
	case rGiB:
		return tr("Between 1 and 128 GiB. To hibernate, at least the size of your memory.")
	}
	return ""
}

// optCol is where the options of a choice start, after its label.
const optCol = 16

// choice draws a setting picked with ← →: its label, then the options side by side with the chosen one on the
// accent. With the focus, the label lights up and arrows say it can be changed.
func choice(env *core.Env, label string, ids, names []string, cur string, focused bool) ui.Line {
	p := env.P
	lst := p.Mu
	mark := "  "
	if focused {
		lst, mark = p.Ac.Bold(true), ui.G.Sel+" "
	}
	segs := []ui.Seg{ui.S(p.Ac, mark), ui.S(lst, fmt.Sprintf("%-*s", optCol-2, label))}
	for i, id := range ids {
		if i > 0 {
			segs = append(segs, ui.S(p.Mu, " "))
		}
		if id == cur {
			segs = append(segs, ui.S(p.Btn, " "+names[i]+" "))
		} else {
			segs = append(segs, ui.S(p.Text, " "+names[i]+" "))
		}
	}
	if focused {
		segs = append(segs, ui.S(p.Mu, "  ← →"))
	}
	return ui.Line{L: segs, Sel: focused}
}

// strengthWord says in a word what the strength meter shows.
func strengthWord(n int) string {
	return [...]string{tr("too weak"), tr("weak"), tr("fair"), tr("good"), tr("strong")}[min(max(n, 0), 4)]
}

// ── 8 · Account ──────────────────────────────────────────────────────

type accountStep struct {
	stepBase
	full, user, host, pass, conf ui.Input
	userEdited, hostEdited       bool
	focus                        int
}

const (
	aFull = iota
	aUser
	aHost
	aPass
	aConf
	aAuto
)

func (*accountStep) ID() string    { return "account" }
func (*accountStep) Title() string { return tr("Your account") }
func (*accountStep) Intro() string { return tr("Who will use this computer?") }

func (s *accountStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.full = ui.Input{Placeholder: tr("your name")}
	s.user = ui.Input{Placeholder: tr("login name"), Filter: install.UserRune}
	s.host = ui.Input{Placeholder: tr("computer name"), Filter: install.HostRune}
	s.pass = ui.Input{Mask: true, Placeholder: "password"}
	s.conf = ui.Input{Mask: true, Placeholder: tr("again")}
	s.full.Set(w.st.Fullname)
	s.user.Set(w.st.Username)
	s.host.Set(w.st.Hostname)
	s.pass.Set(w.st.Password)
	s.conf.Set(w.st.Password)
	s.userEdited = w.st.Username != ""
	s.hostEdited = w.st.Hostname != "" && w.st.Hostname != "maxor"
	s.focus = 0
	return nil
}

func (s *accountStep) Captures() bool { return true }

func (s *accountStep) input() *ui.Input {
	switch s.focus {
	case aFull:
		return &s.full
	case aUser:
		return &s.user
	case aHost:
		return &s.host
	case aPass:
		return &s.pass
	case aConf:
		return &s.conf
	}
	return nil
}

func (s *accountStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "up", "shift+tab":
		s.focus = max(0, s.focus-1)
		return false, nil
	case "down", "tab":
		s.focus = min(aAuto, s.focus+1)
		return false, nil
	case "enter":
		if s.focus < aAuto {
			s.focus++
			return false, nil
		}
		return true, nil
	}
	if s.focus == aAuto {
		if k.String() == " " || k.String() == "left" || k.String() == "right" {
			w.st.Autologin = !w.st.Autologin
		}
		return false, nil
	}
	changed, _ := s.input().Key(k)
	if changed {
		switch s.focus {
		case aFull:
			if !s.userEdited {
				s.user.Set(install.UserHint(s.full.Text()))
			}
			if !s.hostEdited {
				s.host.Set(install.HostHint(s.user.Text()))
			}
		case aUser:
			s.userEdited = true
			if !s.hostEdited {
				s.host.Set(install.HostHint(s.user.Text()))
			}
		case aHost:
			s.hostEdited = true
		}
	}
	return false, nil
}

func (s *accountStep) Gate(w *Installer) string {
	if err := install.ValidateFullname(s.full.Text()); err != nil {
		return tr("Name: %s", err.Error())
	}
	if err := install.ValidateUsername(s.user.Text()); err != nil {
		return tr("Login name: %s", err.Error())
	}
	if err := install.ValidateHostname(s.host.Text()); err != nil {
		return tr("Computer name: %s", err.Error())
	}
	if err := install.ValidatePassword(s.pass.Text(), s.conf.Text()); err != nil {
		return tr("Password: %s", err.Error())
	}
	if w.st.Password != s.pass.Text() {
		w.st.PasswordHash = "" // a new password needs a new hash
	}
	w.st.Fullname, w.st.Username, w.st.Hostname, w.st.Password = s.full.Text(), s.user.Text(), s.host.Text(), s.pass.Text()
	return ""
}

func (s *accountStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	lines := []ui.Line{
		field(env, tr("Your name"), &s.full, s.focus == aFull, width),
		field(env, tr("Login name"), &s.user, s.focus == aUser, width),
		field(env, tr("Computer name"), &s.host, s.focus == aHost, width),
		gap(),
		field(env, tr("Password"), &s.pass, s.focus == aPass, width),
	}
	if s.pass.Text() != "" {
		n := install.Strength(s.pass.Text())
		lines = append(lines, ui.Of(ui.S(p.Mu, fmt.Sprintf("%-18s", "")), meter(env, n), ui.S(p.Mu, "  "+strengthWord(n))))
	}
	lines = append(lines, field(env, tr("Again"), &s.conf, s.focus == aConf, width), gap(),
		radio(env, w.st.Autologin, s.focus == aAuto, tr("Sign in automatically"), tr("no password at the login screen")))
	return lines
}

// ── 9 · Look ─────────────────────────────────────────────────────────

// The installer offers the two faces of Maxor OS, light and dark; the other themes are for later, from the Maxor
// app or `maxor theme`, where they can be tried at leisure.
const (
	lookDark  = "maxor-dark"
	lookLight = "maxor-light"
)

type lookStep struct {
	stepBase
	profs  listState
	onProf bool
	chosen map[string]bool
}

func (*lookStep) ID() string    { return "look" }
func (*lookStep) Title() string { return tr("Look and tools") }
func (*lookStep) Intro() string {
	return tr("Light or dark: the screen shows it as you choose. Other themes are in the Maxor app after installing.")
}

func (s *lookStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.chosen = map[string]bool{}
	for _, p := range w.st.Profiles {
		s.chosen[p] = true
	}
	s.onProf = false
	if w.st.Theme != lookLight {
		w.st.Theme = lookDark
	}
	cmds := []tea.Cmd{s.preview(w)}
	if !env.Data.ThemesLoaded {
		cmds = append(cmds, LoadThemes(env, false))
	}
	if len(env.Data.Profiles) == 0 {
		cmds = append(cmds, LoadProfiles(env, false))
	}
	return tea.Batch(cmds...)
}

func (s *lookStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID == "data.themes" {
		return s.preview(w) // the themes arrived: now the preview can find the one chosen
	}
	return nil
}

func (s *lookStep) preview(w *Installer) tea.Cmd {
	id := w.st.Theme
	return func() tea.Msg { return core.PreviewThemeMsg{ID: id} }
}

func (s *lookStep) Captures() bool { return true }

func (s *lookStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	profs := env.Data.Profiles
	if w.st.Offline {
		profs = nil // their packages are not on the medium: there is nothing to choose without a network
		s.onProf = false
	}
	switch k.String() {
	case "tab":
		s.onProf = !s.onProf && len(profs) > 0
	case "left", "right", "h", "l":
		if !s.onProf {
			if w.st.Theme == lookDark {
				w.st.Theme = lookLight
			} else {
				w.st.Theme = lookDark
			}
			return false, s.preview(w)
		}
	case "down", "j":
		if !s.onProf {
			s.onProf = len(profs) > 0
		} else {
			s.profs.move(1, len(profs), pickRows)
		}
	case "up", "k":
		if s.onProf && s.profs.sel == 0 {
			s.onProf = false
		} else if s.onProf {
			s.profs.move(-1, len(profs), pickRows)
		}
	case " ":
		if s.onProf && s.profs.sel < len(profs) {
			id := profs[s.profs.sel].ID
			s.chosen[id] = !s.chosen[id]
		} else if !s.onProf {
			return s.Key(w, env, tea.KeyMsg{Type: tea.KeyRight})
		}
	case "enter":
		w.st.Profiles = w.st.Profiles[:0]
		if w.st.Offline {
			return true, nil // no profiles without a network
		}
		for _, p := range profs {
			if s.chosen[p.ID] {
				w.st.Profiles = append(w.st.Profiles, p.ID)
			}
		}
		return true, nil
	}
	return false, nil
}

func (s *lookStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	lines := []ui.Line{choice(env, tr("Appearance"), []string{lookDark, lookLight}, []string{tr("Dark"), tr("Light")}, w.st.Theme, !s.onProf)}
	// a strip in the colours of the theme being shown: the screen around it already is that theme
	t := env.Theme.P
	var strip []ui.Seg
	for _, c := range []string{t.Ac, t.Ac2, t.S2, t.Fg} {
		strip = append(strip, ui.S(p.Fill.Foreground(lipgloss.Color(c)), ui.G.Swatch+" "))
	}
	what := map[string]string{lookDark: tr("navy and violet"), lookLight: tr("soft white, pink and violet")}[w.st.Theme]
	lines = append(lines, ui.Of(append(append([]ui.Seg{{T: strings.Repeat(" ", optCol)}}, strip...), ui.S(p.Mu, " "+what))...))

	if w.st.Offline {
		lines = append(lines, gap(), heading(env, tr("Tools for what you do")))
		for _, l := range ui.Wrap(tr("Gaming, development and the other profiles download their packages, so they need a network. Add them after installing with `maxor profile`."), width) {
			lines = append(lines, muted(env, l))
		}
		return lines
	}
	lines = append(lines, gap(), heading(env, tr("What will you use it for?  (space to choose)")))
	for i, pr := range env.Data.Profiles {
		sel := s.onProf && i == s.profs.sel
		lines = append(lines, radio(env, s.chosen[pr.ID], sel, pr.Title, strings.Join(pr.Includes, ", ")))
	}
	return lines
}

// ── 10 · Hardware ────────────────────────────────────────────────────

type hardwareStep struct {
	stepBase
	sel int
}

func (*hardwareStep) ID() string    { return "hardware" }
func (*hardwareStep) Title() string { return tr("Your hardware") }
func (*hardwareStep) Intro() string { return tr("This is what was found. Maxor OS picks the drivers for it.") }

// options are the ways to use the graphics hardware: only where there is a real choice (see install.GPUOptions).
func (*hardwareStep) options(env *core.Env) []install.GPUOption {
	hw := env.Data.Hardware
	if hw == nil {
		return nil
	}
	var vendors []string
	for _, g := range hw.GPUs {
		vendors = append(vendors, g.Vendor)
	}
	return install.GPUOptions(vendors, hw.Laptop)
}

func (s *hardwareStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.sel = 0
	if env.Data.Hardware == nil {
		return LoadHardware(env, false)
	}
	s.pick(w, env)
	return nil
}

// pick puts the cursor on what was chosen before, or on the recommendation.
func (s *hardwareStep) pick(w *Installer, env *core.Env) {
	opts := s.options(env)
	for i, o := range opts {
		if o.Mode == w.st.GPU || (w.st.GPU == "auto" && o.Recommended) {
			s.sel = i
		}
	}
}

func (s *hardwareStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID == "data.hardware" {
		s.pick(w, env)
	}
	return nil
}

func (s *hardwareStep) Captures() bool { return false }

func (s *hardwareStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	opts := s.options(env)
	switch k.String() {
	case "up", "k":
		s.sel = max(0, s.sel-1)
	case "down", "j":
		s.sel = min(max(len(opts)-1, 0), s.sel+1)
	case "enter":
		if len(opts) == 0 {
			w.st.GPU = "auto" // one way to go: the drivers are simply picked
		} else {
			w.st.GPU = opts[min(s.sel, len(opts)-1)].Mode
		}
		return true, nil
	}
	return false, nil
}

func (s *hardwareStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	hw := env.Data.Hardware
	if hw == nil {
		if l, ok := working(env, "data.hardware", tr("Detecting your hardware")); ok {
			return []ui.Line{l}
		}
		return []ui.Line{muted(env, tr("Could not detect the hardware. The installer will try again while installing."))}
	}
	kv := func(k, v string) ui.Line {
		return ui.Of(ui.S(p.Mu, fmt.Sprintf("%-12s", k)), ui.S(p.Text, v))
	}
	kind := tr("Desktop")
	if hw.Laptop {
		kind = tr("Laptop")
	}
	if hw.Virt != "" && hw.Virt != "none" {
		kind = tr("Virtual machine (%s)", virtName(hw.Virt))
	}
	lines := []ui.Line{kv(tr("Type"), kind), kv(tr("Processor"), cpuName(hw.CPU.Model))}
	if env.Install != nil && env.Install.Sys != nil {
		if ram := env.Install.Sys().RAMBytes; ram > 0 {
			lines = append(lines, kv(tr("Memory"), fmt.Sprintf(tr("%.0f GB"), float64(ram)/1e9)))
		}
	}
	var vendors []string
	for i, g := range hw.GPUs {
		vendors = append(vendors, g.Vendor)
		label := ""
		if i == 0 {
			label = tr("Graphics")
		}
		lines = append(lines, kv(label, install.GPUName(g.Vendor, g.ID)))
	}
	if len(hw.GPUs) == 0 {
		lines = append(lines, kv(tr("Graphics"), tr("none detected")))
	}
	lines = append(lines, gap())

	if opts := s.options(env); len(opts) > 0 {
		lines = append(lines, heading(env, tr("Which GPU should draw the desktop?")))
		for i, o := range opts {
			hint := ""
			if o.Recommended {
				hint = tr("recommended")
			}
			lines = append(lines, radio(env, i == s.sel, i == s.sel, o.Title, hint))
		}
		cur := opts[min(s.sel, len(opts)-1)]
		lines = append(lines, gap())
		for _, l := range ui.Wrap(cur.Description, width-3) {
			lines = append(lines, muted(env, "   "+l))
		}
		lines = append(lines, gap(), muted(env, tr("You can change this later in your configuration.")))
		return lines
	}

	lines = append(lines, heading(env, tr("What Maxor OS will set up")))
	has := func(v string) bool {
		for _, x := range vendors {
			if x == v {
				return true
			}
		}
		return false
	}
	item := func(text string) { lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+"  "), ui.S(p.Text, text))) }
	switch {
	case has("nvidia"):
		item(tr("The NVIDIA driver, fetched while installing"))
	case has("amd"):
		item(tr("The open-source AMD graphics stack"))
	case has("intel"):
		item(tr("The Intel graphics stack, with video acceleration"))
	case hw.Virt != "" && hw.Virt != "none":
		item(tr("The guest tools of the virtual machine"))
	default:
		item(tr("Generic graphics drivers"))
	}
	if hw.Laptop {
		item(tr("Power profiles and laptop settings"))
	}
	if hw.Bluetooth {
		item("Bluetooth")
	}
	return lines
}

// cpuName tidies the name a processor reports: «13th Gen Intel(R) Core(TM) i5-13500H» → «13th Gen Intel Core
// i5-13500H», «AMD Ryzen 7 5800H with Radeon Graphics» → «AMD Ryzen 7 5800H».
func cpuName(s string) string {
	for _, junk := range []string{"(R)", "(r)", "(TM)", "(tm)", " CPU", tr(" Processor")} {
		s = strings.ReplaceAll(s, junk, "")
	}
	for _, cut := range []string{" @ ", tr(" with ")} {
		if i := strings.Index(s, cut); i > 0 {
			s = s[:i]
		}
	}
	return strings.Join(strings.Fields(s), " ")
}

// ── 11 · Summary ─────────────────────────────────────────────────────

// prepared is what the summary computes before it lets the user confirm: the password hash, the fingerprint
// of the plan and whether the engine accepts the answers.
type prepared struct {
	hash     string
	planHash string
	problem  error
	err      error
}

type summaryStep struct {
	stepBase
	confirm  ui.Input
	ready    bool
	prep     prepared
	planText string
	showPlan bool
}

func (*summaryStep) ID() string    { return "summary" }
func (*summaryStep) Title() string { return tr("Review") }
func (*summaryStep) Intro() string { return tr("This is exactly what will happen. Nothing has been written yet.") }

func requiredWord(strategy string) string {
	if strategy == "alongside" {
		return "INSTALL"
	}
	return "ERASE"
}

func (s *summaryStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.confirm = ui.Input{Placeholder: requiredWord(w.st.Strategy)}
	s.ready, s.prep, s.planText, s.showPlan = false, prepared{}, "", false
	st := w.st // a copy: the task must not touch the wizard's state from another goroutine
	return runTask(env, "summary.prepare", tr("Checking everything"), false, func(ctx context.Context) (any, error) {
		var out prepared
		if st.PasswordHash == "" {
			h, err := env.Install.Engine.HashPassword(ctx, st.Password)
			if err != nil {
				out.err = err
				return out, nil
			}
			st.PasswordHash = h
			out.hash = h
		}
		st.Confirmed = requiredWord(st.Strategy)
		answers, err := st.Answers()
		if err != nil {
			out.err = err
			return out, nil
		}
		if st.Strategy == "alongside" {
			ph, err := env.Install.Engine.PlanHash(ctx, answers)
			if err != nil {
				out.err = err
				return out, nil
			}
			out.planHash, st.PlanHash = ph, ph
			if answers, err = st.Answers(); err != nil {
				out.err = err
				return out, nil
			}
		}
		out.problem = env.Install.Engine.Validate(ctx, answers)
		return out, nil
	})
}

func (s *summaryStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	switch d.ID {
	case "install.summary.prepare":
		out, _ := d.Value.(prepared)
		s.prep, s.ready = out, true
		if out.hash != "" {
			w.st.PasswordHash = out.hash
		}
		if out.planHash != "" {
			w.st.PlanHash = out.planHash
		}
	case "install.summary.plan":
		if t, ok := d.Value.(string); ok {
			s.planText = t
		} else if d.Err != nil {
			s.planText = d.Err.Error()
		}
	}
	return nil
}

func (s *summaryStep) Captures() bool { return true }

func (s *summaryStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	if len(k.Runes) == 1 && k.Runes[0] >= '1' && k.Runes[0] <= '9' {
		return false, w.editStep(env, int(k.Runes[0]-'0'))
	}
	switch k.String() {
	case "enter":
		return true, nil
	case "ctrl+p":
		s.showPlan = !s.showPlan
		if s.showPlan && s.planText == "" {
			st := w.st
			st.Confirmed = requiredWord(st.Strategy)
			return false, runTask(env, "summary.plan", tr("Writing the plan"), false, func(ctx context.Context) (any, error) {
				a, err := st.Answers()
				if err != nil {
					return nil, err
				}
				return env.Install.Engine.Plan(ctx, a)
			})
		}
		return false, nil
	}
	s.confirm.Key(k)
	return false, nil
}

func (s *summaryStep) Gate(w *Installer) string {
	switch {
	case !s.ready:
		return tr("Still checking…")
	case s.prep.err != nil:
		return tr("Could not prepare the installation: %s", oneLine(s.prep.err.Error()))
	case s.prep.problem != nil:
		return tr("The installer does not accept these choices: %s", oneLine(s.prep.problem.Error()))
	case s.confirm.Text() != requiredWord(w.st.Strategy):
		return tr("Type %s to confirm.", requiredWord(w.st.Strategy))
	}
	w.st.Confirmed = s.confirm.Text()
	return ""
}

func (s *summaryStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	if s.showPlan {
		var lines []ui.Line
		if l, ok := working(env, "install.summary.plan", tr("Writing the plan")); ok {
			return []ui.Line{l}
		}
		for _, l := range strings.Split(strings.TrimSpace(s.planText), "\n") {
			l = strings.TrimPrefix(l, "DRYRUN: ")
			lines = append(lines, muted(env, l))
		}
		return append(lines, gap(), muted(env, tr("ctrl+p goes back to the summary")))
	}
	kv := func(k, v string) ui.Line { return ui.Of(ui.S(p.Mu, fmt.Sprintf("%-14s", k)), ui.S(p.Text, v)) }
	var lines []ui.Line
	if d, ok := w.currentDisk(); ok {
		lines = append(lines, kv(tr("Disk"), d.Label()))
		lines = append(lines, diskMap(env, d.Planned(w.st.Strategy), min(width, 64), 0)...)
		lines = append(lines, gap())
	}
	how := tr("Erase the whole disk")
	if w.st.Strategy == "alongside" {
		how = tr("Alongside what is there, in the free space")
	}
	enc := tr("not encrypted")
	if w.st.Encrypt {
		enc = tr("encrypted (LUKS2)")
	}
	swap := map[string]string{"zram": tr("compressed memory"), "file": fmt.Sprintf(tr("%d GiB swap file"), w.st.SwapGiB), "none": tr("no swap")}[w.st.SwapKind]
	lines = append(lines,
		kv(tr("Method"), how),
		kv(tr("Storage"), w.st.Filesystem+", "+enc+", "+swap),
		kv(tr("Account"), fmt.Sprintf(tr("%s (%s) on %s"), w.st.Username, w.st.Fullname, w.st.Hostname)),
		kv(tr("Language"), strings.TrimSpace(install.LocaleName(w.st.Locale))),
		kv(tr("Time zone"), zoneName(w.st.Timezone, env.Now())),
		kv(tr("Keyboard"), install.FindLayout(w.st.XKBLayout, w.st.XKBVariant).Name),
		kv(tr("Look"), lookName(w.st.Theme)+profilesText(w.st.Profiles)),
		kv(tr("Graphics"), graphicsText(env, w.st.GPU)),
		kv(tr("Network"), map[bool]string{true: tr("none: installing without internet"), false: "online"}[w.st.Offline]))
	if d, ok := w.currentDisk(); ok && w.st.Strategy == "alongside" && d.Windows {
		for i, l := range ui.Wrap(tr("Windows stays untouched and keeps its place in the boot menu."), width-3) {
			mark := ui.S(p.Ok, "   ")
			if i == 0 {
				mark = ui.S(p.Ok, ui.G.Tick+"  ")
				lines = append(lines, gap())
			}
			lines = append(lines, ui.Of(mark, ui.S(p.Text, l)))
		}
	}
	lines = append(lines, gap())
	if l, ok := working(env, "install.summary.prepare", tr("Checking everything")); ok {
		return append(lines, l)
	}
	word := requiredWord(w.st.Strategy)
	if w.st.Strategy == "whole" {
		lines = append(lines, ui.T(p.Warn, tr("Everything on the disk will be erased.")))
	}
	lines = append(lines, field(env, tr("Type %s", word), &s.confirm, true, width))
	if w.railShown {
		lines = append(lines, muted(env, tr("1-9 edits that step (the numbers in the list on the left)")))
	} else {
		lines = append(lines, muted(env, tr("esc goes back through the steps to change one")))
	}
	if s.prep.planHash != "" {
		lines = append(lines, muted(env, tr("plan %s", s.prep.planHash[:12])))
	}
	return append(lines, muted(env, tr("ctrl+p shows the exact steps")))
}

func profilesText(p []string) string {
	if len(p) == 0 {
		return ""
	}
	return " · " + strings.Join(p, ", ")
}

// currentDisk is the disk the user chose, as the disk step listed it.
func (w *Installer) currentDisk() (install.Disk, bool) {
	for _, s := range w.steps {
		if ds, ok := s.(*diskStep); ok {
			for _, d := range ds.disks {
				if d.Path == w.st.Disk {
					return d, true
				}
			}
		}
	}
	return install.Disk{}, false
}

// ── 12 · Install ─────────────────────────────────────────────────────

type installResult struct{ code int }

type installStep struct {
	stepBase
	resume bool
}

func (*installStep) ID() string    { return "install" }
func (*installStep) Title() string { return tr("Installing") }
func (*installStep) Intro() string { return tr("Please do not turn the computer off.") }

func (s *installStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	if !s.resume {
		w.mu.Lock()
		w.events, w.logs = nil, nil
		w.mu.Unlock()
	}
	w.running, w.failed, w.notice = true, false, ""
	st := w.st
	resume := s.resume
	s.resume = false
	return runTask(env, "install.run", tr("Installing Maxor OS"), true, func(ctx context.Context) (any, error) {
		answers, err := st.Answers()
		if err != nil {
			return installResult{code: 1}, err
		}
		code, err := env.Install.Engine.Run(ctx, answers, st.Passphrase, resume, w.push)
		return installResult{code: code}, err
	})
}

func (s *installStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID != "install.install.run" {
		return nil
	}
	res, _ := d.Value.(installResult)
	w.running = false
	if d.Err != nil || res.code != 0 {
		w.failed = true
		msg := fmt.Sprintf(tr("The installer stopped (exit code %d)."), res.code)
		if d.Err != nil {
			msg = oneLine(d.Err.Error())
		}
		for _, e := range w.snapshot() {
			if e.State == "fail" && e.Message != "" {
				msg = e.Message
			}
		}
		w.notice = msg
		return nil
	}
	return w.goTo(env, len(w.steps)-1)
}

func (s *installStep) Captures() bool { return true }

func (s *installStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	if w.failed && k.String() == "r" {
		s.resume = true
		return false, s.Enter(w, env)
	}
	return false, nil
}

func (s *installStep) Gate(w *Installer) string { return "" }

func (s *installStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	events := w.snapshot()
	state := map[string]string{}
	var progress float64
	for _, e := range events {
		// a stage that had nothing to do says skip and then, as every stage does, ok: it stays skipped
		if !(e.State == "ok" && state[e.Stage] == "skip") {
			state[e.Stage] = e.State
		}
		if e.Progress != nil {
			progress = *e.Progress
		}
	}
	var lines []ui.Line
	for _, st := range install.Stages {
		var mark ui.Seg
		switch state[st.ID] {
		case "ok":
			mark = ui.S(p.Ok, ui.G.Tick+"  ")
		case "skip":
			mark = ui.S(p.Mu, ui.G.Info+"  ")
		case "fail":
			mark = ui.S(p.Bad, ui.G.Bad+"  ")
		case "start":
			mark = ui.S(p.Ac, ui.Spin(env.Frame)+"  ")
		default:
			mark = ui.S(p.Mu, ui.G.Off+"  ")
		}
		txt := p.Text
		if state[st.ID] == "" || state[st.ID] == "skip" {
			txt = p.Mu
		}
		lines = append(lines, ui.Of(mark, ui.S(txt, tr(st.Title))))
	}
	bar := progressLine(env, env.P, progress, 36, w.running)
	lines = append(lines, gap(), ui.Line{L: append(bar, ui.S(p.Mu, fmt.Sprintf("  %d%%", int(progress*100))))})
	// what the programs are printing, as far as the window has room for: the newest rows, each line whole (an
	// error is often a long line, and its end is the part that says what went wrong)
	if n := min(6, w.bodyH-19); n > 0 {
		var rows []string
		for _, l := range w.tailLogs(n) {
			rows = append(rows, ui.Wrap(ansi.Strip(l), width-2)...)
		}
		lines = append(lines, gap())
		for _, r := range rows[max(len(rows)-n, 0):] {
			lines = append(lines, muted(env, r))
		}
	}
	if w.failed {
		lines = append(lines, gap(), ui.Of(ui.S(p.Bad, ui.G.Bad+"  "), ui.S(p.Text, tr("The installation did not finish."))))
		// what went wrong, in words, when the log shows a cause the installer knows
		if why := failureHint(w.tailLogs(200), w.st.Offline); why != "" {
			for _, l := range ui.Wrap(why, width-3) {
				lines = append(lines, ui.Of(ui.S(p.Mu, "   "), ui.S(p.Text, l)))
			}
		}
		lines = append(lines,
			muted(env, tr("Nothing is lost: r tries again from where it stopped, esc goes back.")),
			muted(env, tr("The log is /var/log/maxor-install.log")))
	}
	return lines
}

func (s *installStep) Hints() []ui.Hint { return nil }

// ── 13 · Done ────────────────────────────────────────────────────────

type doneStep struct {
	stepBase
	sel     int
	working string // what is being done: "restart", "power off", or ""
}

func (*doneStep) ID() string    { return "done" }
func (*doneStep) Title() string { return tr("All done") }
func (*doneStep) Intro() string { return "" }

func (s *doneStep) Captures() bool { return false }

var doneChoices = []struct{ label, hint string }{
	{i18n.Mark("Restart now"), "recommended"},
	{i18n.Mark("Power off"), ""},
	{i18n.Mark("Stay in a text console"), i18n.Mark("look around first")},
}

func (s *doneStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.sel, s.working = 0, ""
	return core.Note("ok", tr("Installed Maxor OS on %s", w.st.Disk))
}

func (s *doneStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	if s.working != "" {
		return false, nil
	}
	switch k.String() {
	case "up", "k":
		s.sel = max(0, s.sel-1)
	case "down", "j":
		s.sel = min(len(doneChoices)-1, s.sel+1)
	case "enter":
		var f func() error
		switch s.sel {
		case 0:
			s.working, f = tr("Restarting"), env.Install.Reboot
		case 1:
			s.working, f = tr("Turning the computer off"), env.Install.PowerOff
		default:
			s.working, f = tr("Opening a text console"), env.Install.Shell
		}
		if f == nil {
			s.working = ""
			return false, nil
		}
		return false, runTask(env, "done.action", s.working, false, func(ctx context.Context) (any, error) { return nil, f() })
	}
	return false, nil
}

func (s *doneStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID == "install.done.action" {
		failed := s.working
		s.working = ""
		if d.Err != nil {
			w.notice = failed + " did not work: " + oneLine(d.Err.Error())
		}
	}
	return nil
}

func (s *doneStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	lines := []ui.Line{
		ui.Of(ui.S(p.Ok.Bold(true), ui.G.Tick+"  Maxor OS is installed on "+w.st.Disk)),
		gap(),
		plain(env, tr("Remove the installation medium (the USB stick), restart,")),
		plain(env, tr("and sign in as %s.", w.st.Username)),
		gap(),
		muted(env, tr("Your configuration is in ~/nixos-config, a git repository:")),
		muted(env, tr("it is yours to change. The log is /var/log/maxor-install.log.")),
		gap(),
	}
	if s.working != "" {
		return append(lines, ui.Of(ui.S(p.Ac, ui.Spin(env.Frame)+"  "), ui.S(p.Text, s.working+"…")))
	}
	for i, c := range doneChoices {
		lines = append(lines, radio(env, i == s.sel, i == s.sel, tr(c.label), tr(c.hint)))
	}
	return lines
}

// virtName is the product a virtual machine runs on, as its makers write it.
func virtName(v string) string {
	if n, ok := map[string]string{"qemu": "QEMU", "kvm": "KVM", "vmware": "VMware", "oracle": "VirtualBox",
		"virtualbox": "VirtualBox", "microsoft": "Hyper-V", "hyperv": "Hyper-V", "xen": "Xen", "parallels": "Parallels"}[v]; ok {
		return n
	}
	return v
}

// lookName is how the review and the list of steps call the chosen look.
func lookName(id string) string {
	switch id {
	case lookDark:
		return tr("Dark")
	case lookLight:
		return tr("Light")
	}
	return id
}

// Keys: r looks again, for networks and for disks (one was plugged in, a Wi-Fi was switched on).
func (*networkStep) Keys() []ui.Hint { return []ui.Hint{{Key: "r", Action: tr("look again")}} }
func (*diskStep) Keys() []ui.Hint    { return []ui.Hint{{Key: "r", Action: tr("look again")}} }

// zoneName is a time zone as a person reads it: «America/Lima» → «Lima (UTC−5, now 17:40)», so the clock it will
// set can be checked at a glance. The zone database is built into the program (time/tzdata).
func zoneName(tz string, now time.Time) string {
	city := tz
	if i := strings.LastIndex(tz, "/"); i >= 0 {
		city = tz[i+1:]
	}
	city = strings.ReplaceAll(city, "_", " ")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return city
	}
	t := now.In(loc)
	_, off := t.Zone()
	sign, h, m := "+", off/3600, (off%3600)/60
	if off < 0 {
		sign, h, m = "−", -off/3600, (-off%3600)/60
	}
	utc := "UTC" + sign + strconv.Itoa(h)
	if m != 0 {
		utc += fmt.Sprintf(":%02d", m)
	}
	if off == 0 {
		utc = "UTC"
	}
	return fmt.Sprintf(tr("%s (%s, now %s)"), city, utc, t.Format("15:04"))
}

// graphicsText says what the graphics will be: the choice made on a machine with two GPUs, or the GPU found and
// that its drivers come with the installation.
func graphicsText(env *core.Env, mode string) string {
	switch mode {
	case "hybrid":
		return tr("hybrid (integrated + NVIDIA on demand)")
	case "nvidia":
		return tr("NVIDIA only")
	case "integrated":
		return tr("integrated only")
	}
	hw := env.Data.Hardware
	if hw == nil || len(hw.GPUs) == 0 {
		return tr("drivers chosen while installing")
	}
	var names []string
	for _, g := range hw.GPUs {
		names = append(names, install.GPUName(g.Vendor, g.ID))
	}
	return strings.Join(names, " + ") + ", with its drivers"
}

// failureHint reads the end of the log for causes the installer knows and says, in words, what happened and what to
// do; "" when it does not recognise the cause (the log path is shown anyway).
func failureHint(logs []string, offline bool) string {
	text := strings.ToLower(strings.Join(logs, "\n"))
	switch {
	case strings.Contains(text, "no space left on device"):
		return tr("The disk ran out of space. Go back and choose a bigger disk or a bigger free region.")
	case offline && (strings.Contains(text, "cannot build") || strings.Contains(text, "unable to download") ||
		strings.Contains(text, "could not resolve host")):
		return tr("This computer needs a package that is not on the installation medium. Connect to a network (esc goes back to the network step), then press r: it will be downloaded.")
	case strings.Contains(text, "could not resolve host") || strings.Contains(text, "unable to download"):
		return tr("The network stopped answering while downloading. Check the connection, then press r to carry on.")
	case strings.Contains(text, "input/output error"):
		return tr("The disk reported a read or write error: it may be failing. Try another disk.")
	}
	return ""
}
