package app

import (
	"fmt"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/i18n"
	"github.com/bgrados31/maxor-os/tui/internal/install"
)

// ── doubles ──────────────────────────────────────────────────────────

type fakeEngine struct {
	mu          sync.Mutex
	disks       []install.Disk
	probeErr    error
	planHash    string
	validateErr error
	runCode     int
	runEvents   []install.Event
	answers     []byte
	passphrase  string
	runs        []bool // the resume flag of each run
	hashed      []string
	planCalls   int
}

func (f *fakeEngine) Probe(context.Context) ([]install.Disk, error) { return f.disks, f.probeErr }
func (f *fakeEngine) HashPassword(_ context.Context, pw string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hashed = append(f.hashed, pw)
	return "$6$salt$HASHED", nil
}
func (f *fakeEngine) PlanHash(context.Context, []byte) (string, error) { return f.planHash, nil }
func (f *fakeEngine) Validate(context.Context, []byte) error           { return f.validateErr }
func (f *fakeEngine) Plan(context.Context, []byte) (string, error) {
	f.mu.Lock()
	f.planCalls++
	f.mu.Unlock()
	return "DRYRUN: wipefs --all /dev/vda\nDRYRUN: mkfs.btrfs /dev/vda2\n", nil
}
func (f *fakeEngine) Run(_ context.Context, answers []byte, passphrase string, resume bool, onEvent func(install.Event)) (int, error) {
	f.mu.Lock()
	f.answers, f.passphrase = answers, passphrase
	f.runs = append(f.runs, resume)
	f.mu.Unlock()
	for _, e := range f.runEvents {
		onEvent(e)
	}
	return f.runCode, nil
}

type fakeNet struct {
	status    install.NetStatus
	aps       []install.AP
	joinErr   error
	joined    []string
	scanCalls int
}

func (n *fakeNet) Status(context.Context) (install.NetStatus, error) { return n.status, nil }
func (n *fakeNet) Scan(context.Context) ([]install.AP, error)        { n.scanCalls++; return n.aps, nil }
func (n *fakeNet) Connect(_ context.Context, ssid, pw string) error {
	n.joined = append(n.joined, ssid+"|"+pw)
	if n.joinErr == nil {
		n.status = install.NetStatus{Online: true, Kind: "wifi", Name: ssid}
	}
	return n.joinErr
}

type env struct {
	detectCalls int
	detected  string
	detectErr error
	eng      *fakeEngine
	net      *fakeNet
	layouts  []string
	rebooted bool
	sys      install.SysInfo
}

func emptyDisk() install.Disk {
	return install.Disk{Path: "/dev/vda", Model: "QEMU HARDDISK", Size: 107_374_182_400}
}

func windowsDisk() install.Disk {
	return install.Disk{Path: "/dev/nvme0n1", Model: "Samsung SSD 980 PRO", Size: 512_110_190_592, Transport: "nvme", Windows: true,
		Partitions: []install.Partition{{Path: "/dev/nvme0n1p1", Start: 2048, Size: 104_857_600, FSType: "vfat", Label: "SYSTEM"}, {Path: "/dev/nvme0n1p2", Start: 206_848, Size: 200_000_000_000, FSType: "ntfs", Label: "Windows"}},
		Free:       []install.Free{{Start: 400_000_000, End: 1_000_000_000, Sectors: 600_000_000}}}
}

func newInstallEnv(disks ...install.Disk) *env {
	e := &env{
		eng: &fakeEngine{disks: disks, runEvents: []install.Event{{Stage: "disk", State: "start"}, {Stage: "disk", State: "ok"}, {Stage: "done", State: "ok"}}},
		net: &fakeNet{status: install.NetStatus{Online: true, Kind: "ethernet", Name: "Wired connection 1"}},
		sys: install.SysInfo{UEFI: true, RAMBytes: 8_000_000_000},
	}
	return e
}

func (e *env) deps() *install.Deps {
	return &install.Deps{
		Engine: e.eng, Net: e.net,
		Sys:           func() install.SysInfo { return e.sys },
		Zones:         func(context.Context) []string { return []string{"America/Lima", "America/Mexico_City", "Europe/Madrid", "UTC"} },
		DetectZone:    func(context.Context, []string) (string, error) { e.detectCalls++; return e.detected, e.detectErr },
		ApplyKeyboard: func(l install.Layout) { e.layouts = append(e.layouts, l.XKB) },
		Reboot:        func() error { e.rebooted = true; return nil },
	}
}

