import { useQueryClient } from '@tanstack/react-query'
import { useId, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Popover } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { count, dayMonthDate, clock } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { mc, md, mz } from './messages'
import './languages.css'

export type Langs = Schemas['EmailLanguages']
export type Lang = Schemas['EmailLanguage']
type Translate = ReturnType<typeof useT>

const languagesPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages' as const
const templateLanguagesPath = '/api/v1/workspaces/{workspace}/transactional/templates/{template}/languages' as const

/**
 * An email with language versions: a campaign, or a transactional
 * template's draft (the template editor). A plain string is a campaign.
 */
export type LanguageOwner = { kind: 'campaign' | 'template'; id: string }

const ownerOf = (o: LanguageOwner | string): LanguageOwner => (typeof o === 'string' ? { kind: 'campaign', id: o } : o)

/** An email's language versions (the switcher, the banners, the compare view). */
export function useEmailLanguages(owner: LanguageOwner | string) {
  const { slug } = useWorkspace()
  const o = ownerOf(owner)
  const campaign = $api.useQuery('get', languagesPath, { params: { path: { workspace: slug, campaignId: o.id } } }, { staleTime: 5_000, enabled: o.kind === 'campaign' })
  const template = $api.useQuery('get', templateLanguagesPath, { params: { path: { workspace: slug, template: o.id } } }, { staleTime: 5_000, enabled: o.kind === 'template' })
  return o.kind === 'campaign' ? campaign : template
}

/** A campaign's language versions. */
export function useCampaignLanguages(campaignId: string) {
  return useEmailLanguages(campaignId)
}

/** Refreshes the versions after a change, and returns the new list into the cache. */
export function useLanguagesCache(owner: LanguageOwner | string) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const o = ownerOf(owner)
  const key: readonly unknown[] =
    o.kind === 'campaign'
      ? $api.queryOptions('get', languagesPath, { params: { path: { workspace: slug, campaignId: o.id } } }).queryKey
      : $api.queryOptions('get', templateLanguagesPath, { params: { path: { workspace: slug, template: o.id } } }).queryKey
  return {
    set: (data: Langs) => qc.setQueryData(key, data),
    refresh: () => qc.invalidateQueries({ queryKey: ['get', o.kind === 'campaign' ? languagesPath : templateLanguagesPath] }),
  }
}

/** A version's status chip (DSExpansion › Language chip). */
export function statusChip(l: Lang, gets: string, t: Translate): { tone: Tone; text: string; short: string } {
  if (l.held) return { tone: 'neutral', text: t(mc.held, { language: gets }), short: t(mc.held, { language: gets }) }
  switch (l.status) {
    case 'source':
      return { tone: 'info', text: t(mc.source), short: t(mc.source) }
    case 'up_to_date':
      return { tone: 'ok', text: t(mc.upToDate), short: t(mc.upToDate) }
    case 'outdated':
      return { tone: 'warn', text: t(mc.outdatedBlocks, { count: l.attention }), short: t(mc.outdated) }
    case 'draft':
      return { tone: 'neutral', text: t(mc.draft), short: t(mc.draft) }
  }
  return { tone: 'neutral', text: t(mc.missingGets, { language: gets }), short: t(mc.missing) }
}

/** Two panes side by side: Compare languages (the boards' icon). */
export function CompareIcon({ size = 14 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false">
      <rect x="1.5" y="2.5" width="5.6" height="11" rx="1.2" />
      <path d="M9.3 2.5h4a1.2 1.2 0 0 1 1.2 1.2v8.6a1.2 1.2 0 0 1-1.2 1.2h-4z" />
    </svg>
  )
}

/** The name of a language in a list of versions. */
export function nameIn(data: Langs | undefined, code: string): string {
  return data?.versions.find((v) => v.language === code)?.name ?? code
}

function sub(l: Lang, data: Langs, t: Translate): string {
  const gets = nameIn(data, l.gets)
  if (l.held) return t(mz.heldSub, { language: gets })
  switch (l.status) {
    case 'source':
      return t(mz.sourceSub, { language: l.name })
    case 'up_to_date':
    case 'outdated':
      return l.translatedBy && l.translatedAt
        ? t(mz.translatedSub, { name: l.translatedBy, date: `${dayMonthDate(l.translatedAt)} ${clock(l.translatedAt)}`, version: l.translatedFrom })
        : ''
    case 'draft':
      return t(mz.draftSub, { language: gets })
  }
  return t(mz.missingSub, { language: gets })
}

