package jrdev

// DefaultMaxIterations returns 2*N+3 per PRD when --max-iterations is not set.
func DefaultMaxIterations(queueCountN int) int {
	return 2*queueCountN + 3
}

// EffectiveMaxIterations applies explicit override or default.
func EffectiveMaxIterations(n int, override int) int {
	if override > 0 {
		return override
	}
	return DefaultMaxIterations(n)
}
