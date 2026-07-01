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

- `validate`：安装 Go、Node、Linux Wails 依赖和 Wails v3 CLI，生成 bindings，校验 `frontend/bindings` 无未提交差异，运行前端 build、`go test ./...`、RNNoise native DSP + recording runtime 定向测试、`go run ./cmd/preview-smoke`、`go run ./cmd/release-config-check`，并用 `CGO_ENABLED=1 go run -tags "gtk3 rnnoise_native" ./cmd/desktop-doctor -require-rnnoise` 阻断不能创建 RNNoise native suppressor 的构建。
- `macos-native-contract`：在 `macos-15` 且 `CGO_ENABLED=1` 下验证 CoreGraphics source enumeration 与 `CaptureService` capability 合同。
- `wails-build`：在 `windows-latest`、`macos-15`、`ubuntu-latest` 上运行带 `rnnoise_native` 的 Wails 构建；Linux 同时带 `gtk3` 标签。Windows runner 会显式准备 MinGW GCC 供 cgo 编译 RNNoise。构建后每个平台都运行 `desktop-doctor -require-rnnoise`；Windows 还会运行 `-require-video` 和 portable zip 验证。

## 当前 Release 工作流

`release.yml` 由 `v*` tag 触发。当前发布的是用于查验已完成功能和全平台编译的 preview build，不是正式签名安装包。

发布前门禁：

- `Release Gate`：生成 bindings 并校验无差异，运行前端 build、`go test ./...`、RNNoise native DSP + recording runtime 定向测试、`go run ./cmd/preview-smoke`，并运行带 `rnnoise_native` 的 `desktop-doctor -require-rnnoise`。从当前 preview 开始，发布 artifact 默认启用 RNNoise native DSP，不能再发布“按钮存在但二进制没有真降噪”的验收包。
- 三平台 build 只有在 `Release Gate` 通过后才会启动。

平台 runner：

- macOS: `macos-15`，构建 arm64，后续增加 universal。
- Windows: `windows-latest`，先构建 x64。
- Linux: `ubuntu-latest`，先构建未打包二进制，标注 experimental。

当前 preview artifact 命名：

```text
RecordingFreedom-windows-x64-vX.Y.Z-portable.zip
RecordingFreedom-macos-arm64-vX.Y.Z
RecordingFreedom-linux-x64-vX.Y.Z
SHA256SUMS-windows-x64.txt
SHA256SUMS-macos-arm64.txt
SHA256SUMS-linux-x64.txt
```

正式安装包命名仍按后续门禁推进，目标为 `.dmg`、Windows installer/portable zip、Linux AppImage/deb/rpm。

## 预览版本发布操作

在 `RecordingFreedom/` 作为新仓库根目录后执行：

```bash
git remote add origin https://github.com/lemon-casino/RecordingFreedom.git
git push -u origin main
git tag v0.1.0-preview.N
git push origin v0.1.0-preview.N
```

GitHub Actions 会自动运行 `release.yml`，通过后生成 GitHub prerelease 和三平台 preview artifacts。这个版本可用于检查：

- 胶囊 UI shell、托盘和独立设置窗口是否可启动。
- 语言切换是否在简体中文和 English 间全局生效。
- 设置、存储目录、预检、恢复扫描和能力矩阵是否能显示。
- mock `.rfrec` 录制包是否仍写入应用内部 `data/video`。
- 无 GUI preview smoke 是否能完成设置持久化、storage health、预检、mock 开始/暂停/继续/结束、manifest ready 校验和恢复扫描。
- Windows、macOS、Linux 三个平台是否能完成 Wails 构建。
- 发布二进制是否通过 `desktop-doctor -require-rnnoise`，确认 RNNoise native DSP 已真实编入 artifact。
- Windows portable zip 是否通过 `scripts/verify-windows-portable.ps1`：包含 `recordingfreedom.exe`、`tools/ffmpeg.exe`、`tools/ffprobe.exe`、`tools/THIRD_PARTY_FFMPEG.txt`、`tools/desktop-doctor.exe`、`tools/video-smoke.exe`、`tools/audio-smoke.exe` 和 `tools/run-windows-portable-smoke.ps1`，并能在 Windows runner 上执行 FFmpeg/FFprobe；校验脚本还会解析 portable 内的 smoke runner，并检查它覆盖 doctor、video-smoke、audio-smoke、FFmpeg 环境变量、区域/窗口/系统声音/麦克风/RNNoise 关键入口。
- 已发布的 Windows preview asset 可用 `scripts/verify-windows-preview-release.ps1` 下载复验：脚本会从 GitHub Release 下载 Windows portable zip 和 `SHA256SUMS-windows-x64*.txt`，校验 SHA256，再调用 `scripts/verify-windows-portable.ps1` 检查 portable zip 内容、x64 PE 头、`recordingfreedom.exe` GUI subsystem、FFmpeg/FFprobe 依赖。
- Windows artifact 是否保持 GUI subsystem 和隐藏 FFmpeg/DirectShow 子进程命令窗口的配置，避免启动软件或开始录制时弹出控制台窗口。

