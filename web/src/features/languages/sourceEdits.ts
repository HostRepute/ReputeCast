import { useCallback } from 'react'
import { http, errorMessage, type Schemas } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { mz } from './messages'

type Langs = Schemas['EmailLanguages']

/** The links of an email's source: href and src attributes, in order. */
export function linksOf(src: string): string[] {
  const out: string[] = []
  for (const m of src.matchAll(/\b(?:href|src)="([^"]+)"/g)) out.push(m[1]!)
  return out
}

/**
 * The one link (or image) a source edit changed: the same list of links
 * with exactly one of them different.
 */
export function changedLink(before: string, after: string): { from: string; to: string; image: boolean } | undefined {
  const a = linksOf(before)
  const b = linksOf(after)
  if (a.length !== b.length) return undefined
  let found: { from: string; to: string; image: boolean } | undefined
  for (let i = 0; i < a.length; i++) {
    if (a[i] === b[i]) continue
    if (found) return undefined
    found = { from: a[i]!, to: b[i]!, image: /\.(png|jpe?g|gif|webp|svg)(\?|$)/i.test(b[i]!) }
  }
  return found
}

/**
 * After the source of an email with language versions is saved, the
 * composer offers to keep them in step (LanguageStates): "Marked outdated in
 * Spanish and German · Keep them up to date" for an edit that needs no
 * translating, and "Link changed in English · Apply to all languages".
 */
export function useSourceEdits(slug: string, campaignId: string, onChanged: (data: Langs) => void) {
  const t = useT()
  return useCallback(
    async (before: string, after: string, version: number) => {
      let data: Langs | undefined
      try {
        data = (await http.GET('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages', { params: { path: { workspace: slug, campaignId } } })).data
      } catch {
        return
      }
      if (!data || data.versions.length < 2) return
      onChanged(data)
      const names = (codes: string[]) => formatList(codes.map((c) => data!.versions.find((v) => v.language === c)?.name ?? c))
      const source = data.versions.find((v) => v.language === data!.source)?.name ?? data.source
      const sc = data.sourceChange
      const translated = data.versions.filter((v) => v.status === 'up_to_date' || v.status === 'outdated').map((v) => v.language)
      const link = translated.length ? changedLink(before, after) : undefined
      if (link) {
        toast({
          kind: 'info',
          title: link.image ? t(mz.imageToast, { source }) : t(mz.linkToast, { source, url: link.to.replace(/^https?:\/\//, '') }),
          action: {
            label: t(mz.applyAll),
            onClick: () =>
              void http
                .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/apply', {
                  params: { path: { workspace: slug, campaignId } },
                  body: { kind: link.image ? 'image' : 'link', from: link.from, to: link.to },
                })
                .then(({ data: r }) => r && toast({ title: t(mz.appliedToast, { languages: names(r.languages) }) }))
                .catch((e: unknown) => toast({ kind: 'error', title: t(mz.markFailed, { language: source }), detail: errorMessage(e) })),
          },
        })
        return
      }
      if (sc && sc.version === version && sc.outdated.length && data.canEdit) {
        toast({
          kind: 'info',
          title: t(mz.outdatedToast, { languages: names(sc.outdated) }),
          action: {
            label: t(mz.keepUpToDate),
            onClick: () =>
              void http
                .POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/keep-up-to-date', {
                  params: { path: { workspace: slug, campaignId } },
                  body: { blocks: sc.blocks },
                })
                .then(({ data: out }) => {
                  if (!out) return
                  onChanged(out)
                  toast({ title: t(mz.keptToast, { languages: names(sc.outdated) }) })
                })
                .catch((e: unknown) => toast({ kind: 'error', title: t(mz.markFailed, { language: source }), detail: errorMessage(e) })),
          },
        })
      }
    },
    [slug, campaignId, onChanged, t],
  )
}
