import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { useT } from '@/i18n'
import { andList, dayMonthDate, shortDate, clock } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { WS, type Post, type PostInput, type WebCheck } from '../api'
import { ImagePickerDialog } from '../ImagePicker'
import { isoToZonedLocal, visibilityIcon, zonedToISO } from '../postFormat'
import { mc } from './messages'
import './composer.css'

/** The settings a post's Web tab edits, from the post. */
export function inputOfPost(p: Post): PostInput {
  return {
    bylineName: p.byline.userId ? undefined : p.byline.name,
    bylineUserId: p.byline.userId || undefined,
    coverAlt: p.coverAlt,
    coverUrl: p.coverUrl,
    excerpt: p.excerpt,
    featured: p.featured,
    neverFree: p.neverFree,
    publishAt: p.publishAt,
    publishMode: p.publishMode,
    publishTz: p.publishTz,
    seo: { ...p.seo },
    slug: p.slug,
    tags: p.tags.map((x) => x.id),
    tiers: p.tiers.map((x) => x.id),
    title: p.title,
    visibility: p.visibility,
    webOn: p.webOn,
  }
}

export type SlugTaken = { path: string; suggestion: string; owner?: WebCheck['taken'] }

/**
 * A post's Web tab settings: edited here, saved 0.7 s after the last change
 * (PUT the post, or the campaign's web settings), with the post as the
 * server returns it (its checks, its publish state). Before the post has
 * loaded (the campaign composer asks for it after it opens) it is null.
 */
export function useWebSettingsOf({ post, save }: { post: Post | undefined; save: (input: PostInput) => Promise<Post> }) {
  const [draft, setDraft] = useState<PostInput | null>(() => (post ? inputOfPost(post) : null))
  const [current, setCurrent] = useState<Post | null>(post ?? null)
  // The post arrived after the first render: take it as it is.
  if (!current && post) {
    setCurrent(post)
    setDraft(inputOfPost(post))
  }
  const [status, setStatus] = useState<'saved' | 'dirty' | 'saving' | 'error'>('saved')
  const [error, setError] = useState<string>()
  const [taken, setTaken] = useState<SlugTaken>()
  const pending = useRef<PostInput | null>(null)
  const saveRef = useRef(save)
  useLayoutEffect(() => {
    saveRef.current = save
  })
  const flush = useCallback(async () => {
    const next = pending.current
    if (!next) return true
    pending.current = null
    setStatus('saving')
    try {
      const p = await saveRef.current(next)
      setCurrent(p)
      setTaken(undefined)
      setError(undefined)
      setStatus(pending.current ? 'dirty' : 'saved')
      return true
    } catch (e) {
      if (e instanceof ApiError && e.code === 'slug_taken') {
        const meta = (e.problem.meta ?? {}) as { suggestion?: string }
        setTaken({ path: `/posts/${next.slug}`, suggestion: meta.suggestion ?? '' })
        setError(undefined)
      } else {
        setError(errorMessage(e))
      }
      setStatus('error')
      return false
    }
  }, [])
  useEffect(() => {
    if (status !== 'dirty') return
    const h = window.setTimeout(() => void flush(), 700)
    return () => window.clearTimeout(h)
  }, [status, draft, flush])
  const set = useCallback((patch: Partial<PostInput>) => {
    setDraft((d) => {
      if (!d) return d
      const next = { ...d, ...patch }
      pending.current = next
      return next
    })
    setStatus('dirty')
  }, [])
  /** Takes the server's post (after a content save, a publish) without losing unsaved settings. */
  const refresh = useCallback((p: Post) => {
    setCurrent(p)
    if (!pending.current) setDraft(inputOfPost(p))
  }, [])
  return useMemo(() => (draft && current ? { draft, set, post: current, status, error, taken, flush, refresh } : null), [draft, current, set, status, error, taken, flush, refresh])
}

/** useWebSettingsOf for a post that is there from the start (post mode). */
export function useWebSettings({ post, save }: { post: Post; save: (input: PostInput) => Promise<Post> }) {
  return useWebSettingsOf({ post, save })!
}

export type WebSettings = NonNullable<ReturnType<typeof useWebSettingsOf>>

