package activity

// Snapshot is one foreground-window sample.
type Snapshot struct {
	// Exe is the lowercase process image base name (e.g. "chrome.exe").
	Exe string
	// Title is the raw foreground window title.
	Title string
	// IdleSecs is seconds since last input (Any OS; -1 = unknown).
	IdleSecs int64
	// HasWindow is false when no foreground window could be read.
	HasWindow bool
}
