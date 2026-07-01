# RecordingFreedom App

RecordingFreedom is the new Go + React + Wails v3 application shell for the recorder rewrite.

Current state:

- Wails v3 React + TypeScript + Vite project scaffold is in place.
- The capsule recorder UI, tray entry, independent settings window, global Simplified Chinese / English language switching, screen indicator, and region selector are implemented.
- Go backend services provide app data discovery, `.rfrec` package creation, preflight checks, recovery scanning, storage health, Windows WASAPI audio capture, RNNoise native DSP build coverage, and desktop dependency diagnostics under app-managed `data/video`.
- Native video code paths are now wired for macOS ScreenCaptureKit and Windows FFmpeg desktop capture. Windows short smoke covers screen/all-screens/region/window plus system-audio/microphone mux into `screen.mp4`; Windows camera sidecar now uses FFmpeg DirectShow and writes `webcam.mp4`, but still needs real-camera smoke and long-recording validation.
- Linux PipeWire capture, macOS/Linux camera sidecar, PIP export, signed installers, and notarization remain queued.

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

Run RNNoise native tests on a machine with cgo and a C compiler:

```bash
CGO_ENABLED=1 go test -tags rnnoise_native ./internal/audio/rnnoise/native ./internal/recording
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

Enable RNNoise in the audio smoke only on machines with cgo and a C compiler:

```bash
CGO_ENABLED=1 go run -tags rnnoise_native ./cmd/audio-smoke -duration=3s -rnnoise -keep
```

Run the desktop dependency doctor:

```bash
go run ./cmd/desktop-doctor
```

Use `-require-video` when a machine must be able to start real screen/window capture. On Windows this requires a readable `ffmpeg.exe` in `PATH`, beside the app under `tools/`, or configured with `RECORDINGFREEDOM_FFMPEG_PATH`. Release artifacts are built with `rnnoise_native`; validate the same capability with `CGO_ENABLED=1 go run -tags rnnoise_native ./cmd/desktop-doctor -require-rnnoise` on machines with a C toolchain.

Prepare the same Windows FFmpeg tool layout used by release builds:

```powershell
..\scripts\ensure-windows-ffmpeg.ps1
```

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
