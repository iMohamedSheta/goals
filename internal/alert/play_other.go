//go:build !windows

package alert

// Play is silent off Windows (the toast notification still fires).
func Play() {}
