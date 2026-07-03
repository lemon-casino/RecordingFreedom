# 11. 画板胶囊与 Excalidraw 集成计划

状态：计划落地  
日期：2026-07-03  
参考项目：`excalidraw/excalidraw`

## 当前实现状态

更新：2026-07-03

已完成 P1/P2/P3 普通画板代码闭环：

- 主胶囊新增“画板”按钮，按钮位于录制中不会隐藏的控制区；录制前可以打开普通画板，视频录制中可以随时唤出录制标注 overlay。
- 新增 `/#/whiteboard` 路由和独立 Wails `whiteboard` 窗口，窗口置顶、默认隐藏、可调整大小，不改变当前录制状态。
- 集成 `@excalidraw/excalidraw`，画板路由使用 lazy load，主录制胶囊首屏不会静态加载整套画板代码。
- 画板胶囊提供选择、拖动画布、画笔、激光笔、箭头、直线、矩形、圆形、文字、橡皮、撤销、重做、颜色、线宽、透明度、保存、导出 PNG/SVG/`.excalidraw`、清空、关闭。
- 场景保存到应用数据目录 `data/whiteboards/board-current.excalidraw`，导出文件保存到 `data/whiteboards/exports/`。
- 新增白板设置局部 patch：最近工具、颜色、线宽、透明度等只更新 `whiteboard` 字段，不覆盖主题、帧率、画质、音频或摄像头设置。
- 后端对 `.excalidraw` 导出做 JSON 校验，对透明度做 `5..100` 归一化；已有回归测试固定这些行为。
- 已生成 Wails TS bindings，并通过 `npm run build`、`go test ./...`、`wails3 build` 验证。

已新增 P4/P5 第一段基础链路：

