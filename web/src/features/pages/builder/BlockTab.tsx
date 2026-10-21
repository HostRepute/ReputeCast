import { Link } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import type { Component } from 'grapesjs'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { errorMessage, http } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { useCan } from '@/lib/session'
import { featureItemHtml, questionHtml, quoteHtml } from './blocks'
import { useBuilder } from './context'
import { useResumeForm } from './hooks'
import { buttonPrice, moneyText, useForms, useProduct, useProducts, useReminders, useTags, useTiers, type ProductItem } from './data'
import { addItem } from './editor/actions'
import { attrOf, attrsOf, blockOf, byCid, classesOf, columnNumber, elementChildren, findById, isTop, kindOf, liveTagOf, nodeOf, tagOf, topOf } from './editor/nodes'
import { AttrFlag, AttrSeg, AttrSelect, AttrText, Prop, setAttrs } from './fields'
import { BlockIcon, UiIcon, type BlockIconName } from './icons'
import { selectionLabel } from './labels'
import {
  bgOf,
  BGS,
  buttonDisplay,
  buttonTemplate,
  escapeText,
  formatUnits,
  isVideoUrl,
  joinAt,
  joinList,
  parseList,
  parseUnits,
  soleLive,
  spaceOf,
  SPACES,
  splitAt,
  timeLeft,
  timeZones,
  UNITS,
  withBg,
  withSpace,
  zonedInstant,
  type Bg,
  type Space,
  type Unit,
} from './model'
import { m } from './messages'
import { kindDescription, kindName, localTime, wallText, zoneLabel, type T as TFn } from './wording'

const BG_SWATCH: Record<Bg, string> = { white: '#FFFFFF', lavender: '#F5F3FF', indigo: '#4338CA', ink: '#0B1220' }

function bgLabel(t: TFn, bg: Bg): string {
  return bg === 'white' ? t(m.bgWhite) : bg === 'lavender' ? t(m.bgLavender) : bg === 'indigo' ? t(m.bgIndigo) : t(m.bgInk)
}

function spaceLabel(t: TFn, s: Space): string {
  return s === 'compact' ? t(m.spCompact) : s === 'normal' ? t(m.spNormal) : t(m.spRoomy)
}

/** The block's icon, name, what it is, and Remove. */
function BlockHead({ c, onRemove }: { c: Component; onRemove?: () => void }) {
  const t = useT()
  const b = useBuilder()
  const kind = kindOf(c)
  const col = isTop(c) || liveTagOf(c) ? null : columnNumber(c)
  return (
    <Prop row>
      <span className="flex items-center gap-2">
        <BlockIcon name={kind as BlockIconName} size={18} color="#0F766E" />
        <span className="flex flex-col">
          <span className="t">{col ? t(m.columnTitle, { n: col }) : kindName(t, kind)}</span>
          <span className="k">{col ? t(m.columnHint) : kindDescription(t, kind)}</span>
        </span>
      </span>
      {b.canEdit && onRemove ? (
        <button type="button" className="btn sm" aria-label={t(m.removeBlock)} title={t(m.removeBlock)} onClick={onRemove}>
          <UiIcon name="trash" />
        </button>
      ) : null}
    </Prop>
  )
}

/** Background and spacing from the theme (PageBuilder · Style · from the theme). */
function StylePanel({ section, block }: { section: Component; block?: Component }) {
  const t = useT()
  const b = useBuilder()
  void b.rev
  // A product's or a form's box has a background of its own; the rest take the section's.
  const own = block && (liveTagOf(block) === 'rc-product' || liveTagOf(block) === 'rc-form')
  const bg: Bg = own ? (BGS.includes(attrOf(block!, 'bg') as Bg) ? (attrOf(block!, 'bg') as Bg) : 'white') : bgOf(nodeOf(section))
  const space = spaceOf(nodeOf(section))
  const setBg = (v: Bg) => {
    if (own) setAttrs(block!, { bg: v })
    else {
      section.setClass(withBg(classesOf(section), v))
      section.addAttributes({ 'data-pb-bg': v })
    }
  }
  const setSpace = (v: Space) => {
    section.setClass(withSpace(classesOf(section), v))
    section.addAttributes({ 'data-pb-space': v })
  }
  return (
    <Prop>
      <span className="t">{t(m.styleTitle)}</span>
      <span className="k">{t(m.background)}</span>
      <div className="pb-sws" role="group" aria-label={t(m.background)}>
        {BGS.map((x) => (
          <button key={x} type="button" className="pb-swb" aria-pressed={bg === x} disabled={!b.canEdit} onClick={() => setBg(x)}>
            <i style={{ background: BG_SWATCH[x] }} aria-hidden="true" />
            {bgLabel(t, x)}
          </button>
        ))}
      </div>
      <span className="k">{t(m.spacing)}</span>
      <div>
        <div className="seg" role="group" aria-label={t(m.spacing)}>
          {SPACES.map((x) => (
            <button key={x} type="button" aria-pressed={space === x} disabled={!b.canEdit} onClick={() => setSpace(x)}>
              {spaceLabel(t, x)}
            </button>
          ))}
        </div>
      </div>
      <span className="k">{t(m.styleNote)}</span>
    </Prop>
  )
}

