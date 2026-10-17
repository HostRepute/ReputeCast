import { Fragment, useEffect, useId, useRef } from 'react'
import { formatList, useT } from '@/i18n'
import { clock, count } from '@/lib/format'
import { useComposer } from '@/features/composer/context'
import { applyFix } from '@/features/composer/fixes'
import { EyeOff } from './Canvas'
import { m } from './messages'
import { componentWithRule, ruleOf } from './rules'
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

/** The footer bar's "3 conditional blocks · 6 versions", which opens the versions. */
export function ConditionalFooter() {
  const t = useT()
  const c = useComposer()
  const cond = useConditional()
  if (!cond?.enabled || !cond.blocks || c.view !== 'visual') return null
  const r = cond.report
  const label =
    cond.counting && !r?.counted
      ? t(m.footerCounting, { blocks: cond.blocks })
      : r?.counted
        ? t(m.footerVersions, { blocks: cond.blocks, versions: r.versions.length })
        : t(m.footerBlocks, { blocks: cond.blocks })
  return (
    <button type="button" className="cv-foot" aria-expanded={cond.versionsOpen} aria-controls="cv-versions" onClick={() => cond.setVersionsOpen(!cond.versionsOpen)}>
      <EyeOff size={14} />
      {label}
    </button>
  )
}

/**
 * The versions (ComposerVisibility, VisibilityStates): what each version
 * shows, who gets it and its size, docked over the bottom of the canvas.
 */
