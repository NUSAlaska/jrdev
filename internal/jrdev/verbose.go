package jrdev

// vlog prints a line only when verbose mode is on and log is non-nil.
func vlog(cfg Config, log func(string, ...any), format string, args ...any) {
	if !cfg.Verbose || log == nil {
		return
	}
	log(format, args...)
}
