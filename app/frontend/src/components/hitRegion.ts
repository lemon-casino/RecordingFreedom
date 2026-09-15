import type {CapsuleWindowHitRegion} from '../services/recorderBackend'

export function elementHitRegion(
  element: Element | null,
  viewportWidth: number,
  viewportHeight: number,
  kind: CapsuleWindowHitRegion['kind'] = 'round-rect',
  radius = 18,
): CapsuleWindowHitRegion | null {
  if (!element) return null
  const rect = element.getBoundingClientRect()
  if (rect.width <= 0 || rect.height <= 0) return null
  const x = Math.max(0, Math.floor(rect.left))
  const y = Math.max(0, Math.floor(rect.top))
  const right = Math.min(viewportWidth, Math.ceil(rect.right))
  const bottom = Math.min(viewportHeight, Math.ceil(rect.bottom))
  if (right <= x || bottom <= y) return null
  return {x, y, width: right - x, height: bottom - y, kind, radius}
}
