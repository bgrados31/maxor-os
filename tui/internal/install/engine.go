package install

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Event is one line of progress the engine writes while it installs.
type Event struct {
	Stage    string   `json:"stage"`
	State    string   `json:"state"` // start | ok | skip | fail | info
	Message  string   `json:"message"`
	Progress *float64 `json:"progress"`
}

// Stages, in the order the engine runs them, with the words the screen shows.
var Stages = []struct{ ID, Title string }{
	{"preflight", "Checking this machine"},
	{"disk", "Partitioning the disk"},
	{"luks", "Encrypting the system partition"},
	{"filesystem", "Creating the filesystems"},
	{"host", "Writing the machine configuration"},
	{"install", "Installing the system"},
	{"bootloader", "Setting up the boot loader"},
	{"finish", "Finishing"},
}

// Engine is what the wizard needs from the installer engine. The real one runs `maxor-install`; the tests
// use a fake.
type Engine interface {
	Probe(ctx context.Context) ([]Disk, error)
	// HashPassword turns the password into the hash that goes in the answers. The password travels on
	// standard input, never on arguments.
	HashPassword(ctx context.Context, password string) (string, error)
	// PlanHash is the fingerprint of the plan these answers describe (needed to install alongside another system).
	PlanHash(ctx context.Context, answers []byte) (string, error)
	// Validate asks the engine whether it accepts the answers; the error text is what it complained about.
	Validate(ctx context.Context, answers []byte) error
	// Plan prints what the installation would do, without doing it.
	Plan(ctx context.Context, answers []byte) (string, error)
	// Run installs. With resume it continues a previous run (finished stages are skipped). It calls onEvent for every
	// progress event and returns the exit code of the engine.
	Run(ctx context.Context, answers []byte, passphrase string, resume bool, onEvent func(Event)) (int, error)
}

// CLI is the real engine: it runs `maxor-install`, with sudo when the installer is not root.
type CLI struct {
	Bin  string // path or name of maxor-install
	Sudo bool   // run the destructive commands through `sudo -n`
	Env  []string
	Dir  string // where the answers and events files are created (default: the temp dir; NewCLI makes a private one)
}

// NewCLI returns the engine to use on a real machine.
func NewCLI() *CLI {
	c := &CLI{Bin: "maxor-install", Sudo: os.Geteuid() != 0}
	// Not /tmp itself: there the kernel stops root from opening with O_CREAT a file another user created
	// (fs.protected_regular), so the engine, run with sudo, could not write its events. A private directory
	// inside it is not sticky, so root may.
	if d, err := os.MkdirTemp("", "maxor-install-"); err == nil {
		c.Dir = d
	}
	return c
}

func (c *CLI) command(ctx context.Context, sudo bool, args ...string) *exec.Cmd {
	argv := append([]string{c.Bin}, args...)
	if sudo && c.Sudo {
		argv = append([]string{"sudo", "-n", "--"}, argv...)
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), c.Env...)
	return cmd
}

func run(cmd *exec.Cmd, stdin string) (stdout, stderr []byte, code int, err error) {
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	err = cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return out.Bytes(), errb.Bytes(), ee.ExitCode(), nil
	}
	return out.Bytes(), errb.Bytes(), 0, err
}

// lastLines keeps the last n non-empty lines of a text, which is where an error message is.
func lastLines(b []byte, n int) string {
	var lines []string
	for _, l := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, strings.TrimSpace(l))
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func (c *CLI) answersFile(answers []byte) (string, error) {
	f, err := os.CreateTemp(c.Dir, "maxor-answers-*.json")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return "", err
	}
	if _, err := f.Write(answers); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// Probe lists the disks.
func (c *CLI) Probe(ctx context.Context) ([]Disk, error) {
	out, errb, code, err := run(c.command(ctx, false, "probe"), "")
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("could not list the disks: %s", lastLines(errb, 3))
	}
	var disks []Disk
	if err := json.Unmarshal(bytes.TrimSpace(out), &disks); err != nil {
		return nil, fmt.Errorf("unexpected answer from the engine: %w", err)
	}
	return disks, nil
}

