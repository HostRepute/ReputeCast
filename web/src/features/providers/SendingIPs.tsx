import { useState } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, OptionCard, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { ago, count, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { Fact, type ProviderCard } from './common'
import { mo } from './overview.messages'
import { useRefreshProviders } from './ProviderIncidents'

/** How many sending IPs a person lists by hand (providers.MaxSendingIPs on the server). */
export const maxSendingIPs = 64

/** The entries of what was typed or pasted: one per line, or separated by commas, semicolons or spaces. */
export function splitIPs(text: string): string[] {
  return text
    .split(/[\s,;]+/)
    .map((x) => x.trim())
    .filter(Boolean)
}

/** Where a provider's sending IPs come from, in a sentence. */
function sourceLine(t: ReturnType<typeof useT>, card: ProviderCard): string {
  const src = card.ipSource
  if (src.from === 'provider' && src.error) return t(mo.ipsCheckFailed, { provider: card.name, ago: ago(src.errorAt), error: src.error })
  if (src.from === 'provider') return t(mo.ipsFromProvider, { provider: card.name, checked: src.checkedAt ? 'yes' : 'no', ago: ago(src.checkedAt) })
  if (src.from === 'manual') return t(mo.ipsSetBy, { hasName: src.setBy ? 'yes' : 'no', name: src.setBy ?? '', date: src.setAt ? shortDate(src.setAt) : '' })
  return t(mo.ipsNotReported, { provider: card.name })
}

/**
 * A provider's sending IPs (review GS-10), which the canary's new-IP
 * trigger, Send Readiness's IP blocklist rows and warm-up read: the IPs or
 * the shared pool, where they come from (the provider's API, checked daily,
 * or who set them by hand) and the one action that fits: Edit for a
 * provider whose API doesn't list them, Check again after a check that
 * failed.
 */
export function SendingIPsFact({ card }: { card: ProviderCard }) {
  const t = useT()
  const reason = usePermissionReason('providers:manage')
  const [editing, setEditing] = useState(false)
  const src = card.ipSource
  const value =
    card.ipMode !== 'dedicated' ? (
      <span>{t(mo.ipsShared)}</span>
    ) : card.ips.length ? (
      <span className="mono">{t(mo.ipsDedicated, { ips: card.ips.join(', ') })}</span>
    ) : (
      <span>{t(mo.ipsNotSet)}</span>
    )
  return (
    <>
      <Fact className="pv-fact" k={t(mo.factIps)}>
        <span className="flex min-w-0 flex-col items-end gap-1 text-right">
          {value}
          <span className="flex flex-wrap items-center justify-end gap-2">
            <span className={cn('k', src.error && 'text-warn-fg')}>{sourceLine(t, card)}</span>
            {src.editable ? (
              <Button
                size="sm"
                className="max-md:min-h-11"
                disabledReason={reason}
                aria-label={t(mo.editIpsLabel, { name: card.name })}
                onClick={() => setEditing(true)}
              >
                {t(mo.editIps)}
              </Button>
            ) : src.error ? (
              <CheckIPsButton card={card} />
            ) : null}
          </span>
        </span>
      </Fact>
      {editing ? <SendingIPsDialog card={card} onClose={() => setEditing(false)} /> : null}
    </>
  )
}

/** Asks a provider whose API reports its sending IPs for them now, after a check that failed. */
function CheckIPsButton({ card }: { card: ProviderCard }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshProviders()
  const reason = usePermissionReason('providers:manage')
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/ips/check', {
    onSuccess: (got) => {
      void refresh()
      if (got.ipSource.error) {
        toast({ kind: 'error', title: t(mo.ipsCheckFailedTitle, { name: card.name }), detail: got.ipSource.error })
        return
      }
      toast({ kind: 'success', title: t(mo.ipsChecked, { name: card.name, count: got.ipMode === 'dedicated' ? got.ips.length : 0 }) })
    },
    onError: (e) => toast({ kind: 'error', title: t(mo.ipsCheckFailedTitle, { name: card.name }), detail: errorMessage(e) }),
  })
  return (
    <Button
      size="sm"
      className="max-md:min-h-11"
      disabledReason={reason}
      loading={check.isPending}
      aria-label={t(mo.checkIpsLabel, { name: card.name })}
      onClick={() => check.mutate({ params: { path: { workspace: slug, providerId: card.id } } })}
    >
      {t(mo.checkIpsAgain)}
    </Button>
  )
}