/**
 * The composer header's language switcher (ComposerLanguages): the version
 * open, and a list of every workspace language with its status and
 * readers; Compare languages, Add language, the source language and
 * Translate with AI (not available yet).
 */
export function LanguageSwitcher({
  campaignName,
  data,
  language,
  onPick,
  onCompare,
  onAdd,
}: {
  campaignName: string
  data: Langs
  language: string
  onPick: (language: string) => void
  /** Compare languages (campaigns); without it the menu leaves it out. */
  onCompare?: () => void
  onAdd: () => void
}) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const current = data.versions.find((v) => v.language === language) ?? data.versions[0]
  if (!current || data.versions.length < 2) return null
  const chip = statusChip(current, nameIn(data, current.gets), t)
  const versions = data.versions.filter((v) => v.status !== 'missing' && v.status !== 'source').length
  // On the source, the chip counts the workspace languages with a version.
  const offered = data.versions.filter((v) => v.offered)
  const have = offered.filter((v) => v.status !== 'missing').length
  const onSource = current.status === 'source'
  const pick = (code: string) => {
    setOpen(false)
    onPick(code)
  }
  return (
    <div className="lng-sw-wrap">
      <Popover
        open={open}
        onOpenChange={setOpen}
        align="start"
        className="lng-menu"
        label={t(mz.listLabel, { name: campaignName })}
        trigger={
          <button
            type="button"
            className="btn lng-sw"
            aria-haspopup="listbox"
            aria-expanded={open}
            aria-label={
              onSource
                ? t(mz.switcherSourceLabel, { language: current.name, have, total: offered.length })
                : t(mz.switcherLabel, { language: current.name, status: chip.short.toLowerCase() })
            }
          >
            <Icon name="globe" size={16} />
            <span>{current.name}</span>{' '}
            <Badge tone={chip.tone}>{onSource ? t(mz.sourceChip, { have, total: offered.length }) : chip.short}</Badge>
            <Icon name="chevronDown" size={14} />
          </button>
        }
      >
        <div className="mh flex justify-between">
          <span>{t(mz.versions)}</span>
          {/* A template has no audience: no readers to count. */}
          {data.readers !== undefined && data.readers !== null ? <span>{t(mz.readers)}</span> : null}
        </div>
        <div role="listbox" aria-label={t(mz.listLabel, { name: campaignName })}>
          {data.versions.map((l) => {
            const c = statusChip(l, nameIn(data, l.gets), t)
            const on = l.language === current.language
            return (
              <div
                key={l.language}
                role="option"
                aria-selected={on}
                tabIndex={0}
                className={`lng-opt${on ? ' on' : ''}`}
                onClick={() => pick(l.language)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    pick(l.language)
                  }
                }}
              >
                <span aria-hidden="true">{on ? <Icon name="check" size={14} /> : null}</span>
                <span className="nm">
                  {l.name}
                  <Badge tone={c.tone} style={{ height: 20, fontSize: 11 }}>
                    {c.text}
                  </Badge>
                </span>
                <span className="rc">{l.readers === undefined || l.readers === null ? '' : count(l.readers)}</span>
                <span className="sub">{sub(l, data, t)}</span>
              </div>
            )
          })}
        </div>
        <hr />
        {onCompare ? (
          <button type="button" disabled={versions === 0} onClick={() => (setOpen(false), onCompare())}>
            <CompareIcon />
            {t(mz.compare)}
          </button>
        ) : null}
        {data.canEdit ? (
          <button type="button" onClick={() => (setOpen(false), onAdd())}>
            <Icon name="plus" size={14} />
            {t(mz.add)}
          </button>
        ) : null}
        {data.canEdit && !data.canManageWorkspace ? (
          <div className="flex flex-col gap-0.5" style={{ padding: '6px 10px 4px' }}>
            <span className="flex items-center gap-2" style={{ fontSize: 12.5, color: 'var(--ink-3)' }}>
              <Icon name="plus" size={14} />
              {t(mz.addWorkspace)}
            </span>
            <span className="k">
              <T
                msg={mz.addWorkspaceNeeds}
                values={{ helpers: data.workspaceHelpers.length ? formatList(data.workspaceHelpers, 'disjunction') : 'none', code: (c) => <span className="mono">{c}</span> }}
              />
            </span>
          </div>
        ) : null}
        <hr />
        <div className="flex flex-col gap-0.5" style={{ padding: '6px 10px 4px' }}>
          <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--ink-2)' }}>{t(mz.writtenIn, { language: nameIn(data, data.source) })}</span>
          <span className="k">{t(mz.writtenInDetail)}</span>
        </div>
        <button type="button" disabled aria-describedby="lng-ai-na">
          <Icon name="sparkle" size={14} />
          {t(mz.translateAi)}
          <Badge tone="neutral" id="lng-ai-na" style={{ marginLeft: 'auto', height: 20, fontSize: 11 }}>
            {t(mz.notYet)}
          </Badge>
        </button>
      </Popover>
    </div>
  )
}

