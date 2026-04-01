package jrdev

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectConfig is the repository-local jrdev YAML (default .jrdev/config.yaml).
type ProjectConfig struct {
	ConfigReady bool `yaml:"config_ready"`
	Lint        []string
	Unit        []string
	Integration []string
	// Meta holds optional strings or nested values (e.g. source_preset); v1 has no required keys.
	Meta map[string]any `yaml:"meta,omitempty"`
}

type projectConfigDoc struct {
	ConfigReady bool     `yaml:"config_ready"`
	Lint        []string `yaml:"lint,omitempty"`
	Unit        []string `yaml:"unit,omitempty"`
	Integration []string       `yaml:"integration,omitempty"`
	Meta        map[string]any `yaml:"meta,omitempty"`
}

const noChecksConfiguredMessage = "**No checks configured** — `lint`, `unit`, and `integration` are all empty in `.jrdev/config.yaml`. " +
	"Add shell commands under those keys when you want completion prompts to name specific checks."

// LoadProjectConfig reads and parses path as YAML into [ProjectConfig].
func LoadProjectConfig(path string) (ProjectConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ProjectConfig{}, err
	}
	var doc projectConfigDoc
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return ProjectConfig{}, fmt.Errorf("project config yaml: %w", err)
	}
	return ProjectConfig{
		ConfigReady: doc.ConfigReady,
		Lint:        doc.Lint,
		Unit:        doc.Unit,
		Integration: doc.Integration,
		Meta:        doc.Meta,
	}, nil
}

func (p ProjectConfig) allCheckListsEmpty() bool {
	return len(p.Lint) == 0 && len(p.Unit) == 0 && len(p.Integration) == 0
}

func markdownBulletCommands(cmds []string) string {
	if len(cmds) == 0 {
		return ""
	}
	var b strings.Builder
	for _, c := range cmds {
		b.WriteString("- `")
		b.WriteString(c)
		b.WriteString("`\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// PromptLintTests is markdown for templates: lint commands or the all-empty readiness message.
func PromptLintTests(p ProjectConfig) string {
	if p.allCheckListsEmpty() && p.ConfigReady {
		return noChecksConfiguredMessage
	}
	if s := markdownBulletCommands(p.Lint); s != "" {
		return s
	}
	return "*No `lint` checks are listed in `.jrdev/config.yaml`.*"
}

// PromptUnitTests is markdown for templates: unit-test commands or the all-empty readiness message.
func PromptUnitTests(p ProjectConfig) string {
	if p.allCheckListsEmpty() && p.ConfigReady {
		return noChecksConfiguredMessage
	}
	if s := markdownBulletCommands(p.Unit); s != "" {
		return s
	}
	return "*No `unit` checks are listed in `.jrdev/config.yaml`.*"
}

// PromptIntegrationTests is markdown for templates: integration commands or the all-empty readiness message.
func PromptIntegrationTests(p ProjectConfig) string {
	if p.allCheckListsEmpty() && p.ConfigReady {
		return noChecksConfiguredMessage
	}
	if s := markdownBulletCommands(p.Integration); s != "" {
		return s
	}
	return "*No `integration` checks are listed in `.jrdev/config.yaml`.*"
}
