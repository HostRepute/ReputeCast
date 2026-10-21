import { useQueryClient } from '@tanstack/react-query'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN } from '../EarnPage'
import { mof } from './messages'

/** "End FALL26 now?" (EarnOfferEditor): its code stops working; those who used it keep their discount. */
export function EndOfferDialog({ offer, provider, onClose, onEnded }: { offer: Schemas['Offer']; provider: string; onClose: () => void; onEnded?: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const end = $api.useMutation('post', `${EARN}/offers/{id}/end`)
  const submit = () =>
    end.mutate(
      { params: { path: { workspace: slug, id: offer.id } } },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/offers`] })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/offers/{id}`] })
          toast({ kind: 'success', title: t(mof.ended, { name: offer.name }) })
          onEnded?.()
          onClose()
        },
        onError: (e) => toast({ kind: 'error', title: t(mof.endOffer), detail: errorMessage(e) }),
      },
    )
  const code = offer.kind === 'signup' && offer.code
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mof.endEyebrow, { kind: offer.kind })}</span>}
      title={t(mof.endTitle, { name: offer.name })}
      description={offer.kind === 'retention' ? t(mof.endTextRetention) : t(mof.endText, { count: offer.results.redemptions })}
      footer={
        <>
          <Button onClick={onClose}>{t(mof.keepRunning)}</Button>
          <Button variant="danger-primary" loading={end.isPending} onClick={submit}>
            {t(mof.endNowButton)}
          </Button>
        </>
      }
    >
      <dl className="kv">
        <dt>{t(mof.endEnds)}</dt>
        <dd>{offer.endsAt ? t(mof.endNow, { date: dayMonthDate(offer.endsAt) }) : t(mof.endNowPlain)}</dd>
        {code ? (
          <>
            <dt>{t(mof.endCode)}</dt>
            <dd>{t(mof.endCodeLine, { code: offer.code })}</dd>
          </>
        ) : null}
        <dt>{t(mof.endStripe, { provider })}</dt>
        <dd>{t(mof.endStripeLine)}</dd>
      </dl>
    </Dialog>
  )
}
