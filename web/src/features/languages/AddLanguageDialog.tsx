import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { useDeferredValue, useId, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { languagesKey } from './LanguagesCard'
import { md } from './messages'

type Option = Schemas['CatalogLanguage']

/** How many languages the empty search shows: the ones contacts already have. */
const SUGGESTED = 8

/**
 * Add a language (LanguageStates › Add language): search the catalog by
 * name, own name or code. Each option says how many contacts have it and
 * whether the built-in page labels exist. Right-to-left languages are
 * listed but can't be added yet.
 */
export function AddLanguageDialog({ open, onOpenChange, fallback }: { open: boolean; onOpenChange: (open: boolean) => void; fallback: string }) {
  const { slug } = useWorkspace()
  const t = useT()
  const qc = useQueryClient()
  const name = useId()
  const [query, setQuery] = useState('')
  const deferred = useDeferredValue(query.trim())
  const [picked, setPicked] = useState<string>()
  const catalog = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/languages/catalog',
    { params: { path: { workspace: slug }, query: { q: deferred } } },
    { placeholderData: keepPreviousData },
  )
  const add = $api.useMutation('post', '/api/v1/workspaces/{workspace}/languages')
  const all = (catalog.data?.items ?? []).filter((l) => !l.added)
  const options = deferred ? all : all.filter((l) => l.contacts > 0).slice(0, SUGGESTED)
  const selectable = options.filter((l) => !l.rtl)
  const chosen = selectable.find((l) => l.code === picked) ?? selectable[0]
  const onlyRtl = options.length > 0 && selectable.length === 0
  const submit = () => {
    if (!chosen) return
    add.mutate(
      { params: { path: { workspace: slug } }, body: { code: chosen.code } },
      {
        onSuccess: (out) => {
          qc.setQueryData(languagesKey(slug), out)
          toast({ title: t(md.added, { language: chosen.name }), detail: t(md.addedDetail, { language: chosen.name, fallback }) })
          onOpenChange(false)
        },
      },
    )
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(md.eyebrow)}</span>}
      title={t(md.addTitle)}
      hideClose
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(md.cancel)}</Button>
          <Button
            variant="primary"
            loading={add.isPending}
            disabled={!chosen}
            onClick={submit}
          >
            {chosen ? t(md.addButton, { language: chosen.name }) : t(md.addGeneric)}
          </Button>
        </>
      }
    >
      <label className="lbl">
        {t(md.searchLabel)}
        <input
          className="in"
          type="text"
          value={query}
          placeholder={t(md.searchPlaceholder)}
          aria-label={t(md.searchAria)}
          autoComplete="off"
          onChange={(e) => setQuery(e.target.value)}
        />
      </label>
      {options.length > 0 ? (
        <div className="flex flex-col gap-1.5" role="radiogroup" aria-label={t(md.resultsLabel)} style={{ maxHeight: 360, overflowY: 'auto' }}>
          {options.map((l) => (
            <CatalogOption
              key={l.code}
              l={l}
              name={name}
              on={!l.rtl && chosen?.code === l.code}
              alone={onlyRtl && options.length === 1}
              fallback={fallback}
              onPick={() => setPicked(l.code)}
            />
          ))}
        </div>
      ) : catalog.isPending ? null : (
        <span className="k" role="status">
          {deferred ? t(md.noMatch, { query: deferred }) : t(md.noSuggestions)}
        </span>
      )}
      {onlyRtl ? (
        <div className="note n" role="note">
          <Icon name="info" size={16} />
          <div style={{ flex: '1 1 320px', minWidth: 0 }}>{t(md.rtlNote, { language: options[0]?.name ?? '' })}</div>
        </div>
      ) : chosen ? (
        <span className="k">
          {chosen.regions.length
            ? t(md.baseNote, { code: chosen.code, regions: formatList(chosen.regions.slice(0, 2)) })
            : t(md.baseNoteNoRegions)}
        </span>
      ) : null}
      {add.isError ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} />
          <div style={{ flex: '1 1 320px', minWidth: 0 }}>
            {t(md.addFailed)} {errorMessage(add.error)}
          </div>
        </div>
      ) : null}
    </Dialog>
  )
}

function CatalogOption({
  l,
  name,
  on,
  alone,
  fallback,
  onPick,
}: {
  l: Option
  name: string
  on: boolean
  /** The only result: the refusal names who gets the fallback. */
  alone: boolean
  fallback: string
  onPick: () => void
}) {
  const t = useT()
  const single = l.contactCodes.length === 1 && l.contactCodes[0] !== l.code ? l.contactCodes[0] : undefined
  const contacts = single
    ? t(md.contactsWithCodes, { language: l.name, count: count(l.contacts), code: single })
    : t(md.contactsWith, { language: l.name, count: count(l.contacts) })
  const labels = l.rtl ? t(md.rtl) : l.builtinLabels ? t(md.labelsReady) : on ? t(md.noLabelsFirst) : t(md.noLabels)
  return (
    <label className={`lng-rad${on ? ' on' : ''}${l.rtl ? ' off' : ''}`} style={{ alignItems: 'center' }} aria-disabled={l.rtl || undefined}>
      <input type="radio" name={name} checked={on} disabled={l.rtl} onChange={onPick} />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex flex-wrap items-center gap-2">
          <b style={{ fontWeight: 600 }}>{l.name}</b>
          <span className="k">
            {l.native && l.native !== l.name ? (
              <>
                <span lang={l.code} dir={l.rtl ? 'rtl' : undefined}>
                  {l.native}
                </span>
                {' · '}
              </>
            ) : null}
            <span className="mono">{l.code}</span>
          </span>
        </span>
        <span className="k">{t(md.optionDetail, { contacts, labels })}</span>
        {l.rtl ? (
          <span className="k" style={{ color: 'var(--ink-nav)' }}>
            {alone ? t(md.rtlRefusalLong, { language: l.name, fallback }) : t(md.rtlRefusal)}
          </span>
        ) : null}
      </span>
    </label>
  )
}
