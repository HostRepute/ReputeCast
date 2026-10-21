import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, duration } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { copyText } from '../access/parts'
import { m } from './messages'
import { hostOf, type Webhook } from './parts'

type Events = Schemas['WebhookEventInfo'][]

const queryKey = ['get', '/api/v1/workspaces/{workspace}/webhooks'] as const

const mono = (c: ReactNode) => <span className="mono">{c}</span>

/** Groups the event catalogue by prefix: email → email.sent, email.delivered… */
function byGroup(events: Events) {
  const out = new Map<string, Events>()
  for (const e of events) {
    const g = e.name.split('.')[0]!
    out.set(g, [...(out.get(g) ?? []), e])
  }
  return [...out.entries()]
}

/**
 * Add or edit an endpoint; events can be picked by whole group (a wildcard).
 * A failing endpoint opens with the failure explained (SettingsStates).
 */
export function WebhookDialog({
  webhook,
  events,
  maxAttempts,
  onClose,
  onSecret,
  onShowLog,
}: {
  webhook?: Webhook
  events: Events
  maxAttempts: number
  onClose: () => void
  onSecret: (secret: string, w: Webhook, action: 'created' | 'rotated') => void
  onShowLog: (id: string) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const now = useNow()
  const [url, setUrl] = useState(webhook?.url ?? 'https://')
  const [name, setName] = useState(webhook?.name ?? '')
  const [description, setDescription] = useState(webhook?.description ?? '')
  const [picked, setPicked] = useState<string[]>(webhook?.events ?? [])
  const [error, setError] = useState<{ field?: string; message: string }>()
  const [busy, setBusy] = useState<string>()
  const [confirmDelete, setConfirmDelete] = useState(false)
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/webhooks')
  const update = $api.useMutation('put', '/api/v1/workspaces/{workspace}/webhooks/{webhookId}')
  const refresh = () => {
    void qc.invalidateQueries({ queryKey })
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/webhooks/deliveries'] })
  }
  const fail = (e: unknown) => {
    const loc = e instanceof ApiError ? e.problem.errors?.[0]?.location : undefined
    setError({ field: loc?.replace('body.', ''), message: errorMessage(e) })
  }
  const toggle = (ev: string, on: boolean) => setPicked((cur) => (on ? [...cur, ev] : cur.filter((x) => x !== ev)))
  const save = () => {
    setError(undefined)
    if (!/^https?:\/\/[^/\s]+/.test(url.trim())) {
      setError({ field: 'url', message: t(m.urlRequired) })
      return
    }
    if (picked.length === 0) {
      setError({ field: 'events', message: t(m.eventsRequired) })
      return
    }
    const body = { url: url.trim(), name: name.trim() || hostOf(url.trim()), description: description.trim() || undefined, events: picked }
    if (webhook) {
      update.mutate(
        { params: { path: { workspace: slug, webhookId: webhook.id } }, body },
        {
          onSuccess: () => {
            refresh()
            toast({ title: tt(m.savedToast), detail: tt(m.loggedAs, { action: 'webhook.updated' }) })
            onClose()
          },
          onError: fail,
        },
      )
    } else {
      create.mutate(
        { params: { path: { workspace: slug } }, body },
        {
          onSuccess: (r) => {
            refresh()
            onSecret(r.secret, r.webhook, 'created')
          },
          onError: fail,
        },
      )
    }
  }
  const act = async (what: 'test' | 'rotate' | 'pause' | 'resume') => {
    if (!webhook) return
    setBusy(what)
    setError(undefined)
    const path = { params: { path: { workspace: slug, webhookId: webhook.id } } }
    try {
      if (what === 'test') {
        const r = (await http.POST('/api/v1/workspaces/{workspace}/webhooks/{webhookId}/test', path)).data!
        const d = r.delivery
        toast({
          kind: d.state === 'delivered' ? 'success' : 'error',
          title: d.state === 'delivered' ? tt(m.testDelivered) : tt(m.testFailed),
          detail: d.state === 'delivered' ? tt(m.testDeliveredDetail, { code: String(d.statusCode), ms: count(d.durationMs) }) : (d.error ?? `${d.statusCode} ${d.statusText ?? ''}`),
        })
      } else if (what === 'rotate') {
        const r = (await http.POST('/api/v1/workspaces/{workspace}/webhooks/{webhookId}/rotate-secret', path)).data!
        onSecret(r.secret, webhook, 'rotated')
      } else {
        await http.POST(what === 'pause' ? '/api/v1/workspaces/{workspace}/webhooks/{webhookId}/pause' : '/api/v1/workspaces/{workspace}/webhooks/{webhookId}/resume', path)
        toast({ title: what === 'pause' ? tt(m.pausedToast) : tt(m.resumedToast), detail: what === 'pause' ? tt(m.pausedDetail) : tt(m.resumedDetail) })
        onClose()
      }
      refresh()
    } catch (e) {
      fail(e)
    } finally {
      setBusy(undefined)
    }
  }
  const failingFor = webhook?.failingSince ? (now.getTime() - new Date(webhook.failingSince).getTime()) / 60000 : 0
  const opt = (c: ReactNode) => <span className="k font-medium">{c}</span>
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{webhook ? t(m.eyebrowEdit) : t(m.eyebrowAdd)}</span>}
      title={webhook ? t(m.titleEdit, { host: hostOf(webhook.url) }) : t(m.titleAdd)}
      description={<span className="text-[13.5px]">{t(m.dialogDesc, { max: maxAttempts })}</span>}
      size="lg"
      hideClose
      dismissible={false}
      footer={
        <>
          {webhook ? (
            <Button variant="danger" className="mr-auto" onClick={() => setConfirmDelete(true)}>
              {t(m.deleteEndpoint)}
            </Button>
          ) : null}
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={create.isPending || update.isPending} onClick={save}>
            {webhook ? t(m.saveEndpoint) : t(m.submitAdd)}
          </Button>
        </>
      }
    >
      {webhook?.status === 'failing' ? (
        <div className="card flex flex-col gap-2 border-bad-line px-4.5 py-4" role="alert">
          <div className="flex items-center gap-2.5">
            <Badge tone="bad">{t(m.failingBadge)}</Badge>
            <span className="font-semibold">{t(m.failingTitle, { host: hostOf(webhook.url), duration: duration(failingFor) })}</span>
          </div>
          <span className="k">{t(m.failingBody, { count: webhook.queued })}</span>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="primary" onClick={() => onShowLog(webhook.id)}>
              {t(m.openLog)}
            </Button>
            <Button size="sm" loading={busy === 'pause'} onClick={() => void act('pause')}>
              {t(m.pauseThis)}
            </Button>
          </div>
        </div>
      ) : null}
      <div className="grid grid-cols-[2fr_1fr] gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          {t(m.urlLabel)}
          <input
            className="in mono"
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            spellCheck={false}
            aria-invalid={error?.field === 'url' || undefined}
            aria-describedby={error?.field === 'url' ? 'wh-url-err' : 'wh-url-hint'}
          />
          {error?.field === 'url' ? (
            <span className="err" id="wh-url-err" role="alert">
              {error.message}
            </span>
          ) : (
            <span className="k font-medium" id="wh-url-hint">
              {t(m.urlHint)}
            </span>
          )}
        </label>
        <label className="lbl">
          {t(m.nameLabel)}
          <input className="in" type="text" value={name} onChange={(e) => setName(e.target.value)} maxLength={80} placeholder={t(m.namePlaceholder)} />
        </label>
      </div>
      <label className="lbl">
        <span>
          <T msg={m.descriptionLabel} values={{ opt }} />
        </span>
        <input className="in" type="text" value={description} onChange={(e) => setDescription(e.target.value)} maxLength={200} placeholder={t(m.descriptionPlaceholder)} />
      </label>
      <fieldset className="m-0 flex flex-col gap-2 rounded-lg border border-line px-3.5 py-2.5" aria-describedby={error?.field === 'events' ? 'wh-ev-err' : undefined}>
        <legend className="px-1 text-[12px] font-semibold text-ink-2">{picked.length === 0 ? t(m.eventsLegendNone) : t(m.eventsLegend, { events: picked.join(', ') })}</legend>
        {byGroup(events).map(([g, evs]) => {
          const all = `${g}.*`
          const allOn = picked.includes(all)
          return (
            <div key={g} className="flex flex-col gap-1.5 border-t border-line-2 pt-2 first-of-type:border-t-0 first-of-type:pt-0">
              <label className="row gap-2 text-[13px] font-semibold">
                <input type="checkbox" checked={allOn} onChange={(e) => toggle(all, e.target.checked)} />
                <T msg={m.allGroup} values={{ group: g, pattern: all, mono: (c) => <span className="k mono">{c}</span> }} />
              </label>
              <div className="row gap-x-3.5 gap-y-1.5 pl-6">
                {evs.map((e) => (
                  <label key={e.name} className="row gap-1.5 text-[12.5px]" title={e.description}>
                    <input type="checkbox" checked={allOn || picked.includes(e.name)} disabled={allOn} onChange={(x) => toggle(e.name, x.target.checked)} />
                    <span className="mono">{e.name}</span>
                  </label>
                ))}
              </div>
            </div>
          )
        })}
        {error?.field === 'events' ? (
          <span className="err" id="wh-ev-err" role="alert">
            {error.message}
          </span>
        ) : null}
      </fieldset>
      {webhook ? (
        <div className="row gap-2">
          <Button size="sm" loading={busy === 'test'} onClick={() => void act('test')}>
            {t(m.sendTest)}
          </Button>
          <Button size="sm" loading={busy === 'rotate'} onClick={() => void act('rotate')}>
            {t(m.rotateSecret)}
          </Button>
          {webhook.status === 'paused' ? (
            <Button size="sm" loading={busy === 'resume'} onClick={() => void act('resume')}>
              {t(m.resume)}
            </Button>
          ) : webhook.status !== 'failing' ? (
            <Button size="sm" loading={busy === 'pause'} onClick={() => void act('pause')}>
              {t(m.pause)}
            </Button>
          ) : null}
        </div>
      ) : null}
      {error && !['url', 'events'].includes(error.field ?? '') ? (
        <Note tone="bad" label={t(m.notSaved)}>
          {error.message}
        </Note>
      ) : null}
      {webhook && confirmDelete ? (
        <ConfirmDialog
          open
          onOpenChange={setConfirmDelete}
          tone="bad"
          badge={t(m.deleteBadge)}
          title={t(m.deleteTitle, { host: hostOf(webhook.url) })}
          description={
            webhook.queued
              ? t(m.deleteDescQueued, { events: webhook.events.join(', '), count: webhook.queued })
              : t(m.deleteDesc, { events: webhook.events.join(', ') })
          }
          confirmLabel={t(m.deleteConfirm)}
          busyLabel={t(m.deleting)}
          onConfirm={async () => {
            await http.DELETE('/api/v1/workspaces/{workspace}/webhooks/{webhookId}', { params: { path: { workspace: slug, webhookId: webhook.id } } })
            toast({ kind: 'deleted', title: tt(m.deletedToast), detail: tt(m.loggedAs, { action: 'webhook.deleted' }) })
            refresh()
            onClose()
          }}
        />
      ) : null}
    </Dialog>
  )
}

