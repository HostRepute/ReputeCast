import { $api, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { monthDayTimeSeconds } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { copyText } from '../access/parts'
import { attemptsText, endpointText, eventText, responseBadge } from './parts'

/** One delivery: what was sent (the signed JSON body) and what the endpoint answered. */
export function DeliveryDialog({ id, onClose }: { id: string; onClose: () => void }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/webhooks/deliveries/{deliveryId}', { params: { path: { workspace: slug, deliveryId: id } } })
  const d = q.data?.delivery
  const body = q.data ? JSON.stringify(q.data.payload, null, 2) : ''
  const resp = d ? responseBadge(d) : undefined
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">Dialog · delivery payload</span>}
      title={d ? eventText(d) : 'Delivery'}
      description={d ? <span className="text-[13.5px]">{endpointText(d)}</span> : undefined}
      size="lg"
      footer={
        <>
          <Button
            className="mr-auto"
            disabledReason={q.data ? undefined : 'Loading…'}
            onClick={async () => {
              const ok = await copyText(body)
              toast({ kind: ok ? 'success' : 'error', title: ok ? 'Payload copied' : 'Copy failed' })
            }}
          >
            Copy JSON
          </Button>
          <Button variant="primary" onClick={onClose}>
            Close
          </Button>
        </>
      }
    >
      {q.isError ? (
        <Note tone="bad" label="Not loaded">
          {errorMessage(q.error)}
        </Note>
      ) : !q.data || !d ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton w="60%" />
          <Skeleton h={160} r={8} />
        </div>
      ) : (
        <>
          <KeyValues
            className="text-[12.5px]"
            items={[
              ['Created', monthDayTimeSeconds(d.createdAt)],
              ['Response', resp ? <Badge tone={resp.tone}>{resp.label}</Badge> : <span className="k">Not attempted yet</span>],
              ['Attempts', attemptsText(d)],
              ['Delivery ID', <span className="mono">{d.id}</span>],
            ]}
          />
          <span className="text-[12px] font-semibold text-ink-2">Payload</span>
          <pre className="pre max-h-[320px]">{body}</pre>
          {q.data.response ? (
            <>
              <span className="text-[12px] font-semibold text-ink-2">Response body</span>
              <pre className="pre max-h-[160px]">{q.data.response}</pre>
            </>
          ) : null}
        </>
      )}
    </Dialog>
  )
}
