import { useMemo, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { FormsTabs, PublisherMark, throwIfForbidden, useInvalidateForms, usePublicPages, type PublicPages } from './common'
import { T, t, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { m as fb } from './builder.messages'
import { m } from './branding.messages'

type BrandingInput = Schemas['PublicBrandingInput'] & { domain: string }

/** The fonts; IBM Plex Sans is a name. */
const fonts: { value: BrandingInput['font']; label: MessageDescriptor | string }[] = [
  { value: 'plex', label: 'IBM Plex Sans' },
  { value: 'system', label: m.fontSystem },
  { value: 'custom', label: m.fontCustom },
]

function inputOf(s: PublicPages): BrandingInput {
  const b = s.branding
  return {
    publisherName: b.publisherName,
    accent: b.accent,
    background: b.background,
    font: b.font,
    customFontFamily: b.customFontFamily,
    customFontUrl: b.customFontUrl,
    poweredBy: b.poweredBy,
    privacyUrl: b.privacyUrl,
    domain: s.domain.domain,
  }
}

/** WCAG relative luminance of "#RRGGBB" (the preview's text on the accent). */
export function luminance(hex: string): number | undefined {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim())
  if (!m) return undefined
  const v = parseInt(m[1]!, 16)
  const ch = (c: number) => {
    const s = c / 255
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * ch((v >> 16) & 255) + 0.7152 * ch((v >> 8) & 255) + 0.0722 * ch(v & 255)
}

/** The text colour on accent buttons: the saved one, else white or ink by contrast. */
export function onAccentFor(accent: string, saved: Schemas['PublicBranding']): string {
  if (accent.toLowerCase() === saved.accent.toLowerCase()) return saved.onAccent
  const l = luminance(accent)
  if (l === undefined) return saved.onAccent
  return 1.05 / (l + 0.05) >= 4.5 ? '#FFFFFF' : '#0B1220'
}

const hexOK = (s: string) => /^#[0-9a-f]{6}$/i.test(s.trim())

/** Branding tab (top of board FormsStates): how every public page looks. */
export function BrandingPage() {
  const tr = useT()
  const settings = usePublicPages()
  throwIfForbidden(settings.error)
  if (settings.isError) {
    return (
      <Page title={tr(fb.pageTitle)} filters={<FormsTabs />}>
        <ErrorPanel title={tr(m.loadFailed)} actions={<Button onClick={() => settings.refetch()}>{tr(common.retry)}</Button>}>
          {errorMessage(settings.error)} {tr(m.keepLook)}
        </ErrorPanel>
      </Page>
    )
  }
  if (settings.isPending) {
    return (
      <Page title={tr(fb.pageTitle)} filters={<FormsTabs />} contentClassName="fs-main">
        <section className="card flex flex-wrap items-start gap-5 px-5 py-4.5" aria-busy="true">
          <span className="vh" role="status">
            {tr(m.loading)}
          </span>
          <div className="flex flex-[1_1_320px] flex-col gap-2.5">
            <Skeleton w={420} h={16} />
            <Grid min={200} gap={10}>
              {Array.from({ length: 6 }, (_, i) => (
                <div key={i} className="flex flex-col gap-1">
                  <Skeleton w={90} h={12} />
                  <Skeleton h={34} />
                </div>
              ))}
            </Grid>
          </div>
          <div className="flex flex-[0_1_280px] flex-col gap-2">
            <Skeleton w={60} h={12} />
            <Skeleton h={150} r={10} />
          </div>
        </section>
      </Page>
    )
  }
  return <BrandingEditor key={settings.data.updatedAt} settings={settings.data} />
}

function BrandingEditor({ settings }: { settings: PublicPages }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateForms()
  const reason = usePermissionReason('forms:edit')
  const readOnly = Boolean(reason)
  const initial = useMemo(() => inputOf(settings), [settings])
  const [draft, setDraft] = useState<BrandingInput>(initial)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [uploading, setUploading] = useState(false)
  const file = useRef<HTMLInputElement>(null)
  const saveBranding = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/branding')
  const saveDomain = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/domain')
  const checkDomain = $api.useMutation('post', '/api/v1/workspaces/{workspace}/public-pages/domain/check')
  const removeLogo = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/public-pages/logo')
  const dirty = JSON.stringify(draft) !== JSON.stringify(initial)
  const set = <K extends keyof BrandingInput>(k: K, v: BrandingInput[K]) => setDraft((d) => ({ ...d, [k]: v }))
  const b = settings.branding
  const path = { path: { workspace: slug } }

  const save = async () => {
    setErrors({})
    const next: Record<string, string> = {}
    if (!hexOK(draft.accent)) next.accent = t(m.hexColour, { example: '#4338CA' })
    if (!hexOK(draft.background)) next.background = t(m.hexColour, { example: '#F5F3FF' })
    if (!draft.publisherName.trim()) next.publisherName = t(m.namePublisher)
    if (Object.keys(next).length) {
      setErrors(next)
      throw new Error(Object.values(next)[0])
    }
    try {
      const { domain, ...brand } = draft
      const body = { ...brand, customFontFamily: brand.font === 'custom' ? brand.customFontFamily : undefined, customFontUrl: brand.font === 'custom' ? brand.customFontUrl : undefined, privacyUrl: brand.privacyUrl || undefined }
      const out = await saveBranding.mutateAsync({ params: path, body })
      let domainNote = ''
      if (domain.trim() !== initial.domain) {
        const d = await saveDomain.mutateAsync({ params: path, body: { domain: domain.trim() } })
        domainNote = d.domain.domain ? (d.domain.status === 'verified' ? t(m.domainServes, { domain: d.domain.domain }) : d.domain.detail) : t(m.domainStays)
      }
      toast({
        kind: 'success',
        title: t(m.saved),
        detail: [...out.branding.notes, t(m.usedNextVisit), domainNote].filter(Boolean).join(' '),
      })
      await invalidate()
    } catch (e) {
      const fe: Record<string, string> = {}
      if (e instanceof ApiError) for (const x of e.problem.errors ?? []) if (x.location && x.message) fe[x.location.replace(/^body\./, '')] = x.message
      setErrors(fe)
      toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) })
      throw e
    }
  }

  const upload = async (f: File | undefined) => {
    if (!f) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', f)
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/public-pages/logo`, { method: 'PUT', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
      const body = (await r.json().catch(() => ({}))) as { detail?: string; title?: string; errors?: { message?: string }[] }
      if (!r.ok) throw new Error(body.errors?.[0]?.message || body.detail || body.title || t(m.uploadFailedStatus, { status: r.status }))
      toast({ kind: 'success', title: t(m.logoUploaded), detail: t(m.logoUploadedBody) })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: t(m.logoUploadFailed), detail: e instanceof Error ? e.message : String(e) })
    } finally {
      setUploading(false)
      if (file.current) file.current.value = ''
    }
  }

  const accent = hexOK(draft.accent) ? draft.accent : b.accent
  const ground = hexOK(draft.background) ? draft.background : b.background
  // The preview shows the subscribe page's own copy, in the page's language.
  const heading = settings.pages.subscribe.heading || `The ${draft.publisherName || b.publisherName} newsletter`
  const dom = settings.domain
  const domainChanged = draft.domain.trim() !== initial.domain

  return (
    <Page
      title={tr(fb.pageTitle)}
      filters={<FormsTabs />}
      actions={
        dirty ? (
          <>
            <Button onClick={() => setDraft(initial)}>{tr(m.discard)}</Button>
            <Button
              variant="primary"
              onClick={() => void save().catch(() => undefined)}
              loading={saveBranding.isPending || saveDomain.isPending}
              busyLabel={tr(m.saving)}
              disabledReason={reason}
            >
              {tr(m.save)}
            </Button>
          </>
        ) : null
      }
      contentClassName="fs-main"
    >
      <section className="card flex flex-wrap items-start gap-5 px-5 py-4.5" aria-labelledby="branding-title">
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-2.5">
          <h2 className="h2" id="branding-title">
            {tr(m.title)}
          </h2>
          <Grid min={200} gap={10}>
            <label className="lbl">
              {tr(m.publisherName)}
              <input className="fb-in" value={draft.publisherName} maxLength={80} disabled={readOnly} aria-invalid={errors.publisherName ? true : undefined} onChange={(e) => set('publisherName', e.target.value)} />
              {errors.publisherName ? <span className="err">{errors.publisherName}</span> : null}
            </label>
            <div className="lbl">
              <span id="logo-label">{tr(m.logo)}</span>
              <input ref={file} type="file" accept="image/svg+xml,image/png,image/jpeg,image/gif,image/webp" className="vh" tabIndex={-1} aria-hidden="true" onChange={(e) => void upload(e.target.files?.[0])} />
              {b.logo ? (
                <Menu
                  align="start"
                  label={tr(m.logo)}
                  trigger={
                    <button type="button" className="btn min-h-[34px]" aria-labelledby="logo-label logo-btn" id="logo-btn" disabled={readOnly || uploading}>
                      {uploading ? tr(m.uploading) : tr(m.uploadNamed, { name: b.logo.name })}
                    </button>
                  }
                >
                  <MenuItem onSelect={() => file.current?.click()}>{tr(m.uploadNew)}</MenuItem>
                  <MenuItem
                    onSelect={async () => {
                      try {
                        await removeLogo.mutateAsync({ params: path })
                        toast({ kind: 'success', title: t(m.logoRemoved), detail: t(m.logoRemovedBody, { initial: b.initial }) })
                        await invalidate()
                      } catch (e) {
                        toast({ kind: 'error', title: t(m.logoRemoveFailed), detail: errorMessage(e) })
                      }
                    }}
                  >
                    {tr(m.removeLogo)}
                  </MenuItem>
                </Menu>
              ) : (
                <button type="button" className="btn min-h-[34px]" aria-labelledby="logo-label logo-btn" id="logo-btn" disabled={readOnly || uploading} onClick={() => file.current?.click()}>
                  {uploading ? tr(m.uploading) : tr(m.uploadSvg)}
                </button>
              )}
            </div>
            <ColorField label={tr(m.accent)} value={draft.accent} readOnly={readOnly} error={errors.accent} onChange={(v) => set('accent', v)} />
            <ColorField label={tr(m.background)} value={draft.background} readOnly={readOnly} error={errors.background} onChange={(v) => set('background', v)} />
            <label className="lbl">
              {tr(m.font)}
              <select className="fb-in" value={draft.font} disabled={readOnly} onChange={(e) => set('font', e.target.value as BrandingInput['font'])}>
                {fonts.map((f) => (
                  <option key={f.value} value={f.value}>
                    {typeof f.label === 'string' ? f.label : tr(f.label)}
                  </option>
                ))}
              </select>
            </label>
            <label className="lbl">
              {tr(m.customDomain)}
              <input
                className="fb-in mono"
                value={draft.domain}
                placeholder={'news.acme.io'}
                disabled={readOnly}
                aria-invalid={errors.domain ? true : undefined}
                onChange={(e) => set('domain', e.target.value)}
              />
              {errors.domain ? <span className="err">{errors.domain}</span> : null}
            </label>
            {draft.font === 'custom' ? (
              <>
                <label className="lbl">
                  {tr(m.fontFamily)}
                  <input className="fb-in" value={draft.customFontFamily ?? ''} placeholder={'Inter'} disabled={readOnly} aria-invalid={errors.customFontFamily ? true : undefined} onChange={(e) => set('customFontFamily', e.target.value)} />
                  {errors.customFontFamily ? <span className="err">{errors.customFontFamily}</span> : null}
                </label>
                <label className="lbl">
                  {tr(m.fontFile)}
                  <input className="fb-in mono" value={draft.customFontUrl ?? ''} placeholder={'https://acme.io/fonts/inter.woff2'} disabled={readOnly} aria-invalid={errors.customFontUrl ? true : undefined} onChange={(e) => set('customFontUrl', e.target.value)} />
                  {errors.customFontUrl ? <span className="err">{errors.customFontUrl}</span> : null}
                </label>
              </>
            ) : null}
            <label className="lbl">
              {tr(m.privacy)}
              <input className="fb-in mono" value={draft.privacyUrl ?? ''} placeholder={'https://acme.io/privacy'} disabled={readOnly} aria-invalid={errors.privacyUrl ? true : undefined} onChange={(e) => set('privacyUrl', e.target.value)} />
              {errors.privacyUrl ? <span className="err">{errors.privacyUrl}</span> : null}
            </label>
          </Grid>
          {dom.domain && !domainChanged && dom.status !== 'verified' ? (
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone={dom.status === 'failed' ? 'bad' : 'warn'} dot>
                {dom.status === 'failed' ? tr(m.dnsFailed) : tr(m.dnsWaiting)}
              </Badge>
              <span className="k">
                {dom.detail || tr(m.pointCname, { domain: dom.domain, target: dom.target })}{' '}
                <T msg={m.orTxt} values={{ name: dom.txtName, value: dom.txtValue, mono: (c) => <span className="mono">{c}</span> }} />
              </span>
              <Button
                size="sm"
                loading={checkDomain.isPending}
                disabledReason={reason}
                onClick={async () => {
                  try {
                    const out = await checkDomain.mutateAsync({ params: path })
                    toast({
                      kind: out.domain.status === 'verified' ? 'success' : 'info',
                      title: out.domain.status === 'verified' ? t(m.verified, { domain: out.domain.domain }) : t(m.notYet),
                      detail: out.domain.detail,
                    })
                    await invalidate()
                  } catch (e) {
                    toast({ kind: 'error', title: t(m.dnsCheckFailed), detail: errorMessage(e) })
                  }
                }}
              >
                {tr(m.checkDns)}
              </Button>
            </div>
          ) : null}
          <label className="fb-check">
            <input type="checkbox" checked={draft.poweredBy} disabled={readOnly} onChange={(e) => set('poweredBy', e.target.checked)} />
            {tr(m.poweredBy)}
          </label>
        </div>
        <div className="flex flex-[0_1_280px] flex-col gap-2 max-md:flex-[1_1_100%]">
          <span className="k">{tr(m.preview)}</span>
          <div className="fs-preview" style={{ background: ground, fontFamily: draft.font === 'system' ? 'system-ui, sans-serif' : undefined }}>
            <div className="flex items-center gap-2">
              <PublisherMark branding={b} className="fs-mark" logoClassName="logo" />
              <span className="font-bold">{draft.publisherName || b.publisherName}</span>
            </div>
            <span className="text-[15px] font-bold">{heading}</span>
            <span className="fs-input" />
            {/* The subscribe page's own button word, as the page shows it. */}
            <span className="fs-button" style={{ background: accent, color: onAccentFor(accent, b) }}>
              {'Subscribe'}
            </span>
          </div>
        </div>
      </section>
      <UnsavedChangesGuard dirty={dirty && !readOnly} description={tr(m.unsaved)} saveLabel={tr(m.saveLeave)} onSave={save} />
    </Page>
  )
}

/** A colour: a swatch that opens the colour picker, and its hex value. */
function ColorField({ label, value, readOnly, error, onChange }: { label: string; value: string; readOnly: boolean; error?: string; onChange: (v: string) => void }) {
  const tr = useT()
  const ok = hexOK(value)
  return (
    <label className="lbl">
      {label}
      {/* On phones the board wraps the hex value under the swatch. */}
      <div className="flex items-center gap-2 max-md:flex-wrap">
        <span className="fs-sw" style={{ background: ok ? value : '#ffffff' }}>
          <input
            type="color"
            aria-label={tr(m.picker, { label })}
            value={ok ? value.toLowerCase() : '#ffffff'}
            disabled={readOnly}
            onChange={(e) => onChange(e.target.value.toUpperCase())}
          />
        </span>
        <input
          className="fb-in mono max-md:w-0 max-md:min-w-full"
          aria-label={label}
          value={value}
          maxLength={7}
          disabled={readOnly}
          aria-invalid={error ? true : undefined}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
      {error ? <span className="err">{error}</span> : null}
    </label>
  )
}
