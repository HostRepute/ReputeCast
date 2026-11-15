import { useState } from 'react'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { useT } from '@/i18n'
import { md } from './draft.messages'

/** Each confirmation captures one submission, not permission for later input. */
export function useSharingReview(recipient: string, external = false) {
  const t = useT()
  const [pending, setPending] = useState<{ submit: () => Promise<unknown> }>()
  const review = (submit: () => Promise<unknown>) => setPending({ submit })
  const dialog = pending ? <ConfirmDialog open onOpenChange={(open) => { if (!open) setPending(undefined) }}
    tone="warn" badge={t(md.sharingTitle)} title={t(md.sharingTitle)} confirmLabel={t(md.sharingConfirm)}
    description={t(md.sharingDisclosure, { recipient })} onConfirm={() => {
      setPending(undefined)
      // The owning form displays progress/errors and keeps cancellation usable.
      void pending.submit().catch(() => {})
    }}>
    <div className="flex min-w-0 flex-col gap-3">
      <p className="m-0 text-sm">{t(md.sharingRisk)}</p>
      <p className="m-0 text-sm">{t(md.aggregateRisk)}</p>
      <p className="k m-0">{t(external ? md.externalPolicy : md.providerPolicy)}</p>
    </div>
  </ConfirmDialog> : null
  return [review, dialog] as const
}