当前已验证的 preview 是 `v0.1.0-preview.15`。该 tag 的 Release workflow 已通过 Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release，Actions run 为 `https://github.com/lemon-casino/RecordingFreedom/actions/runs/28502127468`，发布到 `https://github.com/lemon-casino/RecordingFreedom/releases/tag/v0.1.0-preview.15`。产物包含 `RecordingFreedom-windows-x64-v0.1.0-preview.15-portable.zip`、`RecordingFreedom-macos-arm64-v0.1.0-preview.15`、`RecordingFreedom-linux-x64-v0.1.0-preview.15` 和三个平台 SHA256SUMS。`scripts/verify-windows-preview-release.ps1 -TagName v0.1.0-preview.15` 已完成真实 GitHub Release 下载复验：Windows portable zip SHA256 为 `99E1EB5C425B925F0F0269EE364C95A4F0CB7278EEE73C8E6D5A31196A8CD7DD`，`recordingfreedom.exe` 是 x64 GUI PE，FFmpeg/FFprobe 是 x64 PE 且 `-version` 可执行，`tools/desktop-doctor.exe`、`tools/video-smoke.exe` 和 `tools/audio-smoke.exe` 均为 x64 console PE，`tools/run-windows-portable-smoke.ps1` 已打入 zip。当前 `main` 的下一次 portable 校验还会解析 runner 并检查关键 smoke 命令内容。`v0.1.0-preview.7` 和 `v0.1.0-preview.8` 保留为失败记录：前者暴露 Linux Wails build tag 拼接问题，后者暴露 Windows FFmpeg bootstrap 下载链路问题；两个问题均已在 `preview.9` 前修复。`v0.1.0-preview.11` 在 `preview.10` 基础上修复 Windows 默认麦克风保留真实 WASAPI endpoint、录制中锁定来源/音频/摄像头配置，以及区域录制选区持久边框。`v0.1.0-preview.13` 修复胶囊透明背景灰底、屏幕编号标识尺寸/居中，并把区域框选后的编辑态改为透明 overlay。`v0.1.0-preview.14` 把区域录制开始后的持久边框也改为鼠标穿透透明 overlay，避免四个窄条 WebView 窗口露出浅色背景和关闭按钮，同时清理 macOS CoreAudio deprecated property element annotation。`v0.1.0-preview.15` 把 Windows clean-machine 验收工具和 runner 纳入 portable zip，并把 release/CI 门禁扩展到这些工具。

当前 preview release 必须在 release notes 中明确：macOS ScreenCaptureKit display/window/region capture 已接入代码路径但仍需真机 smoke 验收，Program/Application 当前是 queued 后续项；Windows portable zip 会携带 FFmpeg desktop writer 依赖和 clean-machine smoke runner，当前 Windows 桌面已从已发布 `v0.1.0-preview.15` portable artifact 解压运行 `tools/run-windows-portable-smoke.ps1 -Duration 3s -ContinueOnError`，12/12 step 通过，覆盖 screen/all-screens/region/window、pause/resume、系统声音、麦克风、RNNoise 和 audio-only 组合；Windows WASAPI 音频已能在停止阶段 mux 到主 `screen.mp4`，且本机 1 分钟、5 分钟和 20 分钟 smoke 已通过。外部 clean machine、长时长 artifact runner、跨平台长录同步、Linux PipeWire、目标桌面 RNNoise 实录听感仍属于后续验收；摄像头 sidecar 和 PIP 当前暂停，等视频录制和语音/音频录制验收后再恢复。不能把 mock package、未验收的 ScreenCaptureKit/FFmpeg artifact 路径或 `audio-smoke` 说成完整正式录制。

Windows preview asset 下载复验命令：

```powershell
.\scripts\verify-windows-preview-release.ps1 -TagName v0.1.0-preview.15
```

