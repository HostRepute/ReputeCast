import { $api } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useWorkspace } from '@/lib/session'
import { mv } from './voice.messages'

/** Kept separate from the settings editor so the composer loads only the selector. */
export function VoiceSelection({ value, onChange }: { value?: number; onChange: (revision: number | undefined) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/brand-kit/voice/history', { params: { path: { workspace: slug } } })
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <Field label={t(mv.voiceSelection)}>
        <Select value={value ?? 'latest'} onChange={(e) => onChange(e.target.value === 'latest' ? undefined : Number(e.target.value))}>
          <option value="latest">{t(mv.latest)}</option>
          <option value={0}>{t(mv.none)}</option>
          {value && !q.data?.items.some((v) => v.revision === value) ? <option value={value}>{t(mv.revision, { revision: value })}</option> : null}
          {q.data?.items.map((v) => <option key={v.revision} value={v.revision}>{t(mv.revision, { revision: v.revision })}</option>)}
        </Select>
      </Field>
      {q.isError ? <p className="k m-0" role="status">{t(mv.loadFailed)} <Button size="sm" onClick={() => void q.refetch()}>{t(common.retry)}</Button></p> : null}
    </div>
  )
}
