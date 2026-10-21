import { Link } from '@tanstack/react-router'
import { useRef, useState, type ReactNode } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorSummary, type FormError } from '@/components/ui/ErrorSummary'
import { Field, Input, OptionCard, Select, Textarea, Toggle } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, count } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import type { Milestone, MilestoneList } from './MilestonesPage'
import { typeLabel } from './MilestonesPage'
import { m } from './messages'
import { mm } from './milestones.messages'
import { copy, fileKind, fileSize, mono, orNames, uploadFile, useGrowRefresh } from './shared'

type RewardType = Milestone['type']
type Input = Schemas['ReferralMilestoneInput']

interface Form {
  count: string
  rewardName: string
  rewardDescription: string
  image?: { id: string; url: string; name: string; width: number; height: number; size: number }
  type: RewardType
  downloadMode: 'file' | 'url'
  fileId: string
  downloadUrl: string
  codesAlertBelow: string
  askAddress: boolean
  couponOfferId: string
  couponValidDays: 0 | 30 | 60 | 90
  webhookUrl: string
  /** A new signing secret, shown until saved. */
  webhookSecret: string
  emailOn: boolean
}

const TYPES: RewardType[] = ['download', 'codes', 'manual', 'coupon', 'webhook']

/** The next round milestone after the ones a program has: 1, 3, 10, 25, 50, 100… */
function nextCount(d: MilestoneList): number {
  const have = new Set(d.items.map((x) => x.count))
  for (const c of [1, 3, 10, 25, 50, 100, 250, 500]) if (!have.has(c)) return c
  return Math.max(...have) * 2
}

function formOf(x: Milestone | undefined, d: MilestoneList): Form {
  return {
    count: String(x?.count ?? nextCount(d)),
    rewardName: x?.rewardName ?? '',
    rewardDescription: x?.rewardDescription ?? '',
    image: x?.image ? { id: x.image.id, url: x.image.url, name: x.image.name, width: x.image.width, height: x.image.height, size: x.image.size } : undefined,
    type: x?.type ?? 'download',
    downloadMode: x && !x.file && x.downloadUrl ? 'url' : 'file',
    fileId: x?.file?.id ?? '',
    downloadUrl: x?.downloadUrl ?? '',
    codesAlertBelow: String(x?.codesAlertBelow || 20),
    askAddress: x?.askAddress ?? false,
    couponOfferId: x?.couponOfferId ?? d.stripe.offers[0]?.id ?? '',
    couponValidDays: (x?.couponValidDays as Form['couponValidDays']) ?? 60,
    webhookUrl: x?.webhookUrl ?? '',
    webhookSecret: '',
    emailOn: x?.email.on ?? true,
  }
}

/** A random signing secret, as other webhooks make them. */
function newSecret(): string {
  const b = new Uint8Array(24)
  crypto.getRandomValues(b)
  return 'whsec_' + Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('')
}

/**
 * The reward editor (ReferralMilestones, ReferralStates): a new milestone
 * starts with the type picker; then the count, name, description, image,
 * the type's settings and the milestone email.
 */