- 新增 `/#/annotation-overlay` 透明标注窗口，录制中点击主胶囊“画板”会打开录制标注 overlay；未录制时仍打开普通画板窗口。
- 主胶囊画板入口已固定为双态规则：未录制、准备中、停止中或音频录制时打开普通画板；视频录制 `recording/paused` 状态打开录制标注 overlay，保证录制前可独立使用，录制中可随时唤出标注。
- 桌面运行时不再用浏览器 localStorage 假保存兜底画板/标注；普通画板必须写入 `data/whiteboards/`，录制标注必须写入当前 `.rfrec/annotations/`。浏览器 fallback 只保留给前端开发预览。
- 标注 overlay 使用 Excalidraw 真实绘制，保存当前 scene 和透明 PNG snapshot。
- 新增 `ShowAnnotationOverlay`、`HideAnnotationOverlay`、`SaveAnnotationCapture` 后端服务；保存时写入当前 active `.rfrec` 包的 `annotations/` 目录。
- 新增 `LoadAnnotationCapture`，录制中关闭再唤出标注 overlay 会从当前 `.rfrec/annotations/scene.excalidraw` 恢复，不依赖前端临时内存。
- 新增 `SetAnnotationOverlayHitRegions`，录制标注 overlay 会随工具切换命中区域：绘制/形状/文字/橡皮工具让画布接收鼠标；选择/空闲状态只保留工具栏命中，画布区域交还给被录制应用。Windows 通过原生 `WM_NCHITTEST` 返回 `HTTRANSPARENT` 实现不裁剪画面的鼠标穿透，前端同时用 CSS `pointer-events` 做基础兜底。
- manifest 新增 `annotations` 节点，记录 `scene.excalidraw`、`events.jsonl`、`exports/annotation.png`、目标 source 和 `export-compose` 策略。
- 导出 plan 和 FFmpeg exporter 已识别 annotation snapshot；有标注且“导出包含标注”开启时会把 PNG 确定性 overlay 到最终 MP4，而不是依赖透明窗口刚好被录屏捕获。
- 设置页新增“导出包含标注”开关，使用 `whiteboard.capturePolicy` 局部 patch 保存；关闭时标注仍写入 `.rfrec/annotations/`，但导出计划不会加入 annotation input。
- 标注保存事件会追加到 `events.jsonl`，每次 scene snapshot 记录 `wallOffsetMs`、`recordingOffsetMs`、session、scene path 和 snapshot path；`recordingOffsetMs` 会扣除 recording service 已知暂停时长。
- 录制标注前端会聚合 Excalidraw 元素级 create/update/delete 事件，随 debounce 保存批量提交；后端每次保存先追加一个 `scene-snapshot` 锚点，再逐行校验元素 JSONL，补齐 `schemaVersion`、`sequence`、`eventId`、录制 offset 和 snapshot 路径后追加，避免鼠标移动期间频繁写盘。
- 导出计划已读取 `.rfrec/annotations/events.jsonl`，流式解析第一条元素级事件的 `recordingOffsetMs`；FFmpeg 合成最终标注 snapshot 时会使用 `enable='gte(t,...)'` 从第一笔真实标注之后才显示，避免标注从视频 0 秒提前出现。旧包没有事件文件时仍兼容最终 snapshot 全片合成，并写出 warning。
- 录制标注每次保存会同时保留最终兼容快照 `annotations/exports/annotation.png` 和一个按事件序号命名的历史快照 `annotations/snapshots/annotation-000001.png`；导出计划会解析这些 `scene-snapshot` 事件，生成 `snapshot-segments` 时间轴，FFmpeg exporter 会按 `gte/lt` 时间窗口逐段 overlay，旧包或缺失历史快照时仍回退到最终快照合成并写出 warning。
- 录制标注保存的最终快照和历史快照现在都会按当前录制画布尺寸导出为整张透明 PNG，只让笔迹区域有像素，避免 Excalidraw 默认按内容裁剪后导致 FFmpeg 左上角 overlay 偏位。
- 新增 `PreviewExportRecordingPackage` 后端方法，设置页会在不启动 FFmpeg 的情况下读取最近录制包的导出计划，显示画中画/标注是否合成、标注分段数量、事件数量、时间范围和 warning 数量；“导出包含标注”开关变化后预览会跟随刷新，保证预览和真实导出使用同一套 `exportplan`。
- 导出预览已增加标注时间线诊断：每个 snapshot segment 会输出包内相对路径、开始时间、结束时间、持续时间和字节数；summary 会输出事件文件体积、snapshot 总体积、已导出/已跳过 snapshot 数。设置页会展示前几段标注时间线，并在事件文件或 snapshot 素材过大、快照缺失、快照为空时给出 warning，方便长录制导出前排查。
- 新增 `ReadAnnotationPreviewImage` 后端方法，设置页导出预览会从当前 `.rfrec` 仅读取 `annotations/snapshots/*.png` 或最终 `annotations/exports/annotation.png`，把真实标注 PNG 渲染为前几段时间线缩略图；接口拒绝包外路径和非 annotation 路径，避免预览成为任意文件读取入口。
- 新增元素级事件重建基础层：导出计划读取 `element-created`、`element-updated`、`element-deleted` 时会按事件顺序还原当前活动元素集合，输出 `elementTimelineMode`、关键帧数、最终元素数、删除元素数、缺失 payload 数、元素类型统计和前若干关键帧摘要；如果旧包或异常事件缺少 `element` payload，会标记为 `element-events-partial` 并写出 warning，不再把不完整事件流当成完整逐笔重建。
- 导出路径已接入元素级 scene 素材准备：`ExportRecordingPackage` 会以显式 `PrepareAnnotationAssets` 模式生成 `annotations/reconstructed/scene-xxxxxx.excalidraw`，每个 scene 对应一个元素关键帧并记录起止时间、元素数量、源事件 sequence 和文件大小；`PreviewExportRecordingPackage` 保持只读，不会反复改写录制包。若元素事件缺 payload 或关键帧数量超过安全上限，则不生成不完整素材，继续回退到 snapshot timeline 并写 warning。
- 新增 `/#/annotation-renderer` 后台渲染窗口和后端任务队列：导出时会把完整元素级 `.excalidraw` scene 通过 Excalidraw 官方 `exportToBlob` 渲染成整张录制画布尺寸的透明 PNG，写入 `annotations/reconstructed/png/annotation-xxxxxx.png`。
- 导出 plan 已优先识别完整的元素级渲染 PNG 时间线，模式为 `element-pngs`；只有 PNG 缺失或渲染不完整时，才回退到保存点 snapshot 分段或旧包最终 snapshot。`ReadAnnotationPreviewImage` 仍限定在当前 `.rfrec` 的 annotation PNG 路径内，并已允许读取 `annotations/reconstructed/png/*.png` 缩略图。
- 已新增回归测试覆盖 annotation manifest 安全路径、包内保存、export plan 和 FFmpeg filter 组合。
- 已新增录制标注 overlay 自动化契约测试：覆盖未录制时拒绝录制标注、音频录制时拒绝录制标注、视频录制中/暂停中允许录制标注，并固定全屏、多屏、区域、锁定窗口来源的 overlay 窗口坐标和画布尺寸必须跟录制目标 geometry 一致。
- 已新增 Playwright `test:e2e` 覆盖普通画板胶囊：使用真实 Vite 页面打开 `/#/whiteboard`，验证中文/英文、暗夜青/远山绿两个主题、工具栏按钮数量、选中态、保存/导出按钮、状态栏、主题 token 和横向溢出。
- 已新增 Playwright `test:e2e` 覆盖主胶囊画板入口双态规则：使用真实 Vite `/#/` 页面验证未录制时点击“画板”打开普通画板；开始视频录制后胶囊进入紧凑态，同一个“画板”按钮仍可见且会打开录制标注 overlay；视频录制暂停中仍会打开录制标注 overlay；开始音频录制后同一个“画板”按钮仍可见且打开普通画板，避免音频录制被误切到视频标注 overlay。
- 已新增 Playwright `test:e2e` 覆盖录制标注 overlay 工具态命中区域：使用真实 Vite 页面打开 `/#/annotation-overlay`，验证默认画笔态为 `is-drawing`、画布可接收指针并发布胶囊 `pill` + 录制画布 `rect` 两个 hit region，且 `rect` 尺寸必须等于录制画布；切换选择工具后变为 `is-pass-through`、画布 `pointer-events:none` 且只发布尺寸受限的胶囊 `pill` hit region，明确不能发布全画布命中区；再切回画笔会恢复画布命中区。
- CI 和 Release Gate 已纳入前端 `test:e2e`，并由 `cmd/release-config-check` 校验门禁文本，防止后续版本把画板入口从录制紧凑态移除、或把未录制/音频录制场景误切到录制标注 overlay。
- 主胶囊“画板”按钮选中态已改为跟随后端 `whiteboard.visibility` 事件：普通画板或录制标注打开后进入 `aria-pressed=true`，隐藏或关闭后恢复 `aria-pressed=false`；浏览器开发 fallback 使用同名自定义事件保持自动化验收一致。
- 已新增 P5 长录制防护回归测试：固定 annotation events 单行 1 MiB、事件 200000、snapshot 合成段 2000、元素级 scene asset 2000 的上限行为；超过 snapshot/scene asset 上限时降级并给 warning，超过事件上限时拒绝导出计划，避免长录制把内存或 FFmpeg filter 链撑爆。
- 已新增 P5 标注时间轴和恢复回归测试：`SaveAnnotationCapture` 在暂停/继续后写入的 `recordingOffsetMs` 必须扣除暂停时长，不能退回墙钟时间；recover 中的 `.rfrec` 包如果已经写入 annotations，`RecoverRecordingPackage` 后 manifest 必须保留 annotations，`PreviewExportRecordingPackage` 必须继续生成可见的 snapshot annotation 导出计划。
- 已新增并强化 `cmd/annotation-export-smoke` 无 UI 导出验收：命令会生成真实短 `screen.mp4`、多段透明 annotation PNG、合法 `.rfrec/manifest.json` 和 `annotations/events.jsonl`，走同一套 `exportplan` + `exporter` 导出 MP4，再用 FFmpeg 对每个 annotation segment 抽帧做像素级检查，确认每段标注都按时间线进入最终视频且背景没有被污染。默认仍保持 2 段红/绿兼容验收，同时新增 `-segments` 参数用于 1 分钟/5 分钟长时间线 smoke，新增 `-timeline=element-pngs` 验证元素级 rendered PNG 优先合成路径，并新增 `-source-type`/geometry 参数把 annotation target 绑定到 screen/all-screens/region/window 来源。当前本机已运行通过默认 2 段、`-duration 5s -segments 5` 的 snapshot timeline、`-duration 5s -segments 5 -timeline element-pngs`、region source snapshot timeline，以及 window source element-pngs timeline；5 段导出逐段像素分别命中红、绿、蓝、黄、紫。
- Windows portable release 工具链已接入 `annotation-export-smoke.exe`：CI/Release 会构建该工具，portable zip 会携带它，`verify-windows-portable.ps1` 会校验它存在且为 x64 console PE，并检查 runner 保留 `-segments=`、`-timeline=element-pngs`、`-source-type=region`、`-source-type=window` 和 `-RunAnnotationLong` 长标注入口；`run-windows-portable-smoke.ps1` 默认执行 region snapshot 和 window element-pngs 两条 `annotation-export-smoke -duration=5s -segments=5`，避免发布包漏掉标注导出和 annotation target 绑定验收入口。需要验证 1 分钟/5 分钟长时间线时，可在同一 portable runner 上显式加 `-RunAnnotationLong`，它会追加 region snapshot 与 window element-pngs 的 `1m`/`5m` 标注导出抽帧验收，默认每条长时间线 60 段。当前本机已直接运行通过 `1m/60` 和 `5m/60` 的 region snapshot、window element-pngs 四条合成长时间线 smoke；这只能证明导出合成、元素级 PNG 时间线和 annotation target 绑定，不替代真实手绘 overlay、真实录屏输入和 macOS/Linux 透明窗口穿透验收。
- Windows portable release 工具链已接入 `annotation-overlay-evidence-check.exe`：CI/Release 会构建该工具，portable zip 会携带它，`verify-windows-portable.ps1` 会校验它存在且为 x64 console PE；`run-windows-portable-smoke.ps1` 会确认该工具可用并写入 summary。真实 overlay smoke 完成后，可用它检查 evidence 目录 README 追溯记录、platform 显示环境记录、app-log 关键运行事件、截图/录屏命名矩阵、真实 `.rfrec/annotations/`、`diagnostics.sync.screen.durationMs`、1 分钟/5 分钟 ready 包、all-screens / screen / region / window 来源矩阵、`scene-snapshot` 与元素级标注事件、`annotations/overlay-diagnostics.jsonl` 中的 show / hit-regions / save-capture 诊断、绘制态满画布 `rect`、穿透态无满画布 `rect`、标注 PNG，以及 `exports/recording.mp4` 是可解析 MP4、包含视频轨并且 movie duration 与 manifest 录制时长匹配。该工具只检查证据结构、README 记录、platform 显示环境记录、app-log 关键运行事件、截图/录屏文件名矩阵、诊断 JSONL 和导出 MP4 元数据，不替代人工确认点击穿透、鼠标闪烁、多屏坐标和标注视觉效果。
- Excalidraw MIT 许可证已汇总到 `app/tools/THIRD_PARTY_NOTICES.txt`，Windows portable、Windows installer 和 macOS app bundle 的复制路径已接入，并由 release config check、portable verifier 和 installer verifier 校验。

