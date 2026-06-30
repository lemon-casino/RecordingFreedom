import type {
  CaptureCapabilityPermission,
  CaptureCapabilityStatus,
  CaptureSourceType,
  LocaleCode,
  PIPPreset,
  RecordingPreflightStatus,
  RecordingQuality,
  RecordingState,
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
  | 'backendMessage'

export type RecoveryMessageKey = 'recovering' | 'recovered' | 'refreshed' | 'failed'
export type StorageMessageKey = 'applying' | 'changed' | 'failed'

export type RecorderCopy = {
  localeNames: Record<LocaleCode, string>
  aria: {
    settingsDialog: string
    settingsShell: string
    recorderShell: string
    recorderControls: string
    moveRecorder: string
    audioCameraControls: string
    toggleSystemAudio: string
    openMicrophoneSettings: string
    toggleNoiseSuppression: string
    openCameraSettings: string
    startRecording: string
    stopRecording: string
    pauseRecording: string
    resumeRecording: string
    selectLanguage: string
    openSettings: string
    menu: (panel: string) => string
    microphoneLevel: string
  }
  common: {
    close: string
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
  sourceTypes: Record<CaptureSourceType, string>
  sourceNames: Record<string, string>
  sourceMeta: Record<string, string>
  sourceUnavailable: string
  mediaDeviceNames: Record<string, string>
  settings: {
    title: string
    storage: string
    storageDetail: string
    storageHealth: string
    dataRoot: string
    dataRootDetail: string
    appData: string
    settingsFile: string
    language: string
    recordingBackend: string
    recordingBackendDetail: string
    preflight: string
    preflightPendingDetail: string
    recordingPackage: string
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
  panels: {
    systemAudio: string
    systemAudioDevice: string
    microphone: string
    rnnoise: string
    microphoneDevice: string
    cameraSidecar: string
    cameraDevice: string
    pipPreset: string
    pipPresetPreview: (label: string) => string
  }
  statusChips: Record<RecordingState, string>
  statusMessages: Record<StatusMessageKey, string>
  recordingQualityLabels: Record<RecordingQuality, string>
  pipPresetLabels: Record<PIPPreset, string>
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
  aria: {
    settingsDialog: '设置',
    settingsShell: 'RecordingFreedom 设置',
    recorderShell: 'RecordingFreedom 录制器',
    recorderControls: '胶囊录制控制',
    moveRecorder: '移动录制器',
    audioCameraControls: '音频和摄像头控制',
    toggleSystemAudio: '切换系统声音',
    openMicrophoneSettings: '打开麦克风设置',
    toggleNoiseSuppression: '切换麦克风降噪',
    openCameraSettings: '打开摄像头设置',
    startRecording: '开始录制',
    stopRecording: '结束录制',
    pauseRecording: '暂停录制',
    resumeRecording: '继续录制',
    selectLanguage: '选择语言',
    openSettings: '打开设置',
    menu: (panel) => `${panel} 菜单`,
    microphoneLevel: '麦克风音量',
  },
  common: {
    close: '关闭',
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
  sourceTypes: {
    screen: '屏幕',
    window: '窗口',
    application: '程序',
  },
  sourceNames: {
    'screen:primary': '内置屏幕',
    'window:browser': '浏览器预览',
    'application:editor': '代码编辑器',
    'screen:native-backend-queued': '原生屏幕源',
    'window:native-backend-queued': '原生窗口源',
    'application:native-backend-queued': '原生程序源',
  },
  sourceMeta: {
    'screen:primary': '3024 x 1964 · 源像素',
    'window:browser': '单个应用窗口',
    'application:editor': '应用程序组',
  },
  sourceUnavailable: '原生采集后端排期中',
  mediaDeviceNames: {
    'system-audio:default': '默认系统声音',
    'microphone:default': '默认麦克风',
    'microphone:studio-usb': 'Studio USB 麦克风',
    'microphone:virtual-broadcast': '虚拟播报麦克风',
    'camera:default': '默认摄像头',
    'camera:facetime-hd': 'FaceTime HD 摄像头',
    'camera:usb-capture': 'USB 采集摄像头',
  },
  settings: {
    title: '设置',
    storage: '存储目录',
    storageDetail: '所有新的录制包都会创建在应用管理的 data/video 目录中。',
    storageHealth: '存储状态',
    dataRoot: '数据根目录',
    dataRootDetail: '修改应用管理的数据根目录；录制仍会写入其下方的 data/video。',
    appData: '应用数据',
    settingsFile: '设置文件',
    language: '语言',
    recordingBackend: '录制后端',
    recordingBackendDetail: '真实采集开始前会在这里显示原生后端 ID；排期中的原生后端仍会被预检拦截。',
    preflight: '开始预检',
    preflightPendingDetail: '下一次开始录制会检查来源、媒体设备、后端能力、存储状态以及 mock/native 状态。',
    recordingPackage: '录制包',
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
  panels: {
    systemAudio: '系统声音',
    systemAudioDevice: '系统声音设备',
    microphone: '麦克风',
    rnnoise: 'RNNoise 降噪',
    microphoneDevice: '麦克风设备',
    cameraSidecar: '摄像头旁路',
    cameraDevice: '摄像头设备',
    pipPreset: '画中画位置',
    pipPresetPreview: (label) => `画中画位置：${label}`,
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
    'camera-sidecar': '摄像头旁路',
    'pip-export': '画中画导出',
    'package-recovery': '录制包恢复',
  },
  capabilityDetails: {
    'source-enumeration': '浏览器预览使用稳定演示来源；桌面运行时会使用原生来源识别。',
    'screen-recording': '真实屏幕采集会由对应平台的原生后端实现。',
    'window-recording': '窗口目标映射会由平台原生后端处理。',
    'application-recording': '程序录制会在原生采集前按应用窗口分组。',
    'system-audio': '系统声音采集依赖平台能力，目前按平台后端逐步落地。',
    microphone: '麦克风采集会在原生设备枚举和音频链路完成后启用。',
    'microphone-enhancement': 'RNNoise 只处理麦克风 PCM，系统声音不会参与降噪。',
    'camera-sidecar': '摄像头会作为独立旁路流录制，后续导出时再合成画中画。',
    'pip-export': '画中画导出会使用屏幕视频和摄像头旁路流进行合成。',
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
    'camera-sidecar': '摄像头旁路',
    'camera-sidecar-device': '摄像头旁路',
    'pip-export': '画中画导出',
    storage: '录制存储',
    'mock-backend': '录制后端',
    'recording-backend': '录制后端',
  },
  preflightCheckDetails: {
    request: '当前录制参数无效，请检查来源、音频或摄像头设置。',
    source: '所选来源当前不可用，请重新选择屏幕、窗口或程序。',
    'source-backend': '来源采集后端尚未完全可用。',
    'system-audio-device': '所选系统声音设备不可用。',
    'system-audio': '系统声音采集仍在平台后端排期中。',
    'microphone-device': '所选麦克风设备不可用。',
    microphone: '麦克风采集仍在平台后端排期中。',
    'microphone-rnnoise-device': '所选麦克风当前不满足 RNNoise 处理条件。',
    'microphone-rnnoise': 'RNNoise 原生处理链路仍在排期中。',
    'microphone-enhancement': '麦克风增强链路仍在排期中。',
    'camera-device': '所选摄像头设备不可用。',
    'camera-sidecar': '摄像头旁路采集仍在平台后端排期中。',
    'camera-sidecar-device': '所选摄像头当前不满足旁路录制条件。',
    'pip-export': '画中画合成导出仍在排期中。',
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
    applying: '正在应用数据根目录...',
    changed: '数据根目录已更新',
    failed: '数据根目录修改失败',
    changedTo: (path) => `新的录制包将写入 ${path}`,
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
  aria: {
    settingsDialog: 'Settings',
    settingsShell: 'RecordingFreedom settings',
    recorderShell: 'RecordingFreedom recorder',
    recorderControls: 'Capsule recorder controls',
    moveRecorder: 'Move recorder',
    audioCameraControls: 'Audio and camera controls',
    toggleSystemAudio: 'Toggle system audio',
    openMicrophoneSettings: 'Open microphone settings',
    toggleNoiseSuppression: 'Toggle microphone noise suppression',
    openCameraSettings: 'Open camera settings',
    startRecording: 'Start recording',
    stopRecording: 'Stop recording',
    pauseRecording: 'Pause recording',
    resumeRecording: 'Resume recording',
    selectLanguage: 'Select language',
    openSettings: 'Open settings',
    menu: (panel) => `${panel} menu`,
    microphoneLevel: 'Microphone level',
  },
  common: {
    close: 'Close',
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
  sourceTypes: {
    screen: 'Screen',
    window: 'Window',
    application: 'Program',
  },
  sourceNames: {
    'screen:primary': 'Built-in Retina',
    'window:browser': 'Browser Preview',
    'application:editor': 'Code Editor',
    'screen:native-backend-queued': 'Native Screen Source',
    'window:native-backend-queued': 'Native Window Source',
    'application:native-backend-queued': 'Native Program Source',
  },
  sourceMeta: {
    'screen:primary': '3024 x 1964 · source pixels',
    'window:browser': 'Single app window',
    'application:editor': 'Application group',
  },
  sourceUnavailable: 'Native capture backend is queued',
  mediaDeviceNames: {
    'system-audio:default': 'Default System Audio',
    'microphone:default': 'Default Microphone',
    'microphone:studio-usb': 'Studio USB Mic',
    'microphone:virtual-broadcast': 'Virtual Broadcast Mic',
    'camera:default': 'Default Camera',
    'camera:facetime-hd': 'FaceTime HD Camera',
    'camera:usb-capture': 'USB Capture Camera',
  },
  settings: {
    title: 'Settings',
    storage: 'Storage',
    storageDetail: 'All new recording packages are created inside this app-managed data/video directory.',
    storageHealth: 'Storage health',
    dataRoot: 'Data root',
    dataRootDetail: 'Change the managed app data root; recordings still go into data/video below it.',
    appData: 'App data',
    settingsFile: 'Settings',
    language: 'Language',
    recordingBackend: 'Recording backend',
    recordingBackendDetail: 'Native backend IDs are visible here before real capture starts; queued native backends are still blocked by preflight.',
    preflight: 'Preflight',
    preflightPendingDetail: 'The next start action will check source, media devices, backend capability, storage, and mock/native status.',
    recordingPackage: 'Recording package',
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
  panels: {
    systemAudio: 'System audio',
    systemAudioDevice: 'System audio device',
    microphone: 'Microphone',
    rnnoise: 'RNNoise',
    microphoneDevice: 'Microphone device',
    cameraSidecar: 'Camera sidecar',
    cameraDevice: 'Camera device',
    pipPreset: 'PIP preset',
    pipPresetPreview: (label) => `PIP preset: ${label}`,
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
    'camera-sidecar': 'Camera Sidecar',
    'pip-export': 'PIP Export',
    'package-recovery': 'Recording Package Recovery',
  },
  capabilityDetails: {
    'source-enumeration': 'Browser preview uses deterministic demo sources; Wails desktop runtime uses native source enumeration.',
    'screen-recording': 'Real screen capture is implemented by the platform native backend.',
    'window-recording': 'Window capture target mapping is handled by the platform backend.',
    'application-recording': 'Program capture groups application windows before native recording starts.',
    'system-audio': 'System audio capture is platform-specific and lands through each native backend.',
    microphone: 'Microphone capture is enabled after native device enumeration and audio plumbing land.',
    'microphone-enhancement': 'RNNoise processes microphone PCM only; system audio is never denoised.',
    'camera-sidecar': 'Camera sidecar capture is separate from the screen video stream.',
    'pip-export': 'PIP composition will use the screen video plus camera sidecar during export.',
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
    'camera-sidecar': 'Camera Sidecar',
    'camera-sidecar-device': 'Camera Sidecar',
    'pip-export': 'PIP Export',
    storage: 'Recording Storage',
    'mock-backend': 'Recording Backend',
    'recording-backend': 'Recording Backend',
  },
  preflightCheckDetails: {
    request: 'The recording request is invalid; check source, audio, or camera settings.',
    source: 'The selected source is not available; choose another screen, window, or program.',
    'source-backend': 'The source capture backend is not fully available yet.',
    'system-audio-device': 'The selected system audio device is not available.',
    'system-audio': 'System audio capture is still queued for the platform backend.',
    'microphone-device': 'The selected microphone device is not available.',
    microphone: 'Microphone capture is still queued for the platform backend.',
    'microphone-rnnoise-device': 'The selected microphone is not marked RNNoise eligible.',
    'microphone-rnnoise': 'RNNoise native DSP is still queued.',
    'microphone-enhancement': 'Microphone enhancement is still queued.',
    'camera-device': 'The selected camera device is not available.',
    'camera-sidecar': 'Camera sidecar capture is still queued for the platform backend.',
    'camera-sidecar-device': 'The selected camera is not sidecar eligible.',
    'pip-export': 'PIP composition export is still queued.',
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
    applying: 'Applying data root...',
    changed: 'Data root changed',
    failed: 'Data root change failed',
    changedTo: (path) => `Recording packages will use ${path}`,
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
