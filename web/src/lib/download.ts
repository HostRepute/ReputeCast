import { errorMessage, http } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { fileName, saveBlob } from '@/features/audience/download'
import { t } from '@/i18n'
import { m } from './download.messages'

/**
 * Downloads a CSV export (an audited GET answering text/csv) and says so
 * (review 2026-11 EA-2): the file is saved under the server's name, or
 * `fallback`.
 */
export async function downloadCsv(path: string, params: Record<string, unknown>, fallback: string): Promise<void> {
  try {
    const { data, response } = await (http.GET as (p: string, o: unknown) => Promise<{ data?: unknown; response: Response }>)(path, { params, parseAs: 'blob' })
    if (data) saveBlob(data as Blob, fileName(response, fallback))
    toast({ kind: 'success', title: t(m.exported), detail: t(m.exportedDetail) })
  } catch (e) {
    toast({ kind: 'error', title: t(m.failed), detail: errorMessage(e) })
  }
}
