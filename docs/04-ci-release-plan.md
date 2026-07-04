# 04. GitHub Actions 与全平台发布计划

## 仓库目标

后续将 `RecordingFreedom/` 作为新仓库内容推送到：

```text
https://github.com/lemon-casino/RecordingFreedom.git
```

新仓库中的 CI 文件位于：

```text
.github/workflows/ci.yml
.github/workflows/release.yml
```

当前父仓库里的旧 Electron workflows 不迁移，只参考其中 artifact 验证思想。

## 当前 CI 工作流

`ci.yml` 触发条件：

- push 到 `main`。
- pull request 到 `main`。

当前已落地 jobs：

- `validate`：安装 Go、Node、Linux Wails 依赖、Wails v3 CLI 和 Playwright Chromium，生成 bindings，校验 `frontend/bindings` 无未提交差异，运行前端 build、前端 `test:e2e`、`go test ./...`、RNNoise native DSP + recording runtime 定向测试、`go run ./cmd/preview-smoke`、`go run ./cmd/release-config-check`，并用 `CGO_ENABLED=1 go run -tags "gtk3 rnnoise_native" ./cmd/desktop-doctor -require-rnnoise` 阻断不能创建 RNNoise native suppressor 的构建。前端 e2e 覆盖普通画板、主胶囊画板入口双态规则和录制标注 overlay 命中区域，防止画板入口在录制紧凑态被隐藏或误切模式；`release-config-check` 同时检查 `12-annotation-overlay-platform-smoke.md` 的实机 overlay 验收标准，避免把导出合成 smoke 误写成透明 overlay 已完成。
- `desktop-build`：使用矩阵构建 Windows x64/arm64、macOS x64/arm64 和 Linux x64/arm64。Windows runner 通过 `msys2/setup-msys2@v2` 按架构准备 MINGW64/CLANGARM64 cgo 工具链供 RNNoise 编译，并继续运行 `desktop-doctor -require-video -require-rnnoise`；macOS/Linux runner 会下载经 SHA256 固定的对应架构 FFmpeg/FFprobe 工具，macOS 打出完整 `.app` bundle，Linux 打出可校验 portable 目录。所有桌面架构都会构建 desktop-doctor/video-smoke/audio-smoke/annotation-export-smoke/annotation-overlay-evidence-check 诊断工具。

## 当前 Release 工作流

`release.yml` 由 `v*` tag 触发。`v0.1.0` 开始作为第一版桌面 release 发布，开启 Windows x64/arm64、macOS x64/arm64 和 Linux x64/arm64 全桌面架构构建。当前仍不是签名/公证后的商业发行包：Windows 每个架构提供 portable zip 和 NSIS setup.exe，macOS 每个架构提供完整 `.app.zip`，Linux 每个架构提供 portable `.tar.gz`。

RNNoise native DSP 当前作为 release gate 覆盖 Windows x64、macOS x64/arm64 和 Linux x64/arm64。Windows ARM64 产物作为原生 ARM64 录制包发布，但不启用 `rnnoise_native` cgo tag，因为当前 Go/Wails Windows ARM64 的 `import "C"` 构建路径会在 metadata 阶段失败；该架构不能在 release notes 中宣称已具备 native RNNoise。

发布前门禁：

- `Release Gate`：生成 bindings 并校验无差异，运行前端 build、前端 `test:e2e`、`go test ./...`、RNNoise native DSP + recording runtime 定向测试、`go run ./cmd/preview-smoke`，并运行带 `rnnoise_native` 的 `desktop-doctor -require-rnnoise`。从当前 preview 开始，发布 artifact 默认启用 RNNoise native DSP，不能再发布“按钮存在但二进制没有真降噪”的验收包；画板入口也必须满足未录制打开普通画板、视频录制/暂停中打开录制标注、音频录制中仍打开普通画板的双态验收。
- 三个平台 build 只有在 `Release Gate` 通过后才会启动；任一平台 artifact 校验失败都会阻断 GitHub Release 发布。

平台 runner：

