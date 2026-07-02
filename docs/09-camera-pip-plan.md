# 09. 摄像头 Sidecar 与画中画计划

本文档把摄像头 sidecar / 画中画从暂停项恢复为当前开发任务。目标是录制视频时可以同时采集摄像头设备，并在后续导出阶段按用户设置合成画中画；原始屏幕视频仍保持干净，不把 PIP 直接烘焙进 `screen.mp4`。

## 需求范围

- 录制视频时可开启摄像头 sidecar，摄像头媒体写入 `.rfrec` 包内 `webcam.mp4` 或 `webcam.mov`。
- 画中画支持圆形和方形样式。
- 画中画边缘支持透明羽化，避免硬边遮挡屏幕内容。
- 画中画支持放大、缩小、拖动位置和镜像摄像头。
- 交互形态类似区域录制选择框：透明覆盖层、可移动、可缩放、录制中不影响屏幕采集控制。
- PIP 配置必须写入 settings、start request、manifest 和 export plan，不能只存在前端临时状态。

## 架构原则

- 摄像头 sidecar 与屏幕录制分离：`screen.mp4` 保存原始屏幕，`webcam.*` 保存摄像头。
- PIP 是布局与导出合同：manifest 保存 `camera.pip`，导出器读取 `screen.mp4 + webcam.* + camera.pip` 合成最终文件。
- 录制中 PIP overlay 只负责预览和编辑布局；它不能阻塞鼠标录制、区域边框、暂停/继续/结束。
- 旧包兼容：保留 `camera.pipPreset`，新增结构化 `camera.pip`，旧设置和旧 manifest 自动归一化。

## 当前已落地

- `internal/pip` 新增结构化 `Config`：`preset`、`shape`、`mirror`、`position`、`scale` 和 `edgeFeather`。
- settings / recording request / `.rfrec` manifest / export plan 已保存和传递 `camera.pip`。
- 前端摄像头面板新增：PIP 位置、圆形/方形、镜像、大小、透明边缘控制和 PIP 预览。
- 前端摄像头面板新增“编辑画中画”入口；摄像头启用且未录制时会打开透明置顶 `/#/pip-overlay` 辅助窗口。
- PIP overlay 已支持中心拖动、八方向缩放、圆形/方形切换、镜像切换和关闭 PIP。拖动过程中只预览移动窗口，松手后才写回配置，避免高频写 manifest。
- 开始视频录制且摄像头开启时，后端会自动显示录制态 PIP overlay；停止录制后自动隐藏。
- `RecordingFreedomService.ShowPIPOverlay()` / `UpdatePIPOverlay()` / `HidePIPOverlay()` 已暴露给前端。`UpdatePIPOverlay()` 会持久化 `settings.camera.pip`，并在存在活动视频录制且摄像头开启时 patch 当前 `.rfrec/manifest.json`。
- PIP overlay 使用 Wails `SetContentProtection()` 尝试排除窗口捕获：Windows 走 `SetWindowDisplayAffinity`，macOS 走 `NSWindowSharingNone`，Linux 当前返回不支持。目标仍是保持原始 `screen.mp4` 干净，最终 PIP 由导出阶段合成。
- PIP overlay 前端会尝试用 `navigator.mediaDevices.getUserMedia()` 显示实时摄像头预览；拿不到权限或运行时不支持时显示明确的摄像头预览不可用状态，不伪造真实视频。
- PIP overlay 请求现在会携带所选摄像头的 `deviceId`、平台 `nativeId` 和显示名。WebView 预览会先获取权限，再枚举 `videoinput`，按显示名/native id/device id 尽量选择与 sidecar writer 相同的摄像头；匹配失败时才回退默认摄像头，并保留明确日志。
- Windows 摄像头 sidecar writer 已使用 FFmpeg DirectShow 采集包内 `webcam.mp4`。
- macOS 摄像头 sidecar writer 已使用 FFmpeg AVFoundation 采集包内 `webcam.mov`，并通过 FFmpeg AVFoundation 设备列表解析真实摄像头索引和名称。
- Linux 摄像头 sidecar writer 已使用 FFmpeg v4l2 采集包内 `webcam.mov`，并从 `/dev/video*` 枚举可用摄像头。Linux 仍按 experimental 验收。
- `CaptureService` 中 `camera-sidecar` 和 `pip-export` 会根据 FFmpeg 可用性返回 `available` 或 `blocked`，不再把已接入的 PIP 导出显示为 queued。
- `internal/exporter` 已新增真实 FFmpeg PIP 合成器：默认输出包内 `exports/recording.mp4`，输入为干净的 `screen.mp4` 和 `webcam.*` sidecar，支持圆形/方形、镜像、边缘羽化、位置、大小和 `webcamStartOffsetMs`。
- 导出器会先写临时 MP4，确认文件可读且用 FFmpeg 解码首个视频帧成功后，才原子安装到 `exports/recording.mp4`；`cmd/pip-export-smoke` 会在 JSON 中返回 `outputVerified: true`。
- Wails 服务已新增 `ExportRecordingPackage()`，设置面板可对最近录制包执行导出。导出会拒绝覆盖原始 `screen.mp4` 或 `webcam.*`。
- 新增 `cmd/pip-export-smoke`，可用现有 `.rfrec` 包执行无 UI 导出验收。
- Wails bindings 已同步生成 `internal/pip` TypeScript 模型。
- Go 测试覆盖旧 preset 兼容、新 PIP 布局归一化、manifest 写入、录制中 PIP manifest patch、export plan 读取、FFmpeg 合成参数、临时文件安装和 AVFoundation 设备解析。
- 本机已用内置 `app/tools/ffmpeg.exe` 生成临时 `.rfrec` 包并执行 `go run ./cmd/pip-export-smoke`，圆形镜像 PIP 与方形羽化 PIP 真实导出成功，输出 `exports/recording.mp4` 分别为 136,536 bytes 和 134,260 bytes。

