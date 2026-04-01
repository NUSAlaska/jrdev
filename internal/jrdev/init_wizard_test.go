package jrdev

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitWizard_selectByNumber_seedsPresetAndSetsReady(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".jrdev", "config.yaml")
	presets, err := EmbeddedPresetsFS()
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	in := bytes.NewBufferString("1\n\n")
	io := InitWizardIO{In: in, Out: &out, ErrOut: &errOut}
	if err := RunInitWizard(cfg, presets, io); err != nil {
		t.Fatal(err)
	}

	pc, err := LoadProjectConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !pc.ConfigReady {
		t.Fatal("expected config_ready true after confirm")
	}
	if len(pc.Lint) == 0 || pc.Lint[0] != "go vet ./..." {
		t.Fatalf("lint %#v", pc.Lint)
	}
	if pc.Meta == nil || pc.Meta["source_preset"] != "go" {
		t.Fatalf("meta %#v", pc.Meta)
	}
	if !strings.Contains(out.String(), "Enter a number") {
		t.Fatal(out.String())
	}
}

func TestRunInitWizard_selectByID(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".jrdev", "config.yaml")
	presets, err := EmbeddedPresetsFS()
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	in := bytes.NewBufferString("go\n\n")
	if err := RunInitWizard(cfg, presets, InitWizardIO{In: in, Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatal(err)
	}
	pc, err := LoadProjectConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if pc.Meta["source_preset"] != "go" {
		t.Fatal(pc.Meta)
	}
}

func TestResolvePresetChoice_invalid(t *testing.T) {
	_, err := resolvePresetChoice("99", []PresetSummary{{ID: "a", Title: "A"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteNonInteractiveStubConfig_omitsMeta(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := WriteNonInteractiveStubConfig(p); err != nil {
		t.Fatal(err)
	}
	rawB, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(rawB)
	if strings.Contains(strings.ToLower(raw), "meta:") {
		t.Fatalf("stub should omit meta, got:\n%s", raw)
	}
	pc, err := LoadProjectConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if pc.ConfigReady || len(pc.Lint) != 0 || len(pc.Unit) != 0 || len(pc.Integration) != 0 {
		t.Fatalf("%#v", pc)
	}
	if pc.Meta != nil {
		t.Fatalf("meta %#v", pc.Meta)
	}
}