仍未完成：

- P4 录制标注 overlay 还需要实机验证和补强：全屏、单屏、区域、锁定窗口、多屏幕、高 DPI、不同缩放比例下的坐标一致性还没有完成验收。
- P4 点击穿透还需要继续细化：当前 overlay 用于绘制时接收指针，非绘制状态的最小 hit region/点击穿透策略还需要按平台验证。
- P4/P6 实机验收矩阵已落地到 `12-annotation-overlay-platform-smoke.md`：后续不能用 `annotation-export-smoke` 代替真实 overlay 验收，完成状态必须保留 evidence、真实 `.rfrec` 包和导出 MP4。
- P5 当前完成的是“最终状态 snapshot 合成”、导出前“包含标注”开关、scene snapshot 事件的录制 offset 写入、元素级事件 JSONL 追加、基于保存点历史 snapshot 的分段时间轴导出合成，以及导出前 plan 预览会跟随标注开关刷新；预览模型已包含分段相对路径、起止时间、持续时间、字节统计、缺失/空快照跳过数量和长录体积 warning，并已能在设置页显示真实 annotation snapshot 缩略图。元素级事件流已能重建活动元素状态和关键帧摘要，导出路径已能生成逐关键帧 `.excalidraw` scene 素材，并通过后台渲染窗口生成 `annotations/reconstructed/png/*.png` 后优先交给 FFmpeg 分段合成；命令行 smoke 已能用 `-segments` 做多段长时间线导出验证，能用 `-timeline=element-pngs` 真实验证元素级 rendered PNG 优先合成，并能用 `-source-type` 验证 annotation target 绑定到 region/window 来源，但真实 1 分钟/5 分钟录制包的实机体积和同步验证仍未完成。