export function RewardEditor({ d, milestone, onClose }: { d: MilestoneList; milestone?: Milestone; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const [step, setStep] = useState<'type' | 'edit'>(milestone ? 'edit' : 'type')
  const [f, setF] = useState<Form>(() => formOf(milestone, d))
  const [errors, setErrors] = useState<FormError[]>([])
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const refresh = useGrowRefresh()
  const set = (p: Partial<Form>) => setF((cur) => ({ ...cur, ...p }))
  const n = Number(f.count) || 0
  const title = milestone
    ? canEdit
      ? t(mm.editTitle, { name: milestone.rewardName })
      : t(mm.viewTitle, { name: milestone.rewardName })
    : n
      ? t(mm.addTitle, { count: n })
      : t(mm.addTitleNew)

  const validate = (): FormError[] => {
    const out: FormError[] = []
    if (!(n >= 1 && n <= 100000)) out.push({ field: 'me-count', message: t(mm.errCount) })
    if (!f.rewardName.trim()) out.push({ field: 'me-name', message: t(mm.errName) })
    if (f.rewardDescription.length > 120) out.push({ field: 'me-desc', message: t(mm.errDescription) })
    if (f.type === 'download') {
      if (f.downloadMode === 'file' && !f.fileId) out.push({ field: 'me-file', message: t(mm.errFile) })
      if (f.downloadMode === 'url' && !/^https:\/\/\S+$/.test(f.downloadUrl.trim())) out.push({ field: 'me-url', message: t(mm.errUrl) })
    }
    if (f.type === 'coupon' && !f.couponOfferId) out.push({ field: 'me-offer', message: t(mm.errOffer) })
    if (f.type === 'webhook' && !/^https:\/\/\S+$/.test(f.webhookUrl.trim())) out.push({ field: 'me-endpoint', message: t(mm.errEndpoint) })
    return out
  }

  const input = (): Input => {
    const body: Input = {
      count: n,
      rewardName: f.rewardName.trim(),
      rewardDescription: f.rewardDescription.trim(),
      type: f.type,
      emailOn: f.emailOn,
      imageId: f.image?.id,
    }
    if (f.type === 'download') {
      if (f.downloadMode === 'file') body.fileId = f.fileId
      else body.downloadUrl = f.downloadUrl.trim()
    }
    if (f.type === 'codes') body.codesAlertBelow = Number(f.codesAlertBelow) || 0
    if (f.type === 'manual') body.askAddress = f.askAddress
    if (f.type === 'coupon') {
      body.couponOfferId = f.couponOfferId
      body.couponValidDays = f.couponValidDays
    }
    if (f.type === 'webhook') {
      body.webhookUrl = f.webhookUrl.trim()
      if (f.webhookSecret) body.webhookSecret = f.webhookSecret
    }
    return body
  }

  const save = async () => {
    const bad = validate()
    setErrors(bad)
    if (bad.length) return
    setSaving(true)
    try {
      if (milestone) {
        await http.PUT('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}', { params: { path: { workspace: slug, milestone: milestone.id } }, body: input() })
        toast({ kind: 'success', title: t(mm.saved), detail: t(mm.savedDetail, { count: n, reward: f.rewardName.trim() }) })
      } else {
        await http.POST('/api/v1/workspaces/{workspace}/grow/referrals/{program}/milestones', { params: { path: { workspace: slug, program: d.programId } }, body: input() })
        toast({ kind: 'success', title: t(mm.added), detail: t(mm.savedDetail, { count: n, reward: f.rewardName.trim() }) })
      }
      await refresh()
      onClose()
    } catch (e) {
      const fieldErrors = e instanceof ApiError ? (e.problem.errors ?? []) : []
      if (fieldErrors.length) {
        setErrors(fieldErrors.map((x) => ({ field: fieldOf(x.location), message: x.message ?? '' })))
      } else {
        toast({ kind: 'error', title: errorMessage(e) })
      }
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!milestone) return
    await http.DELETE('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}', { params: { path: { workspace: slug, milestone: milestone.id } } })
    toast({ title: t(mm.deleted) })
    await refresh()
    onClose()
  }

  if (step === 'type') {
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        size="lg"
        eyebrow={<span className="k">{t(mm.editorEyebrow, { count: n })}</span>}
        title={title}
        description={d.stripe.connected ? t(mm.chooseTypeText) : t(mm.chooseTypeNoStripe)}
        footer={
          <>
            <Button onClick={onClose}>{t(mm.cancel)}</Button>
            <Button variant="primary" onClick={() => setStep('edit')}>
              {t(mm.continue)}
            </Button>
          </>
        }
      >
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="lbl mb-1.5">{t(mm.rewardType)}</legend>
          {TYPES.map((ty) => (
            <TypeOption key={ty} ty={ty} d={d} checked={f.type === ty} onPick={() => set({ type: ty })} />
          ))}
        </fieldset>
      </Dialog>
    )
  }

  const disabled = !canEdit
  return (
    <>
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        size="lg"
        className="max-w-[760px]"
        eyebrow={<span className="k">{t(mm.editorEyebrow, { count: n })}</span>}
        title={title}
        description={milestone ? t(mm.reachedSoFar, { count: milestone.stats.reached }) : undefined}
        footer={
          canEdit ? (
            <>
              {milestone ? (
                <span className="mr-auto flex flex-wrap items-center gap-2.5">
                  <Button variant="danger" onClick={() => setDeleting(true)}>
                    {t(mm.delete)}
                  </Button>
                  <span className="k">{t(mm.deleteNote)}</span>
                </span>
              ) : null}
              <Button onClick={onClose}>{t(mm.cancel)}</Button>
              <Button variant="primary" loading={saving} busyLabel={t(mm.saving)} onClick={() => void save()}>
                {t(mm.save)}
              </Button>
            </>
          ) : (
            <Button onClick={onClose}>{t(m.close)}</Button>
          )
        }
      >
        <div className="flex flex-col gap-3">
          <ErrorSummary errors={errors} title={t(mm.problems, { count: errors.length })} />
          <div className="grid grid-cols-[120px_minmax(0,1fr)] gap-3">
            <Field label={t(mm.count)} error={errors.find((x) => x.field === 'me-count')?.message}>
              <Input id="me-count" type="number" min={1} max={100000} value={f.count} disabled={disabled} onChange={(e) => set({ count: e.target.value })} />
            </Field>
            <Field label={t(mm.name)} error={errors.find((x) => x.field === 'me-name')?.message}>
              <Input id="me-name" value={f.rewardName} maxLength={120} disabled={disabled} onChange={(e) => set({ rewardName: e.target.value })} />
            </Field>
          </div>
          <Field
            label={t(mm.description)}
            hint={t(mm.descriptionHint, { count: f.rewardDescription.length })}
            error={errors.find((x) => x.field === 'me-desc')?.message}
          >
            <Input id="me-desc" value={f.rewardDescription} maxLength={160} disabled={disabled} onChange={(e) => set({ rewardDescription: e.target.value })} />
          </Field>
          <ImageRow f={f} set={set} disabled={disabled} />
          <div className="flex flex-col gap-1.5">
            <span className="lbl">{t(mm.rewardType)}</span>
            <TypeSegments value={f.type} d={d} disabled={disabled} onChange={(ty) => set({ type: ty })} />
            <span className="k">{typeText(t, f.type, true)}</span>
          </div>
          <TypeSettings f={f} set={set} d={d} milestone={milestone} errors={errors} disabled={disabled} />
          <EmailChoice f={f} set={set} d={d} milestone={milestone} disabled={disabled} />
        </div>
      </Dialog>
      {milestone ? (
        <ConfirmDialog
          open={deleting}
          onOpenChange={setDeleting}
          badge={t(mm.delete)}
          title={t(mm.deleteTitle, { count: milestone.count })}
          description={t(mm.deleteText)}
          confirmLabel={t(mm.delete)}
          onConfirm={remove}
        />
      ) : null}
    </>
  )
}