- macOS x64: `macos-15-intel`，构建 x64 `.app.zip`。
- macOS arm64: `macos-15`，构建 arm64 `.app.zip`。
- Windows x64: `windows-latest`，构建 x64 portable zip 和 NSIS setup.exe。
- Windows arm64: `windows-11-arm`，构建 arm64 portable zip 和 NSIS setup.exe。
- Linux x64: `ubuntu-latest`，构建 x64 portable `.tar.gz`。
- Linux arm64: `ubuntu-24.04-arm`，构建 arm64 portable `.tar.gz`；Linux 录制能力仍标注 experimental。

当前 release artifact 命名：

```text
RecordingFreedom-windows-x64-vX.Y.Z-portable.zip
RecordingFreedom-windows-x64-vX.Y.Z-setup.exe
RecordingFreedom-windows-arm64-vX.Y.Z-portable.zip
RecordingFreedom-windows-arm64-vX.Y.Z-setup.exe
RecordingFreedom-macos-x64-vX.Y.Z-app.zip
RecordingFreedom-macos-arm64-vX.Y.Z-app.zip
RecordingFreedom-linux-x64-vX.Y.Z-portable.tar.gz
RecordingFreedom-linux-arm64-vX.Y.Z-portable.tar.gz
SHA256SUMS-windows-x64.txt
SHA256SUMS-windows-arm64.txt
SHA256SUMS-macos-x64.txt
SHA256SUMS-macos-arm64.txt
SHA256SUMS-linux-x64.txt
SHA256SUMS-linux-arm64.txt
```

后续安装包目标仍为 `.dmg`、已签名 Windows installer/MSIX、Linux AppImage/deb/rpm。

## 预览版本发布操作

在 `RecordingFreedom/` 作为新仓库根目录后执行：

```bash
git remote add origin https://github.com/lemon-casino/RecordingFreedom.git
git push -u origin main
git tag v0.1.0-preview.N
git push origin v0.1.0-preview.N
```

GitHub Actions 会自动运行 `release.yml`，通过后生成 GitHub prerelease 和 Windows preview artifact。这个版本可用于检查：

- 胶囊 UI shell、托盘和独立设置窗口是否可启动。
- 语言切换是否在简体中文和 English 间全局生效。
- 设置、存储目录、预检、恢复扫描和能力矩阵是否能显示。
- mock `.rfrec` 录制包是否仍写入应用内部 `data/video`。
- 无 GUI preview smoke 是否能完成设置持久化、storage health、预检、mock 开始/暂停/继续/结束、manifest ready 校验和恢复扫描。
- Windows、macOS、Linux 是否都能完成 Wails 构建，并且发布产物包含对应平台的主程序、FFmpeg/FFprobe 和第三方 notices。
- 发布二进制是否通过 `desktop-doctor -require-rnnoise`，确认 RNNoise native DSP 已真实编入 artifact。
- Windows portable zip 是否通过 `scripts/verify-windows-portable.ps1`：包含 `recordingfreedom.exe`、`tools/ffmpeg.exe`、`tools/ffprobe.exe`、`tools/THIRD_PARTY_FFMPEG.txt`、`tools/THIRD_PARTY_NOTICES.txt`、`tools/desktop-doctor.exe`、`tools/video-smoke.exe`、`tools/audio-smoke.exe`、`tools/annotation-export-smoke.exe`、`tools/annotation-overlay-evidence-check.exe` 和 `tools/run-windows-portable-smoke.ps1`，并能在 Windows runner 上执行 FFmpeg/FFprobe；校验脚本还会解析 portable 内的 smoke runner，检查它覆盖 doctor、video-smoke、audio-smoke、annotation-export-smoke、annotation-overlay-evidence-check、FFmpeg 环境变量、区域/窗口 annotation target 绑定、系统声音/麦克风/RNNoise 关键入口，并确认 Excalidraw MIT 许可证声明已进入 artifact。
- 已发布的 Windows preview asset 可用 `scripts/verify-windows-preview-release.ps1` 下载复验：脚本会从 GitHub Release 下载 Windows portable zip 和 `SHA256SUMS-windows-x64*.txt`，校验 SHA256，再调用 `scripts/verify-windows-portable.ps1` 检查 portable zip 内容、x64 PE 头、`recordingfreedom.exe` GUI subsystem、FFmpeg/FFprobe 依赖。
- Windows artifact 是否保持 GUI subsystem 和隐藏 FFmpeg/DirectShow 子进程命令窗口的配置，避免启动软件或开始录制时弹出控制台窗口。
- macOS `.app.zip` 是否通过 `scripts/verify-macos-app-zip.sh`：必须包含 `RecordingFreedom.app/Contents/MacOS/recordingfreedom`、`Contents/MacOS/tools/ffmpeg`、`Contents/MacOS/tools/ffprobe`、权限 Info.plist 和 notices。
- Linux portable `.tar.gz` 是否通过 `scripts/verify-linux-portable.sh`：必须包含 `recordingfreedom`、`tools/ffmpeg`、`tools/ffprobe`、诊断工具、desktop 文件和 notices。