## 目标

在 RecordingFreedom 中加入录制场景可用的“画板”能力：

- 主胶囊增加“画板”入口。
- 点击后弹出独立的画板胶囊，不挤占主录制胶囊。
- 画板胶囊提供选择、画笔、箭头、矩形、圆形、文字、橡皮、撤销、重做、颜色、线宽、清空、隐藏、关闭等工具。
- 支持普通白板和录制标注两种形态。
- 后续录制时可以把标注内容确定性地进入最终视频，而不是依赖不可控的窗口截图副作用。

这不是把 Excalidraw.com 整个产品搬进软件，也不是先做一个无法录制集成的假画板。画板必须服务录制、可保存、可恢复、可验证。

## Excalidraw 调研结论

已将参考仓库克隆到：

```text
C:\Users\Lemon\Desktop\LikelySnpa\_reference\excalidraw
```

当前同步到 `origin/master`：

```text
cce5001 2026-06-29 fix(editor): keep eraser shortcut consistent with tool switch rules (#11571)
```

关键结论：

- 仓库许可证是 MIT，可以集成，但需要在第三方声明中保留版权和许可证文本。
- 可嵌入包是 `@excalidraw/excalidraw`，当前包版本为 `0.18.0`。
- peer dependency 支持 React 17、18、19；RecordingFreedom 当前是 React 18.2，兼容。
- 包提供 React 组件 `Excalidraw`，使用时需要引入 `@excalidraw/excalidraw/index.css`。
- 包导出 `exportToBlob`、`exportToSvg`、`exportToCanvas`、`serializeAsJSON`、`loadSceneOrLibraryFromBlob`、`restoreElements` 等能力。
- `ExcalidrawProps` 支持 `initialData`、`onChange`、`onExcalidrawAPI`、`renderTopLeftUI`、`renderTopRightUI`、`langCode`、`theme`、`UIOptions`。
- `ExcalidrawImperativeAPI` 支持 `updateScene`、`resetScene`、`getSceneElements`、`getAppState`、`getFiles`、`setActiveTool`、`onChange` 等方法。
- Excalidraw.com 的实时协作、分享链接、端到端加密属于完整网站应用能力，不是 npm 组件的基础交付内容，初版不纳入。

推荐采用 `@excalidraw/excalidraw` 作为画板核心，不重写画布引擎。RecordingFreedom 负责胶囊 UI、透明窗口、点击穿透、录制包、导出合成和平台能力。

## 产品形态

### 双态使用规则

画板功能必须同时支持“未录制前使用”和“录制中唤出使用”，这是固定验收口径：

