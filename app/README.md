# RecordingFreedom App

RecordingFreedom is the new Go + React + Wails v3 application shell for the recorder rewrite.

Current state:

- Wails v3 React + TypeScript + Vite project scaffold is in place.
- The first capsule recorder UI shell is implemented with mock state.
- Go backend services provide app data discovery and mock `.rfrec` package creation under `data/video`.
- Native recording backends are intentionally not implemented in this milestone.

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

Build frontend assets:

```bash
cd frontend
npm run build
```

Run Go tests:

```bash
go test ./...
```

Run the no-GUI preview smoke:

```bash
go run ./cmd/preview-smoke
```

The smoke command uses a temporary data root, verifies settings persistence, storage health, preflight, mock start/pause/resume/stop, and confirms the ready `.rfrec` package is created under `data/video`. Add `-keep` to keep the generated package for inspection.

## Data Directory

All recording output must live under the managed app data structure:

```text
<RecordingFreedomAppData>/data/video/
```

In development, generated mock packages can be directed with:

```bash
RECORDINGFREEDOM_DATA_DIR=./data
```

The mock recorder writes `screen.mock.txt` and a `manifest.json` so UI and package handling can be verified without pretending native capture is complete.
