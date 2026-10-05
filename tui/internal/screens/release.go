package screens

import (
	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Lo que cada pantalla cuenta de las releases firmadas de Maxor OS. El trabajo (red, firma,
// secuencia, avisos) lo hace `maxor release`; aquí solo se enseña su estado.

// releaseAvailable: hay una release nueva y verificada que se puede instalar.
func releaseAvailable(env *core.Env) bool {
	r := env.Data.Release
	return r != nil && r.Status == "ok" && r.Available && r.Latest != ""
}

// releaseInsecure: lo último que dijo el canal no pasó la verificación y se ignoró.
func releaseInsecure(env *core.Env) bool {
	r := env.Data.Release
	return r != nil && r.Status == "insecure"
}

// releaseWhy explica en una frase por qué se ignoró una release.
func releaseWhy(reason string) string {
	switch reason {
	case "bad_signature":
		return tr("its signature is not valid")
	case "unsigned":
		return tr("it has no signature")
	case "invalid_manifest":
		return tr("it is malformed")
	case "rollback":
		return tr("it is older than one already seen")
	}
	return tr("it could not be verified")
}

// releaseLines es el bloque «Maxor OS release» de la pestaña Update.
func releaseLines(env *core.Env) []ui.Line {
	p := env.P
	lines := []ui.Line{heading(env, tr("Maxor OS release"))}
	r := env.Data.Release
	if r == nil {
		if env.Data.Err["release"] != nil && env.Data.Err["releasecache"] != nil {
			return append(lines, muted(env, tr("could not read the release state: maxor logs --last")))
		}
		return append(lines, ui.Skeleton(p, env.Frame, 30), ui.Skeleton(p, env.Frame+3, 20))
	}
	switch {
	case r.Status == "insecure":
		return append(lines,
			ui.T(p.Bad.Bold(true), ui.G.Bad+"  "+tr("A release was ignored: %s", releaseWhy(r.Reason))),
			muted(env, tr("Nothing was changed. Do not trust this update. See: maxor release status")))
	case r.Status == "never":
		return append(lines, muted(env, tr("Not checked yet.")))
	case releaseAvailable(env):
		lines = append(lines, ui.Of(ui.S(p.Warn.Bold(true), ui.G.Up+" "+tr("Maxor OS %s is available", r.Latest)), ui.S(p.Mu, "   "+tr("you have %s", r.Installed))))
		if r.Summary != "" {
			lines = append(lines, muted(env, r.Summary))
		}
		lines = append(lines, ui.Of(ui.S(p.Mu, tr("signed and verified · press ")), ui.S(p.Bold, "v"), ui.S(p.Mu, tr(" to install"))))
	default:
		lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+" "), ui.S(p.Text, tr("Maxor OS %s is the latest release", r.Installed))))
		if r.OkAt > 0 {
			lines = append(lines, muted(env, tr("verified %s", ago(env.Now(), r.OkAt))))
		}
	}
	if r.Status == "unavailable" {
		// sin red o sin canal: no se dice «al día»; se dice cuándo se verificó por última vez
		when := tr("never verified")
		if r.OkAt > 0 {
			when = tr("last verified %s", ago(env.Now(), r.OkAt))
		}
		lines = append(lines, ui.T(p.Warn, ui.G.Warn+"  "+tr("Could not reach the release channel · %s", when)))
	}
	return lines
}

// releaseBanner son las líneas de aviso del Inicio (vacío si no hay nada que decir).
func releaseBanner(env *core.Env) []ui.Line {
	p := env.P
	r := env.Data.Release
	switch {
	case releaseAvailable(env):
		return []ui.Line{ui.Of(ui.S(p.Warn.Bold(true), ui.G.Up+" "+tr("Maxor OS %s is available", r.Latest)), ui.S(p.Mu, "  ·  "+tr("you have %s", r.Installed)+"  ·  "+tr("press")+" "), ui.S(p.Bold, "v"), ui.S(p.Mu, tr(" to review")))}
	case releaseInsecure(env):
		return []ui.Line{ui.Of(ui.S(p.Bad.Bold(true), ui.G.Bad+" "+tr("A release was ignored: %s", releaseWhy(r.Reason))), ui.S(p.Mu, "  ·  "+tr("nothing was changed")))}
	}
	return nil
}
