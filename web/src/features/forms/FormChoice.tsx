import { useId, useState } from 'react'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Table, Td, Th } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m } from './builder.messages'
import { maxOptions, minOptions, tagOf, type ChoiceOption, type FormField } from './steps'

/**
 * "Choice question" (FormsStates "Several steps"): the question, one or
 * several answers, whether it's required, and its answers, each with the
 * tag it adds to the contact. Apply changes the draft only.
 */
export function ChoiceDialog({ field, onApply, onClose }: { field: FormField; onApply: (f: FormField) => void; onClose: () => void }) {
  const t = useT()
  const id = useId()
  const [label, setLabel] = useState(field.label)
  const [multiple, setMultiple] = useState(Boolean(field.multiple))
  const [required, setRequired] = useState(field.required)
  const [options, setOptions] = useState<ChoiceOption[]>(() =>
    field.options?.length ? field.options.map((o) => ({ ...o })) : Array.from({ length: minOptions }, () => ({ label: '', tag: '' })),
  )
  const [errors, setErrors] = useState<{ question?: string; options?: string }>({})
  const setOption = (index: number, patch: Partial<ChoiceOption>) => setOptions((os) => os.map((o, i) => (i === index ? { ...o, ...patch } : o)))
  // The checks the server makes on save, said where they apply.
  const apply = () => {
    const out = options.map((o) => ({ label: o.label.trim(), tag: o.tag.trim().toLowerCase() }))
    const twice = out.find((o, i) => o.tag && out.findIndex((x) => x.tag === o.tag) !== i)
    const next = {
      question: label.trim() ? undefined : t(m.choiceNeedsQuestion),
      options: out.some((o) => !o.label || !o.tag) ? t(m.choiceNeedsAnswer) : twice ? t(m.choiceTagTwice, { tag: twice.tag }) : undefined,
    }
    setErrors(next)
    if (next.question || next.options) return
    onApply({ key: field.key, type: 'choice', label: label.trim(), multiple, required, options: out })
    onClose()
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.choiceTitle)}
      className="max-w-[560px]"
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" onClick={apply}>
            {t(m.apply)}
          </Button>
        </>
      }
    >
      <label className="lbl">
        {t(m.choiceQuestion)}
        <input
          className="in"
          maxLength={80}
          value={label}
          aria-invalid={errors.question ? true : undefined}
          aria-describedby={errors.question ? `${id}-q` : undefined}
          onChange={(e) => setLabel(e.target.value)}
          autoFocus
        />
        {errors.question ? (
          <span className="err" id={`${id}-q`} role="alert">
            {errors.question}
          </span>
        ) : null}
      </label>
      <label className="fb-check">
        <input type="checkbox" checked={multiple} onChange={(e) => setMultiple(e.target.checked)} />
        {t(m.choiceMultiple)}
      </label>
      <label className="fb-check">
        <input type="checkbox" checked={required} onChange={(e) => setRequired(e.target.checked)} />
        {t(common.required)}
      </label>
      <div className="rounded-lg border border-line">
        <Table caption={t(m.choiceAnswers)}>
          <thead>
            <tr>
              <Th>{t(m.optionLabel)}</Th>
              <Th>{t(m.optionTag)}</Th>
              <Th>
                <span className="vh">{t(m.remove)}</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {options.map((o, i) => (
              <tr key={i}>
                <Td>
                  <input
                    className="in min-w-[120px]"
                    maxLength={60}
                    aria-label={t(m.optionLabelN, { n: i + 1 })}
                    value={o.label}
                    onChange={(e) => {
                      const next = e.target.value
                      // The tag follows the label until someone writes their own.
                      setOption(i, o.tag === '' || o.tag === tagOf(o.label) ? { label: next, tag: tagOf(next) } : { label: next })
                    }}
                  />
                </Td>
                <Td>
                  <input className="in min-w-[120px]" maxLength={60} aria-label={t(m.optionTagN, { n: i + 1 })} value={o.tag} onChange={(e) => setOption(i, { tag: e.target.value })} />
                </Td>
                <Td>
                  <Button
                    size="sm"
                    aria-label={o.label.trim() ? t(m.removeOption, { label: o.label.trim() }) : t(m.removeOptionN, { n: i + 1 })}
                    disabledReason={options.length <= minOptions ? t(m.fewestOptions) : undefined}
                    onClick={() => setOptions((os) => os.filter((_, x) => x !== i))}
                  >
                    {t(m.remove)}
                  </Button>
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
      </div>
      <Button
        size="sm"
        className="self-start"
        disabledReason={options.length >= maxOptions ? t(m.mostOptions) : undefined}
        onClick={() => setOptions((os) => [...os, { label: '', tag: '' }])}
      >
        {t(m.addOption)}
      </Button>
      {errors.options ? (
        <span className="err" role="alert">
          {errors.options}
        </span>
      ) : null}
      <span className="k">{t(m.choiceHelp)}</span>
    </Dialog>
  )
}
