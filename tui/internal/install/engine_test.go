package install

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

var shellPath = func() string {
	p, err := exec.LookPath("bash")
	if err != nil {
		return "/bin/sh"
	}
	return p
}()

func TestProbeDecodesDisks(t *testing.T) {
	c := fakeEngine(t, `echo '[{"path":"/dev/vda","model":"QEMU","size":107374182400,"transport":"","removable":false,"readonly":false,"mounted":false,"windows":true,"maxor":false,"partitions":[{"path":"/dev/vda1","size":1000,"fstype":"ntfs","label":"Windows","partlabel":"","mountpoints":[]}],"free":[{"start":2048,"end":209715166,"sectors":209713119}]}]'`)
	disks, err := c.Probe(context.Background())
	if err != nil || len(disks) != 1 {
		t.Fatalf("probe: %v %v", disks, err)
	}
	d := disks[0]
	if d.Path != "/dev/vda" || !d.Windows || d.Contents() != "1 partition (Windows)" || d.LargestFree() == nil {
		t.Fatalf("disk: %+v (%s)", d, d.Contents())
	}
}

func TestProbeFailureCarriesTheEnginesWords(t *testing.T) {
	c := fakeEngine(t, `echo "lsblk exploded" >&2; exit 1`)
	_, err := c.Probe(context.Background())
	if err == nil || !strings.Contains(err.Error(), "lsblk exploded") {
		t.Fatalf("want the engine's message, got %v", err)
	}
}

func TestHashPasswordUsesStdinNeverArguments(t *testing.T) {
	dir := t.TempDir()
	argsFile := dir + "/args"
	c := fakeEngine(t, `echo "$@" > `+argsFile+`; read -r pw; echo '$6$salt$hashof'"$pw"`)
	h, err := c.HashPassword(context.Background(), "s3cret-pass")
	if err != nil || !strings.Contains(h, "hashofs3cret-pass") {
		t.Fatalf("hash: %q %v", h, err)
	}
	args, _ := os.ReadFile(argsFile)
	if strings.Contains(string(args), "s3cret") {
		t.Fatalf("the password reached the arguments: %s", args)
	}
}

func TestHashPasswordRejectsAnythingThatIsNotAHash(t *testing.T) {
	c := fakeEngine(t, `echo "nope"`)
	if _, err := c.HashPassword(context.Background(), "whatever1"); err == nil {
		t.Fatal("a non-hash answer must be an error")
	}
}

func TestValidateReturnsWhatTheEngineComplainedAbout(t *testing.T) {
	c := fakeEngine(t, `echo "answers.disk.device: has an invalid format" >&2; exit 3`)
	err := c.Validate(context.Background(), []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "disk.device") {
		t.Fatalf("validate: %v", err)
	}
	ok := fakeEngine(t, `echo "The answers are valid."`)
	if err := ok.Validate(context.Background(), []byte(`{}`)); err != nil {
		t.Fatalf("valid answers must pass: %v", err)
	}
}

func TestAnswersFilesAreRemovedAndPrivate(t *testing.T) {
	dir := t.TempDir()
	c := fakeEngine(t, `stat -c %a "$3" > `+dir+`/mode; exit 0`)
	c.Dir = dir
	if _, err := c.PlanHash(context.Background(), []byte(`{"secret":true}`)); err != nil {
		t.Fatal(err)
	}
	mode, _ := os.ReadFile(dir + "/mode")
	if strings.TrimSpace(string(mode)) != "600" {
		t.Fatalf("the answers file must be private, mode %q", mode)
	}
	left, _ := os.ReadDir(dir)
	for _, e := range left {
		if strings.HasPrefix(e.Name(), "maxor-answers-") {
			t.Fatalf("the answers file was left behind: %s", e.Name())
		}
	}
}

