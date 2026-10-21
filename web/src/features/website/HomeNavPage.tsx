import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog, UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, count, dayMonthDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { bare, useInvalidateWebsite, WS, type HomeNav, type HomeNavInput, type HomeSection, type NavLink, type Overview, type PostRow, type Tag } from './api'
import { mh } from './homenav.messages'
import { ImagePickerDialog } from './ImagePicker'
import { m } from './messages'
import { mp } from './posts.messages'
import { ReorderList } from './Reorder'
import { LayoutThumbs, TurnOffCard } from './TurnOn'
import { SiteFrame } from './SiteFrame'
import { ReadOnlyNote, SavedLine, WebsitePage } from './WebsiteFrame'

const MAX_FEATURED = 3
const MAX_HEADER = 6

function inputOf(h: HomeNav): HomeNavInput {
  return {
    layout: h.layout,
    featured: h.featured.map((f) => f.id),
    sections: h.sections.map((s) => ({ ...s })),
    about: { ...h.about },
    header: h.header.map((l) => ({ kind: l.kind, label: l.label, ref: l.ref, url: l.url })),
    footer: { ...h.footer },
    privacyUrl: h.privacyUrl,
    poweredBy: h.poweredBy,
    identity: { ...h.identity },
    index: h.index,
  }
}

type Featured = { id: string; title: string; line: string }

