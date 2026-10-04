package install

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeEngine writes a script that behaves like maxor-install, so the CLI client is tested with a real
// process, real files and a real standard input.
func fakeEngine(t *testing.T, script string) *CLI {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "maxor-install")
	if err := os.WriteFile(bin, []byte("#!"+shellPath+"\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return &CLI{Bin: bin, Dir: dir}
}
