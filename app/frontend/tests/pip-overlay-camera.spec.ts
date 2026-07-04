import {expect, test, type Page} from '@playwright/test'

const browserSettingsKey = 'recordingfreedom.settings.v1'

test('pip overlay close stops camera stream and ignores stale drag updates', async ({page}) => {
  await openPipOverlayWithMockCamera(page)

  await expect(page.locator('.pip-live-frame')).toBeVisible()
  await expect.poll(() => pipCameraCounters(page)).toMatchObject({opened: 1})

  await page.getByRole('button', {name: 'Close PIP'}).click()

  await expect(page.locator('.pip-live-frame')).toHaveCount(0)
  await expect.poll(() => pipCameraCounters(page)).toMatchObject({opened: 1, stopped: 1})

  await page.evaluate(() => {
    const stale = (window as Window & {__RF_STALE_PIP_STATE__?: unknown}).__RF_STALE_PIP_STATE__
    window.dispatchEvent(new CustomEvent('rf-pip-overlay', {detail: stale}))
  })

  await page.waitForTimeout(250)
  await expect(page.locator('.pip-live-frame')).toHaveCount(0)
  await expect.poll(() => pipCameraCounters(page)).toMatchObject({opened: 1, stopped: 1})
})

test('recording pip overlay waits for backend preview image without reopening browser camera', async ({page}) => {
  await openRecordingPipOverlayWithBackendPreviewPath(page)

  await expect(page.locator('.pip-live-frame')).toBeVisible()
  await expect(page.getByText('Camera recording')).toBeVisible()
  await expect(page.getByText('HD Webcam')).toBeVisible()
  await expect.poll(() => pipCameraCounters(page)).toMatchObject({opened: 0})
})

async function openPipOverlayWithMockCamera(page: Page) {
  await page.addInitScript(({settingsKey}) => {
    const initialPipState = {
      config: {
        preset: 'bottom-right',
        shape: 'circle',
        mirror: true,
        position: {x: 1, y: 1},
        scale: 0.08,
        edgeFeather: 0.16,
      },
      placement: {
        visible: true,
        rect: {x: 1024, y: 512, width: 96, height: 96, visible: true},
        shape: 'circle',
        mirror: true,
        edgeFeather: 0.16,
      },
      overlayBounds: {x: 0, y: 0, width: 1280, height: 720},
      windowBounds: {x: 1000, y: 488, width: 144, height: 144},
      contentBounds: {x: 24, y: 24, width: 96, height: 96},
      mode: 'edit',
      cameraName: 'HD Webcam',
      camera: {deviceId: 'camera:dshow:hd-webcam', nativeId: 'HD Webcam', name: 'HD Webcam'},
      previewImagePath: '',
      captureExcluded: false,
      clientOperationId: 1,
    }

    window.localStorage.setItem(settingsKey, JSON.stringify({
      schemaVersion: 1,
      locale: 'en',
      source: {lastSourceType: 'screen'},
      storage: {dataRootDir: 'browser-preview'},
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
        enabled: true,
        deviceId: 'camera:dshow:hd-webcam',
        pipPreset: 'bottom-right',
        pip: initialPipState.config,
      },
      whiteboard: {
        enabled: true,
        lastMode: 'board',
        lastTool: 'freedraw',
        lastStrokeColor: '#ef4444',
        lastStrokeWidth: 'medium',
        lastOpacity: 100,
        capturePolicy: 'export-compose',
      },
      window: {
        minimizeToTray: true,
        theme: 'night-teal',
      },
    }))

    const globals = window as Window & {
      __RF_PIP_OVERLAY__?: unknown
      __RF_STALE_PIP_STATE__?: unknown
      __RF_GET_USER_MEDIA_CALLS__?: number
      __RF_STOPPED_TRACKS__?: number
    }
    globals.__RF_PIP_OVERLAY__ = initialPipState
    globals.__RF_STALE_PIP_STATE__ = initialPipState
    globals.__RF_GET_USER_MEDIA_CALLS__ = 0
    globals.__RF_STOPPED_TRACKS__ = 0

    const mockMediaDevices = {
      enumerateDevices: async () => [{
        kind: 'videoinput',
        deviceId: 'browser-hd-webcam',
        label: 'HD Webcam',
        groupId: 'browser-hd-webcam-group',
        toJSON: () => ({}),
      }],
      getUserMedia: async () => {
        globals.__RF_GET_USER_MEDIA_CALLS__ = (globals.__RF_GET_USER_MEDIA_CALLS__ ?? 0) + 1
        const canvas = document.createElement('canvas')
        canvas.width = 160
        canvas.height = 90
        const context = canvas.getContext('2d')
        if (context) {
          context.fillStyle = '#14b8a6'
          context.fillRect(0, 0, canvas.width, canvas.height)
        }
        const stream = canvas.captureStream(15)
        stream.getTracks().forEach((track) => {
          const stop = track.stop.bind(track)
          track.stop = () => {
            if (track.readyState !== 'ended') {
              globals.__RF_STOPPED_TRACKS__ = (globals.__RF_STOPPED_TRACKS__ ?? 0) + 1
            }
            stop()
          }
        })
        return stream
      },
    }
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: mockMediaDevices,
    })
  }, {settingsKey: browserSettingsKey})

  await page.goto('/#/pip-overlay')
}

