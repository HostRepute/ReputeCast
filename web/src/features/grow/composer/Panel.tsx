import type { Component, Editor } from 'grapesjs'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { $api } from '@/api/client'
import { Skeleton } from '@/components/ui/Feedback'
import { OptionCard } from '@/components/ui/Field'
import { Menu, MenuItem, MenuLabel } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { GROW, orNames, tileInitials } from '../shared'
import {
  growBlocksIn,
  growKindOf,
  growRaw,
  hiddenReason,
  setGrowPreview,
  setGrowRaw,
  useGrowPreview,
  useGrowRender,
  type RecsSettings,
  type ReferralSettings,
  type RewardSettings,
} from './blocks'
import { mb } from './messages'
import './composer-grow.css'

type Raw = Record<string, unknown>

// --- Blocks panel ------------------------------------------------------------------

/** "Open Grow" in the Grow group's heading (a new tab: the draft stays open). */
export function OpenGrowLink() {
  const t = useT()
  const { slug } = useWorkspace()
  const canView = useCan('grow:view')
  if (!canView) return null
  return (
    <a className="gw-cmp-open" href={`/w/${slug}/grow`} target="_blank" rel="noreferrer">
      {t(mb.openGrow)}
    </a>
  )
}

/** The line under the Grow blocks. */
export function GrowBlocksNote({ milestone }: { milestone?: boolean }) {
  const t = useT()
  return <p className="gw-cmp-note">{t(milestone ? mb.groupNoteMilestone : mb.groupNote)}</p>
}

// --- Canvas toolbar ----------------------------------------------------------------

/**
 * "Preview as" for Grow's blocks: the program's readers, readers the block
 * hides for, and the sample. It shows while the email has a Referral or
 * Reward block; the composer starts on a reader with a reward still ahead.
 */
export function GrowPreviewAs({ editor, slug, rev, campaignId }: { editor: Editor | null; slug: string; rev: number; campaignId?: string }) {
  const t = useT()
  const id = useId()
  void rev
  // A campaign in several languages has the languages' "Preview as" too
  // (its versions, read from the cache that one fills): this one says what it's for.
  const langs = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages',
    { params: { path: { workspace: slug, campaignId: campaignId ?? '' } } },
    { enabled: false },
  )
  const both = (langs.data?.versions.length ?? 0) > 1
  const blocks = growBlocksIn(editor)
  const referral = blocks.find((b) => growKindOf(b) === 'referral')
  const program = referral ? String(growRaw(referral).program ?? '') : ''
  const has = blocks.some((b) => growKindOf(b) !== 'recommendations')
  const q = $api.useQuery('get', `${GROW}/blocks/preview-contacts`, { params: { path: { workspace: slug }, query: program ? { program } : {} } }, { enabled: has, staleTime: 60_000 })
  const pv = useGrowPreview()
  // Each composer starts on its own reader.
  useEffect(() => {
    setGrowPreview({ slug, contactId: null })
    return () => setGrowPreview({ contactId: null })
  }, [slug])
  useEffect(() => {
    if (pv.contactId !== null) return
    if (q.data) setGrowPreview({ contactId: q.data.items.find((x) => x.onList)?.id ?? '' })
    else if (q.isError) setGrowPreview({ contactId: '' })
  }, [q.data, q.isError, pv.contactId])
  if (!has) return null
  const items = q.data?.items ?? []
  return (
    <label className="gw-pvas" htmlFor={id}>
      <span>{t(both ? mb.previewAsGrow : mb.previewAs)}</span>
      <select id={id} className="in" value={pv.contactId ?? ''} disabled={!q.data && !q.isError} onChange={(e) => setGrowPreview({ contactId: e.target.value })}>
        {items.map((p) => (
          <option key={p.id} value={p.id}>
            {p.onList ? t(mb.previewReader, { name: p.name, list: q.data?.listName ?? '' }) : t(mb.previewReaderOff, { name: p.name, lists: p.lists.join(', ') })}
          </option>
        ))}
        <option value="">{t(mb.previewSample)}</option>
      </select>
    </label>
  )
}

// --- Inspector -----------------------------------------------------------------------

