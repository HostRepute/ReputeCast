import { Link } from '@tanstack/react-router'
import { Popover as RPopover } from 'radix-ui'
import { useMemo, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { timezoneOptions } from '@/features/workspace/timezones'
import { formatList, useT } from '@/i18n'
import { getDisplayTimeZone } from '@/lib/format'
import { useRefreshAgency } from './api'
import { m } from './messages'
import { dayMonth, hhmm } from './parts'

type Template = Schemas['AgencyTemplate']
type Created = Schemas['AgencyClientCreated']
type Plan = Schemas['AgencyTemplatePlan']

function slugify(name: string) {
  return name
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 63)
}

/** What a template sets up, in the lines of its preview. */
export function templateLines(t: ReturnType<typeof useT>, tpl: Template) {
  const s = tpl.setsUp
  const lines: string[] = []
  if (s.lists) lines.push(t(m.tplLists, { n: s.lists, names: formatList(s.listNames ?? []) }))
  if (s.automations) lines.push(t(m.tplAutomations, { n: s.automations, names: formatList(s.automationNames ?? []) }))
  if (s.sections && s.templates) lines.push(t(m.tplSectionsTemplates, { sections: s.sections, templates: s.templates }))
  else if (s.sections) lines.push(t(m.tplSections, { n: s.sections }))
  else if (s.templates) lines.push(t(m.tplTemplates, { n: s.templates }))
  if (s.smsTexts) lines.push(t(m.tplSms, { n: s.smsTexts }))
  if (s.forms) lines.push(t(m.tplForms, { n: s.forms }))
  lines.push(s.clientSignoff ? t(m.tplPolicyRoles) : t(m.tplRoles))
  return lines
}

