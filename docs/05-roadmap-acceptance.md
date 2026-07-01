# 05. 开发路线与验收清单

## M0 文档与脚手架

目标：

- 文档落地。
- 创建 Wails v3 + React + Go 项目骨架。
- 建立 `data/video` 应用数据目录。
- 建立 mock backend，保证 UI 可独立开发。

验收：

- `RecordingFreedom/docs/` 文档齐全。
- `RecordingFreedom/app` 可启动。
- 设置服务能返回默认数据目录和默认录制设置。
- `<AppData>/settings.json` 可保存语言、音频、摄像头和最近源选择。
- mock recording 可以在 `data/video` 下创建 `.rfrec/manifest.json`。
- Bootstrap 和 preflight 能返回 `data/video` 可写性与可用空间健康状态。

## M1 胶囊 UI Shell

目标：

- 完成时尚胶囊录制工具窗口。
- 完成设置窗口。
- 完成源、音频、摄像头、语言菜单。
- 完成录制状态机 UI。

验收：

- 未录制、准备中、录制中、暂停中、停止中、失败状态都有可视反馈。
- 开始、暂停、继续、结束按钮状态正确。
- 菜单不裁切、不重叠。
- 语言、音频、摄像头和最近源选择重启后可恢复；语言在胶囊窗口和独立设置窗口切换后都要全局生效。
- 支持横向和竖向胶囊。
- Playwright 截图覆盖常见桌面尺寸。

## M2 设备和源枚举

目标：

- 接入真实屏幕、窗口、程序枚举。
- 接入真实麦克风和摄像头枚举。
- 权限状态进入 UI。

验收：

- macOS 能通过 CoreGraphics 列出屏幕和窗口，后续录制阶段再映射到 ScreenCaptureKit target。
- Windows 能列出屏幕和窗口。
- 程序视图能按应用分组显示窗口。
- 媒体设备枚举通过 `MediaDeviceProvider` 接入，不让 UI 直接依赖平台 API。
- 麦克风和摄像头设备名称稳定显示。
- 无权限时 UI 给出明确动作，不出现空列表假成功。

## M3 macOS 录制

状态：ScreenCaptureKit display/window/region `screen.mp4` 写盘代码已接入，ScreenCaptureKit system audio 已接入同容器 AAC mux 代码路径，均待 macOS 真机录制 smoke 验收；Program/Application 当前为 queued 后续项，不作为初版验收完成项；麦克风和完整音画同步仍是本里程碑后续任务。

目标：

- 使用 `recording.CreateNativeWritePlan()` 初始化 `.rfrec` 写盘计划。
- 接入 ScreenCaptureKit 基础录制。
- 支持屏幕/窗口录制。
- 支持系统声音、麦克风、暂停、停止。
- 写入 `data/video/*.rfrec`。

验收：

- 1 分钟、5 分钟、20 分钟录制均可停止并播放。
- `screen.mp4` 录制中持续增长。
- native package 初始化后不会创建假 `screen.mp4`；只有真实 writer 采样后才出现媒体文件。
- 停止时如果 `screen.mp4` 缺失或为 0 字节，manifest 必须进入 `failed`，不能写成 `ready`。
- `manifest.json` 状态从 `recording` 到 `ready`。
- raw MP4 音画同步可接受，并有诊断文件。
- kill app 后重启可识别 recoverable 包。

## M4 Windows 录制

状态：FFmpeg gdigrab writer、WASAPI 音频采集、停止阶段音视频 mux、Windows FFmpeg dependency bootstrap 和 portable zip 打包路径已接入。本机 smoke 已通过 screen、all-screens、region、locked-window、pause/resume segment merge、系统声音 mux、麦克风 mux、系统声音 + 麦克风混音 mux，以及 1 分钟、5 分钟和 20 分钟可解码长录；仍需 release artifact 解压后的 clean machine 验证。

目标：

- 接入 Windows FFmpeg gdigrab writer，并保留 Windows.Graphics.Capture / Media Foundation 原生替换路线。
- 接入 WASAPI 系统声音和麦克风。
- 写入统一 `.rfrec` 包。

验收：

- Windows x64 可以录屏幕、区域和锁定窗口。
- 系统声音、麦克风分别可录。
- 暂停和停止不会丢失文件。
- clean machine 验证不依赖开发环境。
- 低空间或不可写的 `data/video` 会在 preflight 中明确 warning/blocked。

## M5 麦克风降噪

目标：

- 抽出 RNNoise 音频增强模块。
- Go/native 音频管线支持开关降噪。
- 诊断记录算法状态。
- 默认录制优先把处理后的麦克风音频写入主媒体音轨；WAV sidecar 只作为 fallback、smoke 或恢复诊断路径。

验收：

- 降噪打开时 manifest 记录 `microphoneNoiseSuppression: "rnnoise"`。
- 降噪关闭时不进入 RNNoise。
- 系统声音不被降噪处理。
- 暂停恢复后没有明显爆音或状态污染。
- 主媒体音频可播放且和视频同步；如果使用 fallback sidecar，ready 门禁必须明确验证 sidecar 非空。

## M5b 单独音频录制

目标：

- 支持只录系统声音、只录麦克风、麦克风 + RNNoise。
- 不创建假的 screen media。
- 仍使用 `.rfrec` 包和 `data/video` 数据根，方便统一恢复和管理。

验收：

- 1 分钟、5 分钟 audio-only 输出可播放。
- 录音包在 UI 中显示为音频录制。
- 关闭的音频设备不会污染 manifest。

## M6 摄像头 sidecar 和画中画预备（暂停）

状态：按当前验收策略暂停，等待视频录制和语音/音频录制验收后再恢复。本节保留为后续路线，不作为当前 preview 发布或验收门槛。

目标：

- 录制摄像头 sidecar。
- 记录 `webcamStartOffsetMs`。
- 设置默认 PIP preset。

验收：

- 开启摄像头后包内出现 `webcam.mov` 或 `webcam.mp4`。
- 摄像头文件可独立播放。
- manifest 记录 sidecar 相对路径和 offset。
- 摄像头开启时，如果 webcam sidecar 缺失或为 0 字节，停止阶段不能把包标记为 `ready`。
- UI 中可以看到后续 PIP 预设，不影响屏幕录制稳定性。

## M7 Linux experimental

目标：

- 接入 XDG ScreenCast Portal + PipeWire。
- 能在支持 portal 的桌面环境中完成基础录制。

验收：

- 支持环境能录制屏幕。
- 不支持环境显示 experimental unavailable。
- release notes 不宣称 Linux 与 macOS/Windows 同等稳定。

## M8 导出与后续编辑

目标：

- 建立导出计划合同，先校验 `.rfrec`、sync diagnostics 和 PIP 布局。
- 使用原始 screen + webcam sidecar 做 PIP 合成导出。
- 支持音画同步诊断。
- 后续再引入 cursor、zoom、字幕和轻量编辑。

验收：

- 导出计划拒绝包外路径、mock 包、缺失 screen media、缺失 webcam sidecar 和缺失 sync diagnostics。
- 导出不依赖最终内存 Blob。
- screen、mic/system audio、webcam offset 对齐。
- 30 分钟项目导出不出现内存暴涨。

## 全局完成定义

每个阶段必须同时满足：

- 文档更新。
- 自动化测试或 smoke test 覆盖。
- 至少一个真实平台手动验证记录。
- 不引入绕过 `data/video` 的录制输出路径。
- 不把临时 mock 行为伪装成真实录制能力。
