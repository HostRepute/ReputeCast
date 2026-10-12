import { useId, useState } from 'react'
import { useInstanceMeta } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'

function codesText(codes: string[], email: string, instance: string) {
  const when = new Date().toISOString().slice(0, 10)
  return [
    `${instance} recovery codes for ${email}`,
    `Generated ${when}. Each code signs you in once if you lose your authenticator or passkey.`,
    '',
    ...codes,
    '',
    'Keep them somewhere safe, like a password manager. Generating new codes replaces these.',
    '',
  ].join('\n')
}

function fileName(instance: string) {
  return `${instance.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'reputecast'}-recovery-codes.txt`
}

/** Download .txt · Print · Copy all for a set of recovery codes. */
export function RecoveryCodeActions({ codes, email }: { codes: string[]; email: string }) {
  const { data: meta } = useInstanceMeta()
  const instance = meta?.instanceName ?? 'ReputeCast'
  const text = codesText(codes, email, instance)
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button
        size="sm"
        onClick={() => {
          const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }))
          const a = document.createElement('a')
          a.href = url
          a.download = fileName(instance)
          document.body.appendChild(a)
          a.click()
          a.remove()
          window.setTimeout(() => URL.revokeObjectURL(url), 1000)
        }}
      >
        Download .txt
      </Button>
      <Button
        size="sm"
        onClick={() => {
          // Print from a throwaway frame so only the codes are printed.
          const frame = document.createElement('iframe')
          frame.setAttribute('aria-hidden', 'true')
          frame.style.position = 'fixed'
          frame.style.width = '0'
          frame.style.height = '0'
          frame.style.border = '0'
          document.body.appendChild(frame)
          const doc = frame.contentDocument
          if (!doc) return
          const pre = doc.createElement('pre')
          pre.textContent = text
          pre.style.font = '14px/1.6 "IBM Plex Mono", ui-monospace, monospace'
          doc.body.appendChild(pre)
          frame.contentWindow?.focus()
          frame.contentWindow?.print()
          window.setTimeout(() => frame.remove(), 1000)
        }}
      >
        Print
      </Button>
      <Button
        size="sm"
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(codes.join('\n'))
            toast({ kind: 'success', title: 'Recovery codes copied' })
          } catch {
            toast({ kind: 'error', title: "Couldn't copy", detail: 'Select the codes and copy them by hand.' })
          }
        }}
      >
        Copy all
      </Button>
    </div>
  )
}

/** The codes grid (`.rc` on the boards). */
export function RecoveryCodeList({ codes }: { codes: string[] }) {
  return (
    <div className="rc-box" role="list" aria-label="Recovery codes">
      {codes.map((c) => (
        <span role="listitem" key={c}>
          {c}
        </span>
      ))}
    </div>
  )
}

/**
 * "Save your recovery codes": shown once, with download, print and copy,
 * and a confirmation before finishing (AccountStates, step 2 of 2).
 */
export function RecoveryCodesPanel({
  codes,
  email,
  onFinish,
  finishLabel = 'Finish',
  busy,
}: {
  codes: string[]
  email: string
  onFinish: () => void
  finishLabel?: string
  busy?: boolean
}) {
  const [saved, setSaved] = useState(false)
  const id = useId()
  return (
    <>
      <Note tone="warn" label="Shown once">
        Each code signs you in once if you lose your authenticator. Store them somewhere safe; you won't see them again.
      </Note>
      <RecoveryCodeList codes={codes} />
      <RecoveryCodeActions codes={codes} email={email} />
      <label className="flex items-center gap-2.5 text-[13.5px]" htmlFor={id}>
        <input type="checkbox" id={id} checked={saved} onChange={(e) => setSaved(e.target.checked)} />I have saved these codes
      </label>
      <div className="flex justify-end">
        <Button variant="primary" loading={busy} disabledReason={saved ? undefined : 'Confirm that you saved the codes first'} onClick={onFinish}>
          {finishLabel}
        </Button>
      </div>
    </>
  )
}
