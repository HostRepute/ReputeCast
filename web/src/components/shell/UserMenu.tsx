import { Link, useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { DropdownMenu } from 'radix-ui'
import { http } from '@/api/client'
import { useMe, useOptionalWorkspace } from '@/lib/session'
import { Icon } from '../icons/Icon'
import { Avatar } from '../ui/Avatar'

/** Account menu: Account & security, notification preferences, settings, instance admin, sign out. */
export function UserMenu() {
  const { data: me } = useMe()
  const ws = useOptionalWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  if (!me) return null
  const u = me.user
  const first = (u.name || u.email).split(/\s+/)[0]
  const signOut = async () => {
    try {
      await http.POST('/api/v1/auth/sign-out')
    } finally {
      qc.clear()
      void navigate({ to: '/sign-in' })
    }
  }
  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>
        <button className="ubtn" type="button" aria-label={`Account menu for ${u.name || u.email}`}>
          <Avatar name={u.name || u.email} color={u.avatarColor} size={26} />
          <span className="unm">{first}</span>
          <Icon name="chevronDown" size={12} strokeWidth={2} />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="menu w-[280px]" align="end" sideOffset={6} collisionPadding={12}>
          <div className="row" style={{ gap: 10, padding: '8px 10px 10px', flexWrap: 'nowrap' }}>
            <Avatar name={u.name || u.email} color={u.avatarColor} size={36} />
            <span className="col min-w-0" style={{ gap: 0 }}>
              <b className="text-[13px]">{u.name || u.email}</b>
              <span className="k mono truncate">{u.email}</span>
            </span>
          </div>
          <DropdownMenu.Separator />
          <DropdownMenu.Item asChild>
            <Link to="/w/$workspace/account" params={{ workspace: ws?.slug ?? me.workspaces[0]?.slug ?? '' }}>
              <Icon name="user" />
              Account &amp; security
            </Link>
          </DropdownMenu.Item>
          <DropdownMenu.Item asChild>
            <Link to="/w/$workspace/account" params={{ workspace: ws?.slug ?? me.workspaces[0]?.slug ?? '' }} hash="notifications">
              <Icon name="bell" />
              Notification preferences
            </Link>
          </DropdownMenu.Item>
          {ws ? (
            <DropdownMenu.Item asChild>
              <Link to="/w/$workspace/settings" params={{ workspace: ws.slug }}>
                <Icon name="sliders" />
                Workspace settings
              </Link>
            </DropdownMenu.Item>
          ) : null}
          {u.isInstanceAdmin ? (
            <>
              <DropdownMenu.Separator />
              <DropdownMenu.Item asChild>
                <Link to="/w/$workspace/admin/users" params={{ workspace: ws?.slug ?? me.workspaces[0]?.slug ?? '' }}>
                  <Icon name="instance" />
                  Instance admin <span className="badge mut ml-auto">admins only</span>
                </Link>
              </DropdownMenu.Item>
            </>
          ) : null}
          <DropdownMenu.Separator />
          <DropdownMenu.Item onSelect={signOut}>
            <Icon name="signOut" />
            Sign out
          </DropdownMenu.Item>
          <span className="k" style={{ padding: '6px 10px 4px' }}>
            {me.instance.name} {me.instance.version}
            {me.instance.latestVersion ? (
              <>
                {' · '}
                {u.isInstanceAdmin && ws ? (
                  <Link to="/w/$workspace/admin/instance" params={{ workspace: ws.slug }}>
                    {me.instance.latestVersion} available
                  </Link>
                ) : (
                  `${me.instance.latestVersion} available`
                )}
              </>
            ) : null}
          </span>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
