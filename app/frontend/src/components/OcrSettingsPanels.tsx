import {cancelOcrModelDownload, getOcrModelDownloads, getOcrStatus, installOcrModelPackage, refreshOcrModelCatalog, removeOcrModel, setActiveOcrModel, startOcrModelDownload} from '../services/recorderBackend'
import {formatBytes} from './panelShared'
import {AppSettings, normalizeOcrTranslationSettings} from '../services/mockBackend'
import {readableError, subscribeOcrModelDownloadEvents} from '../services/recorderBackend'
import {SettingSelect, SettingTextInput, SettingToggle} from './controls'
import {ocrTranslationLanguageOptions, ocrTranslationProviders} from './panelShared'
import {useCallback, useEffect, useState} from 'react'
import type {RecorderCopy} from '../i18n'
import {type CaptureCapability} from '../services/mockBackend'
import {type OcrModelDownloadSnapshot, type OcrModelInfo, type OcrStatus} from '../services/recorderBackend'

export function OcrTranslationSettingsPanel({
  copy,
  translation,
  compact = false,
  onChange }: {
  copy: RecorderCopy
  translation: AppSettings['ocr']['translation']
  compact?: boolean
  onChange: (patch: Partial<AppSettings['ocr']['translation']>) => void
}) {
  const normalized = normalizeOcrTranslationSettings(translation)
  const enabled = normalized.provider !== 'disabled'
  const providerOptions = ocrTranslationProviders.map((provider) => ({
    value: provider,
    label: copy.settings.ocrTranslationProviderLabels[provider] ?? provider }))
  const languageOptions = ocrTranslationLanguageOptions.map((language) => ({value: language, label: language === 'auto' ? 'Auto' : language}))
  return (
    <div className={`setting-line ocr-translation-settings ${compact ? 'compact' : ''}`}>
      <span>{copy.settings.ocrTranslation}</span>
      <div className="ocr-translation-grid">
        <SettingSelect
          title={copy.settings.ocrTranslationProvider}
          value={normalized.provider}
          options={providerOptions}
          detail={copy.settings.ocrTranslationDetail}
          onChange={(provider) => {
            const nextProvider = provider === 'deepl' || provider === 'openai-compatible' ? provider : 'disabled'
            onChange({
              provider: nextProvider,
              privacyConfirmed: nextProvider !== 'disabled' ? normalized.privacyConfirmed : false,
              privacyConfirmedAt: nextProvider !== 'disabled' ? normalized.privacyConfirmedAt : '' })
          }}
        />
        {enabled && (
          <>
            <SettingTextInput
              title={copy.settings.ocrTranslationBaseUrl}
              value={normalized.baseUrl ?? ''}
              detail={copy.settings.ocrTranslationBaseUrlDetail}
              placeholder={normalized.provider === 'deepl' ? 'https://api-free.deepl.com/v2/translate' : 'https://api.example.com/v1'}
              onCommit={(baseUrl) => onChange({baseUrl})}
            />
            <SettingTextInput
              title={copy.settings.ocrTranslationApiKey}
              value={normalized.apiKey ?? ''}
              detail={copy.settings.ocrTranslationApiKeyDetail}
              inputType="password"
              placeholder={normalized.apiKeySet ? copy.settings.ocrTranslationApiKeySaved : 'sk-...'}
              onCommit={(apiKey) => onChange({apiKey, apiKeySet: Boolean(apiKey.trim())})}
            />
            {normalized.provider === 'openai-compatible' && (
              <SettingTextInput
                title={copy.settings.ocrTranslationModel}
                value={normalized.model ?? ''}
                placeholder="gpt-4o-mini"
                onCommit={(model) => onChange({model})}
              />
            )}
            <SettingSelect
              title={copy.settings.ocrTranslationSourceLanguage}
              value={normalized.sourceLanguage}
              options={languageOptions}
              onChange={(sourceLanguage) => onChange({sourceLanguage})}
            />
            <SettingSelect
              title={copy.settings.ocrTranslationTargetLanguage}
              value={normalized.targetLanguage}
              options={languageOptions.filter((option) => option.value !== 'auto')}
              onChange={(targetLanguage) => onChange({targetLanguage})}
            />
            <SettingToggle
              title={copy.settings.ocrTranslationPrivacy}
              checked={normalized.privacyConfirmed}
              detail={copy.settings.ocrTranslationPrivacyDetail}
              onChange={(privacyConfirmed) => onChange({
                privacyConfirmed,
                privacyConfirmedAt: privacyConfirmed ? new Date().toISOString() : '' })}
            />
          </>
        )}
      </div>
    </div>
  )
}