function Prop({ title, children, hint }: { title?: ReactNode; children?: ReactNode; hint?: ReactNode }) {
  const id = useId()
  return (
    <div className="prop" role="group" aria-labelledby={title ? id : undefined}>
      {title ? (
        <span className="t" id={id}>
          {title}
        </span>
      ) : null}
      {children}
      {hint ? <span className="k">{hint}</span> : null}
    </div>
  )
}

function Head({ title, caption }: { title: string; caption: string }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
      <span className="font-semibold">{title}</span>
      <span className="k">{caption}</span>
    </div>
  )
}

/** A text field that commits on Enter or when it loses focus. */
function Draft({
  value,
  onCommit,
  label,
  multiline,
  onCaret,
}: {
  value: string
  onCommit: (v: string) => void
  label: string
  multiline?: boolean
  /** Where the caret was when the text lost focus (Insert token puts the token there). */
  onCaret?: (start: number, end: number) => void
}) {
  const [draft, setDraft] = useState<string | null>(null)
  const shown = draft ?? value
  const commit = () => {
    if (draft !== null && draft !== value) onCommit(draft)
    setDraft(null)
  }
  return multiline ? (
    <textarea
      className="in gw-cmp-text"
      aria-label={label}
      rows={3}
      value={shown}
      onFocus={() => setDraft(value)}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={(e) => {
        onCaret?.(e.target.selectionStart, e.target.selectionEnd)
        commit()
      }}
    />
  ) : (
    <input
      className="in"
      aria-label={label}
      value={shown}
      onFocus={() => setDraft(value)}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
      }}
    />
  )
}

function Check({ checked, onChange, children }: { checked: boolean; onChange: (v: boolean) => void; children: ReactNode }) {
  return (
    <label className="gw-cmp-check">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      {children}
    </label>
  )
}

function Loading() {
  const t = useT()
  return (
    <div className="flex flex-col gap-2" style={{ padding: '12px' }} aria-busy="true">
      <span className="vh" role="status">
        {t(mb.loading)}
      </span>
      <Skeleton h={14} w="60%" />
      <Skeleton h={32} />
      <Skeleton h={64} />
      <Skeleton h={32} />
    </div>
  )
}

/** The settings of a Grow block (Settings tab). */
export function GrowBlockInspector({ c }: { c: Component }) {
  const kind = growKindOf(c)
  if (kind === 'referral') return <ReferralInspector c={c} />
  if (kind === 'recommendations') return <RecsInspector c={c} />
  if (kind === 'reward') return <RewardInspector c={c} />
  return null
}

/** The Style tab of a Grow block: its look comes from Grow. */
export function GrowStyleNote() {
  const t = useT()
  return (
    <div className="k" style={{ padding: '14px 12px' }}>
      {t(mb.styleNote)}
    </div>
  )
}

const SHARE = ['email', 'x', 'linkedin', 'whatsapp'] as const
type Which = 'none' | 'one' | 'many'
const TOKENS = ['referral.count', 'referral.to_next', 'referral.next_milestone.reward', 'referral.next_milestone.count', 'referral.pending', 'referral.link', 'referral.code', 'referral.hub_url'] as const

