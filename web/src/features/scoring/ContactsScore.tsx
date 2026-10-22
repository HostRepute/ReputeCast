import { $api } from '@/api/client'
import { Field, Input, Select } from '@/components/ui/Field'
import { sourceName } from '@/features/growth/sourceText'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { bandColor, bandName, bandOrder, type Band } from './bands'
import { m } from './contacts.messages'
import './scoring.css'

/**
 * Scoring and Growth in Audience › Contacts (ScoringUses › Contacts): the
 * band filter, the sortable Score column, and the score, source, UTM and
 * referring-domain filters under More filters.
 */
export interface ScoreGrowthSearch {
  band?: Band | 'none'
  scoreFrom?: number
  scoreTo?: number
  growthSource?: string
  utmSource?: string
  utmMedium?: string
  utmCampaign?: string
  referringDomain?: string
  sort?: 'activity' | 'score'
}

/** Keys counted by "More filters · n". */
export const scoreGrowthMoreKeys = ['scoreFrom', 'scoreTo', 'growthSource', 'utmSource', 'utmMedium', 'utmCampaign', 'referringDomain'] as const

export function BandSelect({ value, onChange, className }: { value?: Band | 'none'; onChange: (v: Band | 'none' | undefined) => void; className?: string }) {
  const t = useT()
  return (
    <select className={className} aria-label={t(m.band)} value={value ?? ''} onChange={(e) => onChange((e.target.value || undefined) as Band | 'none' | undefined)}>
      <option value="">{t(m.anyBand)}</option>
      {[...bandOrder].reverse().map((b) => (
        <option key={b} value={b}>
          {bandName(b)}
        </option>
      ))}
      <option value="none">{t(m.noScore)}</option>
    </select>
  )
}

/** "91 Hot", or "—" without scoring. */
export function ScoreCell({ score, band }: { score?: number; band?: string }) {
  const t = useT()
  if (score === undefined || score === null) return <span className="k">{t(m.noScoreCell)}</span>
  const b = band as Band | undefined
  return (
    <span className="sc-cell">
      <b>{score}</b>
      {b ? (
        <span className="sc-band">
          <span className="sc-sw" style={{ background: bandColor[b] }} aria-hidden="true" />
          {bandName(b)}
        </span>
      ) : null}
    </span>
  )
}

/** The Score column's header: sorts by score, highest first, and back. */
export function ScoreHeader({ sorted, onSort }: { sorted: boolean; onSort: (score: boolean) => void }) {
  const t = useT()
  return (
    <th scope="col" aria-sort={sorted ? 'descending' : undefined}>
      <button type="button" className="sc-sortbtn" onClick={() => onSort(!sorted)} title={t(sorted ? m.sortByActivity : m.sortByScore)}>
        {t(m.score)}
        <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          {sorted ? <path d="M8 3v10M4 9l4 4 4-4" /> : <path d="M5 6l3-3 3 3M5 10l3 3 3-3" />}
        </svg>
      </button>
    </th>
  )
}

/** "48,213 contacts · sorted by score, highest first". */
export function useTotalLine() {
  const t = useT()
  return (total: number, byScore: boolean) => t(m.totalSorted, { count: total, sort: t(byScore ? m.sortedByScore : m.sortedByActivity) })
}

export type ScoreGrowthDraft = Pick<ScoreGrowthSearch, (typeof scoreGrowthMoreKeys)[number]>

