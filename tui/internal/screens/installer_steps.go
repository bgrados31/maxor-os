package screens

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bgrados31/maxor-os/tui/internal/core"
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
	lines[0] = ui.Line{L: append([]ui.Seg{ui.S(p.Mu, ui.G.Find+"  ")}, pk.in.Segs(p, true, width-6)...)}
	if len(items) == 0 {
		return append(lines, muted(env, "Nothing matches. Keep typing, or delete a letter."))
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
		lines = append(lines, muted(env, fmt.Sprintf("   %d more: type to narrow the list", hidden)))
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
func (*welcomeStep) Title() string { return "Welcome" }
func (*welcomeStep) Intro() string {
	return "A few questions, and nothing changes until the end."
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
	return runTask(env, "welcome.net", "Looking at the network", true, func(ctx context.Context) (any, error) {
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
		}
		return true, nil
	}
	s.pk.handle(k, len(items), pickRows)
	return false, nil
}

func (s *welcomeStep) Gate(w *Installer) string {
	if !s.sys.UEFI {
		return "This machine did not start in UEFI mode, and legacy BIOS is not supported yet. Restart it in UEFI mode."
	}
	if s.sys.RAMBytes > 0 && s.sys.RAMBytes < 1500*1000*1000 {
		return "Maxor OS needs at least 2 GB of memory to install."
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
	net := "Checking the network…"
	netOK := true
	if s.got {
		switch {
		case s.net.Online:
			net, netOK = "Connected to the internet ("+s.net.Name+")", true
		default:
			net, netOK = "Not connected: you will be able to connect in a moment", false
		}
	}
	lines := []ui.Line{
		tick(s.sys.UEFI, "UEFI firmware"),
		tick(s.sys.RAMBytes == 0 || s.sys.RAMBytes >= 1500*1000*1000, fmt.Sprintf("%.1f GB of memory", float64(s.sys.RAMBytes)/1e9)),
		tick(netOK, net),
		gap(), heading(env, "System language"),
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
func (*keyboardStep) Title() string { return "Keyboard" }
func (*keyboardStep) Intro() string { return "Pick your layout, then type in the box to check that the keys are where you expect." }

func (s *keyboardStep) filtered() []install.Layout {
	return install.Filter(install.Layouts, s.pk.in.Text(), func(l install.Layout) string { return l.Name + " " + l.XKB })
}

func (s *keyboardStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pk = newPicker("layout")
	s.test.Placeholder = "try your keys here, accents included"
	s.onTest = false
	cur := install.FindLayout(w.st.XKBLayout, w.st.XKBVariant)
	for i, l := range install.Layouts {
		if l == cur {
			s.pk.list.sel = i
		}
	}
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
	return append(lines, gap(), field(env, "Try it", &s.test, s.onTest, width), muted(env, "tab switches between the list and the test box"))
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
func (*networkStep) Title() string { return "Network" }
func (*networkStep) Intro() string {
	return "The installer can download updates. It also works without a network."
}

func (s *networkStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pw = ui.Input{Mask: true, Placeholder: "Wi-Fi password"}
	s.askPW, s.joining, s.msg = false, false, ""
	return tea.Batch(s.refresh(env), s.scan(env))
}

func (s *networkStep) refresh(env *core.Env) tea.Cmd {
	return runTask(env, "network.status", "Looking at the network", true, func(ctx context.Context) (any, error) {
		return env.Install.Net.Status(ctx)
	})
}

func (s *networkStep) scan(env *core.Env) tea.Cmd {
	s.apsDone = false
	return runTask(env, "network.scan", "Looking for Wi-Fi networks", false, func(ctx context.Context) (any, error) {
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
	return runTask(env, "network.join", "Joining "+ap.SSID, false, func(ctx context.Context) (any, error) {
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
		return "Join a network, or choose «Install without a network»."
	}
	return ""
}

func (s *networkStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	var lines []ui.Line
	switch {
	case !s.loaded:
		lines = append(lines, muted(env, "Looking at the network…"))
	case s.status.Online:
		lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+"  "), ui.S(p.Text, "Connected through "+s.status.Name)))
	default:
		lines = append(lines, ui.Of(ui.S(p.Warn, ui.G.Warn+"  "), ui.S(p.Text, "No internet connection")))
	}
	lines = append(lines, gap(), heading(env, "Wi-Fi"))
	if l, ok := working(env, "install.network.scan", "Looking for Wi-Fi networks"); ok {
		lines = append(lines, l)
	} else if s.apsDone && len(s.aps) == 0 {
		lines = append(lines, muted(env, "No Wi-Fi networks found (a wired connection works without any setup)."))
	}
	from, to := s.list.window(len(s.aps)+1, pickRows)
	for i := from; i < to; i++ {
		sel := i == s.list.sel
		if i == len(s.aps) {
			lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Ac, pad(sel)), ui.S(p.Text, "Install without a network")}, R: []ui.Seg{ui.S(p.Mu, "offline ")}, Sel: sel})
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
			name += "  (connected)"
		}
		lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Ac, pad(sel)), ui.S(p.Text, name)}, R: []ui.Seg{ui.S(p.Mu, right)}, Sel: sel})
	}
	if s.askPW && s.list.sel < len(s.aps) {
		lines = append(lines, gap(), field(env, "Password for "+s.aps[s.list.sel].SSID, &s.pw, true, width))
	}
	if l, ok := working(env, "install.network.join", "Joining the network"); ok {
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
	return append(lines, gap(), muted(env, "r looks again"))
}

