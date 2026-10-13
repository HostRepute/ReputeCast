import { Link, useNavigate } from '@tanstack/react-router'
import { DropdownMenu } from 'radix-ui'
import { count } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { Icon } from '../icons/Icon'
import { Avatar } from '../ui/Avatar'

const wsColors = ['#0F766E', '#1D4ED8', '#7C3AED', '#B45309', '#BE185D', '#047857']

/** The colour of a workspace's initials square, by its position in my list. */
export function wsColor(i: number) {
  return wsColors[i % wsColors.length]!
}

/** Lists every workspace with my role, plus "Create workspace". */
export function WorkspaceSwitcher() {
  const { slug, me: ws } = useWorkspace()
  const { data: me } = useMe()
  const navigate = useNavigate()
  const list = me?.workspaces ?? []
  const idx = Math.max(0, list.findIndex((w) => w.slug === slug))
  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>
        <button className="ws" type="button" aria-label={`${ws.workspace.name}, ${ws.workspace.roleName}. Switch workspace`}>
          <span
            className="av"
            style={{ width: 24, height: 24, fontSize: 10, background: wsColor(idx), color: '#fff', borderRadius: 6 }}
            aria-hidden="true"
            data-initials={initials(ws.workspace.name)}
          />
          {/* The space keeps "name role" two words for speech input (WCAG 2.5.3); the flex column doesn't render it. */}
          <span className="wsn">
            <b>{ws.workspace.name}</b> <span>{ws.workspace.roleName}</span>
          </span>
          <Icon name="chevronDown" size={14} />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="menu w-[300px]" align="start" sideOffset={-10} collisionPadding={12} aria-label="Workspaces">
          <DropdownMenu.Label className="mh">Your workspaces</DropdownMenu.Label>
          {list.map((w, i) => (
            <DropdownMenu.Item key={w.id} asChild onSelect={() => navigate({ to: '/w/$workspace', params: { workspace: w.slug } })}>
              <Link to="/w/$workspace" params={{ workspace: w.slug }} aria-current={w.slug === slug ? 'true' : undefined} style={w.slug === slug ? { background: '#E3F1EE' } : undefined}>
                <span className="av" style={{ width: 26, height: 26, fontSize: 10, background: wsColor(i), color: '#fff', borderRadius: 6 }} aria-hidden="true">
                  {initials(w.name)}
                </span>
                <span className="col min-w-0 flex-1" style={{ gap: 0 }}>
                  <b className="text-[13px]">{w.name}</b>
                  <span className="k">
                    {w.roleName} · {count(w.subscribers)} subscribers
                  </span>
                </span>
                {w.slug === slug ? <span className="badge ok">✓</span> : null}
              </Link>
            </DropdownMenu.Item>
          ))}
          <DropdownMenu.Separator />
          <DropdownMenu.Item asChild>
            <Link to="/workspaces/new">
              <Icon name="plus" strokeWidth={1.6} />
              Create workspace
            </Link>
          </DropdownMenu.Item>
          {me?.user.isInstanceAdmin ? (
            <DropdownMenu.Item asChild>
              <Link to="/w/$workspace/admin/workspaces" params={{ workspace: slug }}>
                <Icon name="table" />
                All workspaces (instance admin)
              </Link>
            </DropdownMenu.Item>
          ) : null}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}

export function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join('')
}

export { Avatar }