不传 `-TagName` 时会自动选择最近一个包含 Windows x64 portable zip 的已发布 release。这个脚本只证明发布 asset 完整、哈希匹配、portable zip 内容正确、Windows exe 是 x64 GUI 子系统且 FFmpeg/FFprobe 可执行；它不替代 clean-machine 真实录制 smoke。Windows portable zip 解压后，在目标 Windows 桌面执行：

```powershell
.\tools\run-windows-portable-smoke.ps1
```

该脚本会把 `RECORDINGFREEDOM_FFMPEG_PATH` 指向随包 `tools/ffmpeg.exe`，再运行 `desktop-doctor -require-video -require-rnnoise`、screen/all-screens/region/window `video-smoke`、pause/resume、系统声音/麦克风 mux 组合，以及 `audio-smoke` 的麦克风/RNNoise、系统声音和混合音频 smoke。默认输出在 portable 根目录的 `data-smoke/data/video` 下；没有窗口源、麦克风或系统播放环境时，可以用 `-SkipWindow`、`-SkipMicrophone`、`-SkipSystemAudio` 等参数做诊断，但完整验收不能跳过这些项目。2026-07-01 当前 Windows 桌面已用 `v0.1.0-preview.15` 发布 zip 运行该 runner，3 秒矩阵 12/12 通过，生成 11 个 ready `.rfrec` 包；代表性混合视频包含 H.264 video + AAC audio，混合 audio-only 包含 AAC audio。

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
- WASAPI system audio 和 microphone capture smoke test 通过。
- Media Foundation webcam smoke test 通过。
- MSVC runtime 静态链接，或随包附带并验证 DLL。
- Windows Defender/SmartScreen 风险在 release notes 中说明；后续接入代码签名。
- portable zip 解压后运行，不能依赖开发机工具链。

## Linux 发布门禁

Linux 初期为 experimental：

- 检查 xdg-desktop-portal 存在。
- 检查 PipeWire session 可用。
- 无 portal 时 UI 必须给出清晰不可用状态。
- 不承诺所有桌面环境录制一致。

## Preview Artifact 验证

当前 preview release 每个平台上传 artifact 前必须保证：

- 应用二进制存在且非空。
- 同目录生成 `SHA256SUMS-*.txt`。
- Wails build 在平台 runner 上完成。
- Release Gate 已运行 `go run ./cmd/preview-smoke`，验证当前可验收能力确实能创建 ready mock `.rfrec` 包到 `data/video`。
- Release Gate 和三平台 build job 已运行带 `rnnoise_native` 的 `CGO_ENABLED=1 go run -tags rnnoise_native ./cmd/desktop-doctor -require-rnnoise`，把 app data、`data/video`、backend、能力矩阵、RNNoise 和 Windows FFmpeg 依赖状态写入日志。Windows build job 会先运行 `scripts/ensure-windows-ffmpeg.ps1`，再用 `CGO_ENABLED=1 go run -tags rnnoise_native ./cmd/desktop-doctor -require-video -require-rnnoise` 阻断缺 FFmpeg 或缺 RNNoise 的 artifact。
- Release Gate 已运行 RNNoise native DSP + recording runtime 定向测试，验证 native wrapper 能处理 48kHz/480-sample frame，且 `recording.NativeBackendRuntime` 在 `rnnoise_native` 标签下可以编译测试。
- Release Gate 已运行 `cmd/release-config-check`，防止 CI/release workflow 意外移除 RNNoise、Windows FFmpeg、Windows portable zip 验证或 release notes 中的能力边界。
- release notes 明确该 artifact 是 UI shell / mock package 验收版本。

正式发布前还必须补齐：

- Wails runtime 资源和平台安装包结构检查。
- native helper 或平台采集模块存在性检查。
- RNNoise native DSP 已进入目标 preview/release toolchain 的 cgo 构建和 doctor 门禁；正式发布前仍需补目标桌面的 `audio-smoke -rnnoise` 实录听感与诊断验证。
- FFmpeg 或系统编码依赖策略检查。
- Windows portable zip 解压后 `recordingfreedom.exe` 能从同级 `tools/ffmpeg.exe` 解析依赖。
- Release workflow 在上传 artifact 前运行 `scripts/verify-windows-portable.ps1`，缺少 exe、FFmpeg、FFprobe 或第三方说明会直接失败；该脚本还会解压 portable zip，检查 `recordingfreedom.exe` 是 x64 GUI PE，并确认 FFmpeg/FFprobe 是 x64 PE，在 Windows host 上继续执行 `-version`。
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