function ReferralInspector({ c }: { c: Component }) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useId()
  const canView = useCan('grow:view')
  const canEditGrow = useCan('grow:edit')
  const r = useGrowRender(c)
  const programs = $api.useQuery('get', `${GROW}/referrals`, { params: { path: { workspace: slug } } }, { enabled: canView, staleTime: 60_000 })
  const [pick, setPick] = useState<Which | null>(null)
  // The caret in the text, for Insert token; null until the text had focus.
  const caret = useRef<[number, number] | null>(null)
  const data = r?.status === 'ok' ? r.referral : undefined
  const raw = growRaw(c)
  if (!data) return r?.status === 'error' ? <Note tone="warn">{t(mb.renderFailed)}</Note> : <Loading />
  const eff: ReferralSettings = data.settings
  const which: Which = pick ?? (data.count === 0 ? 'none' : data.count === 1 ? 'one' : 'many')
  const rawTexts = (raw.texts && typeof raw.texts === 'object' ? raw.texts : {}) as Partial<Record<Which, string>>
  const write = (change: Partial<ReferralSettings>) => {
    const next: Raw = { layout: eff.layout, button: eff.button, share: eff.share, progress: eff.progress }
    if (raw.program || eff.program) next.program = raw.program ?? eff.program
    if (Object.keys(rawTexts).length) next.texts = rawTexts
    setGrowRaw(c, { ...next, ...change })
  }
  const current = rawTexts[which] || eff.texts[which]
  const insert = (token: string) => {
    const tag = `{{ ${token} }}`
    const [at, end] = caret.current && caret.current[1] <= current.length ? caret.current : [current.length, current.length]
    caret.current = [at + tag.length, at + tag.length]
    write({ texts: { ...rawTexts, [which]: current.slice(0, at) + tag + current.slice(end) } as ReferralSettings['texts'] })
  }
  const list = programs.data
  const programId = String(raw.program ?? data.programId)
  return (
    <>
      <Head title={t(mb.referralTitle)} caption={t(mb.referralCaption)} />
      <Prop title={t(mb.program)}>
        {list && list.items.length ? (
          <select className="in" aria-label={t(mb.program)} value={programId} onChange={(e) => write({ program: e.target.value })}>
            {list.items.map((p) => (
              <option key={p.id} value={p.id}>
                {p.status === 'on' && p.startedAt ? t(mb.programOn, { list: p.listName, date: dayMonthDate(p.startedAt) }) : t(mb.programOff, { list: p.listName })}
              </option>
            ))}
          </select>
        ) : (
          <span>{data.listName || t(mb.noProgram)}</span>
        )}
      </Prop>
      <Prop title={t(mb.layout)}>
        <div className="flex flex-col gap-1.5">
          {(
            [
              ['line', mb.layoutLineTitle, mb.layoutLineText],
              ['progress', mb.layoutProgressTitle, mb.layoutProgressText],
              ['rewards', mb.layoutRewardsTitle, mb.layoutRewardsText],
            ] as const
          ).map(([v, title, desc]) => (
            <OptionCard key={v} name={name} value={v} checked={eff.layout === v} onChange={() => write({ layout: v })} title={t(title)} description={t(desc)} />
          ))}
        </div>
      </Prop>
      <Prop title={t(mb.text)}>
        <div className="seg gw-cmp-seg" role="group" aria-label={t(mb.textFor)}>
          {(
            [
              ['none', mb.textNone],
              ['one', mb.textOne],
              ['many', mb.textMany],
            ] as const
          ).map(([v, label]) => (
            <button
              key={v}
              type="button"
              aria-pressed={which === v}
              onClick={() => {
                caret.current = null
                setPick(v)
              }}
            >
              {t(label)}
            </button>
          ))}
        </div>
        <Draft
          key={which}
          multiline
          onCaret={(start, end) => (caret.current = [start, end])}
          label={t(mb.textLabel, { which })}
          value={current}
          onCommit={(v) => write({ texts: { ...rawTexts, [which]: v } as ReferralSettings['texts'] })}
        />
        <div className="flex items-center justify-between gap-2">
          <span className="k">{t(mb.textHint)}</span>
          <Menu
            label={t(mb.insertToken)}
            trigger={
              <button type="button" className="btn sm">
                <span className="mono" aria-hidden="true">
                  {'{}'}
                </span>
                {t(mb.insertToken)}
              </button>
            }
          >
            <MenuLabel>{t(mb.tokensReferral)}</MenuLabel>
            {TOKENS.map((k) => (
              <MenuItem key={k} onSelect={() => insert(k)}>
                <span className="mono">{k}</span>
              </MenuItem>
            ))}
          </Menu>
        </div>
      </Prop>
      {eff.layout !== 'line' ? (
        <>
          <Prop title={t(mb.button)}>
            <Draft label={t(mb.button)} value={eff.button} onCommit={(v) => v.trim() && write({ button: v.trim() })} />
          </Prop>
          <Prop title={t(mb.share)}>
            <div className="row2">
              {SHARE.map((s) => (
                <Check key={s} checked={eff.share.includes(s)} onChange={(on) => write({ share: SHARE.filter((x) => (x === s ? on : eff.share.includes(x))) })}>
                  {t(s === 'email' ? mb.shareEmail : s === 'x' ? mb.shareX : s === 'linkedin' ? mb.shareLinkedin : mb.shareWhatsapp)}
                </Check>
              ))}
            </div>
          </Prop>
        </>
      ) : null}
      {eff.layout === 'progress' ? (
        <div className="prop">
          <label className="flex items-center justify-between gap-3">
            <span className="t">{t(mb.progress)}</span>
            <input type="checkbox" className="tg" role="switch" checked={eff.progress} onChange={(e) => write({ progress: e.target.checked })} />
          </label>
        </div>
      ) : null}
      <div className="prop">
        <Note tone="info">{data.listName ? t(mb.hiddenFor, { list: data.listName }) : t(mb.hiddenForNoProgram)}</Note>
      </div>
      <Prop>
        <span className="k">{t(mb.fromGrow)}</span>
        <span>
          {canEditGrow ? (
            <a className="btn sm" href={`/w/${slug}/grow/referrals/milestones?program=${encodeURIComponent(programId)}`} target="_blank" rel="noreferrer">
              {t(mb.editMilestones)}
            </a>
          ) : (
            <button type="button" className="btn sm" aria-disabled="true" disabled>
              {t(mb.editMilestones)}
            </button>
          )}
        </span>
        {!canEditGrow ? (
          <span className="gw-cmp-why">
            {t(list?.helpers.edit.length ? mb.needsGrowEdit : mb.needsGrowEditNoHelpers, { helpers: orNames(list?.helpers.edit) })}
          </span>
        ) : null}
      </Prop>
    </>
  )
}

