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

- `validate`：安装 Go、Node、Linux Wails 依赖和 Wails v3 CLI，生成 bindings，校验 `frontend/bindings` 无未提交差异，运行前端 build、`go test ./...` 和 `go run ./cmd/preview-smoke`。
- `macos-native-contract`：在 `macos-15` 且 `CGO_ENABLED=1` 下验证 CoreGraphics source enumeration 与 `CaptureService` capability 合同。
- `wails-build`：在 `windows-latest`、`macos-15`、`ubuntu-latest` 上运行默认 `wails3 build`，并上传三平台 preview artifact。RNNoise 原生 DSP 由单独 contract 验证，直到完整 app recording backend 接入前不强制编入 preview artifact。

## 当前 Release 工作流

`release.yml` 由 `v*` tag 触发。当前发布的是用于查验已完成功能和全平台编译的 preview build，不是正式签名安装包。

发布前门禁：

- `Release Gate`：生成 bindings 并校验无差异，运行前端 build、`go test ./...` 和 `go run ./cmd/preview-smoke`。
- 三平台 build 只有在 `Release Gate` 通过后才会启动。

平台 runner：

- macOS: `macos-15`，构建 arm64，后续增加 universal。
- Windows: `windows-latest`，先构建 x64。
- Linux: `ubuntu-latest`，先构建未打包二进制，标注 experimental。

当前 preview artifact 命名：

```text
RecordingFreedom-windows-x64-vX.Y.Z.exe
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
git tag v0.1.0-preview.5
git push origin v0.1.0-preview.5
```

GitHub Actions 会自动运行 `release.yml`，通过后生成 GitHub prerelease 和三平台 preview artifacts。这个版本可用于检查：

- 胶囊 UI shell、托盘和独立设置窗口是否可启动。
- 语言切换是否在简体中文和 English 间全局生效。
- 设置、存储目录、预检、恢复扫描和能力矩阵是否能显示。
- mock `.rfrec` 录制包是否仍写入应用内部 `data/video`。
- 无 GUI preview smoke 是否能完成设置持久化、storage health、预检、mock 开始/暂停/继续/结束、manifest ready 校验和恢复扫描。
- Windows、macOS、Linux 三个平台是否能完成 Wails 构建。

当前 preview release 必须在 release notes 中明确：真实 ScreenCaptureKit/WGC/PipeWire 录制、完整 app recording backend 音频接入、摄像头 sidecar 和 PIP 导出仍属于后续里程碑，不能把 mock package 或 `audio-smoke` 说成完整真实录制。

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

- WGC helper 存在并能启动。
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
- release notes 明确该 artifact 是 UI shell / mock package 验收版本。

正式发布前还必须补齐：

- Wails runtime 资源和平台安装包结构检查。
- native helper 或平台采集模块存在性检查。
- RNNoise native DSP 在目标 release toolchain 中的 cgo 编译与 frame 处理验证。
- FFmpeg 或系统编码依赖策略检查。
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
- `v0.3.0`：Windows WGC 基础录制。
- `v0.4.0`：麦克风 RNNoise 降噪和系统声音/麦克风混音稳定。
- `v0.5.0`：摄像头 sidecar + PIP 预览。
- `v1.0.0`：macOS/Windows 正式发布门禁全部满足，Linux 仍可标 experimental。
