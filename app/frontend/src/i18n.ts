import type {
  CaptureCapabilityPermission,
  CaptureCapabilityStatus,
  CaptureSourceType,
  LocaleCode,
  PIPShape,
  PIPPreset,
  RecordingMode,
  RecordingPreflightStatus,
  RecordingQuality,
  RecordingState,
  ShortcutAction,
  ThemeCode,
} from './services/mockBackend'

export type StatusMessageKey =
  | 'waiting'
  | 'preparing'
  | 'started'
  | 'failedToStart'
  | 'finalizing'
  | 'ready'
  | 'failedToStop'
  | 'pausing'
  | 'paused'
  | 'resuming'
  | 'resumed'
  | 'preflightBlocked'
  | 'exportingPip'
  | 'pipReady'
  | 'backendMessage'

export type RecoveryMessageKey = 'recovering' | 'recovered' | 'refreshed' | 'failed'
export type StorageMessageKey = 'applying' | 'changed' | 'failed'
export type SourceSelectionMessageKey = 'regionSelecting' | 'regionSelected' | 'regionCancelled' | 'regionTooSmall' | 'sourceQueued'

export type RecorderCopy = {
  localeNames: Record<LocaleCode, string>
  themeNames: Record<ThemeCode, string>
  aria: {
    settingsDialog: string
    settingsShell: string
    recorderShell: string
    recorderControls: string
    moveRecorder: string
    audioCameraControls: string
    openAudioSettings: string
    openCameraSettings: string
    startRecording: string
    stopRecording: string
    pauseRecording: string
    resumeRecording: string
    selectLanguage: string
    openSettings: string
    openWhiteboard: string
    openLastRecording: string
    closeApplication: string
    recordingMode: string
    menu: (panel: string) => string
    microphoneLevel: string
    regionOverlay: string
  }
  common: {
    close: string
    cancel: string
    apply: string
    applying: string
    off: string
    notRun: string
    clean: string
    unknownSpace: string
    backend: string
    status: string
    preflight: string
    recovery: string
    packageCount: (count: number) => string
  }
  recordingModes: Record<RecordingMode, string>
  sourceTypes: Record<CaptureSourceType, string>
  sourceAudioOnly: {
    name: string
    systemAndMic: string
    systemOnly: string
    micOnly: string
    noAudio: string
  }
  sourceNames: Record<string, string>
  sourceMeta: Record<string, string>
  sourceUnavailable: string
  sourceGroups: {
    screen: string
    region: string
    window: string
  }
  sourceActions: {
    screenLabel: (index: number) => string
    chooseRegion: string
    chooseLockedWindow: string
    backToSources: string
    lockedWindowHint: string
    noWindows: string
    regionUnavailable: string
  }
  mediaDeviceNames: Record<string, string>
  settings: {
    title: string
    storage: string
    storageDetail: string
    openRecordings: string
    storageHealth: string
    dataRoot: string
    dataRootDetail: string
    appData: string
    settingsFile: string
    language: string
    theme: string
    themeDetail: string
    startAtLogin: string
    startAtLoginDetail: string
    autoOcr: string
    autoOcrDetail: string
    ocrModels: string
    ocrModelsDetail: string
    ocrModelsLoading: string
    ocrModelsUnavailable: string
    ocrStatusLabels: Record<string, string>
    ocrModelChannelLabels: Record<string, string>
    ocrModelPackagePath: string
    ocrModelPackageImport: string
    ocrModelPackageEmpty: string
    ocrModelImporting: string
    ocrModelImported: string
    ocrModelDownload: string
    ocrModelDownloading: string
    ocrModelDownloadQueued: string
    ocrModelCancelDownload: string
    ocrModelCancellingDownload: string
    ocrModelDownloadCancelled: string
    ocrModelDownloadUnavailable: string
    ocrModelDownloadInstalled: string
    ocrModelDownloadFailed: string
    ocrModelDownloadSize: string
    ocrModelCatalogRefresh: string
    ocrModelCatalogRefreshing: string
    ocrModelCatalogRefreshed: string
    ocrModelCatalogDetail: string
    ocrModelUse: string
    ocrModelActivating: string
    ocrModelSwitchConfirm: (modelName: string) => string
    ocrModelSwitchRisk: string
    ocrModelConfirmUse: string
    ocrModelCancelUse: string
    ocrModelActive: string
    ocrModelVerified: string
    ocrModelInvalid: string
    ocrModelMissing: string
    ocrModelRemove: string
    ocrModelRemoving: string
    ocrModelRemoved: string
    ocrModelActivated: string
    ocrModelActionFailed: string
    ocrModelSmokeReady: string
    ocrModelWorkerStatus: string
    ocrModelRuntime: string
    ocrModelRefresh: string
    ocrTranslation: string
    ocrTranslationDetail: string
    ocrTranslationProvider: string
    ocrTranslationProviderLabels: Record<string, string>
    ocrTranslationBaseUrl: string
    ocrTranslationBaseUrlDetail: string
    ocrTranslationApiKey: string
    ocrTranslationApiKeyDetail: string
    ocrTranslationApiKeySaved: string
    ocrTranslationModel: string
    ocrTranslationSourceLanguage: string
    ocrTranslationTargetLanguage: string
    ocrTranslationPrivacy: string
    ocrTranslationPrivacyDetail: string
    shortcuts: string
    shortcutSummary: string
    shortcutDetail: string
    shortcutHint: string
    shortcutRecord: string
    shortcutRecording: string
    shortcutInvalid: string
    shortcutConflict: (action: string) => string
    shortcutActionLabels: Record<ShortcutAction, string>
    recordingBackend: string
    recordingBackendDetail: string
    preflight: string
    preflightAction: string
    preflightRerun: string
    preflightRunning: string
    preflightPendingDetail: string
    recordingPackage: string
    openPackage: string
    noRecordingPackage: string
    packageContentDetail: string
    exportPackage: string
    exporting: string
    exportPackageValue: string
    exportPackageDetail: string
    exportPlan: string
    exportPlanLoading: string
    exportPlanUnavailable: string
    exportPlanPendingDetail: string
    exportPlanPip: string
    exportPlanNoPip: string
    exportPlanAnnotationsOff: string
    exportPlanNoAnnotations: string
    exportPlanRenderedSegments: (segments: number, events: number) => string
    exportPlanSnapshotSegments: (segments: number, events: number) => string
    exportPlanSnapshotFallback: (mode: string, events: number) => string
    exportPlanOutput: (path: string) => string
    exportPlanRange: (start: string, end: string) => string
    exportPlanOpenEnded: string
    exportPlanWarnings: (count: number) => string
    exportPlanTimelineTitle: string
    exportPlanTimelineStats: (events: string, snapshots: string, skipped: number) => string
    exportPlanElementTimelineStats: (frames: number, active: number, missing: number) => string
    exportPlanSegmentLabel: (index: number, start: string, end: string, size: string) => string
    exportPlanSegmentMore: (count: number) => string
    includeAnnotations: string
    includeAnnotationsDetail: string
    exportReady: (path: string) => string
    exportFailed: string
    quality: string
    qualityDetail: string
    fps: string
    captureCursor: string
    countdown: string
    recovery: string
    recoveryCleanDetail: string
    recoveryFoundDetail: (count: number) => string
    recover: string
    recovering: string
    platform: string
    release: string
  }
  closeDialog: {
    idleTitle: string
    recordingTitle: string
    idleMessage: string
    recordingMessage: string
    confirmIdle: string
    confirmRecording: string
  }
  panels: {
    audio: string
    systemAudio: string
    systemAudioDevice: string
    microphone: string
    rnnoise: string
    microphoneDevice: string
    noMicrophones: string
    microphoneLevelLive: string
    microphoneLevelWaiting: string
    microphoneLevelOff: string
    microphoneLevelUnavailable: string
    microphoneLevelError: string
    cameraSidecar: string
    cameraDevice: string
    cameraEnabled: string
    cameraOff: string
    pipPreset: string
    pipShape: string
    pipMirror: string
    pipSize: string
    pipEdge: string
    pipEdit: string
  }
  pipOverlay: {
    label: string
    move: string
    resize: string
    mirror: string
    close: string
    cameraUnavailable: string
    cameraPreparing: string
  }
  whiteboard: {
    title: string
    subtitle: string
    open: string
    close: string
    save: string
    saving: string
    saved: string
    saveFailed: string
    loading: string
    exportPng: string
    exportSvg: string
    exportExc: string
    exported: (path: string) => string
    exportFailed: string
    clear: string
    clearConfirm: string
    undo: string
    redo: string
    reselectRegion: string
    select: string
    hand: string
    pen: string
    laser: string
    arrow: string
    line: string
    rectangle: string
    ellipse: string
    text: string
    eraser: string
    strokeColor: string
    strokeWidth: string
    opacity: string
    thin: string
    medium: string
    bold: string
    unsaved: string
    ready: string
    recognizeText: string
    recognizeSelectedImage: string
    noImageSelection: string
    openOcrResult: string
    toggleOcrBlocks: string
    ocrBlocksHidden: string
    insertOcrText: string
    translateOcrText: string
    insertOcrTranslation: string
    ocrTextInserted: string
    ocrTranslationInserted: string
    ocrPreparing: string
    ocrQueued: string
    ocrStatusRunning: string
    ocrStatusReady: string
    ocrStatusFailed: string
  }
  screenshot: {
    tools: string
    toolsShort: string
    region: string
    full: string
    fullDetail: string
    window: string
    windowDetail: string
    focusedWindow: string
    focusedWindowDetail: string
    scrolling: string
    scrollingDetail: string
    scrollingPreparing: string
    scrollingStarted: string
    scrollingUnavailable: string
    selecting: string
    captured: (width: number, height: number) => string
    captureFailed: string
    history: string
    historyDetail: string
    empty: string
    open: string
    openFolder: string
    annotate: string
    pin: string
    pinned: string
    fix: string
    fixed: string
    unfix: string
    delete: string
    deleted: string
    deleteFailed: string
    ocr: string
    ocrStatusNone: string
    ocrStatusQueued: string
    ocrStatusRunning: string
    ocrStatusReady: string
    ocrStatusFailed: string
    recognizeText: string
    retryOcr: string
    openOcrResult: string
    copyText: string
    copiedText: string
    copyTextEmpty: string
    copyTranslation: string
    copiedTranslation: string
    translateText: string
    translationUnavailable: string
    translationPrivacyRequired: string
    translationMissingBaseUrl: string
    translationMissingApiKey: string
    translationMissingModel: string
    translationWorking: string
    translationReady: string
    translationFailed: string
    ocrQueued: string
    ocrResult: string
    ocrBlocks: (count: number) => string
    ocrNoText: string
  }
  statusChips: Record<RecordingState, string>
  statusMessages: Record<StatusMessageKey, string>
  recordingQualityLabels: Record<RecordingQuality, string>
  pipPresetLabels: Record<PIPPreset, string>
  pipShapeLabels: Record<PIPShape, string>
  capabilityStatusLabels: Record<CaptureCapabilityStatus, string>
  capabilityPermissionLabels: Record<CaptureCapabilityPermission, string>
  capabilityLabels: Record<string, string>
  capabilityDetails: Record<string, string>
  preflightLabels: Record<RecordingPreflightStatus, string>
  preflightMessages: Record<RecordingPreflightStatus, string>
  preflightCheckLabels: Record<string, string>
  preflightCheckDetails: Record<string, string>
  storageStatusLabels: Record<RecordingPreflightStatus, string>
  storage: {
    writable: string
    notWritable: string
    recommendedFreeSpace: (minimum: string) => string
    readyDetail: string
    warningDetail: string
    blockedDetail: string
  }
  recoveryMessages: Record<RecoveryMessageKey, string> & {
    recoveredCount: (count: number) => string
  }
  storageMessages: Record<StorageMessageKey, string> & {
    changedTo: (path: string) => string
  }
  sourceSelectionMessages: Record<SourceSelectionMessageKey, string> & {
    regionSelectedSize: (width: number, height: number) => string
  }
  regionOverlay: {
    cancel: string
    saveScreenshot: string
    esc: string
  }
  strip: {
    micEnhancementOn: string
    micEnhancementOff: string
    recoveryClean: string
    recoveryPackages: (count: number) => string
  }
}