/** Website › Home & navigation (WebsiteHomeNav, WebsiteStates · home & navigation). */
export function HomeNavPage({ overview }: { overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${WS}/home-nav`, { params: { path: { workspace: slug } } })
  if (q.isError && !q.data) {
    return (
      <WebsitePage view="home" overview={overview}>
        <ErrorPanel title={t(mh.saveFailed)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      </WebsitePage>
    )
  }
  if (!q.data) {
    return (
      <WebsitePage view="home" overview={overview}>
        <Loading label={t(m.loading)} className="grid gap-5 lg:grid-cols-2">
          <Skeleton w="100%" h={520} r={10} />
          <Skeleton w="100%" h={520} r={10} />
        </Loading>
      </WebsitePage>
    )
  }
  return <HomeNavEditor key={q.data.updatedAt} data={q.data} overview={overview} />
}

function HomeNavEditor({ data, overview }: { data: HomeNav; overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const canManage = useCan('website:manage')
  const ro = !canManage
  const initial = useMemo(() => inputOf(data), [data])
  const [draft, setDraft] = useState<HomeNavInput>(initial)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [picked, setPicked] = useState<Record<string, Featured>>({})
  const dirty = JSON.stringify(draft) !== JSON.stringify(initial)
  const host = overview.address.domain || overview.address.host
  const set = (patch: Partial<HomeNavInput>) => setDraft((d) => ({ ...d, ...patch }))
  const featuredInfo = (id: string): Featured => {
    const f = data.featured.find((x) => x.id === id)
    if (f) {
      const vis = f.visibility === 'paid' ? `${t(mp.paid)}${f.tiers.length ? `, ${f.tiers.join(', ')}` : ''}` : f.visibility === 'members' ? t(mp.members) : t(mp.public)
      const date = f.publishedAt ? dayMonthDate(f.publishedAt) : t(mh.notPublished)
      return { id, title: f.title, line: f.tag ? t(mh.featuredLine, { tag: f.tag, visibility: vis, date }) : t(mh.featuredLineNoTag, { visibility: vis, date }) }
    }
    return picked[id] ?? { id, title: id, line: '' }
  }
  const save = async (): Promise<boolean> => {
    if (!draft.identity.title.trim()) {
      setError(t(mh.titleRequired))
      return false
    }
    setBusy(true)
    setError(undefined)
    try {
      await http.PUT(`${WS}/home-nav`, { params: { path: { workspace: slug } }, body: draft })
      toast({ kind: 'success', title: t(mh.saved), detail: t(mh.savedDetail, { host }), action: { label: t(m.viewSite), onClick: () => window.open(overview.address.url, '_blank', 'noopener') } })
      await invalidate()
      return true
    } catch (e) {
      setError(errorMessage(e))
      toast({ kind: 'error', title: t(mh.saveFailed), detail: errorMessage(e) })
      return false
    } finally {
      setBusy(false)
    }
  }
  const actions = canManage ? (
    <>
      {dirty ? null : <SavedLine at={data.updatedAt} by={data.updatedBy} />}
      <Button variant="primary" loading={busy} busyLabel={t(mh.saving)} disabled={!dirty} onClick={() => void save()}>
        {t(mh.save)}
      </Button>
    </>
  ) : undefined
  return (
    <WebsitePage view="home" overview={overview} actions={actions}>
      {ro ? <ReadOnlyNote msg={m.readOnlyManage} helpers={overview.helpers.manage} perms={{ perm: 'website:manage' }} /> : null}
      {error ? <Note tone="bad">{error}</Note> : null}
      <div className="grid items-start gap-5 min-[1101px]:grid-cols-[minmax(0,1fr)_minmax(0,1.05fr)]">
        <div className="flex min-w-0 flex-col gap-5">
          <section className="card" aria-labelledby="hn-layout">
            <div className="ws-sec">
              <div className="tx">
                <h2 className="ws-h2" id="hn-layout">
                  {t(mh.layoutTitle)}
                </h2>
                <span className="k">{t(mh.layoutSub)}</span>
              </div>
            </div>
            <div className="pad">
              <LayoutThumbs value={draft.layout} onChange={(layout) => set({ layout })} disabled={ro} label={t(mh.layoutTitle)} />
            </div>
          </section>
          <FeaturedCard draft={draft} set={set} ro={ro} info={featuredInfo} onPicked={(f) => setPicked((p) => ({ ...p, [f.id]: f }))} />
          <SectionsCard draft={draft} set={set} ro={ro} tags={data.tags} featured={draft.featured.length} />
          <NavigationCard draft={draft} set={set} ro={ro} data={data} host={host} />
        </div>
        <div className="flex min-w-0 flex-col gap-5 min-[1101px]:sticky min-[1101px]:top-4">
          <LivePreview draft={draft} host={host} />
          <SeoCard draft={draft} set={set} ro={ro} data={data} />
        </div>
      </div>
      <IdentityCard draft={draft} set={set} ro={ro} data={data} />
      <TagsCard data={data} overview={overview} ro={ro} />
      <TurnOffCard overview={overview} />
      {dirty && canManage ? (
        <div className="card sticky bottom-3 z-10 flex flex-wrap items-center gap-3 px-[18px] py-3 shadow-[0_8px_24px_rgba(11,18,32,.12)]" role="region" aria-label={t(mh.unsaved)}>
          <Badge tone="warn" dot>
            {t(mh.unsaved)}
          </Badge>
          <span className="k min-w-0 flex-1">{changes(t, initial, draft)}</span>
          <Button onClick={() => setDraft(initial)} disabled={busy}>
            {t(mh.discard)}
          </Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {t(mh.save)}
          </Button>
        </div>
      ) : null}
      <UnsavedChangesGuard
        dirty={dirty && canManage}
        title={t(mh.leaveTitle)}
        description={t(mh.leaveText)}
        saveLabel={t(mh.leaveSave)}
        onSave={async () => {
          if (!(await save())) throw new Error(t(mh.saveFailed))
        }}
      />
    </WebsitePage>
  )
}

/** "Layout: Grid instead of Featured and latest · Engineering row hidden". */
function changes(t: ReturnType<typeof useT>, a: HomeNavInput, b: HomeNavInput): string {
  const out: string[] = []
  const layoutName = (l: string) => (l === 'grid' ? t(m.layoutGrid) : l === 'list' ? t(m.layoutList) : t(m.layoutFeatured))
  if (a.layout !== b.layout) out.push(t(mh.chLayout, { layout: layoutName(b.layout) }))
  const same = (x: unknown, y: unknown) => JSON.stringify(x) === JSON.stringify(y)
  if (!same(a.featured, b.featured)) out.push(t(mh.chFeatured))
  if (!same(a.sections, b.sections)) out.push(t(mh.chSections))
  if (!same(a.about, b.about)) out.push(t(mh.chAbout))
  if (!same(a.header, b.header)) out.push(t(mh.chHeader))
  if (!same(a.footer, b.footer) || a.poweredBy !== b.poweredBy || a.privacyUrl !== b.privacyUrl) out.push(t(mh.chFooter))
  if (!same(a.identity, b.identity)) out.push(t(mh.chIdentity))
  if (a.index !== b.index) out.push(t(mh.chIndex))
  return out.join(' · ')
}

type Setter = (patch: Partial<HomeNavInput>) => void

function Remove({ name, onClick, disabled }: { name: string; onClick: () => void; disabled?: boolean }) {
  const t = useT()
  return (
    <button type="button" className="btn sm ic ws-menu-ic" aria-label={t(mh.removeItem, { name })} disabled={disabled} onClick={onClick}>
      <Icon name="close" size={13} />
    </button>
  )
}

function FeaturedCard({ draft, set, ro, info, onPicked }: { draft: HomeNavInput; set: Setter; ro: boolean; info: (id: string) => Featured; onPicked: (f: Featured) => void }) {
  const t = useT()
  const [picking, setPicking] = useState(false)
  const full = draft.featured.length >= MAX_FEATURED
  return (
    <section className="card" aria-labelledby="hn-feat">
      <div className="ws-sec">
        <div className="tx">
          <h2 className="ws-h2" id="hn-feat">
            {t(mh.featuredTitle, { count: draft.featured.length, max: MAX_FEATURED })}
          </h2>
          <span className="k">{t(mh.featuredSub, { max: MAX_FEATURED })}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        {draft.featured.length ? (
          <ReorderList
            items={draft.featured}
            keyOf={(id) => id}
            nameOf={(id) => info(id).title}
            label={t(mh.featuredTitle, { count: draft.featured.length, max: MAX_FEATURED })}
            disabled={ro}
            onChange={(featured) => set({ featured })}
            render={(id) => {
              const f = info(id)
              return (
                <>
                  <span className="tx">
                    <b>{f.title}</b>
                    {f.line ? <span className="k">{f.line}</span> : null}
                  </span>
                  <Remove name={f.title} disabled={ro} onClick={() => set({ featured: draft.featured.filter((x) => x !== id) })} />
                </>
              )
            }}
          />
        ) : (
          <span className="k">{t(mh.featuredNone)}</span>
        )}
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Button size="sm" icon="plus" disabled={ro} disabledReason={full ? t(mh.featuredFull, { max: MAX_FEATURED }) : undefined} onClick={() => setPicking(true)}>
            {t(mh.featuredAdd)}
          </Button>
          {draft.featured.length > 1 ? <span className="k">{t(mh.dragHint)}</span> : null}
        </div>
      </div>
      {picking ? (
        <FeaturedPicker
          exclude={draft.featured}
          onClose={() => setPicking(false)}
          onPick={(p) => {
            const vis = p.visibility === 'paid' ? t(mp.paid) : p.visibility === 'members' ? t(mp.members) : t(mp.public)
            const date = p.publishedAt ? dayMonthDate(p.publishedAt) : ''
            onPicked({ id: p.id, title: p.title, line: p.tags[0] ? t(mh.featuredLine, { tag: p.tags[0].name, visibility: vis, date }) : t(mh.featuredLineNoTag, { visibility: vis, date }) })
            set({ featured: [...draft.featured, p.id] })
            setPicking(false)
          }}
        />
      ) : null}
    </section>
  )
}

function FeaturedPicker({ exclude, onClose, onPick }: { exclude: string[]; onClose: () => void; onPick: (p: PostRow) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [q, setQ] = useState('')
  const [term, setTerm] = useState('')
  useEffect(() => {
    const h = window.setTimeout(() => setTerm(q), 250)
    return () => window.clearTimeout(h)
  }, [q])
  const list = $api.useQuery('get', `${WS}/posts`, { params: { path: { workspace: slug }, query: { status: 'published', q: term || undefined, pageSize: 10 } } }, { placeholderData: (p) => p })
  const rows = (list.data?.posts ?? []).filter((p) => !exclude.includes(p.id))
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} title={t(mh.pickTitle)}>
      <div className="flex flex-col gap-3">
        <label className="ws-search" style={{ width: '100%' }}>
          <Icon name="search" size={14} />
          <input type="search" aria-label={t(mh.pickSearch)} placeholder={t(mh.pickSearch)} value={q} onChange={(e) => setQ(e.target.value)} autoFocus />
        </label>
        {list.isPending ? (
          <Skeleton w="100%" h={180} r={8} />
        ) : rows.length === 0 ? (
          <span className="k">{t(mh.pickNone)}</span>
        ) : (
          <div className="ws-dlist">
            {rows.map((p) => (
              <div key={p.id} className="ws-drag">
                <span className="tx">
                  <b>{p.title}</b>
                  <span className="k">{[p.tags[0]?.name, p.publishedAt ? dayMonthDate(p.publishedAt) : ''].filter(Boolean).join(' · ')}</span>
                </span>
                <Button size="sm" onClick={() => onPick(p)}>
                  {t(mh.pickAdd)}
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}

function sentences(text: string): number {
  const s = text.trim()
  if (!s) return 0
  return s.split(/[.!?]+(?:\s|$)/).filter((x) => x.trim()).length
}

function SectionsCard({ draft, set, ro, tags, featured }: { draft: HomeNavInput; set: Setter; ro: boolean; tags: Tag[]; featured: number }) {
  const t = useT()
  const [about, setAbout] = useState(false)
  const [addTag, setAddTag] = useState('')
  const sections = draft.sections
  const tagOf = (id?: string) => tags.find((x) => x.id === id)
  const key = (s: HomeSection) => (s.kind === 'tag' ? `tag:${s.tag}` : s.kind)
  const update = (k: string, patch: Partial<HomeSection>) => set({ sections: sections.map((s) => (key(s) === k ? { ...s, ...patch } : s)) })
  const name = (s: HomeSection): string => {
    switch (s.kind) {
      case 'subscribe':
        return t(mh.secSubscribe)
      case 'featured':
        return t(mh.secFeatured)
      case 'latest':
        return t(mh.secLatest)
      case 'tag':
        return t(mh.secTag, { tag: tagOf(s.tag)?.name ?? '' })
      case 'about':
        return t(mh.secAbout)
      default:
        return t(mh.secTiers)
    }
  }
  const free = tags.filter((x) => !sections.some((s) => s.kind === 'tag' && s.tag === x.id))
  return (
    <section className="card" aria-labelledby="hn-secs">
      <div className="ws-sec">
        <div className="tx">
          <h2 className="ws-h2" id="hn-secs">
            {t(mh.sectionsTitle)}
          </h2>
          <span className="k">{t(mh.sectionsSub)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <ReorderList
          items={sections}
          keyOf={key}
          nameOf={name}
          label={t(mh.sectionsTitle)}
          disabled={ro}
          onChange={(next) => set({ sections: next })}
          render={(s) => {
            const k = key(s)
            const tag = tagOf(s.tag)
            const toggle = (label: string) => (
              <input type="checkbox" role="switch" className="tg" aria-label={label} checked={s.on} disabled={ro} onChange={(e) => update(k, { on: e.target.checked })} />
            )
            let text: ReactNode
            let extra: ReactNode = null
            let after: ReactNode = null
            let label = t(mh.showSection, { name: name(s) })
            switch (s.kind) {
              case 'subscribe':
                text = t(mh.secSubscribeText)
                break
              case 'featured':
                text = t(mh.secFeaturedText, { count: featured })
                break
              case 'latest':
                text = t(mh.secLatestText)
                extra = (
                  <label className="flex items-center gap-2">
                    <span className="k">{t(mh.show)}</span>
                    <input
                      className="in ws-numin"
                      type="number"
                      min={1}
                      max={24}
                      aria-label={t(mh.secLatestCount)}
                      value={s.count ?? 6}
                      disabled={ro}
                      onChange={(e) => update(k, { count: Math.max(1, Math.min(24, Number(e.target.value) || 1)) })}
                    />
                  </label>
                )
                break
              case 'tag':
                text = t(mh.secTagText, { count: s.count ?? 3, tag: tag?.name ?? '' })
                label = t(mh.showTheRow, { tag: tag?.name ?? '' })
                after = <Remove name={t(mh.removeRow, { tag: tag?.name ?? '' })} disabled={ro} onClick={() => set({ sections: sections.filter((x) => key(x) !== k) })} />
                break
              case 'about':
                text = (
                  <>
                    {t(mh.secAboutText, { count: sentences(draft.about.text) })} ·{' '}
                    <button type="button" className="prose-link border-0 bg-transparent p-0 font-semibold" disabled={ro} onClick={() => setAbout(true)}>
                      {t(mh.editText)}
                    </button>
                  </>
                )
                break
              default:
                text = t(mh.secTiersText)
            }
            return (
              <>
                <span className="tx">
                  <b>{name(s)}</b>
                  <span className="k">{text}</span>
                </span>
                <span className="ct">
                  {extra}
                  {toggle(label)}
                  {after}
                </span>
              </>
            )
          }}
        />
        {free.length && !ro ? (
          <div className="flex flex-wrap items-center gap-2">
            <Select className="w-auto" aria-label={t(mh.addTagRowLabel)} value={addTag} onChange={(e) => setAddTag(e.target.value)}>
              <option value="">{t(mh.addTagRowLabel)}</option>
              {free.map((x) => (
                <option key={x.id} value={x.id}>
                  {x.name}
                </option>
              ))}
            </Select>
            <Button
              size="sm"
              icon="plus"
              disabled={!addTag}
              onClick={() => {
                set({ sections: [...sections, { kind: 'tag', tag: addTag, on: true, count: 3 }] })
                setAddTag('')
              }}
            >
              {t(mh.addTagRow)}
            </Button>
          </div>
        ) : null}
      </div>
      {about ? <AboutDialog draft={draft} set={set} onClose={() => setAbout(false)} /> : null}
    </section>
  )
}

function AboutDialog({ draft, set, onClose }: { draft: HomeNavInput; set: Setter; onClose: () => void }) {
  const t = useT()
  const [a, setA] = useState(draft.about)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mh.aboutDialog)}
      footer={
        <Button
          variant="primary"
          onClick={() => {
            set({ about: a })
            onClose()
          }}
        >
          {t(mh.done)}
        </Button>
      }
    >
      <Field label={t(mh.aboutTitle)}>
        <Input value={a.title} maxLength={80} onChange={(e) => setA({ ...a, title: e.target.value })} />
      </Field>
      <Field label={t(mh.aboutText)}>
        <Textarea value={a.text} maxLength={1200} rows={5} onChange={(e) => setA({ ...a, text: e.target.value })} />
      </Field>
      <Field label={t(mh.aboutLink)} hint={t(mh.aboutLinkHint)}>
        <Input className="mono" value={a.link} maxLength={500} onChange={(e) => setA({ ...a, link: e.target.value })} />
      </Field>
    </Dialog>
  )
}

function NavigationCard({ draft, set, ro, data, host }: { draft: HomeNavInput; set: Setter; ro: boolean; data: HomeNav; host: string }) {
  const t = useT()
  const [kind, setKind] = useState<NavLink['kind']>('tag')
  const [ref, setRef] = useState('')
  const [label, setLabel] = useState('')
  const [url, setUrl] = useState('')
  const [seventh, setSeventh] = useState<{ name: string; detail: string } | null>(null)
  const header = draft.header
  const keyOf = (l: NavLink) => `${l.kind}:${l.ref ?? ''}:${l.url ?? ''}`
  const tagOf = (id?: string) => data.tags.find((x) => x.id === id)
  const pageOf = (slug?: string) => data.pages.find((p) => p.slug === slug)
  const describe = (l: NavLink): { name: string; detail: string } => {
    switch (l.kind) {
      case 'tag': {
        const tag = tagOf(l.ref)
        return { name: l.label || tag?.name || '', detail: t(mh.tagLink, { count: tag?.posts ?? 0 }) }
      }
      case 'page':
        return { name: l.label || pageOf(l.ref)?.title || data.header.find((h) => h.kind === 'page' && h.ref === l.ref)?.name || l.ref || '', detail: t(mh.pageLink, { path: `/${l.ref ?? ''}` }) }
      case 'home':
        return { name: l.label || t(mh.kindHome), detail: t(mh.homeLink) }
      case 'archive':
        return { name: l.label || t(mh.kindArchive), detail: t(mh.archiveLink) }
      default:
        return { name: l.label || bare(l.url ?? ''), detail: t(mh.urlLink, { url: bare(l.url ?? '') }) }
    }
  }
  const candidate = (): NavLink | null => {
    switch (kind) {
      case 'tag':
        return ref ? { kind, ref } : null
      case 'page':
        return ref ? { kind, ref } : null
      case 'url':
        return /^https?:\/\/\S+\.\S+/.test(url.trim()) && label.trim() ? { kind, url: url.trim(), label: label.trim() } : null
      default:
        return { kind }
    }
  }
  const add = () => {
    const l = candidate()
    if (!l) return
    if (header.length >= MAX_HEADER) {
      setSeventh(describe(l))
      return
    }
    set({ header: [...header, l] })
    setRef('')
    setLabel('')
    setUrl('')
  }
  const freeTags = data.tags.filter((x) => !header.some((h) => h.kind === 'tag' && h.ref === x.id))
  const freePages = data.pages.filter((p) => !header.some((h) => h.kind === 'page' && h.ref === p.slug))
  const footer = draft.footer
  const footRow = (id: string, name: string, text: ReactNode, checked: boolean, on: (v: boolean) => void) => (
    <div className="ws-drag" key={id}>
      <span className="tx">
        <b>{name}</b>
        <span className="k">{text}</span>
      </span>
      <input type="checkbox" role="switch" className="tg" aria-label={t(mh.showInFooter, { name })} checked={checked} disabled={ro} onChange={(e) => on(e.target.checked)} />
    </div>
  )
  return (
    <section className="card" aria-labelledby="hn-nav">
      <div className="ws-sec">
        <div className="tx">
          <h2 className="ws-h2" id="hn-nav">
            {t(mh.navTitle)}
          </h2>
          <span className="k">{t(mh.navSub, { host })}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="flex items-center justify-between gap-2">
          <b>{t(mh.headerTitle, { count: header.length, max: MAX_HEADER })}</b>
          {header.length > 1 ? <span className="k">{t(mh.dragOrder)}</span> : null}
        </div>
        <ReorderList
          items={header}
          keyOf={keyOf}
          nameOf={(l) => describe(l).name}
          label={t(mh.headerTitle, { count: header.length, max: MAX_HEADER })}
          disabled={ro}
          onChange={(next) => set({ header: next })}
          render={(l) => {
            const d = describe(l)
            return (
              <>
                <span className="tx">
                  <b>{d.name}</b>
                  <span className="k">{d.detail}</span>
                </span>
                <Remove name={d.name} disabled={ro} onClick={() => set({ header: header.filter((x) => keyOf(x) !== keyOf(l)) })} />
              </>
            )
          }}
        />
        {seventh ? (
          <>
            <div className="ws-drag ghost">
              <span className="tx">
                <b>{seventh.name}</b>
                <span className="k">{t(mh.notAdded, { detail: seventh.detail })}</span>
              </span>
            </div>
            <Note tone="bad">
              <div className="flex flex-col gap-2">
                <b>{t(mh.headerFullTitle, { count: header.length + 1 })}</b>
                <span>{t(mh.headerFullText, { max: MAX_HEADER, name: seventh.name })}</span>
                <span>
                  <Button size="sm" onClick={() => setSeventh(null)}>
                    {t(mh.cancel)}
                  </Button>
                </span>
              </div>
            </Note>
          </>
        ) : null}
        {ro ? null : (
          <div className="flex flex-wrap items-center gap-2">
            <Select
              className="w-auto"
              aria-label={t(mh.kindLabel)}
              value={kind}
              onChange={(e) => {
                setKind(e.target.value as NavLink['kind'])
                setRef('')
              }}
            >
              <option value="tag">{t(mh.kindTag)}</option>
              <option value="page">{t(mh.kindPage)}</option>
              <option value="home">{t(mh.kindHome)}</option>
              <option value="archive">{t(mh.kindArchive)}</option>
              <option value="url">{t(mh.kindUrl)}</option>
            </Select>
            {kind === 'tag' ? (
              <Select className="w-auto" aria-label={t(mh.whichTag)} value={ref} onChange={(e) => setRef(e.target.value)}>
                <option value="">{t(mh.whichTag)}</option>
                {freeTags.map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name}
                  </option>
                ))}
              </Select>
            ) : null}
            {kind === 'page' ? (
              <Select className="w-auto" aria-label={t(mh.whichPage)} value={ref} onChange={(e) => setRef(e.target.value)}>
                <option value="">{freePages.length ? t(mh.whichPage) : t(mh.noPages)}</option>
                {freePages.map((p) => (
                  <option key={p.slug} value={p.slug}>
                    {p.title}
                  </option>
                ))}
              </Select>
            ) : null}
            {kind === 'url' ? (
              <>
                <Input className="w-[140px]" aria-label={t(mh.linkLabel)} placeholder={t(mh.linkLabel)} value={label} maxLength={40} onChange={(e) => setLabel(e.target.value)} />
                <Input className="mono w-[220px]" aria-label={t(mh.linkUrl)} placeholder={'https://'} value={url} onChange={(e) => setUrl(e.target.value)} />
              </>
            ) : null}
            <Button size="sm" icon="plus" disabled={!candidate()} onClick={add}>
              {t(mh.addLink)}
            </Button>
          </div>
        )}
        <span className="k">{t(mh.headerNote)}</span>
      </div>
      <div className="ws-sub">
        <b>{t(mh.footerTitle)}</b>
        <div className="ws-dlist">
          {footRow('archive', t(mh.footArchive), '/archive', footer.archive, (v) => set({ footer: { ...footer, archive: v } }))}
          {footRow('rss', t(mh.footRss), '/feed', footer.rss, (v) => set({ footer: { ...footer, rss: v } }))}
          {footRow('prefs', t(mh.footPrefs), t(mh.footPrefsText), footer.preferences, (v) => set({ footer: { ...footer, preferences: v } }))}
          {footRow('powered', t(mh.footPowered), t(mh.footPoweredText), draft.poweredBy, (v) => set({ poweredBy: v }))}
        </div>
        <Field label={t(mh.privacy)} hint={t(mh.privacyHint)}>
          <Input aria-label={t(mh.privacyLabel)} value={draft.privacyUrl} maxLength={500} disabled={ro} onChange={(e) => set({ privacyUrl: e.target.value })} />
        </Field>
      </div>
    </section>
  )
}

/** The home page as a visitor sees it, rendered by the server with the unsaved settings. */
function LivePreview({ draft, host }: { draft: HomeNavInput; host: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [size, setSize] = useState<'desktop' | 'phone' | 'both'>('both')
  const [body, setBody] = useState(draft)
  useEffect(() => {
    const h = window.setTimeout(() => setBody(draft), 400)
    return () => window.clearTimeout(h)
  }, [draft])
  const preview = useQuery({
    queryKey: ['website-home-preview', slug, JSON.stringify(body)],
    queryFn: async () => {
      const { data } = await http.POST(`${WS}/home-nav/preview`, { params: { path: { workspace: slug } }, body })
      return data!.html
    },
    placeholderData: (p) => p,
    retry: false,
  })
  const pane = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(560)
  useLayoutEffect(() => {
    const el = pane.current
    if (!el) return
    const ro = new ResizeObserver(() => setWidth(el.clientWidth - 36))
    ro.observe(el)
    setWidth(el.clientWidth - 36)
    return () => ro.disconnect()
  }, [])
  const html = preview.data
  const desk = size !== 'phone'
  const phone = size !== 'desktop'
  const deskW = 1280
  const deskScale = Math.max(0.2, Math.min(1, width / deskW))
  const deskH = 4200
  const phoneW = 390
  const phoneScale = size === 'both' ? 0.5 : Math.min(0.85, width / phoneW)
  const phoneH = size === 'both' ? 820 : 1200
  return (
    <section className="card" aria-labelledby="hn-prev">
      <div className="ws-sec" style={{ flexDirection: 'column', alignItems: 'flex-start' }}>
        <div className="tx">
          <h2 className="ws-h2" id="hn-prev">
            {t(mh.previewTitle)}
          </h2>
          <span className="k">{t(mh.previewSub, { host })}</span>
        </div>
        <Segmented
          value={size}
          onChange={setSize}
          label={t(mh.previewSize)}
          items={[
            { value: 'desktop', label: t(mh.desktop) },
            { value: 'phone', label: t(mh.phone) },
            { value: 'both', label: t(mh.both) },
          ]}
        />
      </div>
      <div ref={pane} className={`ws-pane${size === 'both' ? ' both' : ''}`} style={{ justifyContent: size === 'phone' ? 'center' : undefined }} aria-busy={preview.isFetching || undefined}>
        {preview.isError && !html ? <Note tone="bad">{t(mh.previewFailed, { error: errorMessage(preview.error) })}</Note> : null}
        {!html ? <Skeleton w="100%" h={520} r={10} /> : null}
        {html && desk ? (
          <div className="ws-bframe">
            <div className="ws-bbar">
              <span className="u">
                <Icon name="globe" size={11} />
                {host}
              </span>
            </div>
            <SiteFrame html={html} title={t(mh.previewFrame, { host })} width={deskW} scale={deskScale} minHeight={900} maxHeight={deskH} />
          </div>
        ) : null}
        {html && phone ? (
          <div className="ws-pframe" style={{ width: phoneW * phoneScale + 12 }}>
            <SiteFrame html={html} title={t(mh.previewPhone)} width={phoneW} scale={phoneScale} minHeight={phoneH} maxHeight={phoneH} />
          </div>
        ) : null}
      </div>
    </section>
  )
}

function SeoCard({ draft, set, ro, data }: { draft: HomeNavInput; set: Setter; ro: boolean; data: HomeNav }) {
  const t = useT()
  const u = data.urls
  return (
    <section className="card" aria-labelledby="hn-seo">
      <div className="ws-sec">
        <h2 className="ws-h2" id="hn-seo">
          {t(mh.seoTitle)}
        </h2>
      </div>
      <div className="ws-setr" style={{ borderTop: 0 }}>
        <div className="tx">
          <b id="hn-index">{t(mh.indexSite)}</b>
          <span className="k">{t(mh.indexSiteText)}</span>
        </div>
        <div className="ct">
          <input type="checkbox" role="switch" className="tg" aria-labelledby="hn-index" checked={draft.index} disabled={ro} onChange={(e) => set({ index: e.target.checked })} />
        </div>
      </div>
      <dl className="ws-kv pad" style={{ borderTop: '1px solid var(--line-2)' }}>
        <dt>{t(mh.sitemap)}</dt>
        <dd className="mono">{u.sitemap}</dd>
        <dt>{t(mh.siteFeed)}</dt>
        <dd className="mono">{u.feed}</dd>
        <dt>{t(mh.tagFeeds)}</dt>
        <dd className="mono">{u.tagFeed}</dd>
        <dt>{t(mh.archiveFeed)}</dt>
        <dd>
          <T msg={mh.archiveFeedWorks} values={{ url: u.archiveFeed, mono: (c) => <span className="mono">{c}</span> }} />
        </dd>
      </dl>
    </section>
  )
}

function IdentityCard({ draft, set, ro, data }: { draft: HomeNavInput; set: Setter; ro: boolean; data: HomeNav }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const [picking, setPicking] = useState(false)
  const id = draft.identity
  const setId = (patch: Partial<HomeNavInput['identity']>) => set({ identity: { ...id, ...patch } })
  const mark = data.logo ? <img src={data.logo} alt="" style={{ width: '100%', height: '100%', objectFit: 'contain' }} /> : <span>{data.initial}</span>
  return (
    <section className="card" aria-labelledby="hn-id">
      <div className="ws-sec">
        <div className="tx">
          <h2 className="ws-h2" id="hn-id">
            {t(mh.identityTitle)}
          </h2>
          <span className="k">{t(mh.identitySub)}</span>
        </div>
      </div>
      <div className="pad grid gap-5 min-[1101px]:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
        <div className="flex min-w-0 flex-col gap-3">
          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1.7fr)]">
            <Field label={t(mh.siteTitle)} error={id.title.trim() ? undefined : t(mh.titleRequired)}>
              <Input value={id.title} maxLength={80} disabled={ro} onChange={(e) => setId({ title: e.target.value })} />
            </Field>
            <Field label={t(mh.tagline)}>
              <Input value={id.tagline} maxLength={160} disabled={ro} onChange={(e) => setId({ tagline: e.target.value })} />
            </Field>
          </div>
          <Field label={t(mh.description)} hint={t(mh.descriptionHint)}>
            <Textarea value={id.description} maxLength={300} rows={2} disabled={ro} onChange={(e) => setId({ description: e.target.value })} />
          </Field>
        </div>
        <div className="flex min-w-0 flex-col gap-3">
          <div className="flex items-center gap-3">
            <span className="inline-flex flex-none items-center justify-center overflow-hidden rounded-[8px] border border-line bg-[#f5f3ff]" style={{ width: 96, height: 50 }}>
              {id.socialImage ? (
                <img src={id.socialImage} alt={t(mh.socialImageAlt)} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
              ) : (
                <span className="inline-flex h-[24px] w-[24px] items-center justify-center rounded-[6px] bg-ink text-[12px] font-bold text-white">{data.initial}</span>
              )}
            </span>
            <span className="flex min-w-0 flex-1 flex-col">
              <b>{t(mh.socialImage)}</b>
              <span className="k">{t(mh.socialImageText)}</span>
            </span>
            <Button size="sm" disabled={ro} onClick={() => setPicking(true)}>
              {t(mh.replace)}
            </Button>
          </div>
          <div className="flex items-center gap-3">
            <span className="inline-flex h-[32px] w-[32px] flex-none items-center justify-center overflow-hidden rounded-[8px] bg-ink text-[13px] font-bold text-white">{mark}</span>
            <span className="flex min-w-0 flex-1 flex-col">
              <b>{t(mh.favicon)}</b>
              <span className="k">{t(mh.faviconText)}</span>
            </span>
            <Button size="sm" disabled={ro} onClick={() => void navigate({ to: '/w/$workspace/forms/branding', params: { workspace: slug } })}>
              {t(mh.replace)}
            </Button>
          </div>
        </div>
      </div>
      <ImagePickerDialog open={picking} onClose={() => setPicking(false)} onPick={(url) => setId({ socialImage: url })} />
    </section>
  )
}

function TagsCard({ data, overview, ro }: { data: HomeNav; overview: Overview; ro: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const [editing, setEditing] = useState<Tag | 'new' | null>(null)
  const toggleNav = async (tag: Tag, inNav: boolean) => {
    try {
      await http.PUT(`${WS}/tags/{tag}`, { params: { path: { workspace: slug, tag: tag.id } }, body: { name: tag.name, slug: tag.slug, description: tag.description, inNav } })
      void invalidate()
    } catch (e) {
      toast({ kind: 'error', title: t(mh.tagFailed), detail: errorMessage(e) })
    }
  }
  return (
    <section className="card" aria-labelledby="hn-tags">
      <div className="ws-sec">
        <div className="tx">
          <h2 className="ws-h2" id="hn-tags">
            {t(mh.tagsTitle, { count: data.tags.length })}
          </h2>
          <span className="k">{overview.turnedOnAt && overview.progress?.tags.total ? t(mh.tagsSub, { date: dayMonthDate(overview.turnedOnAt) }) : t(mh.tagsSubPlain)}</span>
        </div>
        {ro ? null : (
          <Button size="sm" icon="plus" onClick={() => setEditing('new')}>
            {t(mh.newTag)}
          </Button>
        )}
      </div>
      <div className="tscroll">
        <table className="tbl" aria-label={t(mh.tagsLabel)}>
          <thead>
            <tr>
              <th scope="col">{t(mh.colTag)}</th>
              <th scope="col">{t(mh.colSlug)}</th>
              <th scope="col">{t(mh.colDescription)}</th>
              <th scope="col" className="r">
                {t(mh.colPosts)}
              </th>
              <th scope="col">{t(mh.colNav)}</th>
              <th scope="col">{t(mh.colFeed)}</th>
              {ro ? null : (
                <th scope="col">
                  <span className="vh">{t(mh.edit)}</span>
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {data.tags.length === 0 ? (
              <tr>
                <td colSpan={ro ? 6 : 7} className="k" style={{ padding: '18px 16px' }}>
                  {t(mh.tagsNone)}
                </td>
              </tr>
            ) : null}
            {data.tags.map((x) => (
              <tr key={x.id}>
                <td className="font-semibold">{x.name}</td>
                <td className="mono">{x.slug}</td>
                <td>{x.description}</td>
                <td className="r num">
                  {count(x.posts)}
                  {x.posts === 0 && x.unpublished ? <div className="k">{t(mh.onlyUnpublished)}</div> : null}
                </td>
                <td>
                  <input type="checkbox" role="switch" className="tg" aria-label={t(mh.showInNav, { name: x.name })} checked={x.inNav} disabled={ro} onChange={(e) => void toggleNav(x, e.target.checked)} />
                </td>
                <td className="mono k">{`/tag/${x.slug}/feed`}</td>
                {ro ? null : (
                  <td className="r">
                    <Button size="sm" onClick={() => setEditing(x)}>
                      {t(mh.edit)}
                    </Button>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {editing ? <TagDialog tag={editing === 'new' ? null : editing} host={overview.address.domain || overview.address.host} onClose={() => setEditing(null)} /> : null}
    </section>
  )
}

function slugify(s: string): string {
  return s
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    // "đ" has no decomposition: "Đời sống" → "doi-song", as page addresses do.
    .replace(/đ/g, 'd')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60)
}

/** New tag, or a tag's settings (WebsiteStates · a tag with no posts). */
function TagDialog({ tag, host, onClose }: { tag: Tag | null; host: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const invalidate = useInvalidateWebsite()
  const [name, setName] = useState(tag?.name ?? '')
  const [tagSlug, setTagSlug] = useState(tag?.slug ?? '')
  const [slugEdited, setSlugEdited] = useState(Boolean(tag))
  const [description, setDescription] = useState(tag?.description ?? '')
  const [inNav, setInNav] = useState(tag?.inNav ?? false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [deleting, setDeleting] = useState(false)
  const effSlug = slugEdited ? tagSlug : slugify(name)
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const body = { name: name.trim(), slug: effSlug || undefined, description: description.trim() || undefined, inNav }
      if (tag) await http.PUT(`${WS}/tags/{tag}`, { params: { path: { workspace: slug, tag: tag.id } }, body })
      else await http.POST(`${WS}/tags`, { params: { path: { workspace: slug } }, body })
      toast({ kind: 'success', title: tag ? t(mh.tagSaved) : t(mh.tagCreated), detail: name.trim() })
      void invalidate()
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <>
      <Dialog
        open={!deleting}
        onOpenChange={(o) => !o && onClose()}
        title={tag ? t(mh.tagDialogEdit, { name: tag.name }) : t(mh.tagDialogNew)}
        footer={
          <>
            {tag ? (
              <Button variant="danger" onClick={() => setDeleting(true)}>
                {t(mh.deleteTag)}
              </Button>
            ) : null}
            <Button variant="primary" loading={busy} disabledReason={name.trim() ? undefined : t(mh.tagName)} onClick={() => void save()}>
              {tag ? t(mh.tagSave) : t(mh.tagCreate)}
            </Button>
          </>
        }
      >
        <Field label={t(mh.tagName)} error={error}>
          <Input value={name} maxLength={60} onChange={(e) => setName(e.target.value)} autoFocus />
        </Field>
        <Field label={t(mh.tagSlug)} hint={t(mh.tagSlugHint, { host, slug: effSlug || '…' })}>
          <Input
            className="mono"
            value={effSlug}
            maxLength={60}
            onChange={(e) => {
              setSlugEdited(true)
              setTagSlug(e.target.value)
            }}
          />
        </Field>
        <Field label={t(mh.tagDescription)}>
          <Input value={description} maxLength={200} onChange={(e) => setDescription(e.target.value)} />
        </Field>
        <label className="flex items-center gap-3">
          <input type="checkbox" role="switch" className="tg" checked={inNav} onChange={(e) => setInNav(e.target.checked)} />
          <span className="font-semibold">{t(mh.tagInNav)}</span>
        </label>
        {tag && tag.posts === 0 ? (
          <Note tone="neutral">
            <div className="flex flex-col gap-2">
              <b>{t(mh.tagNoPosts)}</b>
              <span>{t(mh.tagNoPostsText)}</span>
              {tag.unpublished ? (
                <span>
                  <Button
                    size="sm"
                    onClick={() => {
                      onClose()
                      void navigate({ to: '/w/$workspace/forms/website', params: { workspace: slug }, search: { status: 'unpublished', tag: tag.slug } })
                    }}
                  >
                    {t(mh.openUnpublished)}
                  </Button>
                </span>
              ) : null}
            </div>
          </Note>
        ) : null}
      </Dialog>
      {tag ? (
        <ConfirmDialog
          open={deleting}
          onOpenChange={(o) => {
            if (!o) setDeleting(false)
          }}
          tone="bad"
          badge={t(mh.deleteTag)}
          title={t(mh.deleteTagTitle, { name: tag.name })}
          description={t(mh.deleteTagText)}
          confirmLabel={t(mh.deleteTag)}
          onConfirm={async () => {
            await http.DELETE(`${WS}/tags/{tag}`, { params: { path: { workspace: slug, tag: tag.id } } })
            toast({ kind: 'deleted', title: t(mh.tagDeleted), detail: tag.name })
            void invalidate()
            onClose()
          }}
        />
      ) : null}
    </>
  )
}

/** "Weekly Digest and Product Updates" for the tags in a list. */
export function tagNames(tags: Tag[]): string {
  return andList(tags.map((x) => x.name))
}
