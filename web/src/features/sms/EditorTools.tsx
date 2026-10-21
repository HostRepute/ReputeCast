import { useState } from 'react'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { useT, type MessageDescriptor } from '@/i18n'
import { m } from './composer.messages'
import { findLinks } from './measure'

/** Personalize and Insert link. */
export function EditorTools({ body, setBody }: { body: string; setBody: (b: string) => void }) {
  const t = useT()
  const [linking, setLinking] = useState(false)
  const [url, setUrl] = useState('')
  const [error, setError] = useState<string>()
  const insert = (s: string) => setBody(`${body.trimEnd()} ${s}`.trimStart())
  const fields: { key: string; label: MessageDescriptor; fallback: string }[] = [
    { key: 'first_name', label: m.fieldFirstName, fallback: 'there' },
    { key: 'last_name', label: m.fieldLastName, fallback: '' },
    { key: 'company', label: m.fieldCompany, fallback: 'your team' },
  ]
  return (
    <div className="flex flex-wrap gap-1.5">
      <Menu
        trigger={
          <button type="button" className="btn sm">
            <span className="mono text-[11px] font-semibold">{'{ }'}</span>
            {t(m.personalize)}
          </button>
        }
      >
        {fields.map((f) => (
          <MenuItem key={f.key} onSelect={() => insert(f.fallback ? `{{ ${f.key} | default: "${f.fallback}" }}` : `{{ ${f.key} }}`)}>
            {t(f.label)}
            {f.fallback ? <span className="k"> · {t(m.fieldFallback, { fallback: f.fallback })}</span> : null}
          </MenuItem>
        ))}
      </Menu>
      <button type="button" className="btn sm" onClick={() => setLinking(true)}>
        <Icon name="link" size={13} />
        {t(m.insertLink)}
      </button>
      {linking ? (
        <Dialog
          open
          onOpenChange={(o) => !o && setLinking(false)}
          title={t(m.insertLink)}
          size="sm"
          footer={
            <>
              <Button onClick={() => setLinking(false)}>{t(m.close)}</Button>
              <Button
                variant="primary"
                onClick={() => {
                  const u = url.trim()
                  if (!findLinks(u.includes('/') ? u : `${u}/`).length) {
                    setError(t(m.linkInvalid))
                    return
                  }
                  insert(u)
                  setUrl('')
                  setLinking(false)
                }}
              >
                {t(m.linkInsert)}
              </Button>
            </>
          }
        >
          <Field label={t(m.linkPrompt)} error={error}>
            <Input className="mono" value={url} onChange={(e) => setUrl(e.target.value)} placeholder={'acme.io/summit'} autoFocus />
          </Field>
        </Dialog>
      ) : null}
    </div>
  )
}
