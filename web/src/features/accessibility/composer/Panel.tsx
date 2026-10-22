import { Link } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Segmented } from '@/components/ui/Tabs'
import { formatList, useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { pathOf } from '@/features/composer/editor/labels'
import { componentAt, reveal } from '@/features/composer/fixes'
import { AltTextAssist, LinkTextAssist } from '@/features/ai/composer/fixes'
import { clock, dayMonthYear, shortDate } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { CodeText } from '../review'
import { m } from './messages'
import { useAccessibility, type Issue, type Report } from './state'
import './accessibility.css'

type Severity = 'all' | 'serious' | 'moderate'

/** "English", from the report's languages. */
export function languageName(report: Report, code: string): string {
  return report.languages.find((l) => l.code === code)?.name ?? code
}

/** The version on the canvas: the source until another language is edited. */
export function currentLanguage(report: Report, language?: string): string {
  return language ?? report.languages.find((l) => l.source)?.code ?? report.languages[0]?.code ?? 'en'
}

/** What an image issue is about, in the app's language: "The hero image" (the server names it). */
function imageName(issue: Issue): string {
  return issue.name ?? issue.title.replace(/ has no alt text$/, '').replace(/ has no text$/, '')
}

const sameTime = (d: string) => shortDate(d) === shortDate(new Date())

/** The Accessibility tab (5.11): issues as you edit, their fixes, what passed. */
export function AccessibilityPanel({ language }: { language?: string }) {
  const a = useAccessibility()
  const c = useComposer()
  const t = useT()
  const me = useMe().data?.user
  const role = useWorkspace().me.workspace.roleName
  const [severity, setSeverity] = useState<Severity>('all')
  const [scope, setScope] = useState<'current' | 'all'>('all')
  const report = a?.report
  if (!a) return null
  if (a.error) return <ErrorState />
  if (!report) return <CheckingRow first />
  const lang = currentLanguage(report, language)
  const names = formatList(report.languages.map((l) => l.name))
  const serious = report.issues.filter((i) => i.severity === 'serious')
  const moderate = report.issues.filter((i) => i.severity === 'moderate')
  const shown = report.issues.filter(
    (i) => (severity === 'all' || i.severity === severity) && (scope === 'all' || report.languages.length < 2 || i.languages.includes(lang)),
  )
  const access = c.campaign.access
  const html = report.mode === 'html'
  const blocks = report.policy === 'block'
  const canFix = c.canEdit && !html && c.variant === 'A'
  const summary =
    report.issues.length === 0
      ? t(m.noIssues)
      : !moderate.length
        ? t(m.summarySerious, { count: serious.length })
        : !serious.length
          ? t(m.summaryModerate, { count: moderate.length })
          : t(m.summaryBoth, { count: report.issues.length, serious: serious.length, moderate: moderate.length })
  const p = a.policy
  let note: ReactNode = null
  if (!c.canEdit && access.editable) {
    const helpers = access.editHelpers ?? []
    note = (
      <PanelNote>
        {helpers.length
          ? t(m.readOnlyAsk, { name: me?.name ?? '', role, helpers: formatList(helpers, 'disjunction') })
          : t(m.readOnly, { name: me?.name ?? '', role })}
      </PanelNote>
    )
  } else if (!access.editable) {
    note = <PanelNote>{t(m.sent)}</PanelNote>
  } else if (html) {
    note = <PanelNote>{t(m.htmlEmail)}</PanelNote>
  } else if (c.variant !== 'A') {
    note = <PanelNote>{t(m.variant)}</PanelNote>
  } else if (report.issues.length && (serious.length || blocks)) {
    note = (
      <PanelNote bad={blocks}>
        {blocks
          ? p?.updatedBy && p.updatedAt
            ? sameTime(p.updatedAt)
              ? t(m.policyBlockBy, { name: p.updatedBy, time: clock(p.updatedAt) })
              : t(m.policyBlockOn, { name: p.updatedBy, date: dayMonthYear(p.updatedAt) })
            : t(m.policyBlock)
          : t(m.policyWarn)}{' '}
        <Link to="/w/$workspace/settings" params={{ workspace: c.slug }} hash="accessibility">
          {t(m.policyLink)}
        </Link>
      </PanelNote>
    )
  }
  return (
    <div className="a11y-panel">
      <div className="a11y-head">
        <span className="font-semibold">{summary}</span>
        <span className="k">
          {report.issues.length
            ? t(m.checkedLine, { languages: names, time: clock(report.checkedAt) })
            : t(m.noIssuesDetail, { name: c.name || c.campaign.name, languages: names })}
        </span>
        {note}
      </div>
      {a.checking ? <CheckingRow /> : null}
      {report.issues.length ? (
        <div className="a11y-filters">
          <Segmented
            label={t(m.filterLabel)}
            value={severity}
            onChange={setSeverity}
            items={[
              { value: 'all', label: t(m.filterAll, { count: report.issues.length }) },
              { value: 'serious', label: t(m.filterSerious, { count: serious.length }) },
              { value: 'moderate', label: t(m.filterModerate, { count: moderate.length }) },
            ]}
          />
          {report.languages.length > 1 ? (
            <Segmented
              label={t(m.languageLabel)}
              value={scope}
              onChange={setScope}
              items={[
                { value: 'current', label: languageName(report, lang) },
                { value: 'all', label: t(m.allLanguages) },
              ]}
            />
          ) : null}
        </div>
      ) : null}
      <div className="a11y-list">
        {shown.map((i) => (
          <IssueRow key={i.id} issue={i} report={report} lang={lang} canFix={canFix} html={html} blocks={blocks} />
        ))}
        {report.issues.length && !shown.length ? <p className="k a11y-empty">{t(m.filteredOut)}</p> : null}
      </div>
      {a.fixed.length ? <FixedList /> : null}
      <Passing report={report} open={!report.issues.length} />
    </div>
  )
}

/** A note of the panel: the board's note with its icon. */
function PanelNote({ bad, children }: { bad?: boolean; children: ReactNode }) {
  return (
    <div className={bad ? 'note b' : 'note n'} role={bad ? 'alert' : 'note'}>
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke={bad ? '#B91C1C' : '#4B5563'} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <circle cx="8" cy="8" r="6.2" />
        <path d={bad ? 'M8 4.8v3.6M8 10.9v.1' : 'M8 7.2v3.8M8 5v.1'} />
      </svg>
      <div className="min-w-0 flex-[1_1_160px]">{children}</div>
    </div>
  )
}

function CheckingRow({ first }: { first?: boolean }) {
  const a = useAccessibility()
  const t = useT()
  const report = a?.report
  const names = report ? formatList(report.languages.map((l) => l.name)) : ''
  return (
    <div className="a11y-checking" role="status">
      <span className="spin" aria-hidden="true" />
      <span className="flex flex-col gap-0.5">
        <span className="font-semibold">{names ? t(m.checking, { languages: names }) : t(m.checkingEmail)}</span>
        <span className="k">{!first && a?.editedAt ? t(m.checkingAfter, { time: clock(a.editedAt) }) : t(m.checkingFirst)}</span>
      </span>
    </div>
  )
}

function ErrorState() {
  const a = useAccessibility()
  const t = useT()
  if (!a?.error) return null
  return (
    <div className="a11y-panel">
      <div className="a11y-error" role="alert">
        <span className="font-semibold">{t(m.errorTitle)}</span>
        <span className="k">{a.last ? t(m.errorDetailLast, { time: clock(a.last.at), count: a.last.count }) : t(m.errorDetail)}</span>
        <div className="flex flex-wrap items-center gap-2.5">
          <Button size="sm" onClick={a.recheck} loading={a.checking}>
            {t(m.tryAgain)}
          </Button>
          {a.error.reference ? <span className="k mono">{t(m.reference, { ref: a.error.reference })}</span> : null}
        </div>
      </div>
    </div>
  )
}

function IssueRow({ issue, report, lang, canFix, html, blocks }: { issue: Issue; report: Report; lang: string; canFix: boolean; html: boolean; blocks: boolean }) {
  const a = useAccessibility()!
  const c = useComposer()
  const t = useT()
  const serious = issue.severity === 'serious'
  const selected = useMemo(() => (c.selected ? pathOf(c.selected).join('-') : ''), [c.selected])
  const on = !!selected && (issue.paths.length ? issue.paths : [issue.path]).some((p) => p.join('-') === selected)
  const show = () => {
    if (html) {
      c.setView('html')
      return
    }
    reveal(c, issue.path)
  }
  return (
    <div className={`a11y-iss${on ? ' on' : ''}${serious ? ' is-serious' : ''}`}>
      <div className="a11y-t">
        <Badge tone={serious ? 'bad' : 'warn'}>{serious ? (blocks ? t(m.seriousBlocks) : t(m.serious)) : t(m.moderate)}</Badge>
        <span className="font-semibold">{issue.title}</span>
      </div>
      <div className="a11y-in">
        <span className="k">{t(m.in)}</span>
        {issue.languages.map((code) => (
          <span key={code} className="a11y-lang">
            {languageName(report, code)}
          </span>
        ))}
        {issue.path.length || html ? (
          <button type="button" className="a11y-show" onClick={show}>
            <TargetIcon />
            {html ? t(m.showInCode) : t(m.showOnCanvas)}
          </button>
        ) : null}
      </div>
      {issue.location.length ? <span className="a11y-loc mono">{issue.location.join(' › ')}</span> : null}
      {issue.kind === 'link_text' && issue.detail ? <span className="k">{issue.detail}</span> : null}
      {issue.kind === 'contrast' && report.languages.length > 1 ? <span className="k">{t(m.colorShared)}</span> : null}
      {canFix && !a.busy ? <Fixes issue={issue} report={report} lang={lang} /> : null}
      {canFix && a.busy === issue.id ? <span className="spin" aria-label={t(m.checkingFirst)} /> : null}
    </div>
  )
}

function TargetIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
      <circle cx="8" cy="8" r="4.5" />
      <path d="M8 1.5v2.5M8 12v2.5M1.5 8H4M12 8h2.5" />
    </svg>
  )
}

