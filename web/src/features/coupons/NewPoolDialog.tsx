import { Link, useNavigate } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Checkbox, Field, Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { useCatalog } from '@/features/commerce/api'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useCan, useWorkspace } from '@/lib/session'
import { exampleCode, platformName, useRefreshCoupons, useSources, type Source } from './api'
import { m } from './messages'

type Kind = 'percentage' | 'fixed_amount' | 'free_shipping'
type Platform = 'shopify' | 'woocommerce'

const mono = (c: ReactNode[]) => <span className="mono">{c}</span>

// Examples in fields: a prefix and an apply address are code, not copy.
const PREFIX_EXAMPLE = 'WELCOME-'
const APPLY_EXAMPLE = 'https://shop.example.com/discount/{code}'

/** The name templates use, from the pool's name: "Welcome 15%" → welcome_15. */
export function slugOf(name: string): string {
  return name
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 32)
}

/** A source card: a radio, its title and hint, and why it's off. */
function SourceCard({ value, checked, onPick, title, hint, why, disabled }: {
  value: string
  checked: boolean
  onPick: (v: string) => void
  title: ReactNode
  hint: ReactNode
  why?: ReactNode
  disabled?: boolean
}) {
  const id = `np-src-${value}`
  return (
    <label className="cp-src">
      <input type="radio" name="np-src" value={value} checked={checked} disabled={disabled} aria-describedby={why ? `${id}-why` : undefined} onChange={() => onPick(value)} />
      <span>
        <b>{title}</b>
        <span className="k">{hint}</span>
        {why ? (
          <span className="k text-ink-nav" id={`${id}-why`}>
            {why}
          </span>
        ) : null}
      </span>
    </label>
  )
}

