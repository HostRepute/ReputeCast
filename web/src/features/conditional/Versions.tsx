import { Fragment } from 'react'
import { Dialog } from '@/components/ui/Dialog'
import { useT } from '@/i18n'
import { clock, count } from '@/lib/format'
import { useComposer } from '@/features/composer/context'
import { m } from './messages'
import { useConditional, type VisibilityVersion } from './state'

function kbOf(bytes: number): string {
  return String(Math.round(bytes / 1024))
}

/** What a version shows: its blocks' names, or "none of the three". */
export function versionShows(t: ReturnType<typeof useT>, v: VisibilityVersion, blocks: number): string {
  if (v.empty) return t(m.emptyVersion)
  return v.shows || t(m.noneOf, { count: blocks })
}

/** "61 to 104 KB across the versions", for the size summary. */
export function useVersionSizes(): string | null {
  const t = useT()
  const cond = useConditional()
  const r = cond?.report
  if (!cond?.enabled || !r?.counted || r.versions.length < 2) return null
  return t(m.sizesAcross, { min: kbOf(r.minSize), max: kbOf(r.maxSize) })
}

/** The footer bar's "3 conditional blocks · 6 versions", which opens the table. */
export function ConditionalFooter() {
  const t = useT()
  const cond = useConditional()
  if (!cond?.enabled || !cond.blocks) return null
  const r = cond.report
  const label =
    cond.counting && !r?.counted
      ? t(m.footerCounting, { blocks: cond.blocks })
      : r?.counted
        ? t(m.footerVersions, { blocks: cond.blocks, versions: r.versions.length })
        : t(m.footerBlocks, { blocks: cond.blocks })
  return (
    <>
      <button type="button" className="cv-foot" onClick={() => cond.setVersionsOpen(true)}>
        {label}
      </button>
      <VersionsDialog />
    </>
  )
}

/** The versions table: what each version shows, who gets it and its size. */
export function VersionsDialog() {
  const t = useT()
  const c = useComposer()
  const cond = useConditional()
  if (!cond) return null
  const r = cond.report
  const blocks = r?.blocks.length ?? cond.blocks
  const versions = r?.versions ?? []
  const preview = (n: number) => {
    const ct = cond.contacts.find((x) => x.version === n)
    if (ct) cond.requestPreview(ct.id)
    cond.setVersionsOpen(false)
    c.setView('preview')
  }
  const shown = versions.length > 60 ? [...versions.slice(0, 3), ...versions.filter((v, i) => i >= 3 && (v.largest || v.smallest))] : versions
  const hiddenCount = versions.length - shown.length
  const rest = versions.slice(3).filter((v) => !v.largest && !v.smallest)
  const covered = versions.filter((v) => v.checked).reduce((n, v) => n + v.recipients, 0)
  return (
    <Dialog
      open={cond.versionsOpen}
      onOpenChange={cond.setVersionsOpen}
      size="lg"
      title={r?.counted ? t(m.versionsTitle, { blocks, versions: versions.length }) : t(m.versionsCountingTitle, { blocks })}
      description={cond.counting && !r?.counted ? t(m.versionsCounting) : r?.counted && r.countedAt ? t(m.versionsSubtitle, { time: clock(r.countedAt) }) : t(m.versionsNoAudience)}
    >
      {r?.counted ? (
        <div className="flex flex-col gap-3">
          <div className="overflow-x-auto">
            <table className="tbl cv-tbl" aria-label={t(m.versionsTable, { name: c.name })}>
              <thead>
                <tr>
                  <th scope="col">{t(m.colVersion)}</th>
                  <th scope="col">{t(m.colShows)}</th>
                  <th scope="col" className="r">
                    {t(m.colRecipients)}
                  </th>
                  <th scope="col" className="r">
                    {t(m.colSize)}
                  </th>
                  <th scope="col">
                    <span className="vh">{t(m.colAction)}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {shown.map((v, i) => (
                  <Fragment key={v.number}>
                    {hiddenCount > 0 && i === 3 && rest.length ? (
                      <tr key="more" className="xp">
                        <td colSpan={5} className="k">
                          {t(m.moreVersions, { count: hiddenCount, from: count(rest[0]!.recipients), to: count(rest[rest.length - 1]!.recipients) })}
                        </td>
                      </tr>
                    ) : null}
                    <tr className={v.overClip || v.empty ? 'hl' : undefined}>
                      <td className="nowrap font-semibold">{t(m.versionNumber, { number: v.number })}</td>
                      <td>{v.largest && versions.length > 50 ? t(m.largest, { shows: versionShows(t, v, blocks) }) : versionShows(t, v, blocks)}</td>
                      <td className="r num">{count(v.recipients)}</td>
                      <td className="r num nowrap">
                        {kbOf(v.sizeBytes)} KB{' '}
                        {v.overClip ? (
                          <span className="badge warn sm" style={{ marginLeft: 4 }}>
                            {t(m.gmailClips)}
                          </span>
                        ) : null}
                      </td>
                      <td>
                        <button type="button" className="btn sm" onClick={() => preview(v.number)} disabled={!cond.contacts.some((x) => x.version === v.number)}>
                          {t(m.previewVersion)}
                        </button>
                      </td>
                    </tr>
                  </Fragment>
                ))}
              </tbody>
            </table>
          </div>
          {r.missing > 0 ? <p className="k">{t(m.missing, { missing: r.missing, possible: r.possible })}</p> : null}
          <p className="k">{t(m.totals, { total: count(r.total) })}</p>
          {versions.length > 50 ? (
            <p className="k">
              {t(m.sampled, { covered: count(covered), total: count(r.total), percent: `${r.total ? ((covered / r.total) * 100).toFixed(1) : '0'}%` })}
            </p>
          ) : null}
        </div>
      ) : cond.counting ? (
        <div className="flex items-center gap-2" role="status">
          <span className="spin" aria-hidden="true" />
          <span className="k">{t(m.versionsCounting)}</span>
        </div>
      ) : null}
    </Dialog>
  )
}
