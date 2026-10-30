import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useRef, useState, type ChangeEvent } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page, type Crumb } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog, UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { dayMonthYear, orList } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { roleName } from '@/lib/roles'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { me } from '../earn/messages'
import { PRODUCT, invalidateProducts, uploadCover, type ProductDetail } from './api'
import { pe } from './editor.messages'
import { ChecksCard, DefaultPage, NumbersCard, ShopCard, checkRows, useSiteFrame, type Preview } from './EditorAside'
import { EditorFiles } from './EditorFiles'
import { SHORT_MAX, changed, fieldOf, toFields, toForm, validate, type Form, type FormErrors } from './editorForm'
import { bareUrl, changesText, currencySymbol, day, fileSize, money, priceWord } from './format'
import { pm } from './messages'
import { Cover, IconNote, mono } from './parts'
import './products.css'

type SetForm = (p: Partial<Form>) => void
type TemplateOption = ProductDetail['templates'][number]

/** The product editor (ProductEditor, ProductEditorStates): /earn/products/{id}. */
export function ProductEditorPage({ productId }: { productId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('products:view')
  const q = $api.useQuery('get', PRODUCT, { params: { path: { workspace: slug, product: productId } } }, { enabled: can })
  const crumbs = (name: string): Crumb[] => [
    { label: t(me.title), to: '/w/$workspace/earn', params: { workspace: slug } },
    { label: t(pe.crumbProducts), to: '/w/$workspace/earn/products', params: { workspace: slug } },
    { label: name },
  ]
  if (!can) {
    return (
      <Page title={t(pm.tableLabel)}>
        <ForbiddenState permission="products:view" title={t(pe.forbiddenTitle)} />
      </Page>
    )
  }
  if (q.error) {
    return (
      <Page title={t(pe.loadError)} breadcrumb={crumbs(t(pe.loadError))}>
        <ErrorPanel title={t(pe.loadError)} actions={<Button onClick={() => void q.refetch()}>{t(pm.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      </Page>
    )
  }
  if (!q.data) {
    return (
      <Page title={t(me.loading)} breadcrumb={crumbs(t(me.loading))}>
        <div className="pr-ed" aria-busy="true">
          <div className="pr-ed-main">
            <div className="card pad">
              <Skeleton w="100%" h={420} />
            </div>
          </div>
          <div className="pr-ed-side">
            <div className="card pad">
              <Skeleton w="100%" h={320} />
            </div>
          </div>
        </div>
      </Page>
    )
  }
  return <Editor detail={q.data} crumbs={crumbs} />
}

function Editor({ detail, crumbs }: { detail: ProductDetail; crumbs: (name: string) => Crumb[] }) {
  const t = useT()
  const { slug, me: wme } = useWorkspace()
  const user = useMe().data?.user
  const qc = useQueryClient()
  const navigate = useNavigate()
  const canEdit = useCan('products:edit')
  const canPublish = useCan('products:publish')
  const cur = detail.payments.currency || 'usd'
  const [form, setForm] = useState<Form>(() => toForm(detail.draft, cur))
  const [errors, setErrors] = useState<FormErrors>({})
  const [priceOrders, setPriceOrders] = useState<number>()
  const [confirm, setConfirm] = useState<'archive' | 'delete'>()
  const [leaving, setLeaving] = useState(false)
  const path = { workspace: slug, product: detail.id }
  const queryKey = $api.queryOptions('get', PRODUCT, { params: { path } }).queryKey
  const save = $api.useMutation('put', PRODUCT)
  const publish = $api.useMutation('post', `${PRODUCT}/publish`)
  const archive = $api.useMutation('post', `${PRODUCT}/archive`)
  const unarchive = $api.useMutation('post', `${PRODUCT}/unarchive`)
  const remove = $api.useMutation('delete', PRODUCT)
  const dirty = canEdit && changed(form, detail.draft, cur)
  const ro = !canEdit
  const isDraft = detail.status === 'draft'
  const archived = detail.status === 'archived'
  const role = roleName({ key: wme.workspace.roleKey, name: wme.workspace.roleName })
  const name = form.name.trim() || detail.draft.name

  const set: SetForm = (p) => {
    setForm((f) => ({ ...f, ...p }))
    setErrors((e) => {
      const next = { ...e }
      for (const k of Object.keys(p)) delete next[k as keyof Form]
      return next
    })
  }
  const applied = (next: ProductDetail) => {
    qc.setQueryData(queryKey, next)
    setForm(toForm(next.draft, cur))
    setErrors({})
    invalidateProducts(qc)
  }
  const focusFirst = (e: FormErrors) => {
    const key = Object.keys(e)[0]
    if (key) document.getElementById(`pr-f-${key}`)?.focus()
  }
  const doSave = async (): Promise<ProductDetail | undefined> => {
    const e = validate(t, form, cur)
    setErrors(e)
    if (Object.keys(e).length) {
      focusFirst(e)
      toast({ kind: 'error', title: t(pe.saveFailed), detail: Object.values(e)[0] })
      return undefined
    }
    try {
      const next = await save.mutateAsync({ params: { path }, body: toFields(form, cur) })
      applied(next)
      return next
    } catch (err) {
      const found: FormErrors = {}
      if (err instanceof ApiError) {
        for (const x of err.problem.errors ?? []) {
          const k = fieldOf(x.location)
          if (k) found[k] = x.message ?? err.message
        }
        if (err.code === 'slug_taken') found.slug = err.message
      }
      setErrors(found)
      focusFirst(found)
      toast({ kind: 'error', title: t(pe.saveFailed), detail: errorMessage(err) })
      return undefined
    }
  }
  const onSave = async () => {
    const next = await doSave()
    if (next) toast({ kind: 'success', title: next.status === 'draft' ? t(pe.saved) : t(pe.savedLive) })
  }
  const doPublish = async (confirmPrice = false) => {
    let d = detail
    if (dirty) {
      const next = await doSave()
      if (!next) return
      d = next
    }
    const failing = checkRows(t, d).filter((r) => !r.ok).length
    if (failing) {
      toast({ kind: 'error', title: t(pe.publishFailed), detail: t(pe.fixChecksReason, { count: failing }) })
      return
    }
    try {
      const next = await publish.mutateAsync({ params: { path }, body: confirmPrice ? { confirmPrice: true } : {} })
      applied(next)
      setPriceOrders(undefined)
      toast(
        d.status === 'draft'
          ? { kind: 'success', title: t(pe.published, { name: next.draft.name }), detail: t(pe.publishedDetail) }
          : { kind: 'success', title: t(pe.changesPublished, { name: next.draft.name }) },
      )
    } catch (err) {
      if (err instanceof ApiError && err.code === 'price_change') {
        const orders = Number(err.problem.meta?.orders)
        setPriceOrders(Number.isFinite(orders) ? orders : d.orders)
        return
      }
      setPriceOrders(undefined)
      if (err instanceof ApiError && err.code === 'checks_failed') void qc.invalidateQueries({ queryKey })
      toast({ kind: 'error', title: t(pe.publishFailed), detail: errorMessage(err) })
    }
  }
  const onUnarchive = () =>
    unarchive.mutate(
      { params: { path } },
      {
        onSuccess: (next) => {
          applied(next)
          toast({ kind: 'success', title: t(pe.unarchived, { name: next.draft.name }) })
        },
        onError: (e) => toast({ kind: 'error', title: t(pe.publishFailed), detail: errorMessage(e) }),
      },
    )

  const rows = checkRows(t, detail)
  const failing = rows.filter((r) => !r.ok).length
  const helpers = detail.helpers.publish
  const publishReason = !canPublish
    ? helpers.length
      ? t(dirty ? pe.publishNeedsUnsaved : pe.publishNeeds, { helpers: orList(helpers) })
      : t(pe.publishNeedsNoHelpers)
    : failing && !dirty
      ? t(pe.fixChecksReason, { count: failing })
      : !isDraft && !dirty && detail.changes.length === 0
        ? t(pe.nothingToPublish)
        : undefined
  const sideText = dirty
    ? t(pe.unsaved)
    : isDraft
      ? detail.updatedBy || undefined
      : detail.changes.length
        ? t(pe.draftChanges, { changes: changesText(t, detail.changes) })
        : undefined
  const menu = [
    canPublish && detail.status === 'published' ? (
      <MenuItem key="archive" onSelect={() => setConfirm('archive')}>
        {t(pe.archive)}
      </MenuItem>
    ) : null,
    canEdit && isDraft ? (
      <MenuItem key="delete" onSelect={() => setConfirm('delete')}>
        {t(pe.deleteDraft)}
      </MenuItem>
    ) : null,
  ].filter(Boolean)

  const actions = (
    <div className="pr-hdr">
      <StatusBadge detail={detail} />
      {sideText ? <span className="k">{sideText}</span> : null}
      {canPublish && failing > 0 && !dirty && !archived ? <span className="k">{t(pe.fixChecks, { count: failing })}</span> : null}
      {canEdit ? (
        <Button onClick={() => void onSave()} loading={save.isPending && !publish.isPending}>
          {t(pe.save)}
        </Button>
      ) : null}
      {archived ? (
        canPublish ? (
          <Button variant="primary" onClick={onUnarchive} loading={unarchive.isPending}>
            {t(pe.sellAgain)}
          </Button>
        ) : null
      ) : canEdit || canPublish ? (
        <Button variant="primary" disabledReason={publishReason} loading={publish.isPending} busyLabel={t(pe.publishing)} onClick={() => void doPublish()}>
          {isDraft ? t(pe.publish) : t(pe.publishChanges)}
        </Button>
      ) : null}
      {menu.length ? (
        <Menu label={t(pe.more)} trigger={<Button iconOnly icon="more" aria-label={t(pe.more)} />}>
          {menu}
        </Menu>
      ) : null}
    </div>
  )

  const errs = validate(t, form, cur)
  const priceOK = !errs.amount && !errs.minimum && !errs.suggested
  const fields = toFields(form, cur)
  const preview: Preview = { ...fields, price: priceOK ? fields.price : detail.draft.price, offerOn: fields.offer.on, offerText: fields.offer.text }

  return (
    <Page title={name} documentTitle={t(pe.documentTitle, { name })} breadcrumb={crumbs(name)} actions={actions}>
      {ro ? (
        <IconNote tone="n" title={t(pe.readOnlyTitle, { name: user?.name ?? '', role })}>
          <T msg={detail.helpers.edit.length ? pe.readOnlyTextAsk : pe.readOnlyText} values={{ helpers: orList(detail.helpers.edit) }} />
        </IconNote>
      ) : !canPublish ? (
        <IconNote tone="n" title={t(pe.noPublishTitle, { name: user?.name ?? '' })}>
          <T msg={pe.noPublishText} values={{ role }} />
        </IconNote>
      ) : null}
      {archived ? (
        <IconNote tone="n" title={t(pm.statusArchived)}>
          {t(pe.archivedNote, { date: detail.archivedAt ? day(detail.archivedAt) : '' })}
        </IconNote>
      ) : null}
      <div className="pr-ed">
        <div className="pr-ed-main">
          <DetailsCard detail={detail} form={form} set={set} errors={errors} ro={ro} />
          <EditorFiles detail={detail} canEdit={canEdit} />
          <PriceCard detail={detail} form={form} set={set} errors={errors} ro={ro} />
          <SellingCard detail={detail} form={form} set={set} errors={errors} ro={ro} />
          <DeliveryCard detail={detail} form={form} set={set} errors={errors} ro={ro} />
        </div>
        <div className="pr-ed-side">
          <ShopCard detail={detail} p={preview} />
          <DefaultPage detail={detail} p={preview} />
          <ChecksCard detail={detail} rows={rows} />
          {detail.numbers && !isDraft ? <NumbersCard detail={detail} /> : null}
        </div>
      </div>
      {priceOrders !== undefined && detail.live ? (
        <PriceDialog detail={detail} orders={priceOrders} busy={publish.isPending} onCancel={() => setPriceOrders(undefined)} onConfirm={() => void doPublish(true)} />
      ) : null}
      <ConfirmDialog
        open={confirm === 'archive'}
        onOpenChange={(o) => !o && setConfirm(undefined)}
        tone="warn"
        badge={t(pe.archive)}
        title={t(pe.archiveTitle, { name })}
        description={t(pe.archiveText)}
        confirmLabel={t(pe.archive)}
        busyLabel={t(pe.archiving)}
        onConfirm={async () => {
          const next = await archive.mutateAsync({ params: { path } })
          applied(next)
          toast({ kind: 'success', title: t(pe.archived, { name: next.draft.name }) })
        }}
      />
      <ConfirmDialog
        open={confirm === 'delete'}
        onOpenChange={(o) => !o && setConfirm(undefined)}
        tone="bad"
        badge={t(pe.deleteDraft)}
        title={t(pe.deleteTitle, { name })}
        description={t(pe.deleteText)}
        confirmLabel={t(pe.deleteDraft)}
        busyLabel={t(pe.deleting)}
        onConfirm={async () => {
          await remove.mutateAsync({ params: { path } })
          setLeaving(true)
          invalidateProducts(qc)
          toast({ kind: 'deleted', title: t(pe.deleted, { name }) })
          void navigate({ to: '/w/$workspace/earn/products', params: { workspace: slug } })
        }}
      />
      <UnsavedChangesGuard
        dirty={dirty && !leaving}
        title={t(pe.leaveTitle)}
        description={t(pe.leaveText, { name })}
        saveLabel={t(pe.save)}
        onSave={async () => {
          if (!(await doSave())) throw new Error(t(pe.saveFailed))
        }}
      />
    </Page>
  )
}

function StatusBadge({ detail }: { detail: ProductDetail }) {
  const t = useT()
  if (detail.status === 'draft') return <Badge tone="neutral">{t(pe.statusDraft)}</Badge>
  if (detail.status === 'archived') return <Badge tone="neutral">{detail.archivedAt ? t(pe.statusArchived, { date: day(detail.archivedAt) }) : t(pe.statusArchivedNoDate)}</Badge>
  const date = detail.publishedAt ? day(detail.publishedAt) : ''
  return (
    <Badge tone="ok" dot>
      {detail.publishedBy ? t(pe.statusPublished, { date, name: detail.publishedBy }) : t(pe.statusPublishedNoName, { date })}
    </Badge>
  )
}

type CardProps = { detail: ProductDetail; form: Form; set: SetForm; errors: FormErrors; ro: boolean }

/** Name, address, descriptions and the cover. */
function DetailsCard({ detail, form, set, errors, ro }: CardProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const input = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const onCover = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    if (!/^image\/(png|jpeg|gif)$/.test(file.type)) {
      toast({ kind: 'error', title: t(pe.coverFailed), detail: t(pe.coverType) })
      return
    }
    if (file.size > 5 * 1024 * 1024) {
      toast({ kind: 'error', title: t(pe.coverFailed), detail: t(pe.coverTooBig, { size: fileSize(t, file.size) }) })
      return
    }
    setUploading(true)
    try {
      const c = await uploadCover(slug, detail.id, file)
      set({ coverUrl: c.url })
      toast({ kind: 'success', title: t(pe.coverUploaded), detail: t(pe.coverUploadedDetail) })
    } catch (err) {
      toast({ kind: 'error', title: t(pe.coverFailed), detail: errorMessage(err) })
    } finally {
      setUploading(false)
    }
  }
  return (
    <section className="card" aria-labelledby="pr-details-h">
      <CardSection id="pr-details-h" title={t(pe.details)} />
      <div className="pad col">
        <Field label={t(pe.name)} error={errors.name}>
          <Input id="pr-f-name" value={form.name} maxLength={120} disabled={ro} onChange={(e) => set({ name: e.target.value })} />
        </Field>
        <div className="lbl">
          <label htmlFor="pr-f-slug">{t(pe.address)}</label>
          <div className="pr-addr">
            <span className="k pre">{detail.siteHost}</span>
            <Input
              id="pr-f-slug"
              value={form.slug}
              maxLength={80}
              disabled={ro}
              invalid={Boolean(errors.slug)}
              aria-describedby={errors.slug ? `${id}-slug-e ${id}-slug-h` : `${id}-slug-h`}
              onChange={(e) => set({ slug: e.target.value })}
            />
          </div>
          {errors.slug ? (
            <span className="err" id={`${id}-slug-e`} role="alert">
              {errors.slug}
            </span>
          ) : null}
          <span className="k" id={`${id}-slug-h`}>
            {t(pe.addressHint)}
          </span>
        </div>
        <Field label={t(pe.shortDescription)} hint={t(pe.shortHint, { count: form.shortDescription.length, max: SHORT_MAX })} error={errors.shortDescription}>
          <Input id="pr-f-shortDescription" value={form.shortDescription} maxLength={SHORT_MAX} disabled={ro} onChange={(e) => set({ shortDescription: e.target.value })} />
        </Field>
        <Field label={t(pe.longDescription)} hint={t(pe.longHint)}>
          <Textarea id="pr-f-longDescription" value={form.longDescription} maxLength={4000} rows={3} className="h-[88px]" disabled={ro} onChange={(e) => set({ longDescription: e.target.value })} />
        </Field>
        <div className="pr-coverrow">
          <div className="pic">
            <Cover url={form.coverUrl} alt={form.coverAlt} name={form.name} w={72} h={96} />
            {ro ? null : (
              <>
                <Button size="sm" loading={uploading} busyLabel={t(pe.coverUploading)} onClick={() => input.current?.click()}>
                  {form.coverUrl ? t(pe.coverReplace) : t(pe.coverUpload)}
                </Button>
                <input ref={input} type="file" accept="image/png,image/jpeg,image/gif" className="vh" tabIndex={-1} aria-label={t(pe.coverLabel)} onChange={(e) => void onCover(e)} />
              </>
            )}
          </div>
          <div className="alt">
            <Field label={t(pe.coverAlt)} hint={t(pe.coverAltHint)} error={errors.coverAlt ?? errors.coverUrl}>
              <Textarea id="pr-f-coverAlt" value={form.coverAlt} maxLength={300} rows={2} className="h-[60px] min-h-[60px]" disabled={ro} onChange={(e) => set({ coverAlt: e.target.value })} />
            </Field>
          </div>
        </div>
      </div>
    </section>
  )
}

/** A money input with the currency's symbol before it (ProductEditor: .money-in). */
function MoneyField({
  field,
  label,
  ariaLabel,
  currency,
  value,
  onChange,
  error,
  hint,
  disabled,
}: {
  field: 'amount' | 'minimum' | 'suggested'
  label: string
  ariaLabel: string
  currency: string
  value: string
  onChange: (v: string) => void
  error?: string
  hint?: string
  disabled: boolean
}) {
  const id = useId()
  return (
    <div className="lbl">
      <label htmlFor={`pr-f-${field}`}>{label}</label>
      <span className={cn('pr-money', error && 'err')}>
        <span aria-hidden="true">{currencySymbol(currency)}</span>
        <input
          id={`pr-f-${field}`}
          inputMode="decimal"
          autoComplete="off"
          value={value}
          aria-label={ariaLabel}
          aria-invalid={error ? true : undefined}
          aria-describedby={error || hint ? `${id}-d` : undefined}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
        />
      </span>
      {error ? (
        <span className="pr-ferr" id={`${id}-d`} role="alert">
          {error}
        </span>
      ) : hint ? (
        <span className="k" id={`${id}-d`}>
          {hint}
        </span>
      ) : null}
    </div>
  )
}

/** One-time or pay what you want, in the Stripe account's currency. */
function PriceCard({ detail, form, set, errors, ro }: CardProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEarn = useCan('earn:view')
  const p = detail.payments
  const cur = p.currency || 'usd'
  const code = cur.toUpperCase()
  const msg = !p.connected ? pe.currencyNoStripe : !p.taxAtCheckout ? pe.currencyTaxOff : p.pricesIncludeTax ? pe.currencyTaxIncluded : pe.currencyTaxOn
  return (
    <section className="card" aria-labelledby="pr-price-h">
      <CardSection id="pr-price-h" title={t(pe.price)} />
      <div className="pad col">
        <div>
          <div className="seg" role="group" aria-label={t(pe.priceKind)}>
            {(['one_time', 'pwyw'] as const).map((k) => (
              <button key={k} type="button" aria-pressed={form.kind === k} disabled={ro} onClick={() => set({ kind: k })}>
                {k === 'one_time' ? t(pe.oneTime) : t(pe.pwyw)}
              </button>
            ))}
          </div>
        </div>
        {form.kind === 'one_time' ? (
          <MoneyField
            field="amount"
            label={t(pe.price)}
            ariaLabel={t(pe.priceIn, { currency: code })}
            currency={cur}
            value={form.amount}
            error={errors.amount}
            disabled={ro}
            onChange={(v) => set({ amount: v })}
          />
        ) : (
          <div className="pr-fieldrow">
            <MoneyField
              field="minimum"
              label={t(pe.minimum)}
              ariaLabel={t(pe.minimumIn, { currency: code })}
              currency={cur}
              value={form.minimum}
              error={errors.minimum}
              disabled={ro}
              onChange={(v) => set({ minimum: v })}
            />
            <MoneyField
              field="suggested"
              label={t(pe.suggested)}
              ariaLabel={t(pe.suggestedIn, { currency: code })}
              currency={cur}
              value={form.suggested}
              error={errors.suggested}
              hint={t(pe.suggestedHint)}
              disabled={ro}
              onChange={(v) => set({ suggested: v })}
            />
          </div>
        )}
        <div className="pr-info">
          <Icon name="info" size={14} />
          <span>
            <T
              msg={msg}
              values={{
                currency: code,
                account: p.account,
                link: (c) =>
                  canEarn ? (
                    <Link to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'payments' }}>
                      {c}
                    </Link>
                  ) : (
                    <span>{c}</span>
                  ),
              }}
            />
          </span>
        </div>
      </div>
    </section>
  )
}

/** The sales page, the shop and the list offer. */
function SellingCard({ detail, form, set, errors, ro }: CardProps) {
  const t = useT()
  return (
    <section className="card" aria-labelledby="pr-selling-h">
      <CardSection id="pr-selling-h" title={t(pe.selling)} />
      <div className="pad col">
        <Field label={t(pe.salesPage)} hint={t(pe.salesPageHint)} error={errors.salesPageId}>
          <Select id="pr-f-salesPageId" value={form.salesPageId} disabled={ro} onChange={(e) => set({ salesPageId: e.target.value })}>
            {detail.pages.map((p) => (
              <option key={p.id} value={p.id}>
                {t(pe.pageOption, { name: p.name, path: p.path, status: p.status })}
              </option>
            ))}
            <option value="">{t(pe.defaultPageOption)}</option>
          </Select>
        </Field>
        <div>
          <Switch id="pr-f-showInShop" label={t(pe.showInShop)} hint={bareUrl(detail.shopUrl)} checked={form.showInShop} disabled={ro} onChange={(v) => set({ showInShop: v })} />
          <Switch id="pr-f-offerOn" label={t(pe.listOffer)} hint={t(pe.listOfferHint)} checked={form.offerOn} disabled={ro} onChange={(v) => set({ offerOn: v })} />
        </div>
        {form.offerOn ? (
          <div className="pr-fieldrow">
            <Field label={t(pe.list)} error={errors.offerListId}>
              <Select id="pr-f-offerListId" value={form.offerListId} disabled={ro} onChange={(e) => set({ offerListId: e.target.value })}>
                {form.offerListId ? null : <option value="">{t(pe.listPick)}</option>}
                {detail.lists.map((l) => (
                  <option key={l.id} value={l.id}>
                    {t(pe.listOption, { name: l.name, doi: l.doubleOptIn ? 'yes' : 'no' })}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label={t(pe.offerText)} error={errors.offerText}>
              <Input id="pr-f-offerText" value={form.offerText} maxLength={160} disabled={ro} aria-label={t(pe.offerTextLabel)} onChange={(e) => set({ offerText: e.target.value })} />
            </Field>
          </div>
        ) : null}
        <span className="k">{t(pe.consentNote)}</span>
      </div>
    </section>
  )
}

/** A setting's row with a switch on the right (ProductEditor: .srow). */
function Switch({ id, label, hint, checked, disabled, onChange }: { id: string; label: string; hint?: string; checked: boolean; disabled: boolean; onChange: (v: boolean) => void }) {
  const hintId = `${id}-h`
  return (
    <div className="pr-srow">
      <span className="tt">
        <label htmlFor={id}>{label}</label>
        {hint ? (
          <span className="k" id={hintId}>
            {hint}
          </span>
        ) : null}
      </span>
      <input id={id} type="checkbox" role="switch" className="tg" checked={checked} disabled={disabled} aria-describedby={hint ? hintId : undefined} onChange={(e) => onChange(e.target.checked)} />
    </div>
  )
}

/** The delivery email, the link's limits and revoking on refunds. */
function DeliveryCard({ detail, form, set, errors, ro }: CardProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const tpl = detail.templates.find((x) => x.id === form.templateId)
  return (
    <section className="card" aria-labelledby="pr-delivery-h">
      <CardSection id="pr-delivery-h" title={t(pe.delivery)} />
      <div className="pad col">
        <div className="pr-delivery">
          <MailPreview detail={detail} form={form} template={tpl} />
          <div className="pr-tplc">
            <Field label={t(pe.template)} error={errors.templateId}>
              <Select id="pr-f-templateId" value={form.templateId} disabled={ro} onChange={(e) => set({ templateId: e.target.value })}>
                {form.templateId ? null : <option value="">{t(pe.templatePick)}</option>}
                {detail.templates.map((x) => (
                  <option key={x.id} value={x.id}>
                    {t(pe.templateOption, { key: x.key, version: x.version, category: x.category, kind: x.kind })}
                  </option>
                ))}
              </Select>
            </Field>
            <span className="k">
              <T msg={detail.route ? pe.routeNote : pe.routeNone} values={{ route: detail.route, mono }} />
            </span>
            {tpl ? (
              <Link to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: tpl.id }} className="pr-lnk self-start">
                {t(pe.editTemplate)}
              </Link>
            ) : null}
          </div>
        </div>
        <div className="pr-fieldrow">
          <NumberField field="linkDays" label={t(pe.linkDays)} hint={t(pe.linkDaysHint)} min={1} max={30} form={form} set={set} error={errors.linkDays} ro={ro} />
          <NumberField field="downloads" label={t(pe.downloads)} hint={t(pe.downloadsHint)} min={1} max={20} form={form} set={set} error={errors.downloads} ro={ro} />
        </div>
        <div>
          <Switch id="pr-f-revokeOnRefund" label={t(pe.revoke)} hint={t(pe.revokeHint)} checked={form.revokeOnRefund} disabled={ro} onChange={(v) => set({ revokeOnRefund: v })} />
        </div>
      </div>
    </section>
  )
}

function NumberField({
  field,
  label,
  hint,
  min,
  max,
  form,
  set,
  error,
  ro,
}: {
  field: 'linkDays' | 'downloads'
  label: string
  hint: string
  min: number
  max: number
  form: Form
  set: SetForm
  error?: string
  ro: boolean
}) {
  const id = useId()
  return (
    <div className="lbl">
      <label htmlFor={`pr-f-${field}`}>{label}</label>
      <div className="pr-numrow">
        <Input
          id={`pr-f-${field}`}
          type="number"
          inputMode="numeric"
          min={min}
          max={max}
          value={form[field]}
          aria-describedby={`${id}-h`}
          invalid={Boolean(error)}
          disabled={ro}
          onChange={(e) => set({ [field]: e.target.value } as Partial<Form>)}
        />
        <span className="k" id={`${id}-h`}>
          {hint}
        </span>
      </div>
      {error ? (
        <span className="pr-ferr" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  )
}

/** The delivery email, small, with this product's values (ProductEditor: Delivery). */
function MailPreview({ detail, form, template }: { detail: ProductDetail; form: Form; template?: TemplateOption }) {
  const t = useT()
  const now = useNow(60_000)
  const site = useSiteFrame()
  const file = detail.files[0]
  const days = Number(form.linkDays) || 7
  const until = new Date(now.getTime() + days * 86_400_000)
  const product = form.name.trim() || detail.draft.name
  return (
    <div
      className="pr-mail"
      role="img"
      aria-label={template && template.version > 0 ? t(pe.mailLabel, { template: template.key, version: template.version }) : t(pe.mailLabelNoVersion)}
    >
      <div className="mh">{site.title}</div>
      <div className="mb">
        <span className="h">{t(pe.mailHeading)}</span>
        <span className="p">{t(pe.mailThanks, { product })}</span>
        <span className="b">{file ? t(pe.mailButton, { kind: file.kindLabel, size: fileSize(t, file.sizeBytes) }) : t(pe.mailButtonNoFile, { product })}</span>
        <span className="s">{t(pe.mailWorks, { date: dayMonthYear(until), count: Number(form.downloads) || 5 })}</span>
      </div>
    </div>
  )
}

/** "Change the price of The Deliverability Field Guide from $29 to $35?" (ProductEditorStates). */
function PriceDialog({ detail, orders, busy, onCancel, onConfirm }: { detail: ProductDetail; orders: number; busy: boolean; onCancel: () => void; onConfirm: () => void }) {
  const t = useT()
  const cur = detail.payments.currency || 'usd'
  const before = detail.live!.price
  const after = detail.draft.price
  const old = priceWord(t, before, cur)
  const next = priceWord(t, after, cur)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && onCancel()}
      hideClose
      eyebrow={<span className="k">{t(pe.pcEyebrow)}</span>}
      title={t(pe.pcTitle, { name: detail.draft.name, old, new: next })}
      footer={
        <>
          <span className="k mr-auto">{t(pe.pcAudit)}</span>
          <Button onClick={onCancel} disabled={busy}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" loading={busy} busyLabel={t(pe.publishing)} onClick={onConfirm}>
            {t(pe.pcConfirm)}
          </Button>
        </>
      }
    >
      <span>{after.kind === 'pwyw' ? t(pe.pcTextPwyw, { min: money(after.minimumCents, cur) }) : t(pe.pcText, { new: next })}</span>
      <dl className="kv">
        <dt>{t(pe.pcBought)}</dt>
        <dd>{t(pe.pcBoughtValue, { count: orders })}</dd>
        <dt>{t(pe.pcStripe)}</dt>
        <dd>{t(pe.pcStripeValue, { new: next, old })}</dd>
        <dt>{t(pe.pcOpen)}</dt>
        <dd>{t(pe.pcOpenValue, { old })}</dd>
      </dl>
    </Dialog>
  )
}