function RecsInspector({ c }: { c: Component }) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useId()
  const canView = useCan('grow:view')
  const r = useGrowRender(c)
  const data = r?.status === 'ok' ? r.recs : undefined
  if (!data) return r?.status === 'error' ? <Note tone="warn">{t(mb.renderFailed)}</Note> : <Loading />
  const eff: RecsSettings = data.settings
  const write = (change: Partial<RecsSettings>) => setGrowRaw(c, { which: eff.which, ids: eff.ids, layout: eff.layout, heading: eff.heading, intro: eff.intro, ...change })
  const showing = data.options.filter((o) => o.showBlock && !o.hidden)
  const picked = (id: string) => (eff.which === 'pick' ? eff.ids.includes(id) : showing.some((o) => o.id === id))
  return (
    <>
      <Head title={t(mb.recsTitle)} caption={data.pageUrl ? t(mb.recsCaption, { page: data.pageUrl }) : t(mb.recsCaptionNoPage)} />
      <Prop title={t(mb.which)}>
        <div className="flex flex-col gap-1.5">
          <OptionCard
            name={name}
            value="all"
            checked={eff.which === 'all'}
            onChange={() => write({ which: 'all' })}
            title={t(mb.whichAll)}
            description={t(mb.whichAllText, { count: showing.length })}
          />
          <OptionCard
            name={name}
            value="pick"
            checked={eff.which === 'pick'}
            onChange={() => write({ which: 'pick', ids: showing.map((o) => o.id) })}
            title={t(mb.whichPick)}
            description={t(mb.whichPickText)}
          />
        </div>
        {data.options.length ? (
          <ul className="gw-cmp-recs" aria-label={t(mb.which)}>
            {data.options.map((o) => (
              <li key={o.id}>
                <label className="gw-cmp-rec">
                  <input
                    type="checkbox"
                    checked={picked(o.id)}
                    disabled={eff.which !== 'pick'}
                    onChange={(e) => write({ which: 'pick', ids: e.target.checked ? [...eff.ids, o.id] : eff.ids.filter((x) => x !== o.id) })}
                  />
                  <span className="gw-mtile" style={{ width: 28, height: 28, fontSize: 11, borderRadius: 7 }} aria-hidden="true">
                    {o.initials || tileInitials(o.name)}
                  </span>
                  <span className="flex min-w-0 flex-col">
                    <span className="font-semibold">{o.name}</span>
                    <span className="k">{optionLine(t, o)}</span>
                  </span>
                </label>
              </li>
            ))}
          </ul>
        ) : (
          <span className="k">{t(mb.noOptions)}</span>
        )}
      </Prop>
      <Prop title={t(mb.layout)}>
        <div className="seg gw-cmp-seg" role="group" aria-label={t(mb.layout)}>
          {(
            [
              ['stacked', mb.layoutStacked],
              ['columns', mb.layoutColumns],
              ['compact', mb.layoutCompact],
            ] as const
          ).map(([v, label]) => (
            <button key={v} type="button" aria-pressed={eff.layout === v} onClick={() => write({ layout: v })}>
              {t(label)}
            </button>
          ))}
        </div>
      </Prop>
      <Prop title={t(mb.heading)}>
        <Draft label={t(mb.heading)} value={eff.heading} onCommit={(v) => v.trim() && write({ heading: v.trim() })} />
      </Prop>
      <Prop title={t(mb.intro)}>
        <Draft label={t(mb.intro)} value={eff.intro} onCommit={(v) => write({ intro: v.trim() })} />
      </Prop>
      <Prop title={t(mb.buttons)} hint={t(mb.buttonsText)} />
      <div className="prop">
        <Note tone="info">{data.pageUrl ? t(mb.scannersNote, { page: data.pageUrl }) : t(mb.scannersNoteNoPage)}</Note>
        <Note tone="neutral">{t(mb.hiddenWhenNone)}</Note>
      </div>
      {canView ? (
        <Prop>
          <span>
            <a className="btn sm" href={`/w/${slug}/grow/recommendations`} target="_blank" rel="noreferrer">
              {t(mb.manageRecommendations)}
            </a>
          </span>
        </Prop>
      ) : null}
    </>
  )
}