当前已验证的 preview 是 `v0.1.0-preview.15`。该 tag 的 Release workflow 已通过 Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release，Actions run 为 `https://github.com/lemon-casino/RecordingFreedom/actions/runs/28502127468`，发布到 `https://github.com/lemon-casino/RecordingFreedom/releases/tag/v0.1.0-preview.15`。产物包含 `RecordingFreedom-windows-x64-v0.1.0-preview.15-portable.zip`、`RecordingFreedom-macos-arm64-v0.1.0-preview.15`、`RecordingFreedom-linux-x64-v0.1.0-preview.15` 和三个平台 SHA256SUMS。`scripts/verify-windows-preview-release.ps1 -TagName v0.1.0-preview.15` 已完成真实 GitHub Release 下载复验：Windows portable zip SHA256 为 `99E1EB5C425B925F0F0269EE364C95A4F0CB7278EEE73C8E6D5A31196A8CD7DD`，`recordingfreedom.exe` 是 x64 GUI PE，FFmpeg/FFprobe 是 x64 PE 且 `-version` 可执行，`tools/desktop-doctor.exe`、`tools/video-smoke.exe` 和 `tools/audio-smoke.exe` 均为 x64 console PE，`tools/run-windows-portable-smoke.ps1` 已打入 zip。当前 `main` 的下一次 portable 校验还会解析 runner 并检查关键 smoke 命令内容。`v0.1.0-preview.7` 和 `v0.1.0-preview.8` 保留为失败记录：前者暴露 Linux Wails build tag 拼接问题，后者暴露 Windows FFmpeg bootstrap 下载链路问题；两个问题均已在 `preview.9` 前修复。`v0.1.0-preview.11` 在 `preview.10` 基础上修复 Windows 默认麦克风保留真实 WASAPI endpoint、录制中锁定来源/音频/摄像头配置，以及区域录制选区持久边框。`v0.1.0-preview.13` 修复胶囊透明背景灰底、屏幕编号标识尺寸/居中，并把区域框选后的编辑态改为透明 overlay。`v0.1.0-preview.14` 把区域录制开始后的持久边框也改为鼠标穿透透明 overlay，避免四个窄条 WebView 窗口露出浅色背景和关闭按钮，同时清理 macOS CoreAudio deprecated property element annotation。`v0.1.0-preview.15` 把 Windows clean-machine 验收工具和 runner 纳入 portable zip，并把 release/CI 门禁扩展到这些工具。

