import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { count, dayMonthYear } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { AddLanguageDialog } from './AddLanguageDialog'
import { ms } from './messages'
import { RemoveLanguageDialog, defaultSelectId, fallbackSelectId } from './RemoveLanguageDialog'
import './languages.css'

type Langs = Schemas['WorkspaceLanguages']
type Lang = Schemas['WorkspaceLanguage']
type Translate = ReturnType<typeof useT>

export const languagesKey = (slug: string) =>
  $api.queryOptions('get', '/api/v1/workspaces/{workspace}/languages', { params: { path: { workspace: slug } } }).queryKey

/** Trailing words a workspace name often carries that the copy leaves out. */
const generic = new Set(['newsletter', 'newsletters', 'news', 'team', 'workspace', 'inc', 'inc.', 'ltd', 'ltd.', 'llc'])

/** "Acme Newsletter" → "Acme": how the boards name the brand in page copy. */
export function brandOf(name: string): string {
  const words = name.trim().split(/\s+/)
  while (words.length > 1 && generic.has((words[words.length - 1] ?? '').toLowerCase())) words.pop()
  return words.join(' ')
}

/**
 * Settings › General › Languages (SettingsLanguages, LanguageStates): the
 * workspace's languages with their contacts and page copy, the default and
 * the fallback, and how contacts get a language.
 */
export function LanguagesCard({ className }: { className?: string }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/languages', { params: { path: { workspace: slug } } })
  if (q.isPending) return <LanguagesSkeleton className={className} />
  if (q.isError) {
    return (
      <section className={`card ${className ?? ''}`} aria-labelledby="lang-h">
        <CardSection id="lang-h" title={t(ms.title)} />
        <div className="pad">
          <ErrorPanel title={t(ms.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(ms.retry)}</Button>}>
            {t(ms.loadFailedDetail, { error: errorMessage(q.error) })}
          </ErrorPanel>
        </div>
      </section>
    )
  }
  return <Languages data={q.data} className={className} />
}

function Languages({ data, className }: { data: Langs; className?: string }) {
  const { slug, me } = useWorkspace()
  const t = useT()
  const [adding, setAdding] = useState(false)
  const [removing, setRemoving] = useState<Lang>()
  const fallbackName = data.languages.find((l) => l.code === data.fallback)?.name ?? data.fallback
  const brand = brandOf(me.workspace.name)
  return (
    <section className={`card ${className ?? ''}`} aria-labelledby="lang-h">
      <CardSection
        id="lang-h"
        title={t(ms.title)}
        caption={
          <T
            msg={ms.caption}
            values={{
              link: (c) => (
                <Link to="/w/$workspace/account" params={{ workspace: slug }}>
                  {c}
                </Link>
              ),
            }}
          />
        }
        actions={
          data.canManage ? (
            <Button size="sm" variant="primary" icon="plus" onClick={() => setAdding(true)}>
              {t(ms.add)}
            </Button>
          ) : null
        }
      />
      {data.canManage ? null : <ReadOnly helpers={data.helpers} />}
      <LanguageTable data={data} brand={brand} fallbackName={fallbackName} onRemove={setRemoving} />
      <LanguageList data={data} brand={brand} fallbackName={fallbackName} onRemove={setRemoving} />
      <div className="lng-foot">
        <span className="k">{t(ms.footTotal, { total: count(data.totalContacts), ...regionalExample(data) })}</span>
        <span className="k">{t(ms.footRtl)}</span>
      </div>
      <div className="pad flex flex-col gap-5" style={{ borderTop: '1px solid var(--line-2)' }}>
        <Defaults data={data} brand={brand} />
        <HowContacts data={data} brand={brand} fallbackName={fallbackName} />
      </div>
      {adding ? <AddLanguageDialog open onOpenChange={setAdding} fallback={fallbackName} /> : null}
      {removing ? <RemoveLanguageDialog language={removing} data={data} open onOpenChange={(o) => !o && setRemoving(undefined)} /> : null}
    </section>
  )
}

/** "es-MX gets Spanish": a regional code the contacts have, or the board's. */
function regionalExample(data: Langs): { code: string; language: string } {
  for (const l of data.languages) {
    const code = l.codes.find((c) => c.includes('-'))
    if (code) return { code, language: l.name }
  }
  return { code: 'es-MX', language: 'Spanish' }
}