- 未录制、准备中、停止中：点击主胶囊“画板”打开普通画板窗口，不创建录制标注，不影响录制配置。
- 音频录制中：点击主胶囊“画板”仍打开普通画板窗口，因为没有视频画面需要合成标注。
- 视频录制中或暂停中：点击主胶囊“画板”打开透明录制标注 overlay，标注保存到当前 `.rfrec/annotations/`，后续导出可按设置合成进最终视频。
- 打开、隐藏、关闭普通画板或录制标注都不能改变当前录制状态，也不能触发开始、暂停、结束录制。
- 主胶囊“画板”入口必须保留在录制紧凑态中，源、音频、摄像头、语言、设置隐藏后仍然可以唤出标注。

### 主胶囊入口

主录制胶囊增加一个画板按钮：

- 未录制时：显示在源、音频、摄像头、语言、设置这一组工具旁。
- 录制中紧凑模式：保留“画板”入口，方便录制时打开标注；源、音频、摄像头、语言、设置仍按现有规则隐藏。
- 点击画板按钮只控制画板胶囊，不改变当前录制状态。
- 画板打开后按钮进入选中态；画板关闭或隐藏后恢复普通态。选中态以 `whiteboard.visibility` 后端事件为准，不能只靠点击后的前端乐观状态。
- 实现约束：画板按钮不能放进 `.capsule-config-segment` 或 `.capsule-utility-segment`，因为这两个分组在录制紧凑态会隐藏。

### 画板胶囊

画板胶囊是独立悬浮窗口，延续 RecordingFreedom 的胶囊风格，但更窄、更工具化：

- 左侧是拖动区域和当前模式。
- 中间是工具按钮组。
- 右侧是颜色、线宽、更多、隐藏、关闭。
- 所有可点击项都要有小手光标和稳定 hover/pressed 状态。
- 下拉、颜色面板、线宽面板必须使用现有自定义弹层规则：点击外部关闭，点击内部不误关；靠近屏幕边缘时自动向内展开。
- 胶囊外透明区域必须真正穿透，不形成整块不可点击窗口。

初始工具：

| 工具 | Excalidraw 映射 | 说明 |
| --- | --- | --- |
| 选择 | `selection` | 移动、选择、调整元素 |
| 拖动画布 | `hand` | 普通白板模式使用 |
| 画笔 | `freedraw` | 录制标注默认工具 |
| 激光笔 | `laser` | 后续用于临时强调，默认不落盘 |
| 箭头 | `arrow` | 指向说明 |
| 直线 | `line` | 简单连线 |
| 矩形 | `rectangle` | 框选重点 |
| 圆形 | `ellipse` | 圈注重点 |
| 文字 | `text` | 简短说明 |
| 橡皮 | `eraser` | 删除标注 |

### 两种模式

#### 普通白板模式

普通白板用于录制前准备、讲解草稿、会议白板：

- 使用独立非透明画板窗口。
- 窗口内容是完整 Excalidraw canvas。
- 支持保存为 `.excalidraw` JSON。
- 支持导出 PNG/SVG。
- 不默认进入录制画面。

#### 录制标注模式

录制标注用于屏幕录制时在录制区域上画线、箭头、文字：

- 使用透明 overlay 窗口覆盖录制目标区域。
- 画板胶囊作为控制窗口，标注画布和控制胶囊分离。
- 画布只显示笔迹和选中反馈，不显示 Excalidraw 完整菜单。
- 非绘制状态下 overlay 尽可能点击穿透；绘制状态下只让必要画布区域接收指针。
- 标注内容保存进 `.rfrec` 包，后续导出时可确定性合成进最终视频。

## 推荐架构

### 前端

新增前端路由：

```text
/#/whiteboard
/#/annotation-overlay
```

新增组件：

```text
WhiteboardWindow
WhiteboardCapsule
AnnotationOverlayWindow
ExcalidrawHost
BoardToolPalette
BoardStylePopover
```

依赖建议：

```bash
npm install @excalidraw/excalidraw
```

前端集成原则：

- 对 Excalidraw 使用 lazy load，避免主胶囊首屏包体突然变大。
- CSS 只在画板相关入口加载，避免污染主胶囊。
- `onExcalidrawAPI` 保存 API 引用，用 `setActiveTool` 控制工具。
- `onChange` 使用 debounce 保存 scene，不能每次鼠标移动都同步写盘。
- 使用 `UIOptions` 隐藏不需要的内置菜单，工具栏由 RecordingFreedom 自己绘制。
- `langCode` 跟随 RecordingFreedom 全局语言，只保留英文和简体中文。
- `theme` 跟随 RecordingFreedom 当前主题，但画板 stroke 颜色要独立保存。

### 后端

新增服务职责：

```text
ShowWhiteboardWindow()
HideWhiteboardWindow()
ToggleWhiteboardWindow()
ShowAnnotationOverlay(target)
HideAnnotationOverlay()
SetWhiteboardHitRegions(request)
SaveWhiteboardScene(request)
LoadWhiteboardScene(id)
ExportWhiteboardScene(request)
AttachAnnotationSession(recordingSessionID)
AppendAnnotationEvent(event)
FinalizeAnnotationTrack(recordingPackage)
```

