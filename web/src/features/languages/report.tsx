import { keepPreviousData } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clock, count, shortDate } from '@/lib/format'
import { useOptionalWorkspace } from '@/lib/session'
import { mp } from './messages'
import './languages.css'

type Report = Schemas['CampaignReport']
type Langs = Schemas['ReportLanguages']
type Row = Schemas['ReportLanguageRow']
type Translate = ReturnType<typeof useT>

const pct1 = (v: number | null | undefined) => (v === null || v === undefined ? '—' : `${v.toFixed(1)}%`)
const pct2 = (v: number | null | undefined) => (v === null || v === undefined ? '—' : `${v.toFixed(2)}%`)

/** "English 27,396 · Spanish 2,740 · German 1,468": the Sent tile's line. */
export function sentByLanguage(l: Langs | null | undefined): string | undefined {
  if (!l?.sent.length) return undefined
  return l.sent.map((s) => `${s.name} ${count(s.count)}`).join(' · ')
}

/** Who got a version: "English, no language, other languages". */
function readersOf(r: Row, t: Translate): string {
  return r.kinds
    .map((k, i) => (k === 'no_language' ? t(mp.noLanguage) : k === 'not_offered' ? t(mp.otherLanguages) : (r.readers[i] ?? k)))
    .join(', ')
}

/** Under the version's name: "source", "translated from v9", "no Vietnamese version". */
function versionNote(r: Row, t: Translate): string {
  if (r.fallback) {
    const missing = r.missing || ''
    return r.reason === 'draft' ? t(mp.fallbackDraft, { language: missing }) : r.reason === 'held' ? t(mp.fallbackHeld, { language: missing }) : t(mp.fallbackMissing, { language: missing })
  }
  if (r.source) return t(mp.source)
  return r.translatedFrom ? t(mp.translatedFrom, { version: r.translatedFrom }) : t(mp.translated)
}

function versionName(r: Row, t: Translate): string {
  return r.fallback ? t(mp.asFallback, { language: r.name }) : r.name
}

/**
 * A fallback row's rate that stands out against the version's own readers:
 * opens 5 points lower, unsubscribes or complaints half again as high.
 */
function stands(r: Row, own: Row | undefined, key: 'humanOpenRate' | 'unsubscribeRate' | 'complaintRate'): boolean {
  if (!r.fallback || !own) return false
  const a = r[key]
  const b = own[key]
  if (a === undefined || a === null || b === undefined || b === null) return false
  return key === 'humanOpenRate' ? b - a >= 5 : a > 0 && a >= b * 1.5
}

function Rate({ value, warn }: { value: string; warn: boolean }) {
  return warn ? <span style={{ color: 'var(--warn-fg)', fontWeight: 700 }}>{value}</span> : <>{value}</>
}

/**
 * "Results per language" (ReportLanguages): after "Results per sending
 * provider", only for campaigns sent in more than one language. Readers
 * who got the fallback have their own row.
 */
