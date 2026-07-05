# RecordingFreedom App

RecordingFreedom is the new Go + React + Wails v3 application shell for the recorder rewrite.

Current state:

- Wails v3 React + TypeScript + Vite project scaffold is in place.
- The capsule recorder UI, tray entry, independent settings window, global Simplified Chinese / English language switching, screen indicator, and region selector are implemented.
- Go backend services provide app data discovery, `.rfrec` package creation, preflight checks, recovery scanning, storage health, Windows WASAPI audio capture, RNNoise dynamic native module coverage, and desktop dependency diagnostics under app-managed `data/video`.
- Native video code paths are now wired for macOS ScreenCaptureKit and Windows FFmpeg desktop capture. Windows short smoke covers screen/all-screens/region/window plus system-audio/microphone mux into `screen.mp4`; Windows camera sidecar now uses FFmpeg DirectShow and writes `webcam.mp4`, but still needs real-camera smoke and long-recording validation.
- Linux PipeWire capture, macOS/Linux camera sidecar, PIP export, signed Windows installers, macOS notarization, and Linux packages remain queued. Windows preview releases include an unsigned NSIS setup installer and a portable zip.

## Development

Install frontend dependencies:

```bash
cd frontend
npm install
```

Generate Wails bindings after changing Go services:

```bash
wails3 generate bindings -ts -i
```

Run frontend-only UI development:

```bash
cd frontend
npm run dev
```

Run the Wails app:

```bash
wails3 dev
```

Regenerate app icons from a source image:

```bash
go run ./cmd/icon-build -source "D:\图库\43574409.png" -sizes "16,24,32,48,64,128,256,512,1024"
```

Use `-skip-wails` to generate only `build/icons/icon-*.png`, `build/appicon.png`, and frontend favicon PNGs.

See [../docs/07-icon-workflow.md](../docs/07-icon-workflow.md) for the full desktop icon replacement checklist.

Build frontend assets:

```bash
cd frontend
npm run build
```

Run Go tests:

```bash
go test ./...
```

Probe desktop UI element recognition at a real point:

```powershell
# Windows PowerShell
$env:RECORDINGFREEDOM_REGION_PROBE = "cursor" # or "120,340"
go test -run "TestRegion(UIElement|Accessibility)ProbeFromEnv" -v .
go test -run "TestRegionAssistDesktopProbeFromEnv" -v .
Remove-Item Env:\RECORDINGFREEDOM_REGION_PROBE
```

```bash
# macOS, after granting Accessibility permission to the terminal running this command
RECORDINGFREEDOM_REGION_PROBE=cursor go test -run 'TestRegion(UIElement|Accessibility)ProbeFromEnv' -v .
RECORDINGFREEDOM_REGION_PROBE=cursor go test -run TestRegionAssistDesktopProbeFromEnv -v .
```

Or run the dedicated macOS verifier from the repository root:

```bash
bash scripts/verify-region-recognition-macos.sh
```

The first probe prints the native candidate chain; the assist probe verifies the full `AssistRegionSelection` path returns `source=element` and a best element bound at the pointer.
The macOS verifier requires cgo and fails if it cannot prove both an Accessibility candidate chain and `region-assist source=element`.
These probes cover the candidate chain used by region screenshot, scrolling screenshot, annotation-region, and recording-region selection.
On Windows this uses UI Automation; on macOS cgo builds use Accessibility. If macOS Accessibility permission is missing, element recognition falls back to image/window candidates instead of fabricating a match.
All four region-selection entry points share the same dynamic element candidate cache and parent-level cycling; right-click returns a manual drag/edit back to auto-recognition, and right-click in auto mode cancels selection.
Selector sessions include an `initialPointer` when the current cursor is inside the overlay, so the overlay immediately runs hover recognition before the first mouse move.
Multi-display region coordinates must use the per-display `RegionSelectionSession.displayBounds` mapping. Do not map the whole virtual desktop with a single scale ratio; mixed-DPI desktops need each display's logical bounds and capture bounds to be converted independently.

Build and test the RNNoise dynamic native module:

```bash
bash ../scripts/build-rnnoise-unix.sh --platform linux --arch x64
go test -tags rnnoise_dynamic ./internal/audio/rnnoise ./internal/recording
```

Run the no-GUI preview smoke:

```bash
go run ./cmd/preview-smoke
```

The smoke command uses a temporary data root, verifies settings persistence, storage health, preflight, mock start/pause/resume/stop, and confirms the ready `.rfrec` package is created under `data/video`. Add `-keep` to keep the generated package for inspection.

Run the no-GUI audio smoke:

```bash
go run ./cmd/audio-smoke -duration=3s -keep
```

Enable RNNoise in the audio smoke after `app/tools/` contains the platform module:

```bash
go run -tags rnnoise_dynamic ./cmd/audio-smoke -duration=3s -rnnoise -keep
```

Run the desktop dependency doctor:

```bash
go run ./cmd/desktop-doctor
```

Use `-require-video` when a machine must be able to start real screen/window capture. On Windows this requires a readable `ffmpeg.exe` in `PATH`, beside the app under `tools/`, or configured with `RECORDINGFREEDOM_FFMPEG_PATH`. Release artifacts are built with `rnnoise_dynamic`; validate the same capability with `go run -tags rnnoise_dynamic ./cmd/desktop-doctor -require-rnnoise` after building the RNNoise module.

Prepare the same Windows FFmpeg tool layout used by release builds:

```powershell
..\scripts\ensure-windows-ffmpeg.ps1
```

Verify a published Windows preview portable zip from GitHub Releases:

```powershell
..\scripts\verify-windows-preview-release.ps1 -TagName v0.1.0-preview.15
```

The release verifier downloads the Windows x64 portable zip and SHA256SUMS, checks the hash, then verifies the zip contains a x64 GUI `recordingfreedom.exe`, x64 FFmpeg/FFprobe, and the FFmpeg third-party notice. This is an artifact integrity check; real screen/region/window capture still needs the no-GUI video smoke commands below on the target desktop.

Verify a locally produced Windows setup installer:

```powershell
..\scripts\verify-windows-installer.ps1 -InstallerPath .\bin\RecordingFreedom-amd64-installer.exe
```

The setup verifier silently installs to a temporary directory, checks that the installed app includes the executable plus `tools\ffmpeg.exe`, `tools\ffprobe.exe`, `tools\rnnoise.dll`, `tools\THIRD_PARTY_FFMPEG.txt`, and an uninstaller, then removes the test install.

The current Windows portable preview includes target-machine smoke tools. After unzipping the portable zip on a Windows desktop, run:

```powershell
.\tools\run-windows-portable-smoke.ps1
```

The runner uses bundled `desktop-doctor.exe`, `video-smoke.exe`, `audio-smoke.exe`, FFmpeg, and FFprobe, and writes smoke packages under `data-smoke/data/video` unless `-DataDir` is passed.

Run the no-GUI video smoke on a machine with the required native dependency and desktop permissions:

```bash
go run ./cmd/video-smoke -duration=1m -keep
go run ./cmd/video-smoke -source-type=region -duration=1m -keep
go run ./cmd/video-smoke -source-type=window -duration=1m -keep
```

## Data Directory

All recording output must live under the managed app data structure:

```text
<RecordingFreedomAppData>/data/video/
```

In development, generated packages can be directed with:

```bash
RECORDINGFREEDOM_DATA_DIR=./data
```

The mock recorder writes `screen.mock.txt` and a `manifest.json` so UI and package handling can be verified without pretending native capture is complete. Real recording smoke commands must produce non-empty media and diagnostics before a feature is marked ready.