新增窗口：

```go
createWhiteboardWindow(app)
createAnnotationOverlayWindow(app)
```

窗口策略：

- `whiteboard`：普通可调整窗口，默认居中或贴近主胶囊，适合完整画板。
- `annotation-overlay`：透明、无边框、置顶、跨桌面空间，大小跟随录制目标。
- `whiteboard-capsule` 可以和 `whiteboard` 同窗口内实现；如果后续需要更稳定的点击穿透，再拆成独立小窗口。

### 存储

普通白板默认存储：

```text
data/whiteboards/board-YYYY-MM-DD-HH-mm-ss.excalidraw
data/whiteboards/assets/
```

录制包内标注存储：

```text
recording-*.rfrec/
  annotations/
    scene.excalidraw
    events.jsonl
    snapshots/
      annotation-000001.png
      annotation-000004.png
    exports/
      annotation.png
      annotation.svg
```

manifest 建议扩展：

```json
{
  "annotations": {
    "enabled": true,
    "mode": "overlay",
    "scenePath": "annotations/scene.excalidraw",
    "eventsPath": "annotations/events.jsonl",
    "capturePolicy": "export-compose",
    "target": {
      "type": "screen",
      "id": "screen:1"
    }
  }
}
```

### 录制合成策略

初版不要依赖“透明窗口刚好被屏幕录制捕获”作为唯一方案，因为不同平台、不同后端行为不一致。

推荐策略：

1. 录制时显示实时 overlay，用户能看见标注。
2. 标注场景和增量事件同步写入 `.rfrec/annotations/`。
3. 停止录制后，导出阶段根据时间轴把标注合成到最终视频。
4. 如果当前平台的实时录屏已经包含 overlay，也不能因此跳过 `.rfrec` 标注数据写入。

这样可以解决两个长期问题：

- 控制胶囊不会被误录进视频。
- Windows、macOS、Linux 的透明窗口捕获差异不会影响最终导出一致性。

## 数据模型

新增前端状态：

```ts
type WhiteboardMode = 'board' | 'annotation'

type WhiteboardTool =
  | 'selection'
  | 'hand'
  | 'freedraw'
  | 'laser'
  | 'arrow'
  | 'line'
  | 'rectangle'
  | 'ellipse'
  | 'text'
  | 'eraser'

type WhiteboardStyle = {
  strokeColor: string
  backgroundColor: string
  strokeWidth: 'thin' | 'medium' | 'bold'
  opacity: number
}

type AnnotationCapturePolicy = 'preview-only' | 'export-compose'
```

新增 settings：

```json
{
  "whiteboard": {
    "enabled": true,
    "lastMode": "annotation",
    "lastTool": "freedraw",
    "lastStrokeColor": "#ef4444",
    "lastStrokeWidth": "medium",
    "capturePolicy": "export-compose"
  }
}
```

保存规则仍沿用当前设置修复后的标准：

- 后端权威。
- patch API 更新局部字段。
- 不用前端整份 settings 覆盖后端最新值。
- 写入串行化，避免主题、画质、帧率这类问题反复出现。

## 分阶段任务

### P0 文档与技术确认

状态：当前文档覆盖。

验收：

- Excalidraw 克隆、版本、许可证、关键 API 已记录。
- 画板胶囊的模式、架构、存储、录制合成策略已明确。
- README 文档入口已挂载。

### P1 普通画板窗口

目标：

- 安装 `@excalidraw/excalidraw`。
- 新增 `/#/whiteboard` 路由。
- 主胶囊增加画板按钮。
- 点击后打开普通画板窗口。
- 支持画笔、箭头、矩形、圆形、文字、橡皮、撤销、重做。
- 支持保存和恢复 `.excalidraw` 场景。

验收：

- 打开和关闭画板不影响录制胶囊状态。
- 重启软件后能恢复最近一次画板。
- 主胶囊透明穿透不回退。
- Vite build 和 Wails build 通过。

当前状态：已完成。未录制和音频录制中打开普通画板；视频录制中通过主胶囊画板按钮唤出录制标注 overlay。Playwright 已固定主胶囊双态入口规则，确保录制紧凑态仍保留画板按钮。

### P2 画板胶囊工具栏

目标：

- 用 RecordingFreedom 自有胶囊 UI 替换 Excalidraw 默认主工具栏。
- 做颜色、线宽、透明度、清空、导出菜单。
- 弹层边缘自适应，不能裁切。

验收：

- 所有按钮有 hover、pressed、selected、disabled 状态。
- 所有下拉点击内部不误关，点击外部自动关。
- 主题切换、语言切换后画板胶囊不还原设置。
- Playwright 覆盖中文、英文和至少两个主题。

