import { Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, OptionCard } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { useRefreshDigest } from './shared'
import { TextOption, useCreateText, useTextOption } from '@/features/sms/NewCampaign'

/**
 * "New campaign": Regular continues to the composer's template gallery as
 * before; Recurring digest creates a digest from a connected WordPress site
 * or a feed address and opens its settings (DigestSetup, 5.1).
 */
export function NewCampaignDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const canSend = useCan('campaigns:send')
  const refresh = useRefreshDigest()
  const sites = $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests/sites', { params: { path: { workspace: slug } } }, { enabled: open })
  const list = sites.data?.items ?? []
  const [kind, setKind] = useState<'regular' | 'recurring' | 'sms'>('regular')
  // Phase 8: a text message, once texting is set up.
  const texts = useTextOption()
  const createText = useCreateText()
  const [chosen, setSource] = useState<string>('')
  // A connected site is picked by default; without one, a feed address.
  const source = chosen || list[0]?.id || 'url'
  const [address, setAddress] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const submit = async () => {
    if (kind === 'regular') {
      onOpenChange(false)
      void navigate({ to: '/w/$workspace/campaigns/new', params: { workspace: slug } })
      return
    }
    if (kind === 'sms') {
      setBusy(true)
      try {
        await createText()
        onOpenChange(false)
      } catch (e) {
        toast({ kind: 'error', title: t(m.createFailed), detail: errorMessage(e) })
      } finally {
        setBusy(false)
      }
      return
    }
    setError(undefined)
    setBusy(true)
    try {
      const body = source !== 'url' ? { source: 'wordpress' as const, siteId: source } : { source: 'url' as const, feedUrl: address.trim() }
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/digests', { params: { path: { workspace: slug } }, body })
      refresh()
      onOpenChange(false)
      void navigate({ to: '/w/$workspace/campaigns/digests/$digestId/settings', params: { workspace: slug, digestId: data!.id } })
    } catch (e) {
      setError(errorMessage(e))
      toast({ kind: 'error', title: t(m.createFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }

  const urlPicked = source === 'url' || list.length === 0
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t(m.pickerTitle)}
      size="sm"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} busyLabel={t(m.creating)} disabled={kind === 'recurring' && urlPicked && !address.trim()} onClick={() => void submit()}>
            {t(m.continue)}
          </Button>
        </>
      }
    >
      <fieldset className="flex flex-col gap-2.5">
        <legend className="mb-2 text-[13px] font-semibold text-ink-2">{t(m.pickerQuestion)}</legend>
        <OptionCard name="campaign-kind" value="regular" checked={kind === 'regular'} onChange={() => setKind('regular')} title={t(m.regular)} description={t(m.regularBody)}>
          <span className="k">{t(m.regularHint)}</span>
        </OptionCard>
        <OptionCard name="campaign-kind" value="recurring" checked={kind === 'recurring'} onChange={() => setKind('recurring')} title={t(m.recurring)} description={t(m.recurringBody)}>
          {kind === 'recurring' ? (
            <div className="mt-2 flex flex-col gap-2">
              <span className="text-[13px] text-ink-2">
                {list.length === 1 ? t(m.recurringNeedsSite, { site: list[0]!.host }) : list.length > 1 ? t(m.recurringNeedsSites) : t(m.recurringNeeds)}
              </span>
              {list.length > 0 ? (
                <div className="flex flex-col gap-2" role="radiogroup" aria-label={t(m.feedLabel)}>
                  {list.map((s) => (
                    <OptionCard
                      key={s.id}
                      name="digest-source"
                      value={s.id}
                      checked={source === s.id}
                      onChange={setSource}
                      title={s.host}
                      description={s.items ? t(m.siteMeta, { date: dayMonthDate(s.connectedAt), count: s.items }) : t(m.siteMetaUnread, { date: dayMonthDate(s.connectedAt) })}
                    />
                  ))}
                  <OptionCard name="digest-source" value="url" checked={source === 'url'} onChange={setSource} title={t(m.anotherFeed)} />
                </div>
              ) : null}
              {urlPicked ? (
                <Field
                  label={t(m.feedOrPage)}
                  error={error}
                  hint={
                    list.length === 0 ? (
                      <T msg={m.feedOrPageHint} values={{ link: (c) => <Link to="/w/$workspace/settings/integrations" params={{ workspace: slug }}>{c}</Link> }} />
                    ) : undefined
                  }
                >
                  <Input type="url" inputMode="url" placeholder="https://blog.example.com" value={address} onChange={(e) => setAddress(e.target.value)} autoComplete="url" />
                </Field>
              ) : null}
              {!canSend ? <Note tone="info">{t(m.draftModeHint)}</Note> : null}
            </div>
          ) : null}
        </OptionCard>
        {texts ? <TextOption checked={kind === 'sms'} onChange={() => setKind('sms')} /> : null}
      </fieldset>
    </Dialog>
  )
}
