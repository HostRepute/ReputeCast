import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { LanguageRegionCard } from '@/features/language/LanguageRegionCard'
import { meQueryKey, useWorkspace } from '@/lib/session'
import { ConnectedSignInsCard } from './ConnectedSignInsCard'
import { NotificationPreferencesCard, WorkspacesCard } from './OtherCards'
import { ProfileCard } from './ProfileCard'
import { AuthenticatorCard, PasskeysCard, PasswordCard } from './SecurityCards'
import { SessionsCard } from './SessionsCard'

export const accountKey = ['get', '/api/v1/me/account'] as const

/** Refetches everything Account & security shows, and the shell's /me. */
export function useRefreshAccount() {
  const qc = useQueryClient()
  return async () => {
    await Promise.all([qc.invalidateQueries({ queryKey: accountKey }), qc.invalidateQueries({ queryKey: meQueryKey })])
  }
}

/** A5 §5.6 Account & security (AccountSecurity and AccountStates boards). */
export function AccountPage({ verifyEmail }: { verifyEmail?: string }) {
  const { me: ws } = useWorkspace()
  const account = $api.useQuery('get', '/api/v1/me/account')
  const a = account.data
  const subtitle = a ? [a.profile.email, `${ws.workspace.roleName} of ${ws.workspace.name}`, a.profile.isInstanceAdmin ? 'instance admin' : null].filter(Boolean).join(' · ') : undefined
  return (
    <Page title="Account & security" subtitle={subtitle}>
      <VerifyEmail token={verifyEmail} />
      {account.isError ? (
        <ErrorPanel title="Couldn't load your account" actions={<Button onClick={() => account.refetch()}>Retry</Button>}>
          {errorMessage(account.error)} Nothing was changed.
        </ErrorPanel>
      ) : !a ? (
        <AccountSkeleton />
      ) : (
        <Grid min={420} gap={20} className="items-start">
          {a.password.ssoOnly ? (
            <>
              {/* SSO-only members have no Password card: their sign-in card
                  (which explains SSO) sits beside the profile and passkeys. */}
              <div className="col min-w-0" style={{ gap: 20 }}>
                <ProfileCard a={a} />
                <LanguageRegionCard />
                <PasskeysCard a={a} />
                {a.twoFactor.totp ? <AuthenticatorCard a={a} /> : null}
              </div>
              <ConnectedSignInsCard a={a} />
            </>
          ) : (
            <>
              <ProfileCard a={a} />
              {/* Language & region follows Profile (LanguageRegion board). */}
              <LanguageRegionCard />
              <PasswordCard a={a} />
              <PasskeysCard a={a} />
              {a.password.set || a.twoFactor.totp ? <AuthenticatorCard a={a} /> : null}
              <ConnectedSignInsCard a={a} />
            </>
          )}
          <SessionsCard policy={a.sessionPolicy} />
          <WorkspacesCard a={a} />
          <NotificationPreferencesCard a={a} />
        </Grid>
      )}
    </Page>
  )
}

/** Confirms an email change from the verification link (…/account?verifyEmail=…). */
function VerifyEmail({ token }: { token?: string }) {
  const navigate = useNavigate()
  const refresh = useRefreshAccount()
  const qc = useQueryClient()
  const [state, setState] = useState<{ ok?: string; error?: string }>({})
  const ran = useRef(false)
  useEffect(() => {
    if (!token || ran.current) return
    ran.current = true
    void (async () => {
      try {
        const { data } = await http.POST('/api/v1/me/email/verify', { body: { token } })
        setState({ ok: data?.email })
        toast({ kind: 'success', title: 'Email changed', detail: `You sign in with ${data?.email} now.` })
        await refresh()
        await qc.invalidateQueries({ queryKey: meQueryKey })
      } catch (e) {
        setState({ error: errorMessage(e) })
      } finally {
        void navigate({ to: '.', search: {}, replace: true })
      }
    })()
  }, [token, navigate, refresh, qc])
  if (state.ok)
    return (
      <Note tone="ok" label="Email changed">
        You sign in with {state.ok} now. Your old address got a note about the change.
      </Note>
    )
  if (state.error)
    return (
      <Note tone="bad" label="Not changed">
        {state.error}
      </Note>
    )
  return null
}

function CardSkeleton({ rows = 3, wide }: { rows?: number; wide?: boolean }) {
  return (
    <section className="card" style={wide ? { gridColumn: '1 / -1' } : undefined}>
      <div className="sec">
        <div className="flex flex-col gap-1.5">
          <Skeleton w={140} h={15} />
          <Skeleton w={240} h={12} />
        </div>
      </div>
      <div className="pad col" style={{ gap: 12 }}>
        {Array.from({ length: rows }, (_, i) => (
          <Skeleton key={i} w={`${80 - i * 12}%`} h={14} />
        ))}
      </div>
    </section>
  )
}

function AccountSkeleton() {
  return (
    <Grid min={420} gap={20} className="items-start" aria-busy="true">
      <span className="vh" role="status">
        Loading your account…
      </span>
      <CardSkeleton rows={5} />
      <CardSkeleton rows={4} />
      <CardSkeleton rows={3} />
      <CardSkeleton rows={3} />
      <CardSkeleton rows={4} />
      <CardSkeleton rows={3} wide />
    </Grid>
  )
}
