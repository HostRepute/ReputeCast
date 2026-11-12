import type { Component } from 'grapesjs'
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { errorMessage } from '@/api/client'
import { Segmented } from '@/components/ui/Tabs'
import { ContrastColorField } from '@/features/accessibility/composer/ColorPicker'
import '@/features/accessibility/composer/accessibility.css'
import { m as ma } from '@/features/accessibility/composer/messages'
import { formatRatio, parseColor, ratio } from '@/features/accessibility/contrast'
import { useBrandKit } from '@/features/composer/BrandKit'
import { useComposer } from '@/features/composer/context'
import { versionLanguage } from '@/features/languages/composerParts'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { applyCountdown, COUNTDOWN_SIZES, countdownImage, countdownOf, newCountdown, type CountdownSettings } from './block'
import { m } from './messages'
import { inZone, zonedMoment } from './time'

type Field = 'background' | 'digits' | 'labels'

function city(zone: string): string {
  const last = zone.split('/').pop() ?? zone
  return last === 'Ho_Chi_Minh' ? 'Hanoi' : last.replace(/_/g, ' ')
}

function zones(): string[] {
  try {
    return (Intl as unknown as { supportedValuesOf(k: string): string[] }).supportedValuesOf('timeZone')
  } catch {
    return ['UTC', 'America/New_York', 'Europe/Berlin', 'Asia/Ho_Chi_Minh']
  }
}

function Prop({ title, children, id }: { title: ReactNode; children: ReactNode; id: string }) {
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {title}
      </span>
      {children}
    </div>
  )
}

/** "17.9:1 · Passes AA" beside what it compares. */
function RatioChip({ a, b, what }: { a: string; b: string; what: string }) {
  const t = useT()
  const ca = parseColor(a)
  const cb = parseColor(b)
  if (!ca || !cb) return null
  const r = ratio(ca, cb)
  const label = r >= 4.5 ? ma.chipAA : ma.chipFails
  return (
    <div className="flex items-center gap-2">
      <span className={`a11y-cchip ${r >= 4.5 ? 'p' : 'f'}`}>{t(label, { ratio: formatRatio(r) })}</span>
      <span className="k">{what}</span>
    </div>
  )
}

/**
 * The Countdown block's inspector (ComposerTimedBlocks): when it ends and
 * where, the units, the size, the colours (the digits need 4.5:1 on the
 * background), what it shows at the end and its alt text. Each change asks
 * the server for the image of the new settings (500 ms after the last one)
 * and swaps the block's src; the labels follow the language being edited.
 */