当前状态：核心工具栏已完成，含撤销、重做、透明度和局部设置持久化。已新增 Playwright 普通画板胶囊 smoke，覆盖中文/英文和暗夜青/远山绿两个主题，并检查工具栏完整性、选中态、保存/导出入口、状态栏、主题 token 和横向溢出。

### P3 普通白板导出

目标：

- 支持导出 PNG、SVG、`.excalidraw`。
- 第三方许可证说明加入 release artifact。
- 白板文件进入统一应用数据目录，不写入安装目录。

验收：

- 导出的 PNG/SVG 可打开。
- `.excalidraw` 可重新导入。
- clean machine 上没有开发目录依赖。

当前状态：PNG/SVG/`.excalidraw` 导出已接入后端应用数据目录，`.excalidraw` JSON 校验已有测试；第三方许可证汇总已写入 `app/tools/THIRD_PARTY_NOTICES.txt`，并进入 Windows portable、Windows installer 和 macOS app bundle 的发布校验链路。

### P4 录制标注 overlay

目标：

- 新增 `/#/annotation-overlay`。
- overlay 根据当前录制源绑定到全屏、单屏、区域或窗口。
- 支持点击穿透、绘制、移动、隐藏。
- 录制中可以打开画板胶囊并继续录制。

验收：

- 全屏、单屏、区域、锁定窗口下 overlay 坐标正确。
- 多屏幕、高 DPI、不同缩放比例下笔迹位置不偏。
- overlay 不引入鼠标闪烁。
- 控制胶囊不被录入最终视频，除非用户明确选择录入控件。

当前状态：已完成第一版透明 overlay 窗口、录制中入口分流和工具态输入区域切换；窗口边界优先来自 active recording manifest 的 source geometry，缺失时回退虚拟桌面边界；overlay 关闭后再次唤出会恢复当前录制包内已保存的标注 scene。Windows 已接入原生 hit-test 穿透，不会裁剪标注画面；Go hit-region 契约测试已固定“选择态只有胶囊命中、画布中心必须 miss 并交给原生穿透；绘制态加入画布命中后中心必须 hit”的行为，并覆盖高 DPI 客户端缩放；Playwright 已覆盖前端工具态 hit-region 发布规则，固定选择态只保留尺寸受限的胶囊 `pill` 命中区、绘制态增加录制画布 `rect` 命中区且尺寸必须等于画布，并覆盖视频录制紧凑态主胶囊画板按钮打开 annotation overlay。overlay 打开、命中区更新和保存标注时会写入 `annotations/overlay-diagnostics.jsonl`，记录 show / hit-regions / save-capture、窗口 bounds、画布 bounds、target geometry 和命中区状态，供实机 evidence 复查；evidence checker 已要求同一真实包内出现绘制态满画布 `rect` 和穿透态无满画布 `rect` 两类 hit-region 诊断。实机验收矩阵已固定到 `12-annotation-overlay-platform-smoke.md`，后续必须按该文件保存 evidence，覆盖全屏、单屏、区域、锁定窗口、多屏、高 DPI、绘制态和穿透态。macOS/Linux 仍需实机验证透明窗口是否真正把非绘制画布点击交还到底层应用。仍需实机验证多屏/高 DPI/锁定窗口坐标。

### P5 标注写入录制包与导出合成

目标：

- 录制开始时创建 annotation session。
- 标注场景和事件进入 `.rfrec/annotations/`。
- 导出时将标注合成到 screen video。
- 后续可在导出前开关“包含标注”。

验收：

- 录制时画出的标注在导出视频中可见。
- 暂停、继续后标注时间轴正确。
- 录制崩溃后可从 `.rfrec` 恢复标注数据。
- 30 分钟录制不会因为标注事件导致内存暴涨。