当前 preview release 必须在 release notes 中明确：macOS ScreenCaptureKit display/window/region capture 已接入代码路径但仍需真机 smoke 验收，Program/Application 当前是 queued 后续项；Windows portable zip 会携带 FFmpeg desktop writer 依赖和 clean-machine smoke runner，当前 Windows 桌面已从已发布 `v0.1.0-preview.15` portable artifact 解压运行 `tools/run-windows-portable-smoke.ps1 -Duration 3s -ContinueOnError`，12/12 step 通过，覆盖 screen/all-screens/region/window、pause/resume、系统声音、麦克风、RNNoise 和 audio-only 组合；Windows WASAPI 音频已能在停止阶段 mux 到主 `screen.mp4`，且本机 1 分钟、5 分钟和 20 分钟 smoke 已通过。外部 clean machine、长时长 artifact runner、跨平台长录同步、Linux PipeWire、目标桌面 RNNoise 实录听感仍属于后续验收；摄像头 sidecar/PIP 已完成结构化配置合同、透明 PIP 编辑 overlay、WebView 预览与 sidecar 设备最佳匹配逻辑、Windows/macOS/Linux FFmpeg sidecar writer、`ExportRecordingPackage()`、`cmd/pip-export-smoke` 和 FFmpeg PIP 导出合成，本机已用临时真实 MP4 素材跑通圆形镜像 PIP 与方形羽化 PIP 导出，并在安装输出前通过 FFmpeg 解码首个视频帧。标注导出新增 `cmd/annotation-export-smoke`，会生成真实短 MP4 + 可配置多段透明 annotation PNG + `.rfrec`，导出后逐段抽帧检查标注像素进入最终 MP4；该工具已纳入下一版 Windows portable runner，runner 默认执行 region source 的 snapshot-segments 与 window source 的 element-pngs 两条 `-duration=5s -segments=5` 验收，证明 annotation target 会跟随录制来源绑定。`cmd/annotation-overlay-evidence-check` 也会进入 Windows portable tools，用于真实 overlay smoke 完成后校验证据目录中的 `.rfrec/annotations/`、标注 PNG 和 `exports/recording.mp4`。仍不能把跨平台真实摄像头设备预览匹配效果、macOS/Linux 真机 sidecar smoke、暂停片段精确同步和长期原生采集替换说成已完成。不能把 mock package、未验收的 ScreenCaptureKit/FFmpeg artifact 路径、单机导出 smoke 或 `audio-smoke` 说成完整正式录制。

Windows preview asset 下载复验命令：

```powershell
.\scripts\verify-windows-preview-release.ps1 -TagName v0.1.0-preview.15
```

不传 `-TagName` 时会自动选择最近一个包含 Windows x64 portable zip 的已发布 release。这个脚本只证明发布 asset 完整、哈希匹配、portable zip 内容正确、Windows exe 是 x64 GUI 子系统且 FFmpeg/FFprobe 可执行；它不替代 clean-machine 真实录制 smoke。Windows portable zip 解压后，在目标 Windows 桌面执行：

```powershell
.\tools\run-windows-portable-smoke.ps1
```

该脚本会把 `RECORDINGFREEDOM_FFMPEG_PATH` 指向随包 `tools/ffmpeg.exe`，再运行 `desktop-doctor -require-video -require-rnnoise`、`annotation-export-smoke -duration=5s -segments=5 -timeline=snapshot-segments -source-type=region`、`annotation-export-smoke -duration=5s -segments=5 -timeline=element-pngs -source-type=window` 两条标注导出逐段抽帧验收、screen/all-screens/region/window `video-smoke`、pause/resume、系统声音/麦克风 mux 组合，以及 `audio-smoke` 的麦克风/RNNoise、系统声音和混合音频 smoke。默认输出在 portable 根目录的 `data-smoke/data/video` 下；没有窗口源、麦克风或系统播放环境时，可以用 `-SkipWindow`、`-SkipMicrophone`、`-SkipSystemAudio` 等参数做诊断，但完整验收不能跳过这些项目。2026-07-01 当前 Windows 桌面已用 `v0.1.0-preview.15` 发布 zip 运行该 runner，3 秒矩阵 12/12 通过，生成 11 个 ready `.rfrec` 包；代表性混合视频包含 H.264 video + AAC audio，混合 audio-only 包含 AAC audio。下一版 runner 会额外生成 region snapshot 与 window element-pngs 两个 annotation export smoke `.rfrec`，并验证最终 MP4 中 5 段标注分别在对应时间段呈现正确颜色。

长标注验收可以在同一个 portable 入口显式启用：

```powershell
.\tools\run-windows-portable-smoke.ps1 -RunAnnotationLong
```

该模式会在默认 5 秒标注 smoke 之外追加 `1m` 和 `5m` 两组 annotation export smoke，每组分别覆盖 region snapshot timeline 和 window element-pngs timeline，默认每条长时间线 60 段。它证明发布包内 `.rfrec` 标注时间线、元素级 PNG 合成和最终 MP4 抽帧验证可以跑过 1 分钟/5 分钟的可配置长时间线；它仍不是替代用户真实手绘 overlay、真实录屏输入和 macOS/Linux 透明窗口穿透的真机验收。

