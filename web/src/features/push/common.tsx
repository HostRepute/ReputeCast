import { useQueryClient } from '@tanstack/react-query'
import { ApiError, type Schemas } from '@/api/client'
import { NavTabs } from '@/components/ui/Tabs'
import { useT, type MessageDescriptor } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import './push.css'

export type PushSetup = Schemas['PushSetup']
export type PushSite = Schemas['PushSite']
export type PushPromptCard = Schemas['PushPromptCard']
export type PushPromptConfig = Schemas['PushPromptConfig']
export type PromptStyle = PushPromptConfig['style']

/** Push's API paths (internal/push). */
export const PUSH = '/api/v1/workspaces/{workspace}/push' as const

/**
 * A role without access to this screen gets the shell's 403 state (the
 * route's error component); other load errors stay inline with a retry.
 */
export function throwIfForbidden(error: unknown) {
  if (error instanceof ApiError && error.status === 403) throw error
}

/** Setup · Events, inline after the channel switch. */
export function PushTabs() {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  return (
    <NavTabs
      label={t(m.sectionsLabel)}
      className="ml-3 max-md:ml-0"
      tabs={[
        { label: t(m.tabSetup), to: '/w/$workspace/providers/push', params, exact: true },
        { label: t(m.tabEvents), to: '/w/$workspace/providers/push/events', params },
      ]}
    />
  )
}

const browserNames: Record<string, MessageDescriptor> = {
  chrome: m.browserChrome,
  safari: m.browserSafari,
  chrome_android: m.browserChromeAndroid,
  edge: m.browserEdge,
  firefox: m.browserFirefox,
  safari_ios: m.browserSafariIos,
  other: m.browserOther,
}

/** "Chrome desktop", "Safari on iPhone and iPad (Home Screen)". */
export function browserName(key: string, short = false): MessageDescriptor {
  if (short && key === 'safari_ios') return m.browserSafariIosShort
  return browserNames[key] ?? m.browserOther
}

export const styleName: Record<PromptStyle, MessageDescriptor> = { bar: m.styleBar, dialog: m.styleDialog, bell: m.styleBell }

/** Refreshes every push query after a change. */
export function useInvalidatePush() {
  const qc = useQueryClient()
  return () =>
    qc.invalidateQueries({
      predicate: (q) => {
        const k = q.queryKey[1]
        return typeof k === 'string' && (k.includes('/push') || k.includes('/channels'))
      },
    })
}

/** Copies text; false when the browser refused. */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

/** "help.acme.io" for "https://help.acme.io/". */
export function bareHost(url: string): string {
  return url
    .trim()
    .replace(/^https?:\/\//i, '')
    .replace(/\/.*$/, '')
    .toLowerCase()
}

/** "3 KB" for a byte count. */
export function kb(bytes: number): string {
  return `${Math.max(1, Math.round(bytes / 1024))} KB`
}