export function CountdownPanel({ c }: { c: Component }) {
  const t = useT()
  const composer = useComposer()
  const { slug, me } = useWorkspace()
  const kit = useBrandKit().data
  const appLang = document.documentElement.lang || 'en'
  const editing = versionLanguage(composer.campaign) ?? composer.campaign.sourceLanguage ?? 'en'
  const zone = me.timezone || 'UTC'
  const initial = useMemo(() => countdownOf(c) ?? newCountdown(zone, editing), [c, zone, editing])
  const [s, setS] = useState<CountdownSettings>(initial)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [field, setField] = useState<Field>('background')
  const attrs = c.getAttributes() as Record<string, string>
  const [alt, setAlt] = useState(attrs.alt ?? '')
  // The alt text the server wrote last: the reader's own differs from it.
  const [autoAlt, setAutoAlt] = useState(attrs.alt ?? '')
  const own = alt !== '' && alt !== autoAlt
  const timer = useRef<number | undefined>(undefined)
  const kept = useRef(initial.digits)

  // The labels follow the language version being edited.
  const language = (['en', 'de', 'es', 'fr', 'vi'].includes(editing) ? editing : 'en') as CountdownSettings['language']

  const push = (next: CountdownSettings) => {
    window.clearTimeout(timer.current)
    const bg = parseColor(next.background)
    const dg = parseColor(next.digits)
    if (bg && dg && ratio(dg, bg) < 4.5) {
      setError(t(m.refused, { kept: kept.current }))
      return
    }
    setError('')
    timer.current = window.setTimeout(() => {
      setBusy(true)
      countdownImage(slug, next)
        .then((img) => {
          applyCountdown(c, next, img)
          setAutoAlt(img.alt)
          kept.current = next.digits
          if (own) c.addAttributes({ alt })
          else setAlt(img.alt)
        })
        .catch((e: unknown) => setError(t(m.failed, { reason: errorMessage(e) })))
        .finally(() => setBusy(false))
    }, 500)
  }
  const change = (patch: Partial<CountdownSettings>) => {
    const next = { ...s, ...patch }
    setS(next)
    push(next)
  }
  useEffect(() => () => window.clearTimeout(timer.current), [])
  // The labels follow the language version being edited: a block made in
  // another version is drawn again in this one.
  useEffect(() => {
    if (initial.language === language) return
    const id = window.setTimeout(() => change({ language }), 0)
    return () => window.clearTimeout(id)
    // Only when the language version changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [language])
  const [date, time] = s.endsAt.split('T') as [string, string]
  const end = zonedMoment(s.endsAt, s.timeZone)
  const others = ['Europe/Berlin', 'Asia/Ho_Chi_Minh', 'America/New_York'].filter((z) => z !== s.timeZone).slice(0, 2) as [string, string]
  const [w, h] = COUNTDOWN_SIZES[s.size]
  const langName = (() => {
    try {
      return new Intl.DisplayNames([appLang], { type: 'language' }).of(language) ?? language
    } catch {
      return language
    }
  })()
  const brand: { name: string; color: string }[] = kit
    ? [
        { name: t(m.kitButton), color: kit.colors.button },
        { name: t(m.kitText), color: kit.colors.text },
        { name: t(m.kitBackground), color: kit.colors.background },
      ]
    : []
  const colour = (f: Field, label: string) => {
    const spec =
      f === 'background'
        ? { field: 'background' as const, texts: [{ color: s.digits, sizePx: 13, bold: false, kind: 'text' as const }, { color: s.labels, sizePx: 13, bold: false, kind: 'text' as const }], others: [] }
        : { field: 'text' as const, text: { sizePx: 13, bold: false, kind: 'text' as const }, background: s.background, others: [] }
    return (
      <div className="flex items-center gap-2" onFocusCapture={() => setField(f)}>
        <span className="k w-[78px] flex-none">{label}</span>
        <div className="min-w-0 flex-1">
          <ContrastColorField value={s[f]} label={label} spec={spec} onCommit={(v) => change({ [f]: v.toUpperCase() } as Partial<CountdownSettings>)} />
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      <div className="flex flex-col gap-0.5 border-b border-line-2 px-3 py-3">
        <span>
          <b className="font-semibold">{t(m.name)}</b> <span className="k">{end ? t(m.ends, { when: inZone(end, s.timeZone, appLang) }) : ''}</span>
        </span>
        <span className="k">{t(m.labelsFollow, { language: langName })}</span>
      </div>
      <Prop title={t(m.endsLabel)} id="cd-ends">
        <div className="row2">
          <input className="in" type="date" aria-label={t(m.endsDate)} value={date} onChange={(e) => e.target.value && change({ endsAt: `${e.target.value}T${time}` })} />
          <input className="in" type="time" aria-label={t(m.endsTime)} value={time} onChange={(e) => e.target.value && change({ endsAt: `${date}T${e.target.value}` })} />
        </div>
      </Prop>
      <Prop title={t(m.zone)} id="cd-zone">
        <select className="in" value={s.timeZone} aria-labelledby="cd-zone" onChange={(e) => change({ timeZone: e.target.value })}>
          <option value={zone}>{t(m.zoneDefault, { zone })}</option>
          {zones()
            .filter((z) => z !== zone)
            .map((z) => (
              <option key={z} value={z}>
                {z}
              </option>
            ))}
        </select>
        {end ? (
          <span className="k">
            {t(m.sameMoment, {
              first: inZone(end, others[0], appLang, false),
              firstCity: city(others[0]),
              second: inZone(end, others[1], appLang, false).split(', ').pop() ?? '',
              secondCity: city(others[1]),
            })}
          </span>
        ) : null}
      </Prop>
      <Prop title={t(m.show)} id="cd-units">
        <div className="flex flex-col gap-1.5">
          {(['dhms', 'hms', 'dhm'] as const).map((u) => (
            <label key={u} className="flex items-center gap-2 text-[13px]">
              <input type="radio" name="cd-units" checked={s.units === u} onChange={() => change({ units: u })} />
              {t(m[u])}
            </label>
          ))}
        </div>
      </Prop>
      <Prop title={t(m.size)} id="cd-size">
        <Segmented<CountdownSettings['size']>
          label={t(m.size)}
          value={s.size}
          onChange={(v) => change({ size: v })}
          className="flex w-full [&>button]:flex-1 [&>button]:justify-center"
          items={[
            { value: 's', label: 'S' },
            { value: 'm', label: 'M' },
            { value: 'l', label: 'L' },
          ]}
        />
        <span className="k">{s.size === 'l' ? t(m.sizeL) : s.size === 'm' ? t(m.sizeM) : t(m.sizeS)}</span>
      </Prop>
      <Prop title={t(m.colours)} id="cd-colours">
        <div className="flex flex-col gap-2">
          {colour('background', t(m.background))}
          {colour('digits', t(m.digits))}
          {colour('labels', t(m.labels))}
        </div>
        {error ? (
          <span className="text-[12.5px] text-bad-fg" role="alert">
            {error}
          </span>
        ) : null}
        {brand.length ? (
          <>
            <span className="k">{t(m.brandKit)}</span>
            <div className="flex flex-wrap gap-1.5">
              {brand.map((b) => (
                <button
                  key={b.name}
                  type="button"
                  className="chip"
                  aria-label={t(m.applyBrand, { name: b.name, field: t(m[field]) })}
                  onClick={() => change({ [field]: b.color.toUpperCase() } as Partial<CountdownSettings>)}
                >
                  <span className="sw" style={{ background: b.color, width: 14, height: 14, borderRadius: 4 }} aria-hidden="true" />
                  {b.name}
                </button>
              ))}
            </div>
          </>
        ) : null}
        <RatioChip a={s.digits} b={s.background} what={t(m.digitsOn)} />
        <RatioChip a={s.labels} b={s.background} what={t(m.labelsOn)} />
      </Prop>
      <Prop title={t(m.whenEnds)} id="cd-end">
        <div className="flex flex-col gap-1.5">
          <label className="flex items-center gap-2 text-[13px]">
            <input type="radio" name="cd-end" checked={s.ended.mode === 'zeros'} onChange={() => change({ ended: { mode: 'zeros' } })} />
            {t(m.showZeros)}
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="radio" name="cd-end" checked={s.ended.mode === 'text'} onChange={() => setS({ ...s, ended: { mode: 'text', text: s.ended.text ?? '' } })} />
            {t(m.showMessage)}
          </label>
        </div>
        {s.ended.mode === 'text' ? (
          <>
            <input
              className="in"
              aria-label={t(m.message)}
              maxLength={60}
              value={s.ended.text ?? ''}
              onChange={(e) => {
                const next = { ...s, ended: { mode: 'text' as const, text: e.target.value } }
                setS(next)
                if (e.target.value.trim()) push(next)
              }}
            />
            <span className="k">{t(m.messageCount, { count: (s.ended.text ?? '').length })}</span>
          </>
        ) : null}
      </Prop>
      <Prop title={t(m.alt)} id="cd-alt">
        <input
          className="in"
          aria-labelledby="cd-alt"
          value={alt}
          onChange={(e) => setAlt(e.target.value)}
          onBlur={() => {
            const v = alt.trim() || autoAlt
            setAlt(v)
            c.addAttributes({ alt: v })
          }}
        />
        <span className="k">{own ? t(m.altOwn) : t(m.altAuto)}</span>
      </Prop>
      <div className="flex items-start gap-2 px-3 py-3">
        <span className="a11y-cchip p">{t(m.pass)}</span>
        <span className="k">{busy ? t(m.making) : t(m.passNote, { width: w, height: h })}</span>
      </div>
    </div>
  )
}