const zhCN: RecorderCopy = {
  localeNames: {
    'zh-CN': '简体中文',
    en: 'English',
  },
  themeNames: {
    'night-teal': '暗夜青',
    'mountain-green': '远山绿',
    'sky-blue': '天空蓝',
    'sunset-yellow': '落晖黄',
    'ink-purple': '烟墨紫',
    'sage-gray': '草木灰',
  },
  aria: {
    settingsDialog: '设置',
    settingsShell: 'RecordingFreedom 设置',
    recorderShell: 'RecordingFreedom 录制器',
    recorderControls: '胶囊录制控制',
    moveRecorder: '移动录制器',
    audioCameraControls: '音频和摄像头控制',
    openAudioSettings: '打开音频设置',
    openCameraSettings: '打开摄像头设置',
    startRecording: '开始录制',
    stopRecording: '结束录制',
    pauseRecording: '暂停录制',
    resumeRecording: '继续录制',
    selectLanguage: '选择语言',
    openSettings: '打开设置',
    openWhiteboard: '打开画板',
    openLastRecording: '打开最近录制内容',
    closeApplication: '关闭软件',
    recordingMode: '录制模式',
    menu: (panel) => `${panel} 菜单`,
    microphoneLevel: '麦克风音量',
    regionOverlay: '区域录制选择器',
  },
  common: {
    close: '关闭',
    cancel: '取消',
    apply: '应用',
    applying: '应用中',
    off: '关闭',
    notRun: '未运行',
    clean: '正常',
    unknownSpace: '空间未知',
    backend: '后端',
    status: '状态',
    preflight: '预检',
    recovery: '恢复',
    packageCount: (count) => `${count} 个录制包`,
  },
  recordingModes: {
    video: '视频',
    audio: '音频',
  },
  sourceTypes: {
    screen: '单个屏幕',
    'all-screens': '全部屏幕',
    region: '区域',
    window: '锁定窗口',
    application: '程序',
  },
  sourceAudioOnly: {
    name: '单独录音',
    systemAndMic: '系统声音 + 麦克风',
    systemOnly: '系统声音',
    micOnly: '麦克风',
    noAudio: '未选择音频',
  },
  sourceNames: {
    'screen:primary': '内置屏幕',
    'all-screens:virtual-desktop': '全部屏幕',
    'region:custom': '自定义区域',
    'window:browser': '浏览器预览',
    'application:editor': '代码编辑器',
    'screen:native-backend-queued': '原生屏幕源',
    'window:native-backend-queued': '原生窗口源',
    'application:native-backend-queued': '原生程序源',
  },
  sourceMeta: {
    'screen:primary': '3024 x 1964 · 源像素',
    'all-screens:virtual-desktop': '多屏幕合成排期中',
    'region:custom': '区域裁剪写盘排期中',
    'window:browser': '单个应用窗口',
    'application:editor': '应用程序组',
  },
  sourceUnavailable: '原生采集后端排期中',
  sourceGroups: {
    screen: '屏幕',
    region: '区域',
    window: '锁定窗口',
  },
  sourceActions: {
    screenLabel: (index) => `屏幕 ${index}`,
    chooseRegion: '框选',
    chooseLockedWindow: '选择窗口',
    backToSources: '返回来源',
    lockedWindowHint: '锁定一个可见窗口',
    noWindows: '未检测到可锁定窗口',
    regionUnavailable: '区域选择器暂不可用',
  },
  mediaDeviceNames: {
    'system-audio:default': '默认系统声音',
    'microphone:default': '默认麦克风',
    'microphone:browser-preview': '浏览器预览麦克风',
    'camera:default': '默认摄像头',
    'camera:facetime-hd': 'FaceTime HD 摄像头',
    'camera:usb-capture': 'USB 采集摄像头',
  },
  settings: {
    title: '设置',
    storage: '存储目录',
    storageDetail: '所有新的录制包都会创建在软件当前目录或你设置的存储根目录下的 data/video 中。',
    openRecordings: '打开目录',
    storageHealth: '存储状态',
    dataRoot: '存储根目录',
    dataRootDetail: '只修改录制数据保存位置，不会移动软件本体；录制仍会写入其下方的 data/video。',
    appData: '当前存储根目录',
    settingsFile: '设置文件',
    language: '语言',
    theme: '主题',
    themeDetail: '立即应用到胶囊、设置页和下拉列表，并会在下次启动时保留。',
    startAtLogin: '开机自启',
    startAtLoginDetail: '默认关闭。开启后会随系统登录自动启动，并直接最小化到托盘。',
    autoOcr: '截图后自动识别文字',
    autoOcrDetail: '默认关闭。开启后，区域、全屏和滚动截图保存后会在后台排队进行本地 OCR。',
    ocrModels: 'OCR 模型',
    ocrModelsDetail: '本地模型优先。导入通过 RecordingFreedom 校验的模型包后，才能切换为 active 模型。',
    ocrModelsLoading: '读取模型状态中',
    ocrModelsUnavailable: 'OCR 模型状态不可用',
    ocrStatusLabels: {
      ready: '可用',
      'no-model': '缺少模型',
      'model-invalid': '模型无效',
      'worker-absent': '缺少 worker',
      'worker-unavailable': 'worker 不可用',
    },
    ocrModelChannelLabels: {
      stable: '稳定',
      latest: '最新',
      quality: '高质量',
    },
    ocrModelPackagePath: '模型包 .zip 或解压目录路径',
    ocrModelPackageImport: '导入',
    ocrModelPackageEmpty: '请先输入本地模型包 .zip 或解压目录路径',
    ocrModelImporting: '导入中',
    ocrModelImported: '模型包已导入并通过校验',
    ocrModelDownload: '下载',
    ocrModelDownloading: '下载中',
    ocrModelDownloadQueued: '模型下载已开始，完成后仍需手动确认切换',
    ocrModelCancelDownload: '取消下载',
    ocrModelCancellingDownload: '取消中',
    ocrModelDownloadCancelled: '模型下载已取消',
    ocrModelDownloadUnavailable: '暂无 RecordingFreedom 已校验下载包',
    ocrModelDownloadInstalled: '模型已下载并通过校验',
    ocrModelDownloadFailed: '模型下载失败',
    ocrModelDownloadSize: '下载大小',
    ocrModelCatalogRefresh: '刷新目录',
    ocrModelCatalogRefreshing: '刷新目录中',
    ocrModelCatalogRefreshed: '模型目录已刷新，下载仍需手动触发',
    ocrModelCatalogDetail: '从 RecordingFreedom release catalog 获取已校验模型包元数据；不会自动下载、安装或切换模型。',
    ocrModelUse: '使用',
    ocrModelActivating: '切换中',
    ocrModelSwitchConfirm: (modelName) => `确认切换到 ${modelName}？`,
    ocrModelSwitchRisk: '切换只会在模型已安装、校验和 smoke 资产通过后执行；失败时保持当前 active 模型不变。',
    ocrModelConfirmUse: '确认切换',
    ocrModelCancelUse: '取消',
    ocrModelActive: '当前',
    ocrModelVerified: '已校验',
    ocrModelInvalid: '无效',
    ocrModelMissing: '缺失',
    ocrModelRemove: '删除',
    ocrModelRemoving: '删除中',
    ocrModelRemoved: '模型已删除',
    ocrModelActivated: '已切换 OCR 模型',
    ocrModelActionFailed: 'OCR 模型操作失败',
    ocrModelSmokeReady: 'smoke 资产已就绪',
    ocrModelWorkerStatus: 'worker',
    ocrModelRuntime: 'runtime',
    ocrModelRefresh: '刷新',
    ocrTranslation: 'OCR 翻译',
    ocrTranslationDetail: '默认关闭。只有选择 provider、填写配置并确认隐私提示后，才会发送识别文字。',
    ocrTranslationProvider: '翻译提供方',
    ocrTranslationProviderLabels: {
      disabled: '关闭',
      deepl: 'DeepL',
      'openai-compatible': 'OpenAI-compatible',
    },
    ocrTranslationBaseUrl: '接口地址',
    ocrTranslationBaseUrlDetail: 'DeepL 填 translate 接口；OpenAI-compatible 填 /v1 或 /chat/completions。',
    ocrTranslationApiKey: 'API Key',
    ocrTranslationApiKeyDetail: '保存在本地受保护 secret store；输入新 key 会替换，清空并回车会删除。',
    ocrTranslationApiKeySaved: '已保存，出于安全不会回显',
    ocrTranslationModel: '模型',
    ocrTranslationSourceLanguage: '原文语言',
    ocrTranslationTargetLanguage: '目标语言',
    ocrTranslationPrivacy: '允许发送 OCR 文本进行翻译',
    ocrTranslationPrivacyDetail: '开启后，点击翻译会把本次 OCR 文本发送到你配置的翻译服务。',
    shortcuts: '快捷键',
    shortcutSummary: '全局快捷键',
    shortcutDetail: '点击修改后按下组合键即可保存；普通单字母和仅 Shift 的组合不会生效。',
    shortcutHint: '按下新的组合键，Esc 取消。',
    shortcutRecord: '修改',
    shortcutRecording: '录入中',
    shortcutInvalid: '快捷键保存失败，请换一个组合键。',
    shortcutConflict: (action) => `该快捷键已用于 ${action}`,
    shortcutActionLabels: {
      toggleRecording: '开始 / 结束录制',
      togglePause: '暂停 / 继续',
      toggleCamera: '开启 / 关闭摄像头',
      openWhiteboard: '打开画板',
      openScreenshot: '区域截图',
    },
    recordingBackend: '录制后端',
    recordingBackendDetail: '当前会使用这里显示的原生后端进行真实采集；开始录制前仍会执行预检，阻塞不可用的来源、媒体设备或存储状态。',
    preflight: '预检',
    preflightAction: '开始预检',
    preflightRerun: '重新预检',
    preflightRunning: '预检中',
    preflightPendingDetail: '点击开始预检会立即检查来源、媒体设备、后端能力、存储状态以及当前录制模式。',
    recordingPackage: '录制包',
    openPackage: '打开包',
    noRecordingPackage: '暂无真实录制包',
    packageContentDetail: '打开最近录制包目录，用于查看 manifest、媒体文件和诊断文件。',
    exportPackage: '导出视频',
    exporting: '导出中',
    exportPackageValue: 'MP4 成品',
    exportPackageDetail: '将最近录制包导出为 exports/recording.mp4；画中画和标注会按下方开关在导出阶段合成，原始 screen.mp4 保持干净。',
    exportPlan: '导出预览',
    exportPlanLoading: '读取中',
    exportPlanUnavailable: '暂无预览',
    exportPlanPendingDetail: '选择一个已完成录制包后，会在这里显示导出前的画中画和标注合成计划。',
    exportPlanPip: '含画中画',
    exportPlanNoPip: '无画中画',
    exportPlanAnnotationsOff: '不合成标注',
    exportPlanNoAnnotations: '本次导出不会合成录制标注。',
    exportPlanRenderedSegments: (segments, events) => `元素级标注 · ${segments} 段 · ${events} 个事件`,
    exportPlanSnapshotSegments: (segments, events) => `分段标注 · ${segments} 段 · ${events} 个事件`,
    exportPlanSnapshotFallback: (mode, events) => `标注快照 · ${mode || '兼容模式'} · ${events} 个事件`,
    exportPlanOutput: (path) => `输出：${path}`,
    exportPlanRange: (start, end) => `标注时间：${start} 至 ${end}`,
    exportPlanOpenEnded: '视频结束',
    exportPlanWarnings: (count) => `${count} 条导出提示`,
    exportPlanTimelineTitle: '标注时间线',
    exportPlanTimelineStats: (events, snapshots, skipped) => skipped > 0 ? `事件 ${events} · 快照 ${snapshots} · 跳过 ${skipped} 段` : `事件 ${events} · 快照 ${snapshots}`,
    exportPlanElementTimelineStats: (frames, active, missing) => missing > 0 ? `元素重建 ${frames} 帧 · 当前 ${active} 个 · 缺失 ${missing} 个 payload` : `元素重建 ${frames} 帧 · 当前 ${active} 个`,
    exportPlanSegmentLabel: (index, start, end, size) => `#${index} ${start} - ${end} · ${size}`,
    exportPlanSegmentMore: (count) => `还有 ${count} 段将在导出时合成`,
    includeAnnotations: '导出包含标注',
    includeAnnotationsDetail: '开启后，录制中的画板标注会合成进最终 MP4；关闭后只保留在录制包 annotations 目录中。',
    exportReady: (path) => `导出完成：${path}`,
    exportFailed: '导出失败，请检查录制包和 FFmpeg。',
    quality: '画质',
    qualityDetail: '会写入 manifest，真实原生编码器落地后也会读取该配置。',
    fps: '帧率',
    captureCursor: '录制鼠标',
    countdown: '倒计时',
    recovery: '崩溃恢复',
    recoveryCleanDetail: '启动扫描未发现未完成录制包。',
    recoveryFoundDetail: (count) => `发现 ${count} 个可恢复录制包。`,
    recover: '恢复',
    recovering: '恢复中',
    platform: '平台',
    release: '发布',
  },
  closeDialog: {
    idleTitle: '关闭软件',
    recordingTitle: '结束录制并关闭',
    idleMessage: '是否关闭 RecordingFreedom？',
    recordingMessage: '当前正在录制。是否结束录制并关闭软件？',
    confirmIdle: '关闭软件',
    confirmRecording: '结束并关闭',
  },
  panels: {
    audio: '音频',
    systemAudio: '系统声音',
    systemAudioDevice: '系统声音设备',
    microphone: '麦克风',
    rnnoise: 'RNNoise 降噪',
    microphoneDevice: '麦克风设备',
    noMicrophones: '未检测到可用麦克风',
    microphoneLevelLive: '正在监听麦克风',
    microphoneLevelWaiting: '等待麦克风输入',
    microphoneLevelOff: '麦克风已关闭',
    microphoneLevelUnavailable: '麦克风不可用',
    microphoneLevelError: '麦克风监听失败',
    cameraSidecar: '摄像头',
    cameraDevice: '摄像头设备',
    cameraEnabled: '摄像头已开启，录制时会合成为屏幕画中画',
    cameraOff: '摄像头已关闭',
    pipPreset: '画中画位置',
    pipShape: '画中画形状',
    pipMirror: '镜像摄像头',
    pipSize: '画中画大小',
    pipEdge: '透明边缘',
    pipEdit: '编辑画中画',
  },
  pipOverlay: {
    label: '摄像头画中画',
    move: '移动画中画',
    resize: '缩放画中画',
    mirror: '镜像摄像头',
    close: '关闭画中画',
    cameraUnavailable: '摄像头预览不可用',
    cameraPreparing: '正在打开摄像头预览',
  },
  whiteboard: {
    title: '画板',
    subtitle: '录制前和录制中都可使用',
    open: '画板',
    close: '关闭画板',
    save: '保存',
    saving: '保存中',
    saved: '已保存',
    saveFailed: '保存失败',
    loading: '正在载入画板',
    exportPng: '导出 PNG',
    exportSvg: '导出 SVG',
    exportExc: '导出 Excalidraw',
    exported: (path) => `已导出：${path}`,
    exportFailed: '导出失败',
    clear: '清空',
    clearConfirm: '再次点击清空',
    undo: '上一步',
    redo: '重做',
    reselectRegion: '重新选择画板区域',
    select: '选择',
    hand: '拖动画布',
    pen: '画笔',
    laser: '激光笔',
    arrow: '箭头',
    line: '直线',
    rectangle: '矩形',
    ellipse: '圆形',
    text: '文字',
    eraser: '橡皮',
    strokeColor: '颜色',
    strokeWidth: '线宽',
    opacity: '透明度',
    thin: '细',
    medium: '中',
    bold: '粗',
    unsaved: '未保存',
    ready: '可绘制',
    recognizeText: '识别画板文字',
    recognizeSelectedImage: '识别选中图片',
    noImageSelection: '请先选中一张图片',
    openOcrResult: '查看画板识别结果',
    toggleOcrBlocks: '显示 / 隐藏 OCR 标注',
    ocrBlocksHidden: 'OCR 标注已隐藏',
    insertOcrText: '插入识别文字',
    translateOcrText: '翻译识别文字',
    insertOcrTranslation: '插入译文',
    ocrTextInserted: '识别文字已插入画板',
    ocrTranslationInserted: '译文已插入画板',
    ocrPreparing: '正在导出画板快照',
    ocrQueued: '画板 OCR 已加入队列',
    ocrStatusRunning: '画板文字识别中',
    ocrStatusReady: '画板文字已识别',
    ocrStatusFailed: '画板 OCR 失败',
  },
  screenshot: {
    tools: '截图 / 画板',
    toolsShort: '工具',
    region: '区域截图',
    full: '全屏截图',
    fullDetail: '捕获整个虚拟桌面',
    window: '窗口截图',
    windowDetail: '捕获可识别窗口',
    focusedWindow: '焦点窗口截图',
    focusedWindowDetail: '捕获当前最前窗口',
    scrolling: '滚动截图',
    scrollingDetail: '选择滚动区域后自动拼接',
    scrollingPreparing: '请框选需要滚动截图的区域',
    scrollingStarted: '请拖动选择滚动区域',
    scrollingUnavailable: '滚动截图失败，请确认系统允许滚动控制',
    selecting: '拖动框选截图区域',
    captured: (width, height) => `截图已保存：${width} x ${height}`,
    captureFailed: '截图失败',
    history: '截图历史',
    historyDetail: '最近截图会显示在这里',
    empty: '暂无截图历史',
    open: '打开截图',
    openFolder: '打开所在目录',
    annotate: '在画板中标注',
    pin: '钉图',
    pinned: '已钉图',
    fix: '固定',
    fixed: '已固定',
    unfix: '取消固定',
    delete: '删除截图',
    deleted: '截图已删除',
    deleteFailed: '删除截图失败',
    ocr: '文字识别',
    ocrStatusNone: '未识别',
    ocrStatusQueued: '已排队',
    ocrStatusRunning: '识别中',
    ocrStatusReady: '已识别',
    ocrStatusFailed: '识别失败',
    recognizeText: '识别文字',
    retryOcr: '重新识别',
    openOcrResult: '查看识别结果',
    copyText: '复制文字',
    copiedText: '文字已复制',
    copyTextEmpty: '没有可复制的文字',
    copyTranslation: '复制译文',
    copiedTranslation: '译文已复制',
    translateText: '翻译文字',
    translationUnavailable: '翻译提供方未配置，当前不会发送任何识别文字',
    translationPrivacyRequired: '需要先在设置中确认允许发送 OCR 文本进行翻译',
    translationMissingBaseUrl: '需要先填写翻译接口地址',
    translationMissingApiKey: '需要先填写翻译 API Key',
    translationMissingModel: 'OpenAI-compatible 翻译需要先填写模型',
    translationWorking: '翻译中',
    translationReady: '翻译完成',
    translationFailed: '翻译失败',
    ocrQueued: 'OCR 已加入队列',
    ocrResult: '识别结果',
    ocrBlocks: (count) => `${count} 个文字块`,
    ocrNoText: '未识别到文字',
  },
  statusChips: {
    idle: '待机',
    preparing: '准备',
    recording: '录制',
    paused: '暂停',
    stopping: '保存',
    ready: '完成',
    failed: '失败',
  },
  statusMessages: {
    waiting: '等待录制命令',
    preparing: '正在准备录制包',
    started: '录制已开始',
    failedToStart: '录制启动失败',
    finalizing: '正在完成录制包',
    ready: '录制包已就绪',
    failedToStop: '结束录制失败',
    pausing: '正在暂停录制',
    paused: '录制已暂停',
    resuming: '正在继续录制',
    resumed: '录制已继续',
    preflightBlocked: '预检未通过，暂时无法开始录制',
    exportingPip: '正在生成屏幕画中画视频',
    pipReady: '屏幕画中画视频已生成',
    backendMessage: '后端状态已更新',
  },
  recordingQualityLabels: {
    standard: '标准',
    balanced: '均衡',
    high: '高画质',
  },
  pipPresetLabels: {
    off: '关闭',
    'bottom-right': '右下角',
    'bottom-left': '左下角',
    free: '自由布局',
  },
  pipShapeLabels: {
    circle: '圆形',
    square: '方形',
  },
  capabilityStatusLabels: {
    available: '可用',
    queued: '排期中',
    blocked: '已阻塞',
    unsupported: '不支持',
  },
  capabilityPermissionLabels: {
    'not-required': '无需授权',
    unknown: '权限待确认',
    'screen-recording': '屏幕录制权限',
    microphone: '麦克风权限',
    camera: '摄像头权限',
  },
  capabilityLabels: {
    'source-enumeration': '来源识别',
    'screen-recording': '屏幕录制',
    'window-recording': '窗口录制',
    'application-recording': '程序录制',
    'system-audio': '系统声音',
    microphone: '麦克风',
    'microphone-enhancement': '麦克风 RNNoise',
    'camera-sidecar': '摄像头',
    'pip-export': '画中画导出',
    'package-recovery': '录制包恢复',
  },
  capabilityDetails: {
    'source-enumeration': '浏览器预览使用稳定演示来源；桌面运行时会使用原生来源识别。',
    'screen-recording': '真实屏幕采集会由对应平台的原生后端实现。',
    'window-recording': '窗口目标映射会由平台原生后端处理。',
    'application-recording': '程序录制会在原生采集前按应用窗口分组。',
    'system-audio': '系统声音采集依赖平台能力，目前按平台后端逐步落地。',
    microphone: '麦克风采集使用平台原生输入链路；Windows 走 WASAPI，macOS 走 CoreAudio。',
    'microphone-enhancement': 'RNNoise 只处理麦克风 PCM；需要随包提供原生动态模块。',
    'camera-sidecar': '摄像头会随屏幕录制写入包内素材，并在导出阶段合成为画中画；Windows 使用 FFmpeg DirectShow，macOS 使用 FFmpeg AVFoundation，Linux 使用 FFmpeg v4l2。',
    'pip-export': '画中画导出会使用干净的屏幕视频和摄像头素材合成 exports/recording.mp4。',
    'package-recovery': '桌面运行时会扫描 data/video 下的 .rfrec 包并标记可恢复项。',
  },
  preflightLabels: {
    ready: '通过',
    warning: '警告',
    blocked: '阻塞',
  },
  preflightMessages: {
    ready: '可以开始录制。',
    warning: '可继续验证界面录制包；部分原生采集能力仍在排期中。',
    blocked: '需要先解决阻塞项，才能开始录制。',
  },
  preflightCheckLabels: {
    request: '录制请求',
    source: '采集来源',
    'source-backend': '来源后端',
    'system-audio-device': '系统声音设备',
    'system-audio': '系统声音',
    'microphone-device': '麦克风设备',
    microphone: '麦克风',
    'microphone-rnnoise-device': '麦克风 RNNoise',
    'microphone-rnnoise': '麦克风 RNNoise',
    'microphone-enhancement': '麦克风增强',
    'camera-device': '摄像头设备',
    'camera-sidecar': '摄像头',
    'camera-sidecar-device': '摄像头',
    'camera-native-id': '摄像头原生标识',
    'pip-export': '画中画导出',
    storage: '录制存储',
    'mock-backend': '录制后端',
    'recording-backend': '录制后端',
  },
  preflightCheckDetails: {
    request: '当前录制参数无效，请检查来源、音频或摄像头设置。',
    source: '所选来源当前不可用，请重新选择屏幕、区域或窗口。',
    'source-backend': '来源采集后端尚未完全可用。',
    'system-audio-device': '所选系统声音设备不可用。',
    'system-audio': '系统声音采集仍在平台后端排期中。',
    'microphone-device': '所选麦克风设备不可用。',
    microphone: '所选麦克风需要可用的原生输入设备。',
    'microphone-rnnoise-device': '所选麦克风当前不满足 RNNoise 处理条件。',
    'microphone-rnnoise': '当前构建未启用 RNNoise 原生降噪。',
    'microphone-enhancement': '当前构建未启用 RNNoise 原生降噪。',
    'camera-device': '所选摄像头设备不可用。',
    'camera-sidecar': '摄像头画中画需要当前平台提供真实 writer，并写入包内 webcam.mp4 或 webcam.mov。',
    'camera-sidecar-device': '所选摄像头当前不可用于画中画录制。',
    'camera-native-id': '所选摄像头缺少原生采集标识，无法交给平台 writer。',
    'pip-export': '画中画合成导出需要 FFmpeg，并会在导出阶段合成最终 MP4。',
    storage: '录制目录需要可写，并建议保留足够可用空间。',
    'mock-backend': '当前是可验证的界面录制包后端，不采集真实媒体。',
    'recording-backend': '将使用当前原生后端写入录制包。',
  },
  storageStatusLabels: {
    ready: '就绪',
    warning: '警告',
    blocked: '阻塞',
  },
  storage: {
    writable: '可写',
    notWritable: '不可写',
    recommendedFreeSpace: (minimum) => `建议可用空间：${minimum}`,
    readyDetail: '录制目录可写，空间检查通过。',
    warningDetail: '录制目录可写，但空间状态需要注意。',
    blockedDetail: '录制目录当前不可写或无法创建。',
  },
  recoveryMessages: {
    recovering: '正在恢复录制包...',
    recovered: '录制包已恢复',
    refreshed: '恢复扫描已刷新',
    failed: '恢复失败',
    recoveredCount: (count) => `已恢复 ${count} 个录制包`,
  },
  storageMessages: {
    applying: '正在应用存储根目录...',
    changed: '存储根目录已更新',
    failed: '存储目录操作失败',
    changedTo: (path) => `新的录制包将写入 ${path}`,
  },
  sourceSelectionMessages: {
    regionSelecting: '正在选择区域',
    regionSelected: '区域已选择',
    regionCancelled: '区域选择已取消',
    regionTooSmall: '选择区域过小',
    sourceQueued: '该来源等待真实采集后端',
    regionSelectedSize: (width, height) => `区域：${width} x ${height}`,
  },
  regionOverlay: {
    cancel: '取消区域选择',
    saveScreenshot: '保存截图',
    esc: 'Esc 取消',
  },
  strip: {
    micEnhancementOn: '麦克风增强：RNNoise',
    micEnhancementOff: '麦克风增强：关闭',
    recoveryClean: '恢复扫描：正常',
    recoveryPackages: (count) => `恢复：${count} 个录制包`,
  },
}