function productOption(t: TFn, p: ProductItem): string {
  const price = p.price.kind === 'pwyw' ? t(m.pwywShort) : moneyText(p.price.amountCents, p.currency)
  const status = p.status === 'published' ? t(m.productPublished) : p.status === 'draft' ? t(m.productDraft) : t(m.productArchived)
  return t(m.productOption, { name: p.name, price, status })
}

function priceHint(t: TFn, p: ProductItem): string {
  if (p.price.kind === 'pwyw') {
    const min = moneyText(p.price.minimumCents, p.currency)
    return p.price.suggestedCents ? t(m.pricePwywSuggested, { min, suggested: moneyText(p.price.suggestedCents, p.currency) }) : t(m.pricePwyw, { min })
  }
  return t(m.priceOneTime, { price: moneyText(p.price.amountCents, p.currency) })
}

function StripeNote() {
  const t = useT()
  const b = useBuilder()
  const st = b.page.checks.find((c) => c.key === 'stripe')
  if (!st) return null
  if (st.status !== 'pass')
    return (
      <div className="pb-pbody">
        <div className="note w" role="note">
          <UiIcon name="info" size={16} color="#B45309" />
          <div className="flex min-w-0 flex-col gap-2">
            <span>{t(m.stripeMissing)}</span>
            <Link to="/w/$workspace/earn/settings" params={{ workspace: b.slug }} className="btn sm self-start">
              {t(m.connectStripe)}
            </Link>
          </div>
        </div>
      </div>
    )
  return (
    <div className="pb-pbody">
      <div className="note n" role="note">
        <UiIcon name="info" size={16} color="#4B5563" />
        <div>
          <T msg={m.stripeNote} values={{ account: st.account ?? '', mode: st.mode === 'test' ? t(m.modeTest) : t(m.modeLive), b: (c) => <b>{c}</b> }} />
        </div>
      </div>
    </div>
  )
}

function ProductPanel({ c }: { c: Component }) {
  const t = useT()
  const b = useBuilder()
  const products = useProducts()
  const id = attrOf(c, 'product')
  const p = products.items.find((x) => x.id === id)
  const detail = useProduct(id || undefined)
  const offerText = detail?.live?.offer?.on ? detail.live.offer.text : detail && !detail.live ? detail.draft.offer?.text : undefined
  const tpl = detail?.checks.find((x) => x.key === 'template')
  const price = p ? buttonPrice(p) : ''
  const short = p?.shortName || p?.name || ''
  const layout = attrOf(c, 'layout') || 'box'
  // The wide layout with the cover is for a band of its own, not for a column.
  const top = topOf(c)
  const band = Boolean(top && soleLive(nodeOf(top))?.key === c.cid)
  return (
    <>
      <Prop>
        <AttrSelect
          c={c}
          name="product"
          label={t(m.productLabel)}
          id="pb-field-product"
          options={[{ value: '', label: t(m.pickProduct) }, ...products.items.filter((x) => x.status !== 'archived' || x.id === id).map((x) => ({ value: x.id, label: productOption(t, x) }))]}
        />
        {products.ready && !products.items.length ? <span className="k">{t(m.noProducts)}</span> : null}
        {id ? (
          <Link to="/w/$workspace/earn/products/$productId" params={{ workspace: b.slug, productId: id }} className="k pb-klink">
            {t(m.openProduct)}
          </Link>
        ) : null}
      </Prop>
      <Prop>
        <AttrText c={c} name="button" label={t(m.buttonText)} hint={t(m.buttonHint)} show={(v) => buttonDisplay(v, price, short)} store={(v) => buttonTemplate(v, price, short)} />
      </Prop>
      <Prop>
        <AttrFlag c={c} name="price" label={t(m.showPrice)} hint={p ? priceHint(t, p) : undefined} />
      </Prop>
      <Prop>
        <AttrFlag c={c} name="offer" label={t(m.showOffer)} hint={t(m.offerHint)} />
        {offerText ? (
          <div className="pb-offerprev">
            <input type="checkbox" disabled aria-label={t(m.offerPreview)} />
            <span>
              <b style={{ fontWeight: 600 }}>{offerText}</b>
              <br />
              <span className="k">{t(m.offerConsent)}</span>
            </span>
          </div>
        ) : detail ? (
          <span className="k">{t(m.offerOff)}</span>
        ) : null}
        {id ? (
          <Link to="/w/$workspace/earn/products/$productId" params={{ workspace: b.slug, productId: id }} className="k pb-klink">
            {t(m.changeOffer)}
          </Link>
        ) : null}
      </Prop>
      <Prop>
        <span className="t">{t(m.afterPayment)}</span>
        <span className="k" style={{ color: '#374151' }}>
          {tpl?.template ? t(m.afterPaymentText, { template: tpl.template, version: tpl.version ?? 1 }) : t(m.afterPaymentPlain)}
        </span>
      </Prop>
      {band || layout === 'final' ? (
        <Prop>
          <span className="t">{t(m.productLayout)}</span>
          <div>
            <AttrSeg
              c={c}
              name="layout"
              label={t(m.productLayout)}
              fallback="box"
              items={[
                { value: 'box', label: t(m.layoutBox) },
                { value: 'final', label: t(m.layoutFinal) },
              ]}
            />
          </div>
          {layout === 'final' ? <AttrText c={c} name="eyebrow" label={t(m.eyebrow)} /> : null}
        </Prop>
      ) : null}
    </>
  )
}