// ── 4 · Region ───────────────────────────────────────────────────────

type regionStep struct {
	stepBase
	pk    picker
	zones []string
}

func (*regionStep) ID() string    { return "region" }
func (*regionStep) Title() string { return "Time zone" }
func (*regionStep) Intro() string { return "Where is this computer? It sets the clock. Type a city or a region." }

func (s *regionStep) filtered() []string {
	return install.Filter(s.zones, strings.ReplaceAll(s.pk.in.Text(), " ", "_"), func(z string) string { return z })
}

func (s *regionStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pk = newPicker("city, e.g. Lima or Madrid")
	if len(s.zones) > 0 {
		s.select_(w)
		return nil
	}
	return runTask(env, "region.zones", "Reading the time zones", true, func(ctx context.Context) (any, error) {
		return env.Install.Zones(ctx), nil
	})
}

func (s *regionStep) select_(w *Installer) {
	for i, z := range s.zones {
		if z == w.st.Timezone {
			s.pk.list.sel = i
			s.pk.list.top = max(0, i-pickRows/2)
		}
	}
}

func (s *regionStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID == "install.region.zones" {
		s.zones, _ = d.Value.([]string)
		s.select_(w)
	}
	return nil
}

func (s *regionStep) Captures() bool { return true }

func (s *regionStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	items := s.filtered()
	if k.String() == "enter" {
		if s.pk.list.sel >= 0 && s.pk.list.sel < len(items) {
			w.st.Timezone = items[s.pk.list.sel]
		}
		return true, nil
	}
	s.pk.handle(k, len(items), pickRows)
	return false, nil
}

func (s *regionStep) Gate(w *Installer) string {
	items := s.filtered()
	if len(s.zones) > 0 && (s.pk.list.sel < 0 || s.pk.list.sel >= len(items)) {
		return "Choose a time zone from the list."
	}
	return ""
}

