import { useEffect, useMemo, useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, dayMonthYear } from '@/lib/format'
import { useComposer } from '../context'
import { diffLines } from '../diff'
import { formatMjml } from '../editor/mjmlDoc'
import { mp } from './messages'
import { useTx } from './state'
import type { Variable, Version } from './types'

/** Two published versions, or the saved draft against the edits a conflict would overwrite. */
export type CompareTarget = { a: number; b: number } | 'conflict'

interface Side {
  label: string
  text: string
}

/** A version as text for the line diff: subject, preheader and variables, then the MJML (or HTML). */
function describeVersion(t: ReturnType<typeof useT>, subject: string, preheader: string, variables: Variable[], body: string): string {
  const vars = variables.map((v) => t(v.sensitive ? mp.diffVariableSensitive : v.required ? mp.diffVariableRequired : mp.diffVariable, { name: v.name, type: v.type })).join(', ')
  return [t(mp.diffSubject, { value: subject }), t(mp.diffPreheader, { value: preheader }), t(mp.diffVariables, { value: vars }), '', body].join('\n')
}

function Compare({ target, onBack }: { target: CompareTarget; onBack: () => void }) {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const [sides, setSides] = useState<[Side, Side] | null>(null)
  // What a conflict compares is taken when the comparison opens.
  const [mine] = useState(() => describeVersion(t, c.subject, c.preheader, tx.variables, tx.htmlMode ? tx.html : formatMjml(c.currentMjml())))
  useEffect(() => {
    let cancelled = false
    const path = { workspace: c.slug, template: tx.template.id }
    const version = (v: number) =>
      http.GET('/api/v1/workspaces/{workspace}/transactional/templates/{template}/versions/{version}', { params: { path: { ...path, version: v } } }).then((r) => r.data!)
    const run = async (): Promise<[Side, Side]> => {
      if (target === 'conflict') {
        const { data } = await http.GET('/api/v1/workspaces/{workspace}/transactional/templates/{template}', { params: { path } })
        const d = data!
        return [
          { label: t(mp.theirDraft, { name: d.draftSavedBy, revision: d.draftRevision }), text: describeVersion(t, d.subject, d.preheader, d.variables, d.mjml ? formatMjml(d.mjml) : d.html) },
          { label: t(mp.yourEdits), text: mine },
        ]
      }
      const [x, y] = await Promise.all([version(target.a), version(target.b)])
      return [
        { label: t(mp.versionAt, { version: x.version, date: dayMonthYear(x.publishedAt) }), text: describeVersion(t, x.subject, x.preheader, x.variables, x.mjml ? formatMjml(x.mjml) : x.html) },
        { label: t(mp.versionAt, { version: y.version, date: dayMonthYear(y.publishedAt) }), text: describeVersion(t, y.subject, y.preheader, y.variables, y.mjml ? formatMjml(y.mjml) : y.html) },
      ]
    }
    run()
      .then((s) => !cancelled && setSides(s))
      .catch((e: unknown) => toast({ kind: 'error', title: t(mp.compareFailed), detail: errorMessage(e) }))
    return () => {
      cancelled = true
    }
  }, [target, c.slug, tx.template.id, mine, t])
  const lines = useMemo(() => (sides ? diffLines(sides[0].text, sides[1].text) : []), [sides])
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        {sides ? <span className="k">{t(mp.compareLine, { a: sides[0].label, b: sides[1].label })}</span> : <span className="k">{t(mp.loadingVersions)}</span>}
        {target === 'conflict' ? null : (
          <button type="button" className="txe-linklike" onClick={onBack}>
            {t(mp.backToVersions)}
          </button>
        )}
      </div>
      {sides ? (
        // A scrolling region is the exception to the rule: it must take focus.
        // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
        <div className="mono overflow-auto rounded-lg border border-line" style={{ maxHeight: '60vh', fontSize: 12, lineHeight: 1.6 }} tabIndex={0} role="region" aria-label={t(mp.compareLabel)}>
          {lines.map((l, i) => (
            <div key={i} style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', padding: '0 10px', background: l.kind === 'add' ? '#DCFCE7' : l.kind === 'del' ? '#FEE2E2' : undefined, color: l.kind === 'fold' ? 'var(--ink-3)' : undefined }}>
              <span aria-hidden="true" style={{ display: 'inline-block', width: 14 }}>
                {l.kind === 'add' ? '+' : l.kind === 'del' ? '−' : ' '}
              </span>
              <span className="vh">{l.kind === 'add' ? t(mp.added) : l.kind === 'del' ? t(mp.removed) : ''}</span>
              {l.text}
            </div>
          ))}
          {lines.every((l) => l.kind === 'same' || l.kind === 'fold') ? <p className="k p-3">{t(mp.noDifferences)}</p> : null}
        </div>
      ) : null}
    </div>
  )
}

