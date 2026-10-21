import type { ReactNode } from 'react'
import { T, useT } from '@/i18n'
import { count, pct } from '@/lib/format'
import type { PushPromptConfig } from './common'
import { m } from './prompt.messages'
import './prompt.css'

/** How the site looks in the preview: its name, mark, accent, and a post to show behind the prompt. */
export interface SiteLook {
  host: string
  name: string
  initial: string
  logoUrl?: string
  iconUrl?: string
  accent: string
  publication: boolean
  post?: { title: string; meta: string; paragraphs: string[] }
}

export function Mark({ look, size, radius = 7, font = 13 }: { look: SiteLook; size: number; radius?: number; font?: number }) {
  const src = look.iconUrl ?? look.logoUrl
  return (
    <span className="pp-mark" style={{ width: size, height: size, borderRadius: radius, fontSize: font }} aria-hidden="true">
      {src ? <img src={src} alt="" /> : look.initial}
    </span>
  )
}

/** A page of the site in a browser frame, with whatever sits over it. */
export function SiteFrame({
  look,
  device,
  path,
  height,
  children,
}: {
  look: SiteLook
  device: 'desktop' | 'phone' | 'iphone'
  path: string
  height?: number
  children?: ReactNode
}) {
  const t = useT()
  const desktop = device === 'desktop'
  const url = `${look.host}${path}`
  return (
    <div className={desktop ? 'pp-site' : `pp-site ${device}`}>
      <div className="pp-chrome">
        {desktop ? (
          <span className="dots" aria-hidden="true">
            <span />
            <span />
            <span />
          </span>
        ) : null}
        <span className="url">{url}</span>
      </div>
      <div className="pp-page" style={height ? { height } : undefined}>
        <div className="pp-head">
          <span className="logo" aria-hidden="true">
            {look.logoUrl ? <img src={look.logoUrl} alt="" /> : look.initial}
          </span>
          <span className="name">{look.name}</span>
          {desktop ? (
            <span className="pp-nav" aria-hidden="true">
              {look.publication ? (
                <>
                  <span>{t(m.sitePosts)}</span>
                  <span>{t(m.siteAbout)}</span>
                  <span style={{ color: look.accent, fontWeight: 600 }}>{t(m.siteSubscribe)}</span>
                </>
              ) : (
                <>
                  <span className="sk inline-block h-2 w-12 rounded bg-line-2" />
                  <span className="sk inline-block h-2 w-12 rounded bg-line-2" />
                  <span className="sk inline-block h-2 w-12 rounded bg-line-2" />
                </>
              )}
            </span>
          ) : null}
        </div>
        <div className="pp-post">
          {look.post ? (
            <>
              <span className="meta">{look.post.meta}</span>
              <h3>{look.post.title}</h3>
              {look.post.paragraphs.map((p, i) => (
                <p key={i}>{p}</p>
              ))}
            </>
          ) : (
            <>
              <span className="sk" style={{ width: '40%' }} />
              <span className="sk" style={{ width: '75%', height: 18 }} />
              <span className="sk" style={{ width: '95%' }} />
              <span className="sk" style={{ width: '88%' }} />
              <span className="sk" style={{ width: '92%' }} />
              <span className="sk" style={{ width: '60%' }} />
            </>
          )}
        </div>
        {children}
      </div>
    </div>
  )
}

function Buttons({ cfg, look, two }: { cfg: PushPromptConfig; look: SiteLook; two?: boolean }) {
  return (
    <span className={two ? 'two' : 'flex flex-none gap-2'}>
      <span className="pp-btn">{cfg.notNow}</span>
      <span className="pp-btn pri" style={{ background: look.accent }}>
        {cfg.allow}
      </span>
    </span>
  )
}

/** The prompt over the page, as its style draws it. */
export function PromptOver({ cfg, look, device }: { cfg: PushPromptConfig; look: SiteLook; device: 'desktop' | 'phone' }) {
  const text = (
    <span className="tx">
      <b>{cfg.title}</b>
      <span>{cfg.message}</span>
    </span>
  )
  if (cfg.style === 'bell') {
    const left = cfg.corner === 'bottom_left'
    return (
      <>
        <div className={left ? 'pp-pop left' : 'pp-pop'}>
          <span className="flex items-start gap-2.5">
            <Mark look={look} size={30} />
            {text}
          </span>
          <span className="acts">
            <span className="pp-btn h-8">{cfg.notNow}</span>
            <span className="pp-btn pri h-8" style={{ background: look.accent }}>
              {cfg.allow}
            </span>
          </span>
        </div>
        <span className={left ? 'pp-bell left' : 'pp-bell'} style={{ background: look.accent }} aria-hidden="true">
          <svg width="20" height="20" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
            <path d="M4 11V7a4 4 0 0 1 8 0v4l1.5 1.5h-11zM6.5 14a1.5 1.5 0 0 0 3 0" />
          </svg>
        </span>
      </>
    )
  }
  if (cfg.style === 'dialog') {
    return (
      <div className="pp-dim">
        <div className="pp-dialog">
          <span className="flex items-start gap-2.5">
            <Mark look={look} size={30} />
            {text}
          </span>
          <span className="acts">
            <span className="pp-btn">{cfg.notNow}</span>
            <span className="pp-btn pri" style={{ background: look.accent }}>
              {cfg.allow}
            </span>
          </span>
        </div>
      </div>
    )
  }
  if (device === 'phone') {
    return (
      <div className="pp-sheet">
        <span className="flex items-start gap-2.5">
          <Mark look={look} size={28} font={12} />
          {text}
        </span>
        <Buttons cfg={cfg} look={look} two />
      </div>
    )
  }
  return (
    <div className="pp-bar-p">
      <Mark look={look} size={30} />
      {text}
      <Buttons cfg={cfg} look={look} />
    </div>
  )
}

