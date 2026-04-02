package jrdev

// PromptBundle holds raw markdown templates (loaded beside cmd/jrdev main).
type PromptBundle struct {
	Plan              string
	Implement         string
	Review            string
	Merge             string
	PR                string // optional; empty skips agent-generated PR title/body
	PrePRReviewPass1  string
	PrePRReviewPass2  string
	PrePRReviewPass3  string
	PrePRReviewPass5  string
}
