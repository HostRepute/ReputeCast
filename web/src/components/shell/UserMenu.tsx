import { Link, useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { DropdownMenu } from 'radix-ui'
import { http } from '@/api/client'
import { LanguageMenuItem } from '@/features/language/LanguageMenuItem'
import { T, useT } from '@/i18n'
import { useMediaQuery } from '@/lib/hooks'
import { useMe, useOptionalWorkspace } from '@/lib/session'
import { Icon } from '../icons/Icon'
import { Avatar } from '../ui/Avatar'
import { m } from './messages'
import { initials, wsColor } from './WorkspaceSwitcher'

/** Account menu: Account & security, notification preferences, settings, instance admin, sign out (ShellUserMenu). */
export function UserMenu() {
  const { data: me } = useMe()
  const ws = useOptionalWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const t = useT()
  // The sidebar's workspace switcher is hidden on the icon rail and the
  // bottom tab bar (≤1100px), so the menu lists the workspaces there.
  const railed = useMediaQuery('(max-width: 1100px)')
  if (!me) return null
  const u = me.user
  const first = (u.name || u.email).split(/\s+/)[0]
  const slug = ws?.slug ?? me.workspaces[0]?.slug ?? ''
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
        <button className="ubtn" type="button" aria-label={t(m.userMenuLabel, { name: u.name || u.email })}>
          <Avatar name={u.name || u.email} color={u.avatarColor} size={26} src={u.avatarUrl} />
          <span className="unm">{first}</span>
          <Icon name="chevronDown" size={12} strokeWidth={2} />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="menu w-[280px]" align="end" sideOffset={11} collisionPadding={12} aria-label={t(m.userMenuContentLabel)}>
          <div className="row" style={{ gap: 10, padding: '8px 10px 10px', flexWrap: 'nowrap' }}>
            <Avatar name={u.name || u.email} color={u.avatarColor} size={36} fontSize={13} src={u.avatarUrl} />
            <span className="col min-w-0" style={{ gap: 0 }}>
              <b className="text-[13px]">{u.name || u.email}</b>
              <span className="k mono truncate">{u.email}</span>
            </span>
          </div>
          <DropdownMenu.Separator />
          {railed && me.workspaces.length > 1 ? (
            <DropdownMenu.Group aria-label={t(m.switcherHeading)}>
              <DropdownMenu.Label className="mh">{t(m.switcherHeading)}</DropdownMenu.Label>
              {me.workspaces.map((w, i) => (
                <DropdownMenu.Item key={w.id} asChild>
                  <Link to="/w/$workspace" params={{ workspace: w.slug }} aria-current={w.slug === slug ? 'true' : undefined}>
                    <span className="av" style={{ width: 20, height: 20, fontSize: 9, background: wsColor(i), color: '#fff', borderRadius: 5 }} aria-hidden="true">
                      {initials(w.name)}
                    </span>
                    <span className="min-w-0 flex-1 truncate">{w.name}</span>
                    {w.slug === slug ? <Icon name="check" size={14} className="text-accent" /> : null}
                  </Link>
                </DropdownMenu.Item>
              ))}
              <DropdownMenu.Separator />
            </DropdownMenu.Group>
          ) : null}
          <DropdownMenu.Item asChild>
            <Link to="/w/$workspace/account" params={{ workspace: slug }}>
              <Icon name="user" />
              {t(m.userMenuAccount)}
            </Link>
          </DropdownMenu.Item>
          <LanguageMenuItem slug={slug} />

          <DropdownMenu.Item asChild>
            <Link to="/w/$workspace/notifications" params={{ workspace: slug }} hash="preferences">
              <Icon name="bell" />
              {t(m.userMenuNotifications)}
            </Link>
          </DropdownMenu.Item>
          {ws ? (
            <DropdownMenu.Item asChild>
              <Link to="/w/$workspace/settings" params={{ workspace: ws.slug }}>
                <Icon name="sliders" />
                {t(m.userMenuWorkspaceSettings)}
              </Link>
            </DropdownMenu.Item>
          ) : null}
          {u.isInstanceAdmin ? (
            <>
              <DropdownMenu.Separator />
              <DropdownMenu.Item asChild>
                <Link to="/w/$workspace/admin/users" params={{ workspace: slug }}>
                  <Icon name="instance" />
                  {t(m.userMenuInstanceAdmin)} <span className="badge mut ml-auto">{t(m.userMenuAdminsOnly)}</span>
                </Link>
              </DropdownMenu.Item>
            </>
          ) : null}
          <DropdownMenu.Separator />
          <DropdownMenu.Item onSelect={signOut}>
            <Icon name="signOut" />
            {t(m.userMenuSignOut)}
          </DropdownMenu.Item>
          <span className="k" style={{ padding: '6px 10px 4px' }}>
            {me.instance.latestVersion ? (
              <T
                msg={m.userMenuVersionUpdate}
                values={{
                  name: me.instance.name,
                  version: me.instance.version,
                  latest: me.instance.latestVersion,
                  link: (chunks) =>
                    u.isInstanceAdmin && ws ? (
                      <Link to="/w/$workspace/admin/instance" params={{ workspace: ws.slug }} className="underline underline-offset-2">
                        {chunks}
                      </Link>
                    ) : (
                      chunks
                    ),
                }}
              />
            ) : (
              t(m.userMenuVersion, { name: me.instance.name, version: me.instance.version })
            )}
          </span>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
