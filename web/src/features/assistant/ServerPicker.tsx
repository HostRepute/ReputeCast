import { DropdownMenu } from 'radix-ui'
import { useNavigate } from '@tanstack/react-router'
import { useT } from '@/i18n'
import type { MCPServer } from '@/features/mcpservers/api'
import { AsstIcon } from './icons'
import { mx } from './mcp.messages'
import { closeAssistant } from './store'

/** Whether the person still has to sign in before a server's tools work for them. */
export function needsSignIn(s: Pick<MCPServer, 'auth' | 'connected'>): boolean {
  return s.auth === 'oauth' && !s.connected
}

/**
 * The thread's MCP servers (set-ai-thread-mcp-servers): the servers the
 * person may use, each with whether it needs their sign-in, picked per
 * thread. Like the model picker; the boards have no picker of their own.
 */
export function ServerPicker({
  servers,
  value,
  onToggle,
  onSignIn,
  canManage,
  slug,
}: {
  servers: MCPServer[]
  value: string[]
  onToggle: (id: string, on: boolean) => void
  onSignIn: (s: MCPServer) => void
  canManage: boolean
  slug: string
}) {
  const t = useT()
  const navigate = useNavigate()
  const picked = servers.filter((s) => value.includes(s.id))
  const warn = picked.some(needsSignIn)
  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>
        <button type="button" className="asst-picker">
          {warn ? <AsstIcon name="warn" size={14} className="asst-warn-ic" /> : null}
          {t(mx.picker, { count: picked.length })}
          <AsstIcon name="chevron" />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="asst-menu" align="start" sideOffset={6} collisionPadding={12} aria-label={t(mx.menuLabel)}>
          <DropdownMenu.Label className="asst-menu-group">{t(mx.menuGroup)}</DropdownMenu.Label>
          {servers.map((s) => {
            const on = value.includes(s.id)
            return (
              <DropdownMenu.CheckboxItem
                key={s.id}
                className="asst-menu-item"
                checked={on}
                onCheckedChange={(next) => onToggle(s.id, next)}
                // The menu stays open to pick another.
                onSelect={(e) => e.preventDefault()}
              >
                <strong>{s.name}</strong>
                <span className="k">
                  <DropdownMenu.ItemIndicator>
                    <AsstIcon name="check" />
                  </DropdownMenu.ItemIndicator>
                </span>
                <small>{needsSignIn(s) ? t(mx.needsSignIn) : s.auth === 'oauth' ? t(mx.connected) : s.auth === 'header' ? t(mx.shared) : t(mx.noSignIn)}</small>
              </DropdownMenu.CheckboxItem>
            )
          })}
          {servers.filter(needsSignIn).map((s) => (
            <DropdownMenu.Item key={`sign-${s.id}`} className="asst-menu-item asst-menu-action" onSelect={() => onSignIn(s)}>
              <strong>{t(mx.signIn, { server: s.name })}</strong>
            </DropdownMenu.Item>
          ))}
          <DropdownMenu.Label className="asst-menu-note">{t(mx.menuNote)}</DropdownMenu.Label>
          {canManage ? (
            <DropdownMenu.Item
              className="asst-menu-item asst-menu-action"
              onSelect={() => {
                // A sheet or the full screen would cover the page it opens.
                if (!document.querySelector('.asst-docked')) closeAssistant({ refocus: false })
                void navigate({ to: '/w/$workspace/settings/integrations/mcp-servers', params: { workspace: slug } })
              }}
            >
              <strong>{t(mx.manage)}</strong>
            </DropdownMenu.Item>
          ) : null}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
