package install

import (
	"encoding/json"
	"fmt"
	"os"
)

// The installer ships with short curated lists (Locales, Layouts): the common choices, with names in their own
// language and the console keymap that fits each layout. The complete lists come from the system's own data,
// generated at build time (packages/catalog.nix) and found through MAXOR_CATALOG. They are added after the
// curated ones, so the usual choices stay first.

type catalogFile struct {
	Locales []struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"locales"`
	Layouts []struct {
		XKB     string `json:"xkb"`
		Variant string `json:"variant"`
		Name    string `json:"name"`
	} `json:"layouts"`
}

// MergeCatalog adds the entries of a catalog file after the curated lists, without repeating any.
func MergeCatalog(data []byte) error {
	var c catalogFile
	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("catalog: %w", err)
	}
	if len(c.Locales) == 0 && len(c.Layouts) == 0 {
		return fmt.Errorf("catalog: empty")
	}
	haveLoc := map[string]bool{}
	for _, l := range Locales {
		haveLoc[l.Code] = true
	}
	for _, l := range c.Locales {
		if l.Code != "" && !haveLoc[l.Code] {
			haveLoc[l.Code] = true
			Locales = append(Locales, Locale{Code: l.Code, Name: l.Name})
		}
	}
	haveLay := map[string]bool{}
	for _, l := range Layouts {
		haveLay[l.XKB+"/"+l.Variant] = true
	}
	for _, l := range c.Layouts {
		k := l.XKB + "/" + l.Variant
		if l.XKB != "" && !haveLay[k] {
			haveLay[k] = true
			// no console keymap: the console follows the XKB layout (console.useXkbConfig)
			Layouts = append(Layouts, Layout{Name: l.Name, XKB: l.XKB, Variant: l.Variant})
		}
	}
	return nil
}

// LoadCatalogFromEnv reads the file named by MAXOR_CATALOG, if there is one. Without it, or if it cannot be
// read, the curated lists are all there is: the installer still works.
func LoadCatalogFromEnv() error {
	path := os.Getenv("MAXOR_CATALOG")
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return MergeCatalog(data)
}
