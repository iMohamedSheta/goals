# Screenshot the Goals desktop app while it runs.
#
# Launches the exe with an isolated database, waits for its main window,
# captures the window to a PNG, then kills the app.
#
# Usage (local):
#   ./scripts/screenshot.ps1 -ExePath "build/bin/goals.exe" `
#     -DbPath "$env:TEMP/goals-demo/goals.db" -OutFile "docs/screenshot.png"
#
# The release workflow calls the same script on a Windows runner and
# attaches screenshot.png to the GitHub Release.

param(
  [string]$ExePath = "build/bin/goals.exe",
  [string]$DbPath = "",
  [string]$OutFile = "screenshot.png",
  [int]$TimeoutSec = 90,
  [int]$SettleSec = 4
)

$ErrorActionPreference = 'Stop'

Add-Type @"
using System;
using System.Runtime.InteropServices;
public static class WinCap {
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
  [DllImport("user32.dll")] public static extern bool IsIconic(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hWnd, IntPtr hdcBlt, uint nFlags);
  [StructLayout(LayoutKind.Sequential)]
  public struct RECT { public int Left; public int Top; public int Right; public int Bottom; }
}
"@

$exe = (Resolve-Path $ExePath).Path

# The app is single-instance: a running GUI would swallow our launch.
# (Headless `goals.exe mcp` helpers have no window and don't hold the lock.)
$others = Get-Process goals -ErrorAction SilentlyContinue | Where-Object {
  try { $_.MainWindowHandle -ne 0 } catch { $true }
}
if ($others) {
  throw "Another goals.exe is already running (PID $($others.Id -join ',')). Close it first so the screenshot uses a clean isolated database."
}

if ($DbPath -ne '') {
  New-Item -ItemType Directory -Force (Split-Path $DbPath) | Out-Null
  $env:GOALS_DB_PATH = $DbPath
}

$p = Start-Process -FilePath $exe -PassThru
try {
  $deadline = (Get-Date).AddSeconds($TimeoutSec)
  $hWnd = [IntPtr]::Zero
  while ((Get-Date) -lt $deadline) {
    $p.Refresh()
    if ($p.HasExited) { throw "App exited early - code $($p.ExitCode). Is another instance holding the single-instance lock?" }
    if ($p.MainWindowHandle -ne [IntPtr]::Zero) { $hWnd = $p.MainWindowHandle; break }
    Start-Sleep -Milliseconds 500
  }
  if ($hWnd -eq [IntPtr]::Zero) { throw "Main window did not appear within ${TimeoutSec}s." }

  if ([WinCap]::IsIconic($hWnd)) { [WinCap]::ShowWindow($hWnd, 9) | Out-Null } # SW_RESTORE
  [WinCap]::SetForegroundWindow($hWnd) | Out-Null
  Start-Sleep -Seconds $SettleSec # let the WebView render

  $rect = New-Object WinCap+RECT
  if (-not [WinCap]::GetWindowRect($hWnd, [ref]$rect)) { throw "GetWindowRect failed." }
  $w = $rect.Right - $rect.Left
  $h = $rect.Bottom - $rect.Top
  if ($w -le 0 -or $h -le 0) { throw ("Invalid window bounds " + $w + "x" + $h + ".") }

  Add-Type -AssemblyName System.Drawing
  $bmp = New-Object System.Drawing.Bitmap($w, $h)
  try {
    # PrintWindow renders the window itself, so the shot is correct even if
    # another app (game, video) is covering it on screen.
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    try {
      $hdc = $g.GetHdc()
      try { [WinCap]::PrintWindow($hWnd, $hdc, 3) | Out-Null }
      finally { $g.ReleaseHdc($hdc) }
    }
    finally { $g.Dispose() }
    $out = Join-Path (Get-Location) $OutFile
    New-Item -ItemType Directory -Force (Split-Path $out) | Out-Null
    $bmp.Save($out, [System.Drawing.Imaging.ImageFormat]::Png)
    Write-Host ("Screenshot saved: " + $out + " (" + $w + "x" + $h + ")")
  }
  finally { $bmp.Dispose() }
}
finally {
  try {
    $p.Refresh()
    if (-not $p.HasExited) { $p.Kill(); $p.WaitForExit(5000) | Out-Null }
  } catch {}
}