function VersionList({ onCompare }: { onCompare: (a: number, b: number) => void }) {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/transactional/templates/{template}/versions', { params: { path: { workspace: c.slug, template: tx.template.id } } })
  const [again, setAgain] = useState<Version | null>(null)
  if (q.isPending) return <p className="k">{t(mp.loadingVersions)}</p>
  if (q.isError) return <p className="k">{t(mp.versionsFailed, { error: errorMessage(q.error) })}</p>
  const items = q.data.items
  const live = items.find((v) => v.live)
  if (!items.length) return <p className="k">{t(mp.noVersions)}</p>
  return (
    <>
      <Table caption={t(mp.versionsCaption)} density="compact">
        <thead>
          <tr>
            <Th>{t(mp.colVersion)}</Th>
            <Th>{t(mp.colBy)}</Th>
            <Th>{t(mp.colWhen)}</Th>
            <Th>{t(mp.colChange)}</Th>
            <Th>
              <span className="vh">{t(mp.colActions)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {items.map((v, i) => {
            const older = items[i + 1]
            return (
              <tr key={v.version}>
                <Td nowrap>{v.live ? <Badge tone="ok">{t(mp.liveBadge, { version: v.version })}</Badge> : <Badge tone="neutral">{t(mp.versionBadge, { version: v.version })}</Badge>}</Td>
                <Td nowrap>
                  <span className="inline-flex items-center gap-2">
                    <Avatar name={v.publishedBy} color={v.avatarColor} size={22} />
                    <span>{v.publishedBy}</span>
                  </span>
                </Td>
                <Td nowrap>{t(mp.when, { date: dayMonthYear(v.publishedAt), time: clock(v.publishedAt) })}</Td>
                <Td wrap>{v.change}</Td>
                <Td align="right" nowrap>
                  <span className="inline-flex gap-1.5">
                    {v.live ? (
                      older ? (
                        <Button size="sm" onClick={() => onCompare(older.version, v.version)}>
                          {t(mp.compareWith, { version: older.version })}
                        </Button>
                      ) : null
                    ) : (
                      <>
                        {live ? (
                          <Button size="sm" onClick={() => onCompare(v.version, live.version)}>
                            {t(mp.compare)}
                          </Button>
                        ) : null}
                        {tx.canPublish ? (
                          <Button size="sm" onClick={() => setAgain(v)}>
                            {t(mp.publishAgain)}
                          </Button>
                        ) : null}
                      </>
                    )}
                  </span>
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
      <ConfirmDialog
        open={again !== null}
        onOpenChange={(o) => !o && setAgain(null)}
        tone="warn"
        badge={t(mp.rollbackBadge)}
        title={t(mp.rollbackTitle, { version: again?.version ?? 0 })}
        description={live ? t(mp.rollbackBody, { version: again?.version ?? 0, live: live.version }) : t(mp.rollbackBodyNoLive, { version: again?.version ?? 0 })}
        confirmLabel={t(mp.rollbackConfirm, { version: again?.version ?? 0 })}
        confirmVariant="primary"
        cancelLabel={t(mp.cancel)}
        onConfirm={async () => {
          if (!again) return
          const { data } = await http.POST('/api/v1/workspaces/{workspace}/transactional/templates/{template}/rollback', {
            params: { path: { workspace: c.slug, template: tx.template.id } },
            body: { version: again.version },
          })
          if (data) tx.mergeTemplate(data.template)
          void q.refetch()
          toast({ kind: 'success', title: t(mp.rolledBack, { key: tx.template.key, version: again.version }) })
          setAgain(null)
        }}
      />
    </>
  )
}

/**
 * Versions (TransactionalPublishStates · Version history and rolling back):
 * who published what and when, Compare, and "Publish again". Also compares
 * the saved draft with the unsaved edits after a conflict.
 */
export function VersionsDialog({ target, onTarget }: { target: CompareTarget | 'list' | null; onTarget: (t: CompareTarget | 'list' | null) => void }) {
  const t = useT()
  const tx = useTx()
  const open = target !== null
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && onTarget(null)}
      size="lg"
      title={target === 'conflict' ? t(mp.compareConflictTitle) : t(mp.versionsTitle, { name: tx.template.name })}
      description={target === 'list' ? t(mp.versionsLead) : undefined}
    >
      {target === 'list' ? <VersionList onCompare={(a, b) => onTarget({ a, b })} /> : target ? <Compare key={JSON.stringify(target)} target={target} onBack={() => onTarget('list')} /> : null}
    </Dialog>
  )
}
