package jrdev

// AgentPhaseCompleteToken must appear in agent stdout before jrdev advances past implement, review, or merge.
const AgentPhaseCompleteToken = "COMPLETE"

// AgentImplementNoCommitToken must appear in implement-phase stdout when the issue is already satisfied
// (e.g. completed while implementing another issue) so jrdev skips the "at least one commit" requirement.
const AgentImplementNoCommitToken = "COMPLETE NO COMMIT"

// CompletionMarker is a merge-phase convention; any output containing AgentPhaseCompleteToken satisfies validation.
const CompletionMarker = "<promise>COMPLETE</promise>"

// PlannedIssue is one row from planner JSON inside <plan>.
type PlannedIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Branch string `json:"branch"`
}

// PlanDocument is the JSON inside the <plan> wrapper.
type PlanDocument struct {
	Issues []PlannedIssue `json:"issues"`
}

// Config holds resolved CLI / runtime options.
type Config struct {
	RepoRoot    string
	Worktrees   string // relative to repo root, default ".worktrees"
	Label       string
	DryRun      bool
	SkipPR      bool
	MaxIters    int // 0 means use default 2N+3
	Verbose     bool
	AgentBin    string
	AgentModel  string // Cursor --model (default DefaultAgentModel)
	// AgentCursorConfigDir, if set, is passed as CURSOR_CONFIG_DIR (must contain cli-config.json). Mutually exclusive with AgentPermissionsFile.
	AgentCursorConfigDir string
	// AgentPermissionsFile is a JSON file with {"allow":["git","go",...],"deny":[]} (bare names become Shell(name)). Materialized to a temp cli-config.json per agent run.
	AgentPermissionsFile string
	GhBin                string
	Integration          string // rev for new integration branch, default "origin/main"
	FreshStart           bool   // if true: skip resume prompt; clean jrdev worktrees/branches then new run
	// Project is parsed from ProjectPath (.jrdev/config.yaml by default); required before the pipeline runs.
	Project     ProjectConfig
	ProjectPath string
	// IntegrationBlocked, when "abort" or "merge", forces the decision when merge stdout contains
	// IntegrationBlockedLinePrefix; overrides meta. Empty uses meta, then prompt or default abort.
	IntegrationBlocked string
}