/** The signing secret, shown once after adding an endpoint or rotating it. */
export function WebhookSecretDialog({ secret, webhook, action, header, onClose }: { secret: string; webhook: Webhook; action: 'created' | 'rotated'; header: string; onClose: () => void }) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && copied && onClose()}
      eyebrow={<span className="k">{t(m.secretEyebrow, { action })}</span>}
      title={t(m.secretTitle)}
      description={<span className="text-[13.5px]">{t(m.secretDesc, { host: hostOf(webhook.url), events: webhook.events.join(', ') })}</span>}
      hideClose
      dismissible={false}
      footer={
        <>
          <label className="row mr-auto gap-2 text-[13px]">
            <input type="checkbox" checked={copied} onChange={(e) => setCopied(e.target.checked)} />
            {t(m.secretCopiedCheck)}
          </label>
          <Button variant="primary" disabledReason={copied ? undefined : t(m.secretConfirm)} onClick={onClose}>
            {t(common.done)}
          </Button>
        </>
      }
    >
      <div className="row flex-nowrap gap-2">
        <input className="in mono flex-1 text-[12.5px]" value={secret} readOnly aria-label={t(m.secretLabel)} onFocus={(e) => e.currentTarget.select()} />
        <Button
          variant="primary"
          onClick={async () => {
            const ok = await copyText(secret)
            if (ok) setCopied(true)
            toast({ kind: ok ? 'success' : 'error', title: ok ? tt(common.copied) : tt(m.copyFailed) })
          }}
        >
          {t(common.copy)}
        </Button>
      </div>
      <Note tone="warn" label={t(m.shownOnce)}>
        <T msg={action === 'rotated' ? m.secretNoteRotated : m.secretNote} values={{ header, mono }} />
      </Note>
    </Dialog>
  )
}
