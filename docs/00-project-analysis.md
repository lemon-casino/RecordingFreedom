# 00. 当前项目架构识别与重构取舍

## 当前 LikelySnap 架构事实

当前项目是 Electron + Vite + React/TypeScript 桌面录制和轻量剪辑工具。主要结构如下：

- Electron 主进程负责生命周期、托盘、窗口、权限、文件系统和 IPC。
- React 渲染层负责 HUD、源选择、设置、编辑器、导出面板和大量录制编排。
- macOS 原生录制通过 ScreenCaptureKit helper 完成屏幕和音频写入。
- Windows 首版视频录制通过 FFmpeg gdigrab writer 完成，音频通过 WASAPI；WGC / Media Foundation 保留为后续原生替换方向。
- Linux 或 fallback 路径依赖 Electron/Chromium 捕获和 MediaRecorder。
- 长录制方向已经转向持续写盘、录制包、相对 manifest、cursor preview、native webcam sidecar 和 FFmpeg 流式导出。

## 可以吸收的优点

RecordingFreedom 必须继承这些经过验证的产品与工程经验：

- **录制中持续写盘**：长录制不能等停止时才整体保存。
- **录制包目录模型**：目录包比 zip 或单二进制容器更适合持续写盘和崩溃恢复。
- **相对路径 manifest**：录制包移动后仍能打开。
- **可恢复状态机**：`recording`、`finalizing`、`ready`、`recoverable`、`failed` 必须清晰。
- **音画同步诊断**：记录视频、系统声音、麦克风、摄像头的起止时间和 offset。
- **麦克风原生降噪**：旧项目的 `LikelyVoiceEnhancement` 使用 RNNoise、VAD、speech gain smoothing、low-VAD attenuation 和 limiter，只处理麦克风。
- **摄像头 sidecar**：摄像头应作为独立文件录制，并用 offset 对齐屏幕时间线，方便后续画中画。
- **FFmpeg/原生编码思路**：长视频导出不应该依赖最终内存 Blob。
- **设置独立窗口**：透明 HUD 内嵌复杂设置容易有点击和裁剪问题，设置应该是普通独立窗口。

## 必须替换的旧债

RecordingFreedom 是全新项目，不迁移这些历史包袱：

- 不迁移 Electron 主进程、preload、IPC 和 BrowserWindow 架构。
- 不把浏览器 `MediaRecorder` 作为核心录制路径；它只能作为低优先 fallback。
- 不迁移旧黑玻璃 + 玫红主题；Apowersoft 和当前 HUD 只作为布局参考。
- 不把录制目录散落在用户目录、缓存目录和历史兼容路径里；新视频统一进入应用托管的 `data/video/`。
- 不保留旧 `.likelysnap` 兼容加载作为 v1 必需项；新项目使用 `.rfrec` 录制包。
- 不把摄像头直接烘焙进屏幕视频；画中画是编辑/导出层能力。
- 不继续将长录制编辑器打开建立在整文件读取、全量波形和全量索引阻塞上。

## 新项目目标

RecordingFreedom 的第一目标不是复制旧编辑器，而是先做稳定、漂亮、可持续演进的录制工具：

- 胶囊托盘窗口可以完成日常录制入口操作。
- 支持屏幕识别、窗口识别、程序识别。
- 支持系统声音、麦克风、麦克风降噪和设备选择。
- 支持摄像头选择，并为后续画中画预留数据模型。
- 支持开始、暂停、继续、结束录制。
- 支持语言切换和设置。
- 支持全平台发布路线与 GitHub Actions。

## 技术替换原则

- UI 壳使用 Wails v3，不再使用 Electron。
- 业务后端使用 Go 服务，前端只做状态呈现和命令调用。
- 平台录制优先使用稳定可维护方案：macOS ScreenCaptureKit，Windows FFmpeg gdigrab 首版落地并保留 WGC/Media Foundation 替换方向，Linux XDG ScreenCast Portal + PipeWire。
- 音频处理放在 Go/native 层，前端不承担实时降噪、混音和 mux 决策。
- 所有录制产物必须先写入 `data/video/` 下的录制包，再由导出功能产生发布文件。