type Mode = 'dedicated' | 'shared'

/**
 * Sets the sending IPs of a provider whose API doesn't list them (a Postal
 * or SMTP server, a provider without an IP API): its dedicated IPs, one per
 * line, or its shared pool. The server checks every address; what it
 * refuses shows next to the field.
 */
export function SendingIPsDialog({ card, onClose }: { card: ProviderCard; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshProviders()
  const reason = usePermissionReason('providers:manage')
  const [mode, setMode] = useState<Mode>(card.ipMode === 'dedicated' ? 'dedicated' : 'shared')
  const [text, setText] = useState(card.ips.join('\n'))
  const [fieldError, setFieldError] = useState<string>()
  const [formError, setFormError] = useState<string>()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/providers/{providerId}/ips', {
    onSuccess: (saved) => {
      void refresh()
      onClose()
      toast({
        kind: 'success',
        title: t(mo.ipsSaved, { name: card.name }),
        detail: saved.ipMode === 'dedicated' ? t(mo.ipsSavedDedicated, { count: saved.ips.length }) : t(mo.ipsSavedShared),
      })
    },
    onError: (e) => {
      if (e instanceof ApiError && e.status === 422 && e.problem.errors?.some((x) => x.location?.startsWith('body.ips'))) setFieldError(errorMessage(e))
      else setFormError(errorMessage(e))
    },
  })
  const submit = () => {
    setFieldError(undefined)
    setFormError(undefined)
    const ips = mode === 'dedicated' ? splitIPs(text) : []
    if (mode === 'dedicated' && ips.length === 0) {
      setFieldError(t(mo.errIpsEnter))
      return
    }
    if (ips.length > maxSendingIPs) {
      setFieldError(t(mo.errIpsTooMany, { max: count(maxSendingIPs) }))
      return
    }
    save.mutate({ params: { path: { workspace: slug, providerId: card.id } }, body: { mode, ips } })
  }
  return (
    <Dialog
      open
      className="pv-ipdlg"
      onOpenChange={(o) => !o && onClose()}
      title={t(mo.ipsDialogTitle, { name: card.name })}
      description={t(mo.ipsDialogBody, { name: card.name })}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" disabledReason={reason} loading={save.isPending} onClick={submit}>
            {t(mo.ipsSave)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3"
        noValidate
        onSubmit={(e) => {
          e.preventDefault()
          submit()
        }}
      >
        {formError ? <Note tone="bad">{formError}</Note> : null}
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="vh">{t(mo.ipsModeLabel, { name: card.name })}</legend>
          <OptionCard
            name="ip-mode"
            value="dedicated"
            checked={mode === 'dedicated'}
            onChange={() => setMode('dedicated')}
            title={t(mo.ipsModeDedicated)}
            description={t(mo.ipsModeDedicatedHint)}
          />
          <OptionCard
            name="ip-mode"
            value="shared"
            checked={mode === 'shared'}
            onChange={() => {
              setMode('shared')
              setFieldError(undefined)
            }}
            title={t(mo.ipsModeShared)}
            description={t(mo.ipsModeSharedHint)}
          />
        </fieldset>
        {mode === 'dedicated' ? (
          <Field label={t(mo.ipsLabel)} hint={t(mo.ipsHint, { max: count(maxSendingIPs) })} error={fieldError}>
            <Textarea
              className="mono min-h-[96px]"
              rows={3}
              spellCheck={false}
              autoCapitalize="off"
              autoCorrect="off"
              value={text}
              onChange={(e) => {
                setText(e.target.value)
                setFieldError(undefined)
              }}
            />
          </Field>
        ) : fieldError ? (
          <span className="err" role="alert">
            {fieldError}
          </span>
        ) : null}
      </form>
    </Dialog>
  )
}
