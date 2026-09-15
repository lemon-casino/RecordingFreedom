import {translateOcr} from '../services/recorderBackend'
import {writeClipboardText} from '../utils/clipboard'
import {normalizeOcrTranslationSettings, ocrTranslationUnavailableMessage, type AppSettings} from '../services/mockBackend'
import type {OcrResult, OcrTranslationResult} from '../services/recorderBackend'
import type {RecorderCopy} from '../i18n'
import type {ScreenshotPinState, ScreenshotPinnedItem} from '../services/recorderBackend'
import type {ScreenshotItem} from '../services/mockBackend'

export function screenshotPinItems(state: ScreenshotPinState | undefined): ScreenshotPinnedItem[] {
  const sourcePins = state?.pins?.length
    ? state.pins
    : state?.item
      ? [{item: state.item, dataUrl: state.dataUrl, fixed: state.fixed}]
      : []
  const seen = new Set<string>()
  const pins: ScreenshotPinnedItem[] = []
  for (const pin of sourcePins) {
    if (!pin.item?.id || seen.has(pin.item.id)) continue
    seen.add(pin.item.id)
    pins.push({
      item: pin.item,
      dataUrl: pin.dataUrl,
      fixed: pin.fixed === true || pin.item.fixed === true })
  }
  return pins
}

export function screenshotPinStateWithPins(current: ScreenshotPinState | undefined, pins: ScreenshotPinnedItem[]): ScreenshotPinState {
  const normalized = screenshotPinItems({visible: pins.length > 0, fixed: false, pins})
  if (normalized.length === 0) return {visible: false, fixed: false, pins: []}
  const active = normalized[normalized.length - 1]
  return {
    ...(current ?? {}),
    visible: true,
    item: active.item,
    dataUrl: active.dataUrl,
    fixed: active.fixed,
    pins: normalized }
}

export function screenshotOcrBusy(item: ScreenshotItem) {
  return item.ocrStatus === 'queued' || item.ocrStatus === 'running'
}

export function screenshotOcrStatusText(item: ScreenshotItem, copy: RecorderCopy) {
  switch (item.ocrStatus) {
    case 'queued':
      return copy.screenshot.ocrStatusQueued
    case 'running':
      return copy.screenshot.ocrStatusRunning
    case 'ready':
      return copy.screenshot.ocrStatusReady
    case 'failed':
      return item.ocrError || copy.screenshot.ocrStatusFailed
    default:
      return copy.screenshot.ocrStatusNone
  }
}

export function screenshotOcrPrimaryTitle(item: ScreenshotItem, copy: RecorderCopy) {
  if (item.ocrStatus === 'ready') return copy.screenshot.openOcrResult
  if (item.ocrStatus === 'failed') return copy.screenshot.retryOcr
  if (screenshotOcrBusy(item)) return screenshotOcrStatusText(item, copy)
  return copy.screenshot.recognizeText
}

export async function copyOcrResultText(result: OcrResult, copy: RecorderCopy) {
  const text = result.plainText.trim()
  if (!text) return copy.screenshot.copyTextEmpty
  await writeClipboardText(text)
  return copy.screenshot.copiedText
}

export async function translateAndCopyOcrResultText(resultId: string, translation: AppSettings['ocr']['translation'], copy: RecorderCopy) {
  const normalized = normalizeOcrTranslationSettings(translation)
  const unavailable = ocrTranslationUnavailableMessage(normalized, copy)
  if (unavailable) return unavailable
  const result = await translateOcr({
    ocrResultId: resultId,
    provider: normalized.provider,
    sourceLanguage: normalized.sourceLanguage,
    targetLanguage: normalized.targetLanguage,
    baseUrl: normalized.baseUrl,
    apiKey: normalized.apiKey,
    model: normalized.model,
    force: false })
  const text = ocrTranslationPlainText(result)
  if (!text) return copy.screenshot.copyTextEmpty
  await writeClipboardText(text)
  return copy.screenshot.copiedTranslation
}

export function screenshotDisplayName(item: ScreenshotItem) {
  const date = new Date(item.createdAt)
  if (Number.isNaN(date.getTime())) return item.id
  return `${date.toLocaleDateString()} ${date.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'})}`
}

export function ocrTranslationPlainText(result: OcrTranslationResult | null) {
  return result?.blocks.map((block) => block.translated.trim()).filter(Boolean).join('\n').trim() ?? ''
}
