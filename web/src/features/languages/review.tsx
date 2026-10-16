import { useId } from 'react'
import type { Schemas } from '@/api/client'
import { formatList, useT } from '@/i18n'
import { count } from '@/lib/format'
import { mr } from './messages'
import './languages.css'

type Check = Schemas['ReadinessCheck']
type Row = Schemas['ReadinessCheckRow']
type Translate = ReturnType<typeof useT>

/** The series colors of the versions (SendReadinessAccessibility › By version). */
export const versionColors = ['var(--chart-gmail)', 'var(--chart-outlook)', 'var(--chart-yahoo)', 'var(--chart-icloud)', 'var(--chart-tier-2)', 'var(--chart-ses)']
/** Readers who get another language's version. */
export const fallbackColor = 'var(--chart-other)'

/** The rows of the "Recipients per language" check: one per workspace language. */
export function languageRows(checks: Check[] | undefined): Row[] {
  return checks?.find((c) => c.key === 'languages.recipients')?.rows ?? []
}

export interface SentVersion {
  code: string
  name: string
}

/** The versions a send uses, in the rows' order (the source first). */
export function sentVersions(rows: Row[]): SentVersion[] {
  const out: SentVersion[] = []
  for (const r of rows) {
    if (r.gets && !out.some((v) => v.code === r.gets)) out.push({ code: r.gets, name: r.getsName || r.gets })
  }
  return out
}

/** The label, count and what they get, as the check lists them. */
export function CheckRowsGrid({ rows }: { rows: Row[] }) {
  return (
    <div className="lng-lb">
      {rows.map((r) => (
        <div key={r.label} className="contents">
          <span style={{ fontWeight: 600 }}>{r.label}</span>
          <span className="num" style={{ textAlign: 'right', fontWeight: 600 }}>
            {count(r.count)}
          </span>
          <span className="k ld">{r.detail}</span>
        </div>
      ))}
    </div>
  )
}

interface Share {
  key: string
  label: string
  count: number
  color: string
}

function shares(rows: Row[], t: Translate): Share[] {
  const versions = sentVersions(rows)
  const color = (code: string) => versionColors[versions.findIndex((v) => v.code === code) % versionColors.length] ?? versionColors[0]!
  return rows
    .filter((r) => r.count > 0)
    .map((r) => {
      const own = !r.language || r.language === r.gets
      return {
        key: r.language || r.label,
        label: own ? r.label : t(mr.readersGet, { language: r.label, gets: r.getsName || r.gets || '' }),
        count: r.count,
        color: own ? color(r.gets || '') : fallbackColor,
      }
    })
}

/**
 * Audience › By version: the recipients of each version, and the readers
 * who get another language's (SendReadinessAccessibility). Nothing when the
 * email has one language.
 */
export function ByVersion({ checks }: { checks: Check[] | undefined }) {
  const t = useT()
  const rows = languageRows(checks)
  if (sentVersions(rows).length < 2) return null
  const parts = shares(rows, t)
  const label = formatList(parts.map((p) => t(mr.shareLabel, { label: p.label, count: count(p.count) })))
  return (
    <div className="flex flex-col gap-2" style={{ paddingTop: 10, borderTop: '1px solid var(--line)' }}>
      <span className="k">{t(mr.byVersion)}</span>
      <div className="flex flex-col gap-2.5">
        <div className="lng-bar" role="img" aria-label={label}>
          {parts.map((p) => (
            <span key={p.key} style={{ flex: p.count, background: p.color }} />
          ))}
        </div>
        <div className="lng-legend">
          {parts.map((p) => (
            <span key={p.key}>
              <i style={{ background: p.color }} aria-hidden="true" />
              {t(mr.shareLabel, { label: p.label, count: count(p.count) })}
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}

/**
 * The confirmation step's line: "English, Spanish and German; Vietnamese
 * readers get English". Undefined for an email in one language.
 */
export function confirmLanguages(checks: Check[] | undefined, t: Translate): string | undefined {
  const rows = languageRows(checks)
  const versions = sentVersions(rows)
  if (versions.length < 2) return undefined
  const sent = formatList(versions.map((v) => v.name))
  const others = rows.filter((r) => r.count > 0 && r.language && r.gets && r.language !== r.gets)
  if (!others.length) return sent
  return t(mr.confirmWithOthers, {
    sent,
    others: others.map((r) => t(mr.readersGet, { language: r.label, gets: r.getsName || r.gets || '' })).join('; '),
  })
}

/** "One of each language" in the test-send select. */
export const EACH = '*'

/** The language of a test send: one version, or one of each. */
export function TestLanguageSelect({ versions, value, onChange }: { versions: SentVersion[]; value: string; onChange: (v: string) => void }) {
  const t = useT()
  const id = useId()
  return (
    <>
      <label className="vh" htmlFor={id}>
        {t(mr.testLanguage)}
      </label>
      <select id={id} className="in" style={{ flex: '1 1 150px' }} value={value} onChange={(e) => onChange(e.target.value)}>
        {versions.map((v) => (
          <option key={v.code} value={v.code}>
            {v.name}
          </option>
        ))}
        <option value={EACH}>{t(mr.eachLanguage)}</option>
      </select>
    </>
  )
}

/** "English, Spanish, German" after a test send's address. */
export function testLanguages(codes: string[] | undefined, versions: SentVersion[]): string {
  if (!codes?.length || versions.length < 2) return ''
  return codes.map((c) => versions.find((v) => v.code === c)?.name ?? c).join(', ')
}