/** The score range and how contacts first joined, inside More filters. */
export function ScoreGrowthFields({ draft, onChange, open }: { draft: ScoreGrowthDraft; onChange: (d: ScoreGrowthDraft) => void; open: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const sources = $api.useQuery('get', '/api/v1/workspaces/{workspace}/growth/sources', { params: { path: { workspace: slug } } }, { enabled: open })
  const num = (v: string) => (v === '' ? undefined : Math.max(0, Math.min(100, Math.round(Number(v)))))
  const text = (v: string) => v.trim() || undefined
  return (
    <>
      <div className="flex flex-col gap-1">
        <span className="sc-lbl">{t(m.scoreBetween)}</span>
        <div className="flex items-center gap-2">
          <Input className="num" type="number" min={0} max={100} style={{ width: 80 }} aria-label={t(m.lowest)} value={draft.scoreFrom ?? ''} onChange={(e) => onChange({ ...draft, scoreFrom: num(e.target.value) })} />
          <span className="k">{t(m.and)}</span>
          <Input className="num" type="number" min={0} max={100} style={{ width: 80 }} aria-label={t(m.highest)} value={draft.scoreTo ?? ''} onChange={(e) => onChange({ ...draft, scoreTo: num(e.target.value) })} />
        </div>
      </div>
      <Field label={t(m.growthSource)}>
        <Select value={draft.growthSource ?? ''} onChange={(e) => onChange({ ...draft, growthSource: e.target.value || undefined })}>
          <option value="">{t(m.anySource)}</option>
          {(sources.data?.items ?? []).map((s) => (
            <option key={s.source.key} value={s.source.key}>
              {t(m.sourceContacts, { name: sourceName(s.source), count: s.contacts })}
            </option>
          ))}
        </Select>
      </Field>
      <div className="grid grid-cols-3 gap-2">
        <Field label={t(m.utmSource)}>
          <Input value={draft.utmSource ?? ''} onChange={(e) => onChange({ ...draft, utmSource: text(e.target.value) })} />
        </Field>
        <Field label={t(m.utmMedium)}>
          <Input value={draft.utmMedium ?? ''} onChange={(e) => onChange({ ...draft, utmMedium: text(e.target.value) })} />
        </Field>
        <Field label={t(m.utmCampaign)}>
          <Input value={draft.utmCampaign ?? ''} onChange={(e) => onChange({ ...draft, utmCampaign: text(e.target.value) })} />
        </Field>
      </div>
      <Field label={t(m.referringDomain)}>
        <Input value={draft.referringDomain ?? ''} placeholder={'news.ycombinator.com'} onChange={(e) => onChange({ ...draft, referringDomain: text(e.target.value) })} />
      </Field>
      <span className="k">{t(m.moreHint)}</span>
    </>
  )
}

/** "and [80]": the upper end of "is between" in the segment builder (Score is between 50 and 80). */
export function BetweenTo({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const t = useT()
  return (
    <>
      <span className="k">{t(m.and)}</span>
      <input className="in w-[70px]" inputMode="numeric" aria-label={t(m.highest)} value={value} onChange={(e) => onChange(e.target.value)} />
    </>
  )
}

/** Chips for the score and growth filters (Growth links here with one set). */
export function ScoreGrowthChips({ search, onChange }: { search: ScoreGrowthSearch; onChange: (patch: Partial<ScoreGrowthSearch>) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const sources = $api.useQuery('get', '/api/v1/workspaces/{workspace}/growth/sources', { params: { path: { workspace: slug } } }, { enabled: Boolean(search.growthSource) })
  const chips: { key: string; label: string; clear: Partial<ScoreGrowthSearch> }[] = []
  if (search.growthSource) {
    const s = sources.data?.items.find((x) => x.source.key === search.growthSource)
    chips.push({ key: 'src', label: t(m.chipSource, { name: s ? sourceName(s.source) : search.growthSource }), clear: { growthSource: undefined } })
  }
  if (search.utmSource) chips.push({ key: 'us', label: t(m.chipUtmSource, { value: search.utmSource }), clear: { utmSource: undefined } })
  if (search.utmMedium) chips.push({ key: 'um', label: t(m.chipUtmMedium, { value: search.utmMedium }), clear: { utmMedium: undefined } })
  if (search.utmCampaign) chips.push({ key: 'uc', label: t(m.chipUtmCampaign, { value: search.utmCampaign }), clear: { utmCampaign: undefined } })
  if (search.referringDomain) chips.push({ key: 'rd', label: t(m.chipReferrer, { value: search.referringDomain }), clear: { referringDomain: undefined } })
  if (search.scoreFrom !== undefined || search.scoreTo !== undefined) {
    const label =
      search.scoreFrom !== undefined && search.scoreTo !== undefined
        ? t(m.chipScore, { from: search.scoreFrom, to: search.scoreTo })
        : search.scoreFrom !== undefined
          ? t(m.chipScoreFrom, { from: search.scoreFrom })
          : t(m.chipScoreTo, { to: search.scoreTo ?? 100 })
    chips.push({ key: 'sc', label, clear: { scoreFrom: undefined, scoreTo: undefined } })
  }
  if (!chips.length) return null
  return (
    <div className="flex w-full flex-wrap items-center gap-2">
      {chips.map((c) => (
        <span key={c.key} className="sc-chip">
          {c.label}
          <button type="button" aria-label={t(m.removeFilterNamed, { name: c.label })} onClick={() => onChange(c.clear)}>
            ×
          </button>
        </span>
      ))}
    </div>
  )
}
