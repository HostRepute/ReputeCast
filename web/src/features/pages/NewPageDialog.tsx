import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Dialog as RDialog } from 'radix-ui'
import { useId, useRef, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { builderLink, NAME_MAX, pagesPath, SLUG_MAX, SLUG_RE, siteAddress, slugify, typedSlug, type Overview, type TemplateKey } from './format'
import { m } from './messages'
import { TEMPLATES, templateInfo } from './templates'
import './pages.css'

type Errors = { name?: string; slug?: string; form?: string }

/** Why "Start with About" couldn't create the page: the address's problem and a free one. */
export interface StartProblem {
  slug?: string
  suggestion?: string
}

/**
 * New page (PagesStates · the template picker): six templates with their
 * thumbnails, the page's name and address, and beside the dialog what the
 * template adds and what happens after "Create page". Creating starts a
 * draft and opens it in the page builder; an address already in use comes
 * back with a free one to use instead.
 */
export function NewPageDialog({
  d,
  template,
  initialName = '',
  initialProblem,
  onTemplate,
  onClose,
}: {
  d: Overview
  template: TemplateKey
  /** The name to start with ("About" after "Start with About" found /about taken). */
  initialName?: string
  initialProblem?: StartProblem
  onTemplate: (key: TemplateKey) => void
  onClose: () => void
}) {
  const t = useT()
  const { slug: workspace } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const id = useId()
  const nameRef = useRef<HTMLInputElement>(null)
  const slugRef = useRef<HTMLInputElement>(null)
  const [name, setName] = useState(initialName)
  const [slug, setSlug] = useState(() => slugify(initialName))
  const [slugTouched, setSlugTouched] = useState(false)
  const [errors, setErrors] = useState<Errors>(() => ({ slug: initialProblem?.slug }))
  const [suggestion, setSuggestion] = useState(initialProblem?.suggestion ?? '')
  const [busy, setBusy] = useState(false)
  const info = templateInfo(template)
  const address = slug ? siteAddress(d.host, slug) : ''

  const changeName = (v: string) => {
    setName(v)
    if (errors.name || errors.form) setErrors((e) => ({ ...e, name: undefined, form: undefined }))
    if (!slugTouched) {
      setSlug(slugify(v))
      setSuggestion('')
      if (errors.slug) setErrors((e) => ({ ...e, slug: undefined }))
    }
  }
  const changeSlug = (v: string) => {
    setSlugTouched(true)
    setSlug(typedSlug(v))
    setSuggestion('')
    setErrors((e) => ({ ...e, slug: undefined, form: undefined }))
  }
  const takeSuggestion = () => {
    changeSlug(suggestion)
    slugRef.current?.focus()
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (busy) return
    const n = name.trim()
    const s = slug.replace(/^-+|-+$/g, '')
    const next: Errors = {}
    if (!n) next.name = t(m.npNameRequired)
    if (!s) next.slug = t(m.npSlugRequired)
    else if (!SLUG_RE.test(s)) next.slug = t(m.npSlugInvalid)
    if (next.name || next.slug) {
      setErrors(next)
      ;(next.name ? nameRef : slugRef).current?.focus()
      return
    }
    setBusy(true)
    setErrors({})
    try {
      const { data } = await http.POST(pagesPath, { params: { path: { workspace } }, body: { template, name: n, slug: s } })
      void qc.invalidateQueries({ queryKey: ['get', pagesPath] })
      if (data) void navigate(builderLink(workspace, data.id) as never)
      else onClose()
    } catch (err) {
      setBusy(false)
      if (err instanceof ApiError) {
        if (err.code === 'slug_taken') {
          const free = err.problem.meta?.suggestion
          setSuggestion(typeof free === 'string' ? free : '')
          setErrors({ slug: err.message })
          slugRef.current?.focus()
          return
        }
        const at = (field: string) => err.problem.errors?.find((x) => x.location?.endsWith(field))?.message
        const fieldErrors: Errors = { name: at('name'), slug: at('slug') }
        if (fieldErrors.name || fieldErrors.slug) {
          setErrors(fieldErrors)
          ;(fieldErrors.name ? nameRef : slugRef).current?.focus()
          return
        }
      }
      setErrors({ form: errorMessage(err) })
    }
  }

  const nameErr = `${id}-name-err`
  const slugErr = `${id}-slug-err`
  const prefix = `${id}-prefix`
  return (
    <RDialog.Root open onOpenChange={(open) => !open && !busy && onClose()}>
      <RDialog.Portal>
        <RDialog.Overlay className="ovl">
          <RDialog.Content
            className="pg-np"
            aria-describedby={undefined}
            onOpenAutoFocus={(e) => {
              e.preventDefault()
              ;(initialProblem?.slug ? slugRef : nameRef).current?.focus()
            }}
          >
            <form className="dlg" onSubmit={(e) => void submit(e)} noValidate>
              <div className="dh">
                <span className="k">{t(m.npCrumb)}</span>
                <RDialog.Title className="pg-np-title">{t(m.npTitle)}</RDialog.Title>
              </div>
              <div className="db">
                <fieldset>
                  <legend className="lbl">{t(m.npTemplates)}</legend>
                  <div className="pg-tpls">
                    {TEMPLATES.map((x) => (
                      <button key={x.key} type="button" className={cn('pg-tplc', x.key === template && 'on')} aria-pressed={x.key === template} onClick={() => onTemplate(x.key)}>
                        <Thumbnail wires={x.wires} />
                        <span className="nm">{t(x.name)}</span>
                        <span className="k ds">{t(x.line)}</span>
                      </button>
                    ))}
                  </div>
                </fieldset>
                <div className="pg-np-fields">
                  <div className="lbl">
                    <label htmlFor={`${id}-name`}>{t(m.npName)}</label>
                    <input
                      ref={nameRef}
                      id={`${id}-name`}
                      className={cn('in', errors.name && 'err')}
                      value={name}
                      maxLength={NAME_MAX}
                      autoComplete="off"
                      aria-invalid={errors.name ? true : undefined}
                      aria-describedby={errors.name ? nameErr : undefined}
                      onChange={(e) => changeName(e.target.value)}
                    />
                    {errors.name ? (
                      <span className="err" id={nameErr} role="alert">
                        {errors.name}
                      </span>
                    ) : null}
                  </div>
                  <div className="lbl">
                    <label htmlFor={`${id}-slug`}>{t(m.npAddress)}</label>
                    <div className="pg-addr">
                      <span className="k mono" id={prefix} title={siteAddress(d.host, '')}>
                        {siteAddress(d.host, '')}
                      </span>
                      <input
                        ref={slugRef}
                        id={`${id}-slug`}
                        className={cn('in mono', errors.slug && 'err')}
                        value={slug}
                        maxLength={SLUG_MAX}
                        autoComplete="off"
                        spellCheck={false}
                        aria-invalid={errors.slug ? true : undefined}
                        aria-describedby={[prefix, errors.slug ? slugErr : ''].filter(Boolean).join(' ')}
                        onChange={(e) => changeSlug(e.target.value)}
                      />
                    </div>
                    {errors.slug ? (
                      <span className="err" id={slugErr} role="alert">
                        {errors.slug}{' '}
                        {suggestion ? (
                          <button type="button" className="pg-suggest" onClick={takeSuggestion}>
                            {t(m.npUseSuggestion, { slug: suggestion })}
                          </button>
                        ) : null}
                      </span>
                    ) : null}
                  </div>
                </div>
                <span className="k">{t(m.npHint)}</span>
                {errors.form ? <Note tone="bad">{errors.form}</Note> : null}
              </div>
              <div className="df">
                <Button onClick={onClose} disabled={busy}>
                  {t(m.npCancel)}
                </Button>
                <Button type="submit" variant="primary" loading={busy} busyLabel={t(m.npCreating)}>
                  {t(m.npCreate)}
                </Button>
              </div>
            </form>
            <div className="pg-np-side">
              <div className="note i pg-note" role="note">
                <Icon name="info" size={16} className="mt-px flex-none text-[#1D4ED8]" />
                <div className="flex min-w-0 flex-col gap-1">
                  <span className="font-bold">{t(m.npAdds, { template: t(info.name) })}</span>
                  <div>{t(info.adds)}</div>
                </div>
              </div>
              <section className="card">
                <div className="sec">
                  <h2 className="h2">{t(m.npAfter)}</h2>
                </div>
                <ol className="pg-steps">
                  <li className="pg-ev">
                    <span className="ic now" aria-hidden="true">
                      <Icon name="arrowRight" size={14} strokeWidth={1.9} />
                    </span>
                    <span className="tx">
                      <span className="tt">{t(m.npStepOpens)}</span>
                      <span className="k">{address ? t(m.npStepOpensAt, { address }) : t(m.npStepOpensDraft)}</span>
                    </span>
                  </li>
                  <li className="pg-ev">
                    <span className="ic" aria-hidden="true" />
                    <span className="tx">
                      <span className="tt later">{t(info.step)}</span>
                      <span className="k">{t(info.stepSub)}</span>
                    </span>
                  </li>
                  <li className="pg-ev">
                    <span className="ic" aria-hidden="true" />
                    {d.canPublish || d.publishers.length === 0 ? (
                      <span className="tx">
                        <span className="tt later">{t(m.npStepPublish)}</span>
                        <span className="k">{t(m.npStepPublishSub)}</span>
                      </span>
                    ) : (
                      <span className="tx">
                        <span className="tt later">{t(m.npStepAsk)}</span>
                        <span className="k">{t(m.npStepAskSub, { names: orList(d.publishers) })}</span>
                      </span>
                    )}
                  </li>
                </ol>
              </section>
            </div>
          </RDialog.Content>
        </RDialog.Overlay>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

/** A template's thumbnail: its blocks as a wireframe on the publisher's ground. */
function Thumbnail({ wires }: { wires: (typeof TEMPLATES)[number]['wires'] }) {
  const t = useT()
  return (
    <span className="th" aria-hidden="true">
      {wires.length === 0 ? <span className="pg-blank">{t(m.blankThumb)}</span> : null}
      {wires.map(([kind, left, top, width, height], i) => (
        <span key={i} className={cn('pg-wf', kind)} style={{ left: `${left}%`, top, width: `${width}%`, height }} />
      ))}
    </span>
  )
}