/** The time zones, the block's and the workspace's first, with their short names on the block's day. */
function zoneOptions(t: TFn, tz: string, current: string, day: string) {
  const on = new Date(`${/^\d{4}-\d{2}-\d{2}$/.test(day) ? day : '2026-01-01'}T12:00:00Z`)
  return timeZones([current, tz, 'UTC']).map((z) => ({ value: z, label: zoneLabel(t, z, on) }))
}

/** "Opens in 5 days, 3 hours", as the site reads the countdown out. */
function spoken(t: TFn, label: string, at: Date | null, now: Date): string {
  if (!at) return ''
  const left = timeLeft(at, now)
  const l = label || t(m.defaultLabel)
  if (!left.total) return t(m.spokenEnded, { label: l })
  if (left.days) return t(m.spokenDays, { label: l, days: left.days, hours: left.hours })
  if (left.hours) return t(m.spokenHours, { label: l, hours: left.hours, minutes: left.minutes })
  return t(m.spokenMinutes, { label: l, minutes: Math.max(1, left.minutes) })
}

function useNow(): Date {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const h = window.setInterval(() => setNow(new Date()), 30_000)
    return () => window.clearInterval(h)
  }, [])
  return now
}

function CountdownPanel({ c }: { c: Component }) {
  const t = useT()
  const b = useBuilder()
  const forms = useForms()
  const now = useNow()
  void b.rev
  const a = attrsOf(c)
  const { date, time } = splitAt(a.at)
  const tz = a.tz || b.tz
  const at = zonedInstant(date, time, tz)
  const units = parseUnits(a.units)
  const end = a.end || 'hide'
  const wrapper = b.editor?.getWrapper()
  const reveal = parseList(a.reveal)
  const hide = parseList(a.hide)
  // The blocks a countdown can reveal: the page's blocks with an id, the countdown itself left out.
  const candidates = useMemo(() => {
    const out: { id: string; c: Component }[] = []
    wrapper?.onAll((x: Component) => {
      const id = attrOf(x, 'data-pb-id')
      if (!id || x === c) return
      if (isTop(x) || liveTagOf(x) || attrOf(x, 'data-pb-name')) out.push({ id, c: x })
    })
    return out
    // The page's blocks change with every edit: rev bumps then.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wrapper, b.rev, c])
  const main = reveal.find((id) => candidates.some((x) => x.id === id)) ?? ''
  const mainBlock = main ? findById(wrapper, main) : undefined
  const form = candidates.find((x) => liveTagOf(x.c) === 'rc-form')
  const formName = form ? (forms.items.find((f) => f.id === attrOf(form.c, 'form'))?.name ?? (attrOf(form.c, 'data-pb-name') || kindName(t, 'signup'))) : ''
  const w = wallText(date, time, tz)
  const setUnit = (u: Unit, on: boolean) => {
    const next = new Set(units)
    if (on) next.add(u)
    else next.delete(u)
    setAttrs(c, { units: formatUnits(next) })
  }
  const label = (x: { c: Component }) => selectionLabel(t, x.c, { products: [], forms: forms.items, now })
  return (
    <>
      <Prop>
        <div className="row2">
          <label className="lbl">
            {t(m.date)}
            <input type="date" className="in" value={date} disabled={!b.canEdit} onChange={(e) => e.target.value && setAttrs(c, { at: joinAt(e.target.value, time) })} />
          </label>
          <label className="lbl">
            {t(m.time)}
            <input type="time" className="in" value={time} disabled={!b.canEdit} onChange={(e) => e.target.value && setAttrs(c, { at: joinAt(date, e.target.value) })} />
          </label>
        </div>
        <AttrSelect c={c} name="tz" label={t(m.timeZone)} options={zoneOptions(t, b.tz, tz, date)} />
      </Prop>
      <Prop>
        <span className="t">{t(m.show)}</span>
        <div className="flex flex-wrap gap-3" style={{ fontSize: 12.5 }}>
          {UNITS.map((u) => (
            <label key={u} className="flex items-center gap-1.5">
              <input type="checkbox" checked={units.has(u)} disabled={!b.canEdit} onChange={(e) => setUnit(u, e.target.checked)} />
              {u === 'days' ? t(m.unitDays) : u === 'hours' ? t(m.unitHours) : u === 'minutes' ? t(m.unitMinutes) : t(m.unitSeconds)}
            </label>
          ))}
        </div>
        <span className="k">{t(m.readOut, { text: spoken(t, a.label ?? '', at, now) })}</span>
        <AttrText c={c} name="label" label={t(m.cdLabel)} />
      </Prop>
      <Prop>
        <span className="t">{t(m.whenEnds)}</span>
        <label className={`pb-radio7${end === 'hide' ? ' on' : ''}`}>
          <input type="radio" name={`end-${c.cid}`} checked={end === 'hide'} disabled={!b.canEdit} onChange={() => setAttrs(c, { end: 'hide' })} />
          <span>{t(m.endHide)}</span>
        </label>
        <label className={`pb-radio7${end === 'message' ? ' on' : ''}`}>
          <input type="radio" name={`end-${c.cid}`} checked={end === 'message'} disabled={!b.canEdit} onChange={() => setAttrs(c, { end: 'message' })} />
          <span className="flex min-w-0 flex-1 flex-col gap-1.5">
            <span>{t(m.endMessage)}</span>
            {end === 'message' ? <AttrText c={c} name="message" label={t(m.message)} /> : null}
          </span>
        </label>
        <label className={`pb-radio7${end === 'reveal' ? ' on' : ''}`}>
          <input type="radio" name={`end-${c.cid}`} checked={end === 'reveal'} disabled={!b.canEdit} onChange={() => setAttrs(c, { end: 'reveal' })} />
          <span className="flex min-w-0 flex-1 flex-col gap-1.5">
            <b style={{ fontWeight: 600 }}>{t(m.endReveal)}</b>
            {end === 'reveal' ? (
              <>
                <select
                  className="in"
                  aria-label={t(m.blockToReveal)}
                  value={main}
                  disabled={!b.canEdit}
                  onChange={(e) => {
                    const pick = e.target.value
                    // The other blocks it reveals (a second line of text) stay.
                    const rest = reveal.filter((id) => id !== main)
                    setAttrs(c, { reveal: joinList(pick ? [pick, ...rest] : rest), hide: joinList(hide.filter((id) => id !== pick)) })
                  }}
                >
                  <option value="">{t(m.pickBlock)}</option>
                  {candidates.map((x) => (
                    <option key={x.id} value={x.id}>
                      {label(x)}
                    </option>
                  ))}
                </select>
                {form ? (
                  <label className="flex items-center gap-1.5">
                    <input
                      type="checkbox"
                      checked={hide.includes(form.id)}
                      disabled={!b.canEdit}
                      onChange={(e) => setAttrs(c, { hide: joinList(e.target.checked ? [...hide, form.id] : hide.filter((id) => id !== form.id)) })}
                    />
                    {t(m.hideFormToo, { form: formName })}
                  </label>
                ) : null}
              </>
            ) : null}
          </span>
        </label>
      </Prop>
      <div className="pb-pbody">
        <div className="note n" role="note">
          <UiIcon name="info" size={16} color="#4B5563" />
          <div>
            {end === 'reveal'
              ? t(m.cdNoteReveal, { target: mainBlock && liveTagOf(mainBlock) === 'rc-product' ? t(m.cdTargetBuy) : t(m.cdTargetBlock, { name: mainBlock ? attrOf(mainBlock, 'data-pb-name') || kindName(t, kindOf(mainBlock)) : '' }), date: w.day, time: w.time })
              : end === 'message'
                ? t(m.cdNoteMessage, { date: w.day, time: w.time })
                : t(m.cdNoteHide, { date: w.day, time: w.time })}
          </div>
        </div>
      </div>
    </>
  )
}

function ReminderCard() {
  const t = useT()
  const b = useBuilder()
  const qc = useQueryClient()
  const canAutomate = useCan('automations:edit')
  const q = useReminders(b.page.id, true)
  const [hidden, setHidden] = useState(() => {
    try {
      return window.localStorage.getItem(`rc-page-reminders-dismissed:${b.page.id}`) === '1'
    } catch {
      return false
    }
  })
  const [busy, setBusy] = useState(false)
  const r = q.data
  if (!r || hidden) return null
  if (r.automation)
    return (
      <div className="pb-pbody">
        <div className="note i" role="note">
          <UiIcon name="info" size={16} color="#1D4ED8" />
          <div className="flex min-w-0 flex-col gap-2">
            <span>{t(m.remExisting, { name: r.automation.name, status: r.automation.status })}</span>
            <Link to="/w/$workspace/automations/$automationId" params={{ workspace: b.slug, automationId: r.automation.id }} className="btn sm self-start">
              {t(m.remOpen)}
            </Link>
          </div>
        </div>
      </div>
    )
  if (!r.available || !canAutomate || !b.canEdit) return null
  const create = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/reminder-automation', { params: { path: { workspace: b.slug, page: b.page.id } } })
      toast({ kind: 'success', title: t(m.remDone) })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/pages/{page}/reminder-automation'] })
    } catch (e) {
      // Until the server has the automation, the offer goes quietly.
      if ((e as { status?: number }).status === 404) setHidden(true)
      else toast({ kind: 'error', title: t(m.remFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const notNow = () => {
    setHidden(true)
    try {
      window.localStorage.setItem(`rc-page-reminders-dismissed:${b.page.id}`, '1')
    } catch {
      /* storage unavailable */
    }
  }
  return (
    <div className="pb-pbody">
      <div className="note i" role="note">
        <UiIcon name="info" size={16} color="#1D4ED8" />
        <div>
          <b>{t(m.remTitle)}</b>
          <br />
          <T msg={m.remBody} values={{ field: r.field, reminders: r.reminders, mono: (x) => <span className="mono">{x}</span> }} />
          <div className="mt-2 flex flex-wrap gap-1.5">
            <button type="button" className="btn sm pri" aria-busy={busy || undefined} disabled={busy} onClick={() => void create()}>
              {t(m.remCreate)}
            </button>
            <button type="button" className="btn sm" onClick={notNow}>
              {t(m.remNotNow)}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function EventPanel({ c }: { c: Component }) {
  const t = useT()
  const b = useBuilder()
  void b.rev
  const a = attrsOf(c)
  const tz = a.tz || b.tz
  const start = zonedInstant(a.date ?? '', a.start ?? '', tz)
  const endAt = a.end ? zonedInstant(a.date ?? '', a.end, tz) : null
  const where = a.where === 'venue' ? 'venue' : 'online'
  const lt = start ? localTime(start, endAt && endAt > start ? endAt : start) : null
  const local = lt ? t(m.readerTimeExample, { range: lt.time, city: lt.city }) : undefined
  return (
    <>
      <Prop>
        <label className="lbl">
          {t(m.date)}
          <input type="date" className="in" value={a.date ?? ''} disabled={!b.canEdit} onChange={(e) => e.target.value && setAttrs(c, { date: e.target.value })} />
        </label>
        <div className="row2">
          <label className="lbl">
            {t(m.starts)}
            <input type="time" className="in" value={a.start ?? ''} disabled={!b.canEdit} onChange={(e) => e.target.value && setAttrs(c, { start: e.target.value })} />
          </label>
          <label className="lbl">
            {t(m.ends)}
            <input type="time" className="in" value={a.end ?? ''} disabled={!b.canEdit} onChange={(e) => setAttrs(c, { end: e.target.value })} />
          </label>
        </div>
        <AttrSelect c={c} name="tz" label={t(m.timeZone)} options={zoneOptions(t, b.tz, tz, a.date ?? '')} />
        <AttrFlag c={c} name="local" label={t(m.readerTime)} hint={local} />
      </Prop>
      <Prop>
        <span className="t">{t(m.where)}</span>
        <div>
          <AttrSeg
            c={c}
            name="where"
            label={t(m.where)}
            fallback="online"
            items={[
              { value: 'online', label: t(m.online) },
              { value: 'venue', label: t(m.venue) },
            ]}
          />
        </div>
        {where === 'online' ? (
          <>
            <AttrText c={c} name="link" label={t(m.joinLink)} mono type="url" placeholder={'https://'} hint={t(m.joinHint)} />
            <AttrText c={c} name="platform" label={t(m.platform)} />
          </>
        ) : (
          <AttrText c={c} name="venue" label={t(m.venueAddress)} multiline />
        )}
      </Prop>
      <Prop>
        <AttrFlag c={c} name="calendar" label={t(m.addToCalendar)} hint={t(m.calendarHint)} />
      </Prop>
      <Prop>
        <AttrText c={c} name="field" label={t(m.remField)} mono hint={t(m.remFieldHint)} />
      </Prop>
      <ReminderCard />
    </>
  )
}

function FormPanel({ c }: { c: Component }) {
  const t = useT()
  const b = useBuilder()
  const canResume = useCan('forms:edit')
  const forms = useForms()
  const { resume: resumeForm, busy } = useResumeForm()
  void b.rev
  const id = attrOf(c, 'form')
  const f = forms.items.find((x) => x.id === id)
  const statusWord = (s: string) => (s === 'live' ? t(m.formLive) : s === 'paused' ? t(m.formPaused) : t(m.formDraft))
  const resume = async () => {
    if (f) await resumeForm(f.id, f.name)
  }
  return (
    <>
      <Prop>
        <AttrSelect
          c={c}
          name="form"
          label={t(m.formLabel)}
          id="pb-field-form"
          options={[{ value: '', label: t(m.pickForm) }, ...forms.items.map((x) => ({ value: x.id, label: t(m.formOption, { name: x.name, status: statusWord(x.status) }) }))]}
        />
        {forms.ready && !forms.items.length ? <span className="k">{t(m.noForms)}</span> : null}
        {id ? (
          <Link to="/w/$workspace/forms" params={{ workspace: b.slug }} search={{ form: id }} className="k pb-klink">
            {t(m.openForm)}
          </Link>
        ) : null}
        {f && f.status !== 'live' ? (
          <div className="note b" role="note">
            <UiIcon name="alert" size={16} color="#B91C1C" />
            <div className="flex min-w-0 flex-col gap-2">
              <span>{t(m.formPausedNote, { name: f.name })}</span>
              {canResume && b.canEdit ? (
                <span className="flex flex-wrap gap-1.5">
                  <button type="button" className="btn sm pri" aria-busy={busy || undefined} disabled={busy} onClick={() => void resume()}>
                    {t(m.resumeForm, { name: f.name })}
                  </button>
                  <button type="button" className="btn sm" onClick={() => document.getElementById('pb-field-form')?.focus()}>
                    {t(m.pickAnotherForm)}
                  </button>
                </span>
              ) : null}
            </div>
          </div>
        ) : null}
      </Prop>
      <Prop>
        <AttrText c={c} name="heading" label={t(m.formHeading)} />
        <AttrText c={c} name="hint" label={t(m.formHint)} multiline />
      </Prop>
    </>
  )
}

function VideoPanel({ c }: { c: Component }) {
  const t = useT()
  return (
    <Prop>
      <AttrText c={c} name="url" label={t(m.videoUrl)} type="url" mono placeholder={'https://www.youtube.com/watch?v=…'} hint={t(m.videoUrlHint)} invalid={(v) => (isVideoUrl(v) ? undefined : t(m.videoUrlBad))} />
      <AttrText c={c} name="title" label={t(m.videoTitle)} />
    </Prop>
  )
}

function PostsPanel({ c }: { c: Component }) {
  const t = useT()
  const tags = useTags()
  return (
    <Prop>
      <AttrSelect c={c} name="tag" label={t(m.postsTag)} options={[{ value: '', label: t(m.everyTag) }, ...tags.map((x) => ({ value: x.slug, label: x.name }))]} />
      <AttrSelect c={c} name="count" label={t(m.postsCount)} options={[1, 2, 3, 4, 5, 6].map((n) => ({ value: String(n), label: String(n) }))} />
      <AttrText c={c} name="heading" label={t(m.postsHeading)} />
    </Prop>
  )
}

function CheckList({ c, name, label, all, items }: { c: Component; name: string; label: string; all: string; items: readonly { id: string; label: string }[] }) {
  const b = useBuilder()
  const picked = parseList(attrOf(c, name))
  return (
    <fieldset className="pb-fieldset">
      <legend className="lbl">{label}</legend>
      <span className="k">{picked.length ? null : all}</span>
      {items.map((x) => (
        <label key={x.id} className="flex items-center gap-1.5" style={{ fontSize: 12.5 }}>
          <input
            type="checkbox"
            checked={picked.includes(x.id)}
            disabled={!b.canEdit}
            onChange={(e) => setAttrs(c, { [name]: joinList(e.target.checked ? [...picked, x.id] : picked.filter((id) => id !== x.id)) }, [name])}
          />
          {x.label}
        </label>
      ))}
    </fieldset>
  )
}

function ProductsPanel({ c }: { c: Component }) {
  const t = useT()
  const products = useProducts()
  const source = attrOf(c, 'source') || 'digital'
  return (
    <>
      <Prop>
        <span className="t">{t(m.productsSource)}</span>
        <div>
          <AttrSeg
            c={c}
            name="source"
            label={t(m.productsSource)}
            fallback="digital"
            items={[
              { value: 'digital', label: t(m.sourceDigital) },
              { value: 'store', label: t(m.sourceStore) },
            ]}
          />
        </div>
        {source === 'store' ? (
          <span className="k">{t(m.storeNote)}</span>
        ) : (
          <CheckList c={c} name="products" label={t(m.productsPick)} all={t(m.productsAll)} items={products.items.filter((p) => p.status === 'published').map((p) => ({ id: p.id, label: p.name }))} />
        )}
      </Prop>
      <Prop>
        <AttrSelect c={c} name="count" label={t(m.productsCount)} options={Array.from({ length: 12 }, (_, i) => ({ value: String(i + 1), label: String(i + 1) }))} />
        <AttrText c={c} name="button" label={t(m.buttonText)} />
      </Prop>
    </>
  )
}

function StorePanel() {
  const t = useT()
  const b = useBuilder()
  return (
    <div className="pb-pbody">
      <div className="note n" role="note">
        <UiIcon name="info" size={16} color="#4B5563" />
        <div className="flex min-w-0 flex-col gap-2">
          <span>{t(m.storeNote)}</span>
          <Link to="/w/$workspace/settings/integrations" params={{ workspace: b.slug }} className="btn sm self-start">
            {t(m.openIntegrations)}
          </Link>
        </div>
      </div>
    </div>
  )
}

function TiersPanel({ c }: { c: Component }) {
  const t = useT()
  const tiers = useTiers()
  return (
    <Prop>
      {tiers.length ? <CheckList c={c} name="tiers" label={t(m.tiersPick)} all={t(m.tiersAll)} items={tiers.map((x) => ({ id: x.id, label: x.name }))} /> : <span className="k">{t(m.noTiers)}</span>}
      <span className="t">{t(m.period)}</span>
      <div>
        <AttrSeg
          c={c}
          name="period"
          label={t(m.period)}
          fallback="month"
          items={[
            { value: 'month', label: t(m.monthly) },
            { value: 'year', label: t(m.yearly) },
          ]}
        />
      </div>
    </Prop>
  )
}

function ImagePanel({ c }: { c: Component }) {
  const t = useT()
  const b = useBuilder()
  void b.rev
  const src = attrOf(c, 'src') || String(c.get('src') ?? '')
  const alt = attrOf(c, 'alt')
  return (
    <Prop>
      <span className="t">{t(m.imageLabel)}</span>
      {src ? (
        <div className="pb-thumb">
          <img src={src} alt="" />
        </div>
      ) : null}
      {b.canEdit ? (
        <button
          type="button"
          className="btn sm self-start"
          onClick={() =>
            b.openDialog({
              kind: 'media',
              pick: (url) => {
                c.addAttributes({ src: url })
                c.set('src', url)
              },
            })
          }
        >
          {t(m.changeImage)}
        </button>
      ) : null}
      <AttrText c={c} name="alt" label={t(m.altText)} hint={alt.trim() ? t(m.altHint) : undefined} multiline id="pb-field-alt" />
      {alt.trim() ? null : <span className="err">{t(m.altMissing)}</span>}
    </Prop>
  )
}

function ButtonPanel({ c }: { c: Component }) {
  const t = useT()
  const b = useBuilder()
  void b.rev
  const a = tagOf(c) === 'a' ? c : elementChildren(c).find((x) => tagOf(x) === 'a')
  if (!a) return null
  const primary = !classesOf(a).includes('st-btn2')
  const text = (a.getEl()?.textContent ?? '').trim()
  return (
    <Prop>
      <label className="lbl">
        {t(m.buttonLabel)}
        <input
          className="in"
          defaultValue={text}
          disabled={!b.canEdit}
          onBlur={(e) => {
            const v = e.target.value.trim()
            if (v && v !== text) a.components(escapeText(v))
          }}
        />
      </label>
      <AttrText c={a} name="href" label={t(m.buttonLink)} mono type="url" />
      <span className="k">{t(m.buttonStyle)}</span>
      <div>
        <div className="seg" role="group" aria-label={t(m.buttonStyle)}>
          <button type="button" aria-pressed={primary} disabled={!b.canEdit} onClick={() => a.setClass(classesOf(a).map((x) => (x === 'st-btn2' ? 'st-btn' : x)))}>
            {t(m.primary)}
          </button>
          <button type="button" aria-pressed={!primary} disabled={!b.canEdit} onClick={() => a.setClass(classesOf(a).map((x) => (x === 'st-btn' ? 'st-btn2' : x)))}>
            {t(m.secondary)}
          </button>
        </div>
      </div>
    </Prop>
  )
}

/** Lists inside a block: add a question, a testimonial, an item. */
function ListTools({ section }: { section: Component }) {
  const t = useT()
  const b = useBuilder()
  void b.rev
  if (!b.canEdit || !b.pe) return null
  const find = (cls: string) => {
    let hit: Component | undefined
    section.onAll((x: Component) => {
      if (!hit && classesOf(x).includes(cls)) hit = x
    })
    return hit
  }
  const faq = find('faq')
  const quotes = find('quotes')
  const chapters = find('chapters')
  const bens = find('bens')
  const tools: ReactNode[] = []
  const add = (list: Component | undefined, html: string) => {
    if (!list || !b.pe) return
    const added = addItem(b.pe, list, html)
    if (added) b.select(added, { scroll: true })
  }
  if (faq) {
    const n = elementChildren(faq).filter((x) => tagOf(x) === 'details').length
    tools.push(
      <div key="faq" className="flex flex-wrap items-center justify-between gap-2">
        <span className="k">{t(m.questionsCount, { count: n })}</span>
        <button type="button" className="btn sm" onClick={() => add(faq, questionHtml(t(m.newQuestion), t(m.newAnswer)))}>
          {t(m.addQuestion)}
        </button>
      </div>,
    )
  }
  if (quotes) {
    const n = elementChildren(quotes).length
    tools.push(
      <div key="quotes" className="flex flex-wrap items-center justify-between gap-2">
        <span className="k">{t(m.quotesCount, { count: n })}</span>
        <button type="button" className="btn sm" onClick={() => add(quotes, quoteHtml({ quote: t(m.newQuote), quoteName: t(m.newQuoteName), quoteRole: t(m.newQuoteRole) }))}>
          {t(m.addQuote)}
        </button>
      </div>,
    )
  }
  const list = chapters ?? bens
  if (list) {
    const n = elementChildren(list).length
    tools.push(
      <div key="items" className="flex flex-wrap items-center justify-between gap-2">
        <span className="k">{t(m.itemsCount, { count: n })}</span>
        <button type="button" className="btn sm" onClick={() => add(list, featureItemHtml(chapters ? 'chapter' : 'benefit', n + 1, t(m.newItemTitle), t(m.newItemText)))}>
          {t(m.addItem)}
        </button>
      </div>,
    )
  }
  if (!tools.length) return null
  return <Prop>{tools}</Prop>
}

function TextPanel({ c }: { c: Component }) {
  const t = useT()
  const block = blockOf(c)
  const link = tagOf(c) === 'a' && !classesOf(c).some((x) => x === 'st-btn' || x === 'st-btn2')
  return (
    <Prop>
      <span className="t">{t(m.textTitle)}</span>
      <span className="k">{t(m.textHint)}</span>
      {block ? <span className="k">{t(m.textInBlock, { name: attrOf(block, 'data-pb-name') || kindName(t, kindOf(block)) })}</span> : null}
      {link ? <AttrText c={c} name="href" label={t(m.linkUrl)} mono type="url" /> : null}
    </Prop>
  )
}

/** The live block a section stands for (a product band, a signup band). */
function soleLiveOf(section: Component): Component | undefined {
  const sole = soleLive(nodeOf(section))
  return sole ? byCid(section, sole.key) : undefined
}

/**
 * The Block tab (PageBuilder · Block): the selected block's settings. A
 * section that only frames a live block shows the live block's; a text
 * inside a block says how to edit it.
 */
export function BlockTab() {
  const t = useT()
  const b = useBuilder()
  void b.rev
  const sel = b.selected
  // "Pick another form": the field the checks asked for gets the focus.
  useEffect(() => {
    if (!b.focusField) return
    const h = window.setTimeout(() => {
      document.getElementById(b.focusField!)?.focus()
      b.setFocusField(null)
    }, 60)
    return () => window.clearTimeout(h)
  }, [b.focusField, b])
  if (!sel || sel.get('type') === 'wrapper') {
    return (
      <div className="pb-pbody">
        <p className="k m-0">{t(m.noSelection)}</p>
      </div>
    )
  }
  const top = topOf(sel)
  const block = blockOf(sel) ?? sel
  const sole = top && block === top ? soleLiveOf(top) : undefined
  const target = liveTagOf(block) ? block : (sole ?? block)
  const tag = liveTagOf(target)
  const textPart = target !== sel && !liveTagOf(sel) && !isTop(sel) && !columnNumber(sel) && tagOf(sel) !== 'img'
  const remove = () => {
    const victim = sole && target === sole ? top! : target
    b.select(victim)
    b.removeSelected()
  }
  const showStyle = Boolean(top) && (!tag || tag === 'rc-product' || tag === 'rc-form')
  return (
    <div key={target.cid}>
      <BlockHead c={textPart ? sel : target} onRemove={textPart ? undefined : remove} />
      {textPart ? <TextPanel c={sel} /> : null}
      {tag === 'rc-product' ? <ProductPanel c={target} /> : null}
      {tag === 'rc-countdown' ? <CountdownPanel c={target} /> : null}
      {tag === 'rc-event' ? <EventPanel c={target} /> : null}
      {tag === 'rc-form' ? <FormPanel c={target} /> : null}
      {tag === 'rc-video' ? <VideoPanel c={target} /> : null}
      {tag === 'rc-posts' ? <PostsPanel c={target} /> : null}
      {tag === 'rc-products' ? <ProductsPanel c={target} /> : null}
      {tag === 'rc-store-product' ? <StorePanel /> : null}
      {tag === 'rc-tiers' ? <TiersPanel c={target} /> : null}
      {!tag && tagOf(target) === 'img' ? <ImagePanel c={target} /> : null}
      {!tag && kindOf(target) === 'button' && !textPart ? <ButtonPanel c={target} /> : null}
      {!tag && !textPart && (isTop(target) || attrOf(target, 'data-pb-name')) ? (
        <Prop>
          <AttrText c={target} name="data-pb-name" label={t(m.layerName)} />
        </Prop>
      ) : null}
      {top && !tag && !textPart ? <ListTools section={target} /> : null}
      {showStyle && top ? <StylePanel section={top} block={tag ? target : undefined} /> : null}
      {tag === 'rc-product' ? <StripeNote /> : null}
    </div>
  )
}
