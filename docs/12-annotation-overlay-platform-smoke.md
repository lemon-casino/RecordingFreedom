# 12. 录制标注 Overlay 实机验收标准

状态：验收规程落地  
日期：2026-07-04  
关联计划：`11-whiteboard-excalidraw-integration-plan.md` 的 P4/P6

## 目标

这份文档固定录制标注 overlay 的实机验收口径。`annotation-export-smoke` 能证明 `.rfrec` 标注时间线和 FFmpeg 导出合成链路，但它不能替代真实桌面上的透明窗口、点击穿透、多屏坐标、高 DPI 和用户手绘输入验收。

录制标注能力只有同时满足下面两类证据，才能在 release notes 中写成“完成”：

- 自动证据：Go 测试、Playwright e2e、`annotation-export-smoke`、Windows portable runner 证明入口规则、hit-region 合同、包内标注写入、导出合成和长时间线限制。
- 实机证据：在真实 Windows、macOS、Linux 桌面上启动软件，完成本文件的 overlay smoke 矩阵，并保留可复查证据。

## 自动证据边界

自动 smoke 必须保留，但不能扩大解释：

- `npm run test:e2e`：证明普通画板、主胶囊双态入口、录制标注前端 hit-region 发布规则没有回退。
- `go test ./...`：证明后端 annotation manifest、录制包写入、导出计划、FFmpeg filter、hit-region 算法和长时间线限制没有回退。
- `annotation-export-smoke`：证明真实短 MP4 + `.rfrec/annotations/` + transparent PNG timeline 能导出成包含标注的最终 MP4。
- `run-windows-portable-smoke.ps1 -RunAnnotationLong`：证明发布包内标注合成工具链可以跑过 1 分钟/5 分钟长时间线。

这些证据不证明：

- 透明 overlay 窗口在 macOS/Linux 上真正把空白区域点击交还到底层应用。
- 多屏幕、高 DPI、混合缩放比例下用户手绘笔迹和录制目标完全对齐。
- 真实录制期间 overlay 不引入鼠标闪烁。
- 用户真实手绘事件进入 `.rfrec/annotations/events.jsonl` 后再导出的视频与现场预览一致。

## 实机 Smoke 矩阵

每个平台至少覆盖一次完整矩阵。Windows 作为 preview 主线必须先完成；macOS/Linux 只有恢复 artifact 制作或声称平台可用时才允许通过该平台的 release gate。

### 录制来源

来源矩阵必须明确覆盖：全屏、单屏、区域、锁定窗口。

- 全屏或全部屏幕：覆盖虚拟桌面边界；Windows 混合分辨率双屏必须额外观察鼠标闪烁。
- 单个屏幕：有几块屏幕就至少选择两块做抽样；单屏机器记录为 single-display。
- 自定义区域：在单个屏幕内框选区域，录制前确认红框可见且可调整；开始录制后 overlay 画布必须只绑定该区域。
- 锁定窗口：选择一个可移动窗口，录制前后确认 overlay bounds 跟随 manifest 中的 window source geometry。

### 输入状态

- 未录制前点击“画板”：必须打开普通画板，不创建 `.rfrec/annotations/`。
- 视频录制中点击“画板”：必须打开 `annotation-overlay`，控制胶囊可点击，画布处于绘制态时能画线。
- 视频暂停中点击“画板”：必须仍打开 `annotation-overlay`，继续录制后标注时间轴不能倒退。
- 音频录制中点击“画板”：必须打开普通画板，不进入视频标注 overlay。

### 点击穿透

- 选择工具或空闲状态：overlay 画布空白区域必须点击穿透到底层应用；底层按钮、文本框或拖动区域要能响应。
- 绘制工具、箭头、矩形、圆形、文字、橡皮：画布必须接收指针；笔迹要出现在录制目标坐标内。
- 控制胶囊区域始终可点击；胶囊外透明区域不能形成整块 WebView 拦截层。
- 切换工具时不允许出现全屏闪烁、胶囊消失、焦点丢失后无法继续录制。

### 几何与导出

- overlay 窗口 bounds 必须等于当前 active manifest 的 `source.geometry`；没有 geometry 时只能回退到明确记录的虚拟桌面 fallback，不能默默使用 1280x720 当作完成。
- `.rfrec/manifest.json` 必须包含 `annotations` 节点，且 `annotations.target` 的 type/id/geometry 与录制 source 一致。
- `.rfrec/manifest.json` 必须包含 `diagnostics.sync.screen.durationMs`，用于证明真实录制时长；1 分钟和 5 分钟 evidence 不能只靠文件名声明。
- evidence 中的 ready `.rfrec` 必须覆盖 `all-screens`、`screen`、`region`、`window` 四类 sourceType，不能只用单一来源代表完整 overlay 验收矩阵。
- `.rfrec/annotations/events.jsonl` 必须包含真实用户手绘保存事件，不能只有 synthetic smoke 事件；结构检查会要求同时存在 `scene-snapshot` 和至少一条 `element-created` / `element-updated` / `element-deleted` 元素级事件，并检查事件的 `recordingOffsetMs`。
- `.rfrec/annotations/overlay-diagnostics.jsonl` 必须包含 `show`、`hit-regions`、`save-capture` 三类真实 overlay 诊断事件，用于复查窗口 bounds、画布 bounds、target geometry 和命中区状态；其中 `hit-regions` 至少要同时证明绘制态发布满画布 `rect`、穿透态只保留胶囊命中区且没有满画布 `rect`。
- `.rfrec/annotations/snapshots/` 或 `annotations/reconstructed/png/` 必须存在非空透明 PNG。
- 导出的 `exports/recording.mp4` 必须是可解析 MP4，包含视频轨，MP4 movie duration 要和 `diagnostics.sync.screen.durationMs` 在容差内一致，并且人工复查时能看到对应时间段的标注；关闭“导出包含标注”后最终 MP4 不能合成标注，但包内 annotations 仍保留。