/** One section of the Web tab: a title and its fields. */
function Sec({ title, children, id, aside }: { title: ReactNode; children?: ReactNode; id?: string; aside?: ReactNode }) {
  return (
    <section className="wtab-sec" aria-labelledby={id}>
      <div className="wtab-h">
        <span id={id}>{title}</span>
        {aside}
      </div>
      {children}
    </section>
  )
}

/** A radio or checkbox with its title and a line under it (the label's text stays two levels deep). */
function Choice({ type = 'radio', name, checked, disabled, onChange, title, sub, dim }: { type?: 'radio' | 'checkbox'; name?: string; checked: boolean; disabled?: boolean; onChange: (on: boolean) => void; title: string; sub?: string; dim?: boolean }) {
  return (
    <label className={`wtab-radio${dim ? ' dim' : ''}`}>
      <input type={type} name={name} checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
      <span className="tt">
        {title}
        {sub ? <span className="k">{sub}</span> : null}
      </span>
    </label>
  )
}

/** "1st", "2nd", "3rd": a block's place, in words. */
function nth(n: number): string {
  const words = ['first', 'second', 'third', 'fourth', 'fifth', 'sixth', 'seventh', 'eighth', 'ninth', 'tenth']
  return words[n - 1] ?? `${n}th`
}

export interface WebPanelProps {
  ws: WebSettings
  mode: 'post' | 'campaign'
  /** The site's host ("news.acme.io"). */
  host: string
  canEdit: boolean
  /** Campaign mode: the campaign's lists, its send time and whether it went out. */
  campaign?: { lists: string[]; scheduledAt?: string; sentAt?: string; status: string; followsAudience: boolean; heroAlt?: string }
  /** The free preview marker on the canvas (post mode): where it is, and moving it. */
  marker?: { at: number | null; add: () => void; move: () => void; remove: () => void; canMove: boolean }
  /** Shows the blocks left out of the web on the canvas (campaign mode). */
  onShowHidden?: () => void
}

/**
 * The Web tab (ComposerPublishToWeb): address, title and excerpt, cover,
 * tags and byline, visibility, free preview, when, feature on home, SEO and
 * social, and the web checks. Campaign mode adds "Publish to news.acme.io".
 */
