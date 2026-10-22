import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { dayMonthDate } from '@/lib/format'
import { useComposer } from './context'
import { Thumb } from './Thumb'
import { GalleryCaption, GalleryResult, galleryTileClass, useGalleryResults } from '@/features/accessibility/composer/Gallery'
import { isDigestTemplate } from '@/features/digests/composer/DigestMode'
import { AgencyComposerGallery } from '@/features/agency/ComposerLibrary'
import { useT } from '@/i18n'
import { m } from './messages'

/**
 * The empty canvas (ComposerEmpty): start from a starter template, one of
 * the workspace's templates, a copy of the last sent campaign, or an import.
 */
export function EmptyGallery() {
  const c = useComposer()
  const t = useT()
  const g = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/templates',
    // A recurring digest's template also gets the digest starters (Blog digest).
    { params: { path: { workspace: c.slug }, query: isDigestTemplate(c.campaign) ? { mode: 'digest' } : {} } },
    { staleTime: 60_000 },
  )
  const [busy, setBusy] = useState<string | null>(null)
  // Each template's accessibility result (the campaign composer's gallery).
  const a11y = useGalleryResults(c.slug, c.kind !== 'transactional')

  const applyTemplate = async (id: string, name: string) => {
    if (!c.canEdit) return
    setBusy(id)
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/templates/{templateId}', { params: { path: { workspace: c.slug, templateId: id } } })
      if (data) c.loadContent(data.mjml, { source: 'template', label: id === 'starter:blank' ? t(m.noteBlank) : t(m.noteFromTemplate, { name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.templateFailed), detail: errorMessage(e) })
    } finally {
      setBusy(null)
    }
  }
  const copyCampaign = async (id: string, name: string) => {
    if (!c.canEdit) return
    setBusy(id)
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/campaigns/{campaignId}', { params: { path: { workspace: c.slug, campaignId: id } } })
      if (data) c.loadContent(data.mjml, { source: 'template', label: t(m.noteCopied, { name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.copyFailed), detail: errorMessage(e) })
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="flex flex-1 items-start justify-center" style={{ padding: '32px 24px' }}>
      <div className="box-border flex w-full flex-col rounded-[12px] border border-line bg-surface" style={{ maxWidth: 860, padding: 24, gap: 18 }}>
        <div className="flex flex-col gap-1">
          <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
            <h2 className="m-0 text-[20px] font-bold" style={{ letterSpacing: '-.01em' }}>
              {t(m.galleryTitle)}
            </h2>
            <GalleryCaption data={a11y.data} />
          </div>
          <span className="k">{t(m.galleryLead)}</span>
        </div>
        {g.isPending ? (
          <div className="tpl-grid grid gap-3" aria-busy="true">
            {Array.from({ length: 6 }, (_, i) => (
              <div key={i} className="tpl" aria-hidden="true">
                <div className="thumb" />
                <span className="sk" style={{ height: 14, width: '60%' }} />
              </div>
            ))}
          </div>
        ) : g.isError ? (
          <p className="k">{t(m.templatesFailed, { error: errorMessage(g.error) })}</p>
        ) : (
          <>
            <div className="tpl-grid grid gap-3">
              {g.data.starters.map((tpl) => (
                <button key={tpl.id} type="button" className={galleryTileClass(a11y.byId.get(tpl.id))} disabled={!c.canEdit} aria-busy={busy === tpl.id || undefined} onClick={() => void applyTemplate(tpl.id, tpl.name)}>
                  <Thumb items={tpl.thumb} blank={tpl.id === 'starter:blank'} />
                  <span className="font-semibold">{tpl.name}</span>
                  <span className="k">{tpl.description}</span>
                  <GalleryResult result={a11y.byId.get(tpl.id)} />
                </button>
              ))}
            </div>
            <div className="flex flex-col gap-2">
              <span className="font-semibold">{t(m.yours)}</span>
              <div className="tpl-grid grid gap-3">
                {g.data.templates.map((tpl) => (
                  <button key={tpl.id} type="button" className={galleryTileClass(a11y.byId.get(tpl.id))} disabled={!c.canEdit} aria-busy={busy === tpl.id || undefined} onClick={() => void applyTemplate(tpl.id, tpl.name)}>
                    <Thumb items={tpl.thumb} />
                    <span className="font-semibold">{tpl.name}</span>
                    <span className="k">{tpl.updatedAt ? t(m.savedTemplateEdited, { date: dayMonthDate(tpl.updatedAt) }) : t(m.savedTemplate)}</span>
                    <GalleryResult result={a11y.byId.get(tpl.id)} />
                  </button>
                ))}
                {g.data.lastCampaign ? (
                  <button
                    type="button"
                    className={galleryTileClass(a11y.last)}
                    disabled={!c.canEdit}
                    aria-busy={busy === g.data.lastCampaign.id || undefined}
                    onClick={() => g.data.lastCampaign && void copyCampaign(g.data.lastCampaign.id, g.data.lastCampaign.name)}
                  >
                    <Thumb items={g.data.lastCampaign.thumb} />
                    <span className="font-semibold">{t(m.copyOf, { name: g.data.lastCampaign.name })}</span>
                    <span className="k">{t(m.lastSent, { date: dayMonthDate(g.data.lastCampaign.sentAt) })}</span>
                    <GalleryResult result={a11y.last} />
                  </button>
                ) : null}
                <button type="button" className="tpl" style={{ borderStyle: 'dashed', alignItems: 'center', justifyContent: 'center' }} disabled={!c.canEdit} onClick={() => c.openDialog('import')}>
                  <span className="font-semibold">{t(m.importTile)}</span>
                  <span className="k">{t(m.importTileLine)}</span>
                </button>
              </div>
            </div>
            {/* The agency's library, for its staff in a client workspace (agency accounts). */}
            {c.kind !== 'transactional' ? <AgencyComposerGallery canEdit={c.canEdit} onLoad={(mjml, label) => c.loadContent(mjml, { source: 'template', label })} /> : null}
          </>
        )}
      </div>
    </div>
  )
}
