import { useQueryClient } from '@tanstack/react-query'
import { useId, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, dayMonthDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { contrast, grade, hexColour, ratioText } from './contrast'
import { mk } from './messages'
import { SyncedBlockDialog } from './SyncedBlockDialog'
import './brandkit.css'

type Kit = Schemas['BrandKit']
type Imported = Schemas['BrandKitImport']
type Logo = Schemas['BrandLogo']
type Font = Schemas['BrandFont']
type Colors = Schemas['BrandColors']
type Shape = 'square' | 'rounded' | 'pill'

/** The email-safe fallbacks (brandkit.Fallbacks). */
export const FALLBACKS = [
  "Georgia, 'Times New Roman', serif",
  "'Times New Roman', serif",
  'Arial, Helvetica, sans-serif',
  'Helvetica, Arial, sans-serif',
  'Verdana, Geneva, sans-serif',
  "'Trebuchet MS', Arial, sans-serif",
  'Tahoma, Geneva, sans-serif',
] as const

/** What the page edits: the kit, with logos as library images. */
export interface Draft {
  logoLight?: Logo
  logoDark?: Logo
  logoAlt: string
  colors: Colors
  heading: Font
  body: Font
  button: { shape: Shape; size: 14 | 15 }
  website: string
}

export function draftOf(k: Kit | Imported, website?: string): Draft {
  const style = 'style' in k ? k.style : k
  return {
    logoLight: k.logoLight ?? undefined,
    logoDark: k.logoDark ?? undefined,
    logoAlt: k.logoAlt ?? '',
    colors: { ...style.colors },
    heading: { ...style.heading },
    body: { ...style.body },
    button: { shape: style.button.shape as Shape, size: style.button.size as 14 | 15 },
    website: website ?? k.website ?? '',
  }
}

function inputOf(d: Draft): Schemas['BrandKitInput'] {
  return {
    logoLightId: d.logoLight?.mediaId ?? '',
    logoDarkId: d.logoDark?.mediaId ?? '',
    logoAlt: d.logoAlt.trim(),
    colors: d.colors,
    heading: { name: d.heading.name, url: d.heading.url || undefined, fallback: d.heading.fallback as Font['fallback'] },
    body: { name: d.body.name, url: d.body.url || undefined, fallback: d.body.fallback as Font['fallback'] },
    button: d.button,
    website: d.website.trim(),
  }
}

/** The first family of a stack, as people name it: Georgia. */
function firstFamily(stack: string): string {
  return stack.split(',')[0]!.replace(/'/g, '').trim()
}

const radius: Record<Shape, string> = { square: '0', rounded: '6px', pill: '999px' }

/** Settings › Brand kit (BrandKit, BrandKitPhone). */
export function BrandKitPage() {
  const t = useT()
  const { slug, me } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/brand-kit', { params: { path: { workspace: slug } } })
  return (
    <Page title={t(mk.settings)} subtitle={me.workspace.name} tabs={settingsTabs(slug)} documentTitle={t(mk.documentTitle)}>
      {q.isError ? (
        <ErrorPanel title={t(mk.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.isPending ? (
        <div className="flex flex-col gap-5">
          <Skeleton h={64} />
          <Skeleton h={320} />
        </div>
      ) : (
        <Editor key={q.data.updatedAt ?? 'defaults'} kit={q.data} />
      )}
    </Page>
  )
}

function Editor({ kit }: { kit: Kit }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const canEdit = useCan('templates:edit')
  const saved = draftOf(kit)
  const [draft, setDraft] = useState<Draft>(saved)
  const [imported, setImported] = useState<Imported | null>(null)
  const [busy, setBusy] = useState(false)
  const dirty = JSON.stringify(inputOf(draft)) !== JSON.stringify(inputOf(saved))
  const set = (patch: Partial<Draft>) => setDraft((d) => ({ ...d, ...patch }))
  const valid = Object.values(draft.colors).every((c) => hexColour.test(c))
  const save = async () => {
    setBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/brand-kit', { params: { path: { workspace: slug } }, body: inputOf(draft) })
      setImported(null)
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/brand-kit'] })
      toast({ kind: 'success', title: t(mk.saved), detail: t(mk.savedDetail) })
    } catch (e) {
      toast({ kind: 'error', title: t(mk.saveFailed), detail: errorMessage(e) })
      throw e
    } finally {
      setBusy(false)
    }
  }
  const discard = () => {
    setDraft(saved)
    setImported(null)
  }
  const status = imported ? (
    <Badge tone="info" dot>
      {t(mk.imported)}
    </Badge>
  ) : dirty ? (
    <Badge tone="warn" dot>
      {t(mk.unsaved)}
    </Badge>
  ) : !kit.saved ? (
    <Badge tone="neutral">{t(mk.notSetUp)}</Badge>
  ) : null
  return (
    <div className="bk flex flex-col gap-5">
      <UnsavedChangesGuard dirty={dirty} description={t(mk.leave)} saveLabel={t(mk.save)} onSave={save} />
      <div className="bkhead">
        <div className="flex min-w-0 flex-col gap-1" style={{ flex: '1 1 420px' }}>
          <div className="flex flex-wrap items-center gap-2.5">
            <h2 className="ttl">{t(mk.title)}</h2>
            {status}
          </div>
          <p className="lead">{t(mk.lead)}</p>
          {!canEdit ? <p className="k m-0">{t(mk.whoEdits)}</p> : null}
        </div>
        {canEdit ? (
          <div className="flex flex-wrap gap-2">
            {imported || dirty ? (
              <Button onClick={discard} disabled={busy}>
                {imported ? t(mk.discardImport) : t(mk.discardChanges)}
              </Button>
            ) : null}
            <Button variant="primary" onClick={() => void save().catch(() => {})} disabled={busy || !valid || (!dirty && kit.saved)} aria-busy={busy || undefined}>
              {busy ? t(mk.saving) : t(mk.save)}
            </Button>
          </div>
        ) : null}
      </div>
      <div className="bkgrid">
        <div className="flex min-w-0 flex-col gap-5">
          <ImportCard
            website={draft.website}
            canEdit={canEdit}
            imported={imported}
            onImported={(found) => {
              setImported(found)
              setDraft((d) => ({ ...draftOf(found, found.website), logoLight: found.logoLight ?? d.logoLight, logoDark: found.logoDark ?? d.logoDark, logoAlt: found.logoAlt || d.logoAlt }))
            }}
          />
          <LogoCard draft={draft} set={set} canEdit={canEdit} />
          <ColoursCard colors={draft.colors} onChange={(colors) => set({ colors })} canEdit={canEdit} />
          <FontsCard draft={draft} set={set} canEdit={canEdit} />
          <ButtonCard draft={draft} set={set} canEdit={canEdit} />
          <SyncedCard kit={kit} />
        </div>
        <aside className="flex min-w-0 flex-col gap-3" aria-labelledby="bk-prev">
          <PreviewCard draft={draft} />
          <div className="note n">
            <div>{t(mk.savingNote)}</div>
          </div>
        </aside>
      </div>
    </div>
  )
}

function ImportCard({ website, canEdit, imported, onImported }: { website: string; canEdit: boolean; imported: Imported | null; onImported: (found: Imported) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const [site, setSite] = useState(website)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const host = site.trim().replace(/^https?:\/\//, '').replace(/\/.*$/, '')
  const run = async () => {
    setBusy(true)
    setError('')
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/brand-kit/import', { params: { path: { workspace: slug } }, body: { website: site.trim() } })
      if (data) onImported(data)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const logos = imported ? (imported.logoLight && imported.logoDark ? 'both' : imported.logoLight || imported.logoDark ? 'light' : 'none') : 'none'
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <CardSection title={<span id={`${id}-h`}>{host ? t(mk.importTitleSite, { site: host }) : t(mk.importTitle)}</span>} caption={t(mk.importLead)} />
      <div className="pad flex flex-col gap-3.5">
        <form
          className="improw flex gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            if (site.trim()) void run()
          }}
        >
          <label className="lbl">
            {t(mk.website)}
            <input className="in" type="text" inputMode="url" value={site} onChange={(e) => setSite(e.target.value)} placeholder={t(mk.websitePlaceholder)} disabled={!canEdit} />
          </label>
          <Button type="submit" disabled={!canEdit || busy || !site.trim()} aria-busy={busy || undefined}>
            {busy ? t(mk.importing, { site: host }) : imported ? t(mk.importAgain) : t(mk.import)}
          </Button>
        </form>
        {error ? (
          <div className="note w" role="alert">
            <div>
              <b>{t(mk.importFailed)}</b> {error}
            </div>
          </div>
        ) : null}
        {imported ? (
          <>
            <div className="note g" role="status">
              <div>
                <T
                  msg={mk.found}
                  values={{ site: imported.website, time: clock(imported.foundAt), logo: logos, colors: imported.colors.length, fonts: imported.fonts.length, b: (c: ReactNode) => <b>{c}</b> }}
                />
              </div>
            </div>
            <div className="det">
              <div className="dcell">
                <span className="dl">{t(mk.logo)}</span>
                <div className="flex gap-2">
                  {imported.logoLight ? (
                    <span className="lgt" style={{ background: '#FFFFFF' }}>
                      <img src={imported.logoLight.url} alt={t(mk.logoLightAria, { name: imported.logoAlt || imported.website })} style={{ maxHeight: 24, maxWidth: 96 }} />
                    </span>
                  ) : null}
                  {imported.logoDark ? (
                    <span className="lgt" style={{ background: '#111827', borderColor: '#111827' }}>
                      <img src={imported.logoDark.url} alt={t(mk.logoDarkAria, { name: imported.logoAlt || imported.website })} style={{ maxHeight: 24, maxWidth: 96 }} />
                    </span>
                  ) : null}
                  {!imported.logoLight && !imported.logoDark ? <span className="k">{t(mk.noLogo)}</span> : null}
                </div>
              </div>
              <div className="dcell">
                <span className="dl">{t(mk.colours)}</span>
                <div className="flex flex-wrap gap-1.5">
                  {imported.colors.map((c) => (
                    <span key={c.hex} className="dsw">
                      <i style={{ background: c.hex }} />
                      <span className="mono">{c.hex}</span>
                      {!c.used ? <span className="k">{t(mk.notUsed)}</span> : null}
                    </span>
                  ))}
                </div>
              </div>
              <div className="dcell">
                <span className="dl">{t(mk.fonts)}</span>
                <span className="font-semibold">{imported.fonts.join(' · ') || '—'}</span>
              </div>
            </div>
            {imported.notes.map((n) => (
              <div key={n} className="note w">
                <div>{n}</div>
              </div>
            ))}
          </>
        ) : null}
      </div>
    </section>
  )
}

/** Uploads an image to the media library (validated, resized and compressed on the server). */
async function uploadImage(slug: string, file: File): Promise<Logo> {
  const fd = new FormData()
  fd.append('file', file)
  const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/media`, { method: 'POST', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
  const body = (await r.json().catch(() => ({}))) as Schemas['MediaAsset'] & { detail?: string; title?: string; errors?: { message?: string }[] }
  if (!r.ok) throw new Error(body.errors?.[0]?.message || body.detail || body.title || String(r.status))
  return { mediaId: body.id, url: body.url, fileName: body.fileName, width: body.width, height: body.height }
}

function LogoCell({ logo, dark, label, onChange, canEdit, alt }: { logo?: Logo; dark?: boolean; label: string; onChange: (l?: Logo) => void; canEdit: boolean; alt: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const input = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const pick = async (files: FileList | null) => {
    const file = files?.[0]
    if (!file) return
    setBusy(true)
    try {
      onChange(await uploadImage(slug, file))
    } catch (e) {
      toast({ kind: 'error', title: t(mk.uploadFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
      if (input.current) input.current.value = ''
    }
  }
  return (
    <div className="lcell">
      <div className="lprev" style={dark ? { background: '#111827', borderColor: '#111827' } : { background: '#FFFFFF' }}>
        {logo ? <img src={logo.url} alt={alt} style={{ maxWidth: '70%', maxHeight: 64 }} /> : <span className={dark ? 'text-[#C3C2B7]' : 'k'}>{t(mk.noLogo)}</span>}
      </div>
      <div className="flex items-center justify-between gap-2">
        <span className="flex min-w-0 flex-col">
          <span className="font-semibold">{label}</span>
          {logo ? <span className="k mono truncate">{t(mk.logoMeta, { name: logo.fileName, width: logo.width, height: logo.height })}</span> : null}
        </span>
        {canEdit ? (
          <span className="flex gap-1.5">
            {logo ? (
              <Button size="sm" onClick={() => onChange(undefined)} disabled={busy}>
                {t(mk.remove)}
              </Button>
            ) : null}
            <Button size="sm" onClick={() => input.current?.click()} disabled={busy} aria-busy={busy || undefined}>
              {busy ? t(mk.uploading) : logo ? t(mk.replace) : t(mk.upload)}
            </Button>
            <input ref={input} type="file" accept="image/png,image/jpeg,image/gif" className="vh" tabIndex={-1} aria-label={label} onChange={(e) => void pick(e.target.files)} />
          </span>
        ) : null}
      </div>
    </div>
  )
}

function LogoCard({ draft, set, canEdit }: { draft: Draft; set: (p: Partial<Draft>) => void; canEdit: boolean }) {
  const t = useT()
  const id = useId()
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <CardSection title={<span id={`${id}-h`}>{t(mk.logo)}</span>} caption={t(mk.logoLead)} />
      <div className="pad flex flex-col gap-3.5">
        <div className="lgrid">
          <LogoCell logo={draft.logoLight} label={t(mk.forLight)} onChange={(logoLight) => set({ logoLight })} canEdit={canEdit} alt={draft.logoAlt} />
          <LogoCell logo={draft.logoDark} dark label={t(mk.forDark)} onChange={(logoDark) => set({ logoDark })} canEdit={canEdit} alt={draft.logoAlt} />
        </div>
        <div className="lbl" style={{ maxWidth: 420 }}>
          <label htmlFor={`${id}-alt-in`}>{t(mk.altText)}</label>
          <input id={`${id}-alt-in`} className="in" type="text" value={draft.logoAlt} maxLength={120} onChange={(e) => set({ logoAlt: e.target.value })} disabled={!canEdit} aria-describedby={`${id}-alt`} />
          <span id={`${id}-alt`} className="k" style={{ fontWeight: 500 }}>
            {t(mk.altHelp)}
          </span>
        </div>
        <span className="k" style={{ fontWeight: 500 }}>
          {t(mk.darkNote)}
        </span>
      </div>
    </section>
  )
}

function ContrastBadge({ ratio, msg }: { ratio: number; msg: Parameters<ReturnType<typeof useT>>[0] }) {
  const t = useT()
  const g = grade(ratio)
  return <Badge tone={g === 'low' ? 'warn' : 'ok'}>{t(msg, { ratio: ratioText(ratio), grade: g })}</Badge>
}

function ColourRow({ value, onChange, name, hint, aria, ratio, msg, canEdit }: {
  value: string
  onChange: (v: string) => void
  name: string
  hint: string
  aria: string
  ratio: number
  msg: Parameters<ReturnType<typeof useT>>[0]
  canEdit: boolean
}) {
  const t = useT()
  const ok = hexColour.test(value)
  return (
    <div className="crow">
      <span className="csw" style={{ background: ok ? value : '#FFFFFF' }} aria-hidden="true" />
      <span className="cname">
        <b>{name}</b>
        <span className="k">{hint}</span>
      </span>
      <input className="in mono cin" aria-label={aria} value={value} onChange={(e) => onChange(e.target.value.trim())} disabled={!canEdit} aria-invalid={!ok || undefined} maxLength={7} spellCheck={false} />
      {ok ? <ContrastBadge ratio={ratio} msg={msg} /> : <Badge tone="bad">{t(mk.badColour)}</Badge>}
    </div>
  )
}

function ColoursCard({ colors, onChange, canEdit }: { colors: Colors; onChange: (c: Colors) => void; canEdit: boolean }) {
  const t = useT()
  const id = useId()
  const set = (k: keyof Colors) => (v: string) => onChange({ ...colors, [k]: v.startsWith('#') ? v.toUpperCase() : `#${v.toUpperCase()}` })
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <CardSection title={<span id={`${id}-h`}>{t(mk.colours)}</span>} caption={t(mk.coloursLead)} />
      <div className="pad flex flex-col">
        <ColourRow value={colors.button} onChange={set('button')} name={t(mk.button)} hint={t(mk.buttonHint)} aria={t(mk.buttonAria)} ratio={contrast('#FFFFFF', colors.button)} msg={mk.contrastButton} canEdit={canEdit} />
        <ColourRow value={colors.text} onChange={set('text')} name={t(mk.text)} hint={t(mk.textHint)} aria={t(mk.textAria)} ratio={contrast(colors.text, '#FFFFFF')} msg={mk.contrastOnWhite} canEdit={canEdit} />
        <ColourRow value={colors.background} onChange={set('background')} name={t(mk.background)} hint={t(mk.backgroundHint)} aria={t(mk.backgroundAria)} ratio={contrast(colors.text, colors.background)} msg={mk.contrastOnIt} canEdit={canEdit} />
        <ColourRow value={colors.link} onChange={set('link')} name={t(mk.link)} hint={t(mk.linkHint)} aria={t(mk.linkAria)} ratio={contrast(colors.link, '#FFFFFF')} msg={mk.contrastOnWhite} canEdit={canEdit} />
      </div>
    </section>
  )
}

function FontRow({ font, onChange, label, site, sample, canEdit, heading }: { font: Font; onChange: (f: Font) => void; label: string; site: string; sample: string; canEdit: boolean; heading?: boolean }) {
  const t = useT()
  const fallbackName = firstFamily(font.fallback)
  return (
    <div className="fgrid">
      <div className="flex min-w-0 flex-col gap-2.5">
        <label className="lbl">
          {label}
          <select className="in" value={font.name ? 'web' : 'none'} onChange={(e) => onChange(e.target.value === 'none' ? { ...font, name: '', url: undefined } : font)} disabled={!canEdit || !font.name}>
            {font.name ? <option value="web">{site ? t(mk.webFrom, { name: font.name, site }) : t(mk.webFont, { name: font.name })}</option> : null}
            <option value="none">{t(mk.noWebFont)}</option>
          </select>
        </label>
        <label className="lbl">
          {t(mk.fallback)}
          <select className="in" value={font.fallback} onChange={(e) => onChange({ ...font, fallback: e.target.value as Font['fallback'] })} disabled={!canEdit}>
            {FALLBACKS.map((f) => (
              <option key={f} value={f}>
                {f}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="fsamp">
        <span className="k">{t(mk.sample, { font: fallbackName })}</span>
        <span style={heading ? { fontFamily: font.fallback, fontSize: 24, lineHeight: 1.2, fontWeight: 700 } : { fontFamily: font.fallback, fontSize: 15, lineHeight: 1.55 }}>{sample}</span>
      </div>
    </div>
  )
}

function FontsCard({ draft, set, canEdit }: { draft: Draft; set: (p: Partial<Draft>) => void; canEdit: boolean }) {
  const t = useT()
  const id = useId()
  const site = draft.website.replace(/^https?:\/\//, '').replace(/\/.*$/, '')
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <CardSection title={<span id={`${id}-h`}>{t(mk.fonts)}</span>} caption={t(mk.fontsLead)} />
      <div className="pad flex flex-col gap-4">
        <FontRow font={draft.heading} onChange={(heading) => set({ heading })} label={t(mk.headingFont)} site={site} sample={t(mk.sampleHeading)} canEdit={canEdit} heading />
        <FontRow font={draft.body} onChange={(body) => set({ body })} label={t(mk.bodyFont)} site={site} sample={t(mk.sampleBody)} canEdit={canEdit} />
      </div>
    </section>
  )
}

function SampleButton({ draft }: { draft: Draft }) {
  const t = useT()
  return (
    <span className="ebtn" style={{ borderRadius: radius[draft.button.shape], background: draft.colors.button, fontFamily: draft.body.fallback, fontSize: draft.button.size }}>
      {t(mk.sampleButton)}
    </span>
  )
}

function ButtonCard({ draft, set, canEdit }: { draft: Draft; set: (p: Partial<Draft>) => void; canEdit: boolean }) {
  const t = useT()
  const id = useId()
  const shapes: [Shape, string][] = [
    ['square', t(mk.square)],
    ['rounded', t(mk.rounded)],
    ['pill', t(mk.pill)],
  ]
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <CardSection title={<span id={`${id}-h`}>{t(mk.buttonStyle)}</span>} caption={t(mk.buttonLead)} />
      <div className="pad flex flex-col gap-3.5">
        <div className="flex flex-wrap items-end gap-x-5 gap-y-3">
          <div className="flex flex-col gap-1">
            <span className="lbl" id={`${id}-shape`}>
              {t(mk.shape)}
            </span>
            <div className="segm" role="group" aria-labelledby={`${id}-shape`}>
              {shapes.map(([s, label]) => (
                <button key={s} type="button" aria-pressed={draft.button.shape === s} onClick={() => set({ button: { ...draft.button, shape: s } })} disabled={!canEdit}>
                  {label}
                </button>
              ))}
            </div>
          </div>
          <label className="lbl" style={{ flex: '0 1 220px' }}>
            {t(mk.buttonText)}
            <select className="in" value={draft.button.size} onChange={(e) => set({ button: { ...draft.button, size: Number(e.target.value) as 14 | 15 } })} disabled={!canEdit}>
              <option value={15}>{t(mk.size15)}</option>
              <option value={14}>{t(mk.size14)}</option>
            </select>
          </label>
        </div>
        <div className="bprev">
          <div className="bpc" style={{ background: '#FFFFFF' }}>
            <SampleButton draft={draft} />
            <span className="k">{t(mk.onWhite)}</span>
          </div>
          <div className="bpc" style={{ background: hexColour.test(draft.colors.background) ? draft.colors.background : '#FFFFFF' }}>
            <SampleButton draft={draft} />
            <span className="k">{t(mk.onBackground)}</span>
          </div>
        </div>
        <span className="k" style={{ fontWeight: 500 }}>
          {t(mk.buttonNote, { color: draft.colors.button, ratio: ratioText(contrast('#FFFFFF', draft.colors.button)) })}
        </span>
      </div>
    </section>
  )
}

function SyncedCard({ kit }: { kit: Kit }) {
  const t = useT()
  const id = useId()
  const canEdit = useCan('campaigns:edit')
  const [editing, setEditing] = useState<Schemas['SyncedBlock'] | null>(null)
  return (
    <section className="card" aria-labelledby={`${id}-h`}>
      <CardSection title={<span id={`${id}-h`}>{t(mk.synced)}</span>} caption={t(mk.syncedLead)} />
      <div className="pad flex flex-col">
        {!kit.synced.length ? <p className="k m-0">{t(mk.syncedNone)}</p> : null}
        {kit.synced.map((b) => {
          const u = b.uses
          const total = u.drafts + u.scheduled + u.held + u.sent
          return (
            <div key={b.kind} className="sbrow">
              <span className="sbt" aria-hidden="true">
                {b.kind === 'header' ? (
                  <span className="flex w-full items-center justify-between">
                    <i style={{ height: 6, width: 22, background: kit.colors.button }} />
                    <i style={{ height: 3, width: 30, background: '#D4D8DF' }} />
                  </span>
                ) : (
                  <>
                    <i style={{ height: 4, width: '60%', background: '#C9CED6' }} />
                    <i style={{ height: 3, width: '80%', background: '#D4D8DF' }} />
                    <i style={{ height: 3, width: '40%', background: '#D4D8DF' }} />
                  </>
                )}
              </span>
              <span className="flex min-w-0 flex-col gap-0.5" style={{ flex: '1 1 260px' }}>
                <span className="flex flex-wrap items-baseline gap-2">
                  <b>{b.kind === 'header' ? t(mk.header) : t(mk.footer)}</b>
                  <span className="k">{t(mk.usedIn, { count: total })}</span>
                </span>
                <span className="k">
                  {b.kind === 'header' ? t(mk.headerWhat) : t(mk.footerWhat)} ·{' '}
                  {b.updatedBy ? t(mk.edited, { date: dayMonthDate(b.updatedAt), name: b.updatedBy }) : t(mk.editedAnon, { date: dayMonthDate(b.updatedAt) })}
                </span>
                <span className="k">{t(mk.onSave, { drafts: u.drafts, scheduled: u.scheduled, sent: u.sent })}</span>
                {u.held ? <span className="k">{t(mk.onSaveHeld, { held: u.held })}</span> : null}
              </span>
              {canEdit ? (
                <Button size="sm" onClick={() => setEditing(b)}>
                  {t(mk.edit)}
                </Button>
              ) : null}
            </div>
          )
        })}
        {kit.synced.length && !canEdit ? <p className="k m-0 mt-2">{t(mk.whoEditsSynced)}</p> : null}
      </div>
      {editing ? <SyncedBlockDialog block={editing} onClose={() => setEditing(null)} /> : null}
    </section>
  )
}

function PreviewCard({ draft }: { draft: Draft }) {
  const t = useT()
  const { me } = useWorkspace()
  const c = draft.colors
  const ok = (v: string, d: string) => (hexColour.test(v) ? v : d)
  const body = draft.body.fallback
  return (
    <section className="card bkp">
      <CardSection title={<span id="bk-prev">{t(mk.previewTitle)}</span>} caption={t(mk.previewLead)} />
      <div className="pad" style={{ background: ok(c.background, '#F3F4F6'), borderRadius: '0 0 10px 10px' }}>
        <div className="pmail">
          <div className="flex items-center justify-between" style={{ padding: '12px 16px', borderBottom: '1px solid #EEF0F3' }}>
            {draft.logoLight ? (
              <img src={draft.logoLight.url} alt={draft.logoAlt} style={{ maxWidth: 120, maxHeight: 32 }} />
            ) : (
              <b style={{ color: ok(c.text, '#1F2937'), fontFamily: body }}>{draft.logoAlt || me.workspace.name}</b>
            )}
            <span style={{ fontFamily: body, fontSize: 11, color: '#4B5563' }}>{t(mk.previewNav)}</span>
          </div>
          <div style={{ padding: 16 }}>
            <div style={{ fontFamily: draft.heading.fallback, fontSize: 20, lineHeight: 1.25, fontWeight: 700, color: ok(c.text, '#1F2937'), marginBottom: 8 }}>{t(mk.sampleHeading)}</div>
            <p style={{ margin: '0 0 12px', fontFamily: body, fontSize: 13, lineHeight: 1.55, color: ok(c.text, '#1F2937') }}>
              <T msg={mk.previewPara} values={{ link: (chunks: ReactNode) => <span style={{ color: ok(c.link, '#0F766E'), textDecoration: 'underline' }}>{chunks}</span> }} />
            </p>
            <SampleButton draft={draft} />
          </div>
          <div style={{ padding: '12px 16px', background: '#F9FAFB', fontFamily: body, fontSize: 10.5, lineHeight: 1.5, color: '#4B5563', textAlign: 'center' }}>
            {t(mk.previewAddress)}
            <br />
            <span style={{ color: ok(c.link, '#0F766E'), textDecoration: 'underline' }}>{t(mk.previewFooter)}</span>
          </div>
        </div>
      </div>
    </section>
  )
}
