package maxor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fake struct {
	calls [][]string
	resp  map[string]resp
}
type resp struct {
	out  string
	err  string
	code int
}

func (f *fake) Run(_ context.Context, args ...string) ([]byte, []byte, int, error) {
	f.calls = append(f.calls, args)
	r, ok := f.resp[strings.Join(args, " ")]
	if !ok {
		return nil, []byte("sin respuesta para " + strings.Join(args, " ")), 99, nil
	}
	return []byte(r.out), []byte(r.err), r.code, nil
}

func newFake(m map[string]resp) (*Client, *fake) {
	f := &fake{resp: m}
	return NewWith(f), f
}

func TestDoctorToleraElCodigoUnoSiHayJSON(t *testing.T) {
	c, _ := newFake(map[string]resp{"doctor --json": {out: `{"ok":false,"fails":1,"warns":0,"groups":[{"title":"System","items":[{"level":"bad","text":"x"}]}]}`, code: 1}})
	d, err := c.Doctor(context.Background())
	if err != nil || d.OK || d.Fails != 1 || d.Groups[0].Items[0].Level != "bad" {
		t.Fatalf("doctor: %+v %v", d, err)
	}
}

func TestDoctorSinSalidaEsUnError(t *testing.T) {
	c, _ := newFake(map[string]resp{"doctor --json": {err: "boom", code: 1}})
	if _, err := c.Doctor(context.Background()); err == nil {
		t.Fatal("sin JSON debe fallar")
	}
}

func TestErrorConservaElCodigoYLaUltimaLinea(t *testing.T) {
	c, _ := newFake(map[string]resp{"theme apply nada": {err: "uno\n✗ maxor: theme 'nada' does not exist\n", code: 3}})
	err := c.ApplyTheme(context.Background(), "nada")
	var me *Error
	if !errors.As(err, &me) || me.Code != 3 || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("error: %v", err)
	}
}

func TestThemesYApps(t *testing.T) {
	c, _ := newFake(map[string]resp{
		"theme list --json": {out: `[{"id":"sakura","name":"Sakura","mode":"dark","active":true,"colors":{"bg":"#120b12","s":"#1d121d","s2":"#2a1a2a","fg":"#fbe9f2","mu":"#a88a9d","ac":"#ff86b8","ac2":"#ffc2a6","on":"#1b0b14"}}]`},
		"apps --json":       {out: `[{"source":"nix","id":"btop","name":"btop","version":"btop-1.4.7"}]`},
	})
	th, err := c.Themes(context.Background())
	if err != nil || len(th) != 1 || !th[0].Active || th[0].Colors.Ac != "#ff86b8" {
		t.Fatalf("themes: %+v %v", th, err)
	}
	ap, err := c.Apps(context.Background())
	if err != nil || ap[0].ID != "btop" {
		t.Fatalf("apps: %+v %v", ap, err)
	}
}

func TestSearchPasaLasPalabrasPorSeparado(t *testing.T) {
	c, f := newFake(map[string]resp{"search brave browser --json": {out: `[{"source":"nix","id":"brave","name":"brave","version":"1","description":"d"}]`}})
	r, err := c.Search(context.Background(), "  brave   browser ")
	if err != nil || r[0].ID != "brave" {
		t.Fatalf("search: %+v %v", r, err)
	}
	if strings.Join(f.calls[0], " ") != "search brave browser --json" {
		t.Fatalf("argumentos: %v", f.calls[0])
	}
}

func TestInstallElijeElOrigenYFallaSiNoSeInstalo(t *testing.T) {
	c, f := newFake(map[string]resp{
		"install --flatpak com.x.Y --json": {out: `[{"id":"com.x.Y","source":"flatpak","ok":true}]`},
		"install --nix btop --json":        {out: `[{"id":"btop","source":"nix","ok":false}]`, code: 1},
	})
	if err := c.Install(context.Background(), "flatpak", "com.x.Y"); err != nil {
		t.Fatal(err)
	}
	if err := c.Install(context.Background(), "nix", "btop"); err == nil {
		t.Fatal("ok=false debe ser un error")
	}
	if len(f.calls) != 2 {
		t.Fatal("llamadas")
	}
}

