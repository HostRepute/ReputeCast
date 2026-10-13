import { useEffect, useMemo, useState, type ChangeEvent } from 'react'
import { costRows, DEFAULT_INPUTS, PRICE_FIELDS, volumeLabel, type CostField, type CostInputs } from '../lib/cost'

/**
 * "What would your volume cost?": every price is editable. Results update as
 * you type; a short summary is announced to screen readers once typing stops.
 */
export function Cost() {
  const [inputs, setInputs] = useState<CostInputs>(DEFAULT_INPUTS)
  const rows = useMemo(() => costRows(inputs), [inputs])
  const volume = volumeLabel(inputs)
  const [announcement, setAnnouncement] = useState('')

  useEffect(() => {
    if (inputs === DEFAULT_INPUTS) return
    const t = window.setTimeout(() => {
      const cheapest = [...rows].sort((a, b) => a.value - b.value)[0]
      setAnnouncement(`Updated for ${volume} emails a month. Lowest: ${cheapest?.name} at ${cheapest?.cost}.`)
    }, 900)
    return () => window.clearTimeout(t)
  }, [inputs, rows, volume])

  const set = (key: CostField) => (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value
    setInputs((s) => ({ ...s, [key]: value }))
  }

  return (
    <section className="wrap flex flex-col gap-5 pt-10 pb-10" id="cost" aria-labelledby="cost-h">
      <div className="reveal flex flex-col gap-2.5">
        <h2 className="sec" id="cost-h">
          What would your volume cost?
        </h2>
        <p className="lead">
          Edit the volume and the prices. Defaults are example list prices for pay-as-you-go sending; plans, free tiers and dedicated IPs change the picture, so check
          each provider.
        </p>
      </div>
      <div className="card reveal flex flex-wrap items-start gap-7 p-6">
        <div className="flex flex-[1_1_300px] flex-col gap-3.5">
          <label className="flex flex-col gap-1.5 text-[14px] font-semibold">
            Emails per month
            <input className="in" value={inputs.vol} onChange={set('vol')} inputMode="numeric" autoComplete="off" />
          </label>
          <div className="grid grid-cols-[repeat(2,minmax(0,1fr))] gap-2.5">
            {PRICE_FIELDS.map((f) => (
              <label key={f.key} className="flex flex-col gap-1 text-[13px] font-semibold">
                {f.label}
                <input className="in" value={inputs[f.key]} onChange={set(f.key)} inputMode="decimal" autoComplete="off" />
              </label>
            ))}
          </div>
          <span className="k">ReputeCast itself is free. Add the server you run it on; a small VPS is usually enough for 100,000 contacts.</span>
        </div>
        <div className="flex flex-[1.2_1_380px] flex-col gap-2.5">
          <h3 className="m-0 text-[16px] font-bold">Monthly sending cost at {volume} emails</h3>
          <ul className="m-0 flex list-none flex-col gap-2.5 p-0">
            {rows.map((r) => (
              <li key={r.name} className="flex flex-col gap-[5px] border-t border-line-2 py-2">
                <div className="flex items-center justify-between text-[14.5px] max-md:flex-wrap">
                  <span className="flex items-center gap-2">
                    <span className="dot" style={{ color: r.color }} aria-hidden="true" />
                    {r.name}
                  </span>
                  <span className="n">{r.cost}</span>
                </div>
                <div className="bar" aria-hidden="true">
                  <i style={{ width: r.pct, background: r.color }} />
                </div>
                <span className="k">{r.note}</span>
              </li>
            ))}
          </ul>
          <span className="vh" role="status" aria-live="polite">
            {announcement}
          </span>
        </div>
      </div>
    </section>
  )
}
