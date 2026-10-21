import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { m } from './policy.messages'

type Cost = Schemas['ApprovalCostRule']

/**
 * Settings › Approvals: "An SMS or WhatsApp campaign is estimated to cost at
 * least $N", in the markup of the page's other rules.
 */
export function ChannelCostRule({ value, disabled, onChange }: { value?: Cost; disabled: boolean; onChange: (v: Cost) => void }) {
  const t = useT()
  const v = value ?? { enabled: false, amount: 50 }
  const [text, setText] = useState(String(v.amount))
  const [error, setError] = useState<string>()
  const commit = () => {
    const n = Number(text.replace(/[^\d.]/g, ''))
    if (!text.trim() || !Number.isFinite(n) || n <= 0 || n > 1_000_000) {
      setError(t(m.costInvalid))
      return
    }
    setError(undefined)
    if (n !== v.amount) onChange({ ...v, amount: n })
  }
  return (
    <>
      <label className="ap-rule">
        <input type="checkbox" className="tg" role="switch" checked={v.enabled} disabled={disabled} onChange={(e) => onChange({ ...v, enabled: e.target.checked })} />
        <span className="col min-w-0 flex-1 gap-px!">
          <span className="font-semibold">{t(m.costTitle)}</span>
          <span className="k whitespace-normal">{t(m.costDetail)}</span>
        </span>
        <span className="row flex-nowrap! gap-1.5!">
          <span className="k" aria-hidden="true">
            $
          </span>
          <input
            className="in mono w-[90px] max-md:min-w-0"
            inputMode="decimal"
            aria-label={t(m.costAmount)}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? 'ap-cost-err' : undefined}
            value={text}
            readOnly={disabled}
            onChange={(e) => setText(e.target.value)}
            onBlur={commit}
          />
        </span>
      </label>
      {error ? (
        <span className="err" id="ap-cost-err" role="alert">
          {error}
        </span>
      ) : null}
    </>
  )
}
