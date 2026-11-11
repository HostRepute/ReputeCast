import { useState, type KeyboardEvent, type ReactNode } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { AttentionIcon } from '@/features/integrations/shared'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { leadsBase, useFormDetail, useRefreshLeads, type FormDetail, type LeadForm } from './api'
import { ml, mm } from './messages'
import './leads.css'

/*
 * Map a lead form (LeadAdsFormMapping): each question to a contact field
 * or a custom field (or not kept), the lists leads join, tags, and double
 * or single opt-in. Single opt-in needs one of the form's consent
 * checkboxes: its text, as Meta returns it, is the consent people gave.
 */

const standard: [string, MessageDescriptor][] = [
  ['email', mm.targetEmail],
  ['first_name', mm.targetFirstName],
  ['last_name', mm.targetLastName],
  ['full_name', mm.targetFullName],
  ['company', mm.targetCompany],
  ['phone', mm.targetPhone],
]

const mono = (c: ReactNode[]) => <span className="mono text-[12px]">{c}</span>

/** The dialog's starting values: the form's mapping, or what ReputeCast suggests for a form not mapped yet. */
function initial(d: FormDetail) {
  const mapped = d.form.mapped
  return {
    targets: Object.fromEntries(d.questions.map((q) => [q.key, mapped ? q.target : q.suggested])) as Record<string, string>,
    lists: new Set(d.listIds),
    tags: [...d.form.tags],
    mode: d.checkboxes.length ? d.form.consentMode : ('double' as const),
    consentKey: d.consentKey || d.checkboxes[0]?.key || '',
  }
}

export function MappingDialog({ form, onOpenChange }: { form: LeadForm; onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const q = useFormDetail(form.id)
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(mm.title, { name: form.name })}
      eyebrow={<span className="k">{t(ml.eyebrow)}</span>}
      description={t(mm.sub, { page: form.page, count: q.data?.questions.length ?? form.questions })}
      size="lg"
      className="max-w-[720px]"
      hideClose
    >
      {q.isError ? (
        <>
          <div className="note b" role="alert">
            <AttentionIcon size={16} color="var(--bad-fg)" />
            <div className="min-w-0 flex-[1_1_320px]">{errorMessage(q.error) || t(mm.loadError)}</div>
          </div>
          <DialogFooter>
            <Button onClick={() => onOpenChange(false)}>{t(mm.cancel)}</Button>
            <Button variant="primary" onClick={() => void q.refetch()}>
              {t(mm.tryAgain)}
            </Button>
          </DialogFooter>
        </>
      ) : !q.data ? (
        <div className="flex flex-col gap-3" aria-busy="true">
          <Skeleton w={120} h={12} />
          <Skeleton w="100%" h={36} />
          <Skeleton w="100%" h={36} />
          <Skeleton w="100%" h={36} />
        </div>
      ) : (
        <MappingForm detail={q.data} onClose={() => onOpenChange(false)} />
      )}
    </Dialog>
  )
}

/** The dialog's footer: a divider across the dialog, actions on the right. */
function DialogFooter({ children }: { children: ReactNode }) {
  return <div className="-mx-[22px] -mb-4 mt-1 flex flex-wrap items-center justify-end gap-2.5 border-t border-line-2 px-[22px] pt-3.5 pb-[18px]">{children}</div>
}

