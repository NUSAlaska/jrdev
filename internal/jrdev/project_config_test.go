package jrdev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectConfig_notFound(t *testing.T) {
	_, err := LoadProjectConfig(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("got %v want not exist", err)
	}
}

func TestLoadProjectConfig_invalidYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(p, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProjectConfig(p)
	if err == nil || !strings.Contains(err.Error(), "project config yaml") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadProjectConfig_ok(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	const raw = `config_ready: true
lint:
  - a
  - b
unit:
  - c
integration:
  - d
meta:
  source_preset: go
  extra: 1
`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadProjectConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if !c.ConfigReady {
		t.Fatal("ConfigReady")
	}
	if len(c.Lint) != 2 || c.Lint[0] != "a" || c.Lint[1] != "b" {
		t.Fatalf("lint %#v", c.Lint)
	}
	if len(c.Unit) != 1 || c.Unit[0] != "c" {
		t.Fatalf("unit %#v", c.Unit)
	}
	if len(c.Integration) != 1 || c.Integration[0] != "d" {
		t.Fatalf("integration %#v", c.Integration)
	}
	if c.Meta["source_preset"] != "go" {
		t.Fatalf("meta %#v", c.Meta)
	}
}

func TestLoadProjectConfig_missingLists(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(p, []byte("config_ready: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadProjectConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Lint) != 0 || len(c.Unit) != 0 || len(c.Integration) != 0 {
		t.Fatalf("expected empty lists, got lint=%#v unit=%#v int=%#v", c.Lint, c.Unit, c.Integration)
	}
}

func TestPromptCheckBlocks_allEmptyReady(t *testing.T) {
	p := ProjectConfig{ConfigReady: true}
	for _, s := range []string{
		PromptLintTests(p),
		PromptUnitTests(p),
		PromptIntegrationTests(p),
	} {
		if s != noChecksConfiguredMessage {
			t.Fatalf("want no-checks message, got %q", s)
		}
	}
}

func TestPromptCheckBlocks_lists(t *testing.T) {
	p := ProjectConfig{
		ConfigReady: true,
		Lint:        []string{"lint1"},
		Unit:        []string{},
		Integration: []string{"int1", "int2"},
	}
	if !strings.Contains(PromptLintTests(p), "`lint1`") {
		t.Fatal(PromptLintTests(p))
	}
	if !strings.Contains(PromptUnitTests(p), "No `unit` checks") {
		t.Fatal(PromptUnitTests(p))
	}
	if !strings.Contains(PromptIntegrationTests(p), "`int1`") || !strings.Contains(PromptIntegrationTests(p), "`int2`") {
		t.Fatal(PromptIntegrationTests(p))
	}
}

func TestPromptCheckBlocks_partialEmptyCategories(t *testing.T) {
	p := ProjectConfig{
		ConfigReady: true,
		Lint:        []string{"l"},
		Unit:        []string{"u"},
		Integration: []string{},
	}
	if strings.Contains(PromptLintTests(p), "No checks configured") {
		t.Fatal("lint should list commands")
	}
	if strings.Contains(PromptIntegrationTests(p), "No checks configured") {
		t.Fatal("integration empty but not all categories empty")
	}
	if !strings.Contains(PromptIntegrationTests(p), "No `integration` checks") {
		t.Fatal(PromptIntegrationTests(p))
	}
}

func TestImplementPromptTemplate_rendersCheckFields(t *testing.T) {
	body := "L\n{{.LintTests}}\nU\n{{.UnitTests}}\n"
	pc := ProjectConfig{ConfigReady: true, Lint: []string{"x"}, Unit: []string{"y"}}
	out, err := Render("t", body, ImplementPromptData{
		LintTests: PromptLintTests(pc),
		UnitTests: PromptUnitTests(pc),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "`x`") || !strings.Contains(out, "`y`") {
		t.Fatal(out)
	}
}

func TestMergePromptTemplate_rendersIntegrationField(t *testing.T) {
	body := "{{.IntegrationTests}}"
	pc := ProjectConfig{ConfigReady: true, Integration: []string{"go test -count=1 ./..."}}
	out, err := Render("merge", body, MergePromptData{
		IntegrationTests: PromptIntegrationTests(pc),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "`go test -count=1 ./...`") {
		t.Fatal(out)
	}
}

func TestPromptLintTests_configReadyFalseAllEmptyUsesCategoryMessage(t *testing.T) {
	p := ProjectConfig{ConfigReady: false}
	got := PromptLintTests(p)
	if got == noChecksConfiguredMessage {
		t.Fatal("config_ready false should not use the all-empty ready message")
	}
	if !strings.Contains(got, "No `lint` checks") {
		t.Fatal(got)
	}
}

func TestWriteProjectConfig_roundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	cfg := ProjectConfig{
		ConfigReady: false,
		Lint:        []string{"x"},
		Meta:        map[string]any{"source_preset": "go"},
	}
	if err := WriteProjectConfig(p, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := LoadProjectConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigReady || len(got.Lint) != 1 || got.Lint[0] != "x" {
		t.Fatalf("%#v", got)
	}
	if got.Meta["source_preset"] != "go" {
		t.Fatal(got.Meta)
	}
	if err := SetProjectConfigReady(p, true); err != nil {
		t.Fatal(err)
	}
	got, err = LoadProjectConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ConfigReady {
		t.Fatal("ready")
	}
}

func TestParseProjectConfigYAML(t *testing.T) {
	const raw = "config_ready: true\nlint:\n  - a\n"
	c, err := ParseProjectConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !c.ConfigReady || len(c.Lint) != 1 {
		t.Fatalf("%#v", c)
	}
}
