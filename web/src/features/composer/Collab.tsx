import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useId, useRef, useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, andList } from '@/lib/format'
import { useComposer } from './context'
import { mx } from './collab.messages'

/**
 * Who else has the campaign open (CO-10, the Composer board's "Also in this
 * campaign"), and Share for review (ComposerWrite): a link to one version
 * that anyone can view and comment on without signing in.
 */

type Present = Schemas['ComposerPresent']
type ReviewLink = Schemas['ComposerReviewLink']

const presencePath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/presence' as const

/**
 * Says every 20 seconds that this person has the campaign open, and
 * whether they edited in the last minute; returns who else does.
 */
export function usePresence(): Present[] {
  const c = useComposer()
  const [others, setOthers] = useState<Present[]>([])
  const lastEdit = useRef(0)
  const firstRev = useRef(c.rev)
  useEffect(() => {
    if (c.rev !== firstRev.current) lastEdit.current = Date.now()
  }, [c.rev])
  const active = c.kind !== 'transactional'
  const { slug } = c
  const id = c.campaign.id
  useEffect(() => {
    if (!active) return
    let stopped = false
    const path = { workspace: slug, campaignId: id }
    const ping = () => {
      void http
        .PUT(presencePath, { params: { path }, body: { editing: Date.now() - lastEdit.current < 60_000 } })
        .then(({ data }) => {
          if (!stopped && data) setOthers(data.others)
        })
        .catch(() => {})
    }
    ping()
    const h = window.setInterval(ping, 20_000)
    const leave = () => void http.DELETE(presencePath, { params: { path }, keepalive: true }).catch(() => {})
    window.addEventListener('pagehide', leave)
    return () => {
      stopped = true
      window.clearInterval(h)
      window.removeEventListener('pagehide', leave)
      leave()
    }
  }, [active, slug, id])
  return others
}

/** "MC Marcus Chen viewing": the people with the campaign open besides you. */
export function PresenceChip() {
  const t = useT()
  const others = usePresence()
  if (!others.length) return null
  const shown = others.slice(0, 3)
  const words = others.length === 1 ? `${others[0]!.name} ${t(mx.presenceWord, { editing: others[0]!.editing ? 'yes' : 'no' })}` : andList(others.map((o) => o.name))
  return (
    <div className="cb-pres" role="group" aria-label={t(mx.presenceGroup)}>
      {shown.map((o) => (
        <span key={o.userId} className="pav" role="img" aria-label={t(mx.presenceName, { name: o.name, editing: o.editing ? 'yes' : 'no' })} title={t(mx.presenceName, { name: o.name, editing: o.editing ? 'yes' : 'no' })}>
          <Avatar name={o.name} color={o.color} size={28} />
          <i className={o.editing ? 'ed' : undefined} />
        </span>
      ))}
      {others.length > shown.length ? <span className="k">{t(mx.presenceMore, { count: others.length - shown.length })}</span> : null}
      <span className="pt" aria-hidden="true">
        {words}
      </span>
    </div>
  )
}

const linksPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/review-links' as const

