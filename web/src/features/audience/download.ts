/** Saves a blob as a file through a temporary link. */
export function saveBlob(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

/** The file name from a Content-Disposition header, or a fallback. */
export function fileName(res: Response | undefined, fallback: string): string {
  const cd = res?.headers.get('Content-Disposition') ?? ''
  const m = /filename="?([^";]+)"?/.exec(cd)
  return m?.[1] ?? fallback
}
