# 胶囊列表多窗口浮层方案

## 结论

可以用多窗口解决当前列表占用胶囊窗口范围的问题，而且这是更符合 RecordingFreedom 胶囊形态的长期方案。

推荐方案不是继续把列表放在胶囊主 WebView 内部，也不是单纯调 CSS 透明区域，而是拆成三层窗口：

- 胶囊主窗口：只负责常驻录制控制，未展开时窗口真实宽高只等于胶囊占用范围。
- 浮层面板窗口：负责视频源、音频、摄像头、截图/画板、语言、快捷设置等一级面板。
- 浮层选择窗口：负责面板内部的设备下拉、主题下拉、帧率/画质下拉等二级列表。

这样不打开列表时，桌面只存在胶囊主窗口的命中范围；打开列表时，只有实际列表窗口占用对应区域，列表之外仍然可点击穿透到后面的桌面或其他软件。

## 当前问题

现有实现中，`App.tsx` 用 `activePanel` 在胶囊主窗口内渲染 `.popover`：

- `activePanel !== null` 会让 `capsuleExpanded` 为 true。
- `setCapsuleWindowExpanded()` 会把胶囊主窗口从 collapsed size 调整到 expanded reserved size。
- 胶囊窗口命中区域会把 `.popover` 和 `.select-menu-list` 一起加入。
- `.popover` / `.select-menu-list` 虽然视觉上像浮层，但它们仍然属于胶囊主窗口。

结果就是：

- 打开视频、音频、摄像头、语言、工具、设置等列表时，胶囊窗口真实占用范围会变大。
- 即使透明背景已经尽量处理，窗口本体仍然会影响点击穿透、失焦关闭、边缘展示和拖动稳定性。
- 下拉选择框继续在当前窗口内展开，会让面板高度、滚动区域和命中区域变复杂。
- 胶囊靠近屏幕边缘或多屏交界时，列表容易被裁切、遮盖，或在不该出现的方向展开。

## 目标行为

未打开任何列表：

- 胶囊主窗口保持 collapsed 尺寸。
- 普通横向胶囊只占用约 `760 x 96` 或录制紧凑态约 `380 x 96`。
- 左右吸附竖向胶囊只占用竖向胶囊自身尺寸。
- 胶囊外所有透明区域不属于软件可点击区域。

打开一级列表：

- 胶囊主窗口不变宽、不变高、不抖动。
- 独立浮层面板窗口显示在触发按钮附近。
- 面板靠近屏幕边缘时自动向屏幕内侧展开。
- 胶囊吸附在左侧时，面板出现在右侧内侧；吸附在右侧时，面板出现在左侧内侧。
- 胶囊靠近底部时，面板向上展开；靠近顶部时，面板向下展开。
- 多屏交界处优先使用胶囊所在屏幕的工作区，不能把面板放到相邻屏幕外侧或被任务栏遮挡。

打开二级下拉：

- 设备、语言、主题、画质、帧率、摄像头预设等下拉列表不再撑大一级面板。
- 二级列表使用独立小窗口，锚定到对应选择按钮。
- 点击列表内选项不会被误判为外部点击。
- 点击软件外部或切换其他按钮时，二级列表先关闭，再按需要关闭一级面板。

录制中：

- 开始录制后隐藏或关闭所有一级/二级浮层。
- 胶囊进入紧凑态时不因为旧浮层状态再次扩大窗口。
- 录制中允许的按钮，例如停止、暂停、画板，应继续可用；被锁定的来源、音频、摄像头、语言和设置不显示浮层。

## 推荐架构

### 后端窗口

新增一个可复用一级浮层窗口：

```text
floating-panel
URL: /#/floating-panel
Frameless: true
AlwaysOnTop: true
HiddenOnTaskbar: true
DisableResize: true
BackgroundType: transparent
```

新增一个可复用二级选择窗口：

```text
floating-select
URL: /#/floating-select
Frameless: true
AlwaysOnTop: true
HiddenOnTaskbar: true
DisableResize: true
BackgroundType: transparent
```

不建议为每个按钮创建一个永久窗口。一个一级浮层窗口加一个二级选择窗口足够，切换内容时更新状态和尺寸即可，窗口数量稳定，也方便处理 z-order、失焦和关闭。

