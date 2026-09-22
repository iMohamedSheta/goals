//go:build !windows

package ai

// killTree is a no-op off Windows (CommandContext suffices there).
func killTree(pid int) {}
