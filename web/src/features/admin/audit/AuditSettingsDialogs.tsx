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
import { copyText } from '@/features/access/parts'
import { ago, count } from '@/lib/format'
import { useNow } from '@/lib/hooks'

type Settings = Schemas['InstanceAuditSettings']

const RETENTION = [90, 180, 365, 730, 1095, 3650]

function useSettings() {
  return $api.useQuery('get', '/api/v1/admin/audit/settings')
}

function useSave() {
  const qc = useQueryClient()
  const save = $api.useMutation('put', '/api/v1/admin/audit/settings')
  const invalidate = async () => {
    await qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/audit/settings'] })
    await qc.invalidateQueries({ queryKey: ['admin-audit'] })
  }
  return { save, invalidate }
}

function Loading({ title, onClose, error }: { title: string; onClose: () => void; error?: unknown }) {
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} title={title} footer={<Button onClick={onClose}>Close</Button>}>
      {error ? (
        <Note tone="bad" label="Not loaded">
          {errorMessage(error)}
        </Note>
      ) : (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton h={36} />
          <Skeleton w="60%" />
        </div>
      )}
    </Dialog>
  )
}

/** "Change retention": how long instance-level entries are kept. */
export function RetentionDialog({ onClose }: { onClose: () => void }) {
  const q = useSettings()
  if (!q.data) return <Loading title="Instance audit retention" onClose={onClose} error={q.error} />
  return <RetentionForm s={q.data} onClose={onClose} />
}

function RetentionForm({ s, onClose }: { s: Settings; onClose: () => void }) {
  const [days, setDays] = useState(s.retentionDays)
  const [error, setError] = useState<string>()
  const { save, invalidate } = useSave()
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !save.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">Dialog · audit log</span>}
      title="Instance audit retention"
      description={
        <span className="text-[13.5px]">
          How long instance-level entries (sign-ins, SSO, policies and users) are kept. Each workspace keeps its own retention in its audit settings.
        </span>
      }
      footer={
        <>
          <Button onClick={onClose} disabled={save.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            loading={save.isPending}
            disabledReason={days === s.retentionDays ? 'Pick another retention to save' : undefined}
            onClick={() => {
              setError(undefined)
              save.mutate(
                { body: { retentionDays: days } },
                {
                  onSuccess: async () => {
                    await invalidate()
                    toast({ kind: 'success', title: `Instance entries are kept ${days} days`, detail: 'logged as audit.settings_updated' })
                    onClose()
                  },
                  onError: (e) => setError(errorMessage(e)),
                },
              )
            }}
          >
            Save retention
          </Button>
        </>
      }
    >
      <label className="lbl">
        Keep instance-level entries for
        <select className="in max-w-[200px]" value={days} onChange={(e) => setDays(Number(e.target.value))}>
          {[...new Set([...RETENTION, s.retentionDays])]
            .sort((a, b) => a - b)
            .map((d) => (
              <option key={d} value={d}>
                {d} days{d === 365 ? ' (default)' : ''}
              </option>
            ))}
        </select>
        <span className="k font-medium">{count(s.total)} instance-level entries are kept now. Entries are immutable until they age out.</span>
      </label>
      {days < s.retentionDays ? (
        <Note tone="warn" label="Shorter">
          Entries older than {days} days are removed at the next hourly cleanup. Export them first if you need them.
        </Note>
      ) : null}
      {error ? (
        <Note tone="bad" label="Not saved">
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

/** "Stream instance events": post instance-level entries to a webhook. */
export function StreamDialog({ onClose }: { onClose: () => void }) {
  const q = useSettings()
  if (!q.data) return <Loading title="Stream instance events" onClose={onClose} error={q.error} />
  return <StreamForm s={q.data} onClose={onClose} />
}

function StreamForm({ s, onClose }: { s: Settings; onClose: () => void }) {
  const now = useNow()
  const st = s.stream
  const [enabled, setEnabled] = useState(st.enabled || !st.url)
  const [url, setUrl] = useState(st.url ?? '')
  const [rotate, setRotate] = useState(false)
  const [error, setError] = useState<{ field?: string; message: string }>()
  const [secret, setSecret] = useState<string>()
  const { save, invalidate } = useSave()
  const submit = () => {
    setError(undefined)
    if (enabled && !/^https:\/\/[^/\s]+/.test(url.trim())) {
      setError({ field: 'url', message: 'Enter the HTTPS URL that receives the entries.' })
      return
    }
    save.mutate(
      { body: { stream: { enabled, url: url.trim() || undefined, rotateSecret: rotate || undefined } } },
      {
        onSuccess: async (r) => {
          await invalidate()
          if (r.secret) {
            setSecret(r.secret)
            return
          }
          toast({ kind: 'success', title: enabled ? 'Instance events are streamed' : 'Instance events are no longer streamed', detail: 'logged as audit.settings_updated' })
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
          <input className="in mono flex-1 text-[12.5px]" value={secret} readOnly aria-label="Instance audit stream signing secret" onFocus={(e) => e.currentTarget.select()} />
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
      onOpenChange={(o) => !o && !save.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">Dialog · audit log</span>}
      title="Stream instance events"
      description={
        <span className="text-[13.5px]">
          Post sign-ins, SSO, policy and user changes to your SIEM or warehouse as they happen, in signed batches. Workspaces stream their own events from their
          audit settings.
        </span>
      }
      footer={
        <>
          <Button onClick={onClose} disabled={save.isPending}>
            Cancel
          </Button>
          <Button variant="primary" loading={save.isPending} onClick={submit}>
            Save
          </Button>
        </>
      }
    >
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
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        Stream instance-level entries to a webhook
      </label>
      <label className="lbl">
        Destination URL
        <input
          className="in mono"
          type="url"
          value={url}
          disabled={!enabled}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://siem.example.com/ingest/audit"
          aria-invalid={error?.field === 'url' || undefined}
          aria-describedby={error?.field === 'url' ? 'is-url-err' : undefined}
        />
        {error?.field === 'url' ? (
          <span className="err" id="is-url-err" role="alert">
            {error.message}
          </span>
        ) : null}
      </label>
      {st.hasSecret && enabled ? (
        <label className="row gap-2.5 text-[13.5px]">
          <input type="checkbox" checked={rotate} onChange={(e) => setRotate(e.target.checked)} />
          Rotate the signing secret (shown once after saving)
        </label>
      ) : null}
      {s.workspaceStreams.length ? (
        <span className="k font-medium">
          Streaming their own events: {s.workspaceStreams.map((w) => `${w.name} → ${w.host}${w.failing ? ' (failing)' : ''}`).join(', ')}.
        </span>
      ) : null}
      {error && error.field !== 'url' ? (
        <Note tone="bad" label="Not saved">
          {error.message}
        </Note>
      ) : null}
    </Dialog>
  )
}