func (s *regionStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	if len(s.zones) == 0 {
		return []ui.Line{muted(env, "Reading the time zones…")}
	}
	return listRows(env, &s.pk, s.filtered(), func(z string) string { return z }, nil, width)
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
func (*diskStep) Title() string { return "Disk" }
func (*diskStep) Intro() string { return "Where should Maxor OS go? Nothing is written yet." }

func (s *diskStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.loaded, s.err = false, nil
	return runTask(env, "disk.probe", "Looking at the disks", false, func(ctx context.Context) (any, error) {
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
		s.list.move(-1, len(s.disks), pickRows)
	case "down", "j":
		s.list.move(1, len(s.disks), pickRows)
	case "r":
		return false, s.Enter(w, env)
	case "enter":
		if s.list.sel < 0 || s.list.sel >= len(s.disks) {
			return false, nil
		}
		d := s.disks[s.list.sel]
		if !usable(d) {
			w.notice = "This disk cannot be used: " + firstProblem(d)
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
		return "Looking at the disks…"
	}
	if s.list.sel < 0 || s.list.sel >= len(s.disks) {
		return "No disk to install on. Plug one in and press r."
	}
	return ""
}

func (s *diskStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	if l, ok := working(env, "install.disk.probe", "Looking at the disks"); ok {
		return []ui.Line{l}
	}
	if s.err != nil {
		return append(failed(env, "Could not list the disks", s.err), gap(), muted(env, "r tries again"))
	}
	if len(s.disks) == 0 {
		return []ui.Line{muted(env, "No disks found. Plug one in and press r.")}
	}
	var lines []ui.Line
	from, to := s.list.window(len(s.disks), pickRows)
	for i := from; i < to; i++ {
		d := s.disks[i]
		sel := i == s.list.sel
		ok := usable(d)
		st := p.Text
		if !ok {
			st = p.Mu
		}
		lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Ac, pad(sel)), ui.S(st, d.Label())}, Sel: sel})
		detail := "   " + d.Contents()
		if !ok {
			detail = "   " + firstProblem(d)
		}
		lines = append(lines, ui.Of(ui.S(p.Mu, detail)))
	}
	return append(lines, gap(), muted(env, "r looks again. The disk this installer started from is never offered."))
}

// ── 6 · Strategy ─────────────────────────────────────────────────────

type strategyStep struct {
	stepBase
	opts []string
	sel  int
}

func (*strategyStep) ID() string    { return "strategy" }
func (*strategyStep) Title() string { return "How to install" }
func (*strategyStep) Intro() string { return "Use the whole disk, or share it with what is already there." }

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
		return "This disk cannot take Maxor OS. Go back and choose another."
	}
	return ""
}

