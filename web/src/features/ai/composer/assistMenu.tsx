import type { Component } from 'grapesjs'
import { useState } from 'react'
import { Icon } from '@/components/icons/Icon'
import { ButtonLink } from '@/components/ui/Button'
import { Popover } from '@/components/ui/Menu'
import { formatList, useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useComposer } from '@/features/composer/context'
import { useAccessibility } from '@/features/accessibility/composer/state'
import { useCampaignLanguages } from '@/features/languages/composer'
import { versionLanguage } from '@/features/languages/composerParts'
import { useWorkspace } from '@/lib/session'
import { m, ms } from '../messages'
import { NotConfigured, Sparkle, usageLine } from '../suggest'
import { useIdeasInput } from './ideas'
import { useRewriteSelected } from './rewrite'
import { useComposerAssist } from './state'

/** Images on the canvas without alt text (and not marked decorative). */
function imagesWithoutAlt(c: ReturnType<typeof useComposer>): Component[] {
  const body = c.editor?.getWrapper()?.findType('mj-image') ?? []
  return body.filter((x) => {
    const attrs = (x.getAttributes() ?? {}) as Record<string, string | undefined>
    return attrs.alt === undefined && Boolean(attrs.src)
  })
}

/**
 * Assist in the composer header (AIComposer · where Assist appears): subject
 * ideas, rewrite the selected text, translate this variant, alt text, the
 * accessibility issues it can help with. Without a model it invites to
 * connect one (AIStates · not configured), never an upgrade prompt.
 */
export function AssistMenu() {
  const ca = useComposerAssist()
  const c = useComposer()
  const a11y = useAccessibility()
  const { me } = useWorkspace()
  const t = useT()
  const [open, setOpen] = useState(false)
  const ideasInput = useIdeasInput()
  const rewriteSelected = useRewriteSelected()
  const langs = useCampaignLanguages(c.campaign.id)
  if (!ca || !ca.assist.can || c.kind === 'transactional' || !c.canEdit) return null
  const a = ca.assist
  const s = a.status
  const trigger = (
    <button type="button" className="btn ai-assist" aria-haspopup="menu" aria-expanded={open} title={t(ms.assist)}>
      <Sparkle />
      {t(ms.assist)}
      <Icon name="chevronDown" size={14} />
    </button>
  )
  const workspace = me.workspace.name
  if (!a.configured) {
    return (
      <Popover open={open} onOpenChange={setOpen} trigger={trigger} label={t(ms.assistMenu)} className="ai-pop">
        <h3>
          <Sparkle />
          {s?.canManage ? t(ms.assistCanConnect) : t(ms.assistNotSetUp)}
        </h3>
        <NotConfigured assist={a} workspace={workspace} />
      </Popover>
    )
  }
  if (s && (s.state === 'key_rejected' || s.state === 'cap_reached')) {
    const helpers = s.helpers.length ? formatList(s.helpers, 'disjunction') : ''
    return (
      <Popover open={open} onOpenChange={setOpen} trigger={trigger} label={t(ms.assistMenu)} className="ai-pop">
        <h3>
          <Sparkle />
          {s.state === 'key_rejected' ? t(ms.assistKeyRejected) : t(ms.assistCapReached)}
        </h3>
        <p className="m-0 text-[13px] leading-normal">
          {s.state === 'key_rejected'
            ? helpers
              ? t(ms.keyRejectedMenu, { provider: s.providerName ?? '', helpers })
              : t(ms.keyRejectedMenuAlone, { provider: s.providerName ?? '' })
            : helpers
              ? t(ms.capMenu, { date: dayMonthDate(s.usage.resetsAt), helpers })
              : t(ms.capMenuAlone, { date: dayMonthDate(s.usage.resetsAt) })}
        </p>
        {s.canManage ? (
          <div>
            <ButtonLink size="sm" variant="primary" to="/w/$workspace/settings/integrations/ai" params={{ workspace: c.slug }}>
              {s.state === 'key_rejected' ? t(m.replaceKey) : t(m.raiseCap)}
            </ButtonLink>
          </div>
        ) : null}
        <span className="k">{usageLine(t, s)}</span>
      </Popover>
    )
  }
  const close = () => setOpen(false)
  // Translate this variant: the version open, else the languages missing or outdated.
  const data = langs.data
  const open_ = versionLanguage(c.campaign)
  const offered = (data?.versions ?? []).filter((v) => v.offered && v.status !== 'source')
  const missing = offered.filter((v) => v.status === 'missing')
  const outdated = offered.filter((v) => v.status === 'outdated')
  const target = open_ ?? missing[0]?.language ?? outdated[0]?.language
  const images = imagesWithoutAlt(c)
  const fixable = (a11y?.report?.issues ?? []).filter((i) => i.kind === 'alt_text' || i.kind === 'link_text').length
  const item = (label: string, sub: string, onClick: (() => void) | undefined, off?: string) => (
    <button
      type="button"
      role="menuitem"
      className="ai-mi"
      disabled={!onClick || Boolean(off)}
      onClick={() => {
        close()
        onClick?.()
      }}
    >
      <span>{label}</span>
      <span className="k">{off ?? sub}</span>
    </button>
  )
  const offText = (f: Parameters<typeof a.featureOn>[0]) => (a.featureOn(f) ? undefined : f === 'alt_text' && s && !s.imageInput ? t(ms.noImageModel) : t(ms.featureOff))
  return (
    <Popover open={open} onOpenChange={setOpen} trigger={trigger} label={t(ms.assistMenu)} className="ai-menu">
      <div role="menu" aria-label={t(ms.assistMenu)} className="flex flex-col">
        {item(t(ms.assistIdeas), t(ms.assistIdeasSub), () => ca.openIdeas(ideasInput()), offText('subject_ideas'))}
        {item(t(ms.assistRewrite), rewriteSelected ? t(ms.assistRewriteSubOn) : t(ms.assistRewriteSub), rewriteSelected, offText('rewrite'))}
        {data && offered.length
          ? item(
              t(ms.assistTranslate),
              open_
                ? t(ms.translateWith, { language: offered.find((v) => v.language === open_)?.name ?? open_ })
                : missing.length
                  ? t(ms.assistTranslateSub, { count: missing.length, languages: formatList(missing.map((v) => v.name)) })
                  : outdated.length
                    ? t(ms.assistTranslateOutdated, { count: outdated.length, languages: formatList(outdated.map((v) => v.name)) })
                    : t(ms.assistTranslateNone),
              target ? () => ca.translate(target) : undefined,
              offText('translate'),
            )
          : null}
        {item(
          t(ms.assistAlt),
          t(ms.assistAltSub, { count: images.length }),
          images.length
            ? () => {
                const first = images[0]!
                c.editor?.select(first)
                c.setRightTab('settings')
                first.getEl()?.scrollIntoView({ block: 'center', behavior: 'smooth' })
              }
            : undefined,
          offText('alt_text'),
        )}
        {item(t(ms.assistA11y), t(ms.assistA11ySub, { count: fixable }), fixable ? () => c.setRightTab('accessibility') : undefined, offText('accessibility_fix'))}
        <div className="ai-mf">{usageLine(t, s)}</div>
      </div>
    </Popover>
  )
}