function ReadOnly({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me } = useWorkspace()
  const { data } = useMe()
  const name = data?.user.name || data?.user.email || ''
  return (
    <div className="pad" style={{ paddingBottom: 4 }}>
      <div className="note n" role="note">
        <Icon name="info" size={16} />
        <div className="flex min-w-0 flex-col gap-1" style={{ flex: '1 1 320px' }}>
          <span style={{ fontWeight: 700 }}>{t(ms.readOnlyTitle, { name, role: me.workspace.roleName })}</span>
          <div>
            <T
              msg={ms.readOnly}
              values={{ helpers: helpers.length ? formatList(helpers, 'disjunction') : 'none' }}
            />
          </div>
        </div>
      </div>
    </div>
  )
}

function codesText(l: Lang): string {
  return l.codes.join(', ') + (l.moreCodes ? '…' : '')
}

function LanguageName({ l, wide }: { l: Lang; wide?: boolean }) {
  const t = useT()
  return (
    <div className="flex min-w-0 flex-col gap-[3px]" style={wide ? { minWidth: 210 } : undefined}>
      <span className="flex flex-wrap items-center gap-1.5">
        <b style={{ fontWeight: 600 }}>{l.name}</b>
        {l.isDefault ? (
          <Badge tone="neutral" style={{ height: 20, fontSize: 11 }}>
            {t(ms.badgeDefault)}
          </Badge>
        ) : null}
        {l.isFallback ? (
          <Badge tone="neutral" style={{ height: 20, fontSize: 11 }}>
            {t(ms.badgeFallback)}
          </Badge>
        ) : null}
      </span>
      <span className="k" style={{ whiteSpace: 'nowrap' }}>
        {l.native && l.native !== l.name ? (
          <>
            <span lang={l.code}>{l.native}</span>
            {' · '}
          </>
        ) : null}
        <span className="mono" style={{ fontSize: 11.5 }}>
          {codesText(l)}
        </span>
      </span>
    </div>
  )
}

function pageCopyDetail(l: Lang, brand: string, fallbackName: string, t: Translate): string {
  const p = l.pageCopy
  if (p.status === 'live') {
    if (l.isDefault) return t(ms.pageDefault)
    return p.by && p.at ? t(ms.pageBy, { name: p.by, date: dayMonthYear(p.at) }) : ''
  }
  if (p.status === 'incomplete') return t(ms.pageIncompleteDetail, { count: p.missing, fallback: fallbackName })
  return p.builtinLabels ? t(ms.pageNotSetUpReady, { workspace: brand, fallback: fallbackName }) : t(ms.pageNotSetUpNone, { language: l.name, fallback: fallbackName })
}

function PageCopy({ l, brand, fallbackName, stack }: { l: Lang; brand: string; fallbackName: string; stack?: boolean }) {
  const t = useT()
  const status = l.pageCopy.status
  const badge =
    status === 'live' ? (
      <Badge tone="ok">{t(ms.pageLive)}</Badge>
    ) : status === 'incomplete' ? (
      <Badge tone="warn">{t(ms.pageIncomplete)}</Badge>
    ) : (
      <Badge tone="neutral">{t(ms.pageNotSetUp)}</Badge>
    )
  const detail = pageCopyDetail(l, brand, fallbackName, t)
  const wrap = status === 'live' ? undefined : ({ whiteSpace: 'normal' } as const)
  return (
    <div className={stack ? 'flex flex-wrap items-center gap-1.5' : 'flex flex-col items-start gap-1'} style={stack ? undefined : { maxWidth: 320 }}>
      {badge}
      {detail ? (
        <span className="k" style={wrap}>
          {detail}
        </span>
      ) : null}
    </div>
  )
}

function Adder({ l }: { l: Lang }) {
  return (
    <span className="inline-flex items-center gap-2 whitespace-nowrap">
      <Avatar name={l.addedBy} size={22} fontSize={11} />
      <span>{l.addedBy}</span>
    </span>
  )
}