// introModel is the installer at its very first screen.
func introModel(t *testing.T, e *env) *Model {
	t.Helper()
	m, _ := setupWith(t, newCLI(), Options{Screen: "install", Deps: e.deps()})
	return m
}

// installModel is the installer past the introduction, at the first question.
func installModel(t *testing.T, e *env) *Model {
	t.Helper()
	t.Cleanup(func() { i18n.Set("en") }) // choosing a language switches the whole interface: do not leak it to other tests
	m := introModel(t, e)
	send(m, key("enter"))
	return m
}

func enter(m *Model, n int) {
	for i := 0; i < n; i++ {
		send(m, key("enter"))
	}
}

// walk goes through the steps that need no thinking, up to the account.
func walkToAccount(m *Model) {
	send(m, key("enter")) // welcome
	send(m, key("enter")) // keyboard
	send(m, key("enter")) // network
	send(m, key("enter")) // region
	send(m, key("enter")) // disk
	enter(m, 3)           // the three storage rows, then on
}

func fillAccount(m *Model, name, user, host, pw string) {
	typeText(m, name)
	send(m, key("enter"))
	send(m, tea.KeyMsg{Type: tea.KeyCtrlU})
	typeText(m, user)
	send(m, key("enter"))
	send(m, tea.KeyMsg{Type: tea.KeyCtrlU})
	typeText(m, host)
	send(m, key("enter"))
	typeText(m, pw)
	send(m, key("enter"))
	typeText(m, pw)
	send(m, key("enter")) // the confirmation → the sign-in row
}

// ── tests ────────────────────────────────────────────────────────────

