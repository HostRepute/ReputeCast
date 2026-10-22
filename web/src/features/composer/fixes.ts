import type { Component } from 'grapesjs'
import { http } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import type { ComposerState, Issue } from './context'
import { footerBlock } from './editor/blocks'
import { atPath, mjChildren, tagOf } from './editor/labels'
import { t } from '@/i18n'
import { m } from './messages'

export type Fix = Issue['fixes'][number]

/** mj-body of the canvas. */
export function bodyOf(c: ComposerState): Component | undefined {
  return c.editor?.getWrapper()?.findType('mj-body')[0]
}

/** The component an issue points at (element-index path from mj-body). */
export function componentAt(c: ComposerState, path: number[] | undefined): Component | undefined {
  if (!path?.length) return undefined
  return atPath(bodyOf(c), path)
}

/** Selects the component an issue is about and scrolls it into view. */
export function reveal(c: ComposerState, path: number[] | undefined, tab?: 'style' | 'settings') {
  const target = componentAt(c, path)
  if (!target || !c.editor) return
  c.editor.select(target)
  target.getEl()?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  if (tab) c.setRightTab(tab)
}

const LINK_TAGS = new Set(['mj-button', 'mj-image', 'mj-navbar-link', 'mj-social-element'])

function escapeRe(s: string) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** Points every link to `from` inside the component at the new address. */
export function replaceLink(target: Component, from: string, to: string): boolean {
  const tag = tagOf(target)
  if (LINK_TAGS.has(tag)) {
    target.addAttributes({ href: to })
    return true
  }
  // Links inside text: rewrite the href of matching <a> elements.
  let changed = false
  const walk = (x: Component) => {
    const attrs = (x.getAttributes() ?? {}) as Record<string, string>
    if (String(x.get('tagName') ?? '').toLowerCase() === 'a' && attrs.href === from) {
      x.addAttributes({ href: to })
      changed = true
    }
    x.components().forEach(walk)
  }
  walk(target)
  if (!changed) {
    // Text kept as a content string (not parsed into components).
    const content = String(target.get('content') ?? '')
    if (content.includes(from)) {
      target.set('content', content.replace(new RegExp(`href=(["'])${escapeRe(from)}\\1`, 'g'), `href="${to}"`))
      changed = true
    }
  }
  return changed
}

/** Appends the ReputeCast footer block (address + unsubscribe) to the body. */
export function insertFooter(c: ComposerState) {
  const body = bodyOf(c)
  if (!body || !c.editor) return
  const added = body.append(footerBlock(c.context.postalAddress))
  const first = (Array.isArray(added) ? added : [added]).find((x) => tagOf(x as Component))
  if (first) {
    c.editor.select(first as Component)
    ;(first as Component).getEl()?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  }
}

export async function applyFix(c: ComposerState, issue: Issue, fix: Fix, value?: string): Promise<void> {
  switch (fix.kind) {
    case 'replace_link': {
      const target = componentAt(c, fix.path ?? issue.path)
      if (!target || !fix.value || !issue.url) return
      if (replaceLink(target, issue.url, fix.value)) toast({ kind: 'success', title: t(m.linkReplaced, { url: fix.value.replace(/^https?:\/\//, '') }) })
      return
    }
    case 'edit_link':
      reveal(c, fix.path ?? issue.path, 'settings')
      window.setTimeout(() => document.getElementById('insp-href')?.focus(), 80)
      return
    case 'insert_footer':
      insertFooter(c)
      return
    case 'set_alt': {
      const target = componentAt(c, fix.path ?? issue.path)
      const alt = (value ?? '').trim()
      if (!target || !alt) return
      target.addAttributes({ alt })
      return
    }
    case 'remove_section': {
      const body = bodyOf(c)
      const index = (fix.path ?? issue.path)?.[0]
      const sec = body && index !== undefined ? mjChildren(body)[index] : undefined
      if (!sec) return
      sec.remove()
      toast({ kind: 'info', title: t(m.sectionRemoved), detail: t(m.sectionRemovedDetail) })
      return
    }
    case 'use_subject':
      if (fix.value) c.setSubject(fix.value)
      return
    case 'open':
      // The page beside the composer (a Sponsor block's booking): the draft stays open.
      if (fix.value?.startsWith('/')) window.open(fix.value, '_blank', 'noopener')
      return
    case 'minify':
      try {
        await http.PATCH('/api/v1/workspaces/{workspace}/campaigns/{campaignId}', {
          params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
          body: { settings: { minify: true } },
        })
        c.onSettingsChanged()
        toast({ kind: 'success', title: t(m.minified), detail: t(m.minifiedDetail) })
      } catch (e) {
        toast({ kind: 'error', title: t(m.minifyFailed), detail: e instanceof Error ? e.message : String(e) })
      }
      return
  }
}