export function WebPanel({ ws, mode, host, canEdit, campaign, marker, onShowHidden }: WebPanelProps) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const navigate = useNavigate()
  const d = ws.draft
  const p = ws.post
  const ro = !canEdit
  const tags = $api.useQuery('get', `${WS}/tags`, { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const access = $api.useQuery('get', `${WS}/access`, { params: { path: { workspace: slug } } }, { staleTime: 60_000, enabled: useCan('website:view') })
  const [picking, setPicking] = useState(false)
  const [seoOpen, setSeoOpen] = useState(false)
  const tiers = access.data?.tiers ?? []
  const paidOK = Boolean(access.data?.payments.connected && tiers.length)
  const tz = d.publishTz || me.timezone || 'UTC'
  const on = mode === 'post' || d.webOn
  const tagList = tags.data ?? []
  const chosenTags = d.tags.map((id) => tagList.find((x) => x.id === id) ?? p.tags.find((x) => x.id === id)).filter(Boolean) as { id: string; name: string }[]
  const sent = campaign?.status === 'sent' || Boolean(campaign?.sentAt)
  return (
    <div className="wtab">
      {mode === 'campaign' ? (
        <section className="wtab-sec">
          <div className="flex items-start justify-between gap-3">
            <div className="flex min-w-0 flex-col gap-1">
              <b id="wt-on">{t(mc.publishTo, { host })}</b>
              <span className="k">{d.webOn ? (sent ? t(mc.publishToTextSent) : t(mc.publishToText)) : t(mc.publishToOff)}</span>
            </div>
            <input type="checkbox" role="switch" className="tg" aria-labelledby="wt-on" checked={d.webOn} disabled={ro} onChange={(e) => ws.set({ webOn: e.target.checked })} />
          </div>
        </section>
      ) : null}
      {on ? (
        <>
          <Sec title={t(mc.address)} id="wt-addr">
            <div className={`wtab-slug${ws.taken ? ' bad' : ''}`}>
              <span className="pre">{p.base}</span>
              <input
                aria-labelledby="wt-addr"
                aria-invalid={ws.taken ? true : undefined}
                aria-describedby={ws.taken ? 'wt-addr-err' : 'wt-addr-hint'}
                value={d.slug}
                disabled={ro}
                spellCheck={false}
                onChange={(e) => ws.set({ slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '-') })}
              />
            </div>
            {ws.taken ? (
              <>
                <span className="err" id="wt-addr-err" role="alert">
                  {t(mc.slugTaken, { path: ws.taken.path, owner: t(mc.slugTakenPost, { year: String(new Date().getFullYear() - 1) }), suggestion: ws.taken.suggestion })}
                </span>
                {ws.taken.suggestion ? (
                  <div className="flex flex-wrap gap-1.5">
                    <button type="button" className="btn sm" onClick={() => ws.set({ slug: ws.taken!.suggestion })}>
                      {t(mc.useSlug, { slug: ws.taken.suggestion })}
                    </button>
                  </div>
                ) : null}
              </>
            ) : (
              <>
                <span className="mono k wtab-url">
                  {p.base}
                  {d.slug}
                </span>
                {mode === 'post' ? (
                  <span className="k" id="wt-addr-hint">
                    {t(mc.addressHint)}
                  </span>
                ) : null}
              </>
            )}
          </Sec>
          <Sec title={t(mc.titleExcerpt)}>
            <label className="wtab-f">
              <span>{t(mc.title)}</span>
              <input className="in" value={d.title} maxLength={200} disabled={ro} onChange={(e) => ws.set({ title: e.target.value, useSubject: false })} />
            </label>
            {mode === 'campaign' ? (
              <span className="k">
                {p.titleEdited ? t(mc.editedSubject) : t(mc.fromSubject)}{' '}
                {p.titleEdited && !ro ? (
                  <button type="button" className="prose-link border-0 bg-transparent p-0 text-[12px]" onClick={() => ws.set({ useSubject: true, title: '' })}>
                    {t(mc.useSubject)}
                  </button>
                ) : null}
              </span>
            ) : null}
            <label className="wtab-f">
              <span>{t(mc.excerpt)}</span>
              <textarea className="in" rows={3} value={d.excerpt} maxLength={200} disabled={ro} onChange={(e) => ws.set({ excerpt: e.target.value, usePreheader: false })} />
            </label>
            {mode === 'campaign' ? (
              <span className="k">
                {p.excerptEdited ? t(mc.editedPreheader) : t(mc.fromPreheader)}{' '}
                {p.excerptEdited && !ro ? (
                  <button type="button" className="prose-link border-0 bg-transparent p-0 text-[12px]" onClick={() => ws.set({ usePreheader: true, excerpt: '' })}>
                    {t(mc.usePreheader)}
                  </button>
                ) : null}
              </span>
            ) : (
              <span className="k">{t(mc.excerptCount, { count: d.excerpt.length })}</span>
            )}
          </Sec>
          <Sec title={t(mc.cover)}>
            {mode === 'campaign' && !d.coverUrl ? (
              <span className="k">{campaign?.heroAlt ? t(mc.coverFromHero, { alt: campaign.heroAlt }) : t(mc.coverFromHeroNoAlt)}</span>
            ) : null}
            <div className="flex items-start gap-2.5">
              <button type="button" className="wtab-cover" disabled={ro} onClick={() => setPicking(true)} aria-label={d.coverUrl ? t(mc.coverReplace) : t(mc.coverChoose)}>
                {d.coverUrl ? <img src={d.coverUrl} alt="" /> : <Icon name="image" size={18} />}
              </button>
              {d.coverUrl ? (
                <label className="wtab-f flex-1">
                  <span>{t(mc.altText)}</span>
                  <textarea className="in" rows={3} value={d.coverAlt} maxLength={300} placeholder={t(mc.altPlaceholder)} disabled={ro} onChange={(e) => ws.set({ coverAlt: e.target.value })} />
                </label>
              ) : (
                <span className="k flex-1">{t(mc.coverNone)}</span>
              )}
            </div>
            {d.coverUrl && !ro ? (
              <span>
                <button type="button" className="btn sm" onClick={() => ws.set({ coverUrl: '', coverAlt: '' })}>
                  {t(mc.coverRemove)}
                </button>
              </span>
            ) : null}
          </Sec>
          <Sec title={t(mc.tagsByline)}>
            <div className="flex flex-wrap items-center gap-1.5">
              {chosenTags.map((x) => (
                <span key={x.id} className="wtab-tag">
                  {x.name}
                  {ro ? null : (
                    <button type="button" aria-label={t(mc.removeTag, { name: x.name })} onClick={() => ws.set({ tags: d.tags.filter((id) => id !== x.id) })}>
                      <Icon name="close" size={11} />
                    </button>
                  )}
                </span>
              ))}
              {ro || d.tags.length >= 5 ? null : (
                <Menu
                  align="start"
                  label={t(mc.addTagLabel)}
                  trigger={
                    <button type="button" className="wtab-tag add" aria-label={t(mc.addTagLabel)}>
                      <Icon name="plus" size={11} />
                      {t(mc.addTag)}
                    </button>
                  }
                >
                  {tagList
                    .filter((x) => !d.tags.includes(x.id))
                    .map((x) => (
                      <MenuItem key={x.id} onSelect={() => ws.set({ tags: [...d.tags, x.id] })}>
                        {x.name}
                      </MenuItem>
                    ))}
                </Menu>
              )}
            </div>
            <label className="wtab-f">
              <span>{t(mc.byline)}</span>
              <select
                className="in"
                value={d.bylineUserId ?? ''}
                disabled={ro}
                onChange={(e) => (e.target.value ? ws.set({ bylineUserId: e.target.value, bylineName: undefined }) : ws.set({ bylineUserId: undefined, bylineName: t(mc.bylineTeam, { publisher: me.workspace.name }) }))}
              >
                {p.bylines.map((b) => (
                  <option key={b.userId} value={b.userId}>
                    {b.name}
                  </option>
                ))}
                <option value="">{d.bylineName || t(mc.bylineTeam, { publisher: me.workspace.name })}</option>
              </select>
            </label>
          </Sec>
          <Sec title={t(mc.visibility)} id="wt-vis">
            <div className="seg self-start" role="group" aria-labelledby="wt-vis">
              {(['public', 'members', 'paid'] as const).map((v) => (
                <button key={v} type="button" aria-pressed={d.visibility === v} disabled={ro || (v === 'paid' && !paidOK && d.visibility !== 'paid')} onClick={() => ws.set({ visibility: v })}>
                  <Icon name={visibilityIcon(v)} size={13} />
                  {v === 'public' ? t(mc.visPublic) : v === 'members' ? t(mc.visMembers) : t(mc.web === mc.web ? mc.visPaidAll : mc.visPaidAll).split(' · ')[0]}
                </button>
              ))}
            </div>
            {!paidOK && d.visibility !== 'paid' ? <span className="k">{t(mc.paidNeedsStripe)}</span> : null}
            {mode === 'campaign' && campaign?.followsAudience ? (
              <span className="k">{d.visibility === 'paid' ? t(mc.followsAudiencePaid) : t(mc.followsAudience, { lists: andList(campaign.lists), free: campaign.lists.length })}</span>
            ) : null}
            {d.visibility === 'paid' ? (
              <div className="flex flex-col gap-1.5">
                <Choice name="wt-tiers" checked={d.tiers.length === 0} disabled={ro} onChange={() => ws.set({ tiers: [] })} title={t(mc.everyTier)} sub={tiers.length ? andList(tiers.map((x) => x.name)) : undefined} />
                <Choice name="wt-tiers" checked={d.tiers.length > 0} disabled={ro || !tiers.length} onChange={() => ws.set({ tiers: tiers.slice(0, 1).map((x) => x.id) })} title={t(mc.chosenTiers)} />
                {d.tiers.length > 0 ? (
                  <div className="flex flex-col gap-1 pl-6">
                    {tiers.map((x) => (
                      <label key={x.id} className="flex items-center gap-2 text-[13px]">
                        <input
                          type="checkbox"
                          checked={d.tiers.includes(x.id)}
                          disabled={ro}
                          onChange={(e) => {
                            const next = e.target.checked ? [...d.tiers, x.id] : d.tiers.filter((id) => id !== x.id)
                            ws.set({ tiers: next.length ? next : [] })
                          }}
                        />
                        {x.name}
                      </label>
                    ))}
                  </div>
                ) : null}
              </div>
            ) : null}
          </Sec>
          <Sec title={t(mc.freePreview)}>
            {d.visibility === 'public' ? (
              <div className="flex items-center justify-between gap-2">
                <span className="k">{t(mc.freePreviewPublic)}</span>
                <button type="button" className="btn sm flex-none" disabled>
                  {t(mc.addMarker)}
                </button>
              </div>
            ) : marker ? (
              marker.at !== null ? (
                <div className="flex items-center justify-between gap-2">
                  <span className="flex min-w-0 flex-col">
                    <b>{t(mc.markerAt, { n: marker.at })}</b>
                    <span className="k">{t(mc.markerAtText, { nth: nth(marker.at) })}</span>
                  </span>
                  {ro ? null : (
                    <span className="flex flex-none gap-1.5">
                      <button type="button" className="btn sm" disabled={!marker.canMove} title={t(mc.moveHint)} onClick={marker.move}>
                        {t(mc.move)}
                      </button>
                      <button type="button" className="btn sm" onClick={marker.remove}>
                        {t(mc.remove)}
                      </button>
                    </span>
                  )}
                </div>
              ) : (
                <div className="flex items-center justify-between gap-2">
                  <span className="k">{t(mc.noMarker)}</span>
                  {ro ? null : (
                    <button type="button" className="btn sm flex-none" onClick={marker.add}>
                      {t(mc.addMarker)}
                    </button>
                  )}
                </div>
              )
            ) : (
              <div className="flex items-center justify-between gap-2">
                <span className="k">{p.hasMarker ? t(mc.markerAt, { n: p.markerAfter }) : t(mc.noMarker)}</span>
              </div>
            )}
            {d.visibility === 'public' ? null : (
              <Choice type="checkbox" checked={d.neverFree} disabled={ro} onChange={(on) => ws.set({ neverFree: on })} title={t(mc.neverFree)} sub={t(mc.neverFreeText)} />
            )}
          </Sec>
          <Sec title={t(mc.when)}>
            {mode === 'campaign' ? (
              <Choice
                name="wt-when"
                checked={d.publishMode === 'with_send'}
                disabled={ro || sent}
                onChange={() => ws.set({ publishMode: 'with_send', publishAt: undefined })}
                title={t(mc.whenWithSend)}
                sub={campaign?.scheduledAt ? t(mc.whenWithSendText, { date: `${shortDate(campaign.scheduledAt)}, ${clock(campaign.scheduledAt)}` }) : t(mc.whenWithSendNoDate)}
              />
            ) : (
              <Choice name="wt-when" checked={d.publishMode === 'now'} disabled={ro} onChange={() => ws.set({ publishMode: 'now', publishAt: undefined })} title={t(mc.whenNow)} />
            )}
            <Choice
              name="wt-when"
              checked={d.publishMode === 'at'}
              disabled={ro}
              onChange={() => ws.set({ publishMode: 'at', publishTz: tz, publishAt: d.publishAt ?? new Date(Date.now() + 86_400_000).toISOString() })}
              title={t(mc.whenAt)}
              sub={mode === 'campaign' ? t(mc.whenAtCampaign) : t(mc.whenAtText, { tz })}
            />
            {d.publishMode === 'at' ? (
              <input
                className="in ml-6"
                type="datetime-local"
                aria-label={t(mc.whenTime)}
                value={d.publishAt ? isoToZonedLocal(d.publishAt, tz) : ''}
                disabled={ro}
                onChange={(e) => e.target.value && ws.set({ publishAt: zonedToISO(e.target.value, tz), publishTz: tz })}
              />
            ) : null}
            {mode === 'campaign' ? (
              <Choice name="wt-when" dim={!sent} checked={d.publishMode === 'now'} disabled={ro || !sent} onChange={() => ws.set({ publishMode: 'now', publishAt: undefined })} title={t(mc.whenNow)} sub={t(mc.whenNowCampaign)} />
            ) : null}
          </Sec>
          <section className="wtab-sec">
            <div className="flex items-center justify-between gap-3">
              <b id="wt-feat">{t(mc.featureOnHome)}</b>
              <input type="checkbox" role="switch" className="tg" aria-labelledby="wt-feat" checked={d.featured} disabled={ro} onChange={(e) => ws.set({ featured: e.target.checked })} />
            </div>
          </section>
          <section className="wtab-sec">
            <button type="button" className="wtab-row" aria-expanded={seoOpen} onClick={() => setSeoOpen((o) => !o)}>
              <span className="flex min-w-0 flex-col gap-0.5 text-left">
                <b>{t(mc.seo)}</b>
                <span className="k">{mode === 'campaign' ? t(mc.seoSummaryCampaign, { host }) : t(mc.seoSummary)}</span>
              </span>
              <Icon name={seoOpen ? 'chevronDown' : 'chevronRight'} size={14} />
            </button>
            {seoOpen ? (
              <div className="flex flex-col gap-2.5">
                <label className="wtab-f">
                  <span>{t(mc.seoTitle)}</span>
                  <input className="in" value={d.seo.title} placeholder={d.title} maxLength={120} disabled={ro} onChange={(e) => ws.set({ seo: { ...d.seo, title: e.target.value } })} />
                </label>
                <label className="wtab-f">
                  <span>{t(mc.seoDescription)}</span>
                  <textarea className="in" rows={2} value={d.seo.description} placeholder={d.excerpt} maxLength={300} disabled={ro} onChange={(e) => ws.set({ seo: { ...d.seo, description: e.target.value } })} />
                </label>
                <label className="wtab-f">
                  <span>{t(mc.seoSocial)}</span>
                  <input className="in mono" value={d.seo.socialImage} placeholder={d.coverUrl || 'https://'} maxLength={500} disabled={ro} onChange={(e) => ws.set({ seo: { ...d.seo, socialImage: e.target.value } })} />
                </label>
                <label className="wtab-f">
                  <span>{t(mc.seoCanonical)}</span>
                  <input className="in mono" value={d.seo.canonicalUrl} placeholder="https://" maxLength={500} disabled={ro} onChange={(e) => ws.set({ seo: { ...d.seo, canonicalUrl: e.target.value } })} />
                  <span className="k">{t(mc.seoCanonicalHint)}</span>
                </label>
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={d.seo.hideFromSearch} disabled={ro} onChange={(e) => ws.set({ seo: { ...d.seo, hideFromSearch: e.target.checked } })} />
                  {t(mc.seoHide)}
                </label>
              </div>
            ) : null}
          </section>
          <WebChecks post={p} onShowHidden={onShowHidden} onUseSlug={(s) => ws.set({ slug: s })} onOpen={(id) => void navigate({ to: '/w/$workspace/forms/website/posts/$post', params: { workspace: slug, post: id } })} />
        </>
      ) : null}
      {ws.error ? (
        <span className="err px-3 pb-3" role="alert">
          {t(mc.saveSettingsFailed)}: {ws.error}
        </span>
      ) : null}
      <ImagePickerDialog open={picking} onClose={() => setPicking(false)} onPick={(url) => ws.set({ coverUrl: url })} />
    </div>
  )
}

