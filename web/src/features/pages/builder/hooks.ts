import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useBuilder } from './context'
import { m } from './messages'

/** Resumes a paused signup form (PageBuilderStates · a blocking check), then draws its blocks and runs the checks again. */
export function useResumeForm(): { resume: (id: string, name: string) => Promise<void>; busy: boolean } {
  const t = useT()
  const b = useBuilder()
  const qc = useQueryClient()
  const [busy, setBusy] = useState(false)
  const resume = async (id: string, name: string) => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/forms/{form}/status', { params: { path: { workspace: b.slug, form: id } }, body: { status: 'live' } })
      toast({ kind: 'success', title: t(m.formResumed, { name }) })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/forms'] })
      b.pe?.repaintLive()
      await b.refreshChecks()
    } catch (e) {
      toast({ kind: 'error', title: t(m.formResumeFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return { resume, busy }
}
