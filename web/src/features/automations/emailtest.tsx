import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { T, useT } from '@/i18n'
import { useDebounced } from '@/lib/hooks'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { m } from './complete.messages'

type Contact = Schemas['ContactListItem']

/**
 * An email step's Preview as and Send test (review 2026-11 AT-3): the email
 * personalized as a real contact, and sent as a test through the System
 * route.
 */
export function EmailTestFields({ automationId, emailId, version }: { automationId: string; emailId?: string; version?: number }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const me = useMe().data
  const canContacts = useCan('subscribers:view')
  const [query, setQuery] = useState('')
  const [contact, setContact] = useState<Contact>()
  const [to, setTo] = useState('')
  const [sent, setSent] = useState<string>()
  const [err, setErr] = useState<string>()
  const [previewing, setPreviewing] = useState(false)
  const q = useDebounced(query.trim(), 250)
  const found = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/contacts',
    { params: { path: { workspace: slug }, query: { q, size: 5 } } },
    { enabled: canContacts && q.length >= 2 && !contact },
  )
  const test = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automation-emails/{id}/test')
  const address = to || me?.user.email || ''
  if (!emailId) return null
  const who = contact ? contact.name || contact.email : undefined
  const send = () => {
    setErr(undefined)
    setSent(undefined)
    test.mutate(
      { params: { path: { workspace: slug, id: emailId } }, body: { to: [address], contactId: contact?.id, version, automationId } },
      {
        onSuccess: (r) => setSent(tr(m.testSent, { to: r.sent.join(', '), contact: who ?? 'none' })),
        onError: (e) => setErr(errorMessage(e)),
      },
    )
  }
  return (
    <>
      {canContacts ? (
        <div className="flex flex-col gap-1.5">
          <label className="tt" htmlFor="p-previewas">
            {tr(m.previewAs)}
          </label>
          {contact ? (
            <span className="flex items-center gap-2 text-[13px]">
              <b className="min-w-0 flex-1 truncate">{who}</b>
              <button type="button" className="text-[13px] font-semibold text-accent underline underline-offset-2" onClick={() => setContact(undefined)}>
                {tr(m.changeContact)}
              </button>
            </span>
          ) : (
            <>
              <input id="p-previewas" className="in" type="search" placeholder={tr(m.searchContact)} value={query} onChange={(e) => setQuery(e.target.value)} />
              {found.data?.items.length ? (
                <ul className="m-0 flex list-none flex-col gap-0.5 p-0" aria-label={tr(m.searchContact)}>
                  {found.data.items.map((c) => (
                    <li key={c.id}>
                      <button type="button" className="w-full rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-bg-2" onClick={() => setContact(c)}>
                        <b>{c.name || c.email}</b> {c.name ? <span className="k mono">{c.email}</span> : null}
                      </button>
                    </li>
                  ))}
                </ul>
              ) : null}
            </>
          )}
          <span className="k">{tr(m.previewAsHint)}</span>
          {who ? (
            <Button className="mini-btn self-start" onClick={() => setPreviewing(true)}>
              {tr(m.previewTitle, { contact: who })}
            </Button>
          ) : null}
        </div>
      ) : null}
      <div className="flex flex-col gap-1.5">
        <label className="tt" htmlFor="p-testto">
          {tr(m.sendTestTo)}
        </label>
        <span className="flex gap-1.5">
          <input id="p-testto" className="in min-w-0 flex-1" type="email" autoComplete="email" value={address} onChange={(e) => setTo(e.target.value)} />
          <Button className="mini-btn" loading={test.isPending} disabled={!address} onClick={send}>
            {tr(m.sendTest)}
          </Button>
        </span>
        <span className="k">{tr(m.testHint)}</span>
        {sent ? (
          <Note tone="info" role="status">
            {sent}
          </Note>
        ) : null}
        {err ? <Note tone="bad">{err}</Note> : null}
      </div>
      {previewing ? <PreviewAsDialog emailId={emailId} version={version} contact={contact} onClose={() => setPreviewing(false)} /> : null}
    </>
  )
}

function PreviewAsDialog({ emailId, version, contact, onClose }: { emailId: string; version?: number; contact?: Contact; onClose: () => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const pv = $api.useQuery('post', '/api/v1/workspaces/{workspace}/automation-emails/{id}/preview', {
    params: { path: { workspace: slug, id: emailId } },
    body: { contactId: contact?.id, version },
  })
  const who = contact ? contact.name || contact.email : tr(m.noContact)
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} size="lg" title={tr(m.previewTitle, { contact: who })} description={pv.data?.subject}>
      {pv.isError ? (
        <Note tone="bad">{errorMessage(pv.error)}</Note>
      ) : pv.data ? (
        <>
          {pv.data.problems.length ? <Note tone="warn">{pv.data.problems.join(' ')}</Note> : null}
          <iframe title={tr(m.previewTitle, { contact: who })} sandbox="" srcDoc={pv.data.html} className="h-[480px] w-full rounded-lg border border-line bg-white" />
        </>
      ) : (
        <span className="k" role="status">
          …
        </span>
      )}
    </Dialog>
  )
}

/**
 * What an email step does when the contact is over the workspace's
 * marketing cap (review 2026-11 AT-9): wait, or skip.
 */
export function OverCapChoice({ value, onChange, disabled }: { value?: string; onChange: (v: 'wait' | 'skip') => void; disabled: boolean }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const rules = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sending-rules', { params: { path: { workspace: slug } } })
  const cap = rules.data?.marketingCap
  const choice = value === 'skip' ? 'skip' : 'wait'
  const b = (c: ReactNode) => <b>{c}</b>
  return (
    <div role="radiogroup" aria-labelledby="p-overcap" className="flex flex-col gap-1.5">
      {(['wait', 'skip'] as const).map((v) => (
        <label key={v} className={`dopt${choice === v ? ' on' : ''}`}>
          <input type="radio" name="p-overcap" checked={choice === v} disabled={disabled} onChange={() => onChange(v)} />
          <span>
            <T msg={v === 'wait' ? m.overCapWait : m.overCapSkip} values={{ b }} />
          </span>
        </label>
      ))}
      {cap ? (
        <span className="k">
          {cap.count > 0 ? (
            tr(m.capOn, { count: cap.count, days: cap.days })
          ) : (
            <T
              msg={m.capOff}
              values={{
                link: (c) => (
                  <Link to="/w/$workspace/settings/sending-rules" params={{ workspace: slug }}>
                    {c}
                  </Link>
                ),
              }}
            />
          )}
        </span>
      ) : null}
    </div>
  )
}