### 后端服务接口

新增统一浮层服务合同：

```go
type FloatingPanelKind string

const (
  FloatingPanelSource   FloatingPanelKind = "source"
  FloatingPanelAudio    FloatingPanelKind = "audio"
  FloatingPanelCamera   FloatingPanelKind = "camera"
  FloatingPanelBoard    FloatingPanelKind = "board"
  FloatingPanelLanguage FloatingPanelKind = "language"
  FloatingPanelSettings FloatingPanelKind = "settings"
)

type FloatingAnchorRect struct {
  X      int `json:"x"`
  Y      int `json:"y"`
  Width  int `json:"width"`
  Height int `json:"height"`
}

type FloatingPanelRequest struct {
  Kind       FloatingPanelKind `json:"kind"`
  Anchor     FloatingAnchorRect `json:"anchor"`
  DockSide   string             `json:"dockSide"`
  Width      int                `json:"width"`
  Height     int                `json:"height"`
  MinWidth   int                `json:"minWidth,omitempty"`
  MaxHeight  int                `json:"maxHeight,omitempty"`
  Token      uint64             `json:"token"`
}
```

后端方法：

- `ShowFloatingPanel(req FloatingPanelRequest) (FloatingPanelState, error)`
- `UpdateFloatingPanel(req FloatingPanelRequest) (FloatingPanelState, error)`
- `HideFloatingPanel(token uint64) error`
- `ShowFloatingSelect(req FloatingSelectRequest) (FloatingSelectState, error)`
- `HideFloatingSelect(token uint64) error`
- `SetFloatingPanelHitRegions(req CapsuleWindowHitRegionsRequest) error`
- `SetFloatingSelectHitRegions(req CapsuleWindowHitRegionsRequest) error`

`FloatingPanelState` 返回实际窗口位置、尺寸、展开方向和所在屏幕 ID，方便前端记录日志和做 e2e 断言。

### 前端路由

新增两个前端入口：

- `/#/floating-panel`
- `/#/floating-select`

主胶囊仍然走 `/`。

前端组件拆分：

```text
src/components/panels/SourcePanel.tsx
src/components/panels/AudioPanel.tsx
src/components/panels/CameraPanel.tsx
src/components/panels/BoardToolsPanel.tsx
src/components/panels/LanguagePanel.tsx
src/components/panels/QuickSettingsPanel.tsx
src/components/floating/FloatingPanelWindow.tsx
src/components/floating/FloatingSelectWindow.tsx
src/components/floating/floatingPosition.ts
```

`App.tsx` 不再直接渲染 `.popover`。主窗口只保留按钮状态和胶囊布局；按钮点击后调用 `ShowFloatingPanel()`。

### 状态同步

多窗口后不能让每个窗口维护一套互相不知道的本地状态。必须把状态分层：

- 录制状态：继续以 `recording.status` 事件为准。
- 设置状态：继续以 `settings.changed` 事件为准。
- 音频状态：继续以 `audio.state` / `audio.level` 为准。
- 摄像头状态：继续通过 `PatchCameraState()` 修改，并由 `settings.changed` 回流。
- 语言/主题/画质/帧率/倒计时：继续通过 settings preference patch 修改。
- 当前视频来源和录制模式：新增显式 patch 方法或复用 settings source 持久化，不能只存在 `App.tsx` 本地。

建议新增：

```go
type SourceStatePatchRequest struct {
  RecordingMode string `json:"recordingMode,omitempty"`
  SourceID      string `json:"sourceId,omitempty"`
  SourceType    string `json:"sourceType,omitempty"`
}
```

并提供：

- `PatchSourceState(req SourceStatePatchRequest) (settings.Settings, error)`

这样浮层窗口选择屏幕、区域、锁定窗口或 audio-only 模式时，主胶囊会通过 `settings.changed` 更新显示，避免窗口之间状态不同步。

### 定位规则

浮层定位必须用屏幕工作区，而不是只看 `window.innerWidth/innerHeight`。

定位输入：

- 胶囊窗口屏幕坐标。
- 触发按钮在胶囊内的 client rect。
- 胶囊当前 dock side：`none | left | right | top | bottom`。
- `Screens.GetAll()` 的 display bounds / work area。
- 目标面板的 preferred width / height。