/** The fix of one issue: words, a color, or one click. */
function Fixes({ issue, report, lang }: { issue: Issue; report: Report; lang: string }) {
  const a = useAccessibility()!
  const c = useComposer()
  const t = useT()
  const [text, setText] = useState('')
  const [own, setOwn] = useState(false)
  const f = issue.fix
  const language = languageName(report, lang)
  const others = issue.languages.filter((x) => x !== lang).map((x) => languageName(report, x))
  switch (f.kind) {
    case 'alt_text': {
      const name = imageName(issue)
      return (
        <div className="a11y-fix">
          <AltTextAssist
            component={componentAt(c, issue.path)}
            src={issue.element.src ?? ''}
            onAccept={(value) => a.fix(issue, { kind: 'alt_text', path: issue.path, index: issue.element.index, value }, t(m.fixedAlt, { name }))}
            onEdit={(value) => {
              setText(value)
              window.setTimeout(() => document.getElementById(`alt-${issue.id}`)?.focus(), 30)
            }}
          />
          <form
            className="flex flex-col gap-1.5"
            onSubmit={(e) => {
              e.preventDefault()
              if (!text.trim()) {
                document.getElementById(`alt-${issue.id}`)?.focus()
                return
              }
              void a.fix(issue, { kind: 'alt_text', path: issue.path, index: issue.element.index, value: text.trim() }, t(m.fixedAlt, { name }))
            }}
          >
            <label className="lbl" htmlFor={`alt-${issue.id}`}>
              {t(m.altLabel, { language })}
            </label>
            <input id={`alt-${issue.id}`} className="in" placeholder={t(m.altPlaceholder)} value={text} onChange={(e) => setText(e.target.value)} />
            <div className="flex flex-wrap gap-1.5">
              <Button type="submit" size="sm" variant="primary">
                {t(m.save)}
              </Button>
              {f.decorative ? (
                <Button size="sm" onClick={() => void a.fix(issue, { kind: 'decorative', path: issue.path, index: issue.element.index }, t(m.fixedDecorative, { name }))}>
                  {t(m.decorative)}
                </Button>
              ) : null}
            </div>
          </form>
          {others.length ? <span className="k">{t(m.altOthers, { languages: formatList(others) })}</span> : null}
        </div>
      )
    }
    case 'color':
      return (
        <div className="a11y-sugs">
          {f.suggestions.map((s) => (
            <div key={s.color} className="a11y-sug">
              <span className="a11y-aa" style={{ background: f.target === 'background' ? s.color : (f.against ?? '#FFFFFF'), color: f.target === 'background' ? (f.against ?? '#FFFFFF') : s.color }} aria-hidden="true">
                {/* A sample of the letters, the same in every language. */}
                {'Aa'}
              </span>
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="mono font-semibold">{s.color}</span>
                <span className="k">{s.reason === 'other_buttons' ? t(m.reasonOtherButtons) : s.reason === 'nearest_aa' ? t(m.reasonNearestAA) : t(m.reasonAAA)}</span>
              </span>
              <span className="font-semibold">{`${s.ratio.toFixed(1)}:1`}</span>
              <Button
                size="sm"
                aria-label={t(m.useColor, { color: s.color })}
                onClick={() =>
                  void a.fix(issue, { kind: 'color', path: issue.path, target: f.target, from: f.from, value: s.color }, t(m.fixedColor, { label: issue.element.label ?? '', color: s.color }))
                }
              >
                {t(m.use)}
              </Button>
            </div>
          ))}
        </div>
      )
    case 'link_text': {
      const version = issue.versions.find((v) => v.lang === lang)
      const links = f.links.length ? f.links.filter((l) => l.href === f.links[0]!.href) : [{ path: issue.path, index: issue.element.index, href: '' }]
      const apply = async (value: string) => {
        for (const l of links) {
          if (!(await a.fix(issue, { kind: 'link_text', path: l.path, index: l.index, value }, t(m.fixedLink, { text: value })))) return
        }
      }
      return (
        <div className="a11y-fix">
          <div className="flex flex-wrap gap-1.5">
            <LinkTextAssist
              component={componentAt(c, issue.path)}
              text={version?.text ?? issue.element.label ?? ''}
              href={issue.element.href ?? links[0]?.href ?? ''}
              onAccept={(value) => apply(value)}
            />
            {version?.suggestion ? (
              <Button size="sm" onClick={() => void apply(version.suggestion!)}>
                {t(m.useTitle, { title: version.suggestion })}
              </Button>
            ) : null}
            {!own ? (
              <Button size="sm" onClick={() => setOwn(true)}>
                {t(m.writeOwn)}
              </Button>
            ) : null}
          </div>
          {own || !version?.suggestion ? (
            <form
              className="flex flex-col gap-1.5"
              onSubmit={(e) => {
                e.preventDefault()
                if (text.trim()) void apply(text.trim())
              }}
            >
              <label className="lbl" htmlFor={`link-${issue.id}`}>
                {t(m.linkLabel, { language })}
              </label>
              <span className="flex gap-1.5">
                <input id={`link-${issue.id}`} className="in flex-1" placeholder={t(m.linkPlaceholder)} value={text} onChange={(e) => setText(e.target.value)} />
                <Button type="submit" size="sm" variant="primary" disabledReason={text.trim() ? undefined : t(m.linkPlaceholder)}>
                  {t(m.save)}
                </Button>
              </span>
            </form>
          ) : null}
        </div>
      )
    }
    case 'heading': {
      const n = f.headings.length
      const level = f.headings[0]?.to ?? 2
      const label = n === 1 ? t(m.headingOne, { level }) : n === 2 ? t(m.headingBoth, { level }) : t(m.headingAll, { count: n, level })
      const summary = n === 1 ? t(m.fixedHeadingOne, { text: f.headings[0]!.text, level }) : t(m.fixedHeadingMany, { subject: f.subject ?? '', level })
      return (
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" onClick={() => void a.fix(issue, { kind: 'heading', path: issue.path, headings: f.headings }, summary, t(m.toastHeadings, { count: n, level }))}>
            {label}
          </Button>
          <span className="k">{t(m.keepsLook)}</span>
        </div>
      )
    }
    case 'text_size':
      return (
        <div>
          <Button size="sm" onClick={() => void a.fix(issue, { kind: 'text_size', path: issue.path, toPx: f.toPx }, t(m.fixedSize, { subject: f.subject ?? '', px: f.toPx ?? 12 }))}>
            {t(m.setSize, { px: f.toPx ?? 12 })}
          </Button>
        </div>
      )
  }
  return null
}