/**
 * Add a language version (LanguageStates): a workspace language without a
 * version, and what it starts from. It starts as a draft.
 */
export function AddVersionDialog({
  owner,
  campaignName,
  data,
  open,
  onOpenChange,
  onAdded,
  preselect,
}: {
  owner: LanguageOwner | string
  campaignName: string
  data: Langs
  open: boolean
  onOpenChange: (open: boolean) => void
  onAdded: (language: string) => void
  preselect?: string
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const o = ownerOf(owner)
  const cache = useLanguagesCache(o)
  const name = useId()
  const missing = data.versions.filter((v) => v.status === 'missing' && v.offered)
  const have = data.versions.filter((v) => v.status !== 'missing').map((v) => v.name)
  const [pick, setPick] = useState(preselect && missing.some((m) => m.language === preselect) ? preselect : missing[0]?.language)
  const [start, setStart] = useState<'copy' | 'layout'>('copy')
  const addToCampaign = $api.useMutation('post', languagesPath)
  const addToTemplate = $api.useMutation('post', templateLanguagesPath)
  const adding = addToCampaign.isPending || addToTemplate.isPending
  const chosen = missing.find((m) => m.language === pick)
  const source = nameIn(data, data.source)
  const fallback = nameIn(data, data.fallback)
  const submit = () => {
    if (!chosen) return
    const body = { language: chosen.language, start }
    const done = {
      onSuccess: (out: Langs) => {
        cache.set(out)
        toast({ title: t(mz.added, { language: chosen.name }) })
        onOpenChange(false)
        onAdded(chosen.language)
      },
      onError: (e: unknown) => toast({ kind: 'error', title: t(mz.addFailed, { language: chosen.name }), detail: errorMessage(e) }),
    }
    if (o.kind === 'campaign') addToCampaign.mutate({ params: { path: { workspace: slug, campaignId: o.id } }, body }, done)
    else addToTemplate.mutate({ params: { path: { workspace: slug, template: o.id } }, body }, done)
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{campaignName}</span>}
      title={t(mz.addTitle)}
      hideClose
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(md.cancel)}</Button>
          <Button variant="primary" disabled={!chosen} loading={adding} onClick={submit}>
            {chosen ? t(mz.addButton, { language: chosen.name }) : t(mz.add)}
          </Button>
        </>
      }
    >
      {missing.length ? (
        <>
          <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
            <legend className="lbl" style={{ marginBottom: 8, padding: 0 }}>
              {t(mz.addLanguageLegend)}
            </legend>
            {missing.map((m) => (
              <label key={m.language} className={`lng-rad${pick === m.language ? ' on' : ''}`} style={{ alignItems: 'center' }}>
                <input type="radio" name={name} checked={pick === m.language} onChange={() => setPick(m.language)} />
                <span style={{ flex: 1 }}>
                  <b style={{ fontWeight: 600 }}>{m.name}</b>{' '}
                  {m.native && m.native !== m.name ? (
                    <span className="k" lang={m.language}>
                      {m.native}
                    </span>
                  ) : null}
                </span>
                {m.readers !== undefined && m.readers !== null ? <span className="k">{t(mz.recipients, { count: m.readers })}</span> : null}
              </label>
            ))}
          </fieldset>
          <span className="k">{t(mz.alreadyHave, { languages: formatList(have), count: have.length })}</span>
          <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
            <legend className="lbl" style={{ marginBottom: 8, padding: 0 }}>
              {t(mz.startFrom)}
            </legend>
            <label className={`lng-rad two${start === 'copy' ? ' on' : ''}`}>
              <input type="radio" name={`${name}-start`} checked={start === 'copy'} onChange={() => setStart('copy')} />
              <span style={{ fontWeight: 600 }}>{t(mz.startCopy, { language: source })}</span>
              <span className="k">{t(mz.startCopyDetail, { language: source })}</span>
            </label>
            <label className={`lng-rad two${start === 'layout' ? ' on' : ''}`}>
              <input type="radio" name={`${name}-start`} checked={start === 'layout'} onChange={() => setStart('layout')} />
              <span style={{ fontWeight: 600 }}>{t(mz.startLayout)}</span>
              <span className="k">{t(mz.startLayoutDetail)}</span>
            </label>
          </fieldset>
          <div className="row" style={{ gap: 10 }}>
            <Button size="sm" icon="sparkle" disabledReason={t(mz.aiNotYet)}>
              {t(mz.translateAi)}
            </Button>
            <span className="k">{t(mz.notYet)}</span>
          </div>
          {chosen ? (
            <div className="note i" role="note">
              <Icon name="info" size={16} />
              <div style={{ flex: '1 1 320px', minWidth: 0 }}>{t(mz.draftNote, { language: chosen.name, fallback })}</div>
            </div>
          ) : null}
        </>
      ) : (
        <span className="k">{t(mz.noneToAdd)}</span>
      )}
    </Dialog>
  )
}