真实录制标注 overlay 的平台验收必须按 `12-annotation-overlay-platform-smoke.md` 执行并保存 evidence。Windows portable 包会携带 `tools/annotation-overlay-evidence-check.exe`，可在目标机器用它复查 evidence 目录里的真实 `.rfrec/annotations/`、`diagnostics.sync.screen.durationMs`、1 分钟/5 分钟 ready 包、all-screens / screen / region / window 来源矩阵、元素级标注事件、绘制/穿透 hit-region 诊断、`annotations/overlay-diagnostics.jsonl`、标注 PNG 和最终 `exports/recording.mp4`；该工具只检查证据结构和诊断 JSONL，不能替代人工确认点击穿透、鼠标闪烁和多屏坐标。只有在对应平台完成全屏、单屏、区域、锁定窗口、多屏/高 DPI、绘制态、穿透态、真实 `.rfrec/annotations/`、overlay diagnostics 和最终 `exports/recording.mp4` 验收后，release notes 才能把该平台的录制标注 overlay 写为完成。

`preview`、`alpha`、`beta`、`rc` 标签会被 workflow 自动标记为 GitHub prerelease；正式稳定版本再移除这些后缀。

## macOS 发布门禁

正式公开发布前必须满足：

- Developer ID Application 签名。
- notarization。
- staple。
- Info.plist 包含 Screen Recording、Microphone、Camera 权限说明。
- 启动后权限探测使用真实 capture capability probe，而不是只相信状态字符串。
- 验证 `data/video` 可写，不写入 `.app` bundle。

需要的 secrets：

- `APPLE_DEVELOPER_ID_CERTIFICATE_BASE64`
- `APPLE_DEVELOPER_ID_CERTIFICATE_PASSWORD`
- `APPLE_ID`
- `APPLE_APP_SPECIFIC_PASSWORD`
- `APPLE_TEAM_ID`

## Windows 发布门禁

正式公开发布前必须满足：

- Windows FFmpeg desktop writer 能检测 ffmpeg，缺失时 preflight blocked，存在时能启动 video-smoke。
- FFmpeg 二进制来源、SHA256 校验、许可证文本和再分发义务在 release notes 或第三方 notices 中明确；当前 preview 通过 `scripts/ensure-windows-ffmpeg.ps1` 下载 BtbN FFmpeg-Builds static Windows zip，按 `checksums.sha256` 校验，并生成 `tools/THIRD_PARTY_FFMPEG.txt`。
- Excalidraw 等嵌入式前端组件的许可证文本必须进入 `tools/THIRD_PARTY_NOTICES.txt`，Windows portable zip 和安装包 verifier 都要阻断缺失或内容不完整的 artifact。
- WASAPI system audio 和 microphone capture smoke test 通过。
- 摄像头 sidecar smoke test 通过；Windows 当前使用 FFmpeg DirectShow，macOS 当前使用 FFmpeg AVFoundation，Linux 当前使用 FFmpeg v4l2。后续如替换为 Media Foundation/AVFoundation/PipeWire 原生 writer，仍必须复用同一 `.rfrec` 和 PIP 导出合同。
- MSVC runtime 静态链接，或随包附带并验证 DLL。
- Windows Defender/SmartScreen 风险在 release notes 中说明；后续接入代码签名。
- portable zip 解压后运行，不能依赖开发机工具链。

## Linux 发布门禁

Linux 初期为 experimental：

- 检查 xdg-desktop-portal 存在。
- 检查 PipeWire session 可用。
- 无 portal 时 UI 必须给出清晰不可用状态。
- 不承诺所有桌面环境录制一致。

## Release Artifact 验证

当前 release 每个平台上传 artifact 前必须保证：

