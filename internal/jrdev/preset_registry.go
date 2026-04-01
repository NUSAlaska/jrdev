package jrdev

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// PresetMeta is the stable metadata block for picker display and identification.
type PresetMeta struct {
	ID          string `yaml:"id"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

// Preset is one preset: meta plus optional command lists for repo tooling.
type Preset struct {
	Meta          PresetMeta `yaml:"meta"`
	Lint          []string   `yaml:"lint,omitempty"`
	Unit          []string   `yaml:"unit,omitempty"`
	Integration []string   `yaml:"integration,omitempty"`
}

// PresetSummary is returned by [DiscoverPresets] for a user-facing picker.
type PresetSummary struct {
	ID          string
	Title       string
	Description string
}

type presetDoc struct {
	Meta          PresetMeta `yaml:"meta"`
	Lint          []string   `yaml:"lint,omitempty"`
	Unit          []string   `yaml:"unit,omitempty"`
	Integration []string   `yaml:"integration,omitempty"`
}

// ParsePresetYAML parses a single YAML document into [Preset].
func ParsePresetYAML(data []byte) (Preset, error) {
	var doc presetDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Preset{}, fmt.Errorf("preset yaml: %w", err)
	}
	if strings.TrimSpace(doc.Meta.ID) == "" {
		return Preset{}, fmt.Errorf("preset yaml: meta.id is required")
	}
	if strings.TrimSpace(doc.Meta.Title) == "" {
		return Preset{}, fmt.Errorf("preset yaml: meta.title is required")
	}
	return Preset{
		Meta:          doc.Meta,
		Lint:          doc.Lint,
		Unit:          doc.Unit,
		Integration: doc.Integration,
	}, nil
}

// DiscoverPresets lists every *.yaml preset in fsys (non-recursive).
// fsys must be rooted at the directory that contains the preset files.
// Summaries are sorted by Title, then ID. Duplicate meta.id values produce an error.
func DiscoverPresets(fsys fs.FS) ([]PresetSummary, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("discover presets: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.EqualFold(path.Ext(n), ".yaml") {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	seen := make(map[string]string)
	var out []PresetSummary
	for _, name := range names {
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("discover presets: read %s: %w", name, err)
		}
		p, err := ParsePresetYAML(b)
		if err != nil {
			return nil, fmt.Errorf("discover presets: %s: %w", name, err)
		}
		if prev, dup := seen[p.Meta.ID]; dup {
			return nil, fmt.Errorf("discover presets: duplicate meta.id %q in %s and %s", p.Meta.ID, prev, name)
		}
		seen[p.Meta.ID] = name
		out = append(out, PresetSummary{
			ID:          p.Meta.ID,
			Title:       p.Meta.Title,
			Description: p.Meta.Description,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title != out[j].Title {
			return out[i].Title < out[j].Title
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// LoadPreset returns the preset with the given meta id.
// It first tries id.yaml, then scans all *.yaml files for a matching meta.id.
func LoadPreset(fsys fs.FS, id string) (Preset, error) {
	if strings.TrimSpace(id) == "" {
		return Preset{}, fmt.Errorf("load preset: id is empty")
	}
	b, err := fs.ReadFile(fsys, id+".yaml")
	if err == nil {
		p, err := ParsePresetYAML(b)
		if err != nil {
			return Preset{}, fmt.Errorf("load preset: %s: %w", id+".yaml", err)
		}
		if p.Meta.ID != id {
			return Preset{}, fmt.Errorf("load preset: %s: meta.id %q does not match requested id %q", id+".yaml", p.Meta.ID, id)
		}
		return p, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return Preset{}, fmt.Errorf("load preset: %w", err)
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return Preset{}, fmt.Errorf("load preset: %w", err)
	}
	var found *Preset
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.EqualFold(path.Ext(n), ".yaml") {
			continue
		}
		b, err := fs.ReadFile(fsys, n)
		if err != nil {
			return Preset{}, fmt.Errorf("load preset: read %s: %w", n, err)
		}
		p, err := ParsePresetYAML(b)
		if err != nil {
			return Preset{}, fmt.Errorf("load preset: %s: %w", n, err)
		}
		if p.Meta.ID == id {
			if found != nil {
				return Preset{}, fmt.Errorf("load preset: multiple files define meta.id %q", id)
			}
			p := p
			found = &p
		}
	}
	if found == nil {
		return Preset{}, fmt.Errorf("load preset: no preset with id %q", id)
	}
	return *found, nil
}
