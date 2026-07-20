import type {RecorderCopy} from '../i18n'
import type {WhiteboardTool} from '../services/mockBackend'

export type AnnotationStyle = {
  strokeStyle: 'solid' | 'dashed' | 'dotted'
  fillStyle: 'solid' | 'hachure' | 'cross-hatch'
  fillColor: string
  roundness: 'sharp' | 'round'
  roughness: 0 | 1 | 2
  arrowType: 'sharp' | 'round' | 'elbow'
  arrowhead: 'none' | 'arrow' | 'triangle' | 'circle' | 'bar' | 'diamond'
  fontFamily: 1 | 2 | 3 | 5 | 6 | 7 | 8 | 9
  fontSize: number
  textAlign: 'left' | 'center' | 'right'
}

type AnnotationStyleControlsProps = {
  copy: RecorderCopy
  activeTool: WhiteboardTool
  style: AnnotationStyle
  onChange: <K extends keyof AnnotationStyle>(key: K, value: AnnotationStyle[K]) => void
}

const styleTools: WhiteboardTool[] = ['freedraw', 'arrow', 'line', 'rectangle', 'ellipse', 'text']
const fillColors = ['transparent', '#ef4444', '#f59e0b', '#22c55e', '#38bdf8', '#a78bfa', '#f8fafc', '#111827']

export function AnnotationStyleControls({copy, activeTool, style, onChange}: AnnotationStyleControlsProps) {
  if (!styleTools.includes(activeTool)) return null

  const showShapeOptions = activeTool === 'rectangle' || activeTool === 'ellipse'
  const showArrowOptions = activeTool === 'arrow'
  const showTextOptions = activeTool === 'text'
  const showRoughness = activeTool !== 'text'

  return (
    <div className="annotation-style-controls" aria-label={copy.whiteboard.styleSettings}>
      {!showTextOptions && (
        <label className="annotation-style-select">
          <span>{copy.whiteboard.strokeStyle}</span>
          <select value={style.strokeStyle} onChange={(event) => onChange('strokeStyle', event.currentTarget.value as AnnotationStyle['strokeStyle'])}>
            <option value="solid">{copy.whiteboard.solid}</option>
            <option value="dashed">{copy.whiteboard.dashed}</option>
            <option value="dotted">{copy.whiteboard.dotted}</option>
          </select>
        </label>
      )}
      {showRoughness && (
        <label className="annotation-style-select">
          <span>{copy.whiteboard.roughness}</span>
          <select value={style.roughness} onChange={(event) => onChange('roughness', Number(event.currentTarget.value) as AnnotationStyle['roughness'])}>
            <option value={0}>{copy.whiteboard.architect}</option>
            <option value={1}>{copy.whiteboard.artist}</option>
            <option value={2}>{copy.whiteboard.cartoonist}</option>
          </select>
        </label>
      )}
      {showShapeOptions && (
        <>
          <div className="annotation-style-group annotation-fill-group" aria-label={copy.whiteboard.fillColor}>
            <span className="annotation-style-group-label">{copy.whiteboard.fillColor}</span>
            {fillColors.map((color) => (
              <button
                key={color}
                className={style.fillColor.toLowerCase() === color ? 'selected' : ''}
                type="button"
                aria-label={`${copy.whiteboard.fillColor} ${color}`}
                title={color === 'transparent' ? copy.whiteboard.transparent : color}
                style={{'--swatch': color} as any}
                onClick={() => onChange('fillColor', color)}
              />
            ))}
          </div>
          <label className="annotation-style-select">
            <span>{copy.whiteboard.fillStyle}</span>
            <select value={style.fillStyle} onChange={(event) => onChange('fillStyle', event.currentTarget.value as AnnotationStyle['fillStyle'])}>
              <option value="solid">{copy.whiteboard.solid}</option>
              <option value="hachure">{copy.whiteboard.hachure}</option>
              <option value="cross-hatch">{copy.whiteboard.crossHatch}</option>
            </select>
          </label>
          <label className="annotation-style-select">
            <span>{copy.whiteboard.roundness}</span>
            <select value={style.roundness} onChange={(event) => onChange('roundness', event.currentTarget.value as AnnotationStyle['roundness'])}>
              <option value="sharp">{copy.whiteboard.sharp}</option>
              <option value="round">{copy.whiteboard.round}</option>
            </select>
          </label>
        </>
      )}
      {showTextOptions && (
        <>
          <label className="annotation-style-select">
            <span>{copy.whiteboard.fontFamily}</span>
            <select value={style.fontFamily} onChange={(event) => onChange('fontFamily', Number(event.currentTarget.value) as AnnotationStyle['fontFamily'])}>
              <option value={1}>{copy.whiteboard.fontFamilyVirgil}</option>
              <option value={2}>{copy.whiteboard.fontFamilyHelvetica}</option>
              <option value={3}>{copy.whiteboard.fontFamilyCascadia}</option>
              <option value={5}>{copy.whiteboard.fontFamilyExcalifont}</option>
              <option value={6}>{copy.whiteboard.fontFamilyNunito}</option>
              <option value={9}>{copy.whiteboard.fontFamilyLiberation}</option>
            </select>
          </label>
          <label className="annotation-style-select">
            <span>{copy.whiteboard.fontSize}</span>
            <select value={style.fontSize} onChange={(event) => onChange('fontSize', Number(event.currentTarget.value))}>
              {[12, 16, 20, 24, 32, 40, 48].map((size) => <option key={size} value={size}>{size}px</option>)}
            </select>
          </label>
          <label className="annotation-style-select">
            <span>{copy.whiteboard.textAlign}</span>
            <select value={style.textAlign} onChange={(event) => onChange('textAlign', event.currentTarget.value as AnnotationStyle['textAlign'])}>
              <option value="left">{copy.whiteboard.left}</option>
              <option value="center">{copy.whiteboard.center}</option>
              <option value="right">{copy.whiteboard.right}</option>
            </select>
          </label>
        </>
      )}
      {showArrowOptions && (
        <>
          <label className="annotation-style-select">
            <span>{copy.whiteboard.arrowType}</span>
            <select value={style.arrowType} onChange={(event) => onChange('arrowType', event.currentTarget.value as AnnotationStyle['arrowType'])}>
              <option value="sharp">{copy.whiteboard.sharp}</option>
              <option value="round">{copy.whiteboard.round}</option>
              <option value="elbow">{copy.whiteboard.elbow}</option>
            </select>
          </label>
          <label className="annotation-style-select">
            <span>{copy.whiteboard.arrowhead}</span>
            <select value={style.arrowhead} onChange={(event) => onChange('arrowhead', event.currentTarget.value as AnnotationStyle['arrowhead'])}>
              <option value="none">{copy.whiteboard.arrowheadNone}</option>
              <option value="arrow">{copy.whiteboard.arrowheadArrow}</option>
              <option value="triangle">{copy.whiteboard.arrowheadTriangle}</option>
              <option value="circle">{copy.whiteboard.arrowheadCircle}</option>
              <option value="bar">{copy.whiteboard.arrowheadBar}</option>
              <option value="diamond">{copy.whiteboard.arrowheadDiamond}</option>
            </select>
          </label>
        </>
      )}
    </div>
  )
}