/** iPhone and iPad, not on the Home Screen: the steps to add the site, instead of Allow. */
export function IosSheet({ cfg, look }: { cfg: PushPromptConfig; look: SiteLook }) {
  const t = useT()
  const b = (c: ReactNode[]) => <b>{c}</b>
  return (
    <div className="pp-sheet" style={{ gap: 8, padding: '16px 14px 14px' }}>
      <span className="flex items-start gap-2.5">
        <Mark look={look} size={28} font={12} />
        <b className="leading-[1.4]">{t(m.iosTitle, { name: look.name })}</b>
      </span>
      <div className="pp-step">
        <span className="ic" aria-hidden="true">
          <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
            <path d="M8 2v8M5 5l3-3 3 3M3.5 8.5V13h9V8.5" />
          </svg>
        </span>
        <span>
          <T msg={m.iosStep1} values={{ b }} />
        </span>
      </div>
      <div className="pp-step">
        <span className="ic" aria-hidden="true">
          <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
            <rect x="2.5" y="2.5" width="11" height="11" rx="2" />
            <path d="M8 5.5v5M5.5 8h5" />
          </svg>
        </span>
        <span>
          <T msg={m.iosStep2} values={{ b }} />
        </span>
      </div>
      <div className="pp-step">
        <span className="ic" aria-hidden="true">
          <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
            <path d="M4 11V7a4 4 0 0 1 8 0v4l1.5 1.5h-11zM6.5 14a1.5 1.5 0 0 0 3 0" />
          </svg>
        </span>
        <span>{t(m.iosStep3, { name: look.name })}</span>
      </div>
      <span className="pp-btn" style={{ height: 38 }}>
        {cfg.notNow}
      </span>
    </div>
  )
}

/** After Allow: Chrome's and Safari's own permission prompts. */
export function BrowserPrompts({ look, stats }: { look: SiteLook; stats: { granted: number; allowed: number; grantedPct: number } }) {
  const t = useT()
  return (
    <div className="card flex flex-col gap-3.5 px-[18px] py-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2.5">
        <h2 className="h2">{t(m.browserTitle)}</h2>
        <span className="k">{t(m.browserHint)}</span>
      </div>
      <div className="flex flex-wrap items-start gap-[18px]">
        <figure className="px-pvf">
          <figcaption>
            <b>{t(m.chrome)}</b>
            <span className="k">{t(m.chromeSub)}</span>
          </figcaption>
          <div className="px-pvstage">
            <div className="pp-bp">
              <div className="flex items-center justify-between">
                <b>{t(m.chromeWants, { host: look.host })}</b>
                <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="#4B5563" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
                  <path d="M4 4l8 8M12 4l-8 8" />
                </svg>
              </div>
              <div className="flex items-center gap-2.5 text-ink-nav">
                <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M4 11V7a4 4 0 0 1 8 0v4l1.5 1.5h-11zM6.5 14a1.5 1.5 0 0 0 3 0" />
                </svg>
                {t(m.chromeShow)}
              </div>
              <div className="mt-1 flex justify-end gap-2">
                <span className="pill">{t(m.block)}</span>
                <span className="pill pri">{t(m.allow)}</span>
              </div>
            </div>
          </div>
          <span className="cut">
            {stats.allowed ? t(m.grantedLine, { granted: count(stats.granted), allowed: count(stats.allowed), pct: pct(stats.grantedPct) }) : t(m.grantedNone)}
          </span>
        </figure>
        <figure className="px-pvf">
          <figcaption>
            <b>{t(m.safariMac)}</b>
            <span className="k">{t(m.safariSub)}</span>
          </figcaption>
          <div className="px-pvstage">
            <div className="pp-bp safari">
              <Mark look={look} size={40} radius={9} font={17} />
              <b className="text-[13.5px]">{t(m.safariWants, { host: look.host })}</b>
              <span className="text-[12px] leading-[1.45] text-ink-2">{t(m.safariText)}</span>
              <div className="two">
                <span>{t(m.dontAllow)}</span>
                <span className="pri">{t(m.allow)}</span>
              </div>
            </div>
          </div>
          <span className="cut">{t(m.safariNote)}</span>
        </figure>
      </div>
    </div>
  )
}
