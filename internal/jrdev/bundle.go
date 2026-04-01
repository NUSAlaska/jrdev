package jrdev

// PromptBundle holds raw markdown templates (loaded beside cmd/jrdev main).
type PromptBundle struct {
	Plan      string
	Implement string
	Review    string
	Merge     string
}
