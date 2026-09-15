import {readAnnotationPreviewImage} from '../services/recorderBackend'
import {formatBytes} from './panelShared'
import {formatMilliseconds} from './panelShared'
import {CSSProperties, useEffect, useState} from 'react'
import type {RecorderCopy} from '../i18n'
import type {RecordingExportPlan} from '../services/recorderBackend'

export function ExportPlanTimelinePreview({plan, copy}: {plan: RecordingExportPlan | null; copy: RecorderCopy}) {
  const visibleSegments = plan?.annotationsVisible ? plan.annotationSnapshots?.slice(0, 4) ?? [] : []
  const previewKey = visibleSegments.map((segment) => `${segment.relativePath || segment.inputPath}:${segment.startOffsetMs}`).join('|')
  const [previewImages, setPreviewImages] = useState<Record<string, string>>({})

  useEffect(() => {
    setPreviewImages({})
    if (!plan?.packageDir || visibleSegments.length === 0) return
    let cancelled = false
    void (async () => {
      const entries = await Promise.all(visibleSegments.map(async (segment) => {
        const key = annotationSegmentKey(segment)
        const snapshotPath = segment.relativePath || segment.inputPath
        if (!snapshotPath) return null
        const preview = await readAnnotationPreviewImage(plan.packageDir, snapshotPath)
        if (!preview.available || !preview.dataUrl) return null
        return [key, preview.dataUrl] as const
      }))
      if (cancelled) return
      const nextImages: Record<string, string> = {}
      for (const entry of entries) {
        if (entry) nextImages[entry[0]] = entry[1]
      }
      setPreviewImages(nextImages)
    })()
    return () => {
      cancelled = true
    }
  }, [plan?.packageDir, previewKey])

  if (!plan?.annotationsVisible || visibleSegments.length === 0) return null
  const remaining = (plan.annotationSnapshots?.length ?? visibleSegments.length) - visibleSegments.length
  const summary = plan.annotationSummary
  const eventBytes = formatBytes(summary?.eventFileBytes ?? 0)
  const snapshotBytes = formatBytes(summary?.snapshotBytes ?? 0)
  const skipped = summary?.skippedSnapshotCount ?? 0
  return (
    <div className="export-plan-timeline">
      <div className="export-plan-timeline-header">
        <strong>{copy.settings.exportPlanTimelineTitle}</strong>
        <span>{copy.settings.exportPlanTimelineStats(eventBytes, snapshotBytes, skipped)}</span>
        {(summary?.elementKeyframeCount ?? 0) > 0 && (
          <span>
            {copy.settings.exportPlanElementTimelineStats(
              summary?.elementKeyframeCount ?? 0,
              summary?.finalElementCount ?? 0,
              summary?.missingElementPayloads ?? 0,
            )}
          </span>
        )}
      </div>
      <div className="export-plan-segments" aria-label={copy.settings.exportPlanTimelineTitle}>
        {visibleSegments.map((segment, index) => {
          const start = formatMilliseconds(segment.startOffsetMs)
          const end = segment.endOffsetMs && segment.endOffsetMs > segment.startOffsetMs
            ? formatMilliseconds(segment.endOffsetMs)
            : copy.settings.exportPlanOpenEnded
          const size = formatBytes(segment.bytes ?? 0)
          const label = copy.settings.exportPlanSegmentLabel(index + 1, start, end, size)
          const segmentKey = annotationSegmentKey(segment)
          const previewUrl = previewImages[segmentKey]
          return (
            <div className={`export-plan-segment ${previewUrl ? 'has-preview' : ''}`} key={segmentKey}>
              {previewUrl && <img src={previewUrl} alt="" />}
              <span style={{'--segment-start': `${Math.min(92, Math.max(0, segment.startOffsetMs / 1000))}%`} as CSSProperties} />
              <small>{label}</small>
              {segment.relativePath && <em>{segment.relativePath}</em>}
            </div>
          )
        })}
      </div>
      {remaining > 0 && <p>{copy.settings.exportPlanSegmentMore(remaining)}</p>}
    </div>
  )
}

export function annotationSegmentKey(segment: NonNullable<RecordingExportPlan['annotationSnapshots']>[number]) {
  return `${segment.relativePath || segment.inputPath}:${segment.startOffsetMs}`
}

