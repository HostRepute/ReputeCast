import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { GoogleMark } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem, MenuLabel } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { andList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { copyText } from '../access/parts'

const EMAIL = /^[^\s@,;]+@[^\s@,;]+\.[^\s@,;]+$/

/** The invite dialog (TeamInvite, TeamInviteNoProvider). */
export function InviteDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const opts = $api.useQuery('get', '/api/v1/workspaces/{workspace}/invitations/options', { params: { path: { workspace: slug } } })
  if (opts.data) return <InviteForm opts={opts.data} open={open} close={() => onOpenChange(false)} />
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">Team · invite</span>}
      title="Invite members"
      className="card max-w-[680px]"
      hideClose
      footer={<Button onClick={() => onOpenChange(false)}>Cancel</Button>}
    >
      {opts.isError ? (
        <div className="note b" role="alert">
          <span>{errorMessage(opts.error)}</span>
        </div>
      ) : (
        <div className="flex flex-col gap-3" aria-busy="true">
          <span className="vh" role="status">
            Loading
          </span>
          <Skeleton h={40} />
          <Skeleton h={36} />
          <Skeleton h={72} />
        </div>
      )}
    </Dialog>
  )
}

type Created = Schemas['TeamInviteResult']

function InviteForm({ opts, open, close }: { opts: Schemas['TeamInviteOptions']; open: boolean; close: () => void }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const roles = useMemo(() => orderRoles(opts.roles), [opts.roles])
  const [emails, setEmails] = useState<string[]>([])
  const [draft, setDraft] = useState('')
  const [roleId, setRoleId] = useState(opts.defaultRoleId)
  const [days, setDays] = useState(opts.defaultDays)
  const [lists, setLists] = useState<string[]>([])
  const [domains, setDomains] = useState<string[]>([])
  const [message, setMessage] = useState('')
  const [error, setError] = useState<string>()
  const [created, setCreated] = useState<Created>()
  const inputRef = useRef<HTMLInputElement>(null)
  const role = roles.find((r) => r.id === roleId)
  const invite = $api.useMutation('post', '/api/v1/workspaces/{workspace}/invitations')

  const add = (raw: string) => {
    const parts = raw
      .split(/[\s,;]+/)
      .map((p) => p.trim().toLowerCase())
      .filter(Boolean)
    if (parts.length === 0) return
    const bad = parts.filter((p) => !EMAIL.test(p))
    setEmails((cur) => [...cur, ...parts.filter((p) => EMAIL.test(p) && !cur.includes(p))])
    setError(bad.length ? `“${bad[0]}” isn't an email address.` : undefined)
    setDraft(bad.join(' '))
  }
  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',' || e.key === ';' || (e.key === ' ' && draft.includes('@'))) {
      e.preventDefault()
      add(draft)
    } else if (e.key === 'Backspace' && draft === '' && emails.length > 0) {
      setEmails((cur) => cur.slice(0, -1))
    }
  }
  const jitHits = opts.jit.flatMap((j) => emails.filter((e) => e.endsWith('@' + j.domain)).map((e) => ({ email: e, jit: j })))

  const submit = () => {
    const all = draft.trim() ? [...emails, ...draft.split(/[\s,;]+/).filter(Boolean)] : emails
    if (all.length === 0) {
      setError('Add at least one email address.')
      inputRef.current?.focus()
      return
    }
    invite.mutate(
      {
        params: { path: { workspace: slug } },
        body: {
          emails: all,
          roleId,
          expiresInDays: days,
          message: message.trim() || undefined,
          scope: lists.length || domains.length ? { lists, senderDomains: domains } : undefined,
          sendEmail: opts.canEmail,
        },
      },
      {
        onSuccess: async (res) => {
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/invitations'] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/members'] })
          if (res.emailed) {
            toast({ title: `${res.invitations.length} ${res.invitations.length === 1 ? 'invitation' : 'invitations'} sent`, detail: res.sentVia ? `via ${res.sentVia}` : undefined })
            close()
            return
          }
          setCreated(res)
          const ok = await copyText(res.invitations.map((i) => `${i.email}: ${i.link}`).join('\n'))
          toast({ title: `${res.invitations.length} ${res.invitations.length === 1 ? 'invite' : 'invites'} created`, detail: ok ? 'links copied' : 'copy the links below' })
        },
        onError: (e) => setError(e instanceof ApiError ? e.message : errorMessage(e)),
      },
    )
  }

  const chosenLists = opts.lists.filter((l) => lists.includes(l.id))
  const freeLists = opts.lists.filter((l) => !lists.includes(l.id))
  const freeDomains = opts.senderDomains.filter((d) => !domains.includes(d))
  const n = emails.length + (draft.trim() && EMAIL.test(draft.trim()) ? 1 : 0)
  const firstLink = created?.invitations[0]?.link

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && close()}
      eyebrow={<span className="k">Team · invite</span>}
      title={`Invite members to ${opts.workspace}`}
      description={
        <span className="text-[13.5px]">
          {opts.signInSummary}
          {opts.jit[0] ? ` SSO users join through ${opts.jit[0].provider}.` : ''}
        </span>
      }
      className="card max-w-[680px]"
      hideClose
      dismissible={false}
      footer={
        <InviteFooter
          canEmail={opts.canEmail}
          provider={opts.systemProvider}
          n={n}
          busy={invite.isPending}
          created={Boolean(created)}
          roleBlocked={role && !role.grantable ? role.reason : undefined}
          onCancel={close}
          onSubmit={submit}
        />
      }
    >
      <div className="col gap-1.5">
        <span className="text-[12px] font-semibold text-ink-2" id="inv-emails">
          Email addresses
        </span>
        {/* A label, so a click anywhere in the box focuses the input. */}
        <label
          htmlFor="inv-email-input"
          className="row min-h-[40px] cursor-text gap-1.5 rounded-lg border border-line-input px-2 py-1.5 box-border focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-(--focus)"
          style={error ? { borderColor: 'var(--bad-bar)' } : undefined}
        >
          {emails.map((e) => (
            <span key={e} className="chip">
              {e}
              <button type="button" aria-label={`Remove ${e}`} onClick={() => setEmails((cur) => cur.filter((x) => x !== e))}>
                ×
              </button>
            </span>
          ))}
          <input
            ref={inputRef}
            id="inv-email-input"
            aria-labelledby="inv-emails"
            aria-describedby={error ? 'inv-err' : undefined}
            className="in h-[26px] min-w-[160px] flex-1 border-0 px-1 focus:outline-none"
            placeholder={emails.length ? 'Add another, comma or Enter' : 'name@example.com, comma or Enter'}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={onKey}
            onBlur={() => draft.trim() && add(draft)}
            onPaste={(e) => {
              const t = e.clipboardData.getData('text')
              if (/[\s,;]/.test(t.trim())) {
                e.preventDefault()
                add(draft + t)
              }
            }}
            disabled={Boolean(created)}
          />
        </label>
        {error ? (
          <span className="err" id="inv-err" role="alert">
            {error}
          </span>
        ) : null}
        {jitHits.length > 0 ? (
          <span className="note i px-2.5 py-2 text-[12.5px]">
            <GoogleMark size={14} />
            <span>
              {andList(jitHits.map((h) => h.email))} can also join automatically with {jitHits[0]!.jit.provider} (as {jitHits[0]!.jit.roleName}). This invite gives the role below
              instead.
            </span>
          </span>
        ) : null}
      </div>
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          Role
          <select className="in" value={roleId} onChange={(e) => setRoleId(e.target.value)} disabled={Boolean(created)}>
            {roles.map((r) => (
              <option key={r.id} value={r.id} disabled={!r.grantable}>
                {r.name}
                {r.grantable ? '' : ' (you can’t grant it)'}
              </option>
            ))}
          </select>
          <span className="k font-medium">{role?.hint}</span>
        </label>
        <label className="lbl">
          Invitation expires
          <select className="in" value={days} onChange={(e) => setDays(Number(e.target.value))} disabled={Boolean(created)}>
            <option value={opts.defaultDays}>In {opts.defaultDays} days (default)</option>
            <option value={1}>In 24 hours</option>
            <option value={30}>In 30 days</option>
          </select>
        </label>
      </div>
      <div className="lbl">
        <span>
          Scope <span className="k font-medium">· optional</span>
        </span>
        <div className="row gap-1.5">
          {chosenLists.map((l) => (
            <span key={l.id} className="chip">
              {l.name} list
              <button type="button" aria-label={`Remove scope ${l.name}`} onClick={() => setLists((cur) => cur.filter((x) => x !== l.id))}>
                ×
              </button>
            </span>
          ))}
          {domains.map((d) => (
            <span key={d} className="chip">
              {d}
              <button type="button" aria-label={`Remove scope ${d}`} onClick={() => setDomains((cur) => cur.filter((x) => x !== d))}>
                ×
              </button>
            </span>
          ))}
          {freeLists.length + freeDomains.length > 0 && !created ? (
            <Menu trigger={<Button size="sm">Add list or sender domain</Button>} align="start" label="Add to scope">
              {freeLists.length ? <MenuLabel>Lists</MenuLabel> : null}
              {freeLists.map((l) => (
                <MenuItem key={l.id} onSelect={() => setLists((cur) => [...cur, l.id])}>
                  {l.name}
                </MenuItem>
              ))}
              {freeDomains.length ? <MenuLabel>Sender domains</MenuLabel> : null}
              {freeDomains.map((d) => (
                <MenuItem key={d} onSelect={() => setDomains((cur) => [...cur, d])}>
                  {d}
                </MenuItem>
              ))}
            </Menu>
          ) : null}
        </div>
        <span className="k font-medium">Scoped members only see and act on the selected lists or sender domains.</span>
      </div>
      <label className="lbl">
        <span>
          Personal message <span className="k font-medium">· optional</span>
        </span>
        <textarea className="in h-auto px-2.5 py-2" rows={3} value={message} onChange={(e) => setMessage(e.target.value)} maxLength={1000} disabled={Boolean(created)} />
      </label>
      {!opts.canEmail ? (
        <div className="note w" role="status">
          <Badge tone="warn">No System provider</Badge>
          <span className="col gap-1.5">
            <span>
              Email can&apos;t be sent yet: no sending provider is assigned to the System stream. Create the invites and copy the links instead, or{' '}
              <Link to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
                connect a provider
              </Link>
              .
            </span>
            <span className="row gap-1.5">
              <span className="mono max-w-[300px] truncate rounded-md border border-dashed border-muted-line bg-white px-2 py-1.5 text-[12px] text-ink">
                {firstLink ? `${firstLink.slice(0, 40)}…` : `${window.location.origin}/invite/…`}
              </span>
              <Button
                size="sm"
                loading={!created && invite.isPending}
                onClick={async () => {
                  // Before the invites exist, this creates them and copies their links.
                  if (!created) {
                    submit()
                    return
                  }
                  const ok = await copyText(created.invitations.map((i) => `${i.email}: ${i.link}`).join('\n'))
                  toast({ title: ok ? 'Invite links copied' : 'Copy failed', detail: ok ? `${created.invitations.length} ${created.invitations.length === 1 ? 'link' : 'links'}` : 'select the links and copy them' })
                }}
              >
                Copy links
              </Button>
            </span>
          </span>
        </div>
      ) : null}
    </Dialog>
  )
}

