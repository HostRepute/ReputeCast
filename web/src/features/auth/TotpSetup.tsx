import { useMutation } from '@tanstack/react-query'
import QRCode from 'qrcode'
import { useEffect, useId, useMemo, useRef, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'

/** The otpauth:// URL as a QR code, drawn as one SVG path. */
export function QrCode({ value, size = 132, label }: { value: string; size?: number; label: string }) {
  const path = useMemo(() => {
    const qr = QRCode.create(value, { errorCorrectionLevel: 'M' })
    const n = qr.modules.size
    let d = ''
    for (let y = 0; y < n; y++) {
      for (let x = 0; x < n; x++) {
        if (qr.modules.get(y, x)) d += `M${x} ${y}h1v1h-1z`
      }
    }
    return { d, n }
  }, [value])
  return (
    <svg width={size} height={size} viewBox={`0 0 ${path.n} ${path.n}`} role="img" aria-label={label} shapeRendering="crispEdges">
      <rect width={path.n} height={path.n} fill="#fff" />
      <path d={path.d} fill="#0B1220" />
    </svg>
  )
}

/**
 * Set up an authenticator app: scan the QR code or type the key, then
 * confirm with a code. Returns the new recovery codes (AccountStates,
 * "Set up an authenticator app", step 1 of 2).
 */
export function TotpSetupForm({
  onConfirmed,
  onCancel,
  confirmLabel = 'Confirm and continue',
  cancelLabel = 'Cancel',
}: {
  onConfirmed: (codes: string[]) => void
  onCancel?: () => void
  confirmLabel?: string
  cancelLabel?: string
}) {
  const begin = useMutation({ mutationFn: async () => (await http.POST('/api/v1/me/totp')).data! })
  const started = useRef(false)
  useEffect(() => {
    if (!started.current) {
      started.current = true
      begin.mutate()
    }
  }, [begin])
  const setup: Schemas['TOTPSetup'] | undefined = begin.data
  const [code, setCode] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const codeId = useId()
  const errId = useId()

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!setup) return
    const digits = code.replace(/\D/g, '')
    if (digits.length !== 6) {
      setError('Enter the 6 digits the app shows.')
      return
    }
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/me/totp/confirm', { body: { challenge: setup.challenge, code: digits } })
      onConfirmed(data?.codes ?? [])
    } catch (err) {
      if (err instanceof ApiError && err.code === 'challenge_expired') {
        setCode('')
        begin.mutate()
      }
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  if (begin.isError) {
    return (
      <div className="flex flex-col gap-3">
        <span className="err" role="alert">
          {errorMessage(begin.error)}
        </span>
        <div className="flex justify-end gap-2.5">
          {onCancel ? <Button onClick={onCancel}>{cancelLabel}</Button> : null}
          <Button variant="primary" onClick={() => begin.mutate()}>
            Try again
          </Button>
        </div>
      </div>
    )
  }

  return (
    <form className="flex flex-col gap-3.5" onSubmit={submit}>
      <div className="row items-start" style={{ gap: 18 }}>
        <div className="rounded-lg border border-line bg-white p-2">
          {setup ? <QrCode value={setup.url} label="QR code for the authenticator app" /> : <Skeleton w={132} h={132} r={4} />}
        </div>
        <div className="col" style={{ gap: 8, flex: '1 1 220px' }}>
          <span className="text-[13.5px]">Scan with 1Password, Google Authenticator, Authy or any TOTP app. Can't scan? Enter the key by hand:</span>
          <div className="row" style={{ gap: 8 }}>
            <span className="mono rounded-lg border border-dashed border-muted-line px-2.5 py-2 text-[14px] tracking-[.06em]">{setup ? setup.secret : '•••• •••• •••• ••••'}</span>
            <Button
              size="sm"
              disabled={!setup}
              onClick={async () => {
                if (!setup) return
                try {
                  await navigator.clipboard.writeText(setup.secret.replace(/\s/g, ''))
                  toast({ kind: 'success', title: 'Key copied' })
                } catch {
                  toast({ kind: 'error', title: "Couldn't copy", detail: 'Select the key and copy it by hand.' })
                }
              }}
            >
              Copy
            </Button>
          </div>
          <span className="k">{setup ? `Account ${setup.account} · issuer ${setup.issuer} · 30-second codes` : 'Preparing a key…'}</span>
        </div>
      </div>
      <div className="lbl">
        <label htmlFor={codeId}>Enter the 6-digit code the app shows</label>
        <input
          id={codeId}
          className={error ? 'in mono err' : 'in mono'}
          style={{ maxWidth: 220 }}
          value={code}
          onChange={(e) => setCode(e.target.value)}
          placeholder="123 456"
          inputMode="numeric"
          autoComplete="one-time-code"
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? errId : undefined}
        />
        {error ? (
          <span className="err" id={errId} role="alert">
            {error}
          </span>
        ) : null}
      </div>
      <div className="row justify-end" style={{ gap: 10 }}>
        {onCancel ? <Button onClick={onCancel}>{cancelLabel}</Button> : null}
        <Button type="submit" variant="primary" loading={busy} disabled={!setup}>
          {confirmLabel}
        </Button>
      </div>
    </form>
  )
}