当前状态：已完成包内 `annotations/` 基础资产写入、manifest contract、annotation PNG snapshot 导出合成、导出前包含标注开关、scene snapshot 事件 offset 写入、元素级 create/update/delete 事件 JSONL 追加，以及基于 `events.jsonl` 中 `scene-snapshot.snapshotPath` 的保存点分段 snapshot 时间轴导出。每次保存会写入 `annotations/snapshots/annotation-xxxxxx.png` 历史快照，快照按录制画布尺寸保存为整张透明 PNG，避免合成偏位；导出计划输出 `annotationSnapshots` 和 `annotationSummary`，FFmpeg exporter 会按分段时间窗口合成；设置页新增导出前 plan 预览，能在不启动 FFmpeg 的情况下显示 PIP/标注合成状态、分段数、事件数、时间范围和 warning，并随“导出包含标注”开关刷新。导出计划和设置页已补充标注时间线诊断，包含分段相对路径、起止时间、持续时间、snapshot 字节数、事件文件体积、snapshot 总体积、已导出/已跳过 snapshot 数，以及长录体积 warning；设置页已能读取当前 `.rfrec` 中的真实 annotation PNG，并在前几段时间线上显示缩略图；导出计划已能从元素级事件流重建活动元素状态和关键帧摘要，并在事件缺 payload 时标记 partial；导出路径已能把完整元素关键帧写成 `annotations/reconstructed/*.excalidraw` scene 素材，并通过后台 annotation renderer 渲染为 `annotations/reconstructed/png/*.png` 后优先合成。旧包没有历史快照时仍兼容最终 snapshot 合成并写出 warning。已用回归测试固定暂停/继续后的 annotation `recordingOffsetMs` 会扣除暂停时长，recover 中的 `.rfrec` 包不会丢失 annotations，恢复后导出预览仍能生成 snapshot annotation 时间线；已固定长录制上限：events 单行 1 MiB、事件 200000、snapshot 合成段 2000、元素级 scene asset 2000；超出 snapshot/scene asset 上限会降级并提示，超出事件上限会拒绝导出计划。`cmd/annotation-export-smoke` 已提供可命令化导出验收，并已在本机通过默认两段、`-duration 5s -segments 5` 多段 snapshot 时间线、`-duration 5s -segments 5 -timeline element-pngs` 元素级 rendered PNG 时间线、region source snapshot 时间线和 window source element-pngs 时间线的最终 MP4 抽帧像素检查，证明多段标注能按时间切换进入最终视频且 annotation target 与来源 geometry 一致。仍需多平台真实录制包导出验证。

### P6 跨平台验收与发布门禁

目标：

- Windows 继续作为 preview 验收主线。
- macOS 需要验证透明 overlay、ScreenCaptureKit 捕获行为、导出合成一致性。
- Linux 需要验证 Wayland/X11 下窗口置顶、点击穿透和 Portal 录制限制。

验收：

- 每个平台至少完成 1 分钟和 5 分钟录制标注 smoke。
- 导出视频包含标注，音画同步不回退。
- release artifact 包含 Excalidraw 许可证声明。
- 没有新增 mock 能力伪装成真实功能。

当前状态：Excalidraw 许可证声明已进入 release artifact 复制和校验配置；P5 已具备保存点分段 snapshot 时间轴导出合成和元素级 rendered PNG 时间轴合成。Windows portable 工具链已携带并执行 `annotation-export-smoke.exe`，runner 默认用 region snapshot 与 window element-pngs 两条 5 秒 5 段时间线证明发布包内 FFmpeg 标注合成链路能把多段 annotation PNG 按时间窗口写进最终 MP4，并能把 annotation target 绑定到非 screen 来源 geometry；同一 runner 已增加 `-RunAnnotationLong`，用于显式追加 1 分钟/5 分钟、每条 60 段的 region snapshot 与 window element-pngs 长时间线验证。当前本机已通过这四条合成长时间线 smoke。Windows portable 工具链也已携带 `annotation-overlay-evidence-check.exe`，用于真实 overlay smoke 完成后的 evidence 目录结构、`diagnostics.sync.screen.durationMs`、1 分钟/5 分钟 ready 包、all-screens / screen / region / window 来源矩阵、元素级标注事件、绘制/穿透 hit-region 诊断和 `annotations/overlay-diagnostics.jsonl` 复查。`12-annotation-overlay-platform-smoke.md` 已固定真实 overlay smoke 的矩阵、证据目录和通过标准；平台真实 overlay smoke、真实录制包长录事件体积验证和 macOS/Linux 透明穿透仍按 P4/P5/P6 后续项推进，不能被合成 smoke 或 evidence 结构检查替代。

## 风险与处理

| 风险 | 处理 |
| --- | --- |
| 包体变大 | lazy load 画板路由；主胶囊不首屏加载 Excalidraw |
| CSS 污染主界面 | 画板样式隔离；必要时为画板窗口独立 root class |
| 透明 overlay 被不同平台捕获行为不一致 | 不把实时捕获作为唯一结果；以 `.rfrec` 事件和导出合成为准 |
| 指针事件和点击穿透回归 | 复用并扩展现有 hit-region 测试 |
| 多屏坐标偏移 | 录制源选择、region overlay、annotation overlay 使用同一坐标模型 |
| 写盘过频 | `onChange` debounce；事件追加写入；大文件资产单独管理 |
| Excalidraw 内置菜单过重 | 用 `UIOptions` 隐藏不需要的动作，自建画板胶囊 |
| 第三方许可遗漏 | 更新第三方许可证文档和 release artifact |

## 不纳入初版的能力

- Excalidraw.com 实时协作。
- 云同步、分享链接、端到端加密。
- 移动端 Android/iOS。
- 完整图库市场。
- AI 生成图表。
- 多人光标。

## 初版完成定义

画板初版只有同时满足以下条件才算完成：

- 主胶囊可以稳定打开和关闭画板胶囊。
- 可以真实绘制、撤销、重做、清空。
- 场景可以保存、恢复、导出。
- 普通画板不会影响录制。
- 录制标注 overlay 的坐标、点击穿透、写包和导出合成有明确验证。
- 任何未完成能力都必须显示为不可用或排期中，不能用假 UI 顶替。