export function VersionsPanel() {
  const t = useT()
  const c = useComposer()
  const cond = useConditional()
  const hid = useId()
  const close = useRef<HTMLButtonElement>(null)
  const panel = useRef<HTMLElement>(null)
  const open = !!cond?.enabled && cond.versionsOpen && cond.blocks > 0
  const setOpen = cond?.setVersionsOpen
  useEffect(() => {
    if (!open) return
    close.current?.focus()
    // Escape closes the panel while focus is in it.
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && panel.current?.contains(document.activeElement)) {
        e.stopPropagation()
        setOpen?.(false)
      }
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [open, setOpen])
  if (!cond || !open) return null
  const r = cond.report
  const blocks = r?.blocks.length ?? cond.blocks
  const versions = r?.versions ?? []
  const preview = (n: number) => {
    const ct = cond.contacts.find((x) => x.version === n)
    if (ct) cond.requestPreview(ct.id)
    cond.setVersionsOpen(false)
    c.setView('preview')
  }
  const canPreview = (n: number) => cond.contacts.some((x) => x.version === n)
  const shown = versions.length > 60 ? [...versions.slice(0, 3), ...versions.filter((v, i) => i >= 3 && (v.largest || v.smallest))] : versions
  const hiddenCount = versions.length - shown.length
  const rest = versions.slice(3).filter((v) => !v.largest && !v.smallest)
  const covered = versions.filter((v) => v.checked).reduce((n, v) => n + v.recipients, 0)
  // The selected block's versions are marked, as on the board.
  const selected = ruleOf(c.selected)
  const selectedInfo = selected ? cond.blockInfo(selected.id) : undefined
  const withSelected = new Set(selectedInfo?.versions ?? [])
  const over = versions.filter((v) => v.overClip)
  const minify = c.checks?.issues.find((i) => i.fixes.some((f) => f.kind === 'minify'))
  const selectBlock = (id: string) => {
    const root = c.editor?.getWrapper()
    const comp = root ? componentWithRule(root, id) : null
    if (comp) {
      c.editor?.select(comp)
      c.setRightTab('settings')
      cond.setVersionsOpen(false)
    }
  }
  return (
    <section ref={panel} id="cv-versions" className="cv-panel" aria-labelledby={hid}>
      <div className="cv-panel-h">
        <EyeOff size={16} />
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <h2 id={hid}>{r?.counted ? t(m.versionsTitle, { blocks, versions: versions.length }) : t(m.versionsCountingTitle, { blocks })}</h2>
          <span className="k">{cond.counting && !r?.counted ? t(m.versionsCounting) : r?.counted && r.countedAt ? t(m.versionsSubtitle, { time: clock(r.countedAt) }) : t(m.versionsNoAudience)}</span>
        </div>
        <button ref={close} type="button" className="btn sm ic" aria-label={t(m.closeVersions)} title={t(m.closeVersions)} onClick={() => cond.setVersionsOpen(false)}>
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
            <path d="M4 4l8 8M12 4l-8 8" />
          </svg>
        </button>
      </div>
      <div className="cv-panel-b">
        {r?.counted ? (
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
                    <tr className="xp">
                      <td colSpan={5} className="k">
                        {t(m.moreVersions, { count: hiddenCount, from: count(rest[0]!.recipients), to: count(rest[rest.length - 1]!.recipients) })}
                      </td>
                    </tr>
                  ) : null}
                  <tr className={v.overClip || v.empty ? 'hl' : withSelected.has(v.number) ? 'now' : undefined}>
                    <td className="font-semibold">{t(m.versionNumber, { number: v.number })}</td>
                    <td>{v.largest && versions.length > 50 ? t(m.largest, { shows: versionShows(t, v, blocks) }) : versionShows(t, v, blocks)}</td>
                    <td className="r num">{count(v.recipients)}</td>
                    <td className={`r num${v.overClip ? ' over' : ''}`}>
                      {t(m.kb, { kb: kbOf(v.sizeBytes) })}
                      {v.overClip ? (
                        <span className="badge warn sm" style={{ marginLeft: 6 }}>
                          {t(m.gmailClips)}
                        </span>
                      ) : null}
                    </td>
                    <td className="r">
                      <button type="button" className="btn sm" onClick={() => preview(v.number)} disabled={!canPreview(v.number)}>
                        {t(m.previewVersion)}
                      </button>
                    </td>
                  </tr>
                </Fragment>
              ))}
            </tbody>
          </table>
        ) : cond.counting ? (
          <div className="flex items-center gap-2" role="status" style={{ padding: '14px 16px' }}>
            <span className="spin" aria-hidden="true" />
            <span className="k">{t(m.versionsCounting)}</span>
          </div>
        ) : null}
      </div>
      {r?.counted ? (
        <div className="cv-panel-f">
          <div className="flex flex-wrap justify-between gap-x-6 gap-y-1">
            <span className="k">
              {r.missing > 0 ? t(m.missing, { missing: r.missing, possible: r.possible }) : null}
              {selectedInfo && withSelected.size ? ` ${t(m.rowsShow, { count: withSelected.size, rows: formatList([...withSelected].map(String)), name: selectedInfo.name })}` : null}
            </span>
            <span className="k">
              {[...r.blocks.filter((b) => b.words).map((b) => `${b.words} ${count(b.shown)}`), t(m.totals, { total: count(r.total) })].join(' · ')}
            </span>
          </div>
          {versions.length > 50 ? (
            <p className="k m-0">
              {t(m.sampled, { covered: count(covered), total: count(r.total), percent: `${r.total ? ((covered / r.total) * 100).toFixed(1) : '0'}%` })}
            </p>
          ) : null}
          {over.map((v) => {
            const block = v.shown.map((id) => cond.blockInfo(id)).find((b) => b && b.id === selected?.id) ?? cond.blockInfo(v.shown[0] ?? '')
            return (
              <div key={v.number} className="note w" role="status">
                <span className="flex flex-col gap-1">
                  <b>{t(m.overClipTitle, { number: v.number })}</b>
                  <span>{t(m.overClipBody, { number: v.number, shows: versionShows(t, v, blocks), kb: kbOf(v.sizeBytes), count: v.recipients })}</span>
                  <span className="flex flex-wrap gap-1.5" style={{ marginTop: 4 }}>
                    {minify && c.canEdit ? (
                      <button type="button" className="btn sm pri" onClick={() => void applyFix(c, minify, minify.fixes.find((f) => f.kind === 'minify')!)}>
                        {t(m.minifyCss)}
                      </button>
                    ) : null}
                    <button type="button" className="btn sm" disabled={!canPreview(v.number)} onClick={() => preview(v.number)}>
                      {t(m.previewVersionN, { number: v.number })}
                    </button>
                    {block && c.canEdit ? (
                      <button type="button" className="btn sm" onClick={() => selectBlock(block.id)}>
                        {t(m.shortenBlock, { name: block.name })}
                      </button>
                    ) : null}
                  </span>
                </span>
              </div>
            )
          })}
        </div>
      ) : null}
    </section>
  )
}