/** "Web checks · 4 passed": the address, the cover's alt text, tokens and hidden blocks. */
export function WebChecks({ post, onShowHidden, onUseSlug, onOpen, compact }: { post: Post; onShowHidden?: () => void; onUseSlug?: (s: string) => void; onOpen?: (id: string) => void; compact?: boolean }) {
  const t = useT()
  const checks = post.checks
  const passed = checks.filter((c) => c.status === 'pass').length
  const notes = checks.filter((c) => c.status === 'warn' || c.status === 'info').length
  const blocks = checks.filter((c) => c.status === 'block').length
  const title = blocks ? t(mc.checksFix, { count: blocks }) : notes ? t(mc.checksNotes, { passed, notes }) : t(mc.checks, { passed })
  return (
    <section className="wtab-sec" aria-label={title}>
      {compact ? null : <div className="wtab-h">{title}</div>}
      <div className="flex flex-col">
        {checks.map((c) => (
          <CheckRow key={c.key} c={c} post={post} onShowHidden={onShowHidden} onUseSlug={onUseSlug} onOpen={onOpen} />
        ))}
      </div>
    </section>
  )
}

function CheckIcon({ status }: { status: string }) {
  if (status === 'pass') return <Icon name="check" size={14} className="text-ok-fg" />
  if (status === 'block') return <Icon name="alertCircle" size={14} className="text-bad-fg" />
  if (status === 'warn') return <Icon name="alertCircle" size={14} className="text-warn-fg" />
  return <Icon name="info" size={14} className="text-info-fg" />
}