/** Marks a version translated, up to date (all or some blocks), held or released. */
export function useVersionActions(owner: LanguageOwner | string) {
  const { slug } = useWorkspace()
  const t = useT()
  const o = ownerOf(owner)
  const cache = useLanguagesCache(o)
  const translated = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/mark-translated')
  const upToDate = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/mark-up-to-date')
  const hold = $api.useMutation('put', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/hold')
  const tTranslated = $api.useMutation('post', '/api/v1/workspaces/{workspace}/transactional/templates/{template}/languages/{language}/mark-translated')
  const tUpToDate = $api.useMutation('post', '/api/v1/workspaces/{workspace}/transactional/templates/{template}/languages/{language}/mark-up-to-date')
  const path = (language: string) => ({ params: { path: { workspace: slug, campaignId: o.id, language } } })
  const tpath = (language: string) => ({ params: { path: { workspace: slug, template: o.id, language } } })
  const done = (title: string) => (out: Langs) => {
    cache.set(out)
    toast({ title })
  }
  const failed = (name: string) => (e: unknown) => toast({ kind: 'error', title: t(mz.markFailed, { language: name }), detail: errorMessage(e) })
  return {
    busy: translated.isPending || upToDate.isPending || hold.isPending || tTranslated.isPending || tUpToDate.isPending,
    markTranslated: (l: Lang) => {
      const then = { onSuccess: done(t(mz.markedTranslated, { language: l.name })), onError: failed(l.name) }
      if (o.kind === 'campaign') translated.mutate(path(l.language), then)
      else tTranslated.mutate({ ...tpath(l.language), body: {} }, then)
    },
    markUpToDate: (l: Lang, blocks?: string[]) => {
      const then = { onSuccess: done(t(mz.marked, { language: l.name })), onError: failed(l.name) }
      if (o.kind === 'campaign') upToDate.mutate({ ...path(l.language), body: { blocks } }, then)
      else tUpToDate.mutate({ ...tpath(l.language), body: { blocks } }, then)
    },
    setHeld: (l: Lang, held: boolean) => hold.mutate({ ...path(l.language), body: { held } }, { onSuccess: (out) => cache.set(out), onError: failed(l.name) }),
  }
}

/**
 * Above the canvas: where the version open stands (ComposerLanguages,
 * LanguageStates): Outdated with Compare and Mark as up to date, a Draft
 * with Mark as translated, Held, or read-only for members without
 * campaigns:edit.
 */
export function LanguageBanner({ owner, data, language, onCompare }: { owner: LanguageOwner | string; data: Langs; language: string; onCompare?: () => void }) {
  const t = useT()
  const actions = useVersionActions(owner)
  const l = data.versions.find((v) => v.language === language)
  if (!l || l.status === 'source') return <ReadOnlyBanner data={data} />
  const source = nameIn(data, data.source)
  const fallback = nameIn(data, data.fallback)
  let body: ReactNode = null
  if (l.held) {
    body = (
      <Banner tone="n" title={t(mz.heldTitle, { language: l.name, fallback: nameIn(data, l.gets) })} text={t(mz.heldText, { language: l.name })}>
        {data.canEdit ? (
          <Button size="sm" loading={actions.busy} onClick={() => actions.setHeld(l, false)}>
            {t(mz.release, { language: l.name })}
          </Button>
        ) : null}
      </Banner>
    )
  } else if (l.status === 'outdated') {
    const text =
      l.translatedBy && l.translatedAt
        ? t(mz.outdatedText, { count: l.attention, source, name: l.translatedBy, date: dayMonthDate(l.translatedAt), language: l.name })
        : t(mz.outdatedTextNoName, { count: l.attention, source, language: l.name })
    body = (
      <Banner tone="w" title={t(mz.outdatedTitle, { language: l.name })} text={text}>
        {onCompare ? (
          <Button size="sm" variant="primary" onClick={onCompare}>
            <CompareIcon />
            {t(mz.compareButton)}
          </Button>
        ) : null}
        {data.canEdit ? (
          <Button size="sm" loading={actions.busy} onClick={() => actions.markUpToDate(l)}>
            {t(mz.markUpToDate)}
          </Button>
        ) : null}
      </Banner>
    )
  } else if (l.status === 'draft') {
    body = (
      <Banner
        tone="n"
        title={t(mz.draftTitle, { language: l.name, translated: l.translated, blocks: l.blocks })}
        text={t(mz.draftText, { fallback, still: l.still, source })}
      >
        {data.canEdit ? (
          <Button size="sm" variant="primary" loading={actions.busy} onClick={() => actions.markTranslated(l)}>
            {t(mz.markTranslated)}
          </Button>
        ) : null}
        {onCompare ? (
          <Button size="sm" onClick={onCompare}>
            {t(mz.compareWith, { source })}
          </Button>
        ) : null}
      </Banner>
    )
  } else if (l.status === 'up_to_date') {
    body = (
      <div className="lng-ok k" role="status">
        <Icon name="check" size={14} />
        {t(mz.upToDateNote, { version: l.translatedFrom })}
      </div>
    )
  }
  return (
    <>
      <ReadOnlyBanner data={data} />
      {body}
    </>
  )
}

function Banner({ tone, title, text, children }: { tone: 'w' | 'n'; title: string; text: string; children?: ReactNode }) {
  return (
    <div className={`note ${tone} lng-banner`} role="status">
      <Icon name={tone === 'w' ? 'alertCircle' : 'info'} size={16} />
      <div className="flex min-w-0 flex-col gap-1" style={{ flex: '1 1 320px' }}>
        <span style={{ fontWeight: 700 }}>{title}</span>
        <div>
          {text}
          {children ? <div className="mt-2 flex flex-wrap gap-2">{children}</div> : null}
        </div>
      </div>
    </div>
  )
}

/** For members who can't edit: they switch languages and compare them. */
function ReadOnlyBanner({ data }: { data: Langs }) {
  const t = useT()
  const { me } = useWorkspace()
  const { data: who } = useMe()
  if (data.canEdit) return null
  return (
    <div className="note n lng-banner" role="note">
      <Icon name="info" size={16} />
      <div className="flex min-w-0 flex-col gap-1" style={{ flex: '1 1 320px' }}>
        <span style={{ fontWeight: 700 }}>{t(mz.readOnlyTitle, { name: who?.user.name ?? '', role: me.workspace.roleName })}</span>
        <div>
          <T
            msg={mz.readOnlyText}
            values={{ helpers: data.editHelpers.length ? formatList(data.editHelpers, 'disjunction') : 'none', code: (c) => <span className="mono">{c}</span> }}
          />
        </div>
      </div>
    </div>
  )
}

/** In place of the canvas when the language has no version (LanguageStates). */
export function MissingLanguage({ data, language, onAdd }: { data: Langs; language: string; onAdd: () => void }) {
  const t = useT()
  const l = data.versions.find((v) => v.language === language)
  if (!l) return null
  const fallback = nameIn(data, l.gets)
  return (
    <div className="lng-missing" role="status">
      <div className="card lng-empty">
        <span className="tt">{t(mz.missingTitle, { language: l.name })}</span>
        <span className="ds">{t(mz.missingText, { count: l.readers ?? 0, language: l.name, fallback })}</span>
        <div className="flex flex-wrap justify-center gap-2">
          {data.canEdit ? (
            <Button variant="primary" onClick={onAdd}>
              {t(mz.addButton, { language: l.name })}
            </Button>
          ) : null}
          <Button icon="sparkle" disabled aria-describedby="lng-missing-ai">
            {t(mz.translateAi)}
          </Button>
        </div>
        <span className="k" id="lng-missing-ai">
          {t(mz.aiNotYet)}
        </span>
      </div>
    </div>
  )
}