func TestSetProfileNoReconstruye(t *testing.T) {
	c, f := newFake(map[string]resp{"profile enable gaming --no-apply": {}, "profile disable gaming --no-apply": {}})
	if err := c.SetProfile(context.Background(), "gaming", true); err != nil {
		t.Fatal(err)
	}
	if err := c.SetProfile(context.Background(), "gaming", false); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 {
		t.Fatal("llamadas")
	}
}

func TestUpdateCheckUsaNoLockSegunSePida(t *testing.T) {
	c, f := newFake(map[string]resp{
		"update --json":            {out: `{"up_to_date":false,"kernel":true,"counts":{"new":1,"updated":2,"removed":0,"changed":1,"config":3},"changes":[{"kind":"updated","name":"firefox","from":"149.0","to":"150.0","size":"+1 MiB"}]}`},
		"update --json --no-lock": {out: `{"up_to_date":true,"kernel":false,"counts":{},"changes":[]}`},
	})
	u, err := c.UpdateCheck(context.Background(), true)
	if err != nil || !u.Kernel || u.Counts.Updated != 2 || u.Changes[0].To != "150.0" {
		t.Fatalf("check: %+v %v", u, err)
	}
	u, err = c.UpdateCheck(context.Background(), false)
	if err != nil || !u.UpToDate {
		t.Fatalf("check sin lock: %+v %v", u, err)
	}
	if strings.Join(f.calls[1], " ") != "update --json --no-lock" {
		t.Fatalf("argumentos: %v", f.calls[1])
	}
}

func TestRespuestaInvalidaEsUnError(t *testing.T) {
	c, _ := newFake(map[string]resp{"apps --json": {out: "no es json"}})
	if _, err := c.Apps(context.Background()); err == nil || !strings.Contains(err.Error(), "no válida") {
		t.Fatalf("error: %v", err)
	}
}

func TestEjecutorRealCodigosYEntorno(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "maxor")
	script := "#!/bin/sh\ncase \"$1\" in\n  version) echo \"{\\\"version\\\":\\\"9.9\\\",\\\"schema\\\":1,\\\"os\\\":\\\"x\\\"}\"; echo \"$NO_COLOR $MAXOR_NO_TUI\" >&2 ;;\n  fail) echo 'algo falló' >&2; exit 3 ;;\nesac\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAXOR_BIN", bin)
	c := New()
	v, err := c.Version(context.Background())
	if err != nil || v.Version != "9.9" || v.Schema != 1 {
		t.Fatalf("version: %+v %v", v, err)
	}
	_, err = c.run(context.Background(), "fail")
	var me *Error
	if !errors.As(err, &me) || me.Code != 3 || !strings.Contains(me.Stderr, "algo falló") {
		t.Fatalf("fail: %v", err)
	}
	_, errb, _, _ := execRunner{bin: bin}.Run(context.Background(), "version")
	if !strings.Contains(string(errb), "1 1") {
		t.Fatalf("la pantalla debe llamar a la CLI con NO_COLOR y MAXOR_NO_TUI: %q", errb)
	}
}