定位算法：

1. 把按钮 client rect 转换为全局屏幕 rect。
2. 选择工作区：优先取锚点中心所在屏幕；如果锚点在多屏接缝处，取与胶囊窗口可见区域重叠最大的屏幕。
3. 根据 dock side 决定候选方向：
   - `left`：优先右侧。
   - `right`：优先左侧。
   - `top`：优先下方。
   - `bottom`：优先上方。
   - `none`：按可用空间选择下方或上方。
4. 对候选窗口执行 clamp，保证不超出工作区和任务栏。
5. 如果一级面板高度超过工作区，固定窗口高度并让面板内部滚动。
6. 二级选择窗口优先与选择按钮同宽，最大高度不超过工作区剩余空间；不足时向上展开。

### 点击穿透和命中区域

一级/二级浮层窗口都必须有自己的命中区域，不再加入胶囊主窗口命中区域。

Windows：

- 复用现有 `SetWindowRgn` / `WM_NCHITTEST` 思路，但把 `capsuleHitRegions` 泛化为多窗口 hit region 管理。
- 胶囊、浮层面板、浮层选择窗口、标注 overlay 都按 window name 或 native handle 分开维护。

macOS / Linux：

- 优先让窗口 bounds 精确贴合可见面板，不创建大面积透明保留区。
- 透明圆角外的少量区域由窗口精确尺寸和圆角视觉共同控制；如后续 Wails 暴露平台级 shape/hit-test，再接入同一 `SetFloatingPanelHitRegions()` 合同。

全平台：

- 列表外点击应直接落到后面的桌面或应用。
- 如果需要点击外部关闭，不能依赖一个全屏透明遮罩；应使用窗口失焦、全局快捷 Esc、主胶囊按钮再次点击和状态 token 关闭。

### 失焦关闭

需要处理跨窗口焦点跳转，避免“点列表内部也关闭”的旧问题。

规则：

- 一级面板打开后，点击胶囊触发按钮再次关闭。
- 点击一级面板内部不关闭。
- 打开二级选择窗口时，一级面板保持打开。
- 点击二级选择窗口内部不关闭。
- 焦点从一级面板切到二级选择窗口时不关闭。
- 焦点从一级/二级浮层切到胶囊按钮时按按钮逻辑处理。
- 焦点切到其他应用、桌面或非关联窗口时，延迟 `80-120ms` 关闭二级窗口，再关闭一级窗口。
- `Esc` 先关闭二级窗口；没有二级窗口时关闭一级窗口。

实现上使用 token：

- 每次打开一级面板生成 `panelToken`。
- 每次打开二级列表生成 `selectToken`。
- 关闭时只关闭当前 token，旧异步 blur / resize 回调不能关闭新打开的面板。

## 面板尺寸建议

保持内容密度，但不要让面板覆盖半个屏幕。

```text
Source panel:   380-440 px wide, max 520 px high
Audio panel:    420-460 px wide, max 500 px high
Camera panel:   420-480 px wide, max 560 px high
Board panel:    430-500 px wide, max 520 px high
Language panel: 280-360 px wide, max 180 px high
Settings quick: 520-640 px wide, max 680 px high
Select list:    anchor width to 360 px, max 280 px high
```

一级面板内部使用固定 header + scroll body，不允许因为选项增多改变窗口外部锚点。

## 开发顺序

### P0：窗口与定位基建

- 新增 `floating-panel` 和 `floating-select` 窗口。
- 新增 `ShowFloatingPanel()` / `HideFloatingPanel()`。
- 新增 `floatingPosition.ts`，复用并抽出当前 `capsuleWorkAreas()`、dock side 和 clamp 逻辑。
- 胶囊打开浮层时不再调用 `setCapsuleWindowExpanded(true)`。
- 打开/关闭浮层时胶囊主窗口尺寸必须保持 collapsed。

### P1：迁移一级面板

- 把 source/audio/camera/board/language 从 `App.tsx` 拆为独立 panel components。
- `App.tsx` 按钮点击只发送浮层请求。
- 浮层窗口按 kind 渲染对应 panel。
- 录制开始、暂停、停止、恢复时按状态隐藏不允许展示的面板。

