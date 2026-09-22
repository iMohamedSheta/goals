package activity

import "strings"

// Classify maps an executable + window title to a friendly app name, an
// extracted detail (YouTube video title, page name, document…), a domain
// hint and a category: "work" | "distraction" | "other".
//
// NOTE: browsers only expose the window *title* (no extension is installed),
// so site detection is title-based: "Some Video - YouTube - Google Chrome"
// → domain youtube.com, detail "Some Video". Background tabs / background
// audio are NOT visible — whatever app is in the foreground owns the time,
// which is exactly how "I listened to YouTube while working" is counted:
// the foreground work app gets the minutes, not YouTube.
func Classify(exeBase, title string) (app, detail, domain, category string) {
	exe := strings.ToLower(strings.TrimSpace(exeBase))
	t := strings.TrimSpace(title)

	if exe == "" {
		return "Unknown", truncate(t, 180), "", "other"
	}

	// Browsers first — they carry the YouTube / social detection.
	switch exe {
	case "chrome.exe", "msedge.exe", "firefox.exe", "brave.exe", "opera.exe",
		"vivaldi.exe", "arc.exe", "zen.exe", "chromium.exe", "whale.exe":
		return classifyBrowser(exe, t)
	}

	if name, ok := productiveApps[exe]; ok {
		return name, truncate(t, 180), "", "work"
	}
	if name, ok := otherApps[exe]; ok {
		return name, truncate(t, 180), "", "other"
	}
	if name, ok := distractionApps[exe]; ok {
		return name, truncate(t, 180), "", "distraction"
	}
	// Unknown exe → prettified base name, neutral.
	return prettyExe(exe), truncate(t, 180), "", "other"
}

var productiveApps = map[string]string{
	"code.exe": "VS Code", "code - insiders.exe": "VS Code Insiders",
	"cursor.exe": "Cursor", "windsurf.exe": "Windsurf", "zed.exe": "Zed",
	"devenv.exe": "Visual Studio", "rider64.exe": "Rider",
	"idea64.exe": "IntelliJ", "goland64.exe": "GoLand", "pycharm64.exe": "PyCharm",
	"webstorm64.exe": "WebStorm", "phpstorm64.exe": "PhpStorm",
	"sublime_text.exe": "Sublime Text", "notepad++.exe": "Notepad++", "notepad.exe": "Notepad",
	"wt.exe": "Windows Terminal", "windowsterminal.exe": "Windows Terminal",
	"powershell.exe": "PowerShell", "pwsh.exe": "PowerShell", "cmd.exe": "Command Prompt",
	"alacritty.exe": "Alacritty", "wezterm-gui.exe": "WezTerm", "hyper.exe": "Hyper",
	"git-bash.exe": "Git Bash", "mintty.exe": "Git Bash",
	"winword.exe": "Microsoft Word", "excel.exe": "Microsoft Excel", "powerpnt.exe": "Microsoft PowerPoint",
	"outlook.exe": "Outlook", "onenote.exe": "OneNote", "onenotem.exe": "OneNote",
	"figma.exe": "Figma", "figma_agent.exe": "Figma",
	"postman.exe": "Postman", "insomnia.exe": "Insomnia", "dbeaver.exe": "DBeaver",
	"docker desktop.exe": "Docker Desktop", "docker.exe": "Docker",
	"slack.exe": "Slack", "teams.exe": "Microsoft Teams", "ms-teams.exe": "Microsoft Teams",
	"zoom.exe": "Zoom", "zoom_meetings.exe": "Zoom",
	"obsidian.exe": "Obsidian", "notion.exe": "Notion", "todoist.exe": "Todoist",
	"excel": "Microsoft Excel",
	"wps.exe": "WPS Office", "wpp.exe": "WPS Writer",
	"acrobat.exe": "Acrobat Reader", "acrord32.exe": "Acrobat Reader",
	"opencode.exe": "opencode", "go.exe": "Go toolchain", "cargo.exe": "Rust toolchain",
	"node.exe": "Node.js", "python.exe": "Python",
	"ssms.exe": "SQL Server Management", "datagrip64.exe": "DataGrip",
	"android studio.exe": "Android Studio", "studio64.exe": "Android Studio",
	"xcode.exe": "Xcode",
	"eclipse.exe": "Eclipse", "netbeans64.exe": "NetBeans",
	"putty.exe": "PuTTY", "winscp.exe": "WinSCP", "filezilla.exe": "FileZilla",
	"wsl.exe": "WSL", "wslhost.exe": "WSL",
	"thunderbird.exe": "Thunderbird",
	"libreoffice writer.exe": "LibreOffice Writer", "soffice.bin": "LibreOffice",
	"mspaint.exe": "Paint", "photoshop.exe": "Photoshop",
}

var otherApps = map[string]string{
	"spotify.exe": "Spotify", "discord.exe": "Discord",
	"telegram.exe": "Telegram", "whatsapp.exe": "WhatsApp",
	"explorer.exe": "File Explorer", "shellexperiencehost.exe": "Windows Shell",
	"searchhost.exe": "Windows Search", "startmenuexperiencehost.exe": "Start Menu",
	"systemsettings.exe": "Windows Settings", "control.exe": "Control Panel",
	"taskmgr.exe": "Task Manager", "regedit.exe": "Registry Editor",
	"snippingtool.exe": "Snipping Tool", "mspaint.exe": "Paint",
	"calc.exe": "Calculator", "notepad": "Notepad",
	"winrar.exe": "WinRAR", "7zfm.exe": "7-Zip",
	"googledrive.exe": "Google Drive", "dropbox.exe": "Dropbox", "onedrive.exe": "OneDrive",
	"skype.exe": "Skype",
}

