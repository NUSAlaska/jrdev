package jrdev

import (
	"embed"
	"io/fs"
)

//go:embed presets/*.yaml
var embeddedPresetFiles embed.FS

// EmbeddedPresetsFS returns the fs.FS rooted at the embedded presets directory
// (one *.yaml file per preset). Use [DiscoverPresets] and [LoadPreset] against it.
func EmbeddedPresetsFS() (fs.FS, error) {
	return fs.Sub(embeddedPresetFiles, "presets")
}