func TestUpdateStatusYCache(t *testing.T) {
	c, _ := newFake(map[string]resp{
		"update --status": {out: `{"flake":"/home/b/nixos-config","branch":"development","commit":"9b80dfd","dirty":true,"files":5,"fingerprint":"abc","channel":"nixos-26.05","nixpkgs_rev":"774debe","nixpkgs_date":1790920529,"generation":28}`},
		"update --cached": {out: `{"up_to_date":false,"kernel":false,"counts":{"new":1},"changes":[],"checked_at":1790000000,"fingerprint":"abc","lock":false}`},
	})
	st, err := c.UpdateStatus(context.Background())
	if err != nil || st.Branch != "development" || !st.Dirty || st.Files != 5 || st.Generation != 28 || st.Channel != "nixos-26.05" {
		t.Fatalf("status: %+v %v", st, err)
	}
	u, err := c.UpdateCached(context.Background())
	if err != nil || u == nil || u.CheckedAt != 1790000000 || u.Fingerprint != "abc" || u.Counts.New != 1 {
		t.Fatalf("cache: %+v %v", u, err)
	}
	c2, _ := newFake(map[string]resp{"update --cached": {out: "null"}})
	if u, err := c2.UpdateCached(context.Background()); err != nil || u != nil {
		t.Fatalf("sin escaneo guardado debe ser nil: %+v %v", u, err)
	}
}

func TestDoctorTraeIdYArreglo(t *testing.T) {
	c, _ := newFake(map[string]resp{"doctor --json": {out: `{"ok":true,"fails":0,"warns":1,"groups":[{"title":"Configuration","items":[{"level":"warn","text":"uncommitted changes","id":"git_dirty","fix":"git status","confirm":false},{"level":"ok","text":"fine","id":"git_clean","fix":null,"confirm":false}]}]}`}})
	d, err := c.Doctor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	it := d.Groups[0].Items
	if it[0].ID != "git_dirty" || it[0].Fix != "git status" || it[1].Fix != "" {
		t.Fatalf("items: %+v", it)
	}
}

func TestGenerations(t *testing.T) {
	c, _ := newFake(map[string]resp{"rollback --list --json": {out: `[{"generation":34,"date":"2026-10-03 09:24:54","nixos":"26.05","kernel":"6.18.54","current":true},{"generation":33,"date":"2026-10-03 09:12:27","nixos":"26.05","kernel":"6.18.54","current":false}]`}})
	g, err := c.Generations(context.Background())
	if err != nil || len(g) != 2 || !g[0].Current || g[1].Generation != 33 || g[1].Kernel != "6.18.54" {
		t.Fatalf("generaciones: %+v %v", g, err)
	}
}

// El ejecutor real: con un «maxor» de mentira (un script) comprobamos que la contraseña llega
// solo por la entrada estándar, que MAXOR_SUDO_STDIN se activa, que las barras de progreso
// (\r) y los colores no ensucian las líneas y que se conserva el código de salida.
func TestStreamRealEntregaLaContrasenaYLasLineas(t *testing.T) {
	dir := t.TempDir()
	bin := dir + "/maxor"
	script := "#!/bin/sh\nread pw\necho \"flag=$MAXOR_SUDO_STDIN pw=$pw args=$*\"\nprintf 'progreso 10%%\\rprogreso 90%%\\r\\033[32mlisto\\033[0m\\n'\necho 'fallo' >&2\nexit 3\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &Client{r: execRunner{bin: bin}, bin: bin}
	var lines []string
	code, err := c.Stream(context.Background(), "secreto", func(l string) { lines = append(lines, l) }, "update", "-y")
	if err != nil || code != 3 {
		t.Fatalf("código %d, error %v", code, err)
	}
	got := strings.Join(lines, "|")
	for _, want := range []string{"flag=1 pw=secreto args=update -y", "progreso 10%", "progreso 90%", "listo", "fallo"} {
		if !strings.Contains(got, want) {
			t.Fatalf("falta %q en %q", want, got)
		}
	}
	if strings.Contains(got, "\x1b") {
		t.Fatalf("los colores no deben llegar: %q", got)
	}
	// sin contraseña no se activa el modo de sudo por entrada estándar
	lines = nil
	if _, err := c.Stream(context.Background(), "", func(l string) { lines = append(lines, l) }, "x"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(lines, "|"), "flag=1") {
		t.Fatalf("sin contraseña, sin MAXOR_SUDO_STDIN: %v", lines)
	}
}
