import { Dialog as RDialog } from 'radix-ui'
import { useId, useState } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import type { Check, Fix } from './logic'
import { m } from './messages'

export type FixPick = { check: Check; fix: Fix }

/**
 * Review the fixes (SendReadinessFixes, SR-13): each one-click fix with
 * what it changes, picked to apply together. Focus moves into the panel and
 * back to the button that opened it; applying saves a new version (undo it
 * from Version history) and re-runs the checks, which lock sending.
 */
export function FixReview({
  open,
  onOpenChange,
  campaign,
  fixes,
  reasonFor,
  onApply,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  campaign: string
  fixes: FixPick[]
  reasonFor: (f: Fix) => string | undefined
  onApply: (picked: FixPick[]) => Promise<void>
}) {
  const t = useT()
  const id = useId()
  const [picked, setPicked] = useState<string[]>(() => fixes.filter((x) => !reasonFor(x.fix)).map(keyOf))
  const [busy, setBusy] = useState(false)
  const chosen = fixes.filter((x) => picked.includes(keyOf(x)))
  const apply = async () => {
    if (!chosen.length || busy) return
    setBusy(true)
    try {
      await onApply(chosen)
      onOpenChange(false)
    } finally {
      setBusy(false)
    }
  }
  return (
    <RDialog.Root open={open} onOpenChange={(v) => !busy && onOpenChange(v)}>
      <RDialog.Portal>
        <RDialog.Overlay className="ovl rv-drawer-ovl">
          <RDialog.Content className="rv-drawer" aria-describedby={`${id}-d`}>
            <div className="rv-drawer-h">
              <div className="flex min-w-0 flex-col gap-0.5">
                <RDialog.Title className="h2">{t(m.reviewFixes, { count: fixes.length })}</RDialog.Title>
                <RDialog.Description id={`${id}-d`} className="k">
                  {t(m.fixReviewLead, { name: campaign })}
                </RDialog.Description>
              </div>
              <RDialog.Close className="btn sm">{t(m.close)}</RDialog.Close>
            </div>
            <div className="rv-drawer-b">
              {fixes.map((x) => {
                const k = keyOf(x)
                const reason = reasonFor(x.fix)
                return (
                  <section key={k} className="rv-fxr" aria-labelledby={`${id}-${k}`}>
                    <label className="rv-fxr-h">
                      <input
                        type="checkbox"
                        checked={picked.includes(k)}
                        disabled={Boolean(reason) || busy}
                        onChange={(e) => setPicked((v) => (e.target.checked ? [...v, k] : v.filter((y) => y !== k)))}
                      />
                      <span className="flex min-w-0 flex-col gap-0.5">
                        <span className="font-semibold" id={`${id}-${k}`}>
                          {x.fix.label}
                        </span>
                        <span className="k">{t(m.fixesWhat, { title: x.check.flatTitle || x.check.title })}</span>
                        {reason ? <span className="k">{reason}</span> : null}
                      </span>
                    </label>
                    {x.fix.effects?.length ? (
                      <dl className="rv-tdl rv-fxr-e">
                        {x.fix.effects.map((e, i) => (
                          <div key={i} className="contents">
                            <dt>{e.label}</dt>
                            <dd>
                              {e.before ? (
                                <>
                                  <span className="rv-del mono">− {e.before}</span>
                                  <span className="rv-ins mono">+ {e.value}</span>
                                </>
                              ) : (
                                e.value
                              )}
                            </dd>
                          </div>
                        ))}
                      </dl>
                    ) : x.fix.note ? (
                      <span className="k">{x.fix.note}</span>
                    ) : null}
                  </section>
                )
              })}
              <section className="rv-fxr-next" aria-labelledby={`${id}-next`}>
                <h3 className="h3" id={`${id}-next`}>
                  {t(m.whatNext)}
                </h3>
                <ol>
                  <li>{t(m.whatNextVersion)}</li>
                  <li>{t(m.whatNextLocked)}</li>
                  <li>{t(m.whatNextBack)}</li>
                </ol>
              </section>
            </div>
            <div className="rv-drawer-f">
              <span className="k">{t(m.fixesUndo)}</span>
              <div className="flex flex-wrap justify-end gap-2.5">
                <RDialog.Close className="btn">{t(m.cancel)}</RDialog.Close>
                <Button variant="primary" loading={busy} disabledReason={chosen.length ? undefined : t(m.pickAFix)} onClick={() => void apply()}>
                  {t(m.applyFixes, { count: chosen.length })}
                </Button>
              </div>
            </div>
          </RDialog.Content>
        </RDialog.Overlay>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

function keyOf(x: FixPick): string {
  return `${x.check.key}:${x.fix.action}`
}

/** "Warn" or "Block" for a fix's check. */
export function FixBadge({ check }: { check: Check }) {
  return <Badge tone={check.status === 'block' ? 'bad' : 'warn'}>{check.status}</Badge>
}
