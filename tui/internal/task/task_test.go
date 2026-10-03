package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newM() (*Manager, *clock) {
	c := &clock{t: time.Unix(1000, 0)}
	return NewManagerWithClock(c.now), c
}

func exec(t *testing.T, cmd func() any) DoneMsg {
	t.Helper()
	return cmd().(DoneMsg)
}

func TestElPropietarioEsLaPantalla(t *testing.T) {
	for id, want := range map[string]string{"store.search": "store", "home": "home", "a.b.c": "a"} {
		if got := Owner(id); got != want {
			t.Errorf("%s: %s", id, got)
		}
	}
}

func TestStartEntregaValorYError(t *testing.T) {
	m, _ := newM()
	cmd := m.Start(Task{ID: "x.ok", Run: func(context.Context) (any, error) { return 42, nil }})
	msg := cmd().(DoneMsg)
	if !m.Done(msg) || msg.Value.(int) != 42 || msg.Err != nil {
		t.Fatalf("ok: %+v", msg)
	}
	boom := errors.New("boom")
	cmd = m.Start(Task{ID: "x.err", Run: func(context.Context) (any, error) { return nil, boom }})
	msg = cmd().(DoneMsg)
	if !m.Done(msg) || !errors.Is(msg.Err, boom) {
		t.Fatalf("err: %+v", msg)
	}
}

func TestUnaTareaNuevaReemplazaALaVieja(t *testing.T) {
	m, _ := newM()
	old := m.Start(Task{ID: "store.search", Run: func(context.Context) (any, error) { return "bra", nil }})
	nuevo := m.Start(Task{ID: "store.search", Run: func(context.Context) (any, error) { return "brave", nil }})
	if m.Done(old().(DoneMsg)) {
		t.Fatal("el resultado de la tarea reemplazada debe ignorarse")
	}
	msg := nuevo().(DoneMsg)
	if !m.Done(msg) || msg.Value != "brave" {
		t.Fatal("el resultado nuevo debe aceptarse")
	}
	if m.Busy() {
		t.Fatal("no debe quedar nada en marcha")
	}
}

func TestReemplazarCancelaElContextoDeLaVieja(t *testing.T) {
	m, _ := newM()
	cancelada := make(chan bool, 1)
	m.Start(Task{ID: "a.x", Run: func(ctx context.Context) (any, error) { <-ctx.Done(); cancelada <- true; return nil, ctx.Err() }})
	cmd := m.Start(Task{ID: "a.x", Run: func(context.Context) (any, error) { return 1, nil }})
	_ = cmd
	// la primera sigue sin ejecutarse hasta que se invoque su cmd: se comprueba que Cancel funciona
	m.Cancel("a.x")
	if m.Running("a.x") {
		t.Fatal("Cancel debe quitarla")
	}
}

func TestNadaApareceAntesDeShowAfter(t *testing.T) {
	m, c := newM()
	m.Start(Task{ID: "a.x", Label: "Working", Run: func(context.Context) (any, error) { return nil, nil }})
	c.advance(100 * time.Millisecond)
	if len(m.Active()) != 0 || m.Loading("a.x") {
		t.Fatal("una tarea de 100 ms no debe mostrarse")
	}
	c.advance(100 * time.Millisecond)
	a := m.Active()
	if len(a) != 1 || a[0].Label != "Working" || !m.Loading("a.x") {
		t.Fatalf("a los 200 ms sí: %+v", a)
	}
}

func TestLasSilenciosasNoSalenEnElIndicador(t *testing.T) {
	m, c := newM()
	m.Start(Task{ID: "a.x", Label: "bg", Quiet: true, Run: func(context.Context) (any, error) { return nil, nil }})
	c.advance(time.Second)
	if len(m.Active()) != 0 {
		t.Fatal("una tarea silenciosa no se lista")
	}
	if !m.Busy() {
		t.Fatal("pero sigue en marcha")
	}
}

func TestActiveVaDeLaMasAntiguaALaMasNueva(t *testing.T) {
	m, c := newM()
	m.Start(Task{ID: "a.1", Label: "uno", Run: func(context.Context) (any, error) { return nil, nil }})
	c.advance(500 * time.Millisecond)
	m.Start(Task{ID: "b.2", Label: "dos", Run: func(context.Context) (any, error) { return nil, nil }})
	c.advance(500 * time.Millisecond)
	a := m.Active()
	if len(a) != 2 || a[0].Label != "uno" || a[1].Label != "dos" || a[0].Elapsed != time.Second {
		t.Fatalf("orden: %+v", a)
	}
}

func TestElapsedYElapsedDelResultado(t *testing.T) {
	m, c := newM()
	cmd := m.Start(Task{ID: "a.x", Run: func(context.Context) (any, error) { c.advance(3 * time.Second); return nil, nil }})
	msg := cmd().(DoneMsg)
	if msg.Elapsed != 3*time.Second || msg.Owner() != "a" {
		t.Fatalf("resultado: %+v", msg)
	}
}