/** "Partner · connected", "Link · showing", "Confirmation page only · link invalid". */
function optionLine(t: ReturnType<typeof useT>, o: { kind: string; state: string; showBlock: boolean; hidden: boolean; name: string }): string {
  const state = o.hidden ? hiddenReason(o.state) : t(o.state === 'connected' ? mb.stateConnected : o.state === 'unreachable' ? mb.stateUnreachable : mb.stateShowing)
  if (!o.showBlock) return t(mb.optionConfirmOnly, { state })
  return t(o.kind === 'partner' ? mb.optionPartner : mb.optionLink, { state })
}

function RewardInspector({ c }: { c: Component }) {
  const t = useT()
  const r = useGrowRender(c)
  const data = r?.status === 'ok' ? r.referral : undefined
  if (!data) return r?.status === 'error' ? <Note tone="warn">{r.message}</Note> : <Loading />
  const eff: RewardSettings = data.reward
  const write = (change: Partial<RewardSettings>) => setGrowRaw(c, { ...eff, ...change })
  return (
    <>
      <Head title={t(mb.rewardTitle)} caption={t(mb.rewardCaption)} />
      <Prop title={t(mb.rewardButton)}>
        <Draft label={t(mb.rewardButton)} value={eff.button} onCommit={(v) => v.trim() && write({ button: v.trim() })} />
      </Prop>
      <Prop title={t(mb.rewardShow)}>
        <div className="row2">
          <Check checked={eff.image} onChange={(v) => write({ image: v })}>
            {t(mb.rewardImage)}
          </Check>
          <Check checked={eff.description} onChange={(v) => write({ description: v })}>
            {t(mb.rewardDescription)}
          </Check>
          <Check checked={eff.expiry} onChange={(v) => write({ expiry: v })}>
            {t(mb.rewardExpiry)}
          </Check>
          <Check checked={eff.hub} onChange={(v) => write({ hub: v })}>
            {t(mb.rewardHub)}
          </Check>
        </div>
      </Prop>
      <Prop title={t(mb.rewardTokens)}>
        <span className="flex flex-wrap gap-1.5">
          {['reward.name', 'reward.description', 'reward.download_url', 'reward.code', 'reward.claim_url', 'reward.expires_at', 'reward.note'].map((k) => (
            <span key={k} className="gw-cmp-tok">
              {k}
            </span>
          ))}
        </span>
      </Prop>
    </>
  )
}