function RowActions({ l, data, onRemove, stack }: { l: Lang; data: Langs; onRemove: (l: Lang) => void; stack?: boolean }) {
  const { slug } = useWorkspace()
  const t = useT()
  const live = l.pageCopy.status === 'live'
  const label = !data.canManage ? t(ms.viewPageCopy) : live ? t(ms.editPageCopy) : t(ms.writePageCopy, { language: l.name })
  const more = t(ms.moreActions, { language: l.name })
  return (
    <span className={stack ? 'flex flex-wrap gap-1.5' : 'inline-flex flex-nowrap items-center justify-end gap-1.5'}>
      <Link
        to="/w/$workspace/forms/preferences"
        params={{ workspace: slug }}
        search={{ language: l.code } as never}
        className={`btn sm${data.canManage && !live ? ' pri' : ''}`}
      >
        {label}
      </Link>
      {data.canManage ? (
        <Menu
          label={more}
          trigger={
            <button type="button" className="btn sm ic" aria-label={more} style={{ width: 30, padding: 0 }}>
              <Icon name="more" size={14} />
            </button>
          }
        >
          <MenuItem asChild>
            <Link to="/w/$workspace/forms/preferences" params={{ workspace: slug }} search={{ language: l.code } as never}>
              {t(ms.editPageCopy)}
            </Link>
          </MenuItem>
          <MenuItem onSelect={() => onRemove(l)}>{t(ms.remove)}</MenuItem>
        </Menu>
      ) : null}
    </span>
  )
}

/** "35,500 English · 5,508 with no language · 12 other languages (pt-BR, fr, it) · all get English" */
function breakdown(data: Langs, l: Lang, fallbackName: string, t: Translate): string {
  const codes = data.notOfferedCodes.slice(0, 3).join(', ') + (data.notOfferedCodes.length > 3 ? '…' : '')
  return t(ms.breakdown, {
    own: count(l.own),
    language: l.name,
    none: count(data.noLanguage),
    others: data.notOffered,
    codes: data.notOffered > 0 && codes ? t(ms.breakdownCodes, { codes }) : '',
    fallback: fallbackName,
  })
}

function hasBreakdown(data: Langs, l: Lang): boolean {
  return l.isFallback && data.noLanguage + data.notOffered > 0
}

function LanguageTable({ data, brand, fallbackName, onRemove }: { data: Langs; brand: string; fallbackName: string; onRemove: (l: Lang) => void }) {
  const t = useT()
  return (
    <div className="max-lg:hidden" style={{ overflowX: 'auto' }}>
      <table className="tbl lng-tbl" aria-label={t(ms.tableLabel)}>
        <thead>
          <tr>
            <th scope="col">{t(ms.colLanguage)}</th>
            <th scope="col" className="r">
              {t(ms.colContacts)}
            </th>
            <th scope="col">{t(ms.colPageCopy)}</th>
            <th scope="col">{t(ms.colAdded)}</th>
            <th scope="col">
              <span className="vh">{t(ms.colActions)}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {data.languages.flatMap((l) => {
            const sub = hasBreakdown(data, l)
            const rows = [
              <tr key={l.code} className={sub ? 'nob' : undefined}>
                <td>
                  <LanguageName l={l} wide />
                </td>
                <td className="r">
                  <span className="num" style={{ fontWeight: 600 }}>
                    {count(l.contacts)}
                  </span>
                </td>
                <td>
                  <PageCopy l={l} brand={brand} fallbackName={fallbackName} />
                </td>
                <td>
                  <div className="flex flex-col gap-[3px]">
                    {l.addedBy && l.addedAt ? (
                      <>
                        <Adder l={l} />
                        <span className="k">{dayMonthYear(l.addedAt)}</span>
                      </>
                    ) : (
                      <span className="k">–</span>
                    )}
                  </div>
                </td>
                <td className="r">
                  <RowActions l={l} data={data} onRemove={onRemove} />
                </td>
              </tr>,
            ]
            if (sub) {
              rows.push(
                <tr key={`${l.code}-sub`} className="subr">
                  <td colSpan={5}>
                    <span className="k" style={{ color: 'var(--ink-nav)' }}>
                      {breakdown(data, l, fallbackName, t)}
                    </span>
                  </td>
                </tr>,
              )
            }
            return rows
          })}
        </tbody>
      </table>
    </div>
  )
}

