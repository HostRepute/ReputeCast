import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { ago, count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { deniedReason, useWorkspace } from '@/lib/session'
import { copyText } from '../access/parts'

const RETENTION = [90, 180, 365, 730, 1095]

/**
 * The audit stream and retention: entries are posted in signed batches to a
 * webhook (for a SIEM or warehouse) and kept for the chosen number of days.
 */
export function AuditStreamDialog({ onClose }: { onClose: () => void }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/audit/settings', { params: { path: { workspace: slug } } })
  if (!q.data)
    return (
      <Dialog open onOpenChange={(o) => !o && onClose()} title="Audit stream and retention" footer={<Button onClick={onClose}>Close</Button>}>
        {q.isError ? (
          <Note tone="bad" label="Not loaded">
            {errorMessage(q.error)}
          </Note>
        ) : (
          <div className="flex flex-col gap-2" aria-busy="true">
            <Skeleton h={36} />
            <Skeleton w="60%" />
          </div>
        )}
      </Dialog>
    )
  return <StreamForm s={q.data} onClose={onClose} />
}

function StreamForm({ s, onClose }: { s: Schemas['AuditSettings']; onClose: () => void }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const now = useNow()
  const [enabled, setEnabled] = useState(s.stream.enabled)
  const [url, setUrl] = useState(s.stream.url ?? '')
  const [rotate, setRotate] = useState(false)
  const [days, setDays] = useState(s.retentionDays)
  const [error, setError] = useState<{ field?: string; message: string }>()
  const [secret, setSecret] = useState<string>()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/audit/settings')
  const st = s.stream
  const readOnly = !s.canManage
  const submit = () => {
    setError(undefined)
    if (enabled && !/^https:\/\/[^/\s]+/.test(url.trim())) {
      setError({ field: 'url', message: 'Enter the HTTPS URL that receives the entries.' })
      return
    }
    save.mutate(
      {
        params: { path: { workspace: slug } },
        body: { retentionDays: days, stream: { enabled, url: url.trim() || undefined, rotateSecret: rotate || undefined } },
      },
      {
        onSuccess: (r) => {
          void qc.invalidateQueries({ queryKey: ['audit'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/audit/settings'] })
          if (r.secret) {
            setSecret(r.secret)
            return
          }
          toast({ title: 'Audit settings saved', detail: 'logged as audit.settings_updated' })
          onClose()
        },
        onError: (e) => {
          const loc = e instanceof ApiError ? e.problem.errors?.[0]?.location : undefined
          setError({ field: loc?.includes('url') ? 'url' : undefined, message: errorMessage(e) })
        },
      },
    )
  }
  if (secret)
    return (
      <Dialog
        open
        onOpenChange={() => undefined}
        eyebrow={<span className="k">Dialog · audit stream secret</span>}
        title="Copy the signing secret now"
        description={<span className="text-[13.5px]">Batches are signed with HMAC-SHA256 like the webhooks. This is the only time the secret is shown.</span>}
        hideClose
        dismissible={false}
        footer={
          <Button variant="primary" onClick={onClose}>
            Done
          </Button>
        }
      >
        <div className="row flex-nowrap gap-2">
          <input className="in mono flex-1 text-[12.5px]" value={secret} readOnly aria-label="Audit stream signing secret" onFocus={(e) => e.currentTarget.select()} />
          <Button
            variant="primary"
            onClick={async () => {
              const ok = await copyText(secret)
              toast({ kind: ok ? 'success' : 'error', title: ok ? 'Copied' : 'Copy failed' })
            }}
          >
            Copy
          </Button>
        </div>
      </Dialog>
    )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">Dialog · audit log</span>}
      title="Audit stream and retention"
      description={<span className="text-[13.5px]">Stream every entry to your SIEM or warehouse as it happens. Entries are immutable either way.</span>}
      hideClose
      footer={
        <>
          <Button onClick={onClose}>{readOnly ? 'Close' : 'Cancel'}</Button>
          {readOnly ? null : (
            <Button variant="primary" loading={save.isPending} onClick={submit}>
              Save
            </Button>
          )}
        </>
      }
    >
      {readOnly ? <Note tone="neutral">{deniedReason('workspace:manage')}</Note> : null}
      {st.enabled ? (
        st.failingSince ? (
          <Note tone="bad" label="Failing">
            Batches to {st.host} have failed since {ago(st.failingSince, now)}
            {st.lastError ? `: ${st.lastError}` : ''}. Entries wait and go out in order once it answers 2xx.
          </Note>
        ) : (
          <KeyValues
            className="text-[12.5px]"
            items={[
              [
                'Status',
                <Badge key="s" tone="ok" dot>
                  Streaming to {st.host}
                </Badge>,
              ],
              ['Delivered', `${count(st.delivered)} entries${st.lastOkAt ? ` · last batch ${ago(st.lastOkAt, now)}` : ''}`],
            ]}
          />
        )
      ) : null}
      <label className="row gap-2.5 text-[13.5px]">
        <input type="checkbox" checked={enabled} disabled={readOnly} onChange={(e) => setEnabled(e.target.checked)} />
        Stream entries to a webhook
      </label>
      <label className="lbl">
        Destination URL
        <input
          className="in mono"
          type="url"
          value={url}
          disabled={readOnly || !enabled}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://siem.example.com/ingest/audit"
          aria-invalid={error?.field === 'url' || undefined}
          aria-describedby={error?.field === 'url' ? 'as-url-err' : undefined}
        />
        {error?.field === 'url' ? (
          <span className="err" id="as-url-err" role="alert">
            {error.message}
          </span>
        ) : null}
      </label>
      {st.hasSecret && enabled ? (
        <label className="row gap-2.5 text-[13.5px]">
          <input type="checkbox" checked={rotate} disabled={readOnly} onChange={(e) => setRotate(e.target.checked)} />
          Rotate the signing secret (shown once after saving)
        </label>
      ) : null}
      <label className="lbl">
        Keep entries for
        <select className="in max-w-[200px]" value={days} disabled={readOnly} onChange={(e) => setDays(Number(e.target.value))}>
          {[...new Set([...RETENTION, s.retentionDays])]
            .sort((a, b) => a - b)
            .map((d) => (
              <option key={d} value={d}>
                {d} days
              </option>
            ))}
        </select>
        <span className="k font-medium">Older entries are removed every night. {count(s.total)} entries are kept now.</span>
      </label>
      {error && error.field !== 'url' ? (
        <Note tone="bad" label="Not saved">
          {error.message}
        </Note>
      ) : null}
    </Dialog>
  )
}
