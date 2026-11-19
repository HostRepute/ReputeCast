import { DropdownMenu } from 'radix-ui'
import { useT } from '@/i18n'
import { money } from '@/lib/format'
import type { AssistantModel } from './api'
import { AsstIcon } from './icons'
import { m } from './messages'

/** "$2", "$0.25": a price per million tokens without trailing zeros. */
function perMillion(v: number): string {
  return money(v, Number.isInteger(v) ? 0 : v * 10 === Math.round(v * 10) ? 1 : 2)
}

/** Why a model can't answer in a thread (assistant brief §3.2), or undefined. */
export function useReason() {
  const t = useT()
  return (model: AssistantModel): string | undefined => {
    if (model.usable) return undefined
    switch (model.reason) {
      case 'disconnected':
        return t(m.reasonDisconnected)
      case 'key_rejected':
        return t(m.reasonKeyRejected, { provider: model.providerName })
      case 'personal_keys_off':
        return t(m.reasonPersonalOff)
      default:
        return model.tools.state === 'unsupported' ? t(m.reasonToolsFailed) : t(m.reasonToolsUntested)
    }
  }
}

/** A model's second line: "Anthropic · workspace key · Tools · Images · Streaming · $2 in · $10 out per 1M tokens". */
export function useModelLine() {
  const t = useT()
  return (model: AssistantModel) =>
    [
      model.providerName,
      t(model.personal ? m.keyPersonal : m.keyWorkspace),
      ...(model.tools.state === 'supported' ? [t(m.capTools)] : []),
      ...(model.imageInput ? [t(m.capImages)] : []),
      t(m.capStreaming),
      model.prices.known ? t(m.price, { input: perMillion(model.prices.input), output: perMillion(model.prices.output) }) : t(m.priceUnknown),
    ].join(' · ')
}

/**
 * The model picker (AssistantPanel): the workspace's models, then the
 * person's own, each with its capabilities, price and whose key it is; a
 * model that can't answer says why and can't be picked.
 */
export function ModelPicker({
  models,
  value,
  onChange,
  open,
  onOpenChange,
}: {
  models: AssistantModel[]
  value: string | undefined
  onChange: (id: string) => void
  open?: boolean
  onOpenChange?: (open: boolean) => void
}) {
  const t = useT()
  const reason = useReason()
  const line = useModelLine()
  const current = models.find((x) => x.id === value)
  const groups: [string, AssistantModel[]][] = [
    [t(m.groupWorkspace), models.filter((x) => !x.personal)],
    [t(m.groupPersonal), models.filter((x) => x.personal)],
  ]
  const name = current?.modelName || current?.model || t(m.noModelPicked)
  return (
    <DropdownMenu.Root modal={false} open={open} onOpenChange={onOpenChange}>
      <DropdownMenu.Trigger asChild>
        <button type="button" className="asst-picker" aria-label={t(m.modelButton, { model: name })}>
          {name}
          <AsstIcon name="chevron" />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="asst-menu" align="start" sideOffset={6} collisionPadding={12} aria-label={t(m.modelMenu)}>
          <DropdownMenu.RadioGroup value={value ?? ''} onValueChange={onChange}>
            {groups
              .filter(([, list]) => list.length)
              .map(([label, list]) => (
                <DropdownMenu.Group key={label}>
                  <DropdownMenu.Label className="asst-menu-group">{label}</DropdownMenu.Label>
                  {list.map((x) => {
                    const why = reason(x)
                    return (
                      <DropdownMenu.RadioItem key={x.id} value={x.id} disabled={Boolean(why)} className="asst-menu-item">
                        <strong>{x.modelName || x.model}</strong>
                        <span className="k">
                          <DropdownMenu.ItemIndicator>
                            <AsstIcon name="check" />
                          </DropdownMenu.ItemIndicator>
                        </span>
                        <small>{line(x)}</small>
                        {why ? <small>{why}</small> : null}
                      </DropdownMenu.RadioItem>
                    )
                  })}
                </DropdownMenu.Group>
              ))}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