/** body.rewardName → me-name. */
function fieldOf(location?: string): string {
  const k = (location ?? '').replace(/^body\./, '')
  return (
    {
      count: 'me-count',
      rewardName: 'me-name',
      rewardDescription: 'me-desc',
      fileId: 'me-file',
      downloadUrl: 'me-url',
      couponOfferId: 'me-offer',
      webhookUrl: 'me-endpoint',
    }[k] ?? 'me-name'
  )
}

function typeText(t: ReturnType<typeof useT>, ty: RewardType, long = false): string {
  switch (ty) {
    case 'download':
      return t(mm.typeDownloadText)
    case 'codes':
      return t(mm.typeCodesText)
    case 'manual':
      return t(mm.typeManualText)
    case 'coupon':
      return long ? t(mm.typeCouponTextShort) : t(mm.typeCouponText)
    case 'webhook':
      return t(mm.typeWebhookText)
  }
  return ''
}

function TypeOption({ ty, d, checked, onPick }: { ty: RewardType; d: MilestoneList; checked: boolean; onPick: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEarn = useCan('earn:edit')
  const coupon = ty === 'coupon'
  const off = coupon && (!d.stripe.connected || !canEarn)
  return (
    <OptionCard name="me-type" value={ty} checked={checked} onChange={onPick} title={typeLabel(t, ty)} description={typeText(t, ty)} disabled={off}>
      {coupon ? (
        d.stripe.connected ? (
          <span className="k mt-1 font-semibold text-ok-fg">{t(mm.stripeConnectedAs, { account: d.stripe.account })}</span>
        ) : (
          <span className="k mt-1">
            <T
              msg={mm.stripeConnectIn}
              values={{
                link: (c) => (
                  <Link to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'payments' }}>
                    {c}
                  </Link>
                ),
              }}
            />
          </span>
        )
      ) : null}
    </OptionCard>
  )
}

