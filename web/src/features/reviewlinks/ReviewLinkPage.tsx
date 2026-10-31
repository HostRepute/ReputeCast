import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { LogoMark } from '@/components/icons/Icon'
import { FullPageState } from '@/components/shell/SystemScreens'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, setDisplayTimeZone, shortDate } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { mr } from './messages'
import './reviewlinks.css'

type Thread = Schemas['ComposerThread']

/** Keeps the page and its token out of search engines and other sites' logs. */
function useNoIndex() {
  useEffect(() => {
    const metas = [
      ['robots', 'noindex, nofollow'],
      ['referrer', 'no-referrer'],
    ].map(([name, content]) => {
      const meta = document.createElement('meta')
      meta.name = name!
      meta.content = content!
      document.head.appendChild(meta)
      return meta
    })
    return () => metas.forEach((m) => m.remove())
  }, [])
}

const nameKey = 'rc-review-name'

function savedName(): string {
  try {
    return window.localStorage.getItem(nameKey) ?? ''
  } catch {
    return ''
  }
}

function keepName(name: string) {
  try {
    window.localStorage.setItem(nameKey, name)
  } catch {
    // A private window: the name is asked again next time.
  }
}

/** The email, as wide as it is sent (600 px), scaled down on narrow screens. */
function EmailFrame({ html, title }: { html: string; title: string }) {
  const box = useRef<HTMLDivElement>(null)
  const frame = useRef<HTMLIFrameElement>(null)
  const [scale, setScale] = useState(1)
  const [height, setHeight] = useState(480)
  const width = 600
  const srcDoc = useMemo(() => {
    // Links open in a new tab, without telling the site where they came from.
    const base = '<base target="_blank"><meta name="referrer" content="no-referrer">'
    return /<head[^>]*>/i.test(html) ? html.replace(/<head([^>]*)>/i, `<head$1>${base}`) : `${base}${html}`
  }, [html])
  useEffect(() => {
    const el = box.current
    if (!el) return
    const measure = () => setScale(Math.min(1, el.clientWidth / width))
    measure()
    if (typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  const onLoad = () => {
    const doc = frame.current?.contentDocument
    if (!doc) return
    const fit = () => setHeight(doc.documentElement.scrollHeight)
    fit()
    doc.querySelectorAll('img').forEach((img) => img.complete || img.addEventListener('load', fit, { once: true }))
  }
  return (
    <div ref={box} className="rv-mail" style={{ height: height * scale }}>
      <iframe
        ref={frame}
        title={title}
        sandbox="allow-same-origin allow-popups allow-popups-to-escape-sandbox"
        srcDoc={srcDoc}
        onLoad={onLoad}
        style={{ width, height, transform: `scale(${scale})` }}
      />
    </div>
  )
}

function CommentForm({ token, threadId, onDone, onCancel }: { token: string; threadId?: string; onDone: () => void; onCancel?: () => void }) {
  const t = useT()
  const id = useId()
  const qc = useQueryClient()
  const [name, setName] = useState(savedName)
  const [body, setBody] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const post = async () => {
    setBusy(true)
    setError('')
    try {
      await http.POST('/api/v1/review/{token}/comments', { params: { path: { token } }, body: { name: name.trim(), body, threadId } })
      keepName(name.trim())
      setBody('')
      await qc.invalidateQueries({ queryKey: ['review', token] })
      onDone()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <form
      className="rv-form"
      onSubmit={(e) => {
        e.preventDefault()
        void post()
      }}
    >
      <label className="lbl" htmlFor={`${id}-name`}>
        {t(mr.yourName)}
      </label>
      <input id={`${id}-name`} className="in" value={name} maxLength={80} autoComplete="name" placeholder={t(mr.namePlaceholder)} onChange={(e) => setName(e.target.value)} />
      <label className="lbl" htmlFor={`${id}-body`}>
        {threadId ? t(mr.reply) : t(mr.comment)}
      </label>
      <textarea id={`${id}-body`} className="in" rows={3} maxLength={4000} value={body} placeholder={t(mr.commentPlaceholder)} onChange={(e) => setBody(e.target.value)} />
      {error ? (
        <p className="m-0 text-bad-fg" role="alert">
          {t(mr.postFailed)}: {error}
        </p>
      ) : null}
      <div className="flex flex-wrap justify-end gap-2">
        {onCancel ? (
          <button type="button" className="btn" onClick={onCancel}>
            {t(mr.cancel)}
          </button>
        ) : null}
        <button type="submit" className="btn pri" disabled={busy || !name.trim() || !body.trim()} aria-busy={busy || undefined}>
          {busy ? t(mr.posting) : threadId ? t(mr.reply) : t(mr.post)}
        </button>
      </div>
    </form>
  )
}

function ThreadView({ thread, token }: { thread: Thread; token: string }) {
  const t = useT()
  const [replying, setReplying] = useState(false)
  const first = thread.comments[0]!
  return (
    <article className="rv-thread" aria-label={t(mr.threadBy, { name: first.author })}>
      {thread.comments.map((c, i) => (
        <div key={c.id} className={i ? 'rv-c reply' : 'rv-c'}>
          <div className="rv-ch">
            <Avatar name={c.author} color={c.reviewer ? 'neutral' : c.authorColor} size={24} />
            <b>{c.author}</b>
            {c.reviewer ? <Badge tone="neutral">{t(mr.reviewer)}</Badge> : null}
            <span className="k">
              {shortDate(c.createdAt)} {clock(c.createdAt)}
            </span>
          </div>
          <p className="rv-tx">{c.body}</p>
        </div>
      ))}
      {thread.resolvedAt ? (
        <Badge tone="ok" className="self-start">
          {t(mr.resolved)}
        </Badge>
      ) : replying ? (
        <CommentForm token={token} threadId={thread.id} onDone={() => setReplying(false)} onCancel={() => setReplying(false)} />
      ) : (
        <button type="button" className="btn sm self-start" onClick={() => setReplying(true)}>
          {t(mr.reply)}
        </button>
      )}
    </article>
  )
}

/**
 * A review link (`/review/{token}`, CO-10): the email of one version,
 * personalized for a sample reader, and comments. No app shell, no
 * sign-in, nothing about the audience or the workspace's settings.
 */
export function ReviewLinkPage({ token }: { token: string }) {
  const t = useT()
  useNoIndex()
  const q = useQuery({
    queryKey: ['review', token],
    queryFn: async () => {
      const { data } = await http.GET('/api/v1/review/{token}', { params: { path: { token } } })
      return data!
    },
    retry: false,
    refetchOnWindowFocus: false,
  })
  // Times read in the browser's timezone for someone outside the workspace.
  useEffect(() => setDisplayTimeZone(undefined), [])
  const r = q.data
  useDocumentTitle(r ? t(mr.docTitle, { campaign: r.campaign, workspace: r.workspace }) : t(mr.docTitleLoading))
  const gone = q.error instanceof ApiError && q.error.status === 404
  const limited = q.error instanceof ApiError && q.error.status === 429
  return (
    <div className="rc-app min-h-dvh bg-ground">
      <a href="#main" className="skip">
        {t(mr.skip)}
      </a>
      <header className="top">
        <div className="mx-auto flex w-full max-w-[1152px] flex-wrap items-center gap-3">
          <span className="flex items-center gap-2.5 text-ink" aria-label="ReputeCast">
            <LogoMark size={26} />
            <span className="text-[15px] font-bold tracking-[-0.01em]">ReputeCast</span>
          </span>
          <span className="k">{t(mr.badge)}</span>
          {r ? <span className="k ml-auto">{t(mr.sharedBy, { workspace: r.workspace })}</span> : null}
        </div>
      </header>
      <main className="content mx-auto max-w-[1152px]" id="main" tabIndex={-1}>
        {q.isPending ? (
          <div className="rv-grid" aria-busy="true">
            <Skeleton h={480} r={10} />
            <Skeleton h={240} r={10} />
          </div>
        ) : gone || limited || q.isError ? (
          <div className="mx-auto w-full max-w-[640px] py-10">
            <FullPageState
              icon={gone ? 'searchLarge' : 'warningTriangle'}
              tone={gone ? 'neutral' : 'warn'}
              badge={gone ? t(mr.goneBadge) : limited ? t(mr.limitedBadge) : t(mr.errorBadge)}
              title={gone ? t(mr.goneTitle) : limited ? t(mr.limitedTitle) : t(mr.errorTitle)}
              actions={!gone ? <Button onClick={() => void q.refetch()}>{t(common.tryAgain)}</Button> : undefined}
            >
              {gone ? t(mr.goneBody) : limited ? t(mr.limitedBody) : `${errorMessage(q.error)} ${t(mr.errorBody)}`}
            </FullPageState>
          </div>
        ) : r ? (
          <>
            <div className="flex flex-col gap-1">
              <h1 className="m-0 text-[20px] font-bold tracking-[-0.01em]">{r.campaign}</h1>
              <span className="k">{t(mr.caption, { version: r.version, date: shortDate(r.expiresAt) })}</span>
            </div>
            <div className="rv-grid">
              <section className="flex min-w-0 flex-col gap-3" aria-label={t(mr.frame)}>
                <dl className="rv-inbox">
                  <dt>{t(mr.from)}</dt>
                  <dd>{r.fromName}</dd>
                  <dt>{t(mr.subject)}</dt>
                  <dd className="font-semibold">{r.subject}</dd>
                  {r.preheader ? (
                    <>
                      <dt>{t(mr.preheader)}</dt>
                      <dd>{r.preheader}</dd>
                    </>
                  ) : null}
                </dl>
                <EmailFrame html={r.html} title={t(mr.frame)} />
                {r.sample ? <p className="k m-0">{t(mr.sample, { name: r.sample })}</p> : null}
              </section>
              <section className="card rv-side" aria-labelledby="rv-comments">
                <h2 id="rv-comments" className="m-0 text-[15px] font-semibold">
                  {t(mr.comments)}
                </h2>
                <p className="k m-0">{t(mr.commentsLead, { workspace: r.workspace })}</p>
                <CommentForm token={token} onDone={() => {}} />
                {r.threads.length ? r.threads.map((th) => <ThreadView key={th.id} thread={th} token={token} />) : <p className="k m-0">{t(mr.noComments)}</p>}
              </section>
            </div>
            <p className="k m-0">{t(mr.madeWith)}</p>
          </>
        ) : null}
      </main>
    </div>
  )
}
