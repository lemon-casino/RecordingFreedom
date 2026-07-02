export type CaptureSourceType = 'screen' | 'all-screens' | 'region' | 'window' | 'application'
export type RecordingMode = 'video' | 'audio'
export type RecordingState = 'idle' | 'preparing' | 'recording' | 'paused' | 'stopping' | 'ready' | 'failed'
export type LocaleCode = 'zh-CN' | 'en'
export type CaptureCapabilityStatus = 'available' | 'queued' | 'blocked' | 'unsupported'
export type CaptureCapabilityPermission = 'not-required' | 'unknown' | 'screen-recording' | 'microphone' | 'camera'
export type PIPPreset = 'off' | 'bottom-right' | 'bottom-left' | 'free'
export type PIPShape = 'circle' | 'square'
export type RecordingQuality = 'standard' | 'balanced' | 'high'
export type RecordingPreflightStatus = 'ready' | 'warning' | 'blocked'

export type PIPConfig = {
  preset: PIPPreset
  shape: PIPShape
  mirror: boolean
  position: {
    x: number
    y: number
  }
  scale: number
  edgeFeather: number
}

export type RecordingProfile = {
  quality: RecordingQuality
  fps: number
  captureCursor: boolean
  countdownSeconds: number
}

export type AppDataInfo = {
  rootDir: string
  videoDir: string
}

export type AppStorageStatus = {
  rootDir: string
  videoDir: string
  writable: boolean
  freeSpaceKnown: boolean
  availableBytes: number
  minimumRecommendedBytes: number
  status: RecordingPreflightStatus
  reason?: string
}

export type CaptureSource = {
  id: string
  type: CaptureSourceType
  label: string
  name: string
  meta: string
  x?: number
  y?: number
  available?: boolean
  capability?: string
  unavailableReason?: string
  width?: number
  height?: number
  displayIndex?: number
  nativeId?: string
  processId?: number
}

export type MediaDeviceType = 'system-audio' | 'microphone' | 'camera'

export type MediaDevice = {
  id: string
  type: MediaDeviceType
  name: string
  meta: string
  nativeId?: string
  isDefault?: boolean
  available?: boolean
  capability?: string
  unavailableReason?: string
  rnnoiseEligible?: boolean
  sidecarEligible?: boolean
}

export type MediaInventory = {
  systemAudio: MediaDevice[]
  microphones: MediaDevice[]
  cameras: MediaDevice[]
  enhancement: {
    engine: string
    appliesTo: string
    available: boolean
    capability: string
    unavailableReason?: string
  }
}

export type CaptureCapability = {
  id: string
  label: string
  status: CaptureCapabilityStatus
  backend: string
  permission: CaptureCapabilityPermission
  reason?: string
}

export type CaptureCapabilities = {
  platform: string
  sourceEnumeration: CaptureCapability
  screenRecording: CaptureCapability
  windowRecording: CaptureCapability
  applicationRecording: CaptureCapability
  systemAudio: CaptureCapability
  microphone: CaptureCapability
  microphoneEnhancement: CaptureCapability
  cameraSidecar: CaptureCapability
  pipExport: CaptureCapability
  packageRecovery: CaptureCapability
}

export type AppSettings = {
  schemaVersion: number
  locale: LocaleCode
  source: {
    lastSourceId?: string
    lastSourceType: CaptureSourceType
  }
  storage: {
    dataRootDir?: string
  }
  recording: RecordingProfile
  audio: {
    system: boolean
    systemDeviceId?: string
    microphone: boolean
    microphoneDeviceId?: string
    noiseSuppression: boolean
    microphoneGain: number
  }
  camera: {
    enabled: boolean
    deviceId?: string
    pipPreset: PIPPreset
    pip: PIPConfig
  }
  window: {
    minimizeToTray: boolean
  }
  updatedAt?: string
}

export type MockRecordingRequest = {
  source: CaptureSource
  systemAudio: boolean
  systemAudioDeviceId?: string
  recording: RecordingProfile
  microphone: boolean
  microphoneDeviceId?: string
  noiseSuppression: boolean
  camera: boolean
  cameraDeviceId?: string
  cameraDeviceNativeId?: string
  pipPreset: PIPPreset
  pip: PIPConfig
}

export type AudioOnlyRecordingRequest = {
  recording: RecordingProfile
  systemAudio: boolean
  systemAudioDeviceId?: string
  microphone: boolean
  microphoneDeviceId?: string
  noiseSuppression: boolean
}

export type RecordingPreflightCheck = {
  id: string
  label: string
  status: RecordingPreflightStatus
  reason?: string
}

export type RecordingPreflight = {
  status: RecordingPreflightStatus
  backend: string
  message: string
  checks: RecordingPreflightCheck[]
}

