//go:build !windows

package alert

// Play is silent off Windows (the toast notification still fires).
func Play() {}

// PlayAzan is silent off Windows (the toast + frontend modal still fire).
// Prayer alerts must only ever use PlayAzan, never Play.
func PlayAzan() {}
