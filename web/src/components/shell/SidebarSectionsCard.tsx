import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { CardSection } from '@/components/ui/Card'
import { DisabledReason } from '@/components/ui/DisabledReason'
import { toast } from '@/components/ui/Toast'
import { useT, type MessageDescriptor } from '@/i18n'
import { cn } from '@/lib/cn'
import { useWorkspace, workspaceMeKey, type WorkspaceMe } from '@/lib/session'
import { m } from './messages'
import { NAV, hideableSections, type HideableSection } from './nav'

const descriptions: Record<HideableSection, MessageDescriptor> = {
  transactional: m.sectionsTransactional,
  grow: m.sectionsGrow,
  earn: m.sectionsEarn,
}

type Settings = Pick<Schemas['WorkspaceSettings'], 'hiddenSections' | 'canManage' | 'reason'>

/**
 * Settings › General › Sidebar sections (WorkspaceSettings): switches for the
 * sections a workspace may not use. A switch applies at once; hiding a
 * section only takes it out of the sidebar, the rail, the tab bar and the
 * More sheet.
 */
export function SidebarSectionsCard({ s }: { s: Settings }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const t = useT()
  // What the switches show: the saved state, with changes in flight on top.
  const [pending, setPending] = useState<Partial<Record<HideableSection, boolean>>>({})
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/settings/sidebar/{section}')
  const reason = s.canManage ? undefined : s.reason
  const shown = (key: HideableSection) => pending[key] ?? !s.hiddenSections.includes(key)

  const change = (key: HideableSection, on: boolean) => {
    const section = t(NAV.find((i) => i.key === key)!.label)
    setPending((p) => ({ ...p, [key]: on }))
    // The shell follows at once; the settings page catches up from the answer.
    const meKey = workspaceMeKey(slug)
    const before = qc.getQueryData<WorkspaceMe>(meKey)
    qc.setQueryData<WorkspaceMe>(meKey, (me) => (me ? { ...me, hiddenSections: withSection(me.hiddenSections, key, on) } : me))
    save.mutate(
      { params: { path: { workspace: slug, section: key } }, body: { shown: on } },
      {
        onSuccess: (out) => {
          qc.setQueryData<WorkspaceMe>(meKey, (me) => (me ? { ...me, hiddenSections: out.hiddenSections } : me))
          qc.setQueryData($api.queryOptions('get', '/api/v1/workspaces/{workspace}/settings', { params: { path: { workspace: slug } } }).queryKey, (cur) =>
            cur ? { ...cur, hiddenSections: out.hiddenSections } : cur,
          )
          toast(on ? { title: t(m.sectionsShown, { section }) } : { title: t(m.sectionsHidden, { section }), detail: t(m.sectionsHiddenDetail) })
        },
        onError: (e) => {
          if (before) qc.setQueryData(meKey, before)
          toast({ kind: 'error', title: t(m.sectionsFailed), detail: errorMessage(e) })
        },
        onSettled: () => {
          setPending(({ [key]: _, ...rest }) => rest)
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/settings'] })
          void qc.invalidateQueries({ queryKey: meKey })
        },
      },
    )
  }

  return (
    <section className="card" aria-labelledby="sidebar-h">
      <CardSection title={<span id="sidebar-h">{t(m.sectionsTitle)}</span>} caption={t(m.sectionsCaption)} />
      <div className="pad col" style={{ gap: 0 }}>
        {hideableSections.map((key, i) => {
          const name = t(NAV.find((n) => n.key === key)!.label)
          const id = `sidebar-${key}`
          return (
            <div key={key} className={cn('row flex-nowrap justify-between gap-3 py-2.5', i > 0 && 'border-t border-line-2')}>
              <span className="col min-w-0 gap-px">
                <span className="font-semibold">{name}</span>
                <span className="k whitespace-normal" id={`${id}-d`}>
                  {t(descriptions[key])}
                </span>
              </span>
              <DisabledReason reason={reason}>
                <input
                  type="checkbox"
                  role="switch"
                  className="tg"
                  checked={shown(key)}
                  aria-label={t(m.sectionsSwitch, { section: name })}
                  aria-describedby={`${id}-d`}
                  onChange={(e) => change(key, e.target.checked)}
                />
              </DisabledReason>
            </div>
          )
        })}
      </div>
    </section>
  )
}

/** The hidden sections after showing (`on`) or hiding one, in sidebar order. */
export function withSection(hidden: readonly string[], key: HideableSection, on: boolean): WorkspaceMe['hiddenSections'] {
  return hideableSections.filter((k) => (k === key ? !on : hidden.includes(k)))
}
