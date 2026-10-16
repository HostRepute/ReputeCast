import { Link } from '@tanstack/react-router'
import { useId, useMemo, useState } from 'react'
import { errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { toast } from '@/components/ui/Toast'
import { languageName, PSEUDO_LOCALE, regionOptions, resolvePreferences, T, useT, type MemberChoices } from '@/i18n'
import { formatDate, formatInt, formatMoney, formatPercent, formatTime, regionHourCycle, regionWeekStart, type DateParts, type HourCycle, type RegionFormat, type WeekStart } from '@/i18n/regions'
import { zonedParts } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { AppLanguageSelect, type LanguageOption } from './AppLanguageSelect'
import { m } from './messages'
import { FOLLOW, useMemberLanguage, useSaveMemberLanguage, type MemberLanguage } from './useLanguage'
import './language.css'

/** The translation guide ("Help translate ReputeCast"). */
export const TRANSLATE_GUIDE = 'https://github.com/hostrepute/reputecast/blob/main/docs/i18n/translating.md'

const ExternalIcon = () => (
  <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M9 3h4v4M13 3 7 9M11 9.5V13H3V5h3.5" />
  </svg>
)

/** The preview's sample: the latest Tuesday at 09:00, 48,213 contacts, 99.1%, $2.84. */
function sampleDay(): DateParts {
  const now = Date.now()
  for (let i = 0; i < 7; i++) {
    const p = zonedParts(now - i * 86_400_000)
    if (p.weekday === 2) return { ...p, hour: 9, minute: 0, second: 0 }
  }
  return { ...zonedParts(now), hour: 9, minute: 0, second: 0 }
}

/** "Tue 14 Oct 2026 · 09:00 · 48,213 · 99.1% · $2.84" in a region format and clock. */
export function previewLine(region: RegionFormat, hourCycle: HourCycle, day: DateParts = sampleDay()): string {
  return [formatDate(region, 'weekdayDate', day), formatTime(region, day, hourCycle), formatInt(region, 48213), formatPercent(region, 99.1), formatMoney(region, 2.84)].join(' · ')
}

const same = (a: MemberLanguage, b: MemberLanguage) => a.language === b.language && a.regionFormat === b.regionFormat && a.hourCycle === b.hourCycle && a.weekStart === b.weekStart

/** Account & security › Language & region (connect-and-reach brief §5.9, LanguageRegion board). */
export function LanguageRegionCard() {
  const t = useT()
  const { me: ws } = useWorkspace()
  const { choices: saved, inst, languages, pseudo } = useMemberLanguage()
  const save = useSaveMemberLanguage()
  const [draft, setDraft] = useState<MemberLanguage>()
  const [busy, setBusy] = useState(false)
  const ids = { lang: useId(), region: useId(), time: useId(), week: useId(), heading: useId() }
  const choices = draft ?? saved
  const dirty = !same(choices, saved)
  const eff = resolvePreferences(choices as MemberChoices, inst)
  // What "auto" would pick for each control, so choosing it again goes back to following.
  const derived = resolvePreferences({ ...choices, regionFormat: 'auto', hourCycle: 'auto', weekStart: 'auto' } as MemberChoices, inst)
  const day = useMemo(() => sampleDay(), [])

  const set = (patch: Partial<MemberLanguage>) => setDraft({ ...choices, ...patch })

  const options: LanguageOption[] = [
    { value: 'default', label: t(m.instanceDefault, { language: languageName(inst.defaultLanguage) }) },
    ...languages
      .filter((l) => l.code !== PSEUDO_LOCALE)
      .map((l) => ({ value: l.code, label: l.name, lang: l.code })),
  ]
  if (pseudo) options.push({ value: PSEUDO_LOCALE, label: languageName(PSEUDO_LOCALE), badge: t(m.pseudoBadge), description: t(m.pseudoDescription), separated: true })

  const submit = async () => {
    setBusy(true)
    const before = saved
    try {
      const languageChanged = resolvePreferences(before as MemberChoices, inst).locale !== eff.locale
      await save(choices)
      setDraft(undefined)
      const undo = async () => {
        try {
          await save(before)
        } catch (e) {
          toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) })
        }
      }
      // The app is in the new language now: the toast speaks it.
      toast({ kind: 'success', title: languageChanged ? t(m.switched, { language: languageName(eff.locale) }) : t(m.formatsSaved), action: { label: t(m.undo), onClick: () => void undo() }, duration: 10_000 })
    } catch (e) {
      toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  const untouched = !dirty && same(saved, FOLLOW)
  return (
    <section className="card" id="language" aria-labelledby={ids.heading}>
      <CardSection
        id={ids.heading}
        title={t(m.title)}
        caption={t(m.caption)}
        actions={
          untouched ? (
            <Badge tone="info" size="sm" className="text-[11px]">
              {t(m.newBadge)}
            </Badge>
          ) : undefined
        }
      />
      <div className="pad flex flex-col gap-3.5">
        <div className="grid gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(220px,100%),1fr))' }}>
          <label className="lbl" htmlFor={ids.lang}>
            {t(m.appLanguage)}
            <AppLanguageSelect id={ids.lang} label={t(m.appLanguage)} value={choices.language} options={options} onChange={(v) => set({ language: v as MemberLanguage['language'] })} />
          </label>
          <label className="lbl" htmlFor={ids.region}>
            {t(m.regionFormat)}
            <select
              id={ids.region}
              className="in"
              value={eff.regionFormat}
              disabled={eff.locale === PSEUDO_LOCALE}
              onChange={(e) => {
                const r = e.target.value as RegionFormat
                set({ regionFormat: r === derived.regionFormat ? 'auto' : r })
              }}
            >
              {regionOptions().map((r) => (
                <option key={r.code} value={r.code} lang={r.code === 'standard' ? 'en' : r.code}>
                  {r.name}
                </option>
              ))}
            </select>
            {choices.regionFormat === 'auto' ? <span className="k font-medium">{t(m.followsLanguage)}</span> : null}
          </label>
          <label className="lbl" htmlFor={ids.time}>
            {t(m.time)}
            <select
              id={ids.time}
              className="in"
              value={eff.hourCycle}
              onChange={(e) => {
                const hc = e.target.value as HourCycle
                set({ hourCycle: hc === regionHourCycle(eff.regionFormat) ? 'auto' : hc })
              }}
            >
              <option value="h23">{t(m.time24)}</option>
              <option value="h12">{t(m.time12)}</option>
            </select>
            {choices.hourCycle === 'auto' ? <span className="k font-medium">{t(m.fromRegion)}</span> : null}
          </label>
          <label className="lbl" htmlFor={ids.week}>
            {t(m.weekStartsOn)}
            <select
              id={ids.week}
              className="in"
              value={eff.weekStart}
              onChange={(e) => {
                const w = e.target.value as WeekStart
                set({ weekStart: w === regionWeekStart(eff.regionFormat) ? 'auto' : w })
              }}
            >
              <option value="monday">{t(m.monday)}</option>
              <option value="sunday">{t(m.sunday)}</option>
            </select>
            {choices.weekStart === 'auto' ? <span className="k font-medium">{t(m.fromRegion)}</span> : null}
          </label>
        </div>
        <div className="lang-preview" aria-live="polite">
          <span className="k">{t(m.preview)}</span>
          <span className="v" lang={eff.regionFormat === 'standard' ? 'en' : eff.regionFormat}>
            {previewLine(eff.regionFormat, eff.hourCycle, day)}
          </span>
        </div>
        <span className="k" style={{ color: 'var(--ink-nav)' }}>
          <T
            msg={m.timesNote}
            values={{
              workspace: ws.workspace.name,
              timezone: ws.timezone,
              link: (c) => (
                <Link to="/w/$workspace/settings" params={{ workspace: ws.workspace.slug }}>
                  {c}
                </Link>
              ),
            }}
          />
        </span>
      </div>
      <div className="lang-foot">
        <a href={TRANSLATE_GUIDE} className="lnk" target="_blank" rel="noreferrer">
          {t(m.helpTranslate)}
          <ExternalIcon />
          <span className="vh">{t(m.opensInNewTab)}</span>
        </a>
        <span className="flex items-center gap-2">
          {dirty ? (
            <Badge tone="warn" size="sm" className="text-[11px]">
              {t(m.notSaved)}
            </Badge>
          ) : null}
          <Button variant="primary" size="sm" disabled={!dirty || busy} loading={busy} onClick={() => void submit()}>
            {t(m.save)}
          </Button>
        </span>
      </div>
    </section>
  )
}