function MappingForm({ detail: d, onClose }: { detail: FormDetail; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshLeads()
  const [v, setV] = useState(() => initial(d))
  const [error, setError] = useState<string>()
  const [tag, setTag] = useState('')
  const m = $api.useMutation('put', `${leadsBase}/forms/{formId}/mapping`)
  const emails = d.questions.filter((q) => v.targets[q.key] === 'email').length
  const noCheckbox = d.checkboxes.length === 0
  const checkbox = d.checkboxes.find((c) => c.key === v.consentKey) ?? d.checkboxes[0]
  const set = (patch: Partial<typeof v>) => setV((x) => ({ ...x, ...patch }))
  const addTag = () => {
    const x = tag.trim().toLowerCase()
    if (x && !v.tags.includes(x) && v.tags.length < 10) set({ tags: [...v.tags, x] })
    setTag('')
  }
  const onTagKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addTag()
    } else if (e.key === 'Backspace' && !tag && v.tags.length) {
      set({ tags: v.tags.slice(0, -1) })
    }
  }
  const submit = () => {
    if (emails === 0) return setError(t(mm.errEmail))
    if (emails > 1) return setError(t(mm.errEmailTwice))
    if (v.lists.size === 0) return setError(t(mm.errLists))
    setError(undefined)
    const mapping = Object.fromEntries(Object.entries(v.targets).filter(([, target]) => target)) as Record<string, string>
    const tags = tag.trim() && !v.tags.includes(tag.trim().toLowerCase()) ? [...v.tags, tag.trim().toLowerCase()] : v.tags
    m.mutate(
      {
        params: { path: { workspace: slug, formId: d.form.id } },
        body: { mapping, listIds: [...v.lists], tags, consentMode: v.mode, consentKey: v.mode === 'single' ? checkbox?.key : undefined },
      },
      {
        onSuccess: () => {
          toast({ kind: 'success', title: t(mm.saved, { name: d.form.name }) })
          void refresh()
          onClose()
        },
        onError: (e) => setError(e instanceof ApiError && e.problem.errors?.[0]?.message ? e.problem.errors[0].message : errorMessage(e)),
      },
    )
  }
  const emailMissing = !!error && emails === 0
  return (
    <form
      className="contents"
      onSubmit={(e) => {
        e.preventDefault()
        submit()
      }}
    >
      <div className="flex flex-col gap-5">
        <section className="la-sect" aria-labelledby="la-m-q">
          <h3 id="la-m-q">{t(mm.questions)}</h3>
          <div>
            <div className="la-qrow qh" aria-hidden="true">
              <span className="k">{t(mm.colQuestion)}</span>
              <span className="k">{t(mm.colSaveAs)}</span>
            </div>
            {d.questions.map((q) => {
              const isEmail = q.type.toUpperCase() === 'EMAIL' || q.type.toUpperCase() === 'WORK_EMAIL'
              const value = v.targets[q.key] ?? ''
              return (
                <div key={q.key} className="la-qrow">
                  <div className="flex min-w-0 flex-col gap-0.5">
                    <span className="text-[13.5px] font-semibold">
                      {q.label}{' '}
                      {isEmail ? (
                        <Badge tone="neutral" className="h-[18px] px-1.5 text-[10.5px]">
                          {t(mm.required)}
                        </Badge>
                      ) : null}
                    </span>
                    <span className="k">
                      <span className="mono text-[12px]">{q.key}</span> · {q.type}
                    </span>
                  </div>
                  <select
                    className="in"
                    aria-label={t(mm.saveAs, { label: q.label })}
                    value={value}
                    disabled={m.isPending}
                    aria-invalid={(isEmail && emailMissing) || undefined}
                    style={isEmail && emailMissing ? { borderColor: 'var(--bad-strong)' } : undefined}
                    onChange={(e) => set({ targets: { ...v.targets, [q.key]: e.target.value } })}
                  >
                    {standard.map(([k, msg]) => (
                      <option key={k} value={k}>
                        {t(msg)}
                      </option>
                    ))}
                    {d.fields.length ? (
                      <optgroup label={t(mm.customFields)}>
                        {d.fields.map((f) => (
                          <option key={f.key} value={`field:${f.key}`}>
                            {f.label}
                          </option>
                        ))}
                      </optgroup>
                    ) : null}
                    <option value="">{t(mm.dontKeep)}</option>
                  </select>
                </div>
              )
            })}
          </div>
        </section>
        <section className="la-sect" aria-labelledby="la-m-l">
          <h3 id="la-m-l">{t(mm.lists)}</h3>
          <div className="la-chips" role="group" aria-labelledby="la-m-l" aria-describedby="la-m-lk">
            {d.lists.map((l) => {
              const on = v.lists.has(l.id)
              return (
                <label key={l.id} className={`chip${on ? ' on' : ''}`}>
                  <input
                    type="checkbox"
                    checked={on}
                    disabled={m.isPending}
                    onChange={(e) => {
                      const next = new Set(v.lists)
                      if (e.target.checked) next.add(l.id)
                      else next.delete(l.id)
                      set({ lists: next })
                    }}
                  />
                  {l.name}
                </label>
              )
            })}
          </div>
          <span className="k" id="la-m-lk">
            {t(mm.listsHelp)}
          </span>
        </section>
        <section className="la-sect" aria-labelledby="la-m-t">
          <h3 id="la-m-t">{t(mm.tags)}</h3>
          <div className="in la-tags">
            {v.tags.map((x) => (
              <span key={x} className="chip">
                {x}
                <button type="button" aria-label={t(mm.removeTag, { tag: x })} onClick={() => set({ tags: v.tags.filter((y) => y !== x) })}>
                  <Icon name="close" size={11} strokeWidth={2.2} />
                </button>
              </span>
            ))}
            <input type="text" aria-label={t(mm.addTag)} aria-describedby="la-m-th" value={tag} maxLength={60} onChange={(e) => setTag(e.target.value)} onKeyDown={onTagKey} onBlur={addTag} />
          </div>
          <span className="k" id="la-m-th">
            {t(mm.tagsHelp)}
          </span>
        </section>
        <section className="la-sect" aria-labelledby="la-m-c">
          <h3 id="la-m-c">{t(mm.consent)}</h3>
          <div role="radiogroup" aria-labelledby="la-m-c" className="flex flex-col gap-2">
            <div className={`la-radio${v.mode === 'double' ? ' on' : ''}`}>
              <input id="la-m-double" type="radio" name="la-consent" checked={v.mode === 'double'} disabled={m.isPending} onChange={() => set({ mode: 'double' })} />
              <label htmlFor="la-m-double" className="la-rt flex min-w-0 flex-1 cursor-pointer flex-col gap-0.5">
                <b className="font-semibold">{t(mm.double)}</b>
                <span className="k">{t(mm.doubleSub)}</span>
              </label>
            </div>
            <div className={`la-radio${v.mode === 'single' ? ' on' : ''}${noCheckbox ? ' off' : ''}`}>
              <input
                id="la-m-single"
                type="radio"
                name="la-consent"
                checked={v.mode === 'single'}
                disabled={noCheckbox || m.isPending}
                aria-describedby={noCheckbox ? 'la-m-nc' : undefined}
                onChange={() => set({ mode: 'single' })}
              />
              <span className="flex min-w-0 flex-1 flex-col gap-2">
                <label htmlFor="la-m-single" className={`la-rt flex flex-col gap-0.5${noCheckbox ? '' : ' cursor-pointer'}`}>
                  <b className="font-semibold">{t(mm.single)}</b>
                  <span className="k">{t(mm.singleSub)}</span>
                </label>
                {noCheckbox ? (
                  <span className="k text-ink-nav" id="la-m-nc">
                    {t(mm.noCheckbox)}
                  </span>
                ) : v.mode === 'single' && checkbox ? (
                  <>
                    {d.checkboxes.length > 1 ? (
                      <label className="lbl flex flex-col gap-1">
                        {t(mm.checkbox)}
                        <select className="in" value={checkbox.key} onChange={(e) => set({ consentKey: e.target.value })}>
                          {d.checkboxes.map((c) => (
                            <option key={c.key} value={c.key}>
                              {c.key}
                            </option>
                          ))}
                        </select>
                      </label>
                    ) : null}
                    <span className="la-sbox">
                      <span className="k">
                        <T msg={mm.checkboxKey} values={{ key: checkbox.key, mono }} />
                      </span>
                      <blockquote>{t(mm.quoted, { text: checkbox.text })}</blockquote>
                    </span>
                    <span className="k text-ink-nav">{t(mm.unticked)}</span>
                  </>
                ) : null}
              </span>
            </div>
          </div>
        </section>
      </div>
      <DialogFooter>
        {error ? (
          <span role="alert" className="inline-flex flex-[1_1_100%] items-center gap-2 text-[13px] font-semibold text-bad-fg">
            <AttentionIcon size={14} color="var(--bad-fg)" />
            {error}
          </span>
        ) : null}
        <Button onClick={onClose} disabled={m.isPending}>
          {t(mm.cancel)}
        </Button>
        <Button type="submit" variant="primary" loading={m.isPending}>
          {t(mm.save)}
        </Button>
      </DialogFooter>
    </form>
  )
}
