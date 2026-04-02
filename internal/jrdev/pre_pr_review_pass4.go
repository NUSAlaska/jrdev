package jrdev

import (
	"errors"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// Pass4CheckKind identifies which config list a command came from (GM-009).
type Pass4CheckKind string

const (
	Pass4Lint          Pass4CheckKind = "lint"
	Pass4Unit          Pass4CheckKind = "unit"
	Pass4Integration   Pass4CheckKind = "integration"
	prePRPass5MaxRounds               = 4 // 3 regular + 1 bonus per stable fingerprint (GM-010)
)

// Pass4PlannedCheck is one shell command jrdev will run in Pass 4 order.
type Pass4PlannedCheck struct {
	Kind    Pass4CheckKind
	Command string
}

// BuildPass4CheckPlan returns lint → unit → integration commands (GM-009).
func BuildPass4CheckPlan(p ProjectConfig) []Pass4PlannedCheck {
	var plan []Pass4PlannedCheck
	for _, c := range p.Lint {
		cc := strings.TrimSpace(c)
		if cc == "" {
			continue
		}
		plan = append(plan, Pass4PlannedCheck{Kind: Pass4Lint, Command: c})
	}
	for _, c := range p.Unit {
		cc := strings.TrimSpace(c)
		if cc == "" {
			continue
		}
		plan = append(plan, Pass4PlannedCheck{Kind: Pass4Unit, Command: c})
	}
	for _, c := range p.Integration {
		cc := strings.TrimSpace(c)
		if cc == "" {
			continue
		}
		plan = append(plan, Pass4PlannedCheck{Kind: Pass4Integration, Command: c})
	}
	return plan
}

// Pass4StepRecord is one executed check (GM-009).
type Pass4StepRecord struct {
	Kind     Pass4CheckKind `json:"kind"`
	Command  string         `json:"command"`
	ExitCode int            `json:"exitCode"`
	Success  bool           `json:"success"`
	// Output is omitted or truncated for successful steps to keep artifacts small.
	Output string `json:"output,omitempty"`
}

// Pass4RunOutcome is the result of one full Pass 4 invocation (all checks or stop at first failure).
type Pass4RunOutcome struct {
	Success bool
	Steps   []Pass4StepRecord
	// FailedStep is set when Success is false (first failing command).
	FailedStep *Pass4StepRecord
}

// RunPass4Checks runs planned checks in order, stopping at the first failure (GM-009).
func RunPass4Checks(workDir string, plan []Pass4PlannedCheck) Pass4RunOutcome {
	if len(plan) == 0 {
		return Pass4RunOutcome{Success: true, Steps: nil, FailedStep: nil}
	}
	var steps []Pass4StepRecord
	for _, item := range plan {
		out, code, err := runProjectShellCommand(workDir, item.Command)
		s := string(out)
		if err != nil && !errors.Is(err, errExit) {
			rec := Pass4StepRecord{
				Kind:     item.Kind,
				Command:  item.Command,
				ExitCode: code,
				Success:  false,
				Output:   strings.TrimSpace(s),
			}
			if rec.Output == "" {
				rec.Output = err.Error()
			} else {
				rec.Output = rec.Output + "\n" + err.Error()
			}
			steps = append(steps, rec)
			r := rec
			return Pass4RunOutcome{Success: false, Steps: steps, FailedStep: &r}
		}
		if code != 0 {
			rec := Pass4StepRecord{
				Kind:     item.Kind,
				Command:  item.Command,
				ExitCode: code,
				Success:  false,
				Output:   s,
			}
			steps = append(steps, rec)
			r := rec
			return Pass4RunOutcome{Success: false, Steps: steps, FailedStep: &r}
		}
		rec := Pass4StepRecord{
			Kind:     item.Kind,
			Command:  item.Command,
			ExitCode: code,
			Success:  true,
			Output:   truncatePass4Output(s, true),
		}
		steps = append(steps, rec)
	}
	return Pass4RunOutcome{Success: true, Steps: steps, FailedStep: nil}
}

var errExit = errors.New("non-zero exit")

func truncatePass4Output(s string, success bool) string {
	const maxSuccess = 2048
	if !success {
		return s
	}
	s = strings.TrimSpace(s)
	if len(s) <= maxSuccess {
		return s
	}
	return s[:maxSuccess] + "\n…(truncated)"
}

// runProjectShellCommand runs one config line via the system shell (sh -c / cmd /C).
func runProjectShellCommand(workDir, line string) ([]byte, int, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", line)
	} else {
		cmd = exec.Command("sh", "-c", line)
	}
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
			return out, code, errExit
		}
		return out, -1, err
	}
	return out, code, nil
}

var (
	reISO8601Loose = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?`)
	reDateYMD      = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
	reClock        = regexp.MustCompile(`\b\d{1,2}:\d{2}:\d{2}(?:\.\d+)?\b`)
	reGoTestTime   = regexp.MustCompile(`\(\d+(?:\.\d+)?s\)`)
)

// NormalizeCheckFailureKey strips volatile timestamps and condenses whitespace for stable fingerprints (GM-010).
func NormalizeCheckFailureKey(combinedLog string) string {
	s := combinedLog
	s = reISO8601Loose.ReplaceAllString(s, "<ts>")
	s = reDateYMD.ReplaceAllString(s, "<date>")
	s = reClock.ReplaceAllString(s, "<clock>")
	s = reGoTestTime.ReplaceAllString(s, "(<dur>)")
	// Elapsed / cache lines in go test
	s = regexp.MustCompile(`(?m)^ok\s+\S+\s+\d+\.\d+s\s*$`).ReplaceAllString(s, "ok <pkg> <dur>s")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// Pass4FailureFingerprint combines command kind and normalized log text (GM-010).
func Pass4FailureFingerprint(kind Pass4CheckKind, failedStepOutput string) string {
	return string(kind) + "\x1e" + NormalizeCheckFailureKey(failedStepOutput)
}