func TestInstallerOpensWithoutTabsAtTheWelcome(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	if len(m.screens) != 1 || m.screens[0].ID() != "install" {
		t.Fatal("the installer is a single screen with no tabs")
	}
	out := view(m)
	for _, want := range []string{"M A X O R", "install", "Language", "UEFI firmware", "Connected to the internet", "System language", "English (United States)"} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

func TestWelcomeRefusesAMachineWithoutUEFI(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.sys = install.SysInfo{UEFI: false, RAMBytes: 8_000_000_000}
	m := installModel(t, e)
	send(m, key("enter"))
	out := view(m)
	if !has(out, "did not start in UEFI mode") || !has(out, "System language") {
		t.Fatalf("it must stay on the welcome and say why:\n%s", out)
	}
}

func TestWelcomeChoosesTheLanguage(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	typeText(m, "perú")
	if !has(view(m), "Español (Perú)") {
		t.Fatalf("typing filters the list:\n%s", view(m))
	}
	send(m, key("enter"))
	// the installer speaks the language that was chosen, from the next screen on
	if !has(view(m), "Teclado") || has(view(m), "Keyboard") {
		t.Fatalf("moves on to the keyboard, in Spanish:\n%s", view(m))
	}
}

func TestAnUntranslatedLanguageStaysInEnglish(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	typeText(m, "日本語")
	send(m, key("enter"))
	if !has(view(m), "Keyboard") {
		t.Fatalf("a language without a catalog falls back to English:\n%s", view(m))
	}
}

func TestKeyboardAppliesTheLayoutLiveAndHasATestBox(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	m := installModel(t, e)
	send(m, key("enter"))
	typeText(m, "latin")
	if len(e.layouts) == 0 || e.layouts[len(e.layouts)-1] != "latam" {
		t.Fatalf("the layout is applied as you move: %v", e.layouts)
	}
	send(m, key("tab"))
	typeText(m, "ñandú")
	if !has(view(m), "ñandú") {
		t.Fatalf("the test box shows what you type:\n%s", view(m))
	}
	send(m, key("enter"))
	if !has(view(m), "Network") {
		t.Fatal("moves on to the network")
	}
}

func TestNetworkWhenOnlineLetsYouContinueAndWhenOfflineBlocks(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.net.status = install.NetStatus{}
	e.net.aps = []install.AP{{SSID: "Casa", Signal: 80, Security: "WPA2"}, {SSID: "Cafe", Signal: 60}}
	m := installModel(t, e)
	enter(m, 2) // welcome, keyboard → network
	out := view(m)
	if !has(out, "No internet connection") || !has(out, "Casa") || !has(out, "Install without a network") {
		t.Fatalf("network:\n%s", out)
	}
	// down to «without a network» is the last row: Enter chooses offline
	send(m, key("down"), key("down"), key("enter"))
	if !has(view(m), "Time zone") {
		t.Fatalf("offline continues:\n%s", view(m))
	}
}

func TestNetworkJoinsASecuredWifiWithItsPassword(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.net.status = install.NetStatus{}
	e.net.aps = []install.AP{{SSID: "Casa", Signal: 80, Security: "WPA2"}}
	m := installModel(t, e)
	enter(m, 2)
	send(m, key("enter")) // the first network asks for its password
	if !has(view(m), "Password for Casa") {
		t.Fatalf("it asks for the password:\n%s", view(m))
	}
	typeText(m, "s3cret-wifi")
	if strings.Contains(view(m), "s3cret-wifi") {
		t.Fatal("the password is never drawn")
	}
	send(m, key("enter"))
	if len(e.net.joined) != 1 || e.net.joined[0] != "Casa|s3cret-wifi" {
		t.Fatalf("joined: %v", e.net.joined)
	}
	out := view(m)
	if !has(out, "Connected through Casa") {
		t.Fatalf("it reports the new connection:\n%s", out)
	}
}

func TestNetworkShowsAJoinFailure(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.net.status = install.NetStatus{}
	e.net.aps = []install.AP{{SSID: "Casa", Signal: 80, Security: "WPA2"}}
	e.net.joinErr = errors.New("could not join \"Casa\": Secrets were required")
	m := installModel(t, e)
	enter(m, 2)
	send(m, key("enter"))
	typeText(m, "wrong")
	send(m, key("enter"))
	if !has(view(m), "Secrets were required") {
		t.Fatalf("the failure is shown:\n%s", view(m))
	}
}

func TestRegionFiltersAndChoosesATimeZone(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	enter(m, 3)
	typeText(m, "madrid")
	send(m, key("enter"))
	if !has(view(m), "Disk") {
		t.Fatalf("moves on:\n%s", view(m))
	}
}

func TestDiskListsWhatIsOnEachDiskAndRefusesUnusableOnes(t *testing.T) {
	small := install.Disk{Path: "/dev/sdb", Model: "Tiny", Size: 8_000_000_000}
	mounted := install.Disk{Path: "/dev/sdc", Model: "USB stick", Size: 16_000_000_000, Removable: true, Mounted: true}
	m := installModel(t, newInstallEnv(small, mounted, windowsDisk()))
	enter(m, 4)
	out := view(m)
	for _, want := range []string{"/dev/sdb", "smaller than 32 GiB", "/dev/sdc", "in use", "/dev/nvme0n1", "2 partitions (Windows)"} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	send(m, key("enter")) // the first one is too small
	if !has(view(m), "cannot be used") || !has(view(m), "Disk") {
		t.Fatalf("an unusable disk is refused:\n%s", view(m))
	}
}

func TestAnEmptyDiskSkipsTheStrategyAndAWindowsDiskAsksForIt(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	enter(m, 5)
	// the step is settled on its own: the screen is storage, and the list says how it will install
	if out := view(m); has(out, "Use the whole disk, or share it") || !has(out, "Storage") || !has(out, "erase disk") || !has(out, "/ 11") {
		t.Fatalf("an empty disk has a single way:\n%s", out)
	}
	m = installModel(t, newInstallEnv(windowsDisk()))
	enter(m, 5)
	out := view(m)
	for _, want := range []string{"How to install", "Erase the disk", "alongside Windows", "Shrink the Windows partition", "now", "after"} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

func TestStorageEncryptionNeedsAStrongPassphrase(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	enter(m, 5) // → storage
	send(m, key("down"), key("space"))
	if !has(view(m), "Passphrase") || !has(view(m), "LUKS2") {
		t.Fatalf("encryption shows its fields:\n%s", view(m))
	}
	send(m, key("down"))
	if !has(view(m), "no way to recover") {
		t.Fatalf("the passphrase warns that it cannot be recovered:\n%s", view(m))
	}
	typeText(m, "password")
	send(m, key("down"))
	typeText(m, "password")
	enter(m, 5)
	if !has(view(m), "Encryption passphrase") || !has(view(m), "Storage") {
		t.Fatalf("a weak passphrase does not pass:\n%s", view(m))
	}
}

func TestAccountSuggestsNamesAndValidatesBeforeMovingOn(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	walkToAccount(m)
	if !has(view(m), "Your account") {
		t.Fatalf("account:\n%s", view(m))
	}
	typeText(m, "Ana Pérez")
	out := view(m)
	if !has(out, "ana") || !has(out, "ana-pc") {
		t.Fatalf("the login and computer names are suggested:\n%s", out)
	}
	// a short password is refused
	enter(m, 3)
	typeText(m, "short")
	enter(m, 2)
	send(m, key("enter"))
	if !has(view(m), "at least 8") || !has(view(m), "Your account") {
		t.Fatalf("a short password does not pass:\n%s", view(m))
	}
}

func TestFullInstallEndToEnd(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	m := installModel(t, e)
	walkToAccount(m)
	fillAccount(m, "Ana Pérez", "ana", "portatil", "correct-horse-1")
	send(m, key("enter")) // look
	send(m, key("enter")) // hardware
	if !has(view(m), "Your hardware") {
		t.Fatalf("hardware:\n%s", view(m))
	}
	send(m, key("enter")) // review
	out := view(m)
	for _, want := range []string{"Review", "/dev/vda", "Erase the whole disk", "btrfs", "ana (Ana Pérez) on portatil", "Type ERASE"} {
		if !has(out, want) {
			t.Fatalf("missing %q in the review:\n%s", want, out)
		}
	}
	send(m, key("enter")) // without confirming: nothing starts
	if len(e.eng.runs) != 0 || !has(view(m), "Type ERASE to confirm") {
		t.Fatalf("the install must not start without the typed confirmation:\n%s", view(m))
	}
	typeText(m, "ERASE")
	send(m, key("enter"))
	if len(e.eng.runs) != 1 || e.eng.runs[0] {
		t.Fatalf("the engine runs once, not resumed: %v", e.eng.runs)
	}
	var doc struct {
		Disk struct {
			Device, Strategy, Confirmed string
		}
		User struct {
			Name         string
			PasswordHash string `json:"password_hash"`
		}
		Machine struct{ Hostname string }
	}
	if err := json.Unmarshal(e.eng.answers, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Disk.Device != "/dev/vda" || doc.Disk.Strategy != "whole" || doc.Disk.Confirmed != "ERASE" ||
		doc.User.Name != "ana" || doc.Machine.Hostname != "portatil" || doc.User.PasswordHash != "$6$salt$HASHED" {
		t.Fatalf("answers: %s", e.eng.answers)
	}
	if strings.Contains(string(e.eng.answers), "correct-horse-1") {
		t.Fatalf("the plain password reached the engine: %s", e.eng.answers)
	}
	out = view(m)
	if !has(out, "Installed on /dev/vda") || !has(out, "Restart now") {
		t.Fatalf("done:\n%s", out)
	}
	if !strings.Contains(summaryText(m), "Installed Maxor OS on /dev/vda") {
		t.Fatalf("summary: %q", summaryText(m))
	}
	send(m, key("enter"))
	if !e.rebooted {
		t.Fatal("Enter on the last step restarts the machine")
	}
}

func TestAlongsideAsksForINSTALLAndCarriesTheRegionAndTheHash(t *testing.T) {
	e := newInstallEnv(windowsDisk())
	e.eng.planHash = strings.Repeat("ab", 32)
	m := installModel(t, e)
	enter(m, 5)           // → strategy
	send(m, key("down"))  // alongside
	send(m, key("enter")) // → storage
	enter(m, 3)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	enter(m, 3) // look, hardware → review
	out := view(m)
	if !has(out, "Type INSTALL") || !has(out, "Windows stays untouched") || !has(out, "plan abababababab") {
		t.Fatalf("review:\n%s", out)
	}
	typeText(m, "INSTALL")
	send(m, key("enter"))
	if len(e.eng.runs) != 1 {
		t.Fatalf("it should be running: %v\n%s", e.eng.runs, view(m))
	}
	for _, want := range []string{`"strategy": "alongside"`, `"start": 400000000`, `"end": 1000000000`, `"plan_hash": "` + e.eng.planHash + `"`, `"confirmed": "INSTALL"`} {
		if !strings.Contains(string(e.eng.answers), want) {
			t.Fatalf("missing %s in\n%s", want, e.eng.answers)
		}
	}
}

func TestTheEngineRefusingTheAnswersStopsTheReview(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.eng.validateErr = errors.New("answers.machine.hostname: has an invalid format")
	m := installModel(t, e)
	walkToAccount(m)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	enter(m, 3)
	typeText(m, "ERASE")
	send(m, key("enter"))
	if len(e.eng.runs) != 0 || !has(view(m), "does not accept these choices") || !has(view(m), "hostname") {
		t.Fatalf("the engine's objection is shown and nothing starts:\n%s", view(m))
	}
}

func TestAFailedInstallCanBeRetriedFromWhereItStopped(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.eng.runCode = 1
	e.eng.runEvents = []install.Event{{Stage: "disk", State: "ok"}, {Stage: "install", State: "fail", Message: "the build failed"}}
	m := installModel(t, e)
	walkToAccount(m)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	enter(m, 3)
	typeText(m, "ERASE")
	send(m, key("enter"))
	out := view(m)
	if !has(out, "did not finish") || !has(out, "the build failed") || !has(out, "maxor-install.log") {
		t.Fatalf("the failure is explained:\n%s", out)
	}
	e.eng.runCode = 0
	e.eng.runEvents = []install.Event{{Stage: "done", State: "ok"}}
	send(m, key("r"))
	if len(e.eng.runs) != 2 || !e.eng.runs[1] {
		t.Fatalf("the retry resumes: %v", e.eng.runs)
	}
	if !has(view(m), "Installed on /dev/vda") {
		t.Fatalf("it finishes after the retry:\n%s", view(m))
	}
}

func TestWhileInstallingYouCannotLeave(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	m := installModel(t, e)
	w := m.screens[0].(interface{ Blocking() bool })
	if w.Blocking() {
		t.Fatal("not blocked before the install")
	}
	send(m, key("q"), key("ctrl+c"))
	if !m.quitting {
		t.Fatal("before installing, ctrl+c leaves")
	}
}

func TestBackGoesToThePreviousStepAndRemembersTheChoices(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	send(m, key("enter"))
	typeText(m, "latin")
	send(m, key("enter")) // → network
	send(m, key("esc"))   // ← keyboard
	out := view(m)
	if !has(out, "Keyboard") || !has(out, "Spanish (Latin America)") {
		t.Fatalf("going back keeps what was chosen:\n%s", out)
	}
	send(m, key("esc"))
	if !has(view(m), "System language") {
		t.Fatalf("one step back is the language:\n%s", view(m))
	}
	send(m, key("esc"), key("esc"))
	if !has(view(m), "Let's install Maxor OS") {
		t.Fatal("the first step is the introduction, and esc there stays put")
	}
}

func TestEveryStepFitsInASmallWindow(t *testing.T) {
	e := newInstallEnv(windowsDisk())
	m := installModel(t, e)
	for i := 0; i < 12; i++ {
		out := view(m)
		lines := strings.Split(out, "\n")
		if len(lines) != m.h {
			t.Fatalf("step %d: %d rows in a %d-row window", i, len(lines), m.h)
		}
		send(m, key("ctrl+n"))
	}
}

// Translations are longer than English: no step of the installer may overflow the window in any language.
func TestEveryStepFitsInASmallWindowInEveryLanguage(t *testing.T) {
	for _, locale := range []string{"es_PE", "pt_BR", "fr_FR", "de_DE", "it_IT"} {
		t.Run(locale, func(t *testing.T) {
			e := newInstallEnv(windowsDisk())
			m := installModel(t, e)
			typeText(m, locale)
			send(m, key("enter"))
			if i18n.Code() == "en" {
				t.Fatalf("%s did not switch the interface", locale)
			}
			for i := 0; i < 12; i++ {
				out := view(m)
				if lines := strings.Split(out, "\n"); len(lines) != m.h {
					t.Fatalf("step %d: %d rows in a %d-row window:\n%s", i, len(lines), m.h, out)
				}
				send(m, key("ctrl+n"))
			}
		})
	}
}

func TestTheIntroSaysNothingIsWrittenUntilConfirmedAndHasNoStepList(t *testing.T) {
	m := introModel(t, newInstallEnv(emptyDisk()))
	out := view(m)
	for _, want := range []string{"M A X O R", "Let's install Maxor OS", "no demo", "until you confirm"} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if has(out, "install steps") {
		t.Fatalf("the introduction has no list of steps:\n%s", out)
	}
	send(m, key("esc"))
	if !has(view(m), "Let's install") {
		t.Fatal("esc on the introduction stays put")
	}
	send(m, key("enter"))
	if !has(view(m), "System language") {
		t.Fatalf("Enter goes on to the first question:\n%s", view(m))
	}
}

func TestTheStepListShowsWhatWasChosen(t *testing.T) {
	m := installModel(t, newInstallEnv(emptyDisk()))
	send(m, key("enter")) // language
	typeText(m, "latin")
	send(m, key("enter")) // keyboard
	out := view(m)
	for _, want := range []string{"M A X O R", "✓", "English", "Keyboard", "/ "} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if !has(out, "3 / ") {
		t.Fatalf("the counter follows the step:\n%s", out)
	}
}

func TestTheFinalOffersRestartPowerOffAndAConsole(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	var off, shell bool
	deps := e.deps()
	deps.PowerOff = func() error { off = true; return nil }
	deps.Shell = func() error { shell = true; return nil }
	m, _ := setupWith(t, newCLI(), Options{Screen: "install", Deps: deps})
	send(m, key("enter"))
	walkToAccount(m)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	enter(m, 3)
	typeText(m, "ERASE")
	send(m, key("enter"))
	out := view(m)
	for _, want := range []string{"Restart now", "Power off", "Stay in a text console"} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	send(m, key("down"), key("enter"))
	if !off || e.rebooted {
		t.Fatalf("the second choice powers off (off=%v rebooted=%v)", off, e.rebooted)
	}
	send(m, key("up"), key("down"), key("down"), key("enter"))
	if !shell {
		t.Fatal("the third choice opens a text console")
	}
}

func TestRegionOffersToDetectTheZoneFromTheInternetAndUsesTheAnswer(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.detected = "Europe/Madrid"
	m := installModel(t, e)
	enter(m, 3) // → time zone
	out := view(m)
	if !has(out, "Detect automatically") || !has(out, "Detecting tells ipapi.co your address") {
		t.Fatalf("the first row detects, and says what it sends:\n%s", out)
	}
	send(m, key("home"))
	send(m, key("enter"))
	if !has(view(m), "Disk") {
		t.Fatalf("a detected zone moves on:\n%s", view(m))
	}
	if e.detectCalls != 1 {
		t.Fatalf("the service is asked exactly once, and only because the row was chosen: %d", e.detectCalls)
	}
}

func TestRegionShowsWhyDetectionFailedAndStays(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.detectErr = errors.New("could not reach the time zone service")
	m := installModel(t, e)
	enter(m, 3)
	send(m, key("home"), key("enter"))
	out := view(m)
	if !has(out, "could not reach the time zone service") || !has(out, "Time zone") {
		t.Fatalf("a failure is explained and the step stays:\n%s", out)
	}
}

func TestRegionHasNoDetectRowWhenOfflineOrWhenFiltering(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	m := installModel(t, e)
	enter(m, 3)
	typeText(m, "lima")
	if has(view(m), "Detect automatically") {
		t.Fatal("typing a city hides the detect row")
	}
	e2 := newInstallEnv(emptyDisk())
	e2.net.status = install.NetStatus{}
	m2 := installModel(t, e2)
	enter(m2, 2)
	send(m2, key("enter")) // «Install without a network»
	if has(view(m2), "Detect automatically") {
		t.Fatalf("offline cannot detect:\n%s", view(m2))
	}
}

func hardwareScreen(t *testing.T) (*Model, *env) {
	t.Helper()
	e := newInstallEnv(emptyDisk())
	m := installModel(t, e)
	walkToAccount(m)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	enter(m, 2) // account → look → hardware
	return m, e
}

func TestHardwareStepOffersTheGPUChoiceAndRecommendsHybridOnThisLaptop(t *testing.T) {
	m, _ := hardwareScreen(t)
	out := view(m)
	for _, want := range []string{"Which GPU should draw the desktop?", "Hybrid", "recommended", "NVIDIA only", "Integrated only", "PRIME offload"} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	send(m, key("down"))
	if !has(view(m), "Everything runs on the NVIDIA GPU") {
		t.Fatalf("the description follows the cursor:\n%s", view(m))
	}
}

func TestTheChosenGPUGoesToTheEngineAndShowsInTheReview(t *testing.T) {
	m, e := hardwareScreen(t)
	send(m, key("down"), key("down"), key("enter")) // Integrated only → review
	if !has(view(m), "integrated only") {
		t.Fatalf("the review says what was chosen:\n%s", view(m))
	}
	typeText(m, "ERASE")
	send(m, key("enter"))
	if !strings.Contains(string(e.eng.answers), `"gpu": "integrated"`) {
		t.Fatalf("the answers carry the choice:\n%s", e.eng.answers)
	}
}

func TestLeavingTheGPUOnTheRecommendationSendsAuto(t *testing.T) {
	m, e := hardwareScreen(t)
	send(m, key("enter")) // Hybrid, the recommended one
	typeText(m, "ERASE")
	send(m, key("enter"))
	if !strings.Contains(string(e.eng.answers), `"gpu": "hybrid"`) {
		t.Fatalf("the recommendation is an explicit, visible choice:\n%s", e.eng.answers)
	}
}

func TestTheInstallScreenShowsWhatTheEnginePrinted(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.eng.runCode = 1
	e.eng.runEvents = []install.Event{
		{Stage: "disk", State: "ok"},
		{State: "log", Message: "copying path '/nix/store/abc-glibc' from 'https://cache.nixos.org'"},
		{State: "log", Message: "\x1b[31mbuilding '/nix/store/def-maxor.drv'\x1b[0m"},
		{Stage: "install", State: "fail", Message: "the build failed"},
	}
	m := installModel(t, e)
	walkToAccount(m)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	enter(m, 3)
	typeText(m, "ERASE")
	send(m, key("enter"))
	out := view(m)
	for _, want := range []string{"copying path", "building '/nix/store/def-maxor.drv'", "did not finish"} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[31m") || strings.Contains(out, "[31m") {
		t.Fatalf("terminal escapes from the programs must not reach the screen:\n%s", out)
	}
}

func TestWithoutANetworkTheProfilesAreNotOfferedAndNoneIsSent(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.net.status = install.NetStatus{}
	m := installModel(t, e)
	enter(m, 2)
	send(m, key("enter")) // «Install without a network»
	enter(m, 5)           // time zone, disk, the three storage rows
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	send(m, key("enter")) // → look
	out := view(m)
	if !has(out, "they need a network") || has(out, "space to choose") {
		t.Fatalf("offline, the profiles are explained and not offered:\n%s", out)
	}
	send(m, key("tab"), key("space"), key("enter")) // nothing can be chosen
	enter(m, 1)
	typeText(m, "ERASE")
	send(m, key("enter"))
	if !strings.Contains(string(e.eng.answers), `"profiles": []`) {
		t.Fatalf("no profiles are sent when offline:\n%s", e.eng.answers)
	}
}

func TestTheReviewEditsAStepByItsNumberAndComesBackToTheReview(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	m := installModel(t, e)
	walkToAccount(m)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	enter(m, 3) // look, hardware → review
	out := view(m)
	if !has(out, "1  Language") || !has(out, "4  Time zone") || !has(out, "1-9 edits that step") {
		t.Fatalf("the review numbers the steps:\n%s", out)
	}
	send(m, key("4"))
	if !has(view(m), "Where is this computer?") {
		t.Fatalf("4 goes to the time zone step:\n%s", view(m))
	}
	typeText(m, "madrid")
	send(m, key("enter"))
	out = view(m)
	if !has(out, "Type ERASE") || !has(out, "Madrid (UTC") {
		t.Fatalf("after the change the review comes right back, with the change:\n%s", out)
	}
	typeText(m, "ERASE")
	send(m, key("enter"))
	if len(e.eng.runs) != 1 || !strings.Contains(string(e.eng.answers), `"timezone": "Europe/Madrid"`) {
		t.Fatalf("the install carries the edited answer: %v\n%s", e.eng.runs, e.eng.answers)
	}
}

func TestTheLookIsLightOrDarkAndTheChoiceReachesTheEngine(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	m := installModel(t, e)
	walkToAccount(m)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	send(m, key("enter")) // account → look
	out := view(m)
	for _, want := range []string{"Appearance", "Dark", "Light"} {
		if !has(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if has(out, "Sakura") || has(out, "Ember") {
		t.Fatalf("the other themes are for later, not for the installer:\n%s", out)
	}
	send(m, key("right"), key("enter"), key("enter")) // Light → hardware → review
	if !has(view(m), "Light") {
		t.Fatalf("the review says which look:\n%s", view(m))
	}
	typeText(m, "ERASE")
	send(m, key("enter"))
	if !strings.Contains(string(e.eng.answers), `"theme": "maxor-light"`) {
		t.Fatalf("the answers carry the look:\n%s", e.eng.answers)
	}
}

func TestTheLanguageProposesTheKeyboardAndTheTimeZone(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.net.status = install.NetStatus{} // offline: the time zone cannot be found from the network
	m := installModel(t, e)
	typeText(m, "peru")
	send(m, key("enter")) // language → keyboard
	if !has(view(m), "❯ Spanish (Latin America)") {
		t.Fatalf("Spanish (Peru) proposes the Latin American keyboard:\n%s", view(m))
	}
	if len(e.layouts) == 0 || e.layouts[len(e.layouts)-1] != "latam" {
		t.Fatalf("the proposed layout is the one the keys type with: %v", e.layouts)
	}
	send(m, key("enter"), key("enter")) // keyboard → network → time zone
	if !has(view(m), "❯ America/Lima") {
		t.Fatalf("and the time zone of Peru:\n%s", view(m))
	}
}

func TestAFailureSaysWhatHappenedInWords(t *testing.T) {
	e := newInstallEnv(emptyDisk())
	e.net.status = install.NetStatus{}
	e.eng.runCode = 1
	e.eng.runEvents = []install.Event{
		{Stage: "disk", State: "ok"},
		{State: "log", Message: "error: Cannot build '/nix/store/abc-glibc-locales-2.42.drv'."},
		{Stage: "install", State: "fail", Message: "stage failed (exit 1)"},
	}
	m := installModel(t, e)
	walkToAccount(m)
	fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
	enter(m, 3) // look, hardware → review
	typeText(m, "ERASE")
	send(m, key("enter"))
	if out := view(m); !has(out, "not on the installation medium") || !has(out, "Connect to a network") {
		t.Fatalf("an offline build failure is explained:\n%s", out)
	}
}

func TestEveryStepFitsInAPseudoLanguage(t *testing.T) {
	i18n.SetPseudo(true)
	t.Cleanup(func() { i18n.SetPseudo(false) })
	m := installModel(t, newInstallEnv(windowsDisk()))
	for i := 0; i < 12; i++ {
		out := view(m)
		if lines := strings.Split(out, "\n"); len(lines) != m.h {
			t.Fatalf("step %d: %d rows in a %d-row window:\n%s", i, len(lines), m.h, out)
		}
		send(m, key("ctrl+n"))
	}
}

// A label never touches its value: the storage choice and the review keep a space after a long label (Spanish
// «Sistema de archivos», «Almacenamiento»), and the review's list numbers the steps to edit while a step the installer
// skipped shows a dash, in a wide and a narrow window.
func TestLabelsKeepASpaceBeforeTheirValueAndTheReviewListMarksSkippedSteps(t *testing.T) {
	for _, lang := range []string{"es_PE", "pseudo"} {
		for _, size := range [][2]int{{64, 24}, {110, 34}} {
			t.Run(fmt.Sprintf("%s_%dx%d", lang, size[0], size[1]), func(t *testing.T) {
				m := installModel(t, newInstallEnv(emptyDisk()))
				m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				if lang == "pseudo" {
					i18n.SetPseudo(true)
					t.Cleanup(func() { i18n.SetPseudo(false) })
					send(m, key("enter")) // welcome
				} else {
					typeText(m, lang)
					send(m, key("enter"))
				}
				enter(m, 4) // keyboard, network, region, disk → storage
				if out := view(m); !has(out, i18n.T("Filesystem")+"  ") {
					t.Fatalf("the storage label touches its choice:\n%s", out)
				}
				enter(m, 3)
				fillAccount(m, "Ana", "ana", "pc", "correct-horse-1")
				enter(m, 3) // look, hardware → review
				out := view(m)
				for _, label := range []string{"Method", "Storage", "Account", "Time zone"} {
					if !has(out, i18n.T(label)+"  ") {
						t.Fatalf("the review label %q touches its value:\n%s", i18n.T(label), out)
					}
				}
				if size[0] >= 100 && !has(out, "–  "+i18n.T("How to install")) {
					t.Fatalf("the skipped step shows a dash in the list:\n%s", out)
				}
				if lines := strings.Split(out, "\n"); len(lines) != m.h {
					t.Fatalf("%d rows in a %d-row window:\n%s", len(lines), m.h, out)
				}
			})
		}
	}
}