function FixedList() {
  const a = useAccessibility()!
  const t = useT()
  return (
    <div className="a11y-fixed">
      <span className="font-semibold">{t(m.fixedTitle, { count: a.fixed.length })}</span>
      {a.fixed.map((f) => (
        <div key={f.id} className="flex items-start justify-between gap-2">
          <span>{f.summary}</span>
          <button type="button" className="a11y-show" onClick={() => a.undo(f)}>
            {t(m.undo)}
          </button>
        </div>
      ))}
    </div>
  )
}

function Passing({ report, open }: { report: Report; open: boolean }) {
  const t = useT()
  const k = report.counts
  const imageIssues = report.issues.filter((i) => i.kind === 'alt_text' || (i.kind === 'link_no_text' && i.element.role === 'linked_image'))
  const linkIssues = report.issues.filter((i) => i.kind === 'link_text' || i.kind === 'link_no_text')
  const above = (issues: Issue[], missing: number) =>
    missing <= 0
      ? t(m.passNothing)
      : issues.length === 1 && issues[0]!.kind === 'alt_text'
        ? t(m.passNamedAbove, { name: lowerFirst(imageName(issues[0]!)) })
        : t(m.passAbove, { count: missing })
  const rows: { title: string; detail: string }[] = [
    { title: t(m.passLangDir), detail: `${k.languages.join(', ')} · ${k.dir}` },
    { title: t(m.passTables, { count: k.layoutTables }), detail: k.roleAdded > 0 ? t(m.passTablesAdded) : t(m.passTablesNone) },
  ]
  if (k.images > 0) rows.push({ title: t(m.passImages, { ok: k.imagesOk, count: k.images }), detail: above(imageIssues, k.images - k.imagesOk) })
  if (k.links > 0) rows.push({ title: t(m.passLinks, { ok: k.linksOk, count: k.links }), detail: above(linkIssues, k.links - k.linksOk) })
  return (
    <details className="a11y-pass" open={open}>
      <summary>
        <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="m6 3 5 5-5 5" />
        </svg>
        <span className="font-semibold">{t(m.passingTitle, { count: rows.length })}</span>
        <span className="k font-medium">{t(m.passingCaption)}</span>
      </summary>
      <div className="a11y-pass-rows">
        {rows.map((r) => (
          <div key={r.title} className="a11y-pass-row">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#15803D" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="m3 8 3.5 3.5L13 5" />
            </svg>
            <span className="flex flex-1 flex-col gap-px">
              <span>
                <CodeText text={r.title} />
              </span>
              <span className="k">{r.detail}</span>
            </span>
          </div>
        ))}
      </div>
    </details>
  )
}

function lowerFirst(s: string): string {
  return s ? s[0]!.toLowerCase() + s.slice(1) : s
}

/** The ratio a suggestion shows. */
export type Suggestion = Schemas['AccessibilityColorSuggestion']
