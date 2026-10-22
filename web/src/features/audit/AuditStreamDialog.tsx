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
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { deniedReason, useWorkspace } from '@/lib/session'
import { copyText } from '../access/parts'
import { m } from './messages'

const RETENTION = [90, 180, 365, 730, 1095]

/**
 * The audit stream and retention: entries are posted in signed batches to a
 * webhook (for a SIEM or warehouse) and kept for the chosen number of days.
 */
export function AuditStreamDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/audit/settings', { params: { path: { workspace: slug } } })
  if (!q.data)
    return (
      <Dialog open onOpenChange={(o) => !o && onClose()} title={t(m.streamTitle)} footer={<Button onClick={onClose}>{t(common.close)}</Button>}>
        {q.isError ? (
          <Note tone="bad" label={t(m.notLoaded)}>
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
  const t = useT()
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
      setError({ field: 'url', message: t(m.urlRequired) })
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
          toast({ title: tt(m.settingsSaved), detail: tt(m.loggedAs, { action: 'audit.settings_updated' }) })
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
        eyebrow={<span className="k">{t(m.secretEyebrow)}</span>}
        title={t(m.secretTitle)}
        description={<span className="text-[13.5px]">{t(m.secretBody)}</span>}
        hideClose
        dismissible={false}
        footer={
          <Button variant="primary" onClick={onClose}>
            {t(common.done)}
          </Button>
        }
      >
        <div className="row flex-nowrap gap-2">
          <input className="in mono flex-1 text-[12.5px]" value={secret} readOnly aria-label={t(m.secretInput)} onFocus={(e) => e.currentTarget.select()} />
          <Button
            variant="primary"
            onClick={async () => {
              const ok = await copyText(secret)
              toast({ kind: ok ? 'success' : 'error', title: ok ? tt(common.copied) : tt(m.copyFailed) })
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
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(m.dialogEyebrow)}</span>}
      title={t(m.streamTitle)}
      description={<span className="text-[13.5px]">{t(m.streamBody)}</span>}
      hideClose
      footer={
        <>
          <Button onClick={onClose}>{readOnly ? t(common.close) : t(common.cancel)}</Button>
          {readOnly ? null : (
            <Button variant="primary" loading={save.isPending} onClick={submit}>
              {t(common.save)}
            </Button>
          )}
        </>
      }
    >
      {readOnly ? <Note tone="neutral">{deniedReason('workspace:manage')}</Note> : null}
      {st.enabled ? (
        st.failingSince ? (
          <Note tone="bad" label={t(m.failing)}>
            {t(m.failingBody, { host: st.host ?? '', since: ago(st.failingSince, now), hasError: st.lastError ? 'yes' : 'no', error: st.lastError ?? '' })}
          </Note>
        ) : (
          <KeyValues
            className="text-[12.5px]"
            items={[
              [
                t(m.kvStatus),
                <Badge key="s" tone="ok" dot>
                  {t(m.streamingTo, { host: st.host ?? '' })}
                </Badge>,
              ],
              [
                t(m.kvDelivered),
                t(m.delivered, { count: st.delivered, delivered: count(st.delivered), hasLast: st.lastOkAt ? 'yes' : 'no', last: st.lastOkAt ? ago(st.lastOkAt, now) : '' }),
              ],
            ]}
          />
        )
      ) : null}
      <label className="row gap-2.5 text-[13.5px]">
        <input type="checkbox" checked={enabled} disabled={readOnly} onChange={(e) => setEnabled(e.target.checked)} />
        {t(m.streamToggle)}
      </label>
      <label className="lbl">
        {t(m.destinationUrl)}
        <input
          className="in mono"
          type="url"
          value={url}
          disabled={readOnly || !enabled}
          onChange={(e) => setUrl(e.target.value)}
          placeholder={'https://siem.example.com/ingest/audit'}
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
          {t(m.rotateSecret)}
        </label>
      ) : null}
      <label className="lbl">
        {t(m.keepFor)}
        <select className="in max-w-[200px]" value={days} disabled={readOnly} onChange={(e) => setDays(Number(e.target.value))}>
          {[...new Set([...RETENTION, s.retentionDays])]
            .sort((a, b) => a - b)
            .map((d) => (
              <option key={d} value={d}>
                {t(m.retentionDays, { days: d })}
              </option>
            ))}
        </select>
        <span className="k font-medium">{t(m.retentionHint, { count: s.total, total: count(s.total) })}</span>
      </label>
      {error && error.field !== 'url' ? (
        <Note tone="bad" label={t(m.notSaved)}>
          {error.message}
        </Note>
      ) : null}
    </Dialog>
  )
}