/** New pool (DiscountCodes, DiscountCodesStates · store not allowed, key read-only). */
export function NewPoolDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const { slug: ws } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshCoupons()
  const sources = useSources(open)
  const catalog = useCatalog()
  const canIntegrations = useCan('integrations:manage')
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [slugEdited, setSlugEdited] = useState(false)
  const [source, setSource] = useState('upload')
  const [kind, setKind] = useState<Kind>('percentage')
  const [value, setValue] = useState('15')
  const [minimum, setMinimum] = useState('')
  const [ends, setEnds] = useState('')
  const [collection, setCollection] = useState('')
  const [prefix, setPrefix] = useState('')
  const [onePer, setOnePer] = useState(true)
  const [applyUrl, setApplyUrl] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/coupons/pools')
  const allow = $api.useMutation('post', '/api/v1/workspaces/{workspace}/stores/{id}/allow-discounts')

  const list = sources.data?.items ?? []
  const store: Source | undefined = list.find((s) => s.storeId === source)
  const currency = store?.currency || 'USD'
  const collections = useMemo(() => catalog.data?.stores.find((s) => s.id === store?.storeId)?.collections ?? [], [catalog.data, store])
  const token = `coupons.${slug || 'welcome15'}`

  const submit = async () => {
    setErrors({})
    const isStore = !!store
    const n = (s: string) => (s.trim() === '' ? 0 : Math.round(Number(s.replace(',', '.'))))
    const cents = (s: string) => (s.trim() === '' ? 0 : Math.round(Number(s.replace(',', '.')) * 100))
    try {
      const out = await create.mutateAsync({
        params: { path: { workspace: ws } },
        body: {
          name: name.trim(),
          slug,
          source: (isStore ? store.platform : 'upload') as 'upload' | Platform,
          storeId: isStore ? store.storeId : undefined,
          prefix: prefix.trim().toUpperCase() || undefined,
          onePerContact: onePer,
          settings: isStore
            ? {
                kind,
                value: kind === 'percentage' ? n(value) : kind === 'fixed_amount' ? cents(value) : undefined,
                minimumCents: cents(minimum) || undefined,
                endsDays: n(ends) || undefined,
                collectionIds: collection ? [collection] : undefined,
              }
            : { applyUrl: applyUrl.trim() || undefined },
        },
      })
      void refresh()
      onOpenChange(false)
      toast({ kind: 'success', title: t(m.created), detail: t(m.createdDetail, { name: out.name }) })
      void navigate({ to: '/w/$workspace/campaigns/codes/$poolId', params: { workspace: ws, poolId: out.id } })
    } catch (e) {
      if (e instanceof ApiError && e.problem.errors?.length) {
        const next: Record<string, string> = {}
        for (const x of e.problem.errors) next[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
        setErrors(next)
      } else {
        setErrors({ form: errorMessage(e) })
      }
    }
  }

  const allowDiscounts = async (id: string) => {
    try {
      const out = await allow.mutateAsync({ params: { path: { workspace: ws, id } } })
      window.location.assign(out.url)
    } catch (e) {
      setErrors({ form: errorMessage(e) })
    }
  }

  // One card per connected store, and one per platform without a store.
  const platforms: Platform[] = ['shopify', 'woocommerce']
  const cards = platforms.flatMap((p) => {
    const mine = list.filter((s) => s.platform === p)
    if (mine.length === 0) {
      return [
        <SourceCard
          key={p}
          value={p}
          checked={false}
          onPick={setSource}
          disabled
          title={t(m.srcPlatform, { platform: platformName(p) })}
          hint={p === 'woocommerce' ? t(m.srcWooHint) : t(m.srcShopifyHint)}
          why={
            <T
              msg={m.srcNone}
              values={{
                platform: platformName(p),
                a: (c) => (
                  <Link to="/w/$workspace/settings/integrations" params={{ workspace: ws }}>
                    {c}
                  </Link>
                ),
              }}
            />
          }
        />,
      ]
    }
    return mine.map((s) => {
      let why: ReactNode
      if (!s.can) {
        why = (
          <>
            {s.detail}
            {s.reason === 'not_granted' && canIntegrations ? (
              <span className="mt-1.5 block">
                <Button size="sm" onClick={() => void allowDiscounts(s.storeId)} loading={allow.isPending}>
                  {t(m.allowDiscounts)}
                </Button>
              </span>
            ) : s.reason === 'read_only' && canIntegrations ? (
              <span className="mt-1.5 block">
                <Link className="btn sm" to="/w/$workspace/settings/integrations/stores/$storeId" params={{ workspace: ws, storeId: s.storeId }}>
                  {t(m.replaceKey)}
                </Link>
              </span>
            ) : null}
          </>
        )
      }
      return (
        <SourceCard
          key={s.storeId}
          value={s.storeId}
          checked={source === s.storeId}
          onPick={setSource}
          disabled={!s.can}
          title={t(m.srcStore, { platform: platformName(p), store: s.platform === 'woocommerce' && !s.can ? s.domain || s.name : s.name })}
          hint={p === 'woocommerce' ? t(m.srcWooStoreHint) : s.can ? t(m.srcShopifyHint) : t(m.srcShopifyBlocked)}
          why={why}
        />
      )
    })
  })

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.dlgEyebrow)}</span>}
      title={t(m.newPool)}
      size="md"
      footer={
        <>
          <span className="k mr-auto">{t(m.audited)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={create.isPending} onClick={() => void submit()}>
            {t(m.createPool)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {errors.form ? (
          <Note tone="bad" role="alert">
            {errors.form}
          </Note>
        ) : null}
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(240px,100%),1fr))' }}>
          <Field label={t(m.name)} error={errors.name}>
            <Input
              value={name}
              maxLength={80}
              onChange={(e) => {
                setName(e.target.value)
                if (!slugEdited) setSlug(slugOf(e.target.value))
              }}
            />
          </Field>
          <Field label={t(m.slug)} hint={<T msg={m.slugHint} values={{ token: `${token}.code`, mono }} />} error={errors.slug}>
            <Input
              className="mono"
              value={slug}
              maxLength={32}
              onChange={(e) => {
                setSlugEdited(true)
                setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ''))
              }}
            />
          </Field>
        </div>
        <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
          <legend className="lbl mb-1.5">{t(m.source)}</legend>
          <SourceCard value="upload" checked={source === 'upload'} onPick={setSource} title={t(m.srcUpload)} hint={t(m.srcUploadHint)} />
          {cards}
          {errors.storeId ? (
            <span className="text-[12.5px] text-bad-fg" role="alert">
              {errors.storeId}
            </span>
          ) : null}
        </fieldset>
        {store ? (
          <>
            <fieldset className="m-0 flex min-w-0 flex-col gap-3 border-0 p-0">
              <legend className="lbl mb-2">{t(m.discountIn, { store: store.name })}</legend>
              <div className="flex flex-col gap-1.5">
                <span className="lbl">{t(m.kind)}</span>
                <Segmented<Kind>
                  label={t(m.kind)}
                  value={kind}
                  onChange={setKind}
                  className="self-start"
                  items={[
                    { value: 'percentage', label: t(m.kindPercentage) },
                    { value: 'fixed_amount', label: t(m.kindFixed) },
                    { value: 'free_shipping', label: t(m.kindFree) },
                  ]}
                />
              </div>
              <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(200px,100%),1fr))' }}>
                {kind !== 'free_shipping' ? (
                  <Field label={t(m.value)} error={errors['settings.value']}>
                    <span className="flex items-center gap-2">
                      {kind === 'fixed_amount' ? <span className="k">{currencySymbol(currency)}</span> : null}
                      <Input inputMode="decimal" className="w-[80px]" value={value} onChange={(e) => setValue(e.target.value)} />
                      <span className="k">{kind === 'percentage' ? t(m.valuePercent) : t(m.valueAmount)}</span>
                    </span>
                  </Field>
                ) : null}
                <Field label={t(m.minimum)} error={errors['settings.minimumCents']}>
                  <span className="flex items-center gap-2">
                    <span className="k">{currencySymbol(currency)}</span>
                    <Input inputMode="decimal" className="min-w-0 flex-1" placeholder={t(m.minimumNone)} value={minimum} onChange={(e) => setMinimum(e.target.value)} />
                  </span>
                </Field>
              </div>
              <Field label={t(m.endsAfter)} error={errors['settings.endsDays']}>
                <span className="flex items-center gap-2">
                  <Input inputMode="numeric" className="w-[80px]" placeholder={t(m.endsNever)} value={ends} onChange={(e) => setEnds(e.target.value.replace(/\D/g, ''))} />
                  <span className="k">{t(m.days)}</span>
                </span>
              </Field>
              <Field label={t(m.appliesTo)} hint={t(m.eachCode)}>
                <Select value={collection} onChange={(e) => setCollection(e.target.value)} disabled={kind === 'free_shipping'}>
                  <option value="">{t(m.allProducts)}</option>
                  {collections.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.title}
                    </option>
                  ))}
                </Select>
              </Field>
            </fieldset>
            <Field label={t(m.prefix)} hint={t(m.prefixHint, { example: exampleCode(prefix.toUpperCase() || 'WELCOME-') })} error={errors.prefix}>
              <Input className="mono" value={prefix} maxLength={12} placeholder={PREFIX_EXAMPLE} onChange={(e) => setPrefix(e.target.value.toUpperCase().replace(/[^A-Z0-9-]/g, ''))} />
            </Field>
          </>
        ) : (
          <>
            <Field
              label={t(m.applyUrl)}
              hint={<T msg={m.applyUrlHint} values={{ placeholder: '{code}', token: `${token}.url`, mono }} />}
              error={errors['settings.applyUrl']}
            >
              <Input value={applyUrl} placeholder={APPLY_EXAMPLE} onChange={(e) => setApplyUrl(e.target.value)} />
            </Field>
            <span className="k text-[13px] font-normal text-ink-2">{t(m.uploadNext)}</span>
          </>
        )}
        <Checkbox
          checked={onePer}
          onChange={(e) => setOnePer(e.target.checked)}
          label={<b className="font-semibold">{t(m.onePerContact)}</b>}
          hint={t(m.onePerContactHint)}
        />
        {store ? <Note tone="warn">{t(m.storeWarn, { store: store.name })}</Note> : <Note tone="warn">{t(m.uploadWarn)}</Note>}
      </div>
    </Dialog>
  )
}

/** "$" for USD: the symbol before an amount field. */
function currencySymbol(currency: string): string {
  try {
    return (
      new Intl.NumberFormat('en', { style: 'currency', currency: currency || 'USD', currencyDisplay: 'narrowSymbol' }).formatToParts(0).find((p) => p.type === 'currency')?.value ??
      currency
    )
  } catch {
    return currency
  }
}
