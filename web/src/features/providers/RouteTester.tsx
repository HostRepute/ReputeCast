import { keepPreviousData } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Field, Input, Select } from '@/components/ui/Field'
import { useWorkspace } from '@/lib/session'
import { providerColor, streamName, type Stream } from './common'
import { costText, fallbackText, mailboxName, stepText, toInput, type Draft, type RoutingConfig } from './routing-model'

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const h = window.setTimeout(() => setV(value), ms)
    return () => window.clearTimeout(h)
  }, [value, ms])
  return v
}

const looksLikeEmail = (s: string) => /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(s.trim())

/**
 * "Which provider would send this email?" Runs the routing (the unsaved
 * draft when there is one) for one address and explains each step.
 */
export function RouteTester({ cfg, draft }: { cfg: RoutingConfig; draft?: Draft }) {
  const { slug } = useWorkspace()
  const [stream, setStream] = useState<Stream>('broadcast')
  const [listId, setListId] = useState(cfg.lists[0]?.id ?? '')
  const [domain, setDomain] = useState(cfg.senderDomains[0] ?? '')
  const [email, setEmail] = useState('sam.rivera@outlook.com')
  const address = useDebounced(email.trim(), 300)
  const valid = looksLikeEmail(address)
  const q = $api.useQuery(
    'post',
    '/api/v1/workspaces/{workspace}/routing/test',
    {
      params: { path: { workspace: slug } },
      body: { stream, listId: listId || undefined, senderDomain: domain || undefined, email: address, draft: draft ? toInput(draft, true) : undefined },
    },
    { enabled: valid, placeholderData: keepPreviousData, retry: false },
  )
  const r = valid ? q.data : undefined
  const hasConditions = Boolean(draft ? draft[stream].conditions.length : cfg.streams.find((s) => s.stream === stream)?.conditions.length)
  return (
    <section className="card flex flex-col" aria-labelledby="pv-tester">
      <div className="flex flex-col gap-0.5 border-b border-line-2 px-5 py-3.5">
        <h2 className="h2" id="pv-tester">
          Route tester
        </h2>
        <span className="k">Which provider would send this email?</span>
      </div>
      <div className="flex flex-col gap-3 px-5 py-[18px]">
        <Field label="Stream">
          <Select value={stream} onChange={(e) => setStream(e.target.value as Stream)}>
            {(['broadcast', 'automation', 'system'] as Stream[]).map((s) => (
              <option key={s} value={s}>
                {streamName[s]}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="List">
          <Select value={listId} onChange={(e) => setListId(e.target.value)}>
            {cfg.lists.length ? null : <option value="">No lists yet</option>}
            {cfg.lists.map((l) => (
              <option key={l.id} value={l.id}>
                {l.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Sender domain">
          <Select value={domain} onChange={(e) => setDomain(e.target.value)}>
            {cfg.senderDomains.length ? null : <option value="">No sending domain yet</option>}
            {cfg.senderDomains.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Recipient">
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="name@example.com" autoComplete="off" spellCheck={false} />
        </Field>
        <div className="flex flex-col gap-2.5 rounded-[10px] border border-line bg-ground-3 p-3.5" aria-live="polite">
          {!valid ? (
            <span className="k">Enter an address, like name@example.com, to see which provider would send to it.</span>
          ) : q.isError ? (
            <span className="text-[13px] text-bad-fg" role="alert">
              {errorMessage(q.error)}
            </span>
          ) : !r ? (
            <span className="k">Working out the route…</span>
          ) : (
            <>
              <div className="flex items-center gap-2.5">
                <span className="dot h-3 w-3" style={{ color: providerColor[r.color] }} aria-hidden="true" />
                <span className="text-[18px] font-bold">{r.provider}</span>
              </div>
              <Row k="Mailbox provider">
                <span className="n">{mailboxName[r.mailboxProvider] ?? r.mailboxProvider}</span>
              </Row>
              <Row k="DKIM selector">
                <span className="mono">{r.dkimSelector || 'not set up yet'}</span>
              </Row>
              <Row k="Cost">
                <span className="n">{costText(r.cost)}</span>
              </Row>
              <ol className="m-0 flex list-none flex-col gap-2 border-t border-line pt-2.5 pl-0">
                {r.steps.map((st, i) => (
                  <li key={i} className="pv-step">
                    <span className="num" aria-hidden="true">
                      {i + 1}
                    </span>
                    <span>{stepText(st, stream, hasConditions)}</span>
                  </li>
                ))}
              </ol>
              <div className="text-[12.5px] text-ink-2">
                <span className="font-semibold text-ink">If it fails:</span> {fallbackText(r, stream)}
              </div>
            </>
          )}
        </div>
      </div>
    </section>
  )
}

function Row({ k, children }: { k: string; children: React.ReactNode }) {
  return (
    <div className="flex justify-between gap-3 text-[13px] max-md:flex-wrap">
      <span className="k">{k}</span>
      {children}
    </div>
  )
}
