package screens

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
)

// Las cargas de datos compartidos pertenecen a «data»: la aplicación las recibe,
// las guarda en env.Data y las pantallas leen de ahí. Así, lo que carga una
// pestaña lo ve el resto sin volver a pedirlo.

func load(env *core.Env, name, label string, quiet bool, run func(context.Context) (any, error)) tea.Cmd {
	return env.Tasks.Start(task.Task{ID: "data." + name, Label: label, Quiet: quiet, Run: run})
}

func LoadApps(env *core.Env, quiet bool) tea.Cmd {
	return load(env, "apps", "Loading your apps", quiet, func(ctx context.Context) (any, error) { return env.Client.Apps(ctx) })
}
func LoadDoctor(env *core.Env, quiet bool) tea.Cmd {
	return load(env, "doctor", "Running checks", quiet, func(ctx context.Context) (any, error) { return env.Client.Doctor(ctx) })
}
func LoadThemes(env *core.Env, quiet bool) tea.Cmd {
	return load(env, "themes", "Loading themes", quiet, func(ctx context.Context) (any, error) { return env.Client.Themes(ctx) })
}
func LoadHardware(env *core.Env, quiet bool) tea.Cmd {
	return load(env, "hardware", "Detecting your hardware", quiet, func(ctx context.Context) (any, error) { return env.Client.Hardware(ctx) })
}
func LoadProfiles(env *core.Env, quiet bool) tea.Cmd {
	return load(env, "profiles", "Loading profiles", quiet, func(ctx context.Context) (any, error) { return env.Client.Profiles(ctx) })
}

func LoadUpdateStatus(env *core.Env, quiet bool) tea.Cmd {
	return load(env, "updatestatus", "Reading the configuration", quiet, func(ctx context.Context) (any, error) { return env.Client.UpdateStatus(ctx) })
}
func LoadUpdateCache(env *core.Env, quiet bool) tea.Cmd {
	return load(env, "updatecache", "Reading the last scan", quiet, func(ctx context.Context) (any, error) { return env.Client.UpdateCached(ctx) })
}

func LoadAppUpdates(env *core.Env, quiet bool) tea.Cmd {
	return load(env, "appupdates", "Looking for new versions", quiet, func(ctx context.Context) (any, error) { return env.Client.AppUpdates(ctx) })
}

// ApplyData guarda en env.Data el resultado de una carga. Devuelve false si el
// mensaje no era de una carga compartida.
func ApplyData(env *core.Env, d task.DoneMsg) bool {
	if d.Owner() != "data" {
		return false
	}
	if env.Data.Err == nil {
		env.Data.Err = map[string]error{}
	}
	name := d.ID[len("data."):]
	env.Data.Err[name] = d.Err
	if d.Err != nil {
		return true
	}
	switch v := d.Value.(type) {
	case []maxor.App:
		env.Data.Apps, env.Data.AppsLoaded = v, true
	case maxor.Doctor:
		env.Data.Doctor = &v
	case []maxor.Theme:
		env.Data.Themes, env.Data.ThemesLoaded = v, true
	case maxor.Hardware:
		env.Data.Hardware = &v
	case []maxor.Profile:
		env.Data.Profiles = v
	case []maxor.AppUpdate:
		env.Data.AppUpdates, env.Data.UpdatesKnown = v, true
	case maxor.UpdateStatus:
		env.Data.UpdateStatus = &v
	case *maxor.UpdateCheck:
		env.Data.CacheLoaded = true
		if v != nil && (env.Data.Update == nil || v.CheckedAt > env.Data.Update.CheckedAt) {
			env.Data.Update = v
		}
	}
	return true
}

// ActiveTheme devuelve el tema activo de la lista cargada, o nil.
func ActiveTheme(env *core.Env) *maxor.Theme {
	for i := range env.Data.Themes {
		if env.Data.Themes[i].Active {
			return &env.Data.Themes[i]
		}
	}
	return nil
}