func TestRunStreamsEventsPassesThePassphraseOnStdinAndReturnsTheExitCode(t *testing.T) {
	dir := t.TempDir()
	c := fakeEngine(t, `
events=""
while [ $# -gt 0 ]; do case "$1" in --events) events="$2"; shift ;; --secret-fd) fd="$2"; shift ;; esac; shift; done
cat > `+dir+`/stdin-seen
echo '{"stage":"disk","state":"start","message":""}' >> "$events"
sleep 0.3
echo '{"stage":"disk","state":"ok","message":"","progress":0.25}' >> "$events"
echo 'not json at all' >> "$events"
echo '{"stage":"done","state":"ok","message":"Maxor OS is installed","progress":1.0}' >> "$events"
exit 0`)
	var mu sync.Mutex
	var got []Event
	code, err := c.Run(context.Background(), []byte(`{}`), "correct horse battery", false, func(e Event) {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
	})
	if err != nil || code != 0 {
		t.Fatalf("run: %d %v", code, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 || got[0].State != "start" || got[2].Stage != "done" || got[1].Progress == nil || *got[1].Progress != 0.25 {
		t.Fatalf("events: %+v", got)
	}
	seen, _ := os.ReadFile(dir + "/stdin-seen")
	if string(seen) != "correct horse battery" {
		t.Fatalf("the passphrase must arrive on stdin, got %q", seen)
	}
}

func TestRunWithoutPassphraseDoesNotAskForTheSecretFd(t *testing.T) {
	dir := t.TempDir()
	c := fakeEngine(t, `echo "$@" > `+dir+`/args; exit 0`)
	if _, err := c.Run(context.Background(), []byte(`{}`), "", false, func(Event) {}); err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(dir + "/args")
	if strings.Contains(string(args), "--secret-fd") {
		t.Fatalf("no passphrase, no secret fd: %s", args)
	}
}

func TestRunReturnsTheEnginesExitCodeOnFailure(t *testing.T) {
	c := fakeEngine(t, `exit 4`)
	code, err := c.Run(context.Background(), []byte(`{}`), "", false, func(Event) {})
	if err != nil || code != 4 {
		t.Fatalf("a failing engine is a code, not an error: %d %v", code, err)
	}
}

func TestRunCleansUpItsTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	c := fakeEngine(t, `exit 0`)
	c.Dir = dir
	_, _ = c.Run(context.Background(), []byte(`{}`), "", false, func(Event) {})
	left, _ := os.ReadDir(dir)
	for _, e := range left {
		if strings.HasPrefix(e.Name(), "maxor-answers-") || strings.HasPrefix(e.Name(), "maxor-events-") {
			t.Fatalf("left behind: %s", e.Name())
		}
	}
}

func TestParseEventsIgnoresNoise(t *testing.T) {
	var got []Event
	ParseEvents(strings.NewReader("\nxxx\n{\"stage\":\"a\",\"state\":\"ok\"}\n{\"nope\":1}\n"), func(e Event) { got = append(got, e) })
	if len(got) != 1 || got[0].Stage != "a" {
		t.Fatalf("%+v", got)
	}
}

func TestRunResumeAsksTheEngineToContinue(t *testing.T) {
	dir := t.TempDir()
	c := fakeEngine(t, `echo "$@" > `+dir+`/args; exit 0`)
	if _, err := c.Run(context.Background(), []byte(`{}`), "", true, func(Event) {}); err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(dir + "/args")
	if !strings.Contains(string(args), "--resume") {
		t.Fatalf("resume must reach the engine: %s", args)
	}
}

func TestRunFollowsWhatTheEnginePrintsAsLogLines(t *testing.T) {
	c := fakeEngine(t, `
printf 'copying path 1\n' >&2
sleep 0.1
printf 'downloading  10%%\rdownloading  60%%\rdownloading 100%%\n' >&2
printf '\n   \nlast line without newline' >&2
exit 0`)
	var mu sync.Mutex
	var logs []string
	code, err := c.Run(context.Background(), []byte(`{}`), "", false, func(e Event) {
		if e.State == "log" {
			mu.Lock()
			logs = append(logs, e.Message)
			mu.Unlock()
		}
	})
	if err != nil || code != 0 {
		t.Fatalf("run: %d %v", code, err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{"copying path 1", "downloading  10%", "downloading  60%", "downloading 100%", "last line without newline"}
	if strings.Join(logs, "|") != strings.Join(want, "|") {
		t.Fatalf("log lines: %q", logs)
	}
}
