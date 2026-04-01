package jrdev

import (
	"fmt"
	"strings"
)

const agentOutputRetryPreviewMax = 12000

// AppendAgentOutputRetryInstructions appends a correction block for a second agent attempt
// after Validate* or ParsePlan failed on the model's stdout.
func AppendAgentOutputRetryInstructions(basePrompt, phaseTitle string, valErr error, previousOutput string) string {
	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n---\n\n## ")
	sb.WriteString(phaseTitle)
	sb.WriteString(" — correction required (jrdev)\n\n")
	sb.WriteString("Your previous reply failed automated validation:\n\n")
	sb.WriteString(valErr.Error())
	sb.WriteString("\n\n")
	if previousOutput != "" {
		sb.WriteString("Your previous output (reference; may be truncated):\n\n<<<JRDEV_AGENT_PREVIOUS_OUTPUT\n")
		prev := previousOutput
		if len(prev) > agentOutputRetryPreviewMax {
			prev = prev[:agentOutputRetryPreviewMax] + "\n... [truncated by jrdev]\n"
		}
		sb.WriteString(prev)
		sb.WriteString("\nJRDEV_AGENT_PREVIOUS_OUTPUT>>>\n\n")
	}
	sb.WriteString("Correct your response per the phase instructions. **This is your only retry.**\n")
	return sb.String()
}

// ValidateMergeAgentOutput checks merge-phase stdout for AgentPhaseCompleteToken.
func ValidateMergeAgentOutput(agentStdout string) error {
	if strings.Contains(agentStdout, AgentPhaseCompleteToken) {
		return nil
	}
	return fmt.Errorf("merge phase: output missing required substring %q", AgentPhaseCompleteToken)
}