/** Tablets and phones: one stacked block per language (the boards' narrow layout). */
function LanguageList({ data, brand, fallbackName, onRemove }: { data: Langs; brand: string; fallbackName: string; onRemove: (l: Lang) => void }) {
  const t = useT()
  return (
    <ul className="m-0 list-none p-0 lg:hidden" aria-label={t(ms.tableLabel)}>
      {data.languages.map((l) => (
        <li key={l.code} className="flex flex-col gap-2" style={{ padding: '14px 16px', borderTop: '1px solid var(--line-2)' }}>
          <div className="flex items-start justify-between gap-2.5">
            <LanguageName l={l} />
            <span className="num" style={{ fontWeight: 600 }}>
              {count(l.contacts)}
            </span>
          </div>
          {hasBreakdown(data, l) ? <span className="k">{breakdown(data, l, fallbackName, t)}</span> : null}
          <PageCopy l={l} brand={brand} fallbackName={fallbackName} stack />
          <div className="flex flex-wrap items-center gap-2">
            <span className="k">{t(ms.colAdded)}</span>
            {l.addedBy && l.addedAt ? (
              <>
                <Adder l={l} />
                <span className="k">{dayMonthYear(l.addedAt)}</span>
              </>
            ) : (
              <span className="k">–</span>
            )}
          </div>
          <RowActions l={l} data={data} onRemove={onRemove} stack />
        </li>
      ))}
    </ul>
  )
}

type DefaultPair = { default: string; fallback: string }

function Defaults({ data, brand }: { data: Langs; brand: string }) {
  const { slug } = useWorkspace()
  const t = useT()
  const qc = useQueryClient()
  const [def, setDef] = useState(data.default)
  const [fb, setFb] = useState(data.fallback)
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/languages/defaults')
  const changingFallback = fb !== data.fallback
  const impact = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/languages/fallback-impact',
    { params: { path: { workspace: slug }, query: { code: fb } } },
    { enabled: changingFallback },
  )
  const nameOf = (code: string) => data.languages.find((l) => l.code === code)?.name ?? code
  const before: DefaultPair = { default: data.default, fallback: data.fallback }
  const commit = (next: DefaultPair, undo?: DefaultPair) => {
    save.mutate(
      { params: { path: { workspace: slug } }, body: next },
      {
        onSuccess: (out) => {
          qc.setQueryData(languagesKey(slug), out)
          setDef(out.default)
          setFb(out.fallback)
          if (!undo) return
          if (next.fallback !== undo.fallback) {
            toast({
              title: t(ms.fallbackSaved, { language: nameOf(next.fallback) }),
              detail: t(ms.fallbackSavedDetail, { count: impact.data?.moved ?? 0 }),
              undo: () => commit(undo),
            })
          } else {
            toast({ title: t(ms.defaultSaved, { language: nameOf(next.default) }), undo: () => commit(undo) })
          }
        },
        onError: (e) => toast({ kind: 'error', title: t(ms.saveFailed), detail: errorMessage(e) }),
      },
    )
  }
  const options = data.languages.map((l) => (
    <option key={l.code} value={l.code}>
      {l.name}
    </option>
  ))
  return (
    <>
      <div className="grid items-start" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(320px,100%),1fr))', gap: '14px 28px' }}>
        <label className="lbl">
          {t(ms.defaultLabel)}
          <select
            id={defaultSelectId}
            className="in"
            aria-label={t(ms.defaultLabel)}
            aria-describedby={`${defaultSelectId}-help`}
            value={def}
            disabled={!data.canManage}
            onChange={(e) => setDef(e.target.value)}
          >
            {options}
          </select>
          <span className="k" id={`${defaultSelectId}-help`} style={{ fontWeight: 500 }}>
            {t(ms.defaultHelp)}
          </span>
        </label>
        <label className="lbl">
          {t(ms.fallbackLabel)}
          <select
            id={fallbackSelectId}
            className="in"
            aria-label={t(ms.fallbackLabel)}
            aria-describedby={`${fallbackSelectId}-help`}
            value={fb}
            disabled={!data.canManage}
            onChange={(e) => setFb(e.target.value)}
          >
            {options}
          </select>
          <span className="k" id={`${fallbackSelectId}-help`} style={{ fontWeight: 500 }}>
            {t(ms.fallbackHelp)}
          </span>
        </label>
      </div>
      {def !== data.default ? (
        <div className="flex flex-col gap-2.5">
          <div className="note i" role="note">
            <Icon name="info" size={16} />
            <div className="flex min-w-0 flex-col gap-1" style={{ flex: '1 1 320px' }}>
              <span style={{ fontWeight: 700 }}>{t(ms.defaultNote, { language: nameOf(def) })}</span>
              <div>{t(ms.defaultNoteDetail)}</div>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="primary" loading={save.isPending} onClick={() => commit({ default: def, fallback: data.fallback }, before)}>
              {t(ms.saveDefault)}
            </Button>
            <Button size="sm" onClick={() => setDef(data.default)}>
              {t(ms.cancel)}
            </Button>
          </div>
        </div>
      ) : null}
      {changingFallback ? (
        <div className="flex flex-col gap-2.5">
          <div className="note i" role="note">
            <Icon name="info" size={16} />
            {impact.data ? (
              impact.data.moved > 0 ? (
                <div className="flex min-w-0 flex-col gap-1" style={{ flex: '1 1 320px' }}>
                  <span style={{ fontWeight: 700 }}>{t(ms.fallbackNote, { count: count(impact.data.moved), language: nameOf(fb) })}</span>
                  <div>
                    {t(ms.fallbackNoteDetail, {
                      current: nameOf(data.fallback),
                      none: count(impact.data.noLanguage),
                      others: count(impact.data.notOffered),
                      workspace: brand,
                      language: nameOf(fb),
                    })}
                  </div>
                </div>
              ) : (
                <div style={{ flex: '1 1 320px' }}>{t(ms.fallbackNoteNone, { workspace: brand })}</div>
              )
            ) : impact.isError ? (
              <div style={{ flex: '1 1 320px' }}>{errorMessage(impact.error)}</div>
            ) : (
              <div style={{ flex: '1 1 320px' }} aria-busy="true">
                <Skeleton w={320} h={12} />
              </div>
            )}
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="primary" loading={save.isPending} onClick={() => commit({ default: data.default, fallback: fb }, before)}>
              {t(ms.saveFallback)}
            </Button>
            <Button size="sm" onClick={() => setFb(data.fallback)}>
              {t(ms.cancel)}
            </Button>
          </div>
        </div>
      ) : null}
    </>
  )
}

