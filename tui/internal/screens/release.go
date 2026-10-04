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
		return "its signature is not valid"
	case "unsigned":
		return "it has no signature"
	case "invalid_manifest":
		return "it is malformed"
	case "rollback":
		return "it is older than one already seen"
	}
	return "it could not be verified"
}

// releaseLines es el bloque «Maxor OS release» de la pestaña Update.
func releaseLines(env *core.Env) []ui.Line {
	p := env.P
	lines := []ui.Line{heading(env, "Maxor OS release")}
	r := env.Data.Release
	if r == nil {
		if env.Data.Err["release"] != nil && env.Data.Err["releasecache"] != nil {
			return append(lines, muted(env, "could not read the release state: maxor logs --last"))
		}
		return append(lines, ui.Skeleton(p, env.Frame, 30), ui.Skeleton(p, env.Frame+3, 20))
	}
	switch {
	case r.Status == "insecure":
		return append(lines,
			ui.T(p.Bad.Bold(true), ui.G.Bad+"  A release was ignored: "+releaseWhy(r.Reason)),
			muted(env, "Nothing was changed. Do not trust this update. See: maxor release status"))
	case r.Status == "never":
		return append(lines, muted(env, "Not checked yet."))
	case releaseAvailable(env):
		lines = append(lines, ui.Of(ui.S(p.Warn.Bold(true), ui.G.Up+" Maxor OS "+r.Latest+" is available"), ui.S(p.Mu, "   you have "+r.Installed)))
		if r.Summary != "" {
			lines = append(lines, muted(env, r.Summary))
		}
		lines = append(lines, ui.Of(ui.S(p.Mu, "signed and verified · press "), ui.S(p.Bold, "v"), ui.S(p.Mu, " to install")))
	default:
		lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+" "), ui.S(p.Text, "Maxor OS "+r.Installed+" is the latest release")))
		if r.OkAt > 0 {
			lines = append(lines, muted(env, "verified "+ago(env.Now(), r.OkAt)))
		}
	}
	if r.Status == "unavailable" {
		// sin red o sin canal: no se dice «al día»; se dice cuándo se verificó por última vez
		when := "never verified"
		if r.OkAt > 0 {
			when = "last verified " + ago(env.Now(), r.OkAt)
		}
		lines = append(lines, ui.T(p.Warn, ui.G.Warn+"  Could not reach the release channel · "+when))
	}
	return lines
}

// releaseBanner son las líneas de aviso del Inicio (vacío si no hay nada que decir).
func releaseBanner(env *core.Env) []ui.Line {
	p := env.P
	r := env.Data.Release
	switch {
	case releaseAvailable(env):
		return []ui.Line{ui.Of(ui.S(p.Warn.Bold(true), ui.G.Up+" Maxor OS "+r.Latest+" is available"), ui.S(p.Mu, "  ·  you have "+r.Installed+"  ·  press "), ui.S(p.Bold, "v"), ui.S(p.Mu, " to review"))}
	case releaseInsecure(env):
		return []ui.Line{ui.Of(ui.S(p.Bad.Bold(true), ui.G.Bad+" A release was ignored: "+releaseWhy(r.Reason)), ui.S(p.Mu, "  ·  nothing was changed"))}
	}
	return nil
}