function CheckRow({ c, post, onShowHidden, onUseSlug, onOpen }: { c: WebCheck; post: Post; onShowHidden?: () => void; onUseSlug?: (s: string) => void; onOpen?: (id: string) => void }) {
  const t = useT()
  let title: string
  let detail: ReactNode = null
  switch (c.key) {
    case 'address':
      if (c.status === 'block') {
        title = t(mc.checkAddressTaken)
        detail = (
          <>
            <span>{t(mc.checkAddressTakenText, { path: c.path ?? '' })}</span>
            <span className="flex flex-wrap gap-1.5 pt-1">
              {c.suggestion && onUseSlug ? (
                <button type="button" className="btn sm" onClick={() => onUseSlug(c.suggestion!)}>
                  {t(mc.useSlug, { slug: c.suggestion })}
                </button>
              ) : null}
              {c.taken && onOpen ? (
                <button type="button" className="btn sm" onClick={() => onOpen(c.taken!.id)}>
                  {t(mc.openOwner, { year: new Date(c.taken.since).getFullYear() })}
                </button>
              ) : null}
            </span>
          </>
        )
      } else {
        title = t(mc.checkAddressFree)
        detail = c.path
      }
      break
    case 'cover':
      title = c.status === 'pass' ? t(mc.checkCover) : post.coverUrl ? t(mc.checkCoverMissing) : t(mc.checkCoverNone)
      break
    case 'tokens':
      if (!c.fallbacks.length) {
        title = t(mc.checkTokensNone)
        detail = t(mc.checkTokensNoneText)
      } else {
        title = t(mc.checkTokens, { count: c.fallbacks.length })
        detail = c.fallbacks.map((f) => t(mc.checkTokensText, { source: f.source, sample: f.sample })).join(' · ')
      }
      break
    default:
      if (!c.hidden.length) {
        title = t(mc.checkHiddenNone)
      } else {
        title = t(mc.checkHidden, { count: c.hidden.length })
        detail = (
          <>
            <span>{andList(c.hidden.map((h) => hiddenName(t, h)))}</span>
            {onShowHidden ? (
              <span className="pt-1">
                <button type="button" className="btn sm" onClick={onShowHidden}>
                  {t(mc.showOnCanvas)}
                </button>
              </span>
            ) : null}
          </>
        )
      }
  }
  return (
    <div className="ws-chkl">
      <CheckIcon status={c.status} />
      <span className="tx">
        <b>{title}</b>
        {detail ? <span className="k flex flex-col">{detail}</span> : null}
      </span>
    </div>
  )
}