const en: RecorderCopy = {
  localeNames: {
    'zh-CN': '简体中文',
    en: 'English',
  },
  themeNames: {
    'night-teal': 'Night Teal',
    'mountain-green': 'Mountain Green',
    'sky-blue': 'Sky Blue',
    'sunset-yellow': 'Sunset Yellow',
    'ink-purple': 'Ink Purple',
    'sage-gray': 'Sage Gray',
  },
  aria: {
    settingsDialog: 'Settings',
    settingsShell: 'RecordingFreedom settings',
    recorderShell: 'RecordingFreedom recorder',
    recorderControls: 'Capsule recorder controls',
    moveRecorder: 'Move recorder',
    audioCameraControls: 'Audio and camera controls',
    openAudioSettings: 'Open audio settings',
    openCameraSettings: 'Open camera settings',
    startRecording: 'Start recording',
    stopRecording: 'Stop recording',
    pauseRecording: 'Pause recording',
    resumeRecording: 'Resume recording',
    selectLanguage: 'Select language',
    openSettings: 'Open settings',
    openWhiteboard: 'Open whiteboard',
    openLastRecording: 'Open latest recording',
    closeApplication: 'Close app',
    recordingMode: 'Recording mode',
    menu: (panel) => `${panel} menu`,
    microphoneLevel: 'Microphone level',
    regionOverlay: 'Region recording selector',
  },
  common: {
    close: 'Close',
    cancel: 'Cancel',
    apply: 'Apply',
    applying: 'Applying',
    off: 'Off',
    notRun: 'Not run',
    clean: 'Clean',
    unknownSpace: 'Unknown space',
    backend: 'Backend',
    status: 'Status',
    preflight: 'Preflight',
    recovery: 'Recovery',
    packageCount: (count) => `${count} package(s)`,
  },
  recordingModes: {
    video: 'Video',
    audio: 'Audio',
  },
  sourceTypes: {
    screen: 'Single screen',
    'all-screens': 'All screens',
    region: 'Region',
    window: 'Locked window',
    application: 'Program',
  },
  sourceAudioOnly: {
    name: 'Audio only',
    systemAndMic: 'System audio + microphone',
    systemOnly: 'System audio',
    micOnly: 'Microphone',
    noAudio: 'No audio selected',
  },
  sourceNames: {
    'screen:primary': 'Built-in Retina',
    'all-screens:virtual-desktop': 'All screens',
    'region:custom': 'Custom region',
    'window:browser': 'Browser Preview',
    'application:editor': 'Code Editor',
    'screen:native-backend-queued': 'Native Screen Source',
    'window:native-backend-queued': 'Native Window Source',
    'application:native-backend-queued': 'Native Program Source',
  },
  sourceMeta: {
    'screen:primary': '3024 x 1964 · source pixels',
    'all-screens:virtual-desktop': 'Multi-display composition queued',
    'region:custom': 'Native region crop writer queued',
    'window:browser': 'Single app window',
    'application:editor': 'Application group',
  },
  sourceUnavailable: 'Native capture backend is queued',
  sourceGroups: {
    screen: 'Screen',
    region: 'Region',
    window: 'Locked window',
  },
  sourceActions: {
    screenLabel: (index) => `Screen ${index}`,
    chooseRegion: 'Select',
    chooseLockedWindow: 'Choose window',
    backToSources: 'Back to sources',
    lockedWindowHint: 'Lock a visible window',
    noWindows: 'No lockable windows detected',
    regionUnavailable: 'Region selector is unavailable',
  },
  mediaDeviceNames: {
    'system-audio:default': 'Default System Audio',
    'microphone:default': 'Default Microphone',
    'microphone:browser-preview': 'Browser Preview Microphone',
    'camera:default': 'Default Camera',
    'camera:facetime-hd': 'FaceTime HD Camera',
    'camera:usb-capture': 'USB Capture Camera',
  },
  settings: {
    title: 'Settings',
    storage: 'Storage',
    storageDetail: 'All new recording packages are created inside data/video under the app folder or the configured storage root.',
    openRecordings: 'Open folder',
    storageHealth: 'Storage health',
    dataRoot: 'Storage root',
    dataRootDetail: 'Changes only where recording data is saved; it does not move the app itself. Recordings still go into data/video below it.',
    appData: 'Current storage root',
    settingsFile: 'Settings',
    language: 'Language',
    theme: 'Theme',
    themeDetail: 'Applies immediately to the capsule, settings, and dropdown menus, and is kept after restart.',
    startAtLogin: 'Start at login',
    startAtLoginDetail: 'Off by default. When enabled, RecordingFreedom starts on system login and opens minimized to tray.',
    autoOcr: 'Auto OCR after screenshots',
    autoOcrDetail: 'Off by default. When enabled, region, full, and scrolling screenshots queue local OCR after saving.',
    ocrModels: 'OCR models',
    ocrModelsDetail: 'Local-first OCR. Import a RecordingFreedom-verified model package before switching it active.',
    ocrModelsLoading: 'Loading model status',
    ocrModelsUnavailable: 'OCR model status unavailable',
    ocrStatusLabels: {
      ready: 'Ready',
      'no-model': 'No model',
      'model-invalid': 'Invalid model',
      'worker-absent': 'Worker missing',
      'worker-unavailable': 'Worker unavailable',
    },
    ocrModelChannelLabels: {
      stable: 'Stable',
      latest: 'Latest',
      quality: 'Quality',
    },
    ocrModelPackagePath: 'Model package .zip or extracted folder path',
    ocrModelPackageImport: 'Import',
    ocrModelPackageEmpty: 'Enter a local model package .zip or extracted folder path first',
    ocrModelImporting: 'Importing',
    ocrModelImported: 'Model package imported and verified',
    ocrModelDownload: 'Download',
    ocrModelDownloading: 'Downloading',
    ocrModelDownloadQueued: 'Model download started. Switching still requires manual confirmation after install.',
    ocrModelCancelDownload: 'Cancel download',
    ocrModelCancellingDownload: 'Cancelling',
    ocrModelDownloadCancelled: 'Model download cancelled',
    ocrModelDownloadUnavailable: 'No RecordingFreedom-verified download package yet',
    ocrModelDownloadInstalled: 'Model downloaded and verified',
    ocrModelDownloadFailed: 'Model download failed',
    ocrModelDownloadSize: 'Download size',
    ocrModelCatalogRefresh: 'Refresh catalog',
    ocrModelCatalogRefreshing: 'Refreshing catalog',
    ocrModelCatalogRefreshed: 'Model catalog refreshed. Downloads still require a manual action.',
    ocrModelCatalogDetail: 'Fetches verified model package metadata from the RecordingFreedom release catalog. It does not auto-download, install, or switch models.',
    ocrModelUse: 'Use',
    ocrModelActivating: 'Switching',
    ocrModelSwitchConfirm: (modelName) => `Switch to ${modelName}?`,
    ocrModelSwitchRisk: 'Switching runs only after the model is installed, verified, and smoke assets are present; failures keep the current active model unchanged.',
    ocrModelConfirmUse: 'Confirm switch',
    ocrModelCancelUse: 'Cancel',
    ocrModelActive: 'Active',
    ocrModelVerified: 'Verified',
    ocrModelInvalid: 'Invalid',
    ocrModelMissing: 'Missing',
    ocrModelRemove: 'Remove',
    ocrModelRemoving: 'Removing',
    ocrModelRemoved: 'Model removed',
    ocrModelActivated: 'OCR model switched',
    ocrModelActionFailed: 'OCR model action failed',
    ocrModelSmokeReady: 'smoke assets ready',
    ocrModelWorkerStatus: 'worker',
    ocrModelRuntime: 'runtime',
    ocrModelRefresh: 'Refresh',
    ocrTranslation: 'OCR translation',
    ocrTranslationDetail: 'Off by default. Recognized text is sent only after a provider is selected, configured, and privacy is confirmed.',
    ocrTranslationProvider: 'Translation provider',
    ocrTranslationProviderLabels: {
      disabled: 'Off',
      deepl: 'DeepL',
      'openai-compatible': 'OpenAI-compatible',
    },
    ocrTranslationBaseUrl: 'Endpoint',
    ocrTranslationBaseUrlDetail: 'Use the translate endpoint for DeepL; use /v1 or /chat/completions for OpenAI-compatible providers.',
    ocrTranslationApiKey: 'API key',
    ocrTranslationApiKeyDetail: 'Stored in the local protected secret store. Enter a new key to replace it, or clear and press Enter to remove it.',
    ocrTranslationApiKeySaved: 'Saved; hidden for safety',
    ocrTranslationModel: 'Model',
    ocrTranslationSourceLanguage: 'Source language',
    ocrTranslationTargetLanguage: 'Target language',
    ocrTranslationPrivacy: 'Allow sending OCR text for translation',
    ocrTranslationPrivacyDetail: 'When enabled, Translate sends this OCR text to your configured translation service.',
    shortcuts: 'Shortcuts',
    shortcutSummary: 'Global shortcuts',
    shortcutDetail: 'Click change, then press a key combination to save. Plain letters and Shift-only combos are ignored.',
    shortcutHint: 'Press a new key combination. Esc cancels.',
    shortcutRecord: 'Change',
    shortcutRecording: 'Listening',
    shortcutInvalid: 'Shortcut save failed. Choose another combination.',
    shortcutConflict: (action) => `That shortcut is already used by ${action}`,
    shortcutActionLabels: {
      toggleRecording: 'Start / stop recording',
      togglePause: 'Pause / resume',
      toggleCamera: 'Turn camera on / off',
      openWhiteboard: 'Open whiteboard',
      openScreenshot: 'Region screenshot',
    },
    recordingBackend: 'Recording backend',
    recordingBackendDetail: 'Real capture will use the native backend shown here; preflight still blocks unavailable sources, media devices, or storage before recording starts.',
    preflight: 'Preflight',
    preflightAction: 'Run preflight',
    preflightRerun: 'Run again',
    preflightRunning: 'Checking',
    preflightPendingDetail: 'Run preflight now to check the source, media devices, backend capability, storage, and current recording mode.',
    recordingPackage: 'Recording package',
    openPackage: 'Open package',
    noRecordingPackage: 'No real package yet',
    packageContentDetail: 'Open the latest recording package folder to inspect manifest, media, and diagnostics.',
    exportPackage: 'Export video',
    exporting: 'Exporting',
    exportPackageValue: 'MP4 output',
    exportPackageDetail: 'Export the latest package to exports/recording.mp4; PIP and annotations are composed during export according to the switch below, while raw screen.mp4 stays clean.',
    exportPlan: 'Export preview',
    exportPlanLoading: 'Reading',
    exportPlanUnavailable: 'No preview',
    exportPlanPendingDetail: 'Select a completed recording package to preview the PIP and annotation composition plan before export.',
    exportPlanPip: 'PIP included',
    exportPlanNoPip: 'No PIP',
    exportPlanAnnotationsOff: 'No annotations',
    exportPlanNoAnnotations: 'This export will not compose recording annotations.',
    exportPlanRenderedSegments: (segments, events) => `Element annotations · ${segments} segment(s) · ${events} event(s)`,
    exportPlanSnapshotSegments: (segments, events) => `Segmented annotations · ${segments} segment(s) · ${events} event(s)`,
    exportPlanSnapshotFallback: (mode, events) => `Annotation snapshot · ${mode || 'compat'} · ${events} event(s)`,
    exportPlanOutput: (path) => `Output: ${path}`,
    exportPlanRange: (start, end) => `Annotation time: ${start} to ${end}`,
    exportPlanOpenEnded: 'video end',
    exportPlanWarnings: (count) => `${count} export warning(s)`,
    exportPlanTimelineTitle: 'Annotation timeline',
    exportPlanTimelineStats: (events, snapshots, skipped) => skipped > 0 ? `Events ${events} · Snapshots ${snapshots} · ${skipped} skipped` : `Events ${events} · Snapshots ${snapshots}`,
    exportPlanElementTimelineStats: (frames, active, missing) => missing > 0 ? `Element reconstruction ${frames} frame(s) · ${active} active · ${missing} missing payload(s)` : `Element reconstruction ${frames} frame(s) · ${active} active`,
    exportPlanSegmentLabel: (index, start, end, size) => `#${index} ${start} - ${end} · ${size}`,
    exportPlanSegmentMore: (count) => `${count} more segment(s) will be composed during export`,
    includeAnnotations: 'Include annotations',
    includeAnnotationsDetail: 'When enabled, recording whiteboard annotations are composed into the final MP4; when disabled, they stay only in the package annotations folder.',
    exportReady: (path) => `Export complete: ${path}`,
    exportFailed: 'Export failed. Check the package and FFmpeg.',
    quality: 'Quality',
    qualityDetail: 'Saved to manifest and used by native encoders when real capture backends land.',
    fps: 'FPS',
    captureCursor: 'Capture cursor',
    countdown: 'Countdown',
    recovery: 'Recovery',
    recoveryCleanDetail: 'Startup scan found no unfinished packages.',
    recoveryFoundDetail: (count) => `Startup scan found ${count} recoverable package(s).`,
    recover: 'Recover',
    recovering: 'Recovering',
    platform: 'Platform',
    release: 'Release',
  },
  closeDialog: {
    idleTitle: 'Close app',
    recordingTitle: 'Stop recording and close',
    idleMessage: 'Close RecordingFreedom?',
    recordingMessage: 'Recording is in progress. Stop the recording and close the app?',
    confirmIdle: 'Close app',
    confirmRecording: 'Stop and close',
  },
  panels: {
    audio: 'Audio',
    systemAudio: 'System audio',
    systemAudioDevice: 'System audio device',
    microphone: 'Microphone',
    rnnoise: 'RNNoise',
    microphoneDevice: 'Microphone device',
    noMicrophones: 'No available microphones',
    microphoneLevelLive: 'Listening to microphone',
    microphoneLevelWaiting: 'Waiting for microphone input',
    microphoneLevelOff: 'Microphone off',
    microphoneLevelUnavailable: 'Microphone unavailable',
    microphoneLevelError: 'Microphone monitor failed',
    cameraSidecar: 'Camera',
    cameraDevice: 'Camera device',
    cameraEnabled: 'Camera is on and will be composed as screen picture-in-picture',
    cameraOff: 'Camera off',
    pipPreset: 'PIP preset',
    pipShape: 'PIP shape',
    pipMirror: 'Mirror camera',
    pipSize: 'PIP size',
    pipEdge: 'Transparent edge',
    pipEdit: 'Edit PIP',
  },
  pipOverlay: {
    label: 'Camera picture-in-picture',
    move: 'Move PIP',
    resize: 'Resize PIP',
    mirror: 'Mirror camera',
    close: 'Close PIP',
    cameraUnavailable: 'Camera preview unavailable',
    cameraPreparing: 'Opening camera preview',
  },
  whiteboard: {
    title: 'Whiteboard',
    subtitle: 'Available before and during recording',
    open: 'Board',
    close: 'Close whiteboard',
    save: 'Save',
    saving: 'Saving',
    saved: 'Saved',
    saveFailed: 'Save failed',
    loading: 'Loading whiteboard',
    exportPng: 'Export PNG',
    exportSvg: 'Export SVG',
    exportExc: 'Export Excalidraw',
    exported: (path) => `Exported: ${path}`,
    exportFailed: 'Export failed',
    clear: 'Clear',
    clearConfirm: 'Click again to clear',
    undo: 'Undo',
    redo: 'Redo',
    reselectRegion: 'Reselect board area',
    select: 'Select',
    hand: 'Hand',
    pen: 'Pen',
    laser: 'Laser',
    arrow: 'Arrow',
    line: 'Line',
    rectangle: 'Rectangle',
    ellipse: 'Ellipse',
    text: 'Text',
    eraser: 'Eraser',
    strokeColor: 'Color',
    strokeWidth: 'Stroke',
    opacity: 'Opacity',
    thin: 'Thin',
    medium: 'Medium',
    bold: 'Bold',
    unsaved: 'Unsaved',
    ready: 'Ready',
    recognizeText: 'Recognize board text',
    recognizeSelectedImage: 'Recognize selected image',
    noImageSelection: 'Select an image first',
    openOcrResult: 'View board OCR result',
    toggleOcrBlocks: 'Show / hide OCR blocks',
    ocrBlocksHidden: 'OCR blocks hidden',
    insertOcrText: 'Insert recognized text',
    translateOcrText: 'Translate recognized text',
    insertOcrTranslation: 'Insert translated text',
    ocrTextInserted: 'Recognized text inserted on board',
    ocrTranslationInserted: 'Translated text inserted on board',
    ocrPreparing: 'Exporting board snapshot',
    ocrQueued: 'Board OCR queued',
    ocrStatusRunning: 'Recognizing board text',
    ocrStatusReady: 'Board text recognized',
    ocrStatusFailed: 'Board OCR failed',
  },
  screenshot: {
    tools: 'Screenshot / board',
    toolsShort: 'Tools',
    region: 'Region screenshot',
    full: 'Full screenshot',
    fullDetail: 'Capture the virtual desktop',
    window: 'Window screenshot',
    windowDetail: 'Capture a detected window',
    focusedWindow: 'Focused window',
    focusedWindowDetail: 'Capture the frontmost window',
    scrolling: 'Scrolling screenshot',
    scrollingDetail: 'Select a scrollable area to stitch',
    scrollingPreparing: 'Select the area to capture while scrolling',
    scrollingStarted: 'Drag to select a scrollable area',
    scrollingUnavailable: 'Scrolling screenshot failed; make sure system scroll control is allowed',
    selecting: 'Drag to select screenshot area',
    captured: (width, height) => `Screenshot saved: ${width} x ${height}`,
    captureFailed: 'Screenshot failed',
    history: 'Screenshot history',
    historyDetail: 'Recent screenshots appear here',
    empty: 'No screenshot history',
    open: 'Open screenshot',
    openFolder: 'Open containing folder',
    annotate: 'Annotate in whiteboard',
    pin: 'Pin image',
    pinned: 'Pinned',
    fix: 'Fix',
    fixed: 'Fixed',
    unfix: 'Unfix',
    delete: 'Delete screenshot',
    deleted: 'Screenshot deleted',
    deleteFailed: 'Failed to delete screenshot',
    ocr: 'OCR',
    ocrStatusNone: 'Not recognized',
    ocrStatusQueued: 'Queued',
    ocrStatusRunning: 'Recognizing',
    ocrStatusReady: 'Recognized',
    ocrStatusFailed: 'Failed',
    recognizeText: 'Recognize text',
    retryOcr: 'Retry OCR',
    openOcrResult: 'View OCR result',
    copyText: 'Copy text',
    copiedText: 'Text copied',
    copyTextEmpty: 'No text to copy',
    copyTranslation: 'Copy translation',
    copiedTranslation: 'Translation copied',
    translateText: 'Translate text',
    translationUnavailable: 'Translation provider is not configured; no recognized text is sent',
    translationPrivacyRequired: 'Confirm in settings before sending OCR text for translation',
    translationMissingBaseUrl: 'Enter a translation endpoint first',
    translationMissingApiKey: 'Enter a translation API key first',
    translationMissingModel: 'OpenAI-compatible translation needs a model first',
    translationWorking: 'Translating',
    translationReady: 'Translation ready',
    translationFailed: 'Translation failed',
    ocrQueued: 'OCR queued',
    ocrResult: 'OCR result',
    ocrBlocks: (count) => `${count} text blocks`,
    ocrNoText: 'No text recognized',
  },
  statusChips: {
    idle: 'IDLE',
    preparing: 'READY',
    recording: 'REC',
    paused: 'PAUSED',
    stopping: 'SAVING',
    ready: 'SAVED',
    failed: 'FAILED',
  },
  statusMessages: {
    waiting: 'Waiting for recorder command',
    preparing: 'Preparing recording package',
    started: 'Recording started',
    failedToStart: 'Failed to start recording',
    finalizing: 'Finalizing recording package',
    ready: 'Recording package ready',
    failedToStop: 'Failed to stop recording',
    pausing: 'Pausing recording',
    paused: 'Recording paused',
    resuming: 'Resuming recording',
    resumed: 'Recording resumed',
    preflightBlocked: 'Preflight is blocked; recording cannot start yet',
    exportingPip: 'Creating screen picture-in-picture video',
    pipReady: 'Screen picture-in-picture video ready',
    backendMessage: 'Backend status updated',
  },
  recordingQualityLabels: {
    standard: 'Standard',
    balanced: 'Balanced',
    high: 'High',
  },
  pipPresetLabels: {
    off: 'Off',
    'bottom-right': 'Bottom right',
    'bottom-left': 'Bottom left',
    free: 'Free layout',
  },
  pipShapeLabels: {
    circle: 'Circle',
    square: 'Square',
  },
  capabilityStatusLabels: {
    available: 'Ready',
    queued: 'Queued',
    blocked: 'Blocked',
    unsupported: 'Unsupported',
  },
  capabilityPermissionLabels: {
    'not-required': 'No prompt',
    unknown: 'Permission unknown',
    'screen-recording': 'Screen Recording',
    microphone: 'Microphone',
    camera: 'Camera',
  },
  capabilityLabels: {
    'source-enumeration': 'Source Enumeration',
    'screen-recording': 'Screen Recording',
    'window-recording': 'Window Recording',
    'application-recording': 'Program Recording',
    'system-audio': 'System Audio',
    microphone: 'Microphone',
    'microphone-enhancement': 'Microphone RNNoise',
    'camera-sidecar': 'Camera',
    'pip-export': 'PIP Export',
    'package-recovery': 'Recording Package Recovery',
  },
  capabilityDetails: {
    'source-enumeration': 'Browser preview uses deterministic demo sources; Wails desktop runtime uses native source enumeration.',
    'screen-recording': 'Real screen capture is implemented by the platform native backend.',
    'window-recording': 'Window capture target mapping is handled by the platform backend.',
    'application-recording': 'Program capture groups application windows before native recording starts.',
    'system-audio': 'System audio capture is platform-specific and lands through each native backend.',
    microphone: 'Microphone capture uses the platform-native input chain: WASAPI on Windows and CoreAudio on macOS.',
    'microphone-enhancement': 'RNNoise processes microphone PCM only and requires the packaged native dynamic module.',
    'camera-sidecar': 'Camera is captured with screen recordings as package media, then composed into picture-in-picture during export; Windows uses FFmpeg DirectShow, macOS uses FFmpeg AVFoundation, and Linux uses FFmpeg v4l2.',
    'pip-export': 'PIP composition writes exports/recording.mp4 from the clean screen video plus camera media.',
    'package-recovery': 'Desktop runtime scans .rfrec packages under app-managed data/video.',
  },
  preflightLabels: {
    ready: 'Ready',
    warning: 'Warning',
    blocked: 'Blocked',
  },
  preflightMessages: {
    ready: 'Ready to start recording.',
    warning: 'Ready for UI shell package recording; some native capture checks are still queued.',
    blocked: 'Recording cannot start until blocked checks are resolved.',
  },
  preflightCheckLabels: {
    request: 'Recording Request',
    source: 'Capture Source',
    'source-backend': 'Source Backend',
    'system-audio-device': 'System Audio Device',
    'system-audio': 'System Audio',
    'microphone-device': 'Microphone Device',
    microphone: 'Microphone',
    'microphone-rnnoise-device': 'Microphone RNNoise',
    'microphone-rnnoise': 'Microphone RNNoise',
    'microphone-enhancement': 'Microphone Enhancement',
    'camera-device': 'Camera Device',
    'camera-sidecar': 'Camera',
    'camera-sidecar-device': 'Camera',
    'camera-native-id': 'Camera Native ID',
    'pip-export': 'PIP Export',
    storage: 'Recording Storage',
    'mock-backend': 'Recording Backend',
    'recording-backend': 'Recording Backend',
  },
  preflightCheckDetails: {
    request: 'The recording request is invalid; check source, audio, or camera settings.',
    source: 'The selected source is not available; choose another screen, region, or window.',
    'source-backend': 'The source capture backend is not fully available yet.',
    'system-audio-device': 'The selected system audio device is not available.',
    'system-audio': 'System audio capture is still queued for the platform backend.',
    'microphone-device': 'The selected microphone device is not available.',
    microphone: 'The selected microphone must be available through the native input backend.',
    'microphone-rnnoise-device': 'The selected microphone is not marked RNNoise eligible.',
    'microphone-rnnoise': 'This build does not enable native RNNoise suppression.',
    'microphone-enhancement': 'This build does not enable native RNNoise suppression.',
    'camera-device': 'The selected camera device is not available.',
    'camera-sidecar': 'Camera picture-in-picture requires a real platform writer and writes package-local webcam.mp4 or webcam.mov.',
    'camera-sidecar-device': 'The selected camera is not available for picture-in-picture recording.',
    'camera-native-id': 'The selected camera does not expose the native capture id required by the platform writer.',
    'pip-export': 'PIP composition export requires FFmpeg and composes the final MP4 during export.',
    storage: 'The recording directory must be writable and should have enough free space.',
    'mock-backend': 'The current backend writes a verifiable UI package but does not capture real media.',
    'recording-backend': 'The current native backend will write the recording package.',
  },
  storageStatusLabels: {
    ready: 'Ready',
    warning: 'Warning',
    blocked: 'Blocked',
  },
  storage: {
    writable: 'Writable',
    notWritable: 'Not writable',
    recommendedFreeSpace: (minimum) => `Recommended free space: ${minimum}`,
    readyDetail: 'Recording directory is writable and the space check passed.',
    warningDetail: 'Recording directory is writable, but storage needs attention.',
    blockedDetail: 'Recording directory is not writable or cannot be created.',
  },
  recoveryMessages: {
    recovering: 'Recovering packages...',
    recovered: 'Recovered packages',
    refreshed: 'Recovery scan refreshed',
    failed: 'Recovery failed',
    recoveredCount: (count) => `Recovered ${count} package(s)`,
  },
  storageMessages: {
    applying: 'Applying storage root...',
    changed: 'Storage root changed',
    failed: 'Storage operation failed',
    changedTo: (path) => `Recording packages will use ${path}`,
  },
  sourceSelectionMessages: {
    regionSelecting: 'Selecting region',
    regionSelected: 'Region selected',
    regionCancelled: 'Region selection cancelled',
    regionTooSmall: 'Selected region is too small',
    sourceQueued: 'This source is waiting for the native capture backend',
    regionSelectedSize: (width, height) => `Region: ${width} x ${height}`,
  },
  regionOverlay: {
    cancel: 'Cancel region selection',
    saveScreenshot: 'Save screenshot',
    esc: 'Esc cancel',
  },
  strip: {
    micEnhancementOn: 'Mic enhancement: RNNoise',
    micEnhancementOff: 'Mic enhancement: Off',
    recoveryClean: 'Recovery scan: clean',
    recoveryPackages: (count) => `Recovery: ${count} package(s)`,
  },
}

export const copyByLocale: Record<LocaleCode, RecorderCopy> = {
  'zh-CN': zhCN,
  en,
}