async function openRecordingPipOverlayWithBackendPreviewPath(page: Page) {
  await page.addInitScript(({settingsKey}) => {
    const recordingPipState = {
      config: {
        preset: 'bottom-right',
        shape: 'circle',
        mirror: true,
        position: {x: 1, y: 1},
        scale: 0.08,
        edgeFeather: 0.16,
      },
      placement: {
        visible: true,
        rect: {x: 1024, y: 512, width: 96, height: 96, visible: true},
        shape: 'circle',
        mirror: true,
        edgeFeather: 0.16,
      },
      overlayBounds: {x: 0, y: 0, width: 1280, height: 720},
      windowBounds: {x: 1000, y: 488, width: 144, height: 144},
      contentBounds: {x: 24, y: 24, width: 96, height: 96},
      mode: 'recording',
      cameraName: 'HD Webcam',
      camera: {deviceId: 'camera:dshow:hd-webcam', nativeId: 'HD Webcam', name: 'HD Webcam'},
      previewImagePath: 'browser-preview/data/video/recording.rfrec/cache/pip-camera-preview.jpg',
      captureExcluded: false,
      clientOperationId: 1,
    }

    window.localStorage.setItem(settingsKey, JSON.stringify({
      schemaVersion: 1,
      locale: 'en',
      source: {lastSourceType: 'screen'},
      storage: {dataRootDir: 'browser-preview'},
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
        enabled: true,
        deviceId: 'camera:dshow:hd-webcam',
        pipPreset: 'bottom-right',
        pip: recordingPipState.config,
      },
      whiteboard: {
        enabled: true,
        lastMode: 'board',
        lastTool: 'freedraw',
        lastStrokeColor: '#ef4444',
        lastStrokeWidth: 'medium',
        lastOpacity: 100,
        capturePolicy: 'export-compose',
      },
      window: {
        minimizeToTray: true,
        theme: 'night-teal',
      },
    }))

    const globals = window as Window & {
      __RF_PIP_OVERLAY__?: unknown
      __RF_GET_USER_MEDIA_CALLS__?: number
      __RF_STOPPED_TRACKS__?: number
    }
    globals.__RF_PIP_OVERLAY__ = recordingPipState
    globals.__RF_GET_USER_MEDIA_CALLS__ = 0
    globals.__RF_STOPPED_TRACKS__ = 0
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        enumerateDevices: async () => [],
        getUserMedia: async () => {
          globals.__RF_GET_USER_MEDIA_CALLS__ = (globals.__RF_GET_USER_MEDIA_CALLS__ ?? 0) + 1
          throw new Error('recording mode must use backend preview image')
        },
      },
    })
  }, {settingsKey: browserSettingsKey})

  await page.goto('/#/pip-overlay')
}

async function pipCameraCounters(page: Page) {
  return page.evaluate(() => {
    const globals = window as Window & {
      __RF_GET_USER_MEDIA_CALLS__?: number
      __RF_STOPPED_TRACKS__?: number
    }
    return {
      opened: globals.__RF_GET_USER_MEDIA_CALLS__ ?? 0,
      stopped: globals.__RF_STOPPED_TRACKS__ ?? 0,
    }
  })
}