// HashPassword hashes a password with the engine.
func (c *CLI) HashPassword(ctx context.Context, password string) (string, error) {
	out, errb, code, err := run(c.command(ctx, false, "hashpw"), password+"\n")
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("could not hash the password: %s", lastLines(errb, 2))
	}
	h := strings.TrimSpace(string(out))
	if !strings.HasPrefix(h, "$") {
		return "", errors.New("the engine did not return a password hash")
	}
	return h, nil
}

// PlanHash asks for the fingerprint of the plan.
func (c *CLI) PlanHash(ctx context.Context, answers []byte) (string, error) {
	path, err := c.answersFile(answers)
	if err != nil {
		return "", err
	}
	defer os.Remove(path)
	out, errb, code, err := run(c.command(ctx, false, "hash", "--answers", path), "")
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("could not fingerprint the plan: %s", lastLines(errb, 3))
	}
	return strings.TrimSpace(string(out)), nil
}

// Validate checks the answers with the engine.
func (c *CLI) Validate(ctx context.Context, answers []byte) error {
	path, err := c.answersFile(answers)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	_, errb, code, err := run(c.command(ctx, false, "validate", "--answers", path), "")
	if err != nil {
		return err
	}
	if code != 0 {
		return errors.New(lastLines(errb, 8))
	}
	return nil
}

// Plan prints what the installation would do.
func (c *CLI) Plan(ctx context.Context, answers []byte) (string, error) {
	path, err := c.answersFile(answers)
	if err != nil {
		return "", err
	}
	defer os.Remove(path)
	out, errb, code, err := run(c.command(ctx, false, "plan", "--answers", path), "")
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", errors.New(lastLines(errb, 8))
	}
	return string(out), nil
}

// ParseEvents reads JSON-lines events until the reader ends. Lines that are not events are ignored.
func ParseEvents(r io.Reader, onEvent func(Event)) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Event
		if json.Unmarshal([]byte(line), &e) == nil && e.Stage != "" {
			onEvent(e)
		}
	}
}

// Run installs. The passphrase (if any) goes on standard input, and the progress is read from the events
// file the engine writes.
func (c *CLI) Run(ctx context.Context, answers []byte, passphrase string, resume bool, onEvent func(Event)) (int, error) {
	path, err := c.answersFile(answers)
	if err != nil {
		return 0, err
	}
	defer os.Remove(path)
	ev, err := os.CreateTemp(c.Dir, "maxor-events-*.jsonl")
	if err != nil {
		return 0, err
	}
	evPath := ev.Name()
	ev.Close()
	defer os.Remove(evPath)

	args := []string{"run", "--answers", path, "--events", evPath}
	if resume {
		args = append(args, "--resume")
	}
	if passphrase != "" {
		args = append(args, "--secret-fd", "0")
	}
	cmd := c.command(ctx, true, args...)
	if passphrase != "" {
		cmd.Stdin = strings.NewReader(passphrase)
	}
	// What the engine and the programs it runs print: followed live, line by line, as the log the screen shows.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	logDone := make(chan struct{})
	go func() {
		defer close(logDone)
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		sc.Split(splitLines)
		for sc.Scan() {
			if line := strings.TrimSpace(sc.Text()); line != "" {
				onEvent(Event{State: "log", Message: line})
			}
		}
	}()

	// Follow the events file while the engine works.
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		var off int64
		drain := func() {
			f, err := os.Open(evPath)
			if err != nil {
				return
			}
			defer f.Close()
			if _, err := f.Seek(off, io.SeekStart); err != nil {
				return
			}
			data, _ := io.ReadAll(f)
			// only whole lines: a half-written one is read next time
			if i := bytes.LastIndexByte(data, '\n'); i >= 0 {
				off += int64(i + 1)
				ParseEvents(bytes.NewReader(data[:i+1]), onEvent)
			}
		}
		t := time.NewTicker(120 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				drain()
				return
			case <-t.C:
				drain()
			}
		}
	}()

	<-logDone // the pipe has to be read to its end before Wait
	werr := cmd.Wait()
	close(stop)
	wg.Wait()
	var ee *exec.ExitError
	if errors.As(werr, &ee) {
		return ee.ExitCode(), nil
	}
	return 0, werr
}

// splitLines is bufio.ScanLines that also breaks at a carriage return: programs that draw a progress bar rewrite
// one line with \r, and each rewrite is a line of the log, not one endless line.
func splitLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
