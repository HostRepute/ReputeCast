import { useState } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { useBuilder, type PageSettings } from './context'
import { useTiers } from './data'
import { Prop, ToggleRow } from './fields'
import { UiIcon } from './icons'
import { timeZones } from './model'
import { m } from './messages'
import { dayTimeText, zoneLabel } from './wording'

type SettingsInput = Schemas['PageSettingsInput']
type Visibility = PageSettings['visibility']

/** The Page tab's text fields: shown as typed while focused, saved when they're left. */
function useDraftField(stored: string): [string, (v: string) => void, () => string | null, () => void] {
  const [editing, setEditing] = useState<string | null>(null)
  const value = editing ?? stored
  const take = () => {
    const v = editing
    setEditing(null)
    return v !== null && v !== stored ? v : null
  }
  return [value, (v) => setEditing(v), take, () => setEditing(null)]
}

/**
 * The Page tab (PageBuilderStates · Page tab): the title, the address, who
 * can read it, the site's header and navigation, SEO and the social image,
 * and when it publishes. Each change saves the page's settings at once
 * (a version of their own); visibility and navigation need website:manage.
 */
export function PageTab() {
  const t = useT()
  const b = useBuilder()
  const p = b.page
  const s = p.settings
  const tiers = useTiers()
  const canManage = p.publish.canManage && b.canEdit
  const [busy, setBusy] = useState(false)
  const [slugError, setSlugError] = useState<{ slug: string; suggestion: string } | null>(null)
  const [error, setError] = useState('')
  const [name, setName, takeName] = useDraftField(p.name)
  const [slug, setSlug, takeSlug] = useDraftField(p.slug)
  const [seoTitle, setSeoTitle, takeSeoTitle] = useDraftField(s.seoTitle)
  const [seoDesc, setSeoDesc, takeSeoDesc] = useDraftField(s.seoDescription)
  const [navLabel, setNavLabel, takeNavLabel] = useDraftField(s.navLabel)

  const save = async (patch: Partial<SettingsInput>) => {
    const body: SettingsInput = {
      name: p.name,
      slug: p.slug,
      seoTitle: s.seoTitle,
      seoDescription: s.seoDescription,
      socialImage: s.socialImage,
      visibility: s.visibility,
      tiers: s.tiers,
      hideHeader: s.hideHeader,
      showInNav: s.showInNav,
      navLabel: s.navLabel,
      ...patch,
    }
    if (!body.name.trim() || !body.slug.trim()) return
    setBusy(true)
    setError('')
    try {
      const { data } = await http.PUT('/api/v1/workspaces/{workspace}/pages/{page}/settings', { params: { path: { workspace: b.slug, page: p.id } }, body })
      if (data) b.applyDetail(data)
      setSlugError(null)
    } catch (e) {
      if (e instanceof ApiError && e.code === 'slug_taken') {
        setSlugError({ slug: body.slug, suggestion: String((e.problem.meta as Record<string, unknown> | undefined)?.suggestion ?? '') })
      } else if (e instanceof ApiError && e.status === 403) {
        toast({ kind: 'error', title: t(m.settingsFailed), detail: errorMessage(e) })
      } else {
        setError(e instanceof ApiError ? (e.problem.errors?.[0]?.message ?? errorMessage(e)) : errorMessage(e))
      }
    } finally {
      setBusy(false)
    }
  }

  const commit = (field: keyof SettingsInput, take: () => string | null) => {
    const v = take()
    if (v !== null) void save({ [field]: field === 'slug' ? v.trim().toLowerCase() : v })
  }
  const enter = (e: { key: string; currentTarget: HTMLInputElement }) => {
    if (e.key === 'Enter') e.currentTarget.blur()
  }
  const helpers = p.publish.publishers
  const manageWhy = canManage ? '' : helpers.length ? t(m.manageWhy, { helpers: formatList(helpers, 'disjunction') }) : t(m.manageWhyNoHelpers)
  const visibilities: { v: Visibility; label: string }[] = [
    { v: 'public', label: t(m.visPublic) },
    { v: 'members', label: t(m.visMembers) },
    { v: 'paid', label: t(m.visPaid) },
  ]
  const nav = s.navLabel || p.name
  const zones = timeZones([b.publishWhen.tz, b.tz, 'UTC'])
  // The zones' short names on the day it would publish (summer and winter time differ).
  const onDay = new Date(`${b.publishWhen.date || '2026-01-01'}T12:00:00Z`)
  const ro = !b.canEdit
  return (
    <div aria-busy={busy || undefined}>
      <Prop>
        <label className="lbl">
          {t(m.pageTitle)}
          <input className="in" value={name} disabled={ro} onChange={(e) => setName(e.target.value)} onBlur={() => commit('name', takeName)} onKeyDown={enter} maxLength={120} />
        </label>
        <label className="lbl" htmlFor="pb-field-slug">
          {t(m.address)}
          <span className="flex items-center gap-1.5">
            <span className="k mono whitespace-nowrap">{p.host}/</span>
            <input
              id="pb-field-slug"
              className="in mono flex-1"
              style={{ fontSize: 12 }}
              value={slug}
              disabled={ro}
              aria-invalid={slugError ? true : undefined}
              onChange={(e) => setSlug(e.target.value)}
              onBlur={() => commit('slug', takeSlug)}
              onKeyDown={enter}
              maxLength={80}
            />
          </span>
        </label>
        {slugError ? (
          <span className="flex flex-wrap items-center gap-2">
            <span className="err">{t(m.slugTaken, { slug: slugError.slug })}</span>
            {slugError.suggestion ? (
              <button type="button" className="btn sm" onClick={() => void save({ slug: slugError.suggestion })}>
                {t(m.useSuggestion, { slug: slugError.suggestion })}
              </button>
            ) : null}
          </span>
        ) : (
          <span className="k">{t(m.addressHint)}</span>
        )}
      </Prop>
      <Prop>
        <label className="lbl">
          {t(m.visibility)}
          <select className="in" value={s.visibility} disabled={ro || !canManage} onChange={(e) => void save({ visibility: e.target.value as Visibility, tiers: e.target.value === 'paid' ? s.tiers : [] })}>
            {visibilities.map((x) => (
              <option key={x.v} value={x.v} disabled={x.v !== 'public' && !p.websiteOn}>
                {x.label}
              </option>
            ))}
          </select>
        </label>
        {!p.websiteOn ? <span className="k">{t(m.visNeedsWebsite)}</span> : null}
        {s.visibility === 'paid' && tiers.length ? (
          <fieldset className="pb-fieldset">
            <legend className="lbl">{t(m.tiersLabel)}</legend>
            {tiers.map((x) => (
              <label key={x.id} className="flex items-center gap-1.5" style={{ fontSize: 12.5 }}>
                <input
                  type="checkbox"
                  checked={s.tiers.includes(x.id)}
                  disabled={ro || !canManage}
                  onChange={(e) => void save({ tiers: e.target.checked ? [...s.tiers, x.id] : s.tiers.filter((id) => id !== x.id) })}
                />
                {x.name}
              </label>
            ))}
          </fieldset>
        ) : null}
        {manageWhy ? <span className="k">{manageWhy}</span> : null}
      </Prop>
      <Prop>
        <ToggleRow label={t(m.hideHeader)} hint={t(m.hideHeaderHint)} checked={s.hideHeader} disabled={ro} onChange={(v) => void save({ hideHeader: v })} />
        <ToggleRow label={t(m.showInNav)} hint={s.showInNav ? t(m.showInNavHint, { label: nav }) : t(m.showInNavOff)} checked={s.showInNav} disabled={ro || !canManage || (!s.showInNav && p.navFull)} onChange={(v) => void save({ showInNav: v, navLabel: s.navLabel || p.name.slice(0, 40) })} />
        {s.showInNav ? (
          <label className="lbl">
            {t(m.navLabel)}
            <input className="in" value={navLabel} disabled={ro || !canManage} onChange={(e) => setNavLabel(e.target.value)} onBlur={() => commit('navLabel', takeNavLabel)} onKeyDown={enter} maxLength={40} />
          </label>
        ) : null}
        {!s.showInNav && p.navFull ? <span className="k">{t(m.navFull)}</span> : null}
      </Prop>
      <Prop>
        <label className="lbl">
          {t(m.seoTitle)}
          <input className="in" value={seoTitle} disabled={ro} onChange={(e) => setSeoTitle(e.target.value)} onBlur={() => commit('seoTitle', takeSeoTitle)} onKeyDown={enter} maxLength={120} />
        </label>
        <label className="lbl">
          {t(m.seoDescription)}
          <textarea className="in" style={{ height: 56, padding: '6px 9px', resize: 'vertical' }} value={seoDesc} disabled={ro} onChange={(e) => setSeoDesc(e.target.value)} onBlur={() => commit('seoDescription', takeSeoDesc)} maxLength={300} />
        </label>
        <span className="flex flex-wrap items-center gap-2">
          <span className="k min-w-0 flex-1">{s.socialImage ? t(m.socialImage, { name: s.socialImage.split('/').pop() ?? s.socialImage }) : t(m.socialImageNone)}</span>
          {ro ? null : (
            <>
              <button type="button" className="btn sm" onClick={() => b.openDialog({ kind: 'media', pick: (url) => void save({ socialImage: url }) })}>
                {t(m.chooseImage)}
              </button>
              {s.socialImage ? (
                <button type="button" className="btn sm" onClick={() => void save({ socialImage: '' })}>
                  {t(m.removeImage)}
                </button>
              ) : null}
            </>
          )}
        </span>
      </Prop>
      <Prop>
        <span className="t">{t(m.publishTime)}</span>
        {p.status === 'scheduled' && p.publishAt ? <span className="k">{t(m.scheduledFor, { when: dayTimeText(t, p.publishAt) })}</span> : null}
        <div>
          <div className="seg" role="group" aria-label={t(m.publishTime)}>
            <button type="button" aria-pressed={b.publishWhen.when === 'now'} onClick={() => b.setPublishWhen({ ...b.publishWhen, when: 'now' })}>
              {t(m.whenIPublish)}
            </button>
            <button type="button" aria-pressed={b.publishWhen.when === 'at'} onClick={() => b.setPublishWhen({ ...b.publishWhen, when: 'at' })}>
              {t(m.atATime)}
            </button>
          </div>
        </div>
        {b.publishWhen.when === 'at' ? (
          <>
            <div className="row2">
              <label className="lbl">
                {t(m.date)}
                <input type="date" className="in" value={b.publishWhen.date} onChange={(e) => b.setPublishWhen({ ...b.publishWhen, date: e.target.value })} />
              </label>
              <label className="lbl">
                {t(m.time)}
                <input type="time" className="in" value={b.publishWhen.time} onChange={(e) => b.setPublishWhen({ ...b.publishWhen, time: e.target.value })} />
              </label>
            </div>
            <label className="lbl">
              {t(m.timeZone)}
              <select className="in" value={b.publishWhen.tz} onChange={(e) => b.setPublishWhen({ ...b.publishWhen, tz: e.target.value })}>
                {zones.map((z) => (
                  <option key={z} value={z}>
                    {zoneLabel(t, z, onDay)}
                  </option>
                ))}
              </select>
            </label>
            <span className="k">{t(m.publishTimeHint)}</span>
          </>
        ) : null}
      </Prop>
      {error ? (
        <div className="pb-pbody">
          <div className="note b" role="alert">
            <UiIcon name="alert" size={16} color="#B91C1C" />
            <div>
              <b>{t(m.settingsFailed)}</b> {error}
            </div>
          </div>
        </div>
      ) : null}
      {b.canPublishPerm && (p.status === 'published' || p.status === 'scheduled') ? (
        <Prop>
          <span className="t">{p.status === 'scheduled' ? t(m.unscheduleTitle) : t(m.unpublishTitle)}</span>
          <span className="k">{p.status === 'scheduled' ? t(m.unscheduleHint) : t(m.unpublishHint)}</span>
          <button type="button" className="btn sm danger self-start" onClick={() => b.openDialog({ kind: 'unpublish' })}>
            {p.status === 'scheduled' ? t(m.unschedule) : t(m.unpublishOpen)}
          </button>
        </Prop>
      ) : null}
    </div>
  )
}
