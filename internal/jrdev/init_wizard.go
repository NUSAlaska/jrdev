package jrdev

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// InitWizardIO is stdin/stdout/stderr for the init wizard (injectable for tests).
type InitWizardIO struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// StdinIsTTY reports whether fd refers to an interactive terminal.
func StdinIsTTY(fd int) bool {
	return term.IsTerminal(fd)
}

// RunInitWizard lists embedded presets, writes a seeded config at cfgPath, prompts the user
// to edit the file, then sets config_ready to true after confirmation.
func RunInitWizard(cfgPath string, presetsFS fs.FS, io InitWizardIO) error {
	summaries, err := DiscoverPresets(presetsFS)
	if err != nil {
		return err
	}
	if len(summaries) == 0 {
		return fmt.Errorf("init wizard: no presets found")
	}

	fmt.Fprintf(io.ErrOut, "jrdev init — pick a language preset for %s\n\n", cfgPath)
	for i, s := range summaries {
		desc := strings.TrimSpace(s.Description)
		if desc != "" {
			fmt.Fprintf(io.Out, "  %d) %s — %s\n", i+1, s.Title, desc)
		} else {
			fmt.Fprintf(io.Out, "  %d) %s\n", i+1, s.Title)
		}
	}
	fmt.Fprintf(io.Out, "\nEnter a number or preset id: ")

	br := bufio.NewReader(io.In)
	line, err := readLineBuf(br)
	if err != nil {
		return err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("init wizard: empty selection")
	}

	id, err := resolvePresetChoice(line, summaries)
	if err != nil {
		return err
	}
	preset, err := LoadPreset(presetsFS, id)
	if err != nil {
		return err
	}

	meta := map[string]any{"source_preset": preset.Meta.ID}
	cfg := ProjectConfig{
		ConfigReady: false,
		Lint:        append([]string(nil), preset.Lint...),
		Unit:        append([]string(nil), preset.Unit...),
		Integration: append([]string(nil), preset.Integration...),
		Meta:        meta,
	}
	if err := WriteProjectConfig(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(io.Out, "\nWrote %s with preset %q (config_ready is false).\n", cfgPath, preset.Meta.ID)
	fmt.Fprintf(io.Out, "Open the file in your editor, adjust commands if needed, then return here.\n")
	fmt.Fprintf(io.Out, "Press Enter when you are done editing: ")
	if _, err := readLineBuf(br); err != nil {
		return err
	}

	if err := SetProjectConfigReady(cfgPath, true); err != nil {
		return err
	}
	fmt.Fprintf(io.ErrOut, "jrdev: set config_ready: true in %s\n", cfgPath)
	return nil
}

func readLineBuf(br *bufio.Reader) (string, error) {
	s, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}

func resolvePresetChoice(line string, summaries []PresetSummary) (string, error) {
	if n, err := strconv.Atoi(line); err == nil {
		if n < 1 || n > len(summaries) {
			return "", fmt.Errorf("init wizard: choose 1–%d", len(summaries))
		}
		return summaries[n-1].ID, nil
	}
	for _, s := range summaries {
		if strings.EqualFold(s.ID, line) {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("init wizard: unknown preset %q", line)
}