export const sources: CaptureSource[] = [
  {
    id: 'all-screens:virtual-desktop',
    type: 'all-screens',
    label: 'All Screens',
    name: 'All Screens',
    meta: 'Queued · multi-display composition',
    x: 0,
    y: 0,
    width: 3024,
    height: 1964,
    available: false,
    capability: 'native-backend-queued',
    unavailableReason: 'Multi-display composition is queued behind the native video writer.',
  },
  {
    id: 'screen:primary',
    type: 'screen',
    label: 'Screen',
    name: 'Built-in Retina',
    meta: '3024 x 1964 · source pixels',
    x: 0,
    y: 0,
    width: 3024,
    height: 1964,
    displayIndex: 1,
    nativeId: 'display:primary',
    available: true,
    capability: 'enumerated',
  },
  {
    id: 'region:custom',
    type: 'region',
    label: 'Region',
    name: 'Custom Region',
    meta: 'Queued · native crop writer',
    x: 0,
    y: 0,
    width: 3024,
    height: 1964,
    available: false,
    capability: 'native-backend-queued',
    unavailableReason: 'Region selector overlay is available; native region crop writer is queued.',
  },
  {
    id: 'window:browser',
    type: 'window',
    label: 'Window',
    name: 'Browser Preview',
    meta: 'Single app window',
    available: true,
    capability: 'enumerated',
  },
  {
    id: 'application:editor',
    type: 'application',
    label: 'Program',
    name: 'Code Editor',
    meta: 'Application group',
    available: true,
    capability: 'enumerated',
  },
]

export const mediaInventory: MediaInventory = {
  systemAudio: [
    {
      id: 'system-audio:default',
      type: 'system-audio',
      name: 'Default System Audio',
      meta: 'System sound capture endpoint',
      isDefault: true,
      available: true,
      capability: 'enumerated',
    },
  ],
  microphones: [
    {
      id: 'microphone:browser-preview',
      type: 'microphone',
      name: 'Browser Preview Microphone',
      meta: 'Desktop runtime required',
      isDefault: true,
      available: false,
      capability: 'native-backend-queued',
      unavailableReason: 'Real microphone devices are listed by the Wails desktop backend.',
      rnnoiseEligible: true,
    },
  ],
  cameras: [
    {
      id: 'camera:default',
      type: 'camera',
      name: 'Default Camera',
      meta: 'Camera endpoint',
      nativeId: 'browser-default-camera',
      isDefault: true,
      available: true,
      capability: 'enumerated',
      sidecarEligible: true,
    },
    {
      id: 'camera:facetime-hd',
      type: 'camera',
      name: 'FaceTime HD Camera',
      meta: 'Built-in camera endpoint',
      nativeId: 'browser-facetime-hd',
      available: true,
      capability: 'enumerated',
      sidecarEligible: true,
    },
    {
      id: 'camera:usb-capture',
      type: 'camera',
      name: 'USB Capture Camera',
      meta: 'External capture endpoint',
      nativeId: 'browser-usb-capture',
      available: true,
      capability: 'enumerated',
      sidecarEligible: true,
    },
  ],
  enhancement: {
    engine: 'rnnoise',
    appliesTo: 'microphone-only',
    available: true,
    capability: 'enumerated',
  },
}

export const fallbackCapabilities: CaptureCapabilities = {
  platform: 'browser-preview',
  sourceEnumeration: {
    id: 'source-enumeration',
    label: 'Source Enumeration',
    status: 'available',
    backend: 'browser mock',
    permission: 'not-required',
    reason: 'Browser preview uses deterministic demo sources; Wails desktop runtime uses native source enumeration.',
  },
  screenRecording: {
    id: 'screen-recording',
    label: 'Screen Recording',
    status: 'queued',
    backend: 'native capture backend',
    permission: 'screen-recording',
    reason: 'Real capture is implemented in the Go native backend, not in browser preview.',
  },
  windowRecording: {
    id: 'window-recording',
    label: 'Window Recording',
    status: 'queued',
    backend: 'native capture backend',
    permission: 'screen-recording',
    reason: 'Window capture target mapping is handled by the platform backend.',
  },
  applicationRecording: {
    id: 'application-recording',
    label: 'Program Recording',
    status: 'queued',
    backend: 'native capture backend',
    permission: 'screen-recording',
    reason: 'Program capture groups application windows before native recording starts.',
  },
  systemAudio: {
    id: 'system-audio',
    label: 'System Audio',
    status: 'queued',
    backend: 'native audio backend',
    permission: 'unknown',
    reason: 'System audio capture is platform-specific and is not available in browser preview.',
  },
  microphone: {
    id: 'microphone',
    label: 'Microphone',
    status: 'queued',
    backend: 'native audio backend',
    permission: 'microphone',
    reason: 'Microphone capture is implemented after native device enumeration lands.',
  },
  microphoneEnhancement: {
    id: 'microphone-enhancement',
    label: 'Microphone RNNoise',
    status: 'queued',
    backend: 'RNNoise native DSP',
    permission: 'not-required',
    reason: 'RNNoise processes microphone PCM only; system audio is never denoised.',
  },
  cameraSidecar: {
    id: 'camera-sidecar',
    label: 'Camera',
    status: 'queued',
    backend: 'native camera backend',
    permission: 'camera',
    reason: 'Camera media is composed into the screen recording as picture-in-picture.',
  },
  pipExport: {
    id: 'pip-export',
    label: 'PIP Export',
    status: 'queued',
    backend: 'export compositor',
    permission: 'not-required',
    reason: 'PIP composition will use the screen video plus camera sidecar during export.',
  },
  packageRecovery: {
    id: 'package-recovery',
    label: 'Recording Package Recovery',
    status: 'available',
    backend: 'browser mock',
    permission: 'not-required',
    reason: 'Desktop runtime scans .rfrec packages under app-managed data/video.',
  },
}

