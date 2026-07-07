import {showFloatingPanel, type CapsuleWindowDockSide} from '../../services/recorderBackend'
import {resolveFloatingPanelPlacement} from './floatingPosition'

export const ocrResultPanelSize = {width: 380, height: 420, maxHeight: 420, minWidth: 340}
export const ocrResultPanelExpandedSize = {width: 560, height: 620, maxHeight: 620, minWidth: 500}

const ocrPanelAutoTranslatePrefix = 'translate:'

export function ocrPanelContext(resultId: string, autoTranslate = false) {
  const clean = resultId.trim()
  return autoTranslate ? `${ocrPanelAutoTranslatePrefix}${clean}` : clean
}

export function parseOcrPanelContext(contextId: string | undefined) {
  const clean = (contextId ?? '').trim()
  if (clean.startsWith(ocrPanelAutoTranslatePrefix)) {
    return {
      resultId: clean.slice(ocrPanelAutoTranslatePrefix.length).trim(),
      autoTranslate: true,
    }
  }
  return {resultId: clean, autoTranslate: false}
}

export async function showOcrResultFloatingPanel(anchorElement: Element, options: {
  resultId: string
  token: number
  dockSide?: CapsuleWindowDockSide
  autoTranslate?: boolean
}) {
  const resultId = options.resultId.trim()
  if (!resultId) throw new Error('OCR result id is required')
  const placement = await resolveFloatingPanelPlacement(anchorElement, {
    dockSide: options.dockSide ?? 'none',
    width: ocrResultPanelSize.width,
    height: ocrResultPanelSize.height,
    maxHeight: ocrResultPanelSize.maxHeight,
    minWidth: ocrResultPanelSize.minWidth,
  })
  await showFloatingPanel({
    kind: 'ocr-result',
    anchor: placement.anchor,
    bounds: placement.bounds,
    dockSide: options.dockSide ?? 'none',
    width: placement.bounds.width,
    height: placement.bounds.height,
    minWidth: ocrResultPanelSize.minWidth,
    maxHeight: ocrResultPanelSize.maxHeight,
    token: options.token,
    screenId: placement.screenId,
    direction: placement.direction,
    contextId: ocrPanelContext(resultId, options.autoTranslate === true),
  })
  return placement
}