func (s *strategyStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	d := s.disk(w)
	if d == nil || len(s.opts) == 0 {
		return []ui.Line{muted(env, "This disk cannot take Maxor OS. Go back and choose another.")}
	}
	var lines []ui.Line
	for i, o := range s.opts {
		sel := i == s.sel
		switch o {
		case "whole":
			lines = append(lines, radio(env, w.st.Strategy == o, sel, "Erase the disk and install Maxor OS", ""),
				ui.T(p.Warn, "      Everything on "+d.Path+" will be deleted."))
		case "alongside":
			f := d.LargestFree()
			size := ""
			if f != nil {
				size = install.HumanSize(f.Sectors * 512)
			}
			lines = append(lines, radio(env, w.st.Strategy == o, sel, "Install alongside what is already there", size+" free"),
				ui.T(p.Mu, "      Only the free space is used: nothing that exists is changed or resized."))
		}
		lines = append(lines, gap())
	}
	if d.Windows {
		lines = append(lines, muted(env, "Windows was found. To make room, shrink its partition from Windows first."))
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
func (*storageStep) Title() string { return "Storage" }
func (*storageStep) Intro() string { return "How the disk is organised, whether it is encrypted, and what happens when memory runs low." }

func (s *storageStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.pass = ui.Input{Mask: true, Placeholder: "passphrase"}
	s.confirm = ui.Input{Mask: true, Placeholder: "again"}
	s.gib = ui.Input{Placeholder: "GiB"}
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
			return "Encryption passphrase: " + err.Error()
		}
		w.st.Passphrase = s.pass.Text()
	} else {
		w.st.Passphrase = ""
	}
	if w.st.SwapKind == "file" && (w.st.SwapGiB < 1 || w.st.SwapGiB > 128) {
		return "A swap file needs between 1 and 128 GiB."
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
	section := func(t string) { lines = append(lines, gap(), heading(env, t)) }
	section("Filesystem")
	lines = append(lines,
		radio(env, w.st.Filesystem == "btrfs", row == rFS && w.st.Filesystem == "btrfs", "btrfs", "snapshots, compression · recommended"),
		radio(env, w.st.Filesystem == "ext4", row == rFS && w.st.Filesystem == "ext4", "ext4", "plain and proven"))
	section("Encryption")
	lines = append(lines, radio(env, w.st.Encrypt, row == rEnc, "Encrypt the system disk (LUKS2)", "protects your data if the computer is lost"))
	if w.st.Encrypt {
		lines = append(lines, field(env, "Passphrase", &s.pass, row == rPass, width))
		if s.pass.Text() != "" {
			lines = append(lines, ui.Of(ui.S(p.Mu, fmt.Sprintf("%-18s", "")), meter(env, install.Strength(s.pass.Text()))))
		}
		lines = append(lines, field(env, "Again", &s.confirm, row == rConf, width),
			ui.T(p.Warn, "      There is no recovery if you forget it."))
	}
	section("When memory runs low")
	lines = append(lines,
		radio(env, w.st.SwapKind == "zram", row == rSwap && w.st.SwapKind == "zram", "Compressed memory (zram)", "recommended"),
		radio(env, w.st.SwapKind == "file", row == rSwap && w.st.SwapKind == "file", "A swap file", "also lets the computer hibernate"),
		radio(env, w.st.SwapKind == "none", row == rSwap && w.st.SwapKind == "none", "Nothing", ""))
	if w.st.SwapKind == "file" {
		lines = append(lines, field(env, "Size (GiB)", &s.gib, row == rGiB, width))
	}
	return lines
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
func (*accountStep) Title() string { return "Your account" }
func (*accountStep) Intro() string { return "Who will use this computer?" }

func (s *accountStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.full = ui.Input{Placeholder: "your name"}
	s.user = ui.Input{Placeholder: "login name"}
	s.host = ui.Input{Placeholder: "computer name"}
	s.pass = ui.Input{Mask: true, Placeholder: "password"}
	s.conf = ui.Input{Mask: true, Placeholder: "again"}
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
		return "Name: " + err.Error()
	}
	if err := install.ValidateUsername(s.user.Text()); err != nil {
		return "Login name: " + err.Error()
	}
	if err := install.ValidateHostname(s.host.Text()); err != nil {
		return "Computer name: " + err.Error()
	}
	if err := install.ValidatePassword(s.pass.Text(), s.conf.Text()); err != nil {
		return "Password: " + err.Error()
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
		field(env, "Your name", &s.full, s.focus == aFull, width),
		field(env, "Login name", &s.user, s.focus == aUser, width),
		field(env, "Computer name", &s.host, s.focus == aHost, width),
		gap(),
		field(env, "Password", &s.pass, s.focus == aPass, width),
	}
	if s.pass.Text() != "" {
		lines = append(lines, ui.Of(ui.S(p.Mu, fmt.Sprintf("%-18s", "")), meter(env, install.Strength(s.pass.Text()))))
	}
	lines = append(lines, field(env, "Again", &s.conf, s.focus == aConf, width), gap(),
		radio(env, w.st.Autologin, s.focus == aAuto, "Sign in automatically", "no password at the login screen"))
	return lines
}

// ── 9 · Look ─────────────────────────────────────────────────────────

type lookStep struct {
	stepBase
	themes listState
	profs  listState
	onProf bool
	chosen map[string]bool
}

func (*lookStep) ID() string    { return "look" }
func (*lookStep) Title() string { return "Look and tools" }
func (*lookStep) Intro() string { return "Pick a theme (the screen shows it as you move) and what you will use the computer for." }

func (s *lookStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.chosen = map[string]bool{}
	for _, p := range w.st.Profiles {
		s.chosen[p] = true
	}
	s.onProf = false
	var cmds []tea.Cmd
	if !env.Data.ThemesLoaded {
		cmds = append(cmds, LoadThemes(env, false))
	}
	if len(env.Data.Profiles) == 0 {
		cmds = append(cmds, LoadProfiles(env, false))
	}
	for i, t := range ordered(env) {
		if t.ID == w.st.Theme {
			s.themes.sel = i
		}
	}
	return tea.Batch(cmds...)
}

func (s *lookStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID == "data.themes" {
		for i, t := range ordered(env) {
			if t.ID == w.st.Theme {
				s.themes.sel = i
			}
		}
		return s.preview(env)
	}
	return nil
}

func (s *lookStep) current(env *core.Env) string {
	l := ordered(env)
	if s.themes.sel >= 0 && s.themes.sel < len(l) {
		return l[s.themes.sel].ID
	}
	return ""
}

func (s *lookStep) preview(env *core.Env) tea.Cmd {
	id := s.current(env)
	return func() tea.Msg { return core.PreviewThemeMsg{ID: id} }
}

func (s *lookStep) Captures() bool { return true }

func (s *lookStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	profs := env.Data.Profiles
	switch k.String() {
	case "tab":
		s.onProf = !s.onProf
	case "up", "down":
		d := 1
		if k.String() == "up" {
			d = -1
		}
		if s.onProf {
			s.profs.move(d, len(profs), pickRows)
		} else {
			s.themes.move(d, len(ordered(env)), pickRows)
			return false, s.preview(env)
		}
	case " ":
		if s.onProf && s.profs.sel < len(profs) {
			id := profs[s.profs.sel].ID
			s.chosen[id] = !s.chosen[id]
		}
	case "enter":
		if id := s.current(env); id != "" {
			w.st.Theme = id
		}
		w.st.Profiles = w.st.Profiles[:0]
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
	themes := ordered(env)
	if len(themes) == 0 {
		if err := env.Data.Err["themes"]; err != nil {
			lines := []ui.Line{ui.Of(ui.S(p.Warn, ui.G.Warn+"  "), ui.S(p.Text, "The themes could not be loaded; Maxor OS will use Sakura."))}
			for _, l := range ui.Wrap(oneLine(err.Error()), width-3) {
				lines = append(lines, muted(env, "   "+l))
			}
			return append(lines, gap(), muted(env, "You can pick or create another theme once Maxor OS is installed."))
		}
		return []ui.Line{muted(env, "Loading the themes…")}
	}
	lines := []ui.Line{heading(env, "Theme")}
	from, to := s.themes.window(len(themes), 6)
	for i := from; i < to; i++ {
		t := themes[i]
		sel := i == s.themes.sel && !s.onProf
		dot := p.Fill.Foreground(lipgloss.Color(t.Colors.Ac)).Bold(true)
		lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Ac, pad(i == s.themes.sel)), ui.S(dot, ui.G.Swatch+" "), ui.S(p.Text, t.Name)}, R: []ui.Seg{ui.S(p.Mu, t.Mode+" ")}, Sel: sel})
	}
	lines = append(lines, gap(), heading(env, "What will you use it for?  (space to choose, tab to switch)"))
	for i, pr := range env.Data.Profiles {
		sel := s.onProf && i == s.profs.sel
		lines = append(lines, radio(env, s.chosen[pr.ID], sel, pr.Title, strings.Join(pr.Includes, ", ")))
	}
	return lines
}