/** Agency › New client (AgencyStates): a client workspace from a template, then what it added. */
export function NewClientDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const templates = $api.useQuery('get', '/api/v1/agency/templates')
  const staff = $api.useQuery('get', '/api/v1/agency/staff')
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [slugTouched, setSlugTouched] = useState(false)
  const [tz, setTz] = useState(getDisplayTimeZone() ?? 'UTC')
  const [tplID, setTplID] = useState<string>()
  const [owner, setOwner] = useState<string>()
  const [managers, setManagers] = useState<string[]>([])
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<'client_approver' | 'client_viewer'>('client_approver')
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [created, setCreated] = useState<Created>()
  const tzs = useMemo(() => timezoneOptions(tz), [tz])
  const list = templates.data ?? []
  const tpl = list.find((x) => x.id === (tplID ?? list.find((y) => y.default)?.id ?? ''))
  const people = (staff.data?.staff ?? []).filter((s) => !s.invited && s.userId)
  const owners = people.filter((p) => p.level === 'agency_manager')
  const ownerID = owner ?? owners.find((o) => o.instanceAdmin)?.userId ?? owners[0]?.userId ?? ''
  const submit = async () => {
    setBusy(true)
    setErrors({})
    try {
      const { data } = await http.POST('/api/v1/agency/clients', {
        body: {
          name: name.trim(),
          slug: slug.trim() || slugify(name),
          timezone: tz,
          templateId: tpl?.id,
          ownerId: ownerID,
          accountManagerIds: managers,
          firstPerson: email.trim() ? { email: email.trim(), role } : undefined,
        },
      })
      await refresh()
      setCreated(data!)
      toast({ kind: 'success', title: t(m.clientCreated, { name: name.trim() }) })
    } catch (e) {
      const problem = (e as { problem?: { errors?: { location?: string; message?: string }[] } }).problem
      const byField: Record<string, string> = {}
      for (const er of problem?.errors ?? []) byField[(er.location ?? '').replace('body.', '')] = er.message ?? ''
      setErrors(byField)
      if (!Object.keys(byField).length) toast({ kind: 'error', title: t(m.clientCreateFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  if (created) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange} size="lg" title={t(m.readyTitle, { name: created.workspace.name })} hideClose>
        <TemplateApplied created={created} onClose={() => onOpenChange(false)} />
      </Dialog>
    )
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.newClientCrumb)}</span>}
      title={t(m.newClient)}
      size="md"
      footer={
        <>
          <span className="k mr-auto">{t(m.recordedInAudit)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!name.trim() || !ownerID} onClick={submit}>
            {t(m.createClient)}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
        <Field label={t(m.fieldName)} error={errors.name}>
          <Input
            value={name}
            onChange={(e) => {
              setName(e.target.value)
              if (!slugTouched) setSlug(slugify(e.target.value))
            }}
            autoFocus
          />
        </Field>
        <Field label={t(m.fieldSlug)} error={errors.slug}>
          <Input
            value={slug}
            onChange={(e) => {
              setSlugTouched(true)
              setSlug(e.target.value)
            }}
          />
        </Field>
        <Field label={t(m.fieldTimezone)} error={errors.timezone}>
          <Select value={tz} onChange={(e) => setTz(e.target.value)}>
            {tzs.map((z) => (
              <option key={z.value} value={z.value}>
                {z.value}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t(m.fieldTemplate)} error={errors.templateId}>
          <Select value={tpl?.id ?? ''} onChange={(e) => setTplID(e.target.value)}>
            <option value="">{t(m.noTemplate)}</option>
            {list.map((x) => (
              <option key={x.id} value={x.id}>
                {x.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t(m.fieldOwner)} hint={t(m.ownerHint)} error={errors.ownerId}>
          <Select value={ownerID} onChange={(e) => setOwner(e.target.value)}>
            {owners.map((o) => (
              <option key={o.userId} value={o.userId}>
                {t(m.ownerOption, { name: o.name })}
              </option>
            ))}
          </Select>
        </Field>
        <div className="lbl" role="group" aria-labelledby="nc-managers">
          <span id="nc-managers">{t(m.fieldManagers)}</span>
          <div className="flex flex-wrap items-center gap-1.5">
            {managers.map((id) => {
              const p = people.find((x) => x.userId === id)
              if (!p) return null
              return (
                <span key={id} className="chip">
                  <Avatar name={p.name} color={p.avatarColor} size={18} fontSize={9} />
                  {p.name}
                  <button type="button" aria-label={t(m.removeManager, { name: p.name })} onClick={() => setManagers(managers.filter((x) => x !== id))}>
                    <Icon name="close" size={12} />
                  </button>
                </span>
              )
            })}
            <RPopover.Root>
              <RPopover.Trigger asChild>
                <button type="button" className="chip">
                  <Icon name="plus" size={12} />
                  {t(m.addManager)}
                </button>
              </RPopover.Trigger>
              <RPopover.Portal>
                <RPopover.Content className="menu z-[70] w-[260px]" align="start" sideOffset={6}>
                  {people
                    .filter((p) => !managers.includes(p.userId!))
                    .map((p) => (
                      <RPopover.Close asChild key={p.id}>
                        <button type="button" onClick={() => setManagers([...managers, p.userId!])}>
                          <Avatar name={p.name} color={p.avatarColor} size={22} />
                          {p.name}
                        </button>
                      </RPopover.Close>
                    ))}
                </RPopover.Content>
              </RPopover.Portal>
            </RPopover.Root>
          </div>
          {errors.accountManagerIds ? (
            <span className="err" role="alert">
              {errors.accountManagerIds}
            </span>
          ) : null}
        </div>
      </div>
      {tpl ? (
        <div className="flex flex-col gap-1.5 rounded-lg border border-line bg-[#F9FAFB] px-3.5 py-3">
          <b className="text-[13.5px]">{t(m.tplAdds, { name: tpl.name })}</b>
          {templateLines(t, tpl).map((l) => (
            <span key={l} className="flex items-start gap-2 text-[13px]">
              <Icon name="plus" size={13} className="mt-0.5 flex-none text-accent" />
              {l}
            </span>
          ))}
          <span className="k mt-0.5">{t(m.neverCopiedShort)}</span>
        </div>
      ) : null}
      <div className="flex gap-2.5 max-md:flex-col">
        <Field label={t(m.firstPerson)} error={errors['firstPerson.email'] ?? errors['firstPerson.role']} className="flex-1">
          <Input value={email} onChange={(e) => setEmail(e.target.value)} placeholder={t(m.firstPersonPlaceholder, { domain: slugify(name).replace(/-/g, '') || 'client' })} type="email" />
        </Field>
        <Select value={role} onChange={(e) => setRole(e.target.value as typeof role)} aria-label={t(m.firstPersonRole)} className="mt-[22px] w-[220px] max-md:mt-0 max-md:w-full">
          <option value="client_approver">{t(m.roleClientApprover)}</option>
          <option value="client_viewer">{t(m.roleClientViewer)}</option>
        </Select>
      </div>
    </Dialog>
  )
}

const nextStepLinks: Record<string, { to: string; label: typeof m.connect; primary?: boolean }> = {
  provider: { to: '/w/$workspace/providers/connect', label: m.connect, primary: true },
  domain: { to: '/w/$workspace/settings/senders', label: m.addDomain },
  invite: { to: '/w/$workspace/settings/team', label: m.invite },
  shopify: { to: '/w/$workspace/settings/integrations', label: m.connect },
}

const stepLabels: Record<string, typeof m.stepProvider> = { provider: m.stepProvider, domain: m.stepDomain, invite: m.stepInvite, shopify: m.stepShopify }

/** What applying a template added, what it never copies, and the next steps (AgencyStates: Template applied). */
export function TemplateApplied({ created, onClose }: { created: Created; onClose: () => void }) {
  const t = useT()
  const plan = created.plan
  const slug = created.workspace.slug
  const setup = created.setup
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start gap-3 rounded-lg border border-[#BBF7D0] bg-[#F0FDF4] px-4 py-3">
        <Icon name="check" size={18} className="mt-0.5 text-[#15803D]" />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <b>{t(m.readyTitle, { name: created.workspace.name })}</b>
          <span className="k">
            {plan
              ? t(m.createdByFrom, { name: created.createdBy, date: dayMonth(created.createdAt), time: hhmm(created.createdAt), template: plan.template.name, done: setup?.done ?? 0, total: setup?.total ?? 0 })
              : t(m.createdBy, { name: created.createdBy, date: dayMonth(created.createdAt), time: hhmm(created.createdAt) })}
          </span>
        </span>
        <Link to="/w/$workspace" params={{ workspace: slug }} className="btn" onClick={onClose}>
          {t(m.openTheWorkspace)}
        </Link>
      </div>
      <div className="grid gap-5" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(220px,100%),1fr))' }}>
        {plan ? <PlanAdded plan={plan} /> : null}
        <div className="flex flex-col gap-2">
          <b className="text-[13.5px]">{t(m.notCopied)}</b>
          {[m.ncProviders, m.ncDomains, m.ncContacts, m.ncKeys, m.ncIntegrations].map((d) => (
            <span key={d.id} className="flex items-start gap-2 text-[13px]">
              <Icon name="close" size={13} className="mt-0.5 flex-none text-ink-3" />
              {t(d)}
            </span>
          ))}
        </div>
        {setup ? (
          <div className="flex flex-col">
            <b className="mb-1.5 text-[13.5px]">{t(m.nextSteps)}</b>
            {setup.steps.map((s, i) => {
              const link = nextStepLinks[s.key]
              return (
                <div key={s.key} className="flex items-center gap-2.5 border-t border-line-2 py-2 first-of-type:border-t-0">
                  <span className="inline-flex h-[22px] w-[22px] flex-none items-center justify-center rounded-full border border-[#D4D8DF] text-[11px] font-semibold text-ink-2">
                    {i + 1}
                  </span>
                  <span className="flex-1 text-[13px]">{t(stepLabels[s.key] ?? m.stepProvider)}</span>
                  {link ? (
                    <Link to={link.to as never} params={{ workspace: slug } as never} className={i === 0 ? 'btn sm pri' : 'btn sm'} onClick={onClose}>
                      {t(link.label)}
                    </Link>
                  ) : null}
                </div>
              )
            })}
          </div>
        ) : null}
      </div>
      <div className="flex justify-end">
        <Button onClick={onClose}>{t(m.close)}</Button>
      </div>
    </div>
  )
}

/** The plan's "Added" list (and renames). */
export function PlanAdded({ plan, title }: { plan: Plan; title?: string }) {
  const t = useT()
  const lines: string[] = []
  const lists = [...plan.lists.added, ...plan.lists.renamed.map((r) => r.to)]
  if (lists.length) lines.push(t(m.addedLists, { n: lists.length, names: formatList(lists) }))
  const autos = [...plan.automations.added, ...plan.automations.renamed.map((r) => r.to)]
  if (autos.length) lines.push(t(m.addedAutomations, { n: autos.length, names: formatList(autos) }))
  const sections = plan.sections.added.length + plan.sections.renamed.length
  if (sections) lines.push(t(m.addedSections, { n: sections }))
  const templates = plan.templates.added.length + plan.templates.renamed.length
  if (templates) lines.push(t(m.addedTemplates, { n: templates }))
  if (plan.clientSignoff === 'on') lines.push(t(m.addedPolicy))
  lines.push(t(m.addedRoles))
  return (
    <div className="flex flex-col gap-2">
      <b className="text-[13.5px]">{title ?? t(m.added)}</b>
      {lines.map((l) => (
        <span key={l} className="flex items-start gap-2 text-[13px]">
          <Icon name="check" size={13} className="mt-0.5 flex-none text-[#15803D]" />
          {l}
        </span>
      ))}
    </div>
  )
}
