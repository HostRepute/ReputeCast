import { useState } from 'react'
import { ApiError, http } from '@/api/client'
import { Popover } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { useMe } from '@/lib/session'
import { useComposer } from './context'
import { TestLanguagePicker, testLanguageBody } from '@/features/languages/composerParts'
import { TestSplit } from '@/features/placement/EntryPoints'
import { t as tt, useT } from '@/i18n'
import { m } from './messages'

async function sendTest(slug: string, campaignId: string, to: string[], variant: string, language?: string) {
  const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/test', {
    params: { path: { workspace: slug, campaignId } },
    body: { to, variant: variant === 'A' ? undefined : (variant as 'B' | 'C'), ...testLanguageBody(language) },
  })
  return data!
}

function useSend() {
  const c = useComposer()
  const [busy, setBusy] = useState(false)
  const run = async (to: string[], language = c.campaign.language) => {
    setBusy(true)
    try {
      // Tests use the saved content.
      await c.save()
      const r = await sendTest(c.slug, c.campaign.id, to, c.variant, language)
      toast({ kind: 'success', title: tt(m.testSent, { to: r.sent.join(', ') }), detail: r.provider ? tt(m.testVia, { provider: r.provider }) : undefined })
      return true
    } catch (e) {
      toast({ kind: 'error', title: tt(m.testFailed), detail: e instanceof ApiError ? e.message : tt(m.checkConnection) })
      return false
    } finally {
      setBusy(false)
    }
  }
  return { busy, run }
}

/**
 * "Send test": members with campaigns:send pick recipients (themselves and
 * teammates); everyone else gets "Send test to me".
 */
export function SendTestButton({ disabled, toMe }: { disabled?: boolean; toMe?: boolean }) {
  const c = useComposer()
  const t = useT()
  const { data: me } = useMe()
  const email = me?.user.email ?? ''
  const { busy, run } = useSend()
  const [open, setOpen] = useState(false)
  const [picked, setPicked] = useState<string[]>([])
  const [extra, setExtra] = useState('')
  const [language, setLanguage] = useState(c.campaign.language)
  if (disabled) {
    return (
      <button type="button" className="btn" disabled style={{ opacity: 0.5 }}>
        {t(m.sendTest)}
      </button>
    )
  }
  if (!c.campaign.access.canSendTestToOthers || toMe) {
    const own = (
      <button type="button" className="btn" aria-busy={busy || undefined} onClick={() => void run([email])}>
        {toMe && email ? t(m.sendTestTo, { email }) : t(m.sendTestToMe)}
      </button>
    )
    // X14: the menu adds Run a placement test… (not in the preview's row).
    return toMe ? (
      own
    ) : (
      <TestSplit campaignId={c.campaign.id} toMe onSendTest={() => void run([email])} before={() => c.save()}>
        {own}
      </TestSplit>
    )
  }
  const members = c.context.members
  const toggle = (e: string) => setPicked((p) => (p.includes(e) ? p.filter((x) => x !== e) : [...p, e]))
  const recipients = [...picked, ...extra.split(/[\s,;]+/).filter((x) => x.includes('@'))]
  const popover = (
    <Popover
      trigger={
        <button type="button" className="btn">
          {t(m.sendTest)}
        </button>
      }
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (o) setPicked([email])
      }}
      label={t(m.sendATest)}
      className="w-[320px] max-w-[calc(100vw-24px)]"
    >
      <span className="mh">{t(m.sendTestOf, { variant: c.variant })}</span>
      {members.map((m) => (
        <label key={m.email} className="chk items-center rounded-lg px-2.5 py-1.5 hover:bg-ground" aria-label={`${m.name || m.email}, ${m.email}`}>
          <input type="checkbox" checked={picked.includes(m.email)} onChange={() => toggle(m.email)} />
          <span className="flex min-w-0 flex-1 flex-col">
            <span>{m.name || m.email}</span>
            <span className="k">{m.email}</span>
          </span>
        </label>
      ))}
      <TestLanguagePicker value={language} onChange={setLanguage} />
      <label className="lbl px-2.5 py-1.5">
        {t(m.otherAddresses)}
        <input className="in" value={extra} onChange={(e) => setExtra(e.target.value)} placeholder={'name@example.com'} />
      </label>
      <div className="flex items-center gap-2 border-t border-line-2 px-2.5 pt-2.5 pb-1">
        <span className="k flex-1">{t(m.testRoute)}</span>
        <button
          type="button"
          className="btn pri sm"
          disabled={!recipients.length || recipients.length > 10}
          aria-busy={busy || undefined}
          onClick={() => void run(recipients, language).then((ok) => ok && setOpen(false))}
        >
          {t(m.send)}
        </button>
      </div>
    </Popover>
  )
  // X14: Send test is a split button; its menu adds Run a placement test….
  return (
    <TestSplit
      campaignId={c.campaign.id}
      onSendTest={() => {
        setPicked([email])
        setOpen(true)
      }}
      before={() => c.save()}
    >
      {popover}
    </TestSplit>
  )
}
