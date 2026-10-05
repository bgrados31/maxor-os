package install

import "github.com/bgrados31/maxor-os/tui/internal/i18n"

// tr translates a text of the interface (see package i18n); the English text is the key.
var tr = i18n.T

// trn picks the singular or the plural by the language's own rule; trc translates an ambiguous text with its context.
var (
	trn = i18n.N
	trc = i18n.C
)