function InviteFooter({
  canEmail,
  provider,
  n,
  busy,
  created,
  roleBlocked,
  onCancel,
  onSubmit,
}: {
  canEmail: boolean
  provider: string
  n: number
  busy: boolean
  created: boolean
  roleBlocked?: string
  onCancel: () => void
  onSubmit: () => void
}) {
  return (
    <>
      {canEmail ? <span className="k mr-auto">Sent on the System stream{provider ? ` via ${provider}` : ''}</span> : null}
      {created ? (
        <Button variant="primary" onClick={onCancel}>
          Done
        </Button>
      ) : (
        <>
          <Button onClick={onCancel}>Cancel</Button>
          <Button variant="primary" loading={busy} disabledReason={roleBlocked} onClick={onSubmit}>
            {canEmail ? `Send ${n || ''} ${n === 1 ? 'invitation' : 'invitations'}`.replace('  ', ' ') : 'Create invites and copy links'}
          </Button>
        </>
      )}
    </>
  )
}

/** Editor, Viewer, custom roles, then Admin (the order on the board); never Owner. */
function orderRoles(roles: Schemas['TeamInviteRole'][]) {
  const rank = (r: Schemas['TeamInviteRole']) => (r.key === 'editor' ? 0 : r.key === 'viewer' ? 1 : !r.builtin ? 2 : r.key === 'admin' ? 3 : 4)
  return roles.filter((r) => !(r.builtin && r.key === 'owner')).sort((a, b) => rank(a) - rank(b))
}