### P2：迁移二级下拉

- 重构 `SelectMenu`：
  - 桌面 Wails 环境使用 `floating-select`。
  - 浏览器/e2e 环境保留 inline fallback。
- 设备、主题、画质、帧率、倒计时、PIP preset 等下拉统一走同一个选择窗口。
- 下拉选项点击后通过 callback / backend patch 修改状态，并关闭二级窗口。

### P3：状态中心化

- 新增 `PatchSourceState()` 或等价 source state service。
- 主胶囊、浮层面板和浮层选择窗口都从 settings/events 读取状态。
- 删除只在主窗口本地生效、浮层窗口无法同步的状态路径。

### P4：命中区域和平台验收

- Windows 泛化 hit region 管理到多个窗口。
- macOS/Linux 验证透明窗口 bounds 和点击穿透。
- 增加日志：
  - `floating-panel.show`
  - `floating-panel.position`
  - `floating-panel.hide`
  - `floating-select.show`
  - `floating-select.hide`
  - `floating-panel.outside-close`

## 验收标准

基础验收：

- 胶囊未打开列表时，主窗口真实尺寸等于 collapsed 尺寸。
- 打开 source/audio/camera/board/language/settings 面板时，主窗口尺寸不变化。
- 面板关闭后，主窗口位置和尺寸不抖动。
- 列表外区域可以点击到桌面或后面的应用。
- 点击列表内部不会误关闭。
- `Esc`、再次点击触发按钮、点击外部都能正确关闭。

边缘验收：

- 胶囊靠近屏幕顶部，面板向下展开。
- 胶囊靠近屏幕底部，面板向上展开。
- 胶囊左侧吸附，面板向右侧内侧展开。
- 胶囊右侧吸附，面板向左侧内侧展开。
- 两块屏幕相接处，面板仍出现在胶囊所在屏幕的内侧，不跨到不可见区域。
- 二级下拉在边缘处同样向内展开，不撑大一级面板。

录制验收：

- 开始录制时所有配置面板自动隐藏。
- 录制中胶囊紧凑态不因历史 `activePanel` 重新展开。
- 停止录制后，胶囊恢复可用状态，但不自动弹出之前的面板。

回归验收：

- 语言切换、主题切换、画质、帧率、倒计时仍全局生效。
- 音频设备、麦克风开关、RNNoise、麦克风电平仍正常。
- 摄像头开关、PIP preset、形状、镜像、大小仍正常。
- 截图、画板、截图历史、钉图入口仍正常。
- 设置面板不出现横向滚动条。

自动化建议：

- Playwright 覆盖按钮打开浮层后胶囊尺寸不变。
- Playwright 覆盖二级下拉点击内部不关闭，点击外部关闭。
- Go 单测覆盖浮层定位算法：顶部、底部、左侧、右侧、多屏接缝、负坐标屏幕、高 DPI 工作区。
- Windows smoke 覆盖 hit region：浮层外点击不被软件截获。

## 不采用的方案

不建议继续只修当前 `.popover`：

- 它仍然依赖胶囊主窗口扩大，不能满足“未唤出列表只占胶囊大小”。
- 透明区域、点击穿透、失焦关闭和边缘定位会继续互相牵制。
- 下拉选择框越多，主窗口命中区域越复杂，后续仍容易复发闪烁和误关闭。

不建议为每个按钮永久创建一个窗口：

- 窗口数量太多，z-order、焦点、隐藏和主题同步成本会上升。
- 一个一级浮层窗口 + 一个二级选择窗口即可覆盖所有列表。

不建议用全屏透明遮罩处理外部点击：

- 这会重新制造“软件占用整块屏幕”的问题。
- 鼠标点击穿透和录制稳定性会变差。

## 最终维护标准

以后所有胶囊内“打开后会展示列表/面板”的功能，都必须走 floating panel 合同。

允许留在胶囊主窗口内的内容只有：

- 胶囊本体按钮。
- 录制状态和计时。
- 必须常驻的极小状态 chip。
- 不改变主窗口尺寸的确认状态。

不允许再把大型 `.popover`、`.select-menu-list`、设置表单或截图历史直接渲染进胶囊主窗口并触发主窗口 expanded。