export function hiddenName(t: ReturnType<typeof useT>, h: WebCheck['hidden'][number]): string {
  switch (h.kind) {
    case 'view_online':
      return t(mc.hiddenViewOnline)
    case 'footer':
      return t(mc.hiddenFooter)
    case 'referral':
      return t(mc.hiddenReferral)
    case 'header':
      return t(mc.hiddenHeader)
    default:
      return h.name ? t(mc.hiddenConditional, { name: h.name }) : h.condition ? t(mc.hiddenConditional, { name: h.condition }) : t(mc.hiddenBlock)
  }
}

/** Saves a post's settings: PUT the post, and refresh what shows it. */
export function useSavePost(postId: string) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  return useCallback(
    async (input: PostInput): Promise<Post> => {
      const { data } = await http.PUT(`${WS}/posts/{post}`, { params: { path: { workspace: slug, post: postId } }, body: input })
      void qc.invalidateQueries({ predicate: (q) => typeof q.queryKey[1] === 'string' && (q.queryKey[1] as string).includes('/website/posts') && q.queryKey[1] !== `${WS}/posts/{post}` })
      return data!
    },
    [slug, postId, qc],
  )
}

/** "Wed 14 Oct": for the When section's line. */
export function shortDay(d: string): string {
  return dayMonthDate(d)
}