- 应用二进制存在且非空。
- 同目录生成 `SHA256SUMS-*.txt`。
- Wails build 在平台 runner 上完成。
- Release Gate 已运行 `go run ./cmd/preview-smoke`，验证当前可验收能力确实能创建 ready mock `.rfrec` 包到 `data/video`。
- Release Gate 和三平台 build job 已运行带 `rnnoise_native` 的 `desktop-doctor -require-rnnoise`，把 app data、`data/video`、backend、能力矩阵、RNNoise 和平台 FFmpeg 依赖状态写入日志。Windows build job 会先运行 `scripts/ensure-windows-ffmpeg.ps1`，再用 `desktop-doctor -require-video -require-rnnoise` 阻断缺 FFmpeg 或缺 RNNoise 的 artifact；macOS/Linux 会先运行 `scripts/ensure-unix-ffmpeg.sh`，再分别校验 `.app.zip` 和 portable `.tar.gz`。
- Release Gate 已运行 RNNoise native DSP + recording runtime 定向测试，验证 native wrapper 能处理 48kHz/480-sample frame，且 `recording.NativeBackendRuntime` 在 `rnnoise_native` 标签下可以编译测试。
- Release Gate 已运行 `cmd/release-config-check`，防止 CI/release workflow 意外移除 RNNoise、Windows FFmpeg、Windows portable zip 验证或 release notes 中的能力边界。
- release notes 明确三端产物边界：这是第一版桌面 release，但尚未完成 Windows 签名、macOS notarization 和 Linux deb/rpm/AppImage。

正式发布前还必须补齐：

- Wails runtime 资源和平台安装包结构检查。
- native helper 或平台采集模块存在性检查。
- RNNoise native DSP 已进入目标 preview/release toolchain 的 cgo 构建和 doctor 门禁；正式发布前仍需补目标桌面的 `audio-smoke -rnnoise` 实录听感与诊断验证。
- FFmpeg 或系统编码依赖策略检查；PIP 导出和当前三平台摄像头 sidecar 均依赖 FFmpeg，可用性必须进入 capability、doctor 和 release notes。
- Windows portable zip 解压后 `recordingfreedom.exe` 能从同级 `tools/ffmpeg.exe` 解析依赖。
- Release workflow 在上传 artifact 前运行 `scripts/verify-windows-portable.ps1`，缺少 exe、FFmpeg、FFprobe、FFmpeg 第三方说明或 Excalidraw MIT notices 会直接失败；该脚本还会解压 portable zip，按矩阵架构检查 `recordingfreedom.exe` 是 x64 或 ARM64 GUI PE，并确认 FFmpeg/FFprobe 与诊断工具架构一致，在 Windows host 上继续执行 `-version`。
- Release workflow 在上传 macOS artifact 前运行 `scripts/verify-macos-app-zip.sh`，缺少 `.app`、`Contents/MacOS/recordingfreedom`、`Contents/MacOS/tools/ffmpeg`、`Contents/MacOS/tools/ffprobe`、Info.plist 或 notices 会直接失败。
- Release workflow 在上传 Linux artifact 前运行 `scripts/verify-linux-portable.sh`，缺少主程序、FFmpeg/FFprobe、诊断工具、desktop 文件或 notices 会直接失败，并检查主程序、FFmpeg 和 FFprobe 是矩阵指定的 x64 或 ARM64 ELF。
- Release 发布后可运行 `scripts/verify-windows-preview-release.ps1` 对 GitHub Release asset 做下载级复验，覆盖 SHA256SUMS 和 portable zip 结构。
- 正式安装包环境中的 GUI/进程级 smoke test。
- signed/notarized/package 后的 mock `.rfrec/manifest.json` 创建 smoke test。

失败时上传 diagnostics artifact：

```text
build/
dist/
logs/
native/*/build/
data/video/*/manifest.json
```

## 发布策略

- `v0.1.0`：UI Shell + mock recording package + CI。
- `v0.2.0`：macOS ScreenCaptureKit 基础录制。
- `v0.3.0`：Windows FFmpeg desktop 基础录制与后续原生 WGC/Media Foundation 替换评估。
- `v0.4.0`：麦克风 RNNoise 降噪和系统声音/麦克风混音稳定。
- `v0.5.0`：摄像头 sidecar + PIP 预览。
- `v1.0.0`：macOS/Windows 正式发布门禁全部满足，Linux 仍可标 experimental。
