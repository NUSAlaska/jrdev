package jrdev

import (
	"embed"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"
)

//go:embed testdata/preset_registry/*.yaml
var testPresetRegistryFS embed.FS

//go:embed testdata/preset_registry_dup/*.yaml
var testPresetRegistryDupFS embed.FS

func testPresetDir(tb testing.TB, root embed.FS, dir string) fs.FS {
	tb.Helper()
	sub, err := fs.Sub(root, filepath.ToSlash(dir))
	if err != nil {
		tb.Fatal(err)
	}
	return sub
}

func TestDiscoverPresets(t *testing.T) {
	fsys := testPresetDir(t, testPresetRegistryFS, "testdata/preset_registry")
	tests := []struct {
		name string
		want []PresetSummary
	}{
		{
			name: "fixture set",
			want: []PresetSummary{
				{ID: "custom-slug", Title: "Custom", Description: "Filename differs from id; load exercises scan path"},
				{ID: "rust", Title: "Rust", Description: "Cargo-based workflow"},
				{ID: "typescript", Title: "TypeScript", Description: "Node / tsc workflow"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DiscoverPresets(fsys)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DiscoverPresets() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDiscoverPresets_duplicateID(t *testing.T) {
	fsys := testPresetDir(t, testPresetRegistryDupFS, "testdata/preset_registry_dup")
	_, err := DiscoverPresets(fsys)
	if err == nil {
		t.Fatal("expected error for duplicate meta.id")
	}
}

func TestLoadPreset(t *testing.T) {
	fsys := testPresetDir(t, testPresetRegistryFS, "testdata/preset_registry")
	tests := []struct {
		name    string
		id      string
		want    Preset
		wantErr bool
	}{
		{
			name: "by filename",
			id:   "rust",
			want: Preset{
				Meta: PresetMeta{
					ID:          "rust",
					Title:       "Rust",
					Description: "Cargo-based workflow",
				},
				Lint:        []string{"cargo clippy"},
				Unit:        []string{"cargo test"},
				Integration: nil,
			},
		},
		{
			name: "typescript with empty integration slice",
			id:   "typescript",
			want: Preset{
				Meta: PresetMeta{
					ID:          "typescript",
					Title:       "TypeScript",
					Description: "Node / tsc workflow",
				},
				Lint:          []string{"npx eslint ."},
				Unit:          []string{"npm test"},
				Integration: []string{},
			},
		},
		{
			name: "scan when filename is not id.yaml",
			id:   "custom-slug",
			want: Preset{
				Meta: PresetMeta{
					ID:          "custom-slug",
					Title:       "Custom",
					Description: "Filename differs from id; load exercises scan path",
				},
				Lint: []string{"echo lint"},
				Unit: []string{"echo unit"},
			},
		},
		{
			name:    "missing",
			id:      "nope",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadPreset(fsys, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadPreset(%q) = %#v, want %#v", tt.id, got, tt.want)
			}
		})
	}
}

func TestParsePresetYAML_validation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{"missing meta.id", "meta:\n  title: x\n", true},
		{"missing meta.title", "meta:\n  id: a\n", true},
		{"ok minimal lists", "meta:\n  id: a\n  title: A\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePresetYAML([]byte(tt.yaml))
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbeddedPresetsFS(t *testing.T) {
	fsys, err := EmbeddedPresetsFS()
	if err != nil {
		t.Fatal(err)
	}
	sums, err := DiscoverPresets(fsys)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, s := range sums {
		ids = append(ids, s.ID)
	}
	if !reflect.DeepEqual(ids, []string{"go"}) {
		t.Fatalf("embedded preset ids = %v, want [go]", ids)
	}
	p, err := LoadPreset(fsys, "go")
	if err != nil {
		t.Fatal(err)
	}
	if p.Meta.ID != "go" || p.Meta.Title != "Go" {
		t.Fatalf("%#v", p.Meta)
	}
	if len(p.Lint) == 0 || len(p.Unit) == 0 {
		t.Fatalf("expected non-empty lint and unit: %#v", p)
	}
}
