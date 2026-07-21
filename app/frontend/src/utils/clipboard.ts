export async function writeClipboardText(text: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // Some desktop webviews expose clipboard but reject without focus; fall back to a temporary selection.
    }
  }
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'true')
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  textarea.style.top = '0'
  document.body.appendChild(textarea)
  textarea.select()
  try {
    if (!document.execCommand('copy')) throw new Error('clipboard copy failed')
  } finally {
    document.body.removeChild(textarea)
  }
}

export async function writeClipboardImage(blob: Blob) {
  const ClipboardItemConstructor = typeof window !== 'undefined' ? window.ClipboardItem : undefined
  if (!navigator.clipboard?.write || !ClipboardItemConstructor) {
    throw new Error('image clipboard is unavailable')
  }
  await navigator.clipboard.write([
    new ClipboardItemConstructor({'image/png': blob}),
  ])
}
