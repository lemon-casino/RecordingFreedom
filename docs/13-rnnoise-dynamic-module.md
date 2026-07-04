# 13. RNNoise 动态原生模块方案

## 目标

RNNoise 不再作为 Go/Wails 主程序里的 cgo 静态编译代码发布。当前标准是：

```text
RNNoise C source
  -> GitHub Actions 按平台/架构编译动态库
  -> 动态库进入对应安装包/便携包
  -> Go/Wails 主程序使用 rnnoise_dynamic 标签
  -> 运行时从 tools/ 加载原生模块
```

这样 Windows ARM64 不再依赖 Go/Wails 主二进制的 `import "C"` 构建路径，同时 Windows x64、macOS x64/arm64、Linux x64/arm64 也走同一套模块边界。

## 平台产物

| 平台 | 模块文件 | 包内位置 |
| --- | --- | --- |
| Windows x64 | `rnnoise.dll` | `tools/rnnoise.dll` |
| Windows ARM64 | `rnnoise.dll` | `tools/rnnoise.dll` |
| macOS x64 | `librnnoise.dylib` | `RecordingFreedom.app/Contents/MacOS/tools/librnnoise.dylib` |
| macOS ARM64 | `librnnoise.dylib` | `RecordingFreedom.app/Contents/MacOS/tools/librnnoise.dylib` |
| Linux x64 | `librnnoise.so` | `tools/librnnoise.so` |
| Linux ARM64 | `librnnoise.so` | `tools/librnnoise.so` |

运行时也支持 `RECORDINGFREEDOM_RNNOISE_PATH` 指向指定模块，主要用于本机诊断。

## 代码边界

- `app/internal/audio/rnnoise/native/` 保留 RNNoise C 源码和 `likely_voice_enhancer_*` C ABI。
- `rnnoise_dynamic` 构建标签启用动态模块 provider。
- Windows provider 使用 `LoadLibrary` / `GetProcAddress`，主程序可以保持 `CGO_ENABLED=0`。
- macOS/Linux provider 使用 `dlopen` / `dlsym`，对应 Wails 平台本来就需要 `CGO_ENABLED=1`。
- `audio.Enhancer` 和 `recording.NativeBackendRuntime` 继续只依赖 `audio.NoiseSuppressor`，不会关心模块来自 DLL、dylib 还是 so。

## CI / Release 门禁

Release 和 CI 必须执行：

```bash
bash ./scripts/build-rnnoise-unix.sh --platform linux --arch x64 --force
go test -tags rnnoise_dynamic ./internal/audio/rnnoise ./internal/recording
```

平台 build job 必须先构建当前架构的动态模块：

```powershell
.\scripts\build-rnnoise-windows.ps1 -Architecture x64
.\scripts\build-rnnoise-windows.ps1 -Architecture arm64
```

```bash
bash ./scripts/build-rnnoise-unix.sh --platform macos --arch x64
bash ./scripts/build-rnnoise-unix.sh --platform macos --arch arm64
bash ./scripts/build-rnnoise-unix.sh --platform linux --arch x64
bash ./scripts/build-rnnoise-unix.sh --platform linux --arch arm64
```

每个桌面 artifact 都必须通过：

```bash
go run -tags rnnoise_dynamic ./cmd/desktop-doctor -require-rnnoise
go test -tags rnnoise_dynamic ./internal/audio/rnnoise
```

Linux/macOS 命令需要 `CGO_ENABLED=1`；Windows x64/ARM64 使用 `CGO_ENABLED=0`。

## 打包校验

- `scripts/verify-windows-portable.ps1` 检查 `tools/rnnoise.dll` 存在且 PE 架构匹配。
- `scripts/verify-windows-installer.ps1` 检查安装目录包含 `tools/rnnoise.dll`。
- `scripts/verify-macos-app-zip.sh` 检查 `.app` 内含 `Contents/MacOS/tools/librnnoise.dylib` 且架构匹配。
- `scripts/verify-linux-portable.sh` 检查 `tools/librnnoise.so` 存在且 ELF 架构匹配。
- `cmd/release-config-check` 固定上述 workflow、脚本、release notes 和 verifier 文本，防止回退为“某架构无 RNNoise”。

## 验收标准

一个平台只有同时满足以下条件，才能在 release notes 中宣称 RNNoise 可用：

1. 对应架构动态库由 CI 从仓库内 RNNoise C 源编译产生。
2. 动态库被打入该平台 artifact。
3. artifact verifier 检查动态库存在且架构正确。
4. `desktop-doctor -require-rnnoise` 在该平台 build job 通过。
5. `go test -tags rnnoise_dynamic ./internal/audio/rnnoise` 在该平台 build job 通过一帧真实处理。
6. 目标桌面 `audio-smoke -rnnoise` 或 portable smoke 能产生带 RNNoise 诊断的 ready `.rfrec`。

Windows ARM64 从该方案开始不再是 RNNoise 例外架构。