function HowContacts({ data, brand, fallbackName }: { data: Langs; brand: string; fallbackName: string }) {
  const { slug } = useWorkspace()
  const t = useT()
  const codes = data.notOfferedCodes.slice(0, 3).join(', ') + (data.notOfferedCodes.length > 3 ? '…' : '')
  const ways = [
    [ms.howForms, ms.howFormsText],
    [ms.howImports, ms.howImportsText],
    [ms.howApi, ms.howApiText],
    [ms.howPrefs, ms.howPrefsText],
    [ms.howTeam, ms.howTeamText],
  ] as const
  return (
    <div className="flex min-w-0 flex-col gap-2.5">
      <h3 className="h2" style={{ fontSize: 14 }}>
        {t(ms.howTitle)}
      </h3>
      <div className="lng-how">
        {ways.map(([title, text]) => (
          <div key={title.id}>
            <span style={{ fontWeight: 600 }}>{t(title)}</span>
            <span className="k" style={{ color: 'var(--ink-2)' }}>
              <T msg={text} values={{ code: (c) => <span className="mono">{c}</span> }} />
            </span>
          </div>
        ))}
      </div>
      <div className="lng-box row" style={{ gap: 10, justifyContent: 'space-between' }}>
        <span style={{ flex: '1 1 300px' }}>
          <T
            msg={data.noLanguage > 0 ? ms.noLanguage : ms.allHaveLanguage}
            values={{
              count: data.noLanguage,
              others: data.notOffered,
              codes,
              workspace: brand,
              fallback: fallbackName,
              b: (c) => <b style={{ fontWeight: 600 }}>{c}</b>,
              span: (c) => (
                <span className="k" style={{ color: 'var(--ink-nav)' }}>
                  {c}
                </span>
              ),
            }}
          />
        </span>
        {data.noLanguage > 0 ? (
          <Link to="/w/$workspace/audience" params={{ workspace: slug }} search={{ language: 'none' } as never} className="btn sm">
            {t(ms.viewNoLanguage)}
          </Link>
        ) : null}
      </div>
    </div>
  )
}

function LanguagesSkeleton({ className }: { className?: string }) {
  const t = useT()
  return (
    <section
      className={`card ${className ?? ''}`}
      aria-busy="true"
      aria-label={t(ms.loading)}
      style={{ padding: 18, display: 'flex', flexDirection: 'column', gap: 14 }}
    >
      <div className="flex justify-between">
        <Skeleton w={140} h={14} r={6} />
        <Skeleton w={110} h={30} r={8} />
      </div>
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="flex items-center gap-3.5" style={{ padding: '8px 0', borderTop: '1px solid var(--line-2)' }}>
          <Skeleton w={150} h={14} r={6} />
          <Skeleton w={60} h={14} r={6} />
          <Skeleton h={10} className="flex-1" />
          <Skeleton w={90} h={26} r={8} />
        </div>
      ))}
    </section>
  )
}
