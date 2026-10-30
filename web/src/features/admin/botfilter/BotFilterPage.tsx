import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog, DialogRow } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Field } from '@/components/ui/Field'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable } from '@/features/access/parts'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, count, usDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { m } from './messages'

type Network = Schemas['TrackingNetwork']
type Agent = Schemas['TrackingUserAgent']

const networksKey = ['get', '/api/v1/admin/tracking-networks']
const agentsKey = ['get', '/api/v1/admin/tracking-user-agents']

/**
 * Instance admin › Bot filter (InstanceBotFilter; review AN-2): the
 * networks and user agents whose opens and clicks aren't people's, the
 * built-in ones and those admins add, and how the filter decides.
 */
export function BotFilterPage() {
  const t = useT()
  return (
    <AdminPage documentTitle={t(m.documentTitle)}>
      <Note tone="info">
        <span className="col gap-1">
          <span className="font-semibold">{t(m.introTitle)}</span>
          <span>{t(m.intro)}</span>
        </span>
      </Note>
      <NetworksCard />
      <AgentsCard />
      <RulesCard />
    </AdminPage>
  )
}

/** Who added an entry, and when. */
function AddedBy({ name, at }: { name?: string; at: string }) {
  return (
    <span className="col gap-0.5">
      {name ? <span>{name}</span> : null}
      <span className="k">{usDate(at)}</span>
    </span>
  )
}

/** Its name, with the note under it. */
function Named({ name, note }: { name: string; note?: string }) {
  const t = useT()
  if (!name) return <span className="k">{t(m.noName)}</span>
  return (
    <span className="col gap-0.5">
      <span className="font-semibold">{name}</span>
      {note ? <span className="k">{note}</span> : null}
    </span>
  )
}

/** The built-in entries, folded under one line (they come with releases). */
function Builtin({ summary, note, children }: { summary: string; note: string; children: ReactNode }) {
  const t = useT()
  return (
    <details className="group border-t border-line-2">
      <summary className="flex cursor-pointer list-none flex-wrap items-center gap-2 px-4 py-3 text-[13px] font-semibold [&::-webkit-details-marker]:hidden">
        <Icon name="chevronRight" size={13} className="flex-none text-ink-3 transition-transform group-open:rotate-90" />
        {summary}
        <span className="k font-medium">
          {note} · {t(m.builtinSummary)}
        </span>
      </summary>
      {children}
    </details>
  )
}

function Loading({ label }: { label: string }) {
  return (
    <div className="pad col gap-3" aria-busy="true">
      <span className="vh" role="status">
        {label}
      </span>
      <Skeleton w="100%" h={20} />
      <Skeleton w="100%" h={20} />
    </div>
  )
}

function LoadFailed({ title, error, retry }: { title: string; error: unknown; retry: () => void }) {
  const t = useT()
  return (
    <div className="pad">
      <ErrorPanel title={title} actions={<Button onClick={retry}>{t(common.retry)}</Button>}>
        {errorMessage(error)} {t(m.nothingChanged)}
      </ErrorPanel>
    </div>
  )
}

/** Who the entries are, each once, in the order listed: "Apple Mail Privacy Protection, Microsoft Defender and Mimecast". */
function names(items: { name: string }[]): string {
  return andList([...new Set(items.map((i) => i.name).filter(Boolean))])
}

// ── Networks ─────────────────────────────────────────────────────────────