var distractionApps = map[string]string{
	"vlc.exe": "VLC", "wmplayer.exe": "Media Player", "mpc-hc64.exe": "Media Player",
	"steam.exe": "Steam", "steamwebhelper.exe": "Steam",
	"epicgameslauncher.exe": "Epic Games", "gog.exe": "GOG", "origin.exe": "Origin",
	"minecraft.exe": "Minecraft", "minecraftpe.exe": "Minecraft",
	"obs64.exe": "OBS Studio",
	"streamdeck.exe": "Stream Deck",
}

// browserNames maps exe → friendly browser label.
var browserNames = map[string]string{
	"chrome.exe": "Google Chrome", "msedge.exe": "Microsoft Edge",
	"firefox.exe": "Firefox", "brave.exe": "Brave", "opera.exe": "Opera",
	"vivaldi.exe": "Vivaldi", "arc.exe": "Arc", "zen.exe": "Zen",
	"chromium.exe": "Chromium", "whale.exe": "Whale",
}

// browserSuffixes are stripped from the right of the window title.
var browserSuffixes = []string{
	" - google chrome", " - microsoft edge", " - brave", " - mozilla firefox",
	" - firefox", " - opera", " - vivaldi", " - arc", " - zen", " - chromium",
}

// distractionTitleHints maps a lowercase title fragment → domain. Checked
// against the *page* part of the title (browser suffix already removed).
var distractionTitleHints = map[string]string{
	"youtube": "youtube.com", "youtu.be": "youtube.com",
	"facebook": "facebook.com", "fb watch": "facebook.com",
	"tiktok": "tiktok.com", "instagram": "instagram.com",
	"twitter": "x.com", " / x": "x.com", "x.com": "x.com",
	"reddit": "reddit.com", "netflix": "netflix.com",
	"twitch": "twitch.tv", "kick.com": "kick.com",
	"disney+": "disneyplus.com", "disney plus": "disneyplus.com",
	"hulu": "hulu.com", "prime video": "primevideo.com",
	"dailymotion": "dailymotion.com", "9gag": "9gag.com",
	"pinterest": "pinterest.com", "snapchat": "snapchat.com",
	"whatsapp web": "web.whatsapp.com",
}

// workTitleHints marks research/docs/dev pages as work even inside a browser.
var workTitleHints = map[string]string{
	"github": "github.com", "stackoverflow": "stackoverflow.com",
	"stack overflow": "stackoverflow.com", "localhost": "localhost",
	"127.0.0.1": "localhost", "notion": "notion.so",
	"figma": "figma.com", "jira": "atlassian.net", "linear": "linear.app",
	"google docs": "docs.google.com", "google sheets": "docs.google.com",
	"google scholar": "scholar.google.com", "mdn web docs": "developer.mozilla.org",
	"microsoft learn": "learn.microsoft.com", "chatgpt": "chatgpt.com",
	"claude": "claude.ai", "opencode": "",
}

func classifyBrowser(exe, title string) (app, detail, domain, category string) {
	app = browserNames[exe]
	if app == "" {
		app = prettyExe(exe)
	}
	page := stripBrowserSuffix(title)
	low := strings.ToLower(page)

	// YouTube gets first-class detail: "Video Title - YouTube" → "Video Title".
	if dom, ok := distractionTitleHints["youtube"]; ok && strings.Contains(low, "youtube") {
		d := strings.TrimSpace(trimSuffixFold(page, " - YouTube"))
		if d == "" || strings.EqualFold(d, "YouTube") {
			d = "YouTube home / feed"
		}
		if strings.EqualFold(d, "home - youtube") {
			d = "YouTube home / feed"
		}
		return app, truncate(d, 180), dom, "distraction"
	}
	for hint, dom := range distractionTitleHints {
		if hint == "youtube" || hint == "youtu.be" {
			continue
		}
		if strings.Contains(low, hint) {
			return app, truncate(page, 180), dom, "distraction"
		}
	}
	for hint, dom := range workTitleHints {
		if strings.Contains(low, hint) {
			return app, truncate(page, 180), dom, "work"
		}
	}
	if strings.TrimSpace(page) == "" {
		return app, "New tab", "", "other"
	}
	return app, truncate(page, 180), "", "other"
}

func stripBrowserSuffix(title string) string {
	low := strings.ToLower(strings.TrimSpace(title))
	for _, s := range browserSuffixes {
		if strings.HasSuffix(low, s) {
			return strings.TrimSpace(title[:len(title)-len(s)])
		}
	}
	return strings.TrimSpace(title)
}

func trimSuffixFold(s, suffix string) string {
	if len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func prettyExe(exe string) string {
	base := exe
	if i := strings.LastIndex(base, ".exe"); i > 0 {
		base = base[:i]
	}
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	words := strings.Fields(base)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	if len(words) == 0 {
		return "Unknown"
	}
	return strings.Join(words, " ")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n])
}
