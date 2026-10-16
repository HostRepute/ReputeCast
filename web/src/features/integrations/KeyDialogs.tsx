import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { CopyField, Footer, ItemRow, keyPermissions } from './ConnectDialog'
import { m } from './messages'
import { base, entryName, itemName, PermChips, stamp, tags, useRefreshHub, type App, type ConnectKind, type ItemLock, type Secret } from './shared'

/** The location line above the key dialogs ("Settings › Integrations › Zapier"). */
function Where({ name }: { name: string }) {
  const t = useT()
  return <span className="k">{t(m.eyebrowPath, { name })}</span>
}

function Loading() {
  const t = useT()
  return (
    <div className="flex flex-col gap-2.5" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loadingDialog)}
      </span>
      <Skeleton w="80%" />
      <Skeleton w="100%" h={60} />
      <Skeleton w="60%" />
    </div>
  )
}

/** Change what an integration can do: the key's permissions change in place. */
export function ChangeDialog({ app, open, onOpenChange }: { app: App; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[app.kind])
  const info = $api.useQuery('get', `${base}/apps/{kind}/connect`, { params: { path: { workspace: slug, kind: app.kind } } }, { enabled: open, staleTime: 0 })
  const current = new Set(app.items.filter((i) => i.on).map((i) => i.key))
  const [on, setOn] = useState(current)
  const [listId, setListId] = useState(app.key?.scope.all ? '' : (app.key?.scope.lists[0]?.id ?? ''))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const refresh = useRefreshHub()

  // An item that's on but the person can't grant stays on, and can be turned off (not back on).
  const items: ItemLock[] = (info.data?.items ?? []).map((i) => (i.locked && current.has(i.key) && on.has(i.key) ? { ...i, locked: false } : i))
  const perms = info.data ? keyPermissions(info.data, items, on) : null

  async function save() {
    if (!app.connectionId) return
    setBusy(true)
    setError(undefined)
    try {
      const res = await http.POST(`${base}/connections/{id}/change`, {
        params: { path: { workspace: slug, id: app.connectionId } },
        body: { items: [...on], lists: listId ? [listId] : [] },
      })
      await refresh()
      onOpenChange(false)
      toast({ title: t(m.changedToast, { name }), detail: t(m.changedDetail, { removed: res.data?.webhooksRemoved ?? 0 }) })
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  const toggle = (k: string, v: boolean) =>
    setOn((prev) => {
      const next = new Set(prev)
      if (v) next.add(k)
      else next.delete(k)
      return next
    })

  return (
    <Dialog open={open} onOpenChange={onOpenChange} eyebrow={<span className="k">{t(m.changeEyebrow)}</span>} title={t(m.changeTitle, { name })} hideClose>
      {!info.data || !perms ? (
        info.isError ? <Note tone="bad">{errorMessage(info.error)}</Note> : <Loading />
      ) : (
        <>
          <div className="ih-prl">
            <div className="ih-phd">
              {t(m.triggers)} <span className="k">{t(m.triggersHint, { name })}</span>
            </div>
            {items
              .filter((i) => i.kind === 'trigger')
              .map((it) => (
                <ItemRow key={it.key} item={it} on={on.has(it.key)} onChange={(v) => toggle(it.key, v)} />
              ))}
            <div className="ih-phd">
              {t(m.actions)} <span className="k">{t(m.actionsHint)}</span>
            </div>
            {items
              .filter((i) => i.kind === 'action')
              .map((it) => (
                <ItemRow key={it.key} item={it} on={on.has(it.key)} onChange={(v) => toggle(it.key, v)} />
              ))}
          </div>
          <div className="ih-gets">
            <div className="row justify-between gap-2">
              <span className="font-semibold">{t(m.keyGets)}</span>
              <span className="k">{t(m.updatesAsYouSwitch)}</span>
            </div>
            {perms.short.length ? <PermChips perms={perms.short} sensitive={perms.sensitive} /> : <span className="k">{t(m.nothingOn)}</span>}
            <span className="k">
              {[...[...perms.included].map(([p, by]) => t(m.includesView, { perm: by, implied: p })), t(m.worksOnlyFor, { count: on.size })].join(' ')}
            </span>
          </div>
          <label className="lbl max-w-[320px]">
            {t(m.listScope)}
            <select className="in" value={listId} onChange={(e) => setListId(e.target.value)}>
              <option value="">{t(m.allLists)}</option>
              {info.data.lists.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
          </label>
          {error ? (
            <Note tone="bad" role="alert">
              {error}
            </Note>
          ) : null}
          <Footer>
            <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
            <Button variant="primary" onClick={save} loading={busy} disabledReason={on.size === 0 ? t(m.nothingOn) : undefined}>
              {t(m.changeConfirm)}
            </Button>
          </Footer>
        </>
      )}
    </Dialog>
  )
}

/**
 * Rotate: a new key, shown once. The old one keeps working for 24 hours,
 * with "Stop the old key now" offered.
 */
export function RotateDialog({
  connectionId,
  kind,
  name,
  oldMasked,
  open,
  onOpenChange,
}: {
  connectionId: string
  kind: ConnectKind
  /** "Zapier", or a site's host. */
  name: string
  oldMasked: string
  open: boolean
  onOpenChange: (o: boolean) => void
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [secret, setSecret] = useState<Secret | null>(null)
  const [copied, setCopied] = useState(false)
  const [stopped, setStopped] = useState(false)
  const refresh = useRefreshHub()
  const wp = kind === 'wordpress'

  async function rotate() {
    setBusy(true)
    try {
      const res = await http.POST(`${base}/connections/{id}/rotate`, { params: { path: { workspace: slug, id: connectionId } } })
      setSecret(res.data ?? null)
      void refresh()
    } catch (e) {
      toast({ kind: 'error', title: t(m.rotateError), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  async function stopOld() {
    setBusy(true)
    try {
      await http.POST(`${base}/connections/{id}/stop-old-key`, { params: { path: { workspace: slug, id: connectionId } } })
      setStopped(true)
      void refresh()
      toast({ title: t(m.stoppedToast) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.stopError), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  const close = (o: boolean) => {
    if (!o) {
      setSecret(null)
      setCopied(false)
      setStopped(false)
    }
    onOpenChange(o)
  }

  if (!secret) {
    return (
      <Dialog open={open} onOpenChange={close} eyebrow={<Where name={name} />} title={t(m.rotateTitle, { name })} hideClose>
        <span className="text-[13.5px] text-ink-nav">{t(m.rotateBody, { name: wp ? t(m.wordpress) : name })}</span>
        <Footer>
          <Button onClick={() => close(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={rotate} loading={busy}>
            {t(m.rotateConfirm)}
          </Button>
        </Footer>
      </Dialog>
    )
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && copied && close(false)}
      dismissible={false}
      eyebrow={<span className="k">{t(m.rotatedEyebrow, { time: clock(secret.key.createdAt) })}</span>}
      title={t(m.rotatedTitle, { name: wp ? t(m.wordpress) : name })}
      hideClose
    >
      <CopyField label={t(m.instanceUrl)} value={secret.instanceUrl} copyLabel={t(m.copyInstanceUrl)} />
      <CopyField label={t(m.keyLabel, { key: secret.key.name })} value={secret.secret} primary copyLabel={t(m.copyKey)} />
      <Note tone="warn">
        <span className="font-semibold">{t(m.shownOnceTitle)}</span>
        <div>{wp ? t(m.wpShownOnceBody) : t(m.shownOnceBody, { name })}</div>
      </Note>
      {secret.oldKeyUntil && !stopped ? (
        <Note tone="info">
          <div>{t(m.oldKeyWorks, { masked: oldMasked, time: stamp(secret.oldKeyUntil), name: wp ? t(m.wordpress) : name })}</div>
          <div className="mt-2">
            <Button size="sm" variant="danger" onClick={stopOld} loading={busy}>
              {t(m.stopOldKey)}
            </Button>
          </div>
        </Note>
      ) : null}
      <Footer>
        <label className="row mr-auto gap-2 text-[13px]">
          <input type="checkbox" checked={copied} onChange={(e) => setCopied(e.target.checked)} />
          {t(m.iCopied)}
        </label>
        <Button variant="primary" onClick={() => close(false)} disabledReason={copied ? undefined : t(m.iCopied)}>
          {t(m.done)}
        </Button>
      </Footer>
    </Dialog>
  )
}

/** Stop a rotated key before its 24 hours are up (the Connection card's link). */
export function StopOldKeyDialog({ connectionId, masked, open, onOpenChange }: { connectionId: string; masked: string; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const [busy, setBusy] = useState(false)
  const refresh = useRefreshHub()
  async function stop() {
    setBusy(true)
    try {
      await http.POST(`${base}/connections/{id}/stop-old-key`, { params: { path: { workspace: slug, id: connectionId } } })
      await refresh()
      onOpenChange(false)
      toast({ title: t(m.stoppedToast) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.stopError), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange} title={t(m.stopOldTitle)} hideClose dismissible={!busy}>
      <span className="text-[13.5px] text-ink-nav">
        <T msg={m.stopOldBody} values={{ ...tags, masked }} />
      </span>
      <Footer>
        <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
        <Button variant="danger-primary" onClick={stop} loading={busy}>
          {t(m.stopOldConfirm)}
        </Button>
      </Footer>
    </Dialog>
  )
}

/** Take over: the caller becomes the key's owner; anything they don't hold is listed first. */
export function TakeOverDialog({
  connectionId,
  name,
  open,
  onOpenChange,
}: {
  connectionId: string
  name: string
  open: boolean
  onOpenChange: (o: boolean) => void
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', `${base}/connections/{id}/take-over`, { params: { path: { workspace: slug, id: connectionId } } }, { enabled: open, staleTime: 0 })
  const [busy, setBusy] = useState(false)
  const refresh = useRefreshHub()
  const p = q.data

  async function takeOver() {
    setBusy(true)
    try {
      await http.POST(`${base}/connections/{id}/take-over`, { params: { path: { workspace: slug, id: connectionId } } })
      await refresh()
      onOpenChange(false)
      toast({ title: t(m.takenOverToast, { key: p?.key.name ?? name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.takeOverError), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  const owner = p
    ? p.key.ownerStatus === 'deactivated' && p.key.ownerLeftAt
      ? t(m.ownerDeactivatedAt, { name: p.key.owner.name, time: stamp(p.key.ownerLeftAt) })
      : p.key.ownerStatus === 'removed' && p.key.ownerLeftAt
        ? t(m.ownerRemovedAt, { name: p.key.owner.name, time: stamp(p.key.ownerLeftAt) })
        : p.key.owner.name
    : ''
  return (
    <Dialog open={open} onOpenChange={onOpenChange} eyebrow={<Where name={name} />} title={t(m.takeOverTitle, { name })} hideClose dismissible={!busy}>
      {!p ? (
        q.isError ? <Note tone="bad">{errorMessage(q.error)}</Note> : <Loading />
      ) : (
        <>
          <span className="text-[13.5px] text-ink-nav">{t(m.takeOverBody, { key: p.key.name, name })}</span>
          <KeyValues
            items={[
              [
                t(m.key),
                <>
                  {p.key.name} <span className="mono k">{p.key.masked}</span>
                </>,
              ],
              [t(m.ownerNow), owner],
              [t(m.newOwner), t(m.newOwnerValue, { name: p.newOwner.name, role: p.roleName })],
            ]}
          />
          {p.missing.length === 0 ? (
            <div className="flex flex-col gap-2">
              <span className="row flex-nowrap gap-2 font-semibold text-ok-fg">
                <Icon name="check" size={15} strokeWidth={2.2} />
                {t(m.holdsAll, { count: p.held.length })}
              </span>
              <PermChips perms={p.held} sensitive={p.key.sensitive} />
            </div>
          ) : (
            <Note tone="warn">
              <div>{t(m.holdsSome, { perms: formatList(p.missing), count: p.missing.length })}</div>
              <ul className="m-0 mt-1 list-disc pl-[18px]">
                {p.stops.map((s) => (
                  <li key={s}>{itemName(t, s)}</li>
                ))}
              </ul>
            </Note>
          )}
          <Footer>
            <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
            <Button variant="primary" onClick={takeOver} loading={busy}>
              {t(m.takeOverConfirm)}
            </Button>
          </Footer>
        </>
      )}
    </Dialog>
  )
}

/** Disconnect: what stops (by name), what's removed, what stays; a reason for the audit log. */
export function DisconnectDialog({
  connectionId,
  kind,
  name,
  open,
  onOpenChange,
  onDone,
}: {
  connectionId: string
  kind: ConnectKind
  /** "Zapier", or a site's host. */
  name: string
  open: boolean
  onOpenChange: (o: boolean) => void
  onDone?: () => void
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', `${base}/connections/{id}/disconnect`, { params: { path: { workspace: slug, id: connectionId } } }, { enabled: open, staleTime: 0 })
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const refresh = useRefreshHub()
  const p = q.data
  const wp = kind === 'wordpress'
  const tool = wp ? t(m.wordpress) : name

  async function disconnect() {
    setBusy(true)
    try {
      await http.POST(`${base}/connections/{id}/disconnect`, { params: { path: { workspace: slug, id: connectionId } }, body: { reason: reason.trim() || undefined } })
      await refresh()
      onOpenChange(false)
      toast({ kind: 'deleted', title: t(m.disconnectedToast, { name }), detail: t(m.disconnectedDetail) })
      onDone?.()
    } catch (e) {
      toast({ kind: 'error', title: t(m.disconnectError, { name }), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<Where name={name} />}
      title={t(m.disconnectDialogTitle, { name })}
      description={<span className="text-[13.5px]">{wp ? t(m.disconnectSiteBody) : t(m.disconnectDialogBody, { kind })}</span>}
      className="card border-bad-line"
      hideClose
      dismissible={!busy}
    >
      {!p ? (
        q.isError ? <Note tone="bad">{errorMessage(q.error)}</Note> : <Loading />
      ) : (
        <>
          <div className="flex flex-col">
            <div className="ih-bl">
              <Badge tone="bad">{t(m.stops)}</Badge>
              {wp ? (
                <span>{t(m.stopsSite, { host: name })}</span>
              ) : p.workflows.length ? (
                <div className="flex min-w-0 flex-col gap-1">
                  <span>{t(m.stopsList, { kind })}</span>
                  <ul className="m-0 list-disc pl-[18px]">
                    {p.workflows.map((w, i) => (
                      <li key={i}>
                        <span className="font-semibold">{w.unnamed ? t(m.unnamed, { kind }) : w.name}</span> <span className="k">{t(m.lastRun, { time: stamp(w.lastRunAt) })}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              ) : (
                <span>{t(m.stopsNone, { kind })}</span>
              )}
            </div>
            <div className="ih-bl">
              <Badge tone="neutral">{t(m.removed)}</Badge>
              <span>
                {p.key ? <T msg={m.removedKey} values={{ ...tags, key: p.key.name, masked: p.key.masked }} /> : null}{' '}
                {p.webhooks ? t(m.removedHooks, { count: p.webhooks, name: tool, queued: p.queued, dead: p.dead }) : t(m.removedNoHooks, { name: tool })}
              </span>
            </div>
            <div className="ih-bl">
              <Badge tone="ok">{t(m.stays)}</Badge>
              <span>{t(m.staysLine, { name: tool })}</span>
            </div>
          </div>
          <label className="lbl">
            {t(m.reasonLabel)}
            <input className="in" value={reason} onChange={(e) => setReason(e.target.value)} placeholder={t(m.reasonPlaceholder)} maxLength={300} />
          </label>
          <Footer>
            <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
            <Button variant="danger-primary" onClick={disconnect} loading={busy}>
              {wp ? t(m.disconnectSite) : t(m.disconnectConfirm, { name })}
            </Button>
          </Footer>
        </>
      )}
    </Dialog>
  )
}