## 证据目录

每次实机 smoke 需要保存到一个 evidence 目录，目录名建议包含平台、版本、日期：

```text
evidence/annotation-overlay/<platform>-<version>-YYYYMMDD/
  README.md
  app-log.jsonl
  platform.txt
  screenshots/
    source-all-screens.png
    source-screen-1.png
    source-region.png
    source-window.png
    pass-through-click-selection.png
    drawing-state.png
    capsule-controls.png
  recordings/
    source-all-screens.mp4
    source-screen-1.mp4
    source-region.mp4
    source-window.mp4
    export-with-annotations.mp4
    export-without-annotations.mp4
  packages/
    recording-*.rfrec/
      annotations/overlay-diagnostics.jsonl
```

`README.md` 至少记录：

- 软件版本、commit、artifact 来源。
- 操作系统版本、显示器数量、分辨率、缩放比例。
- 录制来源矩阵结果：all-screens、screen、region、window。
- 点击穿透结果：选择态、绘制态、胶囊控制区。
- 导出结果：包含标注和不包含标注各一次。
- 已知失败项和对应阻塞原因。

`platform.txt` 至少记录：

- 操作系统名称和版本/build。
- 显示器数量。
- 每块显示器的分辨率。
- 每块显示器的缩放比例或 DPI。

`app-log.jsonl` 需要从本次实机软件日志复制，至少包含 `app/startup`、四类 source 的 `recording/start-request`、四类 target 的 `annotation-overlay/show` 和一次 `annotation-overlay/save-capture`。

`annotation-overlay-evidence-check` 会检查这些 README 和 platform 记录是否存在，检查 `app-log.jsonl` 是否包含关键运行事件，并检查 `screenshots/`、`recordings/` 是否按上面的命名覆盖来源矩阵、点击穿透、绘制态、胶囊控制区和包含/不包含标注导出对比，避免 evidence 目录只有文件但缺少可追溯的人工验收结论。

保存 evidence 后，可以在 `RecordingFreedom/app` 下先运行结构检查：

```bash
go run ./cmd/annotation-overlay-evidence-check -evidence-dir ../evidence/annotation-overlay/<platform>-<version>-YYYYMMDD
```

Windows portable 验收包会携带同一个检查器：

```powershell
.\tools\annotation-overlay-evidence-check.exe -evidence-dir .\evidence\annotation-overlay\<platform>-<version>-YYYYMMDD
```

该检查只证明 README 追溯记录、platform 显示环境记录、app-log 关键运行事件、截图/录屏 evidence 命名矩阵、证据目录结构、真实 `.rfrec`、`manifest.annotations`、`diagnostics.sync.screen.durationMs`、1 分钟/5 分钟 ready 包、all-screens / screen / region / window 来源矩阵、`scene-snapshot` 与元素级标注事件、`annotations/overlay-diagnostics.jsonl` 中的 show / hit-regions / save-capture 诊断、绘制态满画布 `rect`、穿透态无满画布 `rect`、标注 PNG，以及 `exports/recording.mp4` 是可解析 MP4、包含视频轨、movie duration 与 manifest 录制时长匹配；它不替代人工确认点击穿透、鼠标闪烁、多屏坐标和标注视觉效果是否正确。

## 通过标准

某个平台的录制标注 overlay 可以标为通过，必须同时满足：

- 自动证据全部通过。
- 实机 smoke 矩阵没有阻塞项。
- 至少保留 1 分钟和 5 分钟两个真实录制包，且导出 MP4 中标注可见、时间段正确、音画同步没有明显回退。
- 多屏或高 DPI 环境下，笔迹位置与录制目标无可见偏移。
- 非绘制状态点击穿透真实可用；绘制状态真实可画。
- release notes 明确哪些平台已经完成实机 overlay smoke，哪些平台仍为 queued/experimental。

## 失败处理

- 如果只有 `annotation-export-smoke` 通过，P5 导出合成可记为通过，但 P4/P6 不能记为完成。
- 如果真实 overlay 点击穿透失败，必须修复平台窗口 hit-test 或降级为明确不可用，不能用透明视觉效果冒充可用。
- 如果多屏/高 DPI 坐标偏移，必须统一录制 source、region selector、annotation overlay 和 export plan 的坐标模型后重新验收。
- 如果录制中出现鼠标闪烁，需要先确认是否由录制后端、cursor overlay、overlay WebView 重绘或磁盘写入引起，再进入 release。