function TypeSegments({ value, d, disabled, onChange }: { value: RewardType; d: MilestoneList; disabled: boolean; onChange: (ty: RewardType) => void }) {
  const t = useT()
  const canEarn = useCan('earn:edit')
  return (
    <div className="seg flex-wrap" role="radiogroup" aria-label={t(mm.rewardType)} style={{ alignSelf: 'flex-start' }}>
      {TYPES.map((ty) => {
        const off = disabled || (ty === 'coupon' && ty !== value && (!d.stripe.connected || !canEarn))
        return (
          <button
            key={ty}
            type="button"
            role="radio"
            aria-checked={value === ty}
            className={value === ty ? 'on' : undefined}
            disabled={off}
            onClick={() => onChange(ty)}
          >
            {typeLabel(t, ty)}
          </button>
        )
      })}
    </div>
  )
}

/** The reward's picture on the hub: upload, replace or remove. */
function ImageRow({ f, set, disabled }: { f: Form; set: (p: Partial<Form>) => void; disabled: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const input = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const upload = async (file: File) => {
    setBusy(true)
    try {
      const a = await uploadFile<Schemas['MediaAsset']>(`/api/v1/workspaces/${encodeURIComponent(slug)}/media`, file)
      set({ image: { id: a.id, url: a.url, name: a.fileName, width: a.width, height: a.height, size: a.sizeBytes } })
    } catch (e) {
      toast({ kind: 'error', title: t(mm.imageFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="flex flex-nowrap items-center gap-3 rounded-lg border border-line px-3 py-2.5">
      <span className="gw-mtile" style={{ width: 64, height: 40 }} aria-hidden="true">
        {f.image ? <img src={f.image.url} alt="" /> : null}
      </span>
      <span className="flex min-w-0 flex-1 flex-col gap-px">
        {f.image ? (
          <>
            <span className="mono text-[12.5px] font-semibold">{f.image.name}</span>
            <span className="k">{t(mm.imageLine, { width: f.image.width, height: f.image.height, size: fileSize(f.image.size) })}</span>
          </>
        ) : (
          <span className="k">{t(mm.imageNone)}</span>
        )}
      </span>
      {disabled ? null : (
        <>
          <input ref={input} type="file" accept="image/png,image/jpeg,image/gif" className="vh" tabIndex={-1} aria-hidden="true" onChange={(e) => e.target.files?.[0] && void upload(e.target.files[0])} />
          <Button size="sm" loading={busy} onClick={() => input.current?.click()}>
            {f.image ? t(mm.imageReplace) : t(mm.imageUpload)}
          </Button>
          {f.image ? (
            <Button size="sm" onClick={() => set({ image: undefined })}>
              {t(mm.imageRemove)}
            </Button>
          ) : null}
        </>
      )}
    </div>
  )
}

function Box({ title, aside, children }: { title: string; aside?: ReactNode; children: ReactNode }) {
  return (
    <div className="gw-sbox" style={{ background: 'var(--ground-3)' }}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="font-semibold">{title}</span>
        {aside ? <span className="k">{aside}</span> : null}
      </div>
      {children}
    </div>
  )
}

/** The settings of the chosen reward type. */
function TypeSettings({
  f,
  set,
  d,
  milestone,
  errors,
  disabled,
}: {
  f: Form
  set: (p: Partial<Form>) => void
  d: MilestoneList
  milestone?: Milestone
  errors: FormError[]
  disabled: boolean
}) {
  switch (f.type) {
    case 'download':
      return <DownloadSettings f={f} set={set} d={d} errors={errors} disabled={disabled} />
    case 'codes':
      return <CodesSettings f={f} set={set} milestone={milestone} disabled={disabled} />
    case 'manual':
      return <ManualSettings f={f} set={set} disabled={disabled} />
    case 'coupon':
      return <CouponSettings f={f} set={set} d={d} milestone={milestone} errors={errors} disabled={disabled} />
    case 'webhook':
      return <WebhookSettings f={f} set={set} milestone={milestone} errors={errors} disabled={disabled} />
  }
  return null
}

function DownloadSettings({ f, set, d, errors, disabled }: { f: Form; set: (p: Partial<Form>) => void; d: MilestoneList; errors: FormError[]; disabled: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const input = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState<string>()
  const refresh = useGrowRefresh()
  const file = d.files.find((x) => x.id === f.fileId)
  const [uploaded, setUploaded] = useState<Schemas['ReferralFile']>()
  const shown = file ?? (uploaded?.id === f.fileId ? uploaded : undefined)
  const upload = async (picked: File) => {
    setBusy(picked.name)
    try {
      const r = await uploadFile<Schemas['ReferralFile']>(`/api/v1/workspaces/${encodeURIComponent(slug)}/grow/referrals/${d.programId}/files`, picked)
      setUploaded(r)
      set({ fileId: r.id })
      void refresh()
    } catch (e) {
      toast({ kind: 'error', title: t(mm.fileFailed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  return (
    <Box title={t(mm.downloadTitle)} aside={t(mm.downloadCaption)}>
      <div className="seg" role="group" aria-label={t(mm.downloadTitle)} style={{ alignSelf: 'flex-start' }}>
        <button type="button" aria-pressed={f.downloadMode === 'file'} className={f.downloadMode === 'file' ? 'on' : undefined} disabled={disabled} onClick={() => set({ downloadMode: 'file' })}>
          {t(mm.uploadFile)}
        </button>
        <button type="button" aria-pressed={f.downloadMode === 'url'} className={f.downloadMode === 'url' ? 'on' : undefined} disabled={disabled} onClick={() => set({ downloadMode: 'url' })}>
          {t(mm.pasteUrl)}
        </button>
      </div>
      {f.downloadMode === 'file' ? (
        <>
          <div className="flex flex-wrap items-center gap-3" id="me-file" tabIndex={-1}>
            {shown ? (
              <span className="flex min-w-0 flex-1 flex-col gap-px">
                <span className="mono text-[12.5px] font-semibold">{shown.name}</span>
                <span className="k">
                  {fileKind(shown.contentType)} · {fileSize(shown.size)}
                  {shown.uploadedBy ? ` · ${shown.uploadedBy}` : ''} · {t(mm.downloads, { count: shown.downloads })}
                </span>
              </span>
            ) : (
              <span className="k flex-1">{busy ? t(mm.fileUploading, { name: busy }) : t(mm.lineNoFile)}</span>
            )}
            {disabled ? null : (
              <>
                <input
                  ref={input}
                  type="file"
                  accept="application/pdf,application/zip,image/png,image/jpeg,image/gif,image/webp"
                  className="vh"
                  tabIndex={-1}
                  aria-hidden="true"
                  onChange={(e) => e.target.files?.[0] && void upload(e.target.files[0])}
                />
                <Button size="sm" loading={Boolean(busy)} onClick={() => input.current?.click()}>
                  {shown ? t(mm.fileReplace) : t(mm.fileUpload)}
                </Button>
              </>
            )}
          </div>
          {d.files.length > 1 || (d.files.length === 1 && !shown) ? (
            <Field label={t(mm.pickFile)}>
              <Select value={f.fileId} disabled={disabled} onChange={(e) => set({ fileId: e.target.value })}>
                <option value="">—</option>
                {d.files.map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name} · {fileSize(x.size)}
                  </option>
                ))}
              </Select>
            </Field>
          ) : null}
          {errors.find((x) => x.field === 'me-file') ? <span className="err">{errors.find((x) => x.field === 'me-file')?.message}</span> : null}
          <span className="k">{t(mm.fileHint)}</span>
        </>
      ) : (
        <Field label={t(mm.urlLabel)} hint={t(mm.urlHint)} error={errors.find((x) => x.field === 'me-url')?.message}>
          <Input id="me-url" type="url" className="mono" value={f.downloadUrl} placeholder={'https://'} disabled={disabled} onChange={(e) => set({ downloadUrl: e.target.value })} />
        </Field>
      )}
    </Box>
  )
}

function CodesSettings({ f, set, milestone, disabled }: { f: Form; set: (p: Partial<Form>) => void; milestone?: Milestone; disabled: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  const upload = async () => {
    if (!milestone || !text.trim()) return
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/codes', {
        params: { path: { workspace: slug, milestone: milestone.id } },
        body: { text },
      })
      if (data) {
        toast({
          kind: 'success',
          title: t(mm.codesUploaded, { added: data.added }),
          detail: `${t(mm.codesUploadedDetail, { left: count(data.left), dupes: data.duplicates })}${data.released ? ` ${t(mm.codesReleased, { count: data.released })}` : ''}`,
        })
      }
      setText('')
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const removeUnused = async () => {
    if (!milestone) return
    try {
      const { data } = await http.DELETE('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/codes', { params: { path: { workspace: slug, milestone: milestone.id } } })
      toast({ title: t(mm.codesRemoved, { count: data?.removed ?? 0 }) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  return (
    <Box title={t(mm.codesTitle)} aside={t(mm.codesCaption)}>
      <span className="font-semibold">
        {milestone?.codesUploaded ? t(mm.codesStatus, { uploaded: milestone.codesUploaded, left: count(milestone.codesLeft) }) : t(mm.codesNone)}
      </span>
      {milestone ? (
        disabled ? null : (
          <>
            <Field label={t(mm.codesPaste)}>
              <Textarea className="mono" rows={4} value={text} onChange={(e) => setText(e.target.value)} />
            </Field>
            <div className="flex flex-wrap gap-2">
              <input
                ref={fileInput}
                type="file"
                accept=".csv,.txt,text/plain,text/csv"
                className="vh"
                tabIndex={-1}
                aria-hidden="true"
                onChange={async (e) => {
                  const file = e.target.files?.[0]
                  if (file) setText(await file.text())
                }}
              />
              <Button size="sm" onClick={() => fileInput.current?.click()}>
                {t(mm.codesUploadFile)}
              </Button>
              <Button size="sm" variant="primary" loading={busy} disabled={!text.trim()} onClick={() => void upload()}>
                {t(mm.codesUpload)}
              </Button>
              {milestone.codesLeft ? (
                <Button size="sm" variant="ghost" onClick={() => void removeUnused()}>
                  {t(mm.codesRemove)}
                </Button>
              ) : null}
            </div>
          </>
        )
      ) : (
        <span className="k">{t(mm.codesSaveFirst)}</span>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <span className="k">{t(mm.codesAlert)}</span>
        <Input
          className="gw-inl"
          type="number"
          min={0}
          aria-label={t(mm.codesAlertLabel)}
          value={f.codesAlertBelow}
          disabled={disabled}
          onChange={(e) => set({ codesAlertBelow: e.target.value })}
        />
        <span className="k">{t(mm.codesAlertUnit)}</span>
      </div>
      <span className="k">{t(mm.codesNote)}</span>
    </Box>
  )
}

function ManualSettings({ f, set, disabled }: { f: Form; set: (p: Partial<Form>) => void; disabled: boolean }) {
  const t = useT()
  return (
    <Box title={t(mm.manualTitle)} aside={t(mm.manualCaption)}>
      <Toggle label={t(mm.askAddress)} description={t(mm.askAddressText)} checked={f.askAddress} disabled={disabled} onChange={(e) => set({ askAddress: e.target.checked })} />
      <span className="k">{t(mm.manualNote)}</span>
    </Box>
  )
}

function CouponSettings({
  f,
  set,
  d,
  milestone,
  errors,
  disabled,
}: {
  f: Form
  set: (p: Partial<Form>) => void
  d: MilestoneList
  milestone?: Milestone
  errors: FormError[]
  disabled: boolean
}) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const user = useMe().data?.user
  const canEarn = useCan('earn:edit')
  const locked = disabled || !canEarn
  const offers = d.stripe.offers
  const current = offers.find((o) => o.id === f.couponOfferId)
  const kept = !current && milestone?.couponOfferId === f.couponOfferId && milestone.couponOfferName
  return (
    <Box title={t(mm.couponTitle)} aside={d.stripe.connected ? t(mm.couponConnected, { account: d.stripe.account }) : t(mm.couponNotConnected)}>
      {!canEarn && !disabled ? (
        <Note tone="neutral">
          <T msg={d.helpers.earn.length ? mm.couponNeedsEarn : mm.couponNeedsEarnNoHelpers} values={{ helpers: orNames(d.helpers.earn), mono }} />
        </Note>
      ) : null}
      <div className="grid grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)] gap-3 max-md:grid-cols-1">
        <Field
          label={t(mm.offer)}
          error={errors.find((x) => x.field === 'me-offer')?.message}
          hint={
            <T
              msg={mm.offerHint}
              values={{
                link: (c) => (
                  <Link to="/w/$workspace/earn/plans" params={{ workspace: slug }}>
                    {c}
                  </Link>
                ),
              }}
            />
          }
        >
          <Select id="me-offer" value={f.couponOfferId} disabled={locked} onChange={(e) => set({ couponOfferId: e.target.value })}>
            {kept ? <option value={f.couponOfferId}>{t(mm.offerKeep, { name: milestone.couponOfferName })}</option> : null}
            {!offers.length && !kept ? <option value="">{t(mm.offerNone)}</option> : null}
            {offers.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t(mm.validFor)}>
          <Select value={String(f.couponValidDays)} disabled={locked} onChange={(e) => set({ couponValidDays: Number(e.target.value) as Form['couponValidDays'] })}>
            <option value="30">{t(mm.days30)}</option>
            <option value="60">{t(mm.days60)}</option>
            <option value="90">{t(mm.days90)}</option>
            <option value="0">{t(mm.daysNone)}</option>
          </Select>
        </Field>
      </div>
      {current ? (
        <KeyValues
          items={[
            [t(mm.discount), current.discount],
            [t(mm.appliesTo), current.appliesTo],
            [t(mm.soFar), t(mm.soFarValue, { issued: current.issued, redeemed: count(current.redeemed) })],
          ]}
        />
      ) : null}
      <Note tone="info">{t(mm.couponNote)}</Note>
      {!canEarn && !disabled ? (
        <span className="k">{t(mm.couponLocked, { name: user?.name ?? '', role: roleName({ key: me.workspace.roleKey, name: me.workspace.roleName }) })}</span>
      ) : null}
      {!d.stripe.connected ? (
        <div>
          <Link to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'payments' }} className="btn sm pri">
            {t(mm.connectStripe)}
          </Link>
        </div>
      ) : null}
    </Box>
  )
}

function WebhookSettings({ f, set, milestone, errors, disabled }: { f: Form; set: (p: Partial<Form>) => void; milestone?: Milestone; errors: FormError[]; disabled: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [busy, setBusy] = useState(false)
  const [answer, setAnswer] = useState<{ at: Date; r: Schemas['ReferralWebhookTestResult'] }>()
  const test = async () => {
    if (!milestone) return
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/webhook-test', { params: { path: { workspace: slug, milestone: milestone.id } } })
      if (data) setAnswer({ at: new Date(), r: data })
    } catch (e) {
      toast({ kind: 'error', title: t(mm.testFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Box title={t(mm.webhookTitle)} aside={t(mm.webhookCaption)}>
      <Field label={t(mm.endpoint)} hint={t(mm.endpointHint)} error={errors.find((x) => x.field === 'me-endpoint')?.message}>
        <Input id="me-endpoint" type="url" className="mono" value={f.webhookUrl} placeholder={'https://'} disabled={disabled} onChange={(e) => set({ webhookUrl: e.target.value })} />
      </Field>
      <div className="flex flex-col gap-1">
        <span className="lbl">{t(mm.secret)}</span>
        <div className="flex flex-wrap items-center gap-2">
          {f.webhookSecret ? (
            <>
              <Input className="mono flex-1" value={f.webhookSecret} readOnly aria-label={t(mm.secret)} />
              <Button size="sm" icon="copy" onClick={() => void copy(f.webhookSecret)}>
                {t(mm.secretCopy)}
              </Button>
              <Badge tone="warn">{t(mm.secretNew)}</Badge>
            </>
          ) : (
            <>
              <span className="mono">••••••</span>
              {milestone?.webhookSecretSet ? <Badge tone="neutral">{t(mm.secretSet)}</Badge> : null}
              {disabled ? null : (
                <Button size="sm" onClick={() => set({ webhookSecret: newSecret() })}>
                  {t(mm.secretRotate)}
                </Button>
              )}
            </>
          )}
        </div>
        <span className="k">{t(mm.secretHint)}</span>
      </div>
      {milestone ? (
        <div className="flex flex-col gap-2">
          <div>
            <Button size="sm" loading={busy} busyLabel={t(mm.testing)} disabled={disabled} onClick={() => void test()}>
              {t(mm.sendTest)}
            </Button>
          </div>
          {answer ? (
            <div className="flex flex-col gap-1">
              <span className="k">{t(mm.testAnswer, { time: clock(answer.at) })}</span>
              <span>
                <Badge tone={answer.r.ok ? 'ok' : 'bad'}>{answer.r.status ? String(answer.r.status) : answer.r.error || '—'}</Badge>
              </span>
              {answer.r.code || answer.r.url ? <pre className="code m-0">{JSON.stringify({ code: answer.r.code || undefined, url: answer.r.url || undefined })}</pre> : null}
              {answer.r.error ? <span className="k">{answer.r.error}</span> : null}
            </div>
          ) : null}
          <span className="k">
            <T msg={mm.testHint} values={{ mono }} />
          </span>
        </div>
      ) : (
        <span className="k">{t(mm.testSaveFirst)}</span>
      )}
    </Box>
  )
}

/** Send a milestone email, or none (an automation sends one). */
function EmailChoice({ f, set, d, milestone, disabled }: { f: Form; set: (p: Partial<Form>) => void; d: MilestoneList; milestone?: Milestone; disabled: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const user = useMe().data?.user
  const [busy, setBusy] = useState(false)
  const test = async () => {
    if (!milestone || !user) return
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/email/test', {
        params: { path: { workspace: slug, milestone: milestone.id } },
        body: { to: [user.email] },
      })
      toast({ kind: 'success', title: t(mm.testSent, { email: user.email }), detail: t(mm.testSentDetailNoName) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
      <legend className="lbl mb-1.5">{t(mm.emailLegend)}</legend>
      <OptionCard
        name="me-email"
        value="on"
        checked={f.emailOn}
        disabled={disabled}
        onChange={() => set({ emailOn: true })}
        title={t(mm.emailSend)}
        description={milestone ? t(mm.emailSendLine, { subject: milestone.email.subject, sender: d.sender }) : t(mm.emailSendLineNew)}
      >
        {milestone && f.emailOn ? (
          <span className="mt-1.5 flex flex-wrap gap-2">
            <Link to="/w/$workspace/grow/referrals/emails/$milestoneId" params={{ workspace: slug, milestoneId: milestone.id }} className="btn sm">
              {t(mm.editInComposer)}
            </Link>
            <Button size="sm" loading={busy} onClick={() => void test()}>
              {t(mm.sendMeTest)}
            </Button>
          </span>
        ) : null}
      </OptionCard>
      <OptionCard
        name="me-email"
        value="off"
        checked={!f.emailOn}
        disabled={disabled}
        onChange={() => set({ emailOn: false })}
        title={t(mm.emailNone)}
        description={t(mm.emailNoneText)}
      />
    </fieldset>
  )
}