// ── 10 · Hardware ────────────────────────────────────────────────────

type hardwareStep struct{ stepBase }

func (*hardwareStep) ID() string    { return "hardware" }
func (*hardwareStep) Title() string { return "Your hardware" }
func (*hardwareStep) Intro() string { return "This is what was found. Maxor OS picks the drivers for it." }

func (*hardwareStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	if env.Data.Hardware == nil {
		return LoadHardware(env, false)
	}
	return nil
}

func (*hardwareStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	hw := env.Data.Hardware
	if hw == nil {
		if l, ok := working(env, "data.hardware", "Detecting your hardware"); ok {
			return []ui.Line{l}
		}
		return []ui.Line{muted(env, "Could not detect the hardware. The installer will try again while installing.")}
	}
	kv := func(k, v string) ui.Line {
		return ui.Of(ui.S(p.Mu, fmt.Sprintf("%-14s", k)), ui.S(p.Text, v))
	}
	kind := "desktop"
	if hw.Laptop {
		kind = "laptop"
	}
	if hw.Virt != "" && hw.Virt != "none" {
		kind = "virtual machine (" + hw.Virt + ")"
	}
	lines := []ui.Line{kv("Processor", hw.CPU.Model), kv("Type", kind)}
	var vendors []string
	for _, g := range hw.GPUs {
		vendors = append(vendors, g.Vendor)
		lines = append(lines, kv("Graphics", g.Vendor+" "+g.ID))
	}
	if len(hw.GPUs) == 0 {
		lines = append(lines, kv("Graphics", "none detected"))
	}
	lines = append(lines, gap(), heading(env, "What Maxor OS will set up"))
	has := func(v string) bool {
		for _, x := range vendors {
			if x == v {
				return true
			}
		}
		return false
	}
	switch {
	case has("nvidia") && (has("intel") || has("amd")):
		lines = append(lines, plain(env, "Hybrid graphics: the integrated GPU runs the desktop and the NVIDIA GPU"), plain(env, "is there on demand (PRIME offload). If the NVIDIA driver cannot be fetched,"), plain(env, "the installed system falls back to the integrated GPU alone."))
	case has("nvidia"):
		lines = append(lines, plain(env, "The NVIDIA driver, fetched while installing."))
	case has("amd"):
		lines = append(lines, plain(env, "The open-source AMD graphics stack."))
	case has("intel"):
		lines = append(lines, plain(env, "The Intel graphics stack with video acceleration."))
	case hw.Virt != "" && hw.Virt != "none":
		lines = append(lines, plain(env, "The guest tools of the virtual machine."))
	default:
		lines = append(lines, plain(env, "Generic graphics drivers."))
	}
	if hw.Laptop {
		lines = append(lines, plain(env, "Power profiles and laptop settings."))
	}
	if hw.Bluetooth {
		lines = append(lines, plain(env, "Bluetooth."))
	}
	return lines
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
func (*summaryStep) Title() string { return "Review" }
func (*summaryStep) Intro() string { return "This is exactly what will happen. Nothing has been written yet." }

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
	return runTask(env, "summary.prepare", "Checking everything", false, func(ctx context.Context) (any, error) {
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
	switch k.String() {
	case "enter":
		return true, nil
	case "ctrl+p":
		s.showPlan = !s.showPlan
		if s.showPlan && s.planText == "" {
			st := w.st
			st.Confirmed = requiredWord(st.Strategy)
			return false, runTask(env, "summary.plan", "Writing the plan", false, func(ctx context.Context) (any, error) {
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
		return "Still checking…"
	case s.prep.err != nil:
		return "Could not prepare the installation: " + oneLine(s.prep.err.Error())
	case s.prep.problem != nil:
		return "The installer does not accept these choices: " + oneLine(s.prep.problem.Error())
	case s.confirm.Text() != requiredWord(w.st.Strategy):
		return "Type " + requiredWord(w.st.Strategy) + " to confirm."
	}
	w.st.Confirmed = s.confirm.Text()
	return ""
}

// diskBar draws the disk as a proportional bar: what exists, and what Maxor OS will take.
func diskBar(env *core.Env, d install.Disk, st install.State, width int) []ui.Line {
	p := env.P
	type seg struct {
		label string
		size  int64
		new   bool
	}
	var segs []seg
	if st.Strategy == "whole" {
		segs = []seg{{"Maxor OS", d.Size, true}}
	} else {
		for _, pt := range d.Partitions {
			name := pt.Label
			if name == "" {
				name = pt.FSType
			}
			segs = append(segs, seg{name, pt.Size, false})
		}
		used := (st.Region.End - st.Region.Start + 1) * 512
		segs = append(segs, seg{"Maxor OS", used, true})
	}
	var total int64
	for _, s := range segs {
		total += s.size
	}
	if total == 0 {
		return nil
	}
	var bar []ui.Seg
	var legend []ui.Seg
	left := width
	for i, s := range segs {
		n := int(float64(s.size) / float64(total) * float64(width))
		if n < 1 {
			n = 1
		}
		if i == len(segs)-1 || n > left {
			n = max(left, 1)
		}
		left -= n
		st := p.Mu
		if s.new {
			st = p.Ac.Bold(true)
		}
		bar = append(bar, ui.S(st, strings.Repeat("█", n)))
		legend = append(legend, ui.S(st, ui.G.Swatch+" "), ui.S(p.Text, s.label+" "+install.HumanSize(s.size)+"   "))
	}
	return []ui.Line{ui.Of(bar...), ui.Of(legend...)}
}

func (s *summaryStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	if s.showPlan {
		var lines []ui.Line
		if l, ok := working(env, "install.summary.plan", "Writing the plan"); ok {
			return []ui.Line{l}
		}
		for _, l := range strings.Split(strings.TrimSpace(s.planText), "\n") {
			l = strings.TrimPrefix(l, "DRYRUN: ")
			lines = append(lines, muted(env, l))
		}
		return append(lines, gap(), muted(env, "ctrl+p goes back to the summary"))
	}
	kv := func(k, v string) ui.Line { return ui.Of(ui.S(p.Mu, fmt.Sprintf("%-14s", k)), ui.S(p.Text, v)) }
	var lines []ui.Line
	if d, ok := w.currentDisk(); ok {
		lines = append(lines, kv("Disk", d.Label()))
		lines = append(lines, diskBar(env, d, w.st, min(width-2, 64))...)
		lines = append(lines, gap())
	}
	how := "Erase the whole disk"
	if w.st.Strategy == "alongside" {
		how = "Alongside the existing system (only free space is used)"
	}
	enc := "not encrypted"
	if w.st.Encrypt {
		enc = "encrypted (LUKS2)"
	}
	swap := map[string]string{"zram": "compressed memory", "file": fmt.Sprintf("%d GiB swap file", w.st.SwapGiB), "none": "no swap"}[w.st.SwapKind]
	lines = append(lines,
		kv("Method", how),
		kv("Storage", w.st.Filesystem+", "+enc+", "+swap),
		kv("Account", fmt.Sprintf("%s (%s) on %s", w.st.Username, w.st.Fullname, w.st.Hostname)),
		kv("Region", w.st.Timezone+" · "+w.st.Locale),
		kv("Keyboard", install.FindLayout(w.st.XKBLayout, w.st.XKBVariant).Name),
		kv("Look", w.st.Theme+profilesText(w.st.Profiles)),
		kv("Network", map[bool]string{true: "none: installing without internet", false: "online"}[w.st.Offline]))
	if d, ok := w.currentDisk(); ok && w.st.Strategy == "alongside" && d.Windows {
		lines = append(lines, gap(), ui.Of(ui.S(p.Ok, ui.G.Tick+"  "), ui.S(p.Text, "Windows stays untouched and keeps its place in the boot menu.")))
	}
	lines = append(lines, gap())
	if l, ok := working(env, "install.summary.prepare", "Checking everything"); ok {
		return append(lines, l)
	}
	word := requiredWord(w.st.Strategy)
	if w.st.Strategy == "whole" {
		lines = append(lines, ui.T(p.Warn, "Everything on the disk will be erased."))
	}
	lines = append(lines, field(env, "Type "+word, &s.confirm, true, width))
	if s.prep.planHash != "" {
		lines = append(lines, muted(env, "plan "+s.prep.planHash[:12]))
	}
	return append(lines, muted(env, "ctrl+p shows the exact steps"))
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
func (*installStep) Title() string { return "Installing" }
func (*installStep) Intro() string { return "Please do not turn the computer off." }

func (s *installStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	if !s.resume {
		w.mu.Lock()
		w.events = nil
		w.mu.Unlock()
	}
	w.running, w.failed, w.notice = true, false, ""
	st := w.st
	resume := s.resume
	s.resume = false
	return runTask(env, "install.run", "Installing Maxor OS", true, func(ctx context.Context) (any, error) {
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
		msg := fmt.Sprintf("The installer stopped (exit code %d).", res.code)
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
		state[e.Stage] = e.State
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
		lines = append(lines, ui.Of(mark, ui.S(txt, st.Title)))
	}
	n := int(progress * 36)
	lines = append(lines, gap(), ui.Of(ui.S(p.Ac, strings.Repeat(ui.G.BarOn, n)), ui.S(p.Mu, strings.Repeat(ui.G.BarOff, 36-n)), ui.S(p.Mu, fmt.Sprintf("  %d%%", int(progress*100)))))
	if w.failed {
		lines = append(lines, gap(), ui.Of(ui.S(p.Bad, ui.G.Bad+"  "), ui.S(p.Text, "The installation did not finish.")),
			muted(env, "Nothing is lost: r tries again from where it stopped, esc goes back."),
			muted(env, "The log is /var/log/maxor-install.log"))
	}
	return lines
}

func (s *installStep) Hints() []ui.Hint { return nil }

// ── 13 · Done ────────────────────────────────────────────────────────

type doneStep struct {
	stepBase
	restarting bool
}

func (*doneStep) ID() string    { return "done" }
func (*doneStep) Title() string { return "All done" }
func (*doneStep) Intro() string { return "Maxor OS is installed." }

func (s *doneStep) Captures() bool { return false }

func (s *doneStep) Enter(w *Installer, env *core.Env) tea.Cmd {
	s.restarting = false
	return core.Note("ok", "Installed Maxor OS on "+w.st.Disk)
}

func (s *doneStep) Key(w *Installer, env *core.Env, k tea.KeyMsg) (bool, tea.Cmd) {
	if k.String() == "enter" && !s.restarting {
		s.restarting = true
		return false, runTask(env, "done.reboot", "Restarting", false, func(ctx context.Context) (any, error) {
			return nil, env.Install.Reboot()
		})
	}
	return false, nil
}

func (s *doneStep) Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd {
	if d.ID == "install.done.reboot" && d.Err != nil {
		s.restarting = false
		w.notice = "Could not restart: " + oneLine(d.Err.Error())
	}
	return nil
}

func (s *doneStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	p := env.P
	lines := []ui.Line{
		ui.Of(ui.S(p.Ok.Bold(true), ui.G.Tick+"  Installed on "+w.st.Disk)),
		gap(),
		plain(env, "1.  Remove the installation medium (the USB stick)."),
		plain(env, "2.  Restart, and sign in as "+w.st.Username+"."),
		gap(),
		muted(env, "Your configuration is in ~/nixos-config (a git repository): it is yours to change."),
		muted(env, "The installation log is /var/log/maxor-install.log on the new system."),
		gap(),
	}
	if s.restarting {
		return append(lines, ui.Of(ui.S(p.Ac, ui.Spin(env.Frame)+"  "), ui.S(p.Text, "Restarting…")))
	}
	return append(lines, ui.Of(button(env, true, "Restart now  ⏎")), gap(), muted(env, "or press q to stay in this session"))
}
