import { clock } from '@/lib/format'

/** "64 KB" — sizes are shown in whole KB, as Gmail counts them. */
export function kb(bytes: number | undefined | null): string {
  return `${Math.round((bytes ?? 0) / 1024)} KB`
}

/** "9:00 AM" as inbox lists show it. */
export function inboxTime(d: Date | string): string {
  const [h = '0', m = '00'] = clock(d).split(':')
  const hour = Number(h)
  const ampm = hour >= 12 ? 'PM' : 'AM'
  const h12 = hour % 12 === 0 ? 12 : hour % 12
  return `${h12}:${m} ${ampm}`
}

/** Status word of the header badge ("Draft", "Scheduled"…). */
export function statusWord(status: string): string {
  switch (status) {
    case 'draft':
      return 'Draft'
    case 'approval':
      return 'Waiting for approval'
    case 'approved':
      return 'Approved'
    case 'scheduled':
      return 'Scheduled'
    case 'sending':
      return 'Sending'
    case 'paused':
      return 'Paused'
    case 'sent':
      return 'Sent'
    case 'cancelled':
      return 'Cancelled'
    case 'failed':
      return 'Failed'
  }
  return status
}

/** Lines of a document, for "212 lines". */
export function lineCount(src: string): number {
  return src ? src.split('\n').length : 0
}

/** Read a file the person picked as text. */
export function readText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader()
    r.onload = () => resolve(String(r.result ?? ''))
    r.onerror = () => reject(r.error)
    r.readAsText(file)
  })
}

/** Download text as a file. */
export function download(name: string, text: string, type = 'text/plain') {
  const url = URL.createObjectURL(new Blob([text], { type }))
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/** File-name friendly version of a campaign name. */
export function slugify(s: string): string {
  return (
    s
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '') || 'campaign'
  )
}