export function OcrModelSettings({copy, compact = false}: {copy: RecorderCopy; compact?: boolean}) {
  const [status, setStatus] = useState<OcrStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState('')
  const [packagePath, setPackagePath] = useState('')
  const [message, setMessage] = useState('')
  const [confirmModelId, setConfirmModelId] = useState('')
  const [downloads, setDownloads] = useState<Record<string, OcrModelDownloadSnapshot>>({})

  const refresh = useCallback(async (quiet = false) => {
    if (!quiet) setLoading(true)
    try {
      const [next, nextDownloads] = await Promise.all([
        getOcrStatus(),
        getOcrModelDownloads().catch(() => [] as OcrModelDownloadSnapshot[]),
      ])
      setStatus(next)
      setDownloads(Object.fromEntries(nextDownloads.map((download) => [download.modelId, download])))
      setMessage(next.message || '')
    } catch (error) {
      setMessage(readableError(error) || copy.settings.ocrModelsUnavailable)
    } finally {
      if (!quiet) setLoading(false)
    }
  }, [copy.settings.ocrModelsUnavailable])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.all([
      getOcrStatus(),
      getOcrModelDownloads().catch(() => [] as OcrModelDownloadSnapshot[]),
    ])
      .then(([next, nextDownloads]) => {
        if (cancelled) return
        setStatus(next)
        setDownloads(Object.fromEntries(nextDownloads.map((download) => [download.modelId, download])))
        setMessage(next.message || '')
      })
      .catch((error) => {
        if (!cancelled) setMessage(readableError(error) || copy.settings.ocrModelsUnavailable)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [copy.settings.ocrModelsUnavailable])

  useEffect(() => subscribeOcrModelDownloadEvents((snapshot) => {
    setDownloads((current) => ({...current, [snapshot.modelId]: snapshot}))
    if (snapshot.status === 'installed') {
      void refresh(true).finally(() => setMessage(copy.settings.ocrModelDownloadInstalled))
    } else if (snapshot.status === 'failed') {
      void refresh(true).finally(() => setMessage(snapshot.error || copy.settings.ocrModelDownloadFailed))
    }
  }), [copy.settings.ocrModelDownloadFailed, copy.settings.ocrModelDownloadInstalled, refresh])

  const runAction = async (key: string, action: () => Promise<void>, success: string) => {
    setBusy(key)
    setMessage('')
    try {
      await action()
      await refresh(true)
      setMessage(success)
    } catch (error) {
      await refresh(true).catch(() => undefined)
      setMessage(readableError(error) || copy.settings.ocrModelActionFailed)
    } finally {
      setBusy('')
    }
  }

  const importPackage = () => {
    const trimmed = packagePath.trim()
    if (!trimmed) {
      setMessage(copy.settings.ocrModelPackageEmpty)
      return
    }
    void runAction('import', async () => {
      await installOcrModelPackage(trimmed)
    }, copy.settings.ocrModelImported)
  }

  const downloadModel = (model: OcrModelInfo) => {
    void runAction(`download:${model.id}`, async () => {
      let downloadable = model
      if (!downloadable.downloadAvailable) {
        const nextStatus = await refreshOcrModelCatalog('')
        setStatus(nextStatus)
        downloadable = nextStatus.models.find((candidate) => candidate.id === model.id) ?? downloadable
      }
      if (!downloadable.downloadAvailable) {
        throw new Error(copy.settings.ocrModelDownloadUnavailable)
      }
      const snapshot = await startOcrModelDownload(downloadable.id)
      setDownloads((current) => ({...current, [model.id]: snapshot}))
    }, copy.settings.ocrModelDownloadQueued)
  }

  const refreshCatalog = () => {
    void runAction('catalog', async () => {
      const next = await refreshOcrModelCatalog('')
      setStatus(next)
    }, copy.settings.ocrModelCatalogRefreshed)
  }

  const cancelDownload = (model: OcrModelInfo) => {
    void runAction(`cancel-download:${model.id}`, async () => {
      const snapshot = await cancelOcrModelDownload(model.id)
      setDownloads((current) => ({...current, [model.id]: snapshot}))
    }, copy.settings.ocrModelDownloadCancelled)
  }

  const models = status?.models ?? []
  const statusValue = loading ? copy.settings.ocrModelsLoading : ocrStatusText(status, copy)
  const disabled = loading || busy !== ''

  return (
    <div className={`setting-line ocr-model-settings ${compact ? 'compact' : ''}`}>
      <span>{copy.settings.ocrModels}</span>
      <div className="setting-value">
        <strong>{statusValue}</strong>
        <button className="setting-action" type="button" disabled={disabled} onClick={() => void refresh()}>
          {copy.settings.ocrModelRefresh}
        </button>
        <button className="setting-action" type="button" disabled={disabled} onClick={refreshCatalog}>
          {busy === 'catalog' ? copy.settings.ocrModelCatalogRefreshing : copy.settings.ocrModelCatalogRefresh}
        </button>
      </div>
      <small>{message || `${copy.settings.ocrModelsDetail} ${copy.settings.ocrModelCatalogDetail}`}</small>
      <div className="ocr-model-package-row">
        <input
          className="setting-control-input"
          value={packagePath}
          placeholder={copy.settings.ocrModelPackagePath}
          onChange={(event) => setPackagePath(event.target.value)}
        />
        <button className="setting-action" type="button" disabled={disabled || packagePath.trim() === ''} onClick={importPackage}>
          {busy === 'import' ? copy.settings.ocrModelImporting : copy.settings.ocrModelPackageImport}
        </button>
      </div>
      <div className="ocr-model-list">
        {models.map((model) => (
          <OcrModelRow
            key={model.id}
            model={model}
            copy={copy}
            busy={busy}
            disabled={disabled}
            download={downloads[model.id]}
            confirming={confirmModelId === model.id}
            onUse={() => {
              setConfirmModelId(model.id)
              setMessage(copy.settings.ocrModelSwitchConfirm(model.name))
            }}
            onConfirmUse={() => void runAction(`use:${model.id}`, async () => {
              await setActiveOcrModel(model.id)
            }, copy.settings.ocrModelActivated).finally(() => setConfirmModelId(''))}
            onCancelUse={() => {
              setConfirmModelId('')
              setMessage(status?.message || '')
            }}
            onRemove={() => void runAction(`remove:${model.id}`, async () => {
              await removeOcrModel(model.id)
            }, copy.settings.ocrModelRemoved)}
            onDownload={() => downloadModel(model)}
            onCancelDownload={() => cancelDownload(model)}
          />
        ))}
      </div>
      {status && (
        <small>
          {copy.settings.ocrModelWorkerStatus}: {status.workerPath || copy.common.notRun}
          {status.runtimeDir ? ` · ${copy.settings.ocrModelRuntime}: ${status.runtimeDir}` : ''}
        </small>
      )}
    </div>
  )
}

export function OcrModelRow({
  model,
  copy,
  busy,
  disabled,
  download,
  confirming,
  onUse,
  onConfirmUse,
  onCancelUse,
  onRemove,
  onDownload,
  onCancelDownload }: {
  model: OcrModelInfo
  copy: RecorderCopy
  busy: string
  disabled: boolean
  download?: OcrModelDownloadSnapshot
  confirming: boolean
  onUse: () => void
  onConfirmUse: () => void
  onCancelUse: () => void
  onRemove: () => void
  onDownload: () => void
  onCancelDownload: () => void
}) {
  const downloadActive = download?.status === 'queued' || download?.status === 'running'
  const downloadFinished = download?.status === 'installed'
  const downloadFailed = download?.status === 'failed'
  const state = downloadActive ? copy.settings.ocrModelDownloading : ocrModelStateText(model, copy)
  const channel = copy.settings.ocrModelChannelLabels[model.channel] ?? model.channel
  const detail = ocrModelDetail(model, copy)
  const progressText = downloadActive ? ocrModelDownloadProgressText(download) : ''
  return (
    <div className={`ocr-model-row ${model.active ? 'active' : ''} ${downloadActive ? 'downloading' : ''}`}>
      <div>
        <strong>{model.name}</strong>
        <span>{channel} · {model.id}</span>
        {detail && <small>{detail}</small>}
        {!model.installed && !model.downloadAvailable && <small>{copy.settings.ocrModelDownloadUnavailable}</small>}
        {progressText && <small>{progressText}</small>}
        {downloadFinished && !model.installed && <small>{copy.settings.ocrModelDownloadInstalled}</small>}
        {downloadFailed && <small>{download?.error || copy.settings.ocrModelDownloadFailed}</small>}
      </div>
      <div className="ocr-model-actions">
        <b className={`status-badge ${ocrModelBadge(model)}`}>{state}</b>
        {!model.installed && !downloadActive && (
          <button className="setting-action" type="button" disabled={disabled || busy === `download:${model.id}`} onClick={onDownload}>
            {busy === `download:${model.id}` ? copy.settings.ocrModelDownloading : copy.settings.ocrModelDownload}
          </button>
        )}
        {downloadActive && (
          <button className="setting-action" type="button" disabled={busy === `cancel-download:${model.id}`} onClick={onCancelDownload}>
            {busy === `cancel-download:${model.id}` ? copy.settings.ocrModelCancellingDownload : copy.settings.ocrModelCancelDownload}
          </button>
        )}
        {model.installed && model.verified && !model.active && (
          <button className="setting-action" type="button" disabled={disabled || busy === `use:${model.id}`} onClick={onUse}>
            {copy.settings.ocrModelUse}
          </button>
        )}
        {model.installed && !model.active && (
          <button className="setting-action danger" type="button" disabled={disabled || busy === `remove:${model.id}`} onClick={onRemove}>
            {busy === `remove:${model.id}` ? copy.settings.ocrModelRemoving : copy.settings.ocrModelRemove}
          </button>
        )}
      </div>
      {confirming && (
        <div className="ocr-model-confirm" role="alert">
          <small>{copy.settings.ocrModelSwitchRisk}</small>
          <div>
            <button className="setting-action" type="button" disabled={disabled || busy === `use:${model.id}`} onClick={onConfirmUse}>
              {busy === `use:${model.id}` ? copy.settings.ocrModelActivating : copy.settings.ocrModelConfirmUse}
            </button>
            <button className="setting-action" type="button" disabled={busy !== ''} onClick={onCancelUse}>
              {copy.settings.ocrModelCancelUse}
            </button>
          </div>
        </div>
      )}
      {downloadActive && (
        <div className="ocr-model-progress" aria-label={progressText}>
          <span style={{width: `${Math.max(3, Math.min(100, Math.round(download?.percent ?? 0)))}%`}} />
        </div>
      )}
    </div>
  )
}

export function ocrStatusText(status: OcrStatus | null, copy: RecorderCopy) {
  if (!status) return copy.settings.ocrModelsUnavailable
  return copy.settings.ocrStatusLabels[status.status] ?? status.status
}

export function ocrModelStateText(model: OcrModelInfo, copy: RecorderCopy) {
  if (model.active && model.verified) return copy.settings.ocrModelActive
  if (model.installed && model.verified) return copy.settings.ocrModelVerified
  if (model.installed) return copy.settings.ocrModelInvalid
  return copy.settings.ocrModelMissing
}

export function ocrModelBadge(model: OcrModelInfo): CaptureCapability['status'] {
  if (model.active && model.verified) return 'available'
  if (model.installed && model.verified) return 'queued'
  if (model.installed) return 'blocked'
  return 'unsupported'
}

export function ocrModelDetail(model: OcrModelInfo, copy: RecorderCopy) {
  const details = [
    model.version,
    model.language.length > 0 ? model.language.join('/') : '',
    !model.installed && model.downloadAvailable && model.downloadBytes ? `${copy.settings.ocrModelDownloadSize}: ${formatBytes(model.downloadBytes)}` : '',
    model.smokeAssetReady ? copy.settings.ocrModelSmokeReady : '',
    model.smokeError || model.verificationError || (model.missingFiles.length > 0 ? `${copy.settings.ocrModelMissing}: ${model.missingFiles.join(', ')}` : ''),
  ].filter(Boolean)
  return details.join(' · ')
}

export function ocrModelDownloadProgressText(download?: OcrModelDownloadSnapshot) {
  if (!download) return ''
  const percent = Math.max(0, Math.min(100, Math.round(download.percent || 0)))
  return `${percent}% · ${formatBytes(download.downloadedBytes)} / ${formatBytes(download.totalBytes)}`
}

