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
import { m as ma } from '@/features/audit/messages'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { m as mi } from './messages'

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
  const t = useT()
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} title={title} footer={<Button onClick={onClose}>{t(common.close)}</Button>}>
      {error ? (
        <Note tone="bad" label={t(ma.notLoaded)}>
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
  const t = useT()
  const q = useSettings()
  if (!q.data) return <Loading title={t(mi.retentionTitle)} onClose={onClose} error={q.error} />
  return <RetentionForm s={q.data} onClose={onClose} />
}

function RetentionForm({ s, onClose }: { s: Settings; onClose: () => void }) {
  const t = useT()
  const [days, setDays] = useState(s.retentionDays)
  const [error, setError] = useState<string>()
  const { save, invalidate } = useSave()
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !save.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">{t(ma.dialogEyebrow)}</span>}
      title={t(mi.retentionTitle)}
      description={<span className="text-[13.5px]">{t(mi.retentionBody)}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={save.isPending}>
            {t(common.cancel)}
          </Button>
          <Button
            variant="primary"
            loading={save.isPending}
            disabledReason={days === s.retentionDays ? t(mi.pickAnother) : undefined}
            onClick={() => {
              setError(undefined)
              save.mutate(
                { body: { retentionDays: days } },
                {
                  onSuccess: async () => {
                    await invalidate()
                    toast({ kind: 'success', title: tt(mi.retentionSaved, { days }), detail: tt(ma.loggedAs, { action: 'audit.settings_updated' }) })
                    onClose()
                  },
                  onError: (e) => setError(errorMessage(e)),
                },
              )
            }}
          >
            {t(mi.saveRetention)}
          </Button>
        </>
      }
    >
      <label className="lbl">
        {t(mi.keepFor)}
        <select className="in max-w-[200px]" value={days} onChange={(e) => setDays(Number(e.target.value))}>
          {[...new Set([...RETENTION, s.retentionDays])]
            .sort((a, b) => a - b)
            .map((d) => (
              <option key={d} value={d}>
                {t(mi.retentionOption, { days: d, isDefault: d === 365 ? 'yes' : 'no' })}
              </option>
            ))}
        </select>
        <span className="k font-medium">{t(mi.retentionKept, { count: s.total, total: count(s.total) })}</span>
      </label>
      {days < s.retentionDays ? (
        <Note tone="warn" label={t(mi.shorter)}>
          {t(mi.shorterNote, { days })}
        </Note>
      ) : null}
      {error ? (
        <Note tone="bad" label={t(ma.notSaved)}>
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

/** "Stream instance events": post instance-level entries to a webhook. */
export function StreamDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const q = useSettings()
  if (!q.data) return <Loading title={t(mi.streamTitle)} onClose={onClose} error={q.error} />
  return <StreamForm s={q.data} onClose={onClose} />
}

function StreamForm({ s, onClose }: { s: Settings; onClose: () => void }) {
  const t = useT()
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
      setError({ field: 'url', message: t(ma.urlRequired) })
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
          toast({ kind: 'success', title: enabled ? tt(mi.streamOn) : tt(mi.streamOff), detail: tt(ma.loggedAs, { action: 'audit.settings_updated' }) })
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
        eyebrow={<span className="k">{t(ma.secretEyebrow)}</span>}
        title={t(ma.secretTitle)}
        description={<span className="text-[13.5px]">{t(ma.secretBody)}</span>}
        hideClose
        dismissible={false}
        footer={
          <Button variant="primary" onClick={onClose}>
            {t(common.done)}
          </Button>
        }
      >
        <div className="row flex-nowrap gap-2">
          <input className="in mono flex-1 text-[12.5px]" value={secret} readOnly aria-label={t(mi.secretInput)} onFocus={(e) => e.currentTarget.select()} />
          <Button
            variant="primary"
            onClick={async () => {
              const ok = await copyText(secret)
              toast({ kind: ok ? 'success' : 'error', title: ok ? tt(common.copied) : tt(ma.copyFailed) })
            }}
          >
            {t(common.copy)}
          </Button>
        </div>
      </Dialog>
    )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !save.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">{t(ma.dialogEyebrow)}</span>}
      title={t(mi.streamTitle)}
      description={<span className="text-[13.5px]">{t(mi.streamBody)}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={save.isPending}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" loading={save.isPending} onClick={submit}>
            {t(common.save)}
          </Button>
        </>
      }
    >
      {st.enabled ? (
        st.failingSince ? (
          <Note tone="bad" label={t(ma.failing)}>
            {t(ma.failingBody, { host: st.host ?? '', since: ago(st.failingSince, now), hasError: st.lastError ? 'yes' : 'no', error: st.lastError ?? '' })}
          </Note>
        ) : (
          <KeyValues
            className="text-[12.5px]"
            items={[
              [
                t(ma.kvStatus),
                <Badge key="s" tone="ok" dot>
                  {t(ma.streamingTo, { host: st.host ?? '' })}
                </Badge>,
              ],
              [
                t(ma.kvDelivered),
                t(ma.delivered, { count: st.delivered, delivered: count(st.delivered), hasLast: st.lastOkAt ? 'yes' : 'no', last: st.lastOkAt ? ago(st.lastOkAt, now) : '' }),
              ],
            ]}
          />
        )
      ) : null}
      <label className="row gap-2.5 text-[13.5px]">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        {t(mi.streamToggle)}
      </label>
      <label className="lbl">
        {t(ma.destinationUrl)}
        <input
          className="in mono"
          type="url"
          value={url}
          disabled={!enabled}
          onChange={(e) => setUrl(e.target.value)}
          placeholder={'https://siem.example.com/ingest/audit'}
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
          {t(ma.rotateSecret)}
        </label>
      ) : null}
      {s.workspaceStreams.length ? (
        <span className="k font-medium">
          {t(mi.workspaceStreams, {
            list: s.workspaceStreams.map((w) => t(mi.workspaceStream, { name: w.name, host: w.host, failing: w.failing ? 'yes' : 'no' })).join(', '),
          })}
        </span>
      ) : null}
      {error && error.field !== 'url' ? (
        <Note tone="bad" label={t(ma.notSaved)}>
          {error.message}
        </Note>
      ) : null}
    </Dialog>
  )
}