export const fallbackAppData: AppDataInfo = {
  rootDir: 'browser-preview',
  videoDir: 'data/video',
}

export const fallbackStorageStatus: AppStorageStatus = {
  rootDir: fallbackAppData.rootDir,
  videoDir: fallbackAppData.videoDir,
  writable: true,
  freeSpaceKnown: false,
  availableBytes: 0,
  minimumRecommendedBytes: 1024 * 1024 * 1024,
  status: 'ready',
  reason: 'Browser preview does not inspect desktop free space.',
}

export const localeOptions: LocaleCode[] = ['zh-CN', 'en']

export function normalizeLocale(value: unknown): LocaleCode {
  return value === 'en' || value === 'zh-CN' ? value : 'zh-CN'
}

export const microphoneDevices = mediaInventory.microphones
export const systemAudioDevices = mediaInventory.systemAudio
export const cameraDevices = mediaInventory.cameras

export const defaultSettings: AppSettings = {
  schemaVersion: 1,
  locale: 'zh-CN',
  source: {
    lastSourceType: 'screen',
  },
  storage: {
    dataRootDir: 'browser-preview',
  },
  recording: {
    quality: 'balanced',
    fps: 30,
    captureCursor: true,
    countdownSeconds: 0,
  },
  audio: {
    system: false,
    systemDeviceId: 'system-audio:default',
    microphone: false,
    microphoneDeviceId: 'microphone:browser-preview',
    noiseSuppression: false,
    microphoneGain: 1,
  },
  camera: {
    enabled: false,
    deviceId: 'camera:default',
    pipPreset: 'bottom-right',
    pip: {
      preset: 'bottom-right',
      shape: 'circle',
      mirror: true,
      position: {x: 1, y: 1},
      scale: 0.08,
      edgeFeather: 0.16,
    },
  },
  window: {
    minimizeToTray: true,
  },
}

export function createMockRecordingPackage(request: MockRecordingRequest) {
  const stamp = new Date()
    .toISOString()
    .replace(/:/g, '-')
    .replace('T', '-')
    .replace(/\..+$/, '')

  return {
    id: `recording-${stamp}`,
    packagePath: `data/video/recording-${stamp}.rfrec`,
    manifest: {
      schemaVersion: 1,
      app: 'RecordingFreedom',
      status: 'recording',
      media: {screenVideoPath: 'screen.mock.txt'},
      source: {
        type: request.source.type,
        id: request.source.id,
        name: request.source.name,
        geometry: sourceGeometry(request.source),
      },
      recording: request.recording,
      audio: {
        system: request.systemAudio,
        systemDeviceId: request.systemAudio ? request.systemAudioDeviceId : undefined,
        microphone: request.microphone,
        microphoneDeviceId: request.microphone ? request.microphoneDeviceId : undefined,
        microphoneNoiseSuppression: request.microphone && request.noiseSuppression ? 'rnnoise' : 'off',
      },
      camera: {
        enabled: request.camera,
        deviceId: request.cameraDeviceId,
        pipPreset: request.camera ? request.pipPreset : 'off',
        pip: request.camera ? request.pip : {...defaultSettings.camera.pip, preset: 'off'},
      },
      diagnostics: {
        mock: true,
      },
    },
  }
}

function sourceGeometry(source: CaptureSource) {
  if (!source.width || !source.height) return undefined
  return {
    x: source.x ?? 0,
    y: source.y ?? 0,
    width: source.width,
    height: source.height,
    displayIndex: source.displayIndex,
    nativeId: source.nativeId,
  }
}

export function createMockAudioOnlyRecordingPackage(request: AudioOnlyRecordingRequest) {
  const stamp = new Date()
    .toISOString()
    .replace(/:/g, '-')
    .replace('T', '-')
    .replace(/\..+$/, '')

  return {
    id: `recording-${stamp}`,
    packagePath: `data/video/recording-${stamp}.rfrec`,
    manifest: {
      schemaVersion: 1,
      app: 'RecordingFreedom',
      status: 'recording',
      recordingMode: 'audio-only',
      media: {audioPath: 'audio.mock.wav'},
      recording: request.recording,
      audio: {
        system: request.systemAudio,
        systemDeviceId: request.systemAudio ? request.systemAudioDeviceId : undefined,
        microphone: request.microphone,
        microphoneDeviceId: request.microphone ? request.microphoneDeviceId : undefined,
        microphoneNoiseSuppression: request.microphone && request.noiseSuppression ? 'rnnoise' : 'off',
      },
      diagnostics: {
        mock: true,
      },
    },
  }
}