## 下一步任务

1. 摄像头实时预览与设备匹配验收
   - Windows 验证 DirectShow sidecar 设备与 WebView `getUserMedia()` 匹配后的预览是否稳定对应；必要时增加原生低延迟抓帧预览。
   - macOS 验证 AVFoundation sidecar、权限提示、WebView 预览匹配和实际写入 `webcam.mov`。
   - Linux 验证 v4l2 sidecar 与 WebView 预览匹配；后续可替换为 PipeWire 摄像头路径，保持 experimental 标记直到真机 smoke 通过。

2. 导出合成增强
   - 当前已按 `webcamStartOffsetMs` 对齐 sidecar；下一步补暂停片段的精确时间线压缩和更完整音画同步回归。
   - 增加真实录制包导出 smoke：screen + camera、screen + camera + system audio、screen + camera + microphone。
   - 首帧解码门禁已接入；下一步扩展为 ffprobe 级输出视频轨、音轨和时长验证。

3. 验收矩阵
   - Windows：screen + camera sidecar，screen + camera + system audio + microphone，pause/resume，长录。
   - macOS：screen/window/region + camera sidecar，权限提示，音画同步。
   - Linux：PipeWire experimental smoke。
   - 导出：圆形、方形、镜像、透明边缘、拖动位置、不同尺寸均正确合成。

## 非目标

- 不在原始 `screen.mp4` 中直接烘焙 PIP。
- 不把摄像头失败的录制包标记为 ready。
- 不用假摄像头设备或假媒体冒充真实 sidecar。
- 不把当前 FFmpeg sidecar 实现描述为最终原生采集方案；它是可发布、可验收、可替换的长期接口实现，后续可在不改 manifest/export 合同的前提下替换为 AVFoundation/Media Foundation/PipeWire 原生 writer。