export function ResultsPerLanguage({ r }: { r: Report }) {
  const t = useT()
  const ws = useOptionalWorkspace()
  const l = r.languages
  if (!l) return null
  const own = (row: Row) => l.rows.find((x) => !x.fallback && x.language === row.language)
  const caption = r.campaign.final
    ? t(mp.captionFinal)
    : t(mp.captionLive, { when: `${shortDate(new Date())} ${clock(new Date())}` })
  const fb = l.fallback
  const gapNote = fb ? fallbackNote(fb, l, t) : null
  return (
    <section className="card min-w-0" aria-labelledby="rep-lang-h" style={{ flex: '1 1 100%' }}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="rep-lang-h">
            {t(mp.title)}
          </h2>
          <span className="k">{caption}</span>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="badge mut" style={{ height: 22 }}>
            {t(mp.badge, { languages: l.languages, versions: l.versions })}
          </span>
        </div>
      </div>
      <div className="max-md:hidden" style={{ overflowX: 'auto' }}>
        <table className="tbl" aria-label={t(mp.title)}>
          <thead>
            <tr>
              <th scope="col">{t(mp.colVersion)}</th>
              <th scope="col">{t(mp.colReaders)}</th>
              <th scope="col" className="r">
                {t(mp.colSent)}
              </th>
              <th scope="col" className="r">
                {t(mp.colDelivered)}
              </th>
              <th scope="col" className="r">
                {t(mp.colOpens)}
              </th>
              <th scope="col" className="r">
                {t(mp.colClicks)}
              </th>
              <th scope="col" className="r">
                {t(mp.colUnsubscribes)}
              </th>
              <th scope="col" className="r">
                {t(mp.colComplaints)}
              </th>
            </tr>
          </thead>
          <tbody>
            {l.rows.map((row) => (
              <tr key={`${row.language}:${row.fallback ? row.missing : ''}`} className={row.fallback ? 'xp' : undefined}>
                <td>
                  <div className="flex flex-col gap-0.5" style={{ minWidth: 150 }}>
                    <span style={{ fontWeight: 600 }}>{versionName(row, t)}</span>
                    <span className="k">{versionNote(row, t)}</span>
                  </div>
                </td>
                <td>{readersOf(row, t)}</td>
                <td className="r num">{count(row.sent)}</td>
                <td className="r num">{pct1(row.deliveredRate)}</td>
                <td className="r num">
                  <Rate value={pct1(row.humanOpenRate)} warn={stands(row, own(row), 'humanOpenRate')} />
                </td>
                <td className="r num">{pct1(row.clickRate)}</td>
                <td className="r num">
                  <Rate value={pct2(row.unsubscribeRate)} warn={stands(row, own(row), 'unsubscribeRate')} />
                </td>
                <td className="r num">
                  <Rate value={pct2(row.complaintRate)} warn={stands(row, own(row), 'complaintRate')} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ul className="m-0 list-none p-0 md:hidden" aria-label={t(mp.title)}>
        {l.rows.map((row) => (
          <li key={`${row.language}:${row.fallback ? row.missing : ''}`} className="flex flex-col gap-1" style={{ padding: '12px 16px', borderTop: '1px solid var(--line-2)' }}>
            <div className="flex justify-between gap-2.5">
              <b style={{ fontWeight: 600 }}>{versionName(row, t)}</b>
              <span className="num" style={{ fontWeight: 600 }}>
                {count(row.sent)}
              </span>
            </div>
            <span className="k">
              {t(mp.phoneLine, {
                readers: row.fallback ? t(mp.readersOf, { language: readersOf(row, t) }) : readersOf(row, t),
                opens: pct1(row.humanOpenRate),
                clicks: pct1(row.clickRate),
              })}
            </span>
          </li>
        ))}
      </ul>
      <div className="pad" style={{ borderTop: '1px solid var(--line-2)' }}>
        <div className="flex flex-col gap-1.5" style={{ maxWidth: 640 }}>
          <span className="k">{t(mp.opensByVersion)}</span>
          <div className="flex flex-col gap-2.5">
            {l.rows.map((row) => (
              <div key={`${row.language}:${row.fallback ? row.missing : ''}`} className="flex flex-col gap-1">
                <div className="flex justify-between gap-2.5" style={{ fontSize: 13 }}>
                  <span>{row.fallback ? t(mp.barFallback, { version: versionName(row, t), readers: readersOf(row, t) }) : row.name}</span>
                  <span className="num" style={{ fontWeight: 600 }}>
                    {pct1(row.humanOpenRate)}
                  </span>
                </div>
                <div className="bar">
                  <i style={{ width: `${Math.min(100, (row.humanOpenRate ?? 0) * 2).toFixed(1)}%`, background: 'var(--accent)' }} />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
      {gapNote ? (
        <div className="lng-foot" style={{ justifyContent: 'flex-start', gap: '8px 14px' }}>
          <Icon name="info" size={16} />
          <span style={{ flex: '1 1 320px', minWidth: 0, color: 'var(--ink-nav)', fontSize: 13 }}>{gapNote}</span>
          {fb?.draft && ws ? (
            <Link
              to="/w/$workspace/campaigns/$campaignId/edit"
              params={{ workspace: ws.slug, campaignId: fb.draft.id }}
              search={{ language: fb.readers[0] } as never}
              className="btn sm"
            >
              {t(mp.addVersion, { language: nameIn(l, fb.readers[0] ?? ''), campaign: fb.draft.name })}
            </Link>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

/** A language's name as the rows give it. */
function nameIn(l: Langs, code: string): string {
  for (const r of l.rows) {
    if (r.fallback && r.kinds.includes(code)) return r.readers[r.kinds.indexOf(code)] ?? code
    if (r.language === code) return r.name
  }
  return code
}

/** "Vietnamese readers got English. Their human-open rate is 8.3 points below English readers’." */
function fallbackNote(fb: Schemas['ReportLanguageFallback'], l: Langs, t: Translate): string | null {
  if (!fb.readers.length) return null
  const readers = fb.readers.map((c) => nameIn(l, c)).join(', ')
  const got = l.rows.find((r) => r.language === fb.got)?.name ?? fb.got
  if (fb.gap === undefined || fb.gap === null || Math.abs(fb.gap) < 0.05) return t(mp.gotFallback, { readers, got })
  return t(mp.gotFallbackGap, { readers, got, gap: Math.abs(fb.gap).toFixed(1), below: fb.gap > 0 ? 'below' : 'above' })
}

/**
 * The click map with a language picker (ReportLanguages): it shows one
 * version at a time, the source first. Readers who got the fallback are in
 * that version's map.
 */
export function LanguageClickMap({
  r,
  campaignId,
  children,
}: {
  r: Report
  campaignId: string
  children: (m: Schemas['ReportClickMap'], picker: ReactNode, note: ReactNode) => ReactNode
}) {
  const t = useT()
  const ws = useOptionalWorkspace()
  const options = r.languages?.clickMaps ?? []
  const first = r.clickMap.language ?? options[0]?.language ?? ''
  const [lang, setLang] = useState(first)
  const other = Boolean(ws && options.length > 1 && lang !== first)
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report',
    { params: { path: { workspace: ws?.slug ?? '', campaignId }, query: { language: lang } } },
    { enabled: other, placeholderData: keepPreviousData },
  )
  if (!ws || options.length < 2) return <>{children(r.clickMap, null, null)}</>
  const m = other ? (q.data?.clickMap ?? r.clickMap) : r.clickMap
  const shown = other && q.data ? lang : first
  const picker = (
    <div className="seg" role="group" aria-label={t(mp.pickerLabel)}>
      {options.map((o) => (
        <button key={o.language} type="button" className={cn(o.language === lang && 'on')} aria-pressed={o.language === lang} onClick={() => setLang(o.language)}>
          {o.name}
        </button>
      ))}
    </div>
  )
  const fb = r.languages?.fallback
  const note =
    fb && fb.readers.length && fb.got === shown ? (
      <span className="k">{t(mp.clickMapFallback, { readers: fb.readers.map((c) => nameIn(r.languages!, c)).join(', '), got: options.find((o) => o.language === fb.got)?.name ?? fb.got })}</span>
    ) : null
  return <>{children(m, picker, note)}</>
}