function SharePopover({ id, onClose }: { id: string; onClose: () => void }) {
  const c = useComposer()
  const t = useT()
  const qc = useQueryClient()
  const [link, setLinkState] = useState<ReviewLink | null>(null)
  // "just now" for a link made a moment ago, else "12 min ago".
  const [when, setWhen] = useState('')
  const setLink = (l: ReviewLink | null) => {
    setLinkState(l)
    if (l) setWhen(Date.now() - Date.parse(l.createdAt) < 45_000 ? t(mx.justNow) : ago(l.createdAt))
  }
  const [busy, setBusy] = useState(true)
  const [error, setError] = useState('')
  const [revoked, setRevoked] = useState(false)
  const [savedFirst, setSavedFirst] = useState(false)
  const box = useRef<HTMLDivElement>(null)
  const input = useRef<HTMLInputElement>(null)
  const path = { workspace: c.slug, campaignId: c.campaign.id }
  const version = () => c.currentVersion?.() ?? c.campaign.version
  const create = async () => {
    setBusy(true)
    setError('')
    setRevoked(false)
    try {
      const { data } = await http.POST(linksPath, { params: { path } })
      setLink(data!)
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/versions'] })
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  // Escape closes it while focus is inside.
  useEffect(() => {
    const el = box.current
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      e.stopPropagation()
      onClose()
    }
    el?.addEventListener('keydown', onKey)
    return () => el?.removeEventListener('keydown', onKey)
  }, [onClose])
  useEffect(() => {
    box.current?.focus()
    let cancelled = false
    void (async () => {
      // Reviewers see the saved version: save edits first.
      if (c.hasUnsaved?.() && c.canEdit) {
        const ok = await c.save({ manual: true, quiet: true })
        if (!cancelled && ok) setSavedFirst(true)
      }
      try {
        const { data } = await http.GET(linksPath, { params: { path } })
        if (cancelled) return
        const newest = data?.items[0]
        if (newest) {
          setLink(newest)
          setBusy(false)
        } else await create()
      } catch (e) {
        if (!cancelled) {
          setError(errorMessage(e))
          setBusy(false)
        }
      }
    })()
    return () => {
      cancelled = true
    }
    // Once, when the popover opens.
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
  const copy = async () => {
    if (!link) return
    try {
      await navigator.clipboard.writeText(link.url)
      toast({ kind: 'success', title: t(mx.copied) })
    } catch {
      input.current?.select()
      toast({ kind: 'error', title: t(mx.copyFailed) })
    }
  }
  const revoke = async () => {
    if (!link) return
    setBusy(true)
    try {
      await http.DELETE(`${linksPath}/{linkId}`, { params: { path: { ...path, linkId: link.id } } })
      setLinkState(null)
      setRevoked(true)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const current = version()
  return (
    <div
      ref={box}
      id={id}
      className="sharepop"
      role="dialog"
      aria-labelledby={`${id}-h`}
      tabIndex={-1}
    >
      <div className="flex items-center justify-between gap-2">
        <h2 id={`${id}-h`}>{t(mx.shareTitle)}</h2>
        <button type="button" className="btn ic ghost" style={{ width: 32, minHeight: 32 }} aria-label={t(common.close)} onClick={onClose}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
            <path d="m4 4 8 8M12 4l-8 8" />
          </svg>
        </button>
      </div>
      <div className="sp-l">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true" style={{ flex: 'none' }}>
          <circle cx="8" cy="8" r="6.2" />
          <path d="M1.8 8h12.4M8 1.8c1.8 1.7 2.7 3.8 2.7 6.2S9.8 12.5 8 14.2C6.2 12.5 5.3 10.4 5.3 8S6.2 3.5 8 1.8z" />
        </svg>
        <span>{t(mx.shareLead)}</span>
      </div>
      {link ? (
        <>
          <div className="row">
            <input ref={input} className="in mono" readOnly value={link.url} aria-label={t(mx.reviewLink)} onFocus={(e) => e.target.select()} />
            <button type="button" className="btn sm pri" onClick={() => void copy()}>
              {t(mx.copyLink)}
            </button>
          </div>
          <span className="k">{t(mx.shareNote)}</span>
          {link.version < current ? (
            <div className="row" role="status">
              <span className="k" style={{ flex: '1 1 180px' }}>
                {t(mx.olderVersion, { linkVersion: link.version, version: current })}
              </span>
              <button type="button" className="btn sm" disabled={busy} onClick={() => void create()}>
                {t(mx.shareLatest, { version: current })}
              </button>
            </div>
          ) : null}
          {savedFirst ? <span className="k">{t(mx.savedFirst)}</span> : null}
          <div className="row sp-foot">
            <span className="k">{t(mx.createdBy, { name: link.createdBy, when, version: link.version })}</span>
            <button type="button" className="btn sm danger" disabled={busy} onClick={() => void revoke()}>
              {t(mx.revoke)}
            </button>
          </div>
        </>
      ) : busy ? (
        <span className="k" role="status" aria-busy="true">
          {t(mx.creating)}
        </span>
      ) : revoked ? (
        <div className="row" role="status">
          <span style={{ flex: '1 1 180px' }}>{t(mx.revoked)}</span>
          <button type="button" className="btn sm" onClick={() => void create()}>
            {t(mx.newLink)}
          </button>
        </div>
      ) : null}
      {error ? (
        <p className="m-0 text-bad-fg" role="alert">
          {t(mx.createFailed)}: {error}
        </p>
      ) : null}
    </div>
  )
}

/** "Share for review", for people who edit campaigns: a popover under the button. */
export function ShareReviewButton({ disabled }: { disabled?: boolean }) {
  const c = useComposer()
  const t = useT()
  const id = useId()
  const [open, setOpen] = useState(false)
  const button = useRef<HTMLButtonElement>(null)
  const wrap = useRef<HTMLDivElement>(null)
  // A click elsewhere closes it, as does Escape (focus goes back to the button).
  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (wrap.current && !wrap.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [open])
  if (c.kind === 'transactional' || !c.campaign.access.canEdit) return null
  const close = () => {
    setOpen(false)
    button.current?.focus()
  }
  return (
    <div ref={wrap} className="relative">
      <button ref={button} type="button" className="btn" disabled={disabled} aria-expanded={open} aria-controls={open ? `${id}-share` : undefined} onClick={() => setOpen((o) => !o)}>
        <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true">
          <path d="M6.5 9.5a3 3 0 0 0 4.2 0l2.3-2.3a3 3 0 0 0-4.2-4.2l-.9.9M9.5 6.5a3 3 0 0 0-4.2 0L3 8.8A3 3 0 0 0 7.2 13l.9-.9" />
        </svg>
        {t(mx.share)}
      </button>
      {open ? <SharePopover id={`${id}-share`} onClose={close} /> : null}
    </div>
  )
}