function NetworksCard() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/admin/tracking-networks')
  const [adding, setAdding] = useState(false)
  const [removing, setRemoving] = useState<Network>()
  const kind = (c: Network['class']) => (c === 'privacy' ? <Badge tone="info">{t(m.netPrivacy)}</Badge> : <Badge tone="neutral">{t(m.netScanner)}</Badge>)
  const items = q.data?.items
  const admin = items?.filter((n) => n.source === 'admin') ?? []
  const builtin = items?.filter((n) => n.source === 'builtin') ?? []
  return (
    <section className="card" aria-labelledby="bf-nets-h">
      <CardSection
        title={<span id="bf-nets-h">{items ? t(m.networksTitleCount, { count: count(items.length) }) : t(m.networksTitle)}</span>}
        caption={t(m.networksCaption)}
        actions={
          <Button size="sm" variant="primary" onClick={() => setAdding(true)}>
            {t(m.addNetwork)}
          </Button>
        }
      />
      {q.isError ? (
        <LoadFailed title={t(m.networksLoadFailed)} error={q.error} retry={() => void q.refetch()} />
      ) : q.isPending ? (
        <Loading label={t(m.networksLoading)} />
      ) : (
        <>
          {admin.length ? (
            <AccessTable caption={t(m.networksTable)}>
              <thead>
                <tr>
                  <Th>{t(m.colNetwork)}</Th>
                  <Th>{t(m.colRunBy)}</Th>
                  <Th>{t(m.colKind)}</Th>
                  <Th>{t(m.colAdded)}</Th>
                  <Th>
                    <span className="vh">{t(m.colActions)}</span>
                  </Th>
                </tr>
              </thead>
              <tbody>
                {admin.map((n) => (
                  <tr key={n.cidr}>
                    <Td>
                      <span className="mono">{n.cidr}</span>
                    </Td>
                    <Td>
                      <Named name={n.name} note={n.note} />
                    </Td>
                    <Td>{kind(n.class)}</Td>
                    <Td nowrap>
                      <AddedBy name={n.createdByName} at={n.createdAt} />
                    </Td>
                    <Td>
                      <Button size="sm" aria-label={t(m.removeLabel, { entry: n.cidr })} onClick={() => setRemoving(n)}>
                        {t(m.remove)}
                      </Button>
                    </Td>
                  </tr>
                ))}
              </tbody>
            </AccessTable>
          ) : (
            <div className="pad">
              <span className="k">{t(m.networksNoneAdded)}</span>
            </div>
          )}
          {builtin.length ? (
            <Builtin summary={t(m.networksBuiltin, { count: builtin.length })} note={names(builtin)}>
              <AccessTable caption={t(m.networksBuiltinTable)}>
                <thead>
                  <tr>
                    <Th>{t(m.colNetwork)}</Th>
                    <Th>{t(m.colRunBy)}</Th>
                    <Th>{t(m.colKind)}</Th>
                  </tr>
                </thead>
                <tbody>
                  {builtin.map((n) => (
                    <tr key={n.cidr}>
                      <Td>
                        <span className="mono">{n.cidr}</span>
                      </Td>
                      <Td>{n.name}</Td>
                      <Td>{kind(n.class)}</Td>
                    </tr>
                  ))}
                </tbody>
              </AccessTable>
            </Builtin>
          ) : null}
        </>
      )}
      {adding ? <AddNetworkDialog onClose={() => setAdding(false)} /> : null}
      {removing ? <RemoveNetworkDialog n={removing} onClose={() => setRemoving(undefined)} /> : null}
    </section>
  )
}

/** A form error: on its field when the problem names one, else under the form. */
type FormError = { field?: string; message: string }

function formError(e: unknown): FormError {
  const loc = e instanceof ApiError ? e.problem.errors?.[0]?.location : undefined
  return { field: loc?.replace(/^body\./, ''), message: errorMessage(e) }
}

/** The error on a field, when the problem names it. */
const on = (error: FormError | undefined, field: string) => (error?.field === field ? error.message : undefined)

/** The error under the form, when the problem names none of its fields. */
function FormProblem({ error, fields }: { error?: FormError; fields: string[] }) {
  if (!error || fields.includes(error.field ?? '')) return null
  return (
    <span className="err" role="alert">
      {error.message}
    </span>
  )
}

function NoteField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const t = useT()
  return (
    <Field label={t(m.noteLabel)} optional>
      <textarea className="in h-auto py-2" rows={2} maxLength={300} placeholder={t(m.notePlaceholder)} value={value} onChange={(e) => onChange(e.target.value)} />
    </Field>
  )
}

/** One choice of a dialog's kind (InstanceBotFilterDialogs: the bordered radio rows). */
function Choice({ name, checked, onChange, title, hint }: { name: string; checked: boolean; onChange: () => void; title: string; hint: string }) {
  return (
    // eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its control; the title and hint inside are its text
    <label className="row flex-nowrap gap-2.5 rounded-lg border border-line px-2.5 py-2">
      <input type="radio" name={name} checked={checked} onChange={onChange} />
      <span className="col gap-px">
        <span className="font-semibold">{title}</span>
        <span className="k whitespace-normal">{hint}</span>
      </span>
    </label>
  )
}

function AddNetworkDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const id = useId()
  const qc = useQueryClient()
  const add = $api.useMutation('post', '/api/v1/admin/tracking-networks')
  const [cidr, setCidr] = useState('')
  const [name, setName] = useState('')
  const [cls, setCls] = useState<Network['class']>('scanner')
  const [note, setNote] = useState('')
  const [error, setError] = useState<FormError>()
  const blocked = !cidr.trim() ? t(m.cidrFirst) : !name.trim() ? t(m.nameFirst) : undefined
  const submit = async () => {
    if (blocked) return
    setError(undefined)
    try {
      const n = await add.mutateAsync({ body: { cidr: cidr.trim(), name: name.trim(), class: cls, note: note.trim() || undefined } })
      await qc.invalidateQueries({ queryKey: networksKey })
      toast({ kind: 'success', title: t(m.networkAdded, { cidr: n.cidr }) })
      onClose()
    } catch (e) {
      setError(formError(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !add.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">{t(m.dialogEyebrow)}</span>}
      title={t(m.addNetworkTitle)}
      description={<span className="text-[13.5px]">{t(m.addNetworkDescription)}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={add.isPending}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" loading={add.isPending} disabledReason={blocked} onClick={() => void submit()}>
            {t(m.addNetwork)}
          </Button>
        </>
      }
    >
      <Field label={t(m.cidrLabel)} hint={t(m.cidrHint)} error={on(error, 'cidr')}>
        <input
          className="in mono"
          value={cidr}
          maxLength={64}
          placeholder="203.0.113.0/24"
          onChange={(e) => setCidr(e.target.value)}
          autoFocus
          spellCheck={false}
          autoComplete="off"
        />
      </Field>
      <Field label={t(m.runByLabel)} error={on(error, 'name')}>
        <input className="in" value={name} maxLength={80} placeholder={t(m.runByPlaceholder)} onChange={(e) => setName(e.target.value)} />
      </Field>
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="mb-1.5 p-0 text-[12px] font-semibold text-ink-2">{t(m.kindLegend)}</legend>
        <Choice name={`${id}-kind`} checked={cls === 'scanner'} onChange={() => setCls('scanner')} title={t(m.netScanner)} hint={t(m.netScannerHint)} />
        <Choice name={`${id}-kind`} checked={cls === 'privacy'} onChange={() => setCls('privacy')} title={t(m.netPrivacy)} hint={t(m.netPrivacyHint)} />
      </fieldset>
      <NoteField value={note} onChange={setNote} />
      <FormProblem error={error} fields={['cidr', 'name']} />
    </Dialog>
  )
}

function RemoveNetworkDialog({ n, onClose }: { n: Network; onClose: () => void }) {
  const t = useT()
  const qc = useQueryClient()
  const del = $api.useMutation('delete', '/api/v1/admin/tracking-networks')
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="bad"
      badge={t(m.removeBadge)}
      title={t(m.removeNetworkTitle, { cidr: n.cidr })}
      description={t(m.removeNetworkDescription)}
      confirmLabel={t(m.remove)}
      onConfirm={async () => {
        await del.mutateAsync({ params: { query: { cidr: n.cidr } } })
        await qc.invalidateQueries({ queryKey: networksKey })
        toast({ kind: 'deleted', title: t(m.networkRemoved, { cidr: n.cidr }) })
      }}
    >
      <DialogRow first tone="neutral" label={t(m.addedBy)}>
        {t(m.addedByLine, { hasName: n.createdByName ? 'yes' : 'no', name: n.createdByName ?? '', date: usDate(n.createdAt), what: n.name })}
      </DialogRow>
    </ConfirmDialog>
  )
}

// ── User agents ──────────────────────────────────────────────────────────

const agentKinds: Record<Agent['kind'], { tone: Tone; label: typeof m.kindScanner }> = {
  scanner: { tone: 'neutral', label: m.kindScanner },
  script: { tone: 'neutral', label: m.kindScript },
  headless: { tone: 'neutral', label: m.kindHeadless },
  preview: { tone: 'neutral', label: m.kindPreview },
  person: { tone: 'ok', label: m.kindPerson },
}

function AgentsCard() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/admin/tracking-user-agents')
  const [adding, setAdding] = useState(false)
  const [removing, setRemoving] = useState<Agent>()
  const kind = (k: Agent['kind']) => <Badge tone={agentKinds[k].tone}>{t(agentKinds[k].label)}</Badge>
  const items = q.data?.items
  const admin = items?.filter((a) => a.source === 'admin') ?? []
  const builtin = items?.filter((a) => a.source === 'builtin') ?? []
  const people = names(builtin.filter((a) => a.kind === 'person'))
  return (
    <section className="card" aria-labelledby="bf-uas-h">
      <CardSection
        title={<span id="bf-uas-h">{items ? t(m.agentsTitleCount, { count: count(items.length) }) : t(m.agentsTitle)}</span>}
        caption={t(m.agentsCaption)}
        actions={
          <Button size="sm" variant="primary" onClick={() => setAdding(true)}>
            {t(m.addAgent)}
          </Button>
        }
      />
      {q.isError ? (
        <LoadFailed title={t(m.agentsLoadFailed)} error={q.error} retry={() => void q.refetch()} />
      ) : q.isPending ? (
        <Loading label={t(m.agentsLoading)} />
      ) : (
        <>
          {admin.length ? (
            <AccessTable caption={t(m.agentsTable)}>
              <thead>
                <tr>
                  <Th>{t(m.colPattern)}</Th>
                  <Th>{t(m.colKind)}</Th>
                  <Th>{t(m.colWho)}</Th>
                  <Th>{t(m.colAdded)}</Th>
                  <Th>
                    <span className="vh">{t(m.colActions)}</span>
                  </Th>
                </tr>
              </thead>
              <tbody>
                {admin.map((a) => (
                  <tr key={a.pattern}>
                    <Td>
                      <span className="mono">{a.pattern}</span>
                    </Td>
                    <Td>{kind(a.kind)}</Td>
                    <Td>
                      <Named name={a.name} note={a.note} />
                    </Td>
                    <Td nowrap>
                      <AddedBy name={a.createdByName} at={a.createdAt} />
                    </Td>
                    <Td>
                      <Button size="sm" aria-label={t(m.removeLabel, { entry: a.pattern })} onClick={() => setRemoving(a)}>
                        {t(m.remove)}
                      </Button>
                    </Td>
                  </tr>
                ))}
              </tbody>
            </AccessTable>
          ) : (
            <div className="pad">
              <span className="k">{t(m.agentsNoneAdded)}</span>
            </div>
          )}
          {builtin.length ? (
            <Builtin summary={t(m.agentsBuiltin, { count: builtin.length })} note={people ? t(m.agentsBuiltinNote, { people }) : t(m.agentsBuiltinNoPeople)}>
              <AccessTable caption={t(m.agentsBuiltinTable)}>
                <thead>
                  <tr>
                    <Th>{t(m.colPattern)}</Th>
                    <Th>{t(m.colKind)}</Th>
                    <Th>{t(m.colWho)}</Th>
                  </tr>
                </thead>
                <tbody>
                  {builtin.map((a) => (
                    <tr key={a.pattern}>
                      <Td>
                        <span className="mono">{a.pattern}</span>
                      </Td>
                      <Td>{kind(a.kind)}</Td>
                      <Td>{a.name || <span className="k">{t(m.noName)}</span>}</Td>
                    </tr>
                  ))}
                </tbody>
              </AccessTable>
            </Builtin>
          ) : null}
        </>
      )}
      {adding ? <AddAgentDialog onClose={() => setAdding(false)} /> : null}
      {removing ? <RemoveAgentDialog a={removing} onClose={() => setRemoving(undefined)} /> : null}
    </section>
  )
}

function AddAgentDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const id = useId()
  const qc = useQueryClient()
  const add = $api.useMutation('post', '/api/v1/admin/tracking-user-agents')
  const [kind, setKind] = useState<'scanner' | 'person'>('scanner')
  const [pattern, setPattern] = useState('')
  const [name, setName] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState<FormError>()
  const blocked = pattern.trim().length < 3 ? t(m.patternFirst) : !name.trim() ? t(m.whoFirst) : undefined
  const submit = async () => {
    if (blocked) return
    setError(undefined)
    try {
      const a = await add.mutateAsync({ body: { pattern: pattern.trim(), kind, name: name.trim(), note: note.trim() || undefined } })
      await qc.invalidateQueries({ queryKey: agentsKey })
      toast({ kind: 'success', title: t(m.agentAdded, { pattern: a.pattern }) })
      onClose()
    } catch (e) {
      setError(formError(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !add.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">{t(m.dialogEyebrow)}</span>}
      title={t(m.addAgentTitle)}
      description={<span className="text-[13.5px]">{t(m.addAgentDescription)}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={add.isPending}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" loading={add.isPending} disabledReason={blocked} onClick={() => void submit()}>
            {t(m.addAgent)}
          </Button>
        </>
      }
    >
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="mb-1.5 p-0 text-[12px] font-semibold text-ink-2">{t(m.itIsLegend)}</legend>
        <Choice name={`${id}-kind`} checked={kind === 'scanner'} onChange={() => setKind('scanner')} title={t(m.aMachine)} hint={t(m.aMachineHint)} />
        <Choice name={`${id}-kind`} checked={kind === 'person'} onChange={() => setKind('person')} title={t(m.aPerson)} hint={t(m.aPersonHint)} />
      </fieldset>
      <Field label={t(m.patternLabel)} hint={on(error, 'pattern') ? undefined : t(m.patternHint)} error={on(error, 'pattern')}>
        <input
          className="in mono"
          value={pattern}
          maxLength={80}
          placeholder={kind === 'person' ? 'KioskBot' : 'AcmeGate-LinkCheck'}
          onChange={(e) => setPattern(e.target.value)}
          autoFocus
          spellCheck={false}
          autoComplete="off"
        />
      </Field>
      <Field label={t(m.whoLabel)} error={on(error, 'name')}>
        <input className="in" value={name} maxLength={80} placeholder={t(m.whoPlaceholder)} onChange={(e) => setName(e.target.value)} />
      </Field>
      <NoteField value={note} onChange={setNote} />
      <FormProblem error={error} fields={['pattern', 'name']} />
    </Dialog>
  )
}

function RemoveAgentDialog({ a, onClose }: { a: Agent; onClose: () => void }) {
  const t = useT()
  const qc = useQueryClient()
  const del = $api.useMutation('delete', '/api/v1/admin/tracking-user-agents')
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="bad"
      badge={t(m.removeBadge)}
      title={t(m.removeAgentTitle, { pattern: a.pattern })}
      description={a.kind === 'person' ? t(m.removePersonDescription) : t(m.removeAgentDescription)}
      confirmLabel={t(m.remove)}
      onConfirm={async () => {
        await del.mutateAsync({ params: { query: { pattern: a.pattern } } })
        await qc.invalidateQueries({ queryKey: agentsKey })
        toast({ kind: 'deleted', title: t(m.agentRemoved, { pattern: a.pattern }) })
      }}
    >
      <DialogRow first tone="neutral" label={t(m.addedBy)}>
        {t(m.addedByLine, { hasName: a.createdByName ? 'yes' : 'no', name: a.createdByName ?? '', date: usDate(a.createdAt), what: a.name })}
      </DialogRow>
    </ConfirmDialog>
  )
}

// ── How the filter decides ───────────────────────────────────────────────

function RulesCard() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card" aria-labelledby="bf-rules-h">
      <CardSection title={<span id="bf-rules-h">{t(m.rulesTitle)}</span>} caption={t(m.rulesCaption)} />
      <div className="pad">
        <dl className="kv m-0">
          <dt>{t(m.rulesOpens)}</dt>
          <dd>{t(m.rulesOpensBody)}</dd>
          <dt>{t(m.rulesClicks)}</dt>
          <dd>{t(m.rulesClicksBody)}</dd>
          <dt>{t(m.rulesWords)}</dt>
          <dd>{t(m.rulesWordsBody)}</dd>
          <dt>{t(m.rulesChanges)}</dt>
          <dd>
            <T
              msg={m.rulesChangesBody}
              values={{
                a: (c) => (
                  <Link to="/w/$workspace/admin/audit" params={{ workspace: slug }}>
                    {c}
                  </Link>
                ),
              }}
            />
          </dd>
        </dl>
      </div>
    </section>
  )
}
