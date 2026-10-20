import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { useInvalidateWebsite, WS, type PastCampaign } from './api'
import { mp } from './posts.messages'

/**
 * Publish past campaigns (WebsiteStates · publishing past campaigns): sent
 * campaigns that aren't posts yet, each becoming a post in the site's theme.
 */
export function PastCampaignsDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const past = $api.useQuery('get', `${WS}/past-campaigns`, { params: { path: { workspace: slug } } })
  const [picked, setPicked] = useState<string[] | null>(null)
  const [visibility, setVisibility] = useState<'audience' | 'public' | 'members'>('audience')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const items = past.data ?? []
  // Campaigns with nothing against them start checked; the archive's exceptions wait for a click.
  const chosen = picked ?? items.filter((c) => !c.note).map((c) => c.id)
  const toggle = (id: string, on: boolean) => setPicked(on ? [...chosen, id] : chosen.filter((x) => x !== id))
  const publish = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST(`${WS}/past-campaigns/publish`, { params: { path: { workspace: slug } }, body: { campaigns: chosen, visibility } })
      toast({ kind: 'success', title: t(mp.pastDone, { count: data?.published ?? chosen.length }) })
      void invalidate()
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={past.data ? <span className="k">{t(mp.pastCount, { count: items.length })}</span> : undefined}
      title={t(mp.pastTitle)}
      description={t(mp.pastText)}
      footer={
        <>
          <Button onClick={onClose}>{t(mp.cancel)}</Button>
          <Button variant="primary" loading={busy} disabledReason={chosen.length ? undefined : t(mp.pastPick)} onClick={() => void publish()}>
            {t(mp.pastPublish, { count: chosen.length })}
          </Button>
        </>
      }
    >
      {past.isError ? (
        <Note tone="bad">{errorMessage(past.error)}</Note>
      ) : !past.data ? (
        <Loading label={t(mp.pastLoading)} className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} w="100%" h={54} r={8} />
          ))}
        </Loading>
      ) : items.length === 0 ? (
        <p className="k m-0">{t(mp.pastNone)}</p>
      ) : (
        <div className="flex flex-col gap-2">
          {items.map((c) => (
            <label key={c.id} className="ws-lcheck">
              <input type="checkbox" checked={chosen.includes(c.id)} onChange={(e) => toggle(c.id, e.target.checked)} />
              <span className="tx">
                {c.name}
                <span className="k">{pastLine(t, c)}</span>
              </span>
            </label>
          ))}
        </div>
      )}
      {items.length ? (
        <Field label={t(mp.pastVisibility)}>
          <Select aria-label={t(mp.pastVisibilityLabel)} value={visibility} onChange={(e) => setVisibility(e.target.value as typeof visibility)}>
            <option value="audience">{t(mp.followAudience)}</option>
            <option value="public">{t(mp.public)}</option>
            <option value="members">{t(mp.members)}</option>
          </Select>
        </Field>
      ) : null}
      {error ? (
        <Note tone="bad">
          {t(mp.pastFailed)}: {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

function pastLine(t: ReturnType<typeof useT>, c: PastCampaign): string {
  const lists = c.lists.join(', ')
  const date = dayMonthDate(c.sentAt)
  if (c.note === 'archive_hidden') return t(mp.pastLineHidden, { lists, date })
  if (c.note === 'list_private') return t(mp.pastLinePrivate, { lists, date })
  return t(mp.pastLine, { lists, date, recipients: c.recipients })
}
